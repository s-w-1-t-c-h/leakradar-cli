// Package commands implements the leakradar-cli CLI's cobra command tree.
package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"leakradar-cli/internal/api"
	"leakradar-cli/internal/config"
	"leakradar-cli/internal/output"
	"leakradar-cli/internal/record"
)

var (
	flagJSON    bool
	flagAPIKey  string
	flagTimeout time.Duration
	flagVerbose bool
	flagBaseURL string
	flagOutDir  string
)

const outDirEnvVar = "LEAKRADAR_OUTDIR"

// RootCmd is the top-level `leakradar-cli` command.
var RootCmd = &cobra.Command{
	Use:           "leakradar-cli",
	Short:         "Query the LeakRadar breach/leak-intelligence API from the command line",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	RootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "output machine-readable JSON instead of a table")
	RootCmd.PersistentFlags().StringVar(&flagAPIKey, "api-key", "", "API key override (prefer 'leakradar-cli auth set' or LEAKRADAR_API_KEY; this flag may end up in shell history)")
	RootCmd.PersistentFlags().DurationVar(&flagTimeout, "timeout", 30*time.Second, "per-request HTTP timeout")
	RootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "verbose logging (never logs the API key itself)")
	RootCmd.PersistentFlags().StringVar(&flagBaseURL, "base-url", "", "override the API base URL (advanced/testing use only)")
	RootCmd.PersistentFlags().StringVar(&flagOutDir, "outdir", "", "save results into an organised directory tree here, plus an audit.jsonl log (or set "+outDirEnvVar+")")
}

// newClient resolves the API key (flag > env var > keychain > config file)
// and builds a configured api.Client. It is the single choke point every
// command goes through, so key resolution and verbose logging stay consistent.
func newClient() (*api.Client, error) {
	key := flagAPIKey
	if key == "" {
		resolved, err := config.ResolveAPIKey()
		if err != nil {
			return nil, err
		}
		key = resolved
	}

	c := api.New(key)
	c.HTTPClient.Timeout = flagTimeout
	if flagBaseURL != "" {
		c.BaseURL = flagBaseURL
	}
	if flagVerbose {
		fmt.Fprintf(os.Stderr, "verbose: using API base %s, key source resolved (key not shown)\n", output.TerminalSafe(c.BaseURL))
	}
	return c, nil
}

// rootCtx is set by main() to a context cancelled on SIGINT/SIGTERM, so a
// Ctrl-C during a long batch run stops in-flight retries/backoff cleanly.
var rootCtx = context.Background()

// SetContext lets main() install the signal-aware context before Execute().
func SetContext(ctx context.Context) { rootCtx = ctx }

func cmdContext() context.Context {
	return rootCtx
}

// jsonOut is a tiny readability helper for command bodies.
func jsonOut() bool { return flagJSON }

// rec resolves the --outdir flag (falling back to LEAKRADAR_OUTDIR) and
// returns a Recorder. Called fresh each time rather than cached, since the
// flag isn't parsed yet when package-level vars would otherwise initialise.
func rec() *record.Recorder {
	dir := flagOutDir
	if dir == "" {
		dir = os.Getenv(outDirEnvVar)
	}
	return record.New(dir)
}

// renderText runs render (an output.XXX call bound to a fresh writer, table
// mode) into a buffer and returns the result, so a Recorder.Record() call
// can save the same human-readable view shown on stdout alongside the JSON.
// Skips the work entirely when the recorder is disabled, since --outdir is
// off in the common case and this would otherwise double every render.
func renderText(render func(w io.Writer) error) string {
	if !rec().Enabled() {
		return ""
	}
	var buf bytes.Buffer
	if err := render(&buf); err != nil {
		return ""
	}
	return buf.String()
}

// redactedCommandLine reconstructs the invocation for the audit log with
// any --api-key value replaced, so a secret never lands in audit.jsonl.
func redactedCommandLine() string {
	args := make([]string, 0, len(os.Args))
	args = append(args, "leakradar-cli")
	skipNext := false
	for _, a := range os.Args[1:] {
		if skipNext {
			args = append(args, "REDACTED")
			skipNext = false
			continue
		}
		if a == "--api-key" {
			args = append(args, a)
			skipNext = true
			continue
		}
		if strings.HasPrefix(a, "--api-key=") {
			args = append(args, "--api-key=REDACTED")
			continue
		}
		args = append(args, a)
	}
	return strings.Join(args, " ")
}

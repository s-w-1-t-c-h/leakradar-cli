// Command leakradar-cli is a cross-platform CLI for the LeakRadar breach/leak
// intelligence API (https://api.leakradar.io): email/domain/advanced/dark-web
// search, batch checks, unlocks and exports.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"leakradar-cli/cmd/leakradar-cli/commands"
	"leakradar-cli/internal/output"
)

// version/commit are set via -ldflags by goreleaser; "dev" otherwise.
var (
	version = "dev"
	commit  = "none"
)

func main() {
	commands.Version = version
	commands.Commit = commit
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	commands.SetContext(ctx)

	// Exactly one blank line frames every command's output, front and
	// back — individual commands print their content plus any internal
	// separators they need (e.g. between table and summary line), but
	// never their own leading/trailing blank line; that's handled once,
	// here, so it's consistent across the whole CLI regardless of which
	// command ran or whether it errored. (A custom shell prompt that also
	// inserts its own blank line before the next prompt will stack with
	// this — that's a shell-config concern, not something to design
	// leakradar-cli's own output around.)
	fmt.Println()
	err := commands.RootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", output.TerminalSafe(err.Error()))
	}
	fmt.Println()

	if err != nil {
		os.Exit(1)
	}
}

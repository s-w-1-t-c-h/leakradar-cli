package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"leakradar-cli/internal/api"
	"leakradar-cli/internal/output"
	"leakradar-cli/internal/record"
)

var batchIncludeCounts bool

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Bulk existence checks against a newline-delimited file of emails or domains",
}

var batchEmailsCmd = &cobra.Command{
	Use:   "emails <file>",
	Short: "Check up to any number of emails for locked-exists matches (auto-chunked at 100/request)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		values, err := readLines(args[0])
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		var all []api.EmailMassResult
		for _, chunk := range chunk(values, api.MaxBatchSize) {
			res, err := c.EmailsLockedExists(cmdContext(), chunk, batchIncludeCounts)
			if err != nil {
				return err
			}
			all = append(all, res...)
		}
		text := renderText(func(w io.Writer) error { return output.EmailMassResults(w, all, false) })
		rec().Record(record.TimestampPath("batch", "emails"), "batch-emails", args[0], redactedCommandLine(), all, text)
		return output.EmailMassResults(os.Stdout, all, jsonOut())
	},
}

var batchDomainsCmd = &cobra.Command{
	Use:   "domains <file>",
	Short: "Check up to any number of domains for locked-exists matches (auto-chunked at 100/request)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		values, err := readLines(args[0])
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		var all []api.DomainMassResult
		for _, chunk := range chunk(values, api.MaxBatchSize) {
			res, err := c.DomainsLockedExists(cmdContext(), chunk, batchIncludeCounts)
			if err != nil {
				return err
			}
			all = append(all, res...)
		}
		text := renderText(func(w io.Writer) error { return output.DomainMassResults(w, all, false) })
		rec().Record(record.TimestampPath("batch", "domains"), "batch-domains", args[0], redactedCommandLine(), all, text)
		return output.DomainMassResults(os.Stdout, all, jsonOut())
	},
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("%s contains no non-empty lines", path)
	}
	return lines, nil
}

func chunk(values []string, size int) [][]string {
	var out [][]string
	for i := 0; i < len(values); i += size {
		end := i + size
		if end > len(values) {
			end = len(values)
		}
		out = append(out, values[i:end])
	}
	return out
}

func init() {
	for _, c := range []*cobra.Command{batchEmailsCmd, batchDomainsCmd} {
		c.Flags().BoolVar(&batchIncludeCounts, "include-counts", false, "include per-category match counts")
	}
	batchCmd.AddCommand(batchEmailsCmd, batchDomainsCmd)
	RootCmd.AddCommand(batchCmd)
}

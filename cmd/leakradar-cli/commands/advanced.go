package commands

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"leakradar-cli/internal/output"
	"leakradar-cli/internal/record"
)

var advancedCmd = &cobra.Command{
	Use:   "advanced",
	Short: "Multi-field advanced search (username/password/url/domain/host/hash/TLD/port filters)",
	Long: "Multi-field advanced search across the LeakRadar corpus. Every field flag is repeatable and " +
		"combines with OR by default (use --force-and to require all values within a field). " +
		"Rate-limited to 5 req/s by the API. Run 'leakradar-cli advanced --help' for the full filter list.",
	RunE: func(cmd *cobra.Command, args []string) error {
		filters, err := buildAdvancedFilters()
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.SearchAdvanced(cmdContext(), filters)
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.Leaks(w, resp, false) })
		rec().Record(record.TimestampPath("advanced", "search"), "advanced-search", "", redactedCommandLine(), resp, text)
		return output.Leaks(os.Stdout, resp, jsonOut())
	},
}

func init() {
	registerAdvancedFlags(advancedCmd.Flags())
	RootCmd.AddCommand(advancedCmd)
}

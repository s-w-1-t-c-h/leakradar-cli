package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"leakradar-cli/internal/api"
	"leakradar-cli/internal/output"
	"leakradar-cli/internal/record"
)

var comboCmd = &cobra.Command{
	Use:   "combolist",
	Short: "Search the dedicated combolist dataset (separate from the main leak corpus)",
	Long: "The combolist dataset is a separate, larger credential-pair index from the main stealer-log/" +
		"database-breach corpus searched by 'email'/'domain'/'advanced'. It has no URL data at all — just " +
		"identifier+password pairs. Unlocking by search criteria is async-only: the API's synchronous unlock " +
		"endpoint takes explicit record IDs, and locked combolist rows never expose one (id is null until " +
		"already unlocked), so 'combolist unlock' always queues a background task.",
}

// printComboTaskResult prints an async combolist unlock-task result (a
// typed TaskStatus, unlike the main dataset's freeform AsyncTaskResult) and
// hints at how to poll it.
func printComboTaskResult(task *api.TaskStatus) error {
	if err := output.JSON(os.Stdout, task); err != nil {
		return err
	}
	if task.TaskID != "" {
		fmt.Fprintf(os.Stderr, "poll with: leakradar-cli task status %s\n", output.TerminalSafe(task.TaskID))
	}
	return nil
}

// --- combolist email / username ---

var comboEmailOpts struct {
	search         string
	page, pageSize int
	autoUnlock     bool
}

var comboEmailCmd = &cobra.Command{
	Use:   "email <email>",
	Short: "Search the combolist dataset by exact email",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.SearchComboEmail(cmdContext(), api.CombolistEmailSearchRequest{Email: args[0], Search: comboEmailOpts.search}, comboEmailOpts.page, comboEmailOpts.pageSize, comboEmailOpts.autoUnlock)
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.ComboLeaks(w, resp, false) })
		rec().Record(record.EmailPath(args[0], "combolist_search.json"), "combolist-email-search", args[0], redactedCommandLine(), resp, text)
		return output.ComboLeaks(os.Stdout, resp, jsonOut())
	},
}

var comboUsernameOpts struct {
	search         string
	page, pageSize int
	autoUnlock     bool
}

var comboUsernameCmd = &cobra.Command{
	Use:   "username <username>",
	Short: "Search the combolist dataset by exact non-email username",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.SearchComboUsername(cmdContext(), api.CombolistUsernameSearchRequest{Username: args[0], Search: comboUsernameOpts.search}, comboUsernameOpts.page, comboUsernameOpts.pageSize, comboUsernameOpts.autoUnlock)
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.ComboLeaks(w, resp, false) })
		rec().Record(record.TargetPath("combolist-username", args[0], "search.json"), "combolist-username-search", args[0], redactedCommandLine(), resp, text)
		return output.ComboLeaks(os.Stdout, resp, jsonOut())
	},
}

// --- combolist domain (report + records) ---

var comboDomainListOpts struct {
	page, pageSize int
	search         string
	autoUnlock     bool
}

var comboDomainCmd = &cobra.Command{
	Use:   "domain <domain>",
	Short: "Combolist summary report for a domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.ComboDomainReport(cmdContext(), args[0])
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.ComboDomainReport(w, resp, false) })
		rec().Record(record.DomainPath(args[0], "combolist_report.json"), "combolist-domain-report", args[0], redactedCommandLine(), resp, text)
		return output.ComboDomainReport(os.Stdout, resp, jsonOut())
	},
}

var comboDomainRecordsCmd = &cobra.Command{
	Use:   "records <domain>",
	Short: "List individual combolist records for a domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.ComboDomainList(cmdContext(), args[0], comboDomainListOpts.search, comboDomainListOpts.page, comboDomainListOpts.pageSize, comboDomainListOpts.autoUnlock)
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.ComboLeaks(w, resp, false) })
		rec().Record(record.DomainPath(args[0], "combolist_records.json"), "combolist-domain-records", args[0], redactedCommandLine(), resp, text)
		return output.ComboLeaks(os.Stdout, resp, jsonOut())
	},
}

// --- combolist advanced ---

var comboAdvancedOpts struct {
	page, pageSize int
	autoUnlock     bool
}

var comboAdvancedCmd = &cobra.Command{
	Use:   "advanced",
	Short: "Multi-field advanced search against the combolist dataset",
	Long:  "Narrower filter set than 'leakradar-cli advanced' — the combolist dataset has no URL data. Run --help for the full filter list.",
	RunE: func(cmd *cobra.Command, args []string) error {
		filters, err := buildComboAdvancedFilters()
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.SearchComboAdvanced(cmdContext(), filters, comboAdvancedOpts.page, comboAdvancedOpts.pageSize, comboAdvancedOpts.autoUnlock)
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.ComboLeaks(w, resp, false) })
		rec().Record(record.TimestampPath("combolist-advanced", "search"), "combolist-advanced-search", "", redactedCommandLine(), resp, text)
		return output.ComboLeaks(os.Stdout, resp, jsonOut())
	},
}

// --- combolist unlock (async-only; see comboCmd's Long text for why) ---

var comboUnlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Queue an async unlock of combolist records matching a search, spending account points",
	Long: "There is no synchronous 'unlock and print results now' call for the combolist dataset — every " +
		"subcommand here previews match/lock counts and your balance (reusing the same preview as 'leakradar-cli " +
		"unlock'), queues the task on confirmation, and prints its task_id for 'leakradar-cli task status'.",
}

var comboUnlockEmailOpts struct {
	search      string
	max, listID int
	yes         bool
}

var comboUnlockEmailCmd = &cobra.Command{
	Use:   "email <email>",
	Short: "Queue an async unlock of combolist email matches",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		preview, err := c.SearchComboEmail(cmdContext(), api.CombolistEmailSearchRequest{Email: args[0], Search: comboUnlockEmailOpts.search}, 1, 1, false)
		if err != nil {
			return err
		}
		proceed, err := previewUnlock(c, comboUnlockEmailOpts.yes, fmt.Sprintf("combolist email %q", args[0]), comboUnlockEmailOpts.max, preview.Total, preview.TotalUnlocked)
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}

		scoped := api.CombolistScopedRequest{Scope: "email", Value: args[0], Search: comboUnlockEmailOpts.search}
		task, err := c.ComboUnlockTask(cmdContext(), scoped, comboUnlockEmailOpts.max, comboUnlockEmailOpts.listID)
		if err != nil {
			return err
		}
		rec().Record(record.EmailPath(args[0], "combolist_unlock_task.json"), "combolist-email-unlock-task", args[0], redactedCommandLine(), task, "")
		return printComboTaskResult(task)
	},
}

var comboUnlockUsernameOpts struct {
	search      string
	max, listID int
	yes         bool
}

var comboUnlockUsernameCmd = &cobra.Command{
	Use:   "username <username>",
	Short: "Queue an async unlock of combolist username matches",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		preview, err := c.SearchComboUsername(cmdContext(), api.CombolistUsernameSearchRequest{Username: args[0], Search: comboUnlockUsernameOpts.search}, 1, 1, false)
		if err != nil {
			return err
		}
		proceed, err := previewUnlock(c, comboUnlockUsernameOpts.yes, fmt.Sprintf("combolist username %q", args[0]), comboUnlockUsernameOpts.max, preview.Total, preview.TotalUnlocked)
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}

		scoped := api.CombolistScopedRequest{Scope: "username", Value: args[0], Search: comboUnlockUsernameOpts.search}
		task, err := c.ComboUnlockTask(cmdContext(), scoped, comboUnlockUsernameOpts.max, comboUnlockUsernameOpts.listID)
		if err != nil {
			return err
		}
		rec().Record(record.TargetPath("combolist-username", args[0], "unlock_task.json"), "combolist-username-unlock-task", args[0], redactedCommandLine(), task, "")
		return printComboTaskResult(task)
	},
}

var comboUnlockDomainOpts struct {
	search      string
	max, listID int
	yes         bool
}

var comboUnlockDomainCmd = &cobra.Command{
	Use:   "domain <domain>",
	Short: "Queue an async unlock of combolist domain matches",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		preview, err := c.ComboDomainList(cmdContext(), args[0], comboUnlockDomainOpts.search, 1, 1, false)
		if err != nil {
			return err
		}
		proceed, err := previewUnlock(c, comboUnlockDomainOpts.yes, fmt.Sprintf("combolist domain %q", args[0]), comboUnlockDomainOpts.max, preview.Total, preview.TotalUnlocked)
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}

		scoped := api.CombolistScopedRequest{Scope: "domain", Value: args[0], Search: comboUnlockDomainOpts.search}
		task, err := c.ComboUnlockTask(cmdContext(), scoped, comboUnlockDomainOpts.max, comboUnlockDomainOpts.listID)
		if err != nil {
			return err
		}
		rec().Record(record.DomainPath(args[0], "combolist_unlock_task.json"), "combolist-domain-unlock-task", args[0], redactedCommandLine(), task, "")
		return printComboTaskResult(task)
	},
}

var comboUnlockAdvancedOpts struct {
	max, listID int
	yes         bool
}

var comboUnlockAdvancedCmd = &cobra.Command{
	Use:   "advanced",
	Short: "Queue an async unlock of combolist advanced-search matches",
	Long:  "Accepts the same filter flags as 'leakradar-cli combolist advanced'.",
	RunE: func(cmd *cobra.Command, args []string) error {
		filters, err := buildComboAdvancedFilters()
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		preview, err := c.SearchComboAdvanced(cmdContext(), filters, 1, 1, false)
		if err != nil {
			return err
		}
		proceed, err := previewUnlock(c, comboUnlockAdvancedOpts.yes, "combolist advanced-search matches", comboUnlockAdvancedOpts.max, preview.Total, preview.TotalUnlocked)
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}

		scoped := api.CombolistScopedRequest{Scope: "advanced", Filters: &filters}
		task, err := c.ComboUnlockTask(cmdContext(), scoped, comboUnlockAdvancedOpts.max, comboUnlockAdvancedOpts.listID)
		if err != nil {
			return err
		}
		rec().Record(record.TimestampPath("combolist-advanced", "unlock_task"), "combolist-advanced-unlock-task", "", redactedCommandLine(), task, "")
		return printComboTaskResult(task)
	},
}

// --- combolist export ---

var comboExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Queue CSV/TXT/JSON exports of combolist search results",
	Long:  "Same no-download-by-ID caveat as the main dataset's exports — see 'leakradar-cli export --help'.",
}

var comboExportEmailOpts struct{ search, format string }

var comboExportEmailCmd = &cobra.Command{
	Use:   "email <email>",
	Short: "Queue an export of combolist email search results",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		scoped := api.CombolistScopedRequest{Scope: "email", Value: args[0], Search: comboExportEmailOpts.search}
		resp, err := c.ComboExport(cmdContext(), scoped, comboExportEmailOpts.format)
		if err != nil {
			return err
		}
		return printQueuedExport("combolist-export-email", args[0], resp)
	},
}

var comboExportUsernameOpts struct{ search, format string }

var comboExportUsernameCmd = &cobra.Command{
	Use:   "username <username>",
	Short: "Queue an export of combolist username search results",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		scoped := api.CombolistScopedRequest{Scope: "username", Value: args[0], Search: comboExportUsernameOpts.search}
		resp, err := c.ComboExport(cmdContext(), scoped, comboExportUsernameOpts.format)
		if err != nil {
			return err
		}
		return printQueuedExport("combolist-export-username", args[0], resp)
	},
}

var comboExportDomainOpts struct{ search, format string }

var comboExportDomainCmd = &cobra.Command{
	Use:   "domain <domain>",
	Short: "Queue an export of combolist domain search results",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		scoped := api.CombolistScopedRequest{Scope: "domain", Value: args[0], Search: comboExportDomainOpts.search}
		resp, err := c.ComboExport(cmdContext(), scoped, comboExportDomainOpts.format)
		if err != nil {
			return err
		}
		return printQueuedExport("combolist-export-domain", args[0], resp)
	},
}

var comboExportAdvancedFormat string

var comboExportAdvancedCmd = &cobra.Command{
	Use:   "advanced",
	Short: "Queue an export of combolist advanced-search results",
	Long:  "Accepts the same filter flags as 'leakradar-cli combolist advanced'.",
	RunE: func(cmd *cobra.Command, args []string) error {
		filters, err := buildComboAdvancedFilters()
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		scoped := api.CombolistScopedRequest{Scope: "advanced", Filters: &filters}
		resp, err := c.ComboExport(cmdContext(), scoped, comboExportAdvancedFormat)
		if err != nil {
			return err
		}
		return printQueuedExport("combolist-export-advanced", "", resp)
	},
}

func init() {
	comboEmailCmd.Flags().StringVar(&comboEmailOpts.search, "search", "", "free-text filter on username or password")
	comboEmailCmd.Flags().IntVar(&comboEmailOpts.page, "page", 1, "page number")
	comboEmailCmd.Flags().IntVar(&comboEmailOpts.pageSize, "page-size", 100, "results per page")
	comboEmailCmd.Flags().BoolVar(&comboEmailOpts.autoUnlock, "auto-unlock", false, "auto-unlock matches as part of this search (spends points)")

	comboUsernameCmd.Flags().StringVar(&comboUsernameOpts.search, "search", "", "free-text filter on username or password")
	comboUsernameCmd.Flags().IntVar(&comboUsernameOpts.page, "page", 1, "page number")
	comboUsernameCmd.Flags().IntVar(&comboUsernameOpts.pageSize, "page-size", 100, "results per page")
	comboUsernameCmd.Flags().BoolVar(&comboUsernameOpts.autoUnlock, "auto-unlock", false, "auto-unlock matches as part of this search (spends points)")

	comboDomainRecordsCmd.Flags().StringVar(&comboDomainListOpts.search, "search", "", "free-text filter")
	comboDomainRecordsCmd.Flags().IntVar(&comboDomainListOpts.page, "page", 1, "page number")
	comboDomainRecordsCmd.Flags().IntVar(&comboDomainListOpts.pageSize, "page-size", 100, "results per page")
	comboDomainRecordsCmd.Flags().BoolVar(&comboDomainListOpts.autoUnlock, "auto-unlock", false, "auto-unlock matches as part of this search (spends points)")
	comboDomainCmd.AddCommand(comboDomainRecordsCmd)

	registerComboAdvancedFlags(comboAdvancedCmd.Flags())
	comboAdvancedCmd.Flags().IntVar(&comboAdvancedOpts.page, "page", 1, "page number")
	comboAdvancedCmd.Flags().IntVar(&comboAdvancedOpts.pageSize, "page-size", 100, "results per page")
	comboAdvancedCmd.Flags().BoolVar(&comboAdvancedOpts.autoUnlock, "auto-unlock", false, "auto-unlock matches as part of this search (spends points)")

	comboUnlockEmailCmd.Flags().StringVar(&comboUnlockEmailOpts.search, "search", "", "free-text filter on username or password")
	comboUnlockEmailCmd.Flags().IntVar(&comboUnlockEmailOpts.max, "max", 0, "max records to unlock (0 = no cap - uses as many points as needed)")
	comboUnlockEmailCmd.Flags().IntVar(&comboUnlockEmailOpts.listID, "list-id", 0, "assign unlocked records to this list ID")
	comboUnlockEmailCmd.Flags().BoolVar(&comboUnlockEmailOpts.yes, "yes", false, "skip the confirmation prompt (for scripts/CI)")

	comboUnlockUsernameCmd.Flags().StringVar(&comboUnlockUsernameOpts.search, "search", "", "free-text filter on username or password")
	comboUnlockUsernameCmd.Flags().IntVar(&comboUnlockUsernameOpts.max, "max", 0, "max records to unlock (0 = no cap - uses as many points as needed)")
	comboUnlockUsernameCmd.Flags().IntVar(&comboUnlockUsernameOpts.listID, "list-id", 0, "assign unlocked records to this list ID")
	comboUnlockUsernameCmd.Flags().BoolVar(&comboUnlockUsernameOpts.yes, "yes", false, "skip the confirmation prompt (for scripts/CI)")

	comboUnlockDomainCmd.Flags().StringVar(&comboUnlockDomainOpts.search, "search", "", "free-text filter")
	comboUnlockDomainCmd.Flags().IntVar(&comboUnlockDomainOpts.max, "max", 0, "max records to unlock (0 = no cap - uses as many points as needed)")
	comboUnlockDomainCmd.Flags().IntVar(&comboUnlockDomainOpts.listID, "list-id", 0, "assign unlocked records to this list ID")
	comboUnlockDomainCmd.Flags().BoolVar(&comboUnlockDomainOpts.yes, "yes", false, "skip the confirmation prompt (for scripts/CI)")

	registerComboAdvancedFlags(comboUnlockAdvancedCmd.Flags())
	comboUnlockAdvancedCmd.Flags().IntVar(&comboUnlockAdvancedOpts.max, "max", 0, "max records to unlock (0 = no cap - uses as many points as needed)")
	comboUnlockAdvancedCmd.Flags().IntVar(&comboUnlockAdvancedOpts.listID, "list-id", 0, "assign unlocked records to this list ID")
	comboUnlockAdvancedCmd.Flags().BoolVar(&comboUnlockAdvancedOpts.yes, "yes", false, "skip the confirmation prompt (for scripts/CI)")
	comboUnlockCmd.AddCommand(comboUnlockEmailCmd, comboUnlockUsernameCmd, comboUnlockDomainCmd, comboUnlockAdvancedCmd)

	comboExportEmailCmd.Flags().StringVar(&comboExportEmailOpts.search, "search", "", "free-text filter on username or password")
	comboExportEmailCmd.Flags().StringVar(&comboExportEmailOpts.format, "format", "csv", "csv|txt|json")
	comboExportUsernameCmd.Flags().StringVar(&comboExportUsernameOpts.search, "search", "", "free-text filter on username or password")
	comboExportUsernameCmd.Flags().StringVar(&comboExportUsernameOpts.format, "format", "csv", "csv|txt|json")
	comboExportDomainCmd.Flags().StringVar(&comboExportDomainOpts.search, "search", "", "free-text filter")
	comboExportDomainCmd.Flags().StringVar(&comboExportDomainOpts.format, "format", "csv", "csv|txt|json")
	registerComboAdvancedFlags(comboExportAdvancedCmd.Flags())
	comboExportAdvancedCmd.Flags().StringVar(&comboExportAdvancedFormat, "format", "csv", "csv|txt|json")
	comboExportCmd.AddCommand(comboExportEmailCmd, comboExportUsernameCmd, comboExportDomainCmd, comboExportAdvancedCmd)

	comboCmd.AddCommand(comboEmailCmd, comboUsernameCmd, comboDomainCmd, comboAdvancedCmd, comboUnlockCmd, comboExportCmd)
	RootCmd.AddCommand(comboCmd)
}

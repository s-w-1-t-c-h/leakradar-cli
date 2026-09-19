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

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Queue CSV/TXT/JSON exports of search results, and stream the domain subdomains/urls exports directly",
	Long: "email/domain/advanced exports are queued server-side jobs: there is no download-by-ID API, so " +
		"completed files are retrieved from the LeakRadar member portal's Exports page, not this CLI. " +
		"'domain-subdomains'/'domain-urls' are different: those stream the file directly to --out.",
}

func printQueuedExport(kind, target string, resp *api.ExportResponse) error {
	fmt.Fprintf(os.Stderr, "queued export #%d (%s): %s\nCheck 'leakradar-cli export list' or the LeakRadar member portal's Exports page for the finished file.\n", resp.ExportID, output.TerminalSafe(resp.Status), output.TerminalSafe(resp.Message))
	rec().Record(record.TimestampPath("exports", kind), kind, target, redactedCommandLine(), resp, "")
	return output.JSON(os.Stdout, resp)
}

// --- export email ---

var exportEmailOpts struct {
	search              string
	isEmail, isUsername bool
	format              string
}

var exportEmailCmd = &cobra.Command{
	Use:   "email <email-or-username>",
	Short: "Queue an export of email search results",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		isEmail, err := isEmailPtr(exportEmailOpts.isEmail, exportEmailOpts.isUsername)
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.EmailExport(cmdContext(), api.EmailSearchRequest{
			Email: args[0], Search: exportEmailOpts.search, IsEmail: isEmail,
		}, exportEmailOpts.format)
		if err != nil {
			return err
		}
		return printQueuedExport("export-email", args[0], resp)
	},
}

// --- export domain ---

var exportDomainOpts struct {
	category            string
	search              string
	isEmail, isUsername bool
	format              string
}

var exportDomainCmd = &cobra.Command{
	Use:   "domain <domain>",
	Short: "Queue an export of domain search results for one category",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		lt, err := parseLeakType(exportDomainOpts.category)
		if err != nil {
			return err
		}
		isEmail, err := isEmailPtr(exportDomainOpts.isEmail, exportDomainOpts.isUsername)
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.DomainExport(cmdContext(), args[0], lt, exportDomainOpts.search, isEmail, exportDomainOpts.format)
		if err != nil {
			return err
		}
		return printQueuedExport("export-domain-"+string(lt), args[0], resp)
	},
}

// --- export domain-subdomains / domain-urls (synchronous streamed files) ---

var exportDomainListOpts struct {
	search, format, out string
}

var exportDomainSubdomainsCmd = &cobra.Command{
	Use:   "domain-subdomains <domain>",
	Short: "Stream an export of a domain's observed subdomains directly to --out",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := exportDomainListOpts.out
		recorder := rec()
		var outRel string
		if out == "" && recorder.Enabled() {
			outRel = record.DomainPath(args[0], "subdomains_export."+exportDomainListOpts.format)
			out = recorder.DefaultPath(outRel)
		}
		if out == "" {
			return fmt.Errorf("--out is required (or set --outdir/%s for a default location)", outDirEnvVar)
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		body, err := c.DomainSubdomainsExport(cmdContext(), args[0], exportDomainListOpts.search, exportDomainListOpts.format)
		if err != nil {
			return err
		}
		defer body.Close()
		if err := writeRecordedFile(out, outRel, recorder, body); err != nil {
			return err
		}
		recorder.RecordFile("export-domain-subdomains", args[0], redactedCommandLine(), out)
		return nil
	},
}

var exportDomainURLsCmd = &cobra.Command{
	Use:   "domain-urls <domain>",
	Short: "Stream an export of a domain's observed URLs directly to --out",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := exportDomainListOpts.out
		recorder := rec()
		var outRel string
		if out == "" && recorder.Enabled() {
			outRel = record.DomainPath(args[0], "urls_export."+exportDomainListOpts.format)
			out = recorder.DefaultPath(outRel)
		}
		if out == "" {
			return fmt.Errorf("--out is required (or set --outdir/%s for a default location)", outDirEnvVar)
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		body, err := c.DomainURLsExport(cmdContext(), args[0], exportDomainListOpts.search, exportDomainListOpts.format)
		if err != nil {
			return err
		}
		defer body.Close()
		if err := writeRecordedFile(out, outRel, recorder, body); err != nil {
			return err
		}
		recorder.RecordFile("export-domain-urls", args[0], redactedCommandLine(), out)
		return nil
	},
}

// --- export advanced ---

var exportAdvancedFormat string

var exportAdvancedCmd = &cobra.Command{
	Use:   "advanced",
	Short: "Queue an export of advanced search results",
	Long:  "Accepts the same filter flags as 'leakradar-cli advanced'.",
	RunE: func(cmd *cobra.Command, args []string) error {
		filters, err := buildAdvancedFilters()
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.AdvancedExport(cmdContext(), filters, exportAdvancedFormat)
		if err != nil {
			return err
		}
		return printQueuedExport("export-advanced", "", resp)
	},
}

// --- export list ---

var exportListOpts struct{ page, pageSize int }

var exportListCmd = &cobra.Command{
	Use:   "list",
	Short: "List export jobs",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.ListExports(cmdContext(), exportListOpts.page, exportListOpts.pageSize)
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.Exports(w, resp, false) })
		rec().Record(record.TimestampPath("exports", "list"), "export-list", "", redactedCommandLine(), resp, text)
		return output.Exports(os.Stdout, resp, jsonOut())
	},
}

func init() {
	exportEmailCmd.Flags().StringVar(&exportEmailOpts.search, "search", "", "free-text filter on URL or username")
	exportEmailCmd.Flags().BoolVar(&exportEmailOpts.isEmail, "is-email", false, "restrict to email matches only")
	exportEmailCmd.Flags().BoolVar(&exportEmailOpts.isUsername, "is-username", false, "restrict to username matches only")
	exportEmailCmd.Flags().StringVar(&exportEmailOpts.format, "format", "csv", "csv|txt|json")

	exportDomainCmd.Flags().StringVar(&exportDomainOpts.category, "category", "all", "employees|customers|third_parties|all")
	exportDomainCmd.Flags().StringVar(&exportDomainOpts.search, "search", "", "free-text filter")
	exportDomainCmd.Flags().BoolVar(&exportDomainOpts.isEmail, "is-email", false, "restrict to email matches only")
	exportDomainCmd.Flags().BoolVar(&exportDomainOpts.isUsername, "is-username", false, "restrict to username matches only")
	exportDomainCmd.Flags().StringVar(&exportDomainOpts.format, "format", "csv", "csv|txt|json")

	for _, c := range []*cobra.Command{exportDomainSubdomainsCmd, exportDomainURLsCmd} {
		c.Flags().StringVar(&exportDomainListOpts.search, "search", "", "free-text filter")
		c.Flags().StringVar(&exportDomainListOpts.format, "format", "csv", "csv|txt|json")
		c.Flags().StringVar(&exportDomainListOpts.out, "out", "", "output file path (required unless --outdir/"+outDirEnvVar+" is set, which supplies a default)")
	}

	registerAdvancedFlags(exportAdvancedCmd.Flags())
	exportAdvancedCmd.Flags().StringVar(&exportAdvancedFormat, "format", "csv", "csv|txt|json")

	exportListCmd.Flags().IntVar(&exportListOpts.page, "page", 1, "page number")
	exportListCmd.Flags().IntVar(&exportListOpts.pageSize, "page-size", 20, "results per page (max 100)")

	exportCmd.AddCommand(exportEmailCmd, exportDomainCmd, exportDomainSubdomainsCmd, exportDomainURLsCmd, exportAdvancedCmd, exportListCmd)
	RootCmd.AddCommand(exportCmd)
}

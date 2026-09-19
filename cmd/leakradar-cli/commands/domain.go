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

var domainOpts struct {
	category            string
	light               bool
	includeSearchCount  bool
	page, pageSize      int
	search              string
	isEmail, isUsername bool
}

var domainCmd = &cobra.Command{
	Use:   "domain <domain>",
	Short: "Domain leak report, or a paginated list for one category",
	Long: "With no --category, prints the summary counts (employees/customers/third_parties) for the domain. " +
		"With --category, lists the individual leak records for that category.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		c, err := newClient()
		if err != nil {
			return err
		}

		if domainOpts.category == "" {
			resp, err := c.DomainReport(cmdContext(), domain, domainOpts.light, domainOpts.includeSearchCount)
			if err != nil {
				return err
			}
			view := output.DomainReportView{DomainSearchResponse: resp}
			if !domainOpts.light {
				emp, cus, third, err := fetchDomainUnlockedCounts(c, domain)
				if err != nil {
					if flagVerbose {
						fmt.Fprintf(os.Stderr, "warning: could not fetch unlocked counts per category: %s\n", output.TerminalSafe(err.Error()))
					}
				} else {
					view.UnlockedEmployees, view.UnlockedCustomers, view.UnlockedThirdParties = &emp, &cus, &third
				}
			}
			text := renderText(func(w io.Writer) error { return output.DomainReport(w, domain, view, false) })
			rec().Record(record.DomainPath(domain, "report.json"), "domain-report", domain, redactedCommandLine(), view, text)
			return output.DomainReport(os.Stdout, domain, view, jsonOut())
		}

		lt, err := parseLeakType(domainOpts.category)
		if err != nil {
			return err
		}
		isEmail, err := isEmailPtr(domainOpts.isEmail, domainOpts.isUsername)
		if err != nil {
			return err
		}
		params := api.DomainListParams{
			Page: domainOpts.page, PageSize: domainOpts.pageSize, Search: domainOpts.search, IsEmail: isEmail,
		}

		if lt == api.LeakTypeAll {
			resp, err := c.DomainListAll(cmdContext(), domain, params)
			if err != nil {
				return err
			}
			text := renderText(func(w io.Writer) error { return output.AllLeaks(w, resp, false) })
			rec().Record(record.DomainPath(domain, "all.json"), "domain-all", domain, redactedCommandLine(), resp, text)
			return output.AllLeaks(os.Stdout, resp, jsonOut())
		}
		resp, err := c.DomainList(cmdContext(), domain, lt, params)
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.Leaks(w, resp, false) })
		rec().Record(record.DomainPath(domain, string(lt)+".json"), "domain-"+string(lt), domain, redactedCommandLine(), resp, text)
		return output.Leaks(os.Stdout, resp, jsonOut())
	},
}

var domainListOpts struct {
	page, pageSize int
	search         string
}

var domainSubdomainsCmd = &cobra.Command{
	Use:   "subdomains <domain>",
	Short: "List distinct subdomains observed in leaks for a domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.DomainSubdomains(cmdContext(), args[0], domainListOpts.page, domainListOpts.pageSize, domainListOpts.search)
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.DomainSubdomains(w, resp, false) })
		rec().Record(record.DomainPath(args[0], "subdomains.json"), "domain-subdomains", args[0], redactedCommandLine(), resp, text)
		return output.DomainSubdomains(os.Stdout, resp, jsonOut())
	},
}

var domainURLsCmd = &cobra.Command{
	Use:   "urls <domain>",
	Short: "List distinct URLs observed in leaks for a domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.DomainURLs(cmdContext(), args[0], domainListOpts.page, domainListOpts.pageSize, domainListOpts.search)
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.DomainURLs(w, resp, false) })
		rec().Record(record.DomainPath(args[0], "urls.json"), "domain-urls", args[0], redactedCommandLine(), resp, text)
		return output.DomainURLs(os.Stdout, resp, jsonOut())
	},
}

var domainPDFOut string

var domainPDFCmd = &cobra.Command{
	Use:   "pdf <domain>",
	Short: "Download the domain leak report as a PDF",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := domainPDFOut
		recorder := rec()
		var outRel string
		if out == "" && recorder.Enabled() {
			outRel = record.DomainPath(args[0], "report.pdf")
			out = recorder.DefaultPath(outRel)
		}
		if out == "" {
			return fmt.Errorf("--out is required (or set --outdir/%s for a default location)", outDirEnvVar)
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		body, err := c.DomainReportPDF(cmdContext(), args[0])
		if err != nil {
			return err
		}
		defer body.Close()
		if err := writeRecordedFile(out, outRel, recorder, body); err != nil {
			return err
		}
		recorder.RecordFile("domain-pdf", args[0], redactedCommandLine(), out)
		return nil
	},
}

// fetchDomainUnlockedCounts fetches TotalUnlocked for each of
// employees/customers/third_parties (page_size=1, since only the total
// counts are needed) — the domain-report endpoint itself doesn't return
// unlocked counts, so this backfills them for the plain `domain <domain>`
// summary view. Best-effort: callers treat an error as "omit the column".
func fetchDomainUnlockedCounts(c *api.Client, domain string) (employees, customers, thirdParties int, err error) {
	p := api.DomainListParams{Page: 1, PageSize: 1}
	empResp, err := c.DomainList(cmdContext(), domain, api.LeakTypeEmployees, p)
	if err != nil {
		return 0, 0, 0, err
	}
	cusResp, err := c.DomainList(cmdContext(), domain, api.LeakTypeCustomers, p)
	if err != nil {
		return 0, 0, 0, err
	}
	thirdResp, err := c.DomainList(cmdContext(), domain, api.LeakTypeThirdParties, p)
	if err != nil {
		return 0, 0, 0, err
	}
	return empResp.TotalUnlocked, cusResp.TotalUnlocked, thirdResp.TotalUnlocked, nil
}

func parseLeakType(s string) (api.LeakType, error) {
	switch s {
	case "all":
		return api.LeakTypeAll, nil
	case "employees":
		return api.LeakTypeEmployees, nil
	case "customers":
		return api.LeakTypeCustomers, nil
	case "third_parties":
		return api.LeakTypeThirdParties, nil
	default:
		return "", fmt.Errorf("invalid --category %q (must be one of all, employees, customers, third_parties)", s)
	}
}

func isEmailPtr(isEmail, isUsername bool) (*bool, error) {
	if isEmail && isUsername {
		return nil, fmt.Errorf("--is-email and --is-username are mutually exclusive")
	}
	if isEmail {
		v := true
		return &v, nil
	}
	if isUsername {
		v := false
		return &v, nil
	}
	return nil, nil
}

func init() {
	domainCmd.Flags().StringVar(&domainOpts.category, "category", "", "employees|customers|third_parties|all (omit for the summary report)")
	domainCmd.Flags().BoolVar(&domainOpts.light, "light", false, "summary report only, without password-strength stats")
	domainCmd.Flags().BoolVar(&domainOpts.includeSearchCount, "include-search-count", false, "include how many other users have searched this domain")
	domainCmd.Flags().IntVar(&domainOpts.page, "page", 1, "page number")
	domainCmd.Flags().IntVar(&domainOpts.pageSize, "page-size", 100, "results per page (max 1000)")
	domainCmd.Flags().StringVar(&domainOpts.search, "search", "", "free-text filter")
	domainCmd.Flags().BoolVar(&domainOpts.isEmail, "is-email", false, "restrict to email matches only")
	domainCmd.Flags().BoolVar(&domainOpts.isUsername, "is-username", false, "restrict to username matches only")

	for _, c := range []*cobra.Command{domainSubdomainsCmd, domainURLsCmd} {
		c.Flags().IntVar(&domainListOpts.page, "page", 1, "page number")
		c.Flags().IntVar(&domainListOpts.pageSize, "page-size", 100, "results per page (max 1000)")
		c.Flags().StringVar(&domainListOpts.search, "search", "", "free-text filter")
	}
	domainPDFCmd.Flags().StringVar(&domainPDFOut, "out", "", "output file path (required unless --outdir/"+outDirEnvVar+" is set, which supplies a default)")

	domainCmd.AddCommand(domainSubdomainsCmd, domainURLsCmd, domainPDFCmd)
	RootCmd.AddCommand(domainCmd)
}

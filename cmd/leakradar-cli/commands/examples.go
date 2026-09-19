package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// exampleEntry is one runnable example (Cmd, without a leading "leakradar-cli ")
// plus a one-line description. Cmd is verified syntactically correct by
// TestAllExamples_ParseAgainstRealCommandTree in examples_test.go — every
// entry here is checked against the live cobra command tree (flag names,
// subcommand resolution, positional arg counts) so this list can't silently
// drift out of sync with the actual CLI.
type exampleEntry struct {
	Cmd  string
	Desc string
}

type exampleSection struct {
	Title    string
	Examples []exampleEntry
}

var exampleSections = []exampleSection{
	{"Auth & account", []exampleEntry{
		{"auth set", "store your API key (masked prompt, or pipe it via stdin)"},
		{"auth status", "validate the stored key against the API"},
		{"auth clear", "remove the stored key"},
		{"profile", "account info, plan, and point/quota balance"},
		{"balance", "unlock credits (points) and live daily-request/unlocked-lists quota"},
	}},
	{"Email search", []exampleEntry{
		{"email jsmith@example.com", "search by exact email"},
		{"email jsmith --is-username --search admin", "restrict to username matches, filter by free text"},
	}},
	{"Domain report & lists", []exampleEntry{
		{"domain example.com", "summary report, plus unlocked counts per category"},
		{"domain example.com --light", "sampled/approximate summary only, skips the unlocked-count lookups"},
		{"domain example.com --category customers --page-size 200", "paginated list for one category"},
		{"domain example.com --category all --page-size 50", "paginated list across all three categories, tagged per row"},
		{"domain subdomains example.com", "distinct subdomains observed in leaks"},
		{"domain urls example.com", "distinct URLs observed in leaks"},
		{"domain pdf example.com --out report.pdf", "download the domain report as a PDF"},
	}},
	{"Advanced search (structured dataset)", []exampleEntry{
		{"advanced --url-domain example.com --password-strength weak", "url_domain filter + password strength"},
		{"advanced --email-domain example.com --is-email", "email_domain filter, emails only"},
		{"advanced --url-tld au --url-scheme https", "TLD + scheme filters (broad, not domain-scoped)"},
	}},
	{"Dark web", []exampleEntry{
		{"darkweb --query example.com", "simple full-text search"},
		{"darkweb --title example.com --date-from 2025-01-01T00:00:00Z", "field-specific search with a date lower bound"},
	}},
	{"Password range (k-anonymity)", []exampleEntry{
		{"password-range 5BAA6 --limit 20", "SHA-1 hash prefix lookup"},
	}},
	{"Batch existence checks", []exampleEntry{
		{"batch emails emails.txt --include-counts", "check a file of emails, auto-chunked at 100/request"},
		{"batch domains domains.txt", "check a file of domains"},
	}},
	{"Unlock (spends points; every command previews cost/balance first)", []exampleEntry{
		{"unlock email jsmith@example.com --max 5", "unlock up to 5 email matches"},
		{"unlock domain example.com --category employees --max 20 --yes", "unlock one category, skip the confirm prompt"},
		{"unlock domain example.com --max 10", "default --category all: 3 real calls sharing one --max budget, tagged per category"},
		{"unlock advanced --url-domain example.com --async", "async unlock via 'leakradar-cli advanced' filters"},
		{"task status abc123", "poll an --async unlock task"},
	}},
	{"Export (queues a job; no download-by-ID API — retrieve from the member portal)", []exampleEntry{
		{"export email jsmith@example.com --format csv", "queue an email-search export"},
		{"export domain example.com --category customers --format json", "queue a domain-category export"},
		{"export list", "list export jobs"},
		{"export domain-subdomains example.com --out subdomains.csv", "synchronous, streamed straight to --out"},
		{"export domain-urls example.com --out urls.csv", "synchronous, streamed straight to --out"},
	}},
	{"Combolist (separate dataset; search usually works even if unlock/export 403 on your plan)", []exampleEntry{
		{"combolist email jsmith@example.com", "search by exact email"},
		{"combolist username jsmith", "search by exact non-email username"},
		{"combolist domain example.com", "summary report"},
		{"combolist domain records example.com --page-size 50", "paginated individual records"},
		{"combolist advanced --email-domain example.com --password-strength weak", "narrower filter set (no URL fields)"},
		{"combolist unlock domain example.com --max 10", "always async — no synchronous unlock exists for this dataset"},
		{"combolist export domain example.com --format csv", "queue an export"},
	}},
	{"Raw (full-text search over unparsed source files; needs --q or a metadata filter)", []exampleEntry{
		{"raw count --q example.com", "cheap upper-bound cost check before searching/unlocking"},
		{"raw search --q example.com --page-size 25", "full-text search"},
		{"raw search --q example.com --category stealer-logs --ext txt", "narrow by category and extension"},
		{"raw search --q example.com --page-size 25 --cursor abc123", "continue from a previous next_cursor"},
		{"raw unlock --q example.com --max 10", "always async; cost preview is exact (1 point/part, per the API)"},
		{"raw export --q example.com --mode rows", "export already-unlocked rows as CSV (does not unlock anything)"},
	}},
	{"Saving results to disk", []exampleEntry{
		{"--outdir ./engagement/leakradar-cli domain example.com", "save every result into an organised tree + audit log"},
	}},
	{"Misc", []exampleEntry{
		{"version", "print the CLI version"},
	}},
}

var examplesCmd = &cobra.Command{
	Use:   "examples [filter]",
	Short: "Print runnable example commands for every feature, grouped by area",
	Long: "Every example here is verified to parse correctly against the actual command tree (flag names, " +
		"subcommand paths, positional argument counts) as part of this tool's test suite — if a flag gets " +
		"renamed, this list breaks the build until it's updated, so it can't silently drift out of date. " +
		"Placeholder values (example.com, jsmith@example.com, task/export IDs) need swapping for your own. " +
		"Pass an optional filter word (e.g. 'raw', 'unlock', 'combolist') to show only matching sections/lines.",
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		filter := ""
		if len(args) > 0 {
			filter = strings.ToLower(strings.Join(args, " "))
		}
		printExamples(os.Stdout, filter)
		return nil
	},
}

func printExamples(w io.Writer, filter string) {
	shown := 0
	first := true
	for _, section := range exampleSections {
		var matched []exampleEntry
		sectionMatches := filter == "" || strings.Contains(strings.ToLower(section.Title), filter)
		for _, ex := range section.Examples {
			if sectionMatches || strings.Contains(strings.ToLower(ex.Cmd), filter) || strings.Contains(strings.ToLower(ex.Desc), filter) {
				matched = append(matched, ex)
			}
		}
		if len(matched) == 0 {
			continue
		}
		// Blank line *between* sections only (not after the last one) —
		// the leading/trailing blank line around the whole command's
		// output is main.go's job, not this function's.
		if !first {
			fmt.Fprintln(w)
		}
		first = false
		fmt.Fprintf(w, "# %s\n", section.Title)
		for _, ex := range matched {
			fmt.Fprintf(w, "leakradar-cli %s\n", ex.Cmd)
			fmt.Fprintf(w, "  # %s\n", ex.Desc)
		}
		shown += len(matched)
	}
	if shown == 0 {
		fmt.Fprintf(w, "no examples match %q — try 'leakradar-cli examples' with no filter to see everything\n", filter)
	}
}

func init() {
	RootCmd.AddCommand(examplesCmd)
}

// Package output renders API results either as a JSON blob (for scripting,
// via --json) or as a human-readable table, so every command shares the
// exact same two code paths instead of each hand-rolling its own printing.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"golang.org/x/term"

	"leakradar-cli/internal/api"
)

// JSON marshals v as indented JSON to w.
func JSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// TerminalSafe preserves printable text while rendering control and format
// characters visibly. API data is inherently untrusted and must not be able to
// emit ANSI/OSC sequences or alter terminal state in human-readable output.
func TerminalSafe(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strconv.IsPrint(r) {
			b.WriteRune(r)
			continue
		}
		switch r {
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			switch {
			case r <= 0xff:
				fmt.Fprintf(&b, `\x%02x`, r)
			case r <= 0xffff:
				fmt.Fprintf(&b, `\u%04x`, r)
			default:
				fmt.Fprintf(&b, `\U%08x`, r)
			}
		}
	}
	return b.String()
}

// Table renders a simple header + rows table to w.
func Table(w io.Writer, header []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Fprintln(w, "(no results)")
		return
	}
	safeHeader := make([]string, len(header))
	for i, cell := range header {
		safeHeader[i] = TerminalSafe(cell)
	}
	safeRows := make([][]string, len(rows))
	for i, row := range rows {
		safeRows[i] = make([]string, len(row))
		for j, cell := range row {
			safeRows[i][j] = TerminalSafe(cell)
		}
	}

	t := tablewriter.NewTable(w, tableOptions(safeHeader, safeRows)...)
	t.Header(safeHeader)
	t.Bulk(safeRows)
	t.Render()
}

// tableOptions bounds column widths to the terminal width, so long
// unstructured content (raw snippet/URL text, which often has little or no
// whitespace to word-wrap on) is force-wrapped by tablewriter into properly
// bordered multi-line cells, rather than left full width and hard-wrapped
// mid-row by the terminal itself — which garbles the box-drawing.
//
// Only columns that actually need shrinking are capped: each column gets a
// fair (equal) share of the available width, and columns that already fit
// their share keep their natural size, so short columns (ids, booleans,
// timestamps) aren't squeezed just because one column (e.g. SNIPPET) needs
// more room.
func tableOptions(header []string, rows [][]string) []tablewriter.Option {
	width, ok := terminalWidth()
	if !ok {
		return nil
	}
	return tableOptionsForWidth(width, header, rows)
}

// tableOptionsForWidth is the width-parameterized half of tableOptions,
// split out so the column-allocation logic can be tested without a real
// terminal.
func tableOptionsForWidth(width int, header []string, rows [][]string) []tablewriter.Option {
	numCols := len(header)
	if numCols == 0 {
		return nil
	}

	natural := make([]int, numCols)
	for i, h := range header {
		natural[i] = utf8.RuneCountInString(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i >= numCols {
				continue
			}
			if n := utf8.RuneCountInString(cell); n > natural[i] {
				natural[i] = n
			}
		}
	}

	const overheadPerCol = 3 // 1 char border + 1 char padding each side
	budget := width - (numCols*overheadPerCol + 1)
	if budget <= 0 {
		return nil
	}
	total := 0
	for _, n := range natural {
		total += n
	}
	if total <= budget {
		return nil // fits as-is
	}

	alloc := fairShareWidths(natural, budget)
	perColumn := make(tw.Mapper[int, int])
	for i, c := range alloc {
		if c < natural[i] {
			// Never cap a column below its own header text — better to
			// slightly overrun the budget than mangle the column title.
			if h := utf8.RuneCountInString(header[i]); c < h {
				c = h
			}
			perColumn[i] = c
		}
	}
	if len(perColumn) == 0 {
		return nil
	}
	return []tablewriter.Option{tablewriter.WithConfig(tablewriter.Config{
		Row: tw.CellConfig{
			Formatting:   tw.CellFormatting{AutoWrap: tw.WrapBreak},
			ColMaxWidths: tw.CellWidth{PerColumn: perColumn},
		},
	})}
}

// fairShareWidths distributes budget across columns by max-min fairness:
// process columns narrowest-first, giving each its natural width as long as
// that's no more than an equal split of what's left; once a column wants
// more than its equal split, it and every wider column left split the
// remaining budget evenly.
func fairShareWidths(natural []int, budget int) []int {
	n := len(natural)
	alloc := make([]int, n)
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool { return natural[order[a]] < natural[order[b]] })

	remaining := budget
	for idx, i := range order {
		share := remaining / (n - idx)
		if natural[i] > share {
			rest := order[idx:]
			base, extra := remaining/len(rest), remaining%len(rest)
			for k, j := range rest {
				alloc[j] = base
				if k < extra {
					alloc[j]++
				}
			}
			return alloc
		}
		alloc[i] = natural[i]
		remaining -= natural[i]
	}
	return alloc
}

// terminalWidth reports the width of the controlling terminal, so wide
// tables wrap into properly-bordered multi-line cells (tablewriter's own
// wrapping) instead of being hard-wrapped mid-row by the terminal itself,
// which garbles the box-drawing. It's read from stdout specifically since
// that's what a human is actually looking at; Table's own w is often a
// bytes.Buffer used for audit-log capture, where a width doesn't apply.
func terminalWidth() (int, bool) {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		return 0, false
	}
	width, _, err := term.GetSize(fd)
	if err != nil || width <= 0 {
		return 0, false
	}
	return width, true
}

func isEmailStr(p *bool) string {
	if p == nil {
		return ""
	}
	if *p {
		return "email"
	}
	return "username"
}

// Leaks renders a PaginatedLeaksResponse as either JSON or a table.
func Leaks(w io.Writer, resp *api.PaginatedLeaksResponse, jsonOut bool) error {
	if jsonOut {
		return JSON(w, resp)
	}
	header := []string{"USERNAME/EMAIL", "TYPE", "PASSWORD", "URL", "STRENGTH", "UNLOCKED", "STATUS"}
	rows := make([][]string, 0, len(resp.Items))
	for _, it := range resp.Items {
		ident := it.Username
		if ident == "" {
			ident = it.UsernameMasked
		}
		rows = append(rows, []string{
			ident, isEmailStr(it.IsEmail), it.Password, it.URL,
			strconv.Itoa(it.PasswordStrength), strconv.FormatBool(it.Unlocked), it.Status,
		})
	}
	Table(w, header, rows)
	fmt.Fprintf(w, "\ntotal: %d  unlocked: %d  page: %d  page_size: %d\n", resp.Total, resp.TotalUnlocked, resp.Page, resp.PageSize)
	if resp.AutoUnlockPointsSpent > 0 {
		fmt.Fprintf(w, "auto-unlock points spent: %d\n", resp.AutoUnlockPointsSpent)
	}
	if resp.BlacklistedValue != "" {
		fmt.Fprintf(w, "blacklisted value matched: %s\n", TerminalSafe(resp.BlacklistedValue))
	}
	return nil
}

// AllLeaks renders a PaginatedAllLeaksResponse (the /search/domain/{domain}/all
// endpoint), which additionally carries a Category per item.
func AllLeaks(w io.Writer, resp *api.PaginatedAllLeaksResponse, jsonOut bool) error {
	if jsonOut {
		return JSON(w, resp)
	}
	header := []string{"USERNAME/EMAIL", "TYPE", "CATEGORY", "PASSWORD", "URL", "STRENGTH", "UNLOCKED", "STATUS"}
	rows := make([][]string, 0, len(resp.Items))
	for _, it := range resp.Items {
		ident := it.Username
		if ident == "" {
			ident = it.UsernameMasked
		}
		rows = append(rows, []string{
			ident, isEmailStr(it.IsEmail), it.Category, it.Password, it.URL,
			strconv.Itoa(it.PasswordStrength), strconv.FormatBool(it.Unlocked), it.Status,
		})
	}
	Table(w, header, rows)
	fmt.Fprintf(w, "\ntotal: %d  unlocked: %d  page: %d  page_size: %d\n", resp.Total, resp.TotalUnlocked, resp.Page, resp.PageSize)
	return nil
}

// LeakSlice renders a bare []LeakDetails (used by unlock responses).
func LeakSlice(w io.Writer, items []api.LeakDetails, jsonOut bool) error {
	if jsonOut {
		return JSON(w, items)
	}
	header := []string{"USERNAME/EMAIL", "TYPE", "PASSWORD", "URL", "STRENGTH", "STATUS"}
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		ident := it.Username
		if ident == "" {
			ident = it.UsernameMasked
		}
		rows = append(rows, []string{ident, isEmailStr(it.IsEmail), it.Password, it.URL, strconv.Itoa(it.PasswordStrength), it.Status})
	}
	Table(w, header, rows)
	fmt.Fprintf(w, "\nunlocked %d record(s)\n", len(items))
	return nil
}

// LeakSliceWithCategory renders unlock results gathered across multiple
// domain categories in one command (`unlock domain <domain>` with the
// default --category all), tagging each row with which category it came
// from. The unlock endpoint itself has no combined-category response of
// its own — only employees/customers/third_parties are documented leak
// types for unlocking — so the CLI makes one call per category and merges
// the results here.
func LeakSliceWithCategory(w io.Writer, items []api.AllLeakDetails, jsonOut bool) error {
	if jsonOut {
		return JSON(w, items)
	}
	header := []string{"USERNAME/EMAIL", "TYPE", "CATEGORY", "PASSWORD", "URL", "STRENGTH", "STATUS"}
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		ident := it.Username
		if ident == "" {
			ident = it.UsernameMasked
		}
		rows = append(rows, []string{ident, isEmailStr(it.IsEmail), it.Category, it.Password, it.URL, strconv.Itoa(it.PasswordStrength), it.Status})
	}
	Table(w, header, rows)
	fmt.Fprintf(w, "\nunlocked %d record(s)\n", len(items))
	return nil
}

func strengthRow(label string, s *api.PasswordStrengthSummary) []string {
	if s == nil {
		return []string{label, "-", "-", "-", "-", "-"}
	}
	return []string{
		label, strconv.Itoa(s.TotalPass),
		fmt.Sprintf("%d (%.1f%%)", s.TooWeak.Qty, s.TooWeak.Perc),
		fmt.Sprintf("%d (%.1f%%)", s.Weak.Qty, s.Weak.Perc),
		fmt.Sprintf("%d (%.1f%%)", s.Medium.Qty, s.Medium.Perc),
		fmt.Sprintf("%d (%.1f%%)", s.Strong.Qty, s.Strong.Perc),
	}
}

// DomainReportView combines the summary counts from DomainSearchResponse
// with unlocked counts per category. The domain-report endpoint itself
// doesn't return unlocked counts (see DomainSearchResponse) — the CLI
// fetches them separately (one lightweight call per category) and merges
// them here, skipped entirely in --light mode.
type DomainReportView struct {
	*api.DomainSearchResponse
	UnlockedEmployees    *int `json:"unlocked_employees,omitempty"`
	UnlockedCustomers    *int `json:"unlocked_customers,omitempty"`
	UnlockedThirdParties *int `json:"unlocked_third_parties,omitempty"`
}

// DomainReport renders a DomainReportView.
func DomainReport(w io.Writer, domain string, view DomainReportView, jsonOut bool) error {
	if jsonOut {
		return JSON(w, view)
	}
	fmt.Fprintf(w, "domain: %s\n\n", TerminalSafe(domain))

	header := []string{"CATEGORY", "COMPROMISED", "UNLOCKED"}
	rows := [][]string{
		{"employees", strconv.Itoa(view.EmployeesCompromised), intPtrStr(view.UnlockedEmployees)},
		{"customers", strconv.Itoa(view.CustomersCompromised), intPtrStr(view.UnlockedCustomers)},
		{"third_parties", strconv.Itoa(view.ThirdPartiesCompromised), intPtrStr(view.UnlockedThirdParties)},
	}
	Table(w, header, rows)

	if view.EmployeePasswords != nil || view.CustomerPasswords != nil || view.ThirdPartiesPasswords != nil {
		fmt.Fprintln(w, "\npassword strength:")
		pwHeader := []string{"CATEGORY", "ANALYSED", "TOO_WEAK", "WEAK", "MEDIUM", "STRONG"}
		pwRows := [][]string{
			strengthRow("employees", view.EmployeePasswords),
			strengthRow("customers", view.CustomerPasswords),
			strengthRow("third_parties", view.ThirdPartiesPasswords),
		}
		Table(w, pwHeader, pwRows)
	}

	if view.CountsApproximate {
		fmt.Fprintln(w, "\nnote: counts are sampled/approximate (light mode). Authenticate or omit --light for exact counts.")
	}
	if view.SearchedByCount > 0 {
		fmt.Fprintf(w, "searched by %d other user(s) in the last 7 days\n", view.SearchedByCount)
	}
	return nil
}

// ComboLeaks renders a CombolistSearchResponse — the dedicated combolist
// dataset (separate from the main leak corpus). It has no URL/leak-name
// data, so the columns are narrower than Leaks/LeakSlice.
func ComboLeaks(w io.Writer, resp *api.CombolistSearchResponse, jsonOut bool) error {
	if jsonOut {
		return JSON(w, resp)
	}
	header := []string{"USERNAME/EMAIL", "TYPE", "PASSWORD", "EMAIL DOMAIN", "STRENGTH", "UNLOCKED", "STATUS"}
	rows := make([][]string, 0, len(resp.Items))
	for _, it := range resp.Items {
		ident := it.Username
		if ident == "" {
			ident = it.UsernameMasked
		}
		typ := "username"
		if it.IsEmail {
			typ = "email"
		}
		rows = append(rows, []string{ident, typ, it.Password, it.EmailDomain, strconv.Itoa(it.PasswordStrength), strconv.FormatBool(it.Unlocked), it.Status})
	}
	Table(w, header, rows)
	fmt.Fprintf(w, "\ntotal: %d  unlocked: %d  page: %d  page_size: %d\n", resp.Total, resp.TotalUnlocked, resp.Page, resp.PageSize)
	if resp.AutoUnlockPointsSpent > 0 {
		fmt.Fprintf(w, "auto-unlock points spent: %d\n", resp.AutoUnlockPointsSpent)
	}
	if resp.BlacklistedValue != "" {
		fmt.Fprintf(w, "blacklisted value matched: %s\n", TerminalSafe(resp.BlacklistedValue))
	}
	return nil
}

// ComboDomainReport renders a CombolistDomainReportResponse.
func ComboDomainReport(w io.Writer, resp *api.CombolistDomainReportResponse, jsonOut bool) error {
	if jsonOut {
		return JSON(w, resp)
	}
	fmt.Fprintf(w, "domain: %s (combolist dataset)\n\n", TerminalSafe(resp.Domain))
	fmt.Fprintf(w, "total credentials: %d\n", resp.TotalCredentials)
	fmt.Fprintf(w, "unique emails:     %d\n", resp.UniqueEmails)
	if resp.FirstSeen != "" {
		fmt.Fprintf(w, "first seen:        %s\n", TerminalSafe(resp.FirstSeen))
	}
	if resp.LastSeen != "" {
		fmt.Fprintf(w, "last seen:         %s\n", TerminalSafe(resp.LastSeen))
	}
	s := resp.PasswordStrength
	fmt.Fprintln(w, "\npassword strength:")
	Table(w, []string{"TOO_WEAK", "WEAK", "MEDIUM", "STRONG"}, [][]string{
		{strconv.Itoa(s.TooWeak), strconv.Itoa(s.Weak), strconv.Itoa(s.Medium), strconv.Itoa(s.Strong)},
	})
	return nil
}

// RawSearch renders a RawSearchResponse — locations inside raw breach
// files, not structured credentials.
func RawSearch(w io.Writer, resp *api.RawSearchResponse, jsonOut bool) error {
	if jsonOut {
		return JSON(w, resp)
	}
	header := []string{"CONTAINER", "FILE", "CATEGORY", "EXT", "INGESTED", "UNLOCKED", "SNIPPET"}
	rows := make([][]string, 0, len(resp.Items))
	for _, it := range resp.Items {
		name := it.DisplayName
		if name == "" {
			name = it.EntryPath
		}
		rows = append(rows, []string{it.ContainerID, name, it.Category, it.Ext, it.IngestedAt, strconv.FormatBool(it.AlreadyUnlocked), it.Snippet})
	}
	Table(w, header, rows)
	if resp.Total >= 0 {
		fmt.Fprintf(w, "\ntotal: %d  page: %d  page_size: %d\n", resp.Total, resp.Page, resp.PageSize)
	} else {
		fmt.Fprintf(w, "\npage: %d  page_size: %d (cursor mode, no total count)\n", resp.Page, resp.PageSize)
	}
	if resp.NextCursor != "" {
		fmt.Fprintf(w, "next cursor: %s\n", TerminalSafe(resp.NextCursor))
	}
	if resp.BlacklistedValue != "" {
		fmt.Fprintf(w, "blacklisted value matched: %s\n", TerminalSafe(resp.BlacklistedValue))
	}
	return nil
}

// RawCount renders a RawCountResponse.
func RawCount(w io.Writer, resp *api.RawCountResponse, jsonOut bool) error {
	if jsonOut {
		return JSON(w, resp)
	}
	fmt.Fprintf(w, "total: %d  exact: %v  capped: %v\n", resp.Total, resp.Exact, resp.Capped)
	if resp.BlacklistedValue != "" {
		fmt.Fprintf(w, "blacklisted value matched: %s\n", TerminalSafe(resp.BlacklistedValue))
	}
	return nil
}

// DomainSubdomains renders the subdomains list endpoint.
func DomainSubdomains(w io.Writer, resp *api.DomainSubdomainsResponse, jsonOut bool) error {
	if jsonOut {
		return JSON(w, resp)
	}
	header := []string{"SUBDOMAIN", "OCCURRENCES"}
	rows := make([][]string, 0, len(resp.Items))
	for _, it := range resp.Items {
		rows = append(rows, []string{it.Subdomain, strconv.Itoa(it.Occurrences)})
	}
	Table(w, header, rows)
	fmt.Fprintf(w, "\ntotal: %d  page: %d  page_size: %d\n", resp.Total, resp.Page, resp.PageSize)
	return nil
}

// DomainURLs renders the urls list endpoint.
func DomainURLs(w io.Writer, resp *api.DomainURLsResponse, jsonOut bool) error {
	if jsonOut {
		return JSON(w, resp)
	}
	header := []string{"URL", "OCCURRENCES"}
	rows := make([][]string, 0, len(resp.Items))
	for _, it := range resp.Items {
		rows = append(rows, []string{it.URL, strconv.Itoa(it.Occurrences)})
	}
	Table(w, header, rows)
	fmt.Fprintf(w, "\ntotal: %d  page: %d  page_size: %d\n", resp.Total, resp.Page, resp.PageSize)
	return nil
}

// DarkWeb renders a DarkWebSearchResponse.
func DarkWeb(w io.Writer, resp *api.DarkWebSearchResponse, jsonOut bool) error {
	if jsonOut {
		return JSON(w, resp)
	}
	header := []string{"TITLE", "AUTHOR", "SOURCE", "PUBLISHED", "TARGET"}
	rows := make([][]string, 0, len(resp.Items))
	for _, it := range resp.Items {
		source := it.SourceName
		if source == "" {
			source = it.Source
		}
		rows = append(rows, []string{it.Title, it.Author, source, it.PublishedAt, it.Target})
	}
	Table(w, header, rows)
	fmt.Fprintf(w, "\ntotal: %d  page: %d  page_size: %d\n", resp.Total, resp.Page, resp.PageSize)
	return nil
}

// PasswordRange renders a PasswordRangeResponse.
func PasswordRange(w io.Writer, resp *api.PasswordRangeResponse, jsonOut bool) error {
	if jsonOut {
		return JSON(w, resp)
	}
	header := []string{"HASH SUFFIX", "COUNT"}
	rows := make([][]string, 0, len(resp.Hashes))
	for _, e := range resp.Hashes {
		rows = append(rows, []string{e.Hash, strconv.Itoa(e.Count)})
	}
	fmt.Fprintf(w, "prefix: %s  total: %d\n\n", TerminalSafe(resp.Prefix), resp.Total)
	Table(w, header, rows)
	return nil
}

// EmailMassResults renders a batch emails locked-exists check.
func EmailMassResults(w io.Writer, results []api.EmailMassResult, jsonOut bool) error {
	if jsonOut {
		return JSON(w, results)
	}
	header := []string{"EMAIL/USERNAME", "ANY MATCH", "TOTAL", "LOCKED", "UNLOCKED"}
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		rows = append(rows, []string{r.Email, strconv.FormatBool(r.Any), intPtrStr(r.Total), intPtrStr(r.Locked), intPtrStr(r.Unlocked)})
	}
	Table(w, header, rows)
	return nil
}

// DomainMassResults renders a batch domains locked-exists check.
func DomainMassResults(w io.Writer, results []api.DomainMassResult, jsonOut bool) error {
	if jsonOut {
		return JSON(w, results)
	}
	header := []string{"DOMAIN", "ANY MATCH", "EMPLOYEES", "CUSTOMERS", "THIRD_PARTIES"}
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		rows = append(rows, []string{
			r.Domain, strconv.FormatBool(r.Any),
			categoryCell(r.Categories["employees"]),
			categoryCell(r.Categories["customers"]),
			categoryCell(r.Categories["third_parties"]),
		})
	}
	Table(w, header, rows)
	return nil
}

func categoryCell(c api.CategoryInfo) string {
	if !c.HasLocked && c.Total == nil && c.Count == nil {
		return "-"
	}
	return fmt.Sprintf("total=%s locked=%v", intPtrStr(c.Total), c.HasLocked)
}

func intPtrStr(p *int) string {
	if p == nil {
		return "-"
	}
	return strconv.Itoa(*p)
}

// BalanceView combines the points balance (from Profile) with live plan
// quota usage (from PlanUsageResponse) into one printable summary.
type BalanceView struct {
	Points          int                    `json:"points"`
	ExtraPoints     int                    `json:"extra_points"`
	TotalPoints     int                    `json:"total_points"`
	PlanName        string                 `json:"plan_name"`
	SubscriptionEnd string                 `json:"subscription_end_date,omitempty"`
	Usage           *api.PlanUsageResponse `json:"usage,omitempty"`
}

func usageStr(m api.UsageMetric) string {
	used := "?"
	if m.Used != nil {
		used = strconv.Itoa(*m.Used)
	}
	if m.Limit == nil {
		return used + "/unlimited"
	}
	return fmt.Sprintf("%s/%d", used, *m.Limit)
}

// Balance renders a BalanceView — unlock credits (points) plus, when
// available, live daily-request and unlocked-lists quota usage.
func Balance(w io.Writer, b BalanceView, jsonOut bool) error {
	if jsonOut {
		return JSON(w, b)
	}
	fmt.Fprintf(w, "unlock credits: %d (%d subscription + %d extra)\n", b.TotalPoints, b.Points, b.ExtraPoints)
	fmt.Fprintf(w, "plan:           %s\n", TerminalSafe(b.PlanName))
	if b.SubscriptionEnd != "" {
		fmt.Fprintf(w, "renews:         %s\n", TerminalSafe(b.SubscriptionEnd))
	}
	if b.Usage != nil {
		fmt.Fprintf(w, "daily requests: %s\n", usageStr(b.Usage.RequestsDaily))
		fmt.Fprintf(w, "unlocked lists: %s\n", usageStr(b.Usage.UnlockedLists))
	}
	return nil
}

// Profile renders the account profile summary.
func Profile(w io.Writer, p *api.Profile, jsonOut bool) error {
	if jsonOut {
		return JSON(w, p)
	}
	fmt.Fprintf(w, "account:       %s (%s)\n", TerminalSafe(p.Email), TerminalSafe(p.Organization))
	fmt.Fprintf(w, "plan:          %s\n", TerminalSafe(p.Plan.PlanName))
	fmt.Fprintf(w, "active:        %v\n", p.SubscriptionActive)
	fmt.Fprintf(w, "points:        %d (+%d extra)\n", p.SubscriptionPoints, p.ExtraPoints)
	fmt.Fprintf(w, "daily cap:     %d req/day\n", p.Plan.DailyRequestCap)
	fmt.Fprintf(w, "renews:        %s\n", TerminalSafe(p.SubscriptionEndDate))
	fmt.Fprintf(w, "features:      email=%v domain=%v advanced=%v raw=%v combolist=%v darkweb=%v\n",
		p.Plan.EmailSearch, p.Plan.DomainSearch, p.Plan.AdvancedSearch, p.Plan.RawSearch, p.Plan.ComboListSearch, p.Plan.DarkWebSearch)
	if p.Banned {
		fmt.Fprintln(w, "WARNING: this account is banned")
	}
	return nil
}

// Exports renders a paginated list of export jobs.
func Exports(w io.Writer, resp *api.ExportListResponse, jsonOut bool) error {
	if jsonOut {
		return JSON(w, resp)
	}
	header := []string{"ID", "STATUS", "TYPE", "FILENAME", "CREATED"}
	rows := make([][]string, 0, len(resp.Items))
	for _, e := range resp.Items {
		rows = append(rows, []string{strconv.Itoa(e.ID), e.Status, e.Type, e.Filename, e.Timestamp})
	}
	Table(w, header, rows)
	fmt.Fprintf(w, "\ntotal: %d  page: %d  page_size: %d\n", resp.Total, resp.Page, resp.PageSize)
	return nil
}

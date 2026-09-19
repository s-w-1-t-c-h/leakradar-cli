package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"leakradar-cli/internal/api"
	"leakradar-cli/internal/output"
	"leakradar-cli/internal/record"
)

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock (reveal) matching records, spending account points",
	Long: "Unlocking spends points from your LeakRadar plan. Every unlock subcommand previews how many " +
		"records would actually be newly unlocked (and compares that against your current balance) before " +
		"prompting for confirmation, unless --yes is passed.",
}

// confirm asks the user to type 'y' before a points-spending action, unless
// yes is true (needed for non-interactive/automation use).
func confirm(yes bool, prompt string) (bool, error) {
	if yes {
		return true, nil
	}
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes", nil
}

// previewUnlock runs before any unlock spends points: it reports how many
// matching records exist, how many are already unlocked (free to re-view)
// vs locked (would newly cost points), what --max actually caps that at,
// and the account's current balance — then asks for confirmation. Returns
// proceed=false with no error when there's nothing to unlock, so the caller
// can just return nil without ever hitting the unlock endpoint.
func previewUnlock(c *api.Client, yes bool, label string, max, total, totalUnlocked int) (bool, error) {
	locked := total - totalUnlocked
	if locked <= 0 {
		fmt.Fprintf(os.Stderr, "%s: %d matching record(s), all already unlocked — nothing to spend points on.\n", label, total)
		return false, nil
	}

	willUnlock := locked
	maxDesc := "no cap set (will use as many points as needed)"
	if max > 0 {
		maxDesc = strconv.Itoa(max)
		if max < willUnlock {
			willUnlock = max
		}
	}
	fmt.Fprintf(os.Stderr, "%s: %d matching record(s) (%d already unlocked free, %d locked).\n", label, total, totalUnlocked, locked)
	fmt.Fprintf(os.Stderr, "--max: %s -> up to %d record(s) may be newly unlocked (~1 point each typically; actual cost can vary by plan/type).\n", maxDesc, willUnlock)

	if p, err := c.GetProfile(cmdContext()); err == nil {
		balance := p.SubscriptionPoints + p.ExtraPoints
		fmt.Fprintf(os.Stderr, "current balance: %d points.\n", balance)
		if willUnlock > balance {
			fmt.Fprintln(os.Stderr, "WARNING: this could exceed your remaining balance.")
		}
	} else if flagVerbose {
		fmt.Fprintf(os.Stderr, "warning: could not fetch balance for comparison: %s\n", output.TerminalSafe(err.Error()))
	}

	ok, err := confirm(yes, "Proceed?")
	if err != nil {
		return false, err
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "aborted.")
	}
	return ok, nil
}

// --- unlock email ---

var unlockEmailOpts struct {
	search              string
	isEmail, isUsername bool
	max, listID         int
	async, yes          bool
}

var unlockEmailCmd = &cobra.Command{
	Use:   "email <email-or-username>",
	Short: "Unlock email/username search results",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		isEmail, err := isEmailPtr(unlockEmailOpts.isEmail, unlockEmailOpts.isUsername)
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		req := api.EmailSearchRequest{Email: args[0], Search: unlockEmailOpts.search, IsEmail: isEmail}

		preview, err := c.SearchEmail(cmdContext(), req)
		if err != nil {
			return err
		}
		proceed, err := previewUnlock(c, unlockEmailOpts.yes, fmt.Sprintf("email %q", args[0]), unlockEmailOpts.max, preview.Total, preview.TotalUnlocked)
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}

		params := api.UnlockParams{Max: unlockEmailOpts.max, ListID: unlockEmailOpts.listID}

		if unlockEmailOpts.async {
			task, err := c.EmailUnlockTask(cmdContext(), req, params)
			if err != nil {
				return err
			}
			rec().Record(record.EmailPath(args[0], "unlock_task.json"), "email-unlock-task", args[0], redactedCommandLine(), task, "")
			return printTaskResult(task)
		}
		items, err := c.EmailUnlock(cmdContext(), req, params)
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.LeakSlice(w, items, false) })
		rec().Record(record.EmailPath(args[0], "unlock.json"), "email-unlock", args[0], redactedCommandLine(), items, text)
		return output.LeakSlice(os.Stdout, items, jsonOut())
	},
}

// --- unlock domain ---

var unlockDomainOpts struct {
	category            string
	search              string
	isEmail, isUsername bool
	max, listID         int
	async, yes          bool
}

var unlockDomainCmd = &cobra.Command{
	Use:   "domain <domain>",
	Short: "Unlock domain-scoped search results for one category, or all three",
	Long: "With --category all (the default), this unlocks employees, customers, and third_parties in turn, " +
		"sharing one --max budget across all three, and tags each result with its category. The unlock endpoint " +
		"itself only documents employees/customers/third_parties as valid categories to unlock — 'all' is a " +
		"convenience the CLI implements as three separate calls, not something the API does in one request.",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		lt, err := parseLeakType(unlockDomainOpts.category)
		if err != nil {
			return err
		}
		isEmail, err := isEmailPtr(unlockDomainOpts.isEmail, unlockDomainOpts.isUsername)
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}

		if lt == api.LeakTypeAll {
			return unlockDomainAll(c, args[0], unlockDomainOpts.search, isEmail, unlockDomainOpts.max, unlockDomainOpts.listID, unlockDomainOpts.async, unlockDomainOpts.yes)
		}

		previewParams := api.DomainListParams{Page: 1, PageSize: 1, Search: unlockDomainOpts.search, IsEmail: isEmail}
		resp, err := c.DomainList(cmdContext(), args[0], lt, previewParams)
		if err != nil {
			return err
		}
		proceed, err := previewUnlock(c, unlockDomainOpts.yes, fmt.Sprintf("%s records for %q", lt, args[0]), unlockDomainOpts.max, resp.Total, resp.TotalUnlocked)
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}

		p := api.DomainUnlockParams{Search: unlockDomainOpts.search, IsEmail: isEmail, Max: unlockDomainOpts.max, ListID: unlockDomainOpts.listID}

		if unlockDomainOpts.async {
			task, err := c.DomainUnlockTask(cmdContext(), args[0], lt, p)
			if err != nil {
				return err
			}
			rec().Record(record.DomainPath(args[0], "unlock_"+string(lt)+"_task.json"), "domain-unlock-task", args[0], redactedCommandLine(), task, "")
			return printTaskResult(task)
		}
		items, err := c.DomainUnlock(cmdContext(), args[0], lt, p)
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.LeakSlice(w, items, false) })
		rec().Record(record.DomainPath(args[0], "unlock_"+string(lt)+".json"), "domain-unlock", args[0], redactedCommandLine(), items, text)
		return output.LeakSlice(os.Stdout, items, jsonOut())
	},
}

// domainUnlockCategories is the order unlockDomainAll walks categories in
// when sharing one --max budget across all three.
var domainUnlockCategories = []api.LeakType{api.LeakTypeEmployees, api.LeakTypeCustomers, api.LeakTypeThirdParties}

// categoryLockStatus is one category's match/lock counts, used to preview
// and then drive an --category all unlock.
type categoryLockStatus struct {
	category      api.LeakType
	total, locked int
}

// unlockDomainAll previews and, on confirmation, executes an unlock across
// employees/customers/third_parties — one real per-category call each
// (see unlockDomainCmd's Long text for why), sharing a single --max budget
// in that order and tagging each result with the category it came from.
func unlockDomainAll(c *api.Client, domain, search string, isEmail *bool, max, listID int, async, yes bool) error {
	var statuses []categoryLockStatus
	for _, lt := range domainUnlockCategories {
		resp, err := c.DomainList(cmdContext(), domain, lt, api.DomainListParams{Page: 1, PageSize: 1, Search: search, IsEmail: isEmail})
		if err != nil {
			return err
		}
		statuses = append(statuses, categoryLockStatus{category: lt, total: resp.Total, locked: resp.Total - resp.TotalUnlocked})
	}

	totalLocked := 0
	for _, s := range statuses {
		totalLocked += s.locked
	}
	if totalLocked <= 0 {
		fmt.Fprintf(os.Stderr, "all categories for %q: nothing locked in any category — nothing to spend points on.\n", domain)
		return nil
	}

	fmt.Fprintf(os.Stderr, "all categories for %q:\n", domain)
	for _, s := range statuses {
		fmt.Fprintf(os.Stderr, "  %-14s %d matching (%d locked)\n", s.category, s.total, s.locked)
	}
	fmt.Fprintln(os.Stderr, "(pass --category employees|customers|third_parties to unlock just one of these instead of all three)")
	willUnlock := totalLocked
	maxDesc := "no cap set (will use as many points as needed)"
	if max > 0 {
		maxDesc = strconv.Itoa(max)
		if max < willUnlock {
			willUnlock = max
		}
	}
	fmt.Fprintf(os.Stderr, "--max: %s -> up to %d record(s) may be newly unlocked, spent employees first then customers then third_parties (~1 point each typically).\n", maxDesc, willUnlock)
	if p, err := c.GetProfile(cmdContext()); err == nil {
		balance := p.SubscriptionPoints + p.ExtraPoints
		fmt.Fprintf(os.Stderr, "current balance: %d points.\n", balance)
		if willUnlock > balance {
			fmt.Fprintln(os.Stderr, "WARNING: this could exceed your remaining balance.")
		}
	} else if flagVerbose {
		fmt.Fprintf(os.Stderr, "warning: could not fetch balance for comparison: %s\n", output.TerminalSafe(err.Error()))
	}
	ok, err := confirm(yes, "Proceed?")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "aborted.")
		return nil
	}

	remaining := max
	if async {
		for _, s := range statuses {
			if s.locked <= 0 || (max > 0 && remaining <= 0) {
				continue
			}
			perCallMax := 0
			if max > 0 {
				perCallMax = remaining
			}
			task, err := c.DomainUnlockTask(cmdContext(), domain, s.category, api.DomainUnlockParams{Search: search, IsEmail: isEmail, Max: perCallMax, ListID: listID})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "-- %s --\n", s.category)
			rec().Record(record.DomainPath(domain, "unlock_"+string(s.category)+"_task.json"), "domain-unlock-task", domain, redactedCommandLine(), task, "")
			if err := printTaskResult(task); err != nil {
				return err
			}
			if max > 0 {
				// Async tasks don't report a count up front, so this is a
				// conservative estimate: assume the cap requested was used.
				remaining -= perCallMax
			}
		}
		return nil
	}

	var all []api.AllLeakDetails
	for _, s := range statuses {
		if s.locked <= 0 || (max > 0 && remaining <= 0) {
			continue
		}
		perCallMax := 0
		if max > 0 {
			perCallMax = remaining
		}
		items, err := c.DomainUnlock(cmdContext(), domain, s.category, api.DomainUnlockParams{Search: search, IsEmail: isEmail, Max: perCallMax, ListID: listID})
		if err != nil {
			return err
		}
		for _, it := range items {
			all = append(all, api.AllLeakDetails{LeakDetails: it, Category: string(s.category)})
		}
		if max > 0 {
			remaining -= len(items)
		}
	}
	text := renderText(func(w io.Writer) error { return output.LeakSliceWithCategory(w, all, false) })
	rec().Record(record.DomainPath(domain, "unlock_all.json"), "domain-unlock-all", domain, redactedCommandLine(), all, text)
	return output.LeakSliceWithCategory(os.Stdout, all, jsonOut())
}

// --- unlock advanced ---

var unlockAdvancedOpts struct {
	max, listID int
	async, yes  bool
}

var unlockAdvancedCmd = &cobra.Command{
	Use:   "advanced",
	Short: "Unlock advanced multi-field search results",
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

		preview, err := c.SearchAdvanced(cmdContext(), filters)
		if err != nil {
			return err
		}
		proceed, err := previewUnlock(c, unlockAdvancedOpts.yes, "advanced-search matches", unlockAdvancedOpts.max, preview.Total, preview.TotalUnlocked)
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}

		params := api.UnlockParams{Max: unlockAdvancedOpts.max, ListID: unlockAdvancedOpts.listID}

		if unlockAdvancedOpts.async {
			task, err := c.AdvancedUnlockTask(cmdContext(), filters, params)
			if err != nil {
				return err
			}
			rec().Record(record.TimestampPath("advanced", "unlock_task"), "advanced-unlock-task", "", redactedCommandLine(), task, "")
			return printTaskResult(task)
		}
		items, err := c.AdvancedUnlock(cmdContext(), filters, params)
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.LeakSlice(w, items, false) })
		rec().Record(record.TimestampPath("advanced", "unlock"), "advanced-unlock", "", redactedCommandLine(), items, text)
		return output.LeakSlice(os.Stdout, items, jsonOut())
	},
}

func init() {
	unlockEmailCmd.Flags().StringVar(&unlockEmailOpts.search, "search", "", "free-text filter on URL or username")
	unlockEmailCmd.Flags().BoolVar(&unlockEmailOpts.isEmail, "is-email", false, "restrict to email matches only")
	unlockEmailCmd.Flags().BoolVar(&unlockEmailOpts.isUsername, "is-username", false, "restrict to username matches only")
	unlockEmailCmd.Flags().IntVar(&unlockEmailOpts.max, "max", 0, "max records to unlock (0 = no cap - uses as many points as needed)")
	unlockEmailCmd.Flags().IntVar(&unlockEmailOpts.listID, "list-id", 0, "assign unlocked records to this list ID")
	unlockEmailCmd.Flags().BoolVar(&unlockEmailOpts.async, "async", false, "queue an async unlock task instead of waiting synchronously")
	unlockEmailCmd.Flags().BoolVar(&unlockEmailOpts.yes, "yes", false, "skip the confirmation prompt (for scripts/CI)")

	unlockDomainCmd.Flags().StringVar(&unlockDomainOpts.category, "category", "all", "employees|customers|third_parties|all (all = 3 real calls, one per category, sharing one --max budget)")
	unlockDomainCmd.Flags().StringVar(&unlockDomainOpts.search, "search", "", "free-text filter")
	unlockDomainCmd.Flags().BoolVar(&unlockDomainOpts.isEmail, "is-email", false, "restrict to email matches only")
	unlockDomainCmd.Flags().BoolVar(&unlockDomainOpts.isUsername, "is-username", false, "restrict to username matches only")
	unlockDomainCmd.Flags().IntVar(&unlockDomainOpts.max, "max", 0, "max records to unlock (0 = no cap - uses as many points as needed)")
	unlockDomainCmd.Flags().IntVar(&unlockDomainOpts.listID, "list-id", 0, "assign unlocked records to this list ID")
	unlockDomainCmd.Flags().BoolVar(&unlockDomainOpts.async, "async", false, "queue an async unlock task instead of waiting synchronously")
	unlockDomainCmd.Flags().BoolVar(&unlockDomainOpts.yes, "yes", false, "skip the confirmation prompt (for scripts/CI)")

	registerAdvancedFlags(unlockAdvancedCmd.Flags())
	unlockAdvancedCmd.Flags().IntVar(&unlockAdvancedOpts.max, "max", 0, "max records to unlock (0 = no cap - uses as many points as needed)")
	unlockAdvancedCmd.Flags().IntVar(&unlockAdvancedOpts.listID, "list-id", 0, "assign unlocked records to this list ID")
	unlockAdvancedCmd.Flags().BoolVar(&unlockAdvancedOpts.async, "async", false, "queue an async unlock task instead of waiting synchronously")
	unlockAdvancedCmd.Flags().BoolVar(&unlockAdvancedOpts.yes, "yes", false, "skip the confirmation prompt (for scripts/CI)")

	unlockCmd.AddCommand(unlockEmailCmd, unlockDomainCmd, unlockAdvancedCmd)
	RootCmd.AddCommand(unlockCmd)
}

package commands

import (
	"errors"
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

// wrapRawFilterError adds a pointed hint when the API rejects a raw query
// for having no filters at all — the single most common mistake with this
// command (see 'leakradar-cli raw --help'). The API's own message already
// lists every valid filter, but --q specifically is what most people are
// missing, so it's called out by name rather than leaving them to re-read
// the full list.
func wrapRawFilterError(err error) error {
	var apiErr *api.Error
	if err == nil || !errors.As(err, &apiErr) {
		return err
	}
	if apiErr.StatusCode == 400 && strings.Contains(strings.ToLower(apiErr.APIError.Message()), "empty raw") {
		return fmt.Errorf("%w\nhint: did you forget --q? every raw command needs --q or one of --container-id/--ext/--category/--file-name/--folder-name — see 'leakradar-cli raw --help'", err)
	}
	return err
}

var rawCmd = &cobra.Command{
	Use:   "raw",
	Short: "Full-text search across raw (unparsed) breach file content",
	Long: "The raw dataset is full-text search across unparsed source files (stealer logs, database dumps, " +
		"combolists) at the block/line level — a much larger, noisier surface than the structured email/domain/" +
		"advanced or combolist datasets (a single-domain query can return tens of thousands of blocks). Every " +
		"subcommand shares the same filter flags; run 'leakradar-cli raw search --help' for the full list. " +
		"Unlocking is async-only, like combolist.\n\n" +
		"You must provide --q (search block content) or at least one of --container-id/--ext/--category/" +
		"--file-name (select by file metadata instead) — providing none of these is rejected by the API. " +
		"For 'find mentions of this domain/email', --q is what you want; the others are for browsing/filtering " +
		"by file rather than by content.",
}

// --- raw search ---

var rawSearchOpts struct {
	page, pageSize int
	cursor         string
	dedup          bool
}

var rawSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search raw breach file content",
	RunE: func(cmd *cobra.Command, args []string) error {
		req, err := buildRawSearchRequest()
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.SearchRaw(cmdContext(), req, rawSearchOpts.page, rawSearchOpts.pageSize, rawSearchOpts.cursor, rawSearchOpts.dedup)
		if err != nil {
			return wrapRawFilterError(err)
		}
		text := renderText(func(w io.Writer) error { return output.RawSearch(w, resp, false) })
		rec().Record(record.TimestampPath("raw", "search"), "raw-search", "", redactedCommandLine(), resp, text)
		return output.RawSearch(os.Stdout, resp, jsonOut())
	},
}

// --- raw count ---

var rawCountDedup bool

var rawCountCmd = &cobra.Command{
	Use:   "count",
	Short: "Cheap match count for the same filters as 'raw search' (1 point per part if unlocked)",
	RunE: func(cmd *cobra.Command, args []string) error {
		req, err := buildRawSearchRequest()
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.CountRaw(cmdContext(), req, rawCountDedup)
		if err != nil {
			return wrapRawFilterError(err)
		}
		text := renderText(func(w io.Writer) error { return output.RawCount(w, resp, false) })
		rec().Record(record.TimestampPath("raw", "count"), "raw-count", "", redactedCommandLine(), resp, text)
		return output.RawCount(os.Stdout, resp, jsonOut())
	},
}

// --- raw unlock ---

var rawUnlockOpts struct {
	max   int
	dedup bool
	yes   bool
}

var rawUnlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Queue an async unlock of raw blocks matching a search, spending account points",
	Long: "Always queues a background task — same reason as combolist (no synchronous 'unlock and print now' " +
		"call exists for search-driven unlocks here). Previews cost via 'raw count', which the API documents " +
		"as exactly 1 point per part (not the '~1 typically' hedge used for the other datasets).",
	RunE: func(cmd *cobra.Command, args []string) error {
		req, err := buildRawSearchRequest()
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		proceed, err := previewRawUnlock(c, rawUnlockOpts.yes, req, rawUnlockOpts.max, rawUnlockOpts.dedup)
		if err != nil {
			return wrapRawFilterError(err)
		}
		if !proceed {
			return nil
		}

		task, err := c.RawUnlockTask(cmdContext(), req, rawUnlockOpts.max, rawUnlockOpts.dedup)
		if err != nil {
			return wrapRawFilterError(err)
		}
		rec().Record(record.TimestampPath("raw", "unlock_task"), "raw-unlock-task", "", redactedCommandLine(), task, "")
		return printTaskResult(task)
	},
}

// previewRawUnlock mirrors previewUnlock's shape but is built on
// RawCountResponse, since raw's count endpoint directly documents the
// points relationship (1 point/part) rather than us inferring it.
func previewRawUnlock(c *api.Client, yes bool, req api.RawSearchRequest, max int, dedup bool) (bool, error) {
	count, err := c.CountRaw(cmdContext(), req, dedup)
	if err != nil {
		return false, err
	}
	if count.BlacklistedValue != "" {
		fmt.Fprintf(os.Stderr, "query matched a blacklist rule (%q) — nothing to unlock.\n", count.BlacklistedValue)
		return false, nil
	}
	if count.Total <= 0 {
		fmt.Fprintln(os.Stderr, "no matching raw blocks — nothing to unlock.")
		return false, nil
	}

	note := "exact count"
	switch {
	case count.Capped:
		note = "capped — actual total may be higher"
	case !count.Exact:
		note = "upper bound — actual total may be lower"
	}
	fmt.Fprintf(os.Stderr, "raw search: %d matching block(s) (%s); this already excludes blocks you've already unlocked.\n", count.Total, note)

	willUnlock := count.Total
	maxDesc := "no cap set (will use as many points as needed)"
	if max > 0 {
		maxDesc = strconv.Itoa(max)
		if max < willUnlock {
			willUnlock = max
		}
	}
	fmt.Fprintf(os.Stderr, "--max: %s -> up to %d part(s) may be newly unlocked, documented at 1 point each.\n", maxDesc, willUnlock)

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

// --- raw export ---

var rawExportOpts struct {
	mode  string
	dedup bool
}

var rawExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Queue an export of raw search results",
	Long: "--mode rows exports matching lines as CSV; --mode parts exports a text snapshot of matching blocks. " +
		"Only previously unlocked results are included — this does not unlock anything itself, so run " +
		"'leakradar-cli raw unlock' first for anything you want in the export.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if rawExportOpts.mode != "rows" && rawExportOpts.mode != "parts" {
			return fmt.Errorf("invalid --mode %q (must be rows or parts)", rawExportOpts.mode)
		}
		req, err := buildRawSearchRequest()
		if err != nil {
			return err
		}
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.RawExport(cmdContext(), req, rawExportOpts.mode, rawExportOpts.dedup)
		if err != nil {
			return wrapRawFilterError(err)
		}
		return printQueuedExport("raw-export-"+rawExportOpts.mode, "", resp)
	},
}

func init() {
	for _, c := range []*cobra.Command{rawSearchCmd, rawCountCmd, rawUnlockCmd, rawExportCmd} {
		registerRawFilterFlags(c.Flags())
	}

	rawSearchCmd.Flags().IntVar(&rawSearchOpts.page, "page", 1, "page number (ignored when --cursor is set)")
	rawSearchCmd.Flags().IntVar(&rawSearchOpts.pageSize, "page-size", 10, "results per page (max 100)")
	rawSearchCmd.Flags().StringVar(&rawSearchOpts.cursor, "cursor", "", "cursor from a previous response's next_cursor, for cursor-based pagination")
	rawSearchCmd.Flags().BoolVar(&rawSearchOpts.dedup, "dedup", false, "collapse blocks with identical content (per-page only)")

	rawCountCmd.Flags().BoolVar(&rawCountDedup, "dedup", false, "collapse blocks with identical content before counting")

	rawUnlockCmd.Flags().IntVar(&rawUnlockOpts.max, "max", 0, "max parts to unlock (0 = no cap - uses as many points as needed)")
	rawUnlockCmd.Flags().BoolVar(&rawUnlockOpts.dedup, "dedup", false, "collapse blocks with identical content before unlocking")
	rawUnlockCmd.Flags().BoolVar(&rawUnlockOpts.yes, "yes", false, "skip the confirmation prompt (for scripts/CI)")

	rawExportCmd.Flags().StringVar(&rawExportOpts.mode, "mode", "rows", "rows|parts (required)")
	rawExportCmd.Flags().BoolVar(&rawExportOpts.dedup, "dedup", false, "deduplicate the export")

	rawCmd.AddCommand(rawSearchCmd, rawCountCmd, rawUnlockCmd, rawExportCmd)
	RootCmd.AddCommand(rawCmd)
}

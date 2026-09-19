package commands

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"leakradar-cli/internal/api"
	"leakradar-cli/internal/output"
	"leakradar-cli/internal/record"
)

var dw struct {
	query, title, content, author, sourceURL     string
	sources, dateFrom, dateTo, sortBy, sortOrder string
	page, pageSize                               int
}

var darkwebCmd = &cobra.Command{
	Use:   "darkweb",
	Short: "Search dark-web forum/Telegram posts",
	Long:  "Use --query for a simple full-text search, or the --title/--content/--author/--source-url flags for field-specific search.",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := newClient()
		if err != nil {
			return err
		}
		resp, err := c.SearchDarkWeb(cmdContext(), api.DarkWebSearchRequest{
			Query: dw.query, Title: dw.title, Content: dw.content, Author: dw.author, SourceURL: dw.sourceURL,
		}, api.DarkWebSearchParams{
			Page: dw.page, PageSize: dw.pageSize, Sources: dw.sources,
			DateFrom: dw.dateFrom, DateTo: dw.dateTo, SortBy: dw.sortBy, SortOrder: dw.sortOrder,
		})
		if err != nil {
			return err
		}
		text := renderText(func(w io.Writer) error { return output.DarkWeb(w, resp, false) })
		rec().Record(record.TimestampPath("darkweb", "search"), "darkweb-search", "", redactedCommandLine(), resp, text)
		return output.DarkWeb(os.Stdout, resp, jsonOut())
	},
}

func init() {
	f := darkwebCmd.Flags()
	f.StringVar(&dw.query, "query", "", "simple full-text search")
	f.StringVar(&dw.title, "title", "", "field search: title")
	f.StringVar(&dw.content, "content", "", "field search: content")
	f.StringVar(&dw.author, "author", "", "field search: author")
	f.StringVar(&dw.sourceURL, "source-url", "", "field search: source URL")
	f.StringVar(&dw.sources, "sources", "", "comma-separated source filter")
	f.StringVar(&dw.dateFrom, "date-from", "", "ISO 8601 datetime lower bound")
	f.StringVar(&dw.dateTo, "date-to", "", "ISO 8601 datetime upper bound")
	f.StringVar(&dw.sortBy, "sort-by", "", "ingested_at|published_at (default ingested_at)")
	f.StringVar(&dw.sortOrder, "sort-order", "", "asc|desc (default desc)")
	f.IntVar(&dw.page, "page", 1, "page number")
	f.IntVar(&dw.pageSize, "page-size", 25, "results per page (max 100)")
	RootCmd.AddCommand(darkwebCmd)
}

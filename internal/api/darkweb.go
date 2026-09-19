package api

import "context"

// DarkWebSearchParams are the query-string params for POST /search/dark-web.
type DarkWebSearchParams struct {
	Page      int
	PageSize  int
	Sources   string // comma-separated
	DateFrom  string // ISO 8601
	DateTo    string // ISO 8601
	SortBy    string // "ingested_at" | "published_at"
	SortOrder string // "asc" | "desc"
}

// SearchDarkWeb calls POST /search/dark-web.
func (c *Client) SearchDarkWeb(ctx context.Context, body DarkWebSearchRequest, p DarkWebSearchParams) (*DarkWebSearchResponse, error) {
	q := newQuery()
	setInt(q, "page", p.Page)
	setInt(q, "page_size", p.PageSize)
	setStr(q, "sources", p.Sources)
	setStr(q, "date_from", p.DateFrom)
	setStr(q, "date_to", p.DateTo)
	setStr(q, "sort_by", p.SortBy)
	setStr(q, "sort_order", p.SortOrder)

	var out DarkWebSearchResponse
	err := c.doJSON(ctx, requestOpts{
		method: "POST",
		path:   "/search/dark-web",
		query:  q,
		body:   body,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

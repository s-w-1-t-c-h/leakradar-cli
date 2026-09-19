package api

import "context"

// SearchRaw calls POST /search/raw — full-text search across raw
// (unparsed) breach file content: stealer logs, database dumps, and
// combolists at the block/line level. A much larger, noisier surface than
// the structured email/domain/advanced or combolist datasets.
func (c *Client) SearchRaw(ctx context.Context, req RawSearchRequest, page, pageSize int, cursor string, dedup bool) (*RawSearchResponse, error) {
	q := newQuery()
	setInt(q, "page", page)
	setInt(q, "page_size", pageSize)
	setStr(q, "cursor", cursor)
	setBool(q, "dedup", dedup)

	var out RawSearchResponse
	err := c.doJSON(ctx, requestOpts{method: "POST", path: "/search/raw", query: q, body: req}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// CountRaw calls POST /search/raw/count — a cheap upper-bound cost
// estimate (documented as 1 point per part) for the same filters, meant to
// be checked before committing to an unlock.
func (c *Client) CountRaw(ctx context.Context, req RawSearchRequest, dedup bool) (*RawCountResponse, error) {
	q := newQuery()
	setBool(q, "dedup", dedup)

	var out RawCountResponse
	err := c.doJSON(ctx, requestOpts{method: "POST", path: "/search/raw/count", query: q, body: req}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// RawUnlockTask calls POST /search/raw/unlock/task. Raw unlocking by
// search criteria is async-only, like the rest of raw's write operations.
// Note: unlike the other datasets' unlock/task endpoints, there is no
// list_id support here.
func (c *Client) RawUnlockTask(ctx context.Context, req RawSearchRequest, max int, dedup bool) (AsyncTaskResult, error) {
	q := newQuery()
	setInt(q, "max", max)
	setBool(q, "dedup", dedup)

	var out AsyncTaskResult
	err := c.doJSON(ctx, requestOpts{method: "POST", path: "/search/raw/unlock/task", query: q, body: req}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// RawExport calls POST /search/raw/export, queuing an async export job.
// mode is "rows" (CSV of matching lines) or "parts" (text snapshot of
// matching parts) — required by the API, unlike the other export endpoints
// which default to csv.
func (c *Client) RawExport(ctx context.Context, req RawSearchRequest, mode string, dedup bool) (*ExportResponse, error) {
	q := newQuery()
	setStr(q, "export", mode)
	setBool(q, "dedup", dedup)

	var out ExportResponse
	err := c.doJSON(ctx, requestOpts{method: "POST", path: "/search/raw/export", query: q, body: req}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

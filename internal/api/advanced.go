package api

import "context"

// SearchAdvanced calls POST /search/advanced (rate-limited to 5 req/s by the API).
func (c *Client) SearchAdvanced(ctx context.Context, filters LeakSearchFilters) (*PaginatedLeaksResponse, error) {
	var out PaginatedLeaksResponse
	err := c.doJSON(ctx, requestOpts{
		method:      "POST",
		path:        "/search/advanced",
		body:        filters,
		useAdvanced: true,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// AdvancedUnlock calls POST /search/advanced/unlock, spending points.
func (c *Client) AdvancedUnlock(ctx context.Context, filters LeakSearchFilters, p UnlockParams) ([]LeakDetails, error) {
	q := newQuery()
	setInt(q, "max", p.Max)
	setInt(q, "list_id", p.ListID)

	var out []LeakDetails
	err := c.doJSON(ctx, requestOpts{
		method:      "POST",
		path:        "/search/advanced/unlock",
		query:       q,
		body:        filters,
		useAdvanced: true,
	}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AdvancedUnlockTask calls POST /search/advanced/unlock/task for async
// unlocking; poll the result with GetTaskStatus.
func (c *Client) AdvancedUnlockTask(ctx context.Context, filters LeakSearchFilters, p UnlockParams) (AsyncTaskResult, error) {
	q := newQuery()
	setInt(q, "max", p.Max)
	setInt(q, "list_id", p.ListID)

	var out AsyncTaskResult
	err := c.doJSON(ctx, requestOpts{
		method:      "POST",
		path:        "/search/advanced/unlock/task",
		query:       q,
		body:        filters,
		useAdvanced: true,
	}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AdvancedExport calls POST /search/advanced/export, queuing an async export job.
func (c *Client) AdvancedExport(ctx context.Context, filters LeakSearchFilters, format string) (*ExportResponse, error) {
	q := newQuery()
	setStr(q, "format", format)

	var out ExportResponse
	err := c.doJSON(ctx, requestOpts{
		method:      "POST",
		path:        "/search/advanced/export",
		query:       q,
		body:        filters,
		useAdvanced: true,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

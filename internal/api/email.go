package api

import "context"

// SearchEmail calls POST /search/email.
func (c *Client) SearchEmail(ctx context.Context, req EmailSearchRequest) (*PaginatedLeaksResponse, error) {
	var out PaginatedLeaksResponse
	err := c.doJSON(ctx, requestOpts{
		method: "POST",
		path:   "/search/email",
		body:   req,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// UnlockParams configures the query-string side of an unlock/unlock-task call.
type UnlockParams struct {
	Max    int
	ListID int
}

// EmailUnlock calls POST /search/email/unlock, spending points.
func (c *Client) EmailUnlock(ctx context.Context, req EmailSearchRequest, p UnlockParams) ([]LeakDetails, error) {
	q := newQuery()
	setInt(q, "max", p.Max)
	setInt(q, "list_id", p.ListID)

	var out []LeakDetails
	err := c.doJSON(ctx, requestOpts{
		method: "POST",
		path:   "/search/email/unlock",
		query:  q,
		body:   req,
	}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EmailUnlockTask calls POST /search/email/unlock/task for async unlocking.
// The response has no fixed schema server-side beyond carrying a task_id;
// poll it with GetTaskStatus.
func (c *Client) EmailUnlockTask(ctx context.Context, req EmailSearchRequest, p UnlockParams) (AsyncTaskResult, error) {
	q := newQuery()
	setInt(q, "max", p.Max)
	setInt(q, "list_id", p.ListID)

	var out AsyncTaskResult
	err := c.doJSON(ctx, requestOpts{
		method: "POST",
		path:   "/search/email/unlock/task",
		query:  q,
		body:   req,
	}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EmailExport calls POST /search/email/export, queuing an async export job.
// search/is_email travel in the body (EmailSearchRequest); only format is a
// query param per the API.
func (c *Client) EmailExport(ctx context.Context, req EmailSearchRequest, format string) (*ExportResponse, error) {
	q := newQuery()
	setStr(q, "format", format)

	var out ExportResponse
	err := c.doJSON(ctx, requestOpts{
		method: "POST",
		path:   "/search/email/export",
		query:  q,
		body:   req,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

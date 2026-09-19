package api

import "context"

// ListExports calls GET /exports. There is no per-ID GET endpoint in this
// API — check status by paging through this list, or via the LeakRadar
// member portal's Exports page.
func (c *Client) ListExports(ctx context.Context, page, pageSize int) (*ExportListResponse, error) {
	q := newQuery()
	setInt(q, "page", page)
	setInt(q, "page_size", pageSize)

	var out ExportListResponse
	err := c.doJSON(ctx, requestOpts{
		method: "GET",
		path:   "/exports",
		query:  q,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

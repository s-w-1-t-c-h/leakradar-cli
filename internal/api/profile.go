package api

import "context"

// GetProfile calls GET /profile — also used by `auth status` to validate a key.
func (c *Client) GetProfile(ctx context.Context) (*Profile, error) {
	var out Profile
	err := c.doJSON(ctx, requestOpts{
		method: "GET",
		path:   "/profile",
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUsage calls GET /profile/usage — live plan quota consumption (daily
// requests, concurrency caps, unlocked-lists limit). Points balance for
// unlocks lives on Profile, not here; see GetProfile.
func (c *Client) GetUsage(ctx context.Context) (*PlanUsageResponse, error) {
	var out PlanUsageResponse
	err := c.doJSON(ctx, requestOpts{
		method: "GET",
		path:   "/profile/usage",
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUnlocked calls GET /profile/unlocked.
func (c *Client) GetUnlocked(ctx context.Context, page, pageSize int) (*PaginatedLeaksResponse, error) {
	q := newQuery()
	setInt(q, "page", page)
	setInt(q, "page_size", pageSize)

	var out PaginatedLeaksResponse
	err := c.doJSON(ctx, requestOpts{
		method: "GET",
		path:   "/profile/unlocked",
		query:  q,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

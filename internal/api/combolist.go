package api

import "context"

// SearchComboEmail calls POST /search/combolist/email — the dedicated
// combolist dataset, separate from the main /search/email corpus.
func (c *Client) SearchComboEmail(ctx context.Context, req CombolistEmailSearchRequest, page, pageSize int, autoUnlock bool) (*CombolistSearchResponse, error) {
	q := newQuery()
	setInt(q, "page", page)
	setInt(q, "page_size", pageSize)
	setBool(q, "auto_unlock", autoUnlock)

	var out CombolistSearchResponse
	err := c.doJSON(ctx, requestOpts{method: "POST", path: "/search/combolist/email", query: q, body: req}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchComboUsername calls POST /search/combolist/username.
func (c *Client) SearchComboUsername(ctx context.Context, req CombolistUsernameSearchRequest, page, pageSize int, autoUnlock bool) (*CombolistSearchResponse, error) {
	q := newQuery()
	setInt(q, "page", page)
	setInt(q, "page_size", pageSize)
	setBool(q, "auto_unlock", autoUnlock)

	var out CombolistSearchResponse
	err := c.doJSON(ctx, requestOpts{method: "POST", path: "/search/combolist/username", query: q, body: req}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ComboDomainList calls GET /search/combolist/domain/{domain} — the
// paginated list of individual records (as opposed to ComboDomainReport's
// summary counts).
func (c *Client) ComboDomainList(ctx context.Context, domain, search string, page, pageSize int, autoUnlock bool) (*CombolistSearchResponse, error) {
	q := newQuery()
	setStr(q, "search", search)
	setInt(q, "page", page)
	setInt(q, "page_size", pageSize)
	setBool(q, "auto_unlock", autoUnlock)

	var out CombolistSearchResponse
	err := c.doJSON(ctx, requestOpts{method: "GET", path: "/search/combolist/domain/" + pathEscape(domain), query: q}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ComboDomainReport calls GET /search/combolist/domain/{domain}/report.
func (c *Client) ComboDomainReport(ctx context.Context, domain string) (*CombolistDomainReportResponse, error) {
	var out CombolistDomainReportResponse
	err := c.doJSON(ctx, requestOpts{method: "GET", path: "/search/combolist/domain/" + pathEscape(domain) + "/report"}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SearchComboAdvanced calls POST /search/combolist/advanced.
func (c *Client) SearchComboAdvanced(ctx context.Context, filters CombolistAdvancedSearchRequest, page, pageSize int, autoUnlock bool) (*CombolistSearchResponse, error) {
	q := newQuery()
	setInt(q, "page", page)
	setInt(q, "page_size", pageSize)
	setBool(q, "auto_unlock", autoUnlock)

	var out CombolistSearchResponse
	err := c.doJSON(ctx, requestOpts{method: "POST", path: "/search/combolist/advanced", query: q, body: filters, useAdvanced: true}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ComboUnlockTask calls POST /search/combolist/unlock/task. Combolist
// unlocking by search criteria only exists as an async task: the
// synchronous /search/combolist/unlock endpoint takes explicit record IDs,
// and locked rows never expose one (id is null until already unlocked), so
// there is no synchronous "unlock everything matching this search" call —
// this task endpoint, which replays the search server-side, is the only path.
func (c *Client) ComboUnlockTask(ctx context.Context, scoped CombolistScopedRequest, max, listID int) (*TaskStatus, error) {
	q := newQuery()
	setInt(q, "max", max)
	setInt(q, "list_id", listID)

	var out TaskStatus
	err := c.doJSON(ctx, requestOpts{method: "POST", path: "/search/combolist/unlock/task", query: q, body: scoped}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ComboExport calls POST /search/combolist/export, queuing an async export
// job (same no-download-by-ID caveat as the main dataset's export jobs).
func (c *Client) ComboExport(ctx context.Context, scoped CombolistScopedRequest, format string) (*ExportResponse, error) {
	q := newQuery()
	setStr(q, "format", format)

	var out ExportResponse
	err := c.doJSON(ctx, requestOpts{method: "POST", path: "/search/combolist/export", query: q, body: scoped}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

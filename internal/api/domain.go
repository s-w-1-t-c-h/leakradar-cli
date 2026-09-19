package api

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

// LeakType is one of the domain-scoped leak categories.
type LeakType string

const (
	LeakTypeAll          LeakType = "all"
	LeakTypeEmployees    LeakType = "employees"
	LeakTypeCustomers    LeakType = "customers"
	LeakTypeThirdParties LeakType = "third_parties"
)

// DomainListParams are the shared pagination/filter params for the
// employees/customers/third_parties/all list endpoints.
type DomainListParams struct {
	Page       int
	PageSize   int
	Search     string
	IsEmail    *bool
	AutoUnlock bool
}

// DomainReport calls GET /search/domain/{domain}.
func (c *Client) DomainReport(ctx context.Context, domain string, light, includeSearchCount bool) (*DomainSearchResponse, error) {
	q := newQuery()
	setBool(q, "light", light)
	setBool(q, "include_search_count", includeSearchCount)

	var out DomainSearchResponse
	err := c.doJSON(ctx, requestOpts{
		method: "GET",
		path:   "/search/domain/" + pathEscape(domain),
		query:  q,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DomainList calls GET /search/domain/{domain}/{leakType} for
// employees|customers|third_parties. Use DomainListAll for the "all" category,
// which has a different response shape (items carry a Category field).
func (c *Client) DomainList(ctx context.Context, domain string, leakType LeakType, p DomainListParams) (*PaginatedLeaksResponse, error) {
	q := newQuery()
	setInt(q, "page", p.Page)
	setInt(q, "page_size", p.PageSize)
	setStr(q, "search", p.Search)
	setBoolPtr(q, "is_email", p.IsEmail)
	setBool(q, "auto_unlock", p.AutoUnlock)

	var out PaginatedLeaksResponse
	err := c.doJSON(ctx, requestOpts{
		method: "GET",
		path:   fmt.Sprintf("/search/domain/%s/%s", pathEscape(domain), leakType),
		query:  q,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DomainListAll calls GET /search/domain/{domain}/all.
func (c *Client) DomainListAll(ctx context.Context, domain string, p DomainListParams) (*PaginatedAllLeaksResponse, error) {
	q := newQuery()
	setInt(q, "page", p.Page)
	setInt(q, "page_size", p.PageSize)
	setStr(q, "search", p.Search)
	setBoolPtr(q, "is_email", p.IsEmail)
	setBool(q, "auto_unlock", p.AutoUnlock)

	var out PaginatedAllLeaksResponse
	err := c.doJSON(ctx, requestOpts{
		method: "GET",
		path:   "/search/domain/" + pathEscape(domain) + "/all",
		query:  q,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DomainSubdomains calls GET /search/domain/{domain}/subdomains.
func (c *Client) DomainSubdomains(ctx context.Context, domain string, page, pageSize int, search string) (*DomainSubdomainsResponse, error) {
	q := newQuery()
	setInt(q, "page", page)
	setInt(q, "page_size", pageSize)
	setStr(q, "search", search)

	var out DomainSubdomainsResponse
	err := c.doJSON(ctx, requestOpts{
		method: "GET",
		path:   "/search/domain/" + pathEscape(domain) + "/subdomains",
		query:  q,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DomainURLs calls GET /search/domain/{domain}/urls.
func (c *Client) DomainURLs(ctx context.Context, domain string, page, pageSize int, search string) (*DomainURLsResponse, error) {
	q := newQuery()
	setInt(q, "page", page)
	setInt(q, "page_size", pageSize)
	setStr(q, "search", search)

	var out DomainURLsResponse
	err := c.doJSON(ctx, requestOpts{
		method: "GET",
		path:   "/search/domain/" + pathEscape(domain) + "/urls",
		query:  q,
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DomainReportPDF calls GET /search/domain/{domain}/report/pdf and returns
// the raw PDF bytes as a stream for the caller to write to disk.
func (c *Client) DomainReportPDF(ctx context.Context, domain string) (io.ReadCloser, error) {
	return c.doRaw(ctx, requestOpts{
		method:      "GET",
		path:        "/search/domain/" + pathEscape(domain) + "/report/pdf",
		rawResponse: true,
	})
}

// DomainUnlockParams configures a domain unlock call.
type DomainUnlockParams struct {
	Search  string
	IsEmail *bool
	Max     int
	ListID  int
}

// DomainUnlock calls POST /search/domain/{domain}/{leakType}/unlock, spending points.
func (c *Client) DomainUnlock(ctx context.Context, domain string, leakType LeakType, p DomainUnlockParams) ([]LeakDetails, error) {
	q := newQuery()
	setStr(q, "search", p.Search)
	setBoolPtr(q, "is_email", p.IsEmail)
	setInt(q, "max", p.Max)
	setInt(q, "list_id", p.ListID)

	var out []LeakDetails
	err := c.doJSON(ctx, requestOpts{
		method: "POST",
		path:   fmt.Sprintf("/search/domain/%s/%s/unlock", pathEscape(domain), leakType),
		query:  q,
	}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DomainUnlockTask calls POST /search/domain/{domain}/{leakType}/unlock/task
// for async unlocking; poll the result with GetTaskStatus.
func (c *Client) DomainUnlockTask(ctx context.Context, domain string, leakType LeakType, p DomainUnlockParams) (AsyncTaskResult, error) {
	q := newQuery()
	setStr(q, "search", p.Search)
	setBoolPtr(q, "is_email", p.IsEmail)
	setInt(q, "max", p.Max)
	setInt(q, "list_id", p.ListID)

	var out AsyncTaskResult
	err := c.doJSON(ctx, requestOpts{
		method: "POST",
		path:   fmt.Sprintf("/search/domain/%s/%s/unlock/task", pathEscape(domain), leakType),
		query:  q,
	}, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type domainExportBody struct {
	Search  string `json:"search,omitempty"`
	IsEmail *bool  `json:"is_email,omitempty"`
	Format  string `json:"format,omitempty"`
}

// DomainExport calls POST /search/domain/{domain}/{leakType}/export, queuing an async export job.
func (c *Client) DomainExport(ctx context.Context, domain string, leakType LeakType, search string, isEmail *bool, format string) (*ExportResponse, error) {
	q := newQuery()
	setStr(q, "search", search)
	setBoolPtr(q, "is_email", isEmail)
	setStr(q, "format", format)

	var out ExportResponse
	err := c.doJSON(ctx, requestOpts{
		method: "POST",
		path:   fmt.Sprintf("/search/domain/%s/%s/export", pathEscape(domain), leakType),
		query:  q,
		body:   domainExportBody{Search: search, IsEmail: isEmail, Format: format},
	}, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// DomainSubdomainsExport / DomainURLsExport call the streamed CSV/TXT/JSON
// export endpoints for subdomains and URLs, returning the raw file body.
func (c *Client) DomainSubdomainsExport(ctx context.Context, domain, search, format string) (io.ReadCloser, error) {
	return c.domainSubExport(ctx, domain, "subdomains", search, format)
}

func (c *Client) DomainURLsExport(ctx context.Context, domain, search, format string) (io.ReadCloser, error) {
	return c.domainSubExport(ctx, domain, "urls", search, format)
}

func (c *Client) domainSubExport(ctx context.Context, domain, sub, search, format string) (io.ReadCloser, error) {
	q := newQuery()
	setStr(q, "search", search)
	setStr(q, "format", format)

	return c.doRaw(ctx, requestOpts{
		method: "POST",
		path:   fmt.Sprintf("/search/domain/%s/%s/export", pathEscape(domain), sub),
		query:  q,
		body: struct {
			Search string `json:"search,omitempty"`
		}{Search: search},
		rawResponse: true,
	})
}

// pathEscape escapes a user-supplied domain for safe inclusion as a single
// URL path segment. url.PathEscape leaves '.', '-', '_' and '~' untouched
// (they are unreserved per RFC 3986), so ordinary domains stay readable
// while anything unexpected (spaces, slashes, control characters) is
// percent-encoded instead of being able to alter the request path.
func pathEscape(s string) string {
	return url.PathEscape(s)
}

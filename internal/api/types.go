package api

// Field names below were verified against the live OpenAPI spec fetched
// directly from https://api.leakradar.io/openapi.json (curl, not a
// summarising fetch) — see internal/api doc comment in client.go for why
// that distinction matters.

// LeakDetails mirrors the LeakDetails schema returned by /search/email,
// /search/advanced and the domain employees/customers/third_parties
// endpoints. There is no separate "email" field: the leaked identifier is
// always in Username, disambiguated by IsEmail.
type LeakDetails struct {
	ID               string `json:"id,omitempty"`
	URL              string `json:"url,omitempty"`
	Username         string `json:"username,omitempty"`
	UsernameMasked   string `json:"username_masked,omitempty"`
	Password         string `json:"password,omitempty"`
	PasswordStrength int    `json:"password_strength,omitempty"`
	Unlocked         bool   `json:"unlocked"`
	IsEmail          *bool  `json:"is_email,omitempty"`
	AddedAt          string `json:"added_at,omitempty"`
	Status           string `json:"status,omitempty"`
}

// AllLeakDetails is LeakDetails plus Category, returned only by the
// /search/domain/{domain}/all endpoint (which merges all three categories).
type AllLeakDetails struct {
	LeakDetails
	Category string `json:"category"`
}

// EmailSearchRequest is the body for POST /search/email (and the email
// unlock/export endpoints, which reuse the same shape).
type EmailSearchRequest struct {
	Email   string `json:"email"`
	Search  string `json:"search,omitempty"`
	IsEmail *bool  `json:"is_email,omitempty"`
}

// PaginatedLeaksResponse is returned by /search/email, /search/advanced,
// and the domain employees/customers/third_parties list endpoints.
type PaginatedLeaksResponse struct {
	Items                 []LeakDetails `json:"items"`
	Total                 int           `json:"total"`
	TotalUnlocked         int           `json:"total_unlocked"`
	Page                  int           `json:"page"`
	PageSize              int           `json:"page_size"`
	BlacklistedValue      string        `json:"blacklisted_value,omitempty"`
	AutoUnlockPointsSpent int           `json:"auto_unlock_points_consumed,omitempty"`
}

// PaginatedAllLeaksResponse is the /search/domain/{domain}/all variant,
// whose items carry a Category field.
type PaginatedAllLeaksResponse struct {
	Items                 []AllLeakDetails `json:"items"`
	Total                 int              `json:"total"`
	TotalUnlocked         int              `json:"total_unlocked"`
	Page                  int              `json:"page"`
	PageSize              int              `json:"page_size"`
	BlacklistedValue      string           `json:"blacklisted_value,omitempty"`
	AutoUnlockPointsSpent int              `json:"auto_unlock_points_consumed,omitempty"`
}

// TextMatchType is the matching mode for the *_match_type filter fields:
// contains | starts_with | ends_with.
type TextMatchType string

const (
	MatchContains   TextMatchType = "contains"
	MatchStartsWith TextMatchType = "starts_with"
	MatchEndsWith   TextMatchType = "ends_with"
)

// PasswordStrengthCategory is one of too_weak | weak | medium | strong.
type PasswordStrengthCategory string

const (
	StrengthTooWeak PasswordStrengthCategory = "too_weak"
	StrengthWeak    PasswordStrengthCategory = "weak"
	StrengthMedium  PasswordStrengthCategory = "medium"
	StrengthStrong  PasswordStrengthCategory = "strong"
)

// LeakSearchFilters is the body for POST /search/advanced (and the
// advanced unlock/export endpoints). Every *_not field excludes matches;
// every *_match_type field controls how the corresponding values match
// (default "contains" server-side when omitted).
type LeakSearchFilters struct {
	Username             []string       `json:"username,omitempty"`
	UsernameNot          []string       `json:"username_not,omitempty"`
	UsernameMatchType    *TextMatchType `json:"username_match_type,omitempty"`
	UsernameNotMatchType *TextMatchType `json:"username_not_match_type,omitempty"`

	Password             []string       `json:"password,omitempty"`
	PasswordNot          []string       `json:"password_not,omitempty"`
	PasswordMatchType    *TextMatchType `json:"password_match_type,omitempty"`
	PasswordNotMatchType *TextMatchType `json:"password_not_match_type,omitempty"`

	URL             []string       `json:"url,omitempty"`
	URLNot          []string       `json:"url_not,omitempty"`
	URLMatchType    *TextMatchType `json:"url_match_type,omitempty"`
	URLNotMatchType *TextMatchType `json:"url_not_match_type,omitempty"`

	URLDomain             []string       `json:"url_domain,omitempty"`
	URLDomainNot          []string       `json:"url_domain_not,omitempty"`
	URLDomainMatchType    *TextMatchType `json:"url_domain_match_type,omitempty"`
	URLDomainNotMatchType *TextMatchType `json:"url_domain_not_match_type,omitempty"`

	URLHost             []string       `json:"url_host,omitempty"`
	URLHostNot          []string       `json:"url_host_not,omitempty"`
	URLHostMatchType    *TextMatchType `json:"url_host_match_type,omitempty"`
	URLHostNotMatchType *TextMatchType `json:"url_host_not_match_type,omitempty"`

	UsernameHash []string `json:"username_hash,omitempty"`
	PasswordHash []string `json:"password_hash,omitempty"`

	URLScheme    []string `json:"url_scheme,omitempty"`
	URLSchemeNot []string `json:"url_scheme_not,omitempty"`
	URLPort      []int    `json:"url_port,omitempty"`
	URLPortNot   []int    `json:"url_port_not,omitempty"`
	URLTLD       []string `json:"url_tld,omitempty"`
	URLTLDNot    []string `json:"url_tld_not,omitempty"`

	IsEmail *bool `json:"is_email,omitempty"`

	EmailDomain             []string       `json:"email_domain,omitempty"`
	EmailDomainNot          []string       `json:"email_domain_not,omitempty"`
	EmailDomainMatchType    *TextMatchType `json:"email_domain_match_type,omitempty"`
	EmailDomainNotMatchType *TextMatchType `json:"email_domain_not_match_type,omitempty"`

	EmailHost             []string       `json:"email_host,omitempty"`
	EmailHostNot          []string       `json:"email_host_not,omitempty"`
	EmailHostMatchType    *TextMatchType `json:"email_host_match_type,omitempty"`
	EmailHostNotMatchType *TextMatchType `json:"email_host_not_match_type,omitempty"`

	EmailTLD    []string `json:"email_tld,omitempty"`
	EmailTLDNot []string `json:"email_tld_not,omitempty"`

	PasswordStrength *PasswordStrengthCategory `json:"password_strength,omitempty"`

	AddedFrom string `json:"added_from,omitempty"` // RFC3339 datetime
	AddedTo   string `json:"added_to,omitempty"`   // RFC3339 datetime

	ForceAnd *bool `json:"force_and,omitempty"`
}

// PasswordStrengthBucket is one strength bucket's count/percentage.
type PasswordStrengthBucket struct {
	Qty  int     `json:"qty"`
	Perc float64 `json:"perc"`
}

// PasswordStrengthSummary is the weak/medium/strong breakdown used in domain reports.
type PasswordStrengthSummary struct {
	TotalPass int                    `json:"total_pass"`
	TooWeak   PasswordStrengthBucket `json:"too_weak"`
	Weak      PasswordStrengthBucket `json:"weak"`
	Medium    PasswordStrengthBucket `json:"medium"`
	Strong    PasswordStrengthBucket `json:"strong"`
}

// DomainSearchResponse is the report from GET /search/domain/{domain}. The
// API returns either a "light" (sampled/unauthenticated) or "full" shape
// depending on the light query param; this struct is a superset of both,
// so whichever fields the response doesn't include simply stay zero-valued.
type DomainSearchResponse struct {
	EmployeesCompromised    int                      `json:"employees_compromised"`
	ThirdPartiesCompromised int                      `json:"third_parties_compromised"`
	CustomersCompromised    int                      `json:"customers_compromised"`
	EmployeePasswords       *PasswordStrengthSummary `json:"employee_passwords,omitempty"`
	ThirdPartiesPasswords   *PasswordStrengthSummary `json:"third_parties_passwords,omitempty"`
	CustomerPasswords       *PasswordStrengthSummary `json:"customer_passwords,omitempty"`
	BlacklistedValue        string                   `json:"blacklisted_value,omitempty"`
	SearchedByCount         int                      `json:"searched_by_count,omitempty"`
	// Light-mode-only fields:
	HasMatches          *bool    `json:"has_matches,omitempty"`
	CountsApproximate   bool     `json:"counts_approximate,omitempty"`
	SamplingProbability *float64 `json:"sampling_probability,omitempty"`
}

// SubdomainOccurrences is one row from GET /search/domain/{domain}/subdomains.
type SubdomainOccurrences struct {
	Subdomain   string `json:"subdomain"`
	Occurrences int    `json:"occurrences"`
}

// DomainSubdomainsResponse is the paginated result of the subdomains endpoint.
type DomainSubdomainsResponse struct {
	Items            []SubdomainOccurrences `json:"items"`
	Total            int                    `json:"total"`
	Page             int                    `json:"page"`
	PageSize         int                    `json:"page_size"`
	BlacklistedValue string                 `json:"blacklisted_value,omitempty"`
}

// URLOccurrences is one row from GET /search/domain/{domain}/urls.
type URLOccurrences struct {
	URL         string `json:"url"`
	Occurrences int    `json:"occurrences"`
}

// DomainURLsResponse is the paginated result of the urls endpoint.
type DomainURLsResponse struct {
	Items            []URLOccurrences `json:"items"`
	Total            int              `json:"total"`
	Page             int              `json:"page"`
	PageSize         int              `json:"page_size"`
	BlacklistedValue string           `json:"blacklisted_value,omitempty"`
}

// DarkWebSearchRequest is the body for POST /search/dark-web. Either set
// Query alone (simple mode) or one/more of the field-specific filters
// (advanced mode), combined per Logic ("AND"/"OR", default "AND").
type DarkWebSearchRequest struct {
	Query     string `json:"query,omitempty"`
	Title     string `json:"title,omitempty"`
	Content   string `json:"content,omitempty"`
	Author    string `json:"author,omitempty"`
	SourceURL string `json:"source_url,omitempty"`
	Logic     string `json:"logic,omitempty"`
}

// DarkWebItem is one item in a dark-web search result.
type DarkWebItem struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Content     string `json:"content,omitempty"`
	Source      string `json:"source,omitempty"`
	SourceName  string `json:"source_name,omitempty"`
	Target      string `json:"target,omitempty"`
	SourceRef   string `json:"source_ref,omitempty"`
	Author      string `json:"author,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`
	IngestedAt  string `json:"ingested_at,omitempty"`
	Truncated   bool   `json:"truncated"`
	Censored    bool   `json:"censored"`
}

// DarkWebSearchResponse is the paginated result of a dark-web search.
type DarkWebSearchResponse struct {
	Items    []DarkWebItem `json:"items"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
}

// PasswordRangeItem is one hash suffix + occurrence count.
type PasswordRangeItem struct {
	Hash  string `json:"hash"`
	Count int    `json:"count"`
}

// PasswordRangeResponse is the result of GET /password-range.
type PasswordRangeResponse struct {
	Prefix string              `json:"prefix"`
	Total  int                 `json:"total"`
	Hashes []PasswordRangeItem `json:"hashes"`
}

// EmailMassRequest is the body for POST /search/emails/locked-exists (max 100 entries).
type EmailMassRequest struct {
	Emails        []string `json:"emails"`
	IncludeCounts bool     `json:"include_counts,omitempty"`
}

// EmailMassResult is one row of an emails locked-exists check.
type EmailMassResult struct {
	Email    string `json:"email"`
	Any      bool   `json:"any"`
	Locked   *int   `json:"locked,omitempty"`
	Total    *int   `json:"total,omitempty"`
	Unlocked *int   `json:"unlocked,omitempty"`
}

// EmailMassResponse wraps the results of an emails locked-exists check.
type EmailMassResponse struct {
	Results []EmailMassResult `json:"results"`
}

// DomainMassRequest is the body for POST /search/domains/locked-exists (max 100 entries).
type DomainMassRequest struct {
	Domains       []string `json:"domains"`
	Categories    []string `json:"categories,omitempty"` // subset of employees/customers/third_parties
	IncludeCounts bool     `json:"include_counts,omitempty"`
}

// CategoryInfo is the per-category breakdown in a domain locked-exists result.
type CategoryInfo struct {
	HasLocked bool `json:"has_locked"`
	Count     *int `json:"count,omitempty"`
	Total     *int `json:"total,omitempty"`
	Unlocked  *int `json:"unlocked,omitempty"`
}

// DomainMassResult is one row of a domains locked-exists check.
type DomainMassResult struct {
	Domain     string                  `json:"domain"`
	Any        bool                    `json:"any"`
	Categories map[string]CategoryInfo `json:"categories"`
}

// DomainMassResponse wraps the results of a domains locked-exists check.
type DomainMassResponse struct {
	Results []DomainMassResult `json:"results"`
}

// ExportResponse is returned by POST .../export (a queued job).
type ExportResponse struct {
	Status   string `json:"status"` // queued | processing | completed | failed
	Message  string `json:"message"`
	ExportID int    `json:"export_id"`
}

// ExportJob is one row from GET /exports. The API has no per-ID GET or
// download-by-URL endpoint for these jobs today — completed exports are
// retrieved from the LeakRadar member portal's Exports page.
type ExportJob struct {
	ID         int                    `json:"id"`
	UserID     int                    `json:"user_id"`
	Filename   string                 `json:"filename"`
	Type       string                 `json:"type"`
	Params     map[string]interface{} `json:"params,omitempty"`
	Status     string                 `json:"status"`
	Timestamp  string                 `json:"timestamp"`
	StartedAt  string                 `json:"started_at,omitempty"`
	FinishedAt string                 `json:"finished_at,omitempty"`
}

// ExportListResponse is the paginated result of GET /exports.
type ExportListResponse struct {
	Items            []ExportJob `json:"items"`
	Total            int         `json:"total"`
	Page             int         `json:"page"`
	PageSize         int         `json:"page_size"`
	BlacklistedValue string      `json:"blacklisted_value,omitempty"`
}

// TaskStatus is the result of GET /tasks/{task_id}, used to poll an async
// unlock task queued via one of the .../unlock/task endpoints.
type TaskStatus struct {
	TaskID           string `json:"task_id"`
	Running          bool   `json:"running"`
	Completed        bool   `json:"completed"`
	Total            *int   `json:"total,omitempty"`
	Updated          *int   `json:"updated,omitempty"`
	VersionConflicts *int   `json:"version_conflicts,omitempty"`
}

// Invoice mirrors one entry in Profile.Invoices.
type Invoice struct {
	ID        int     `json:"id"`
	Amount    float64 `json:"amount"`
	Period    string  `json:"period"`
	Paid      bool    `json:"paid"`
	IsRenewal bool    `json:"is_renewal"`
}

// Plan describes the account's subscription plan.
type Plan struct {
	ID              int     `json:"id"`
	PlanName        string  `json:"plan_name"`
	Name            string  `json:"name"`
	Price           float64 `json:"price"`
	Period          string  `json:"period"`
	Points          int     `json:"points"`
	GB              int     `json:"gb"`
	EmailSearch     bool    `json:"email_search"`
	DomainSearch    bool    `json:"domain_search"`
	AdvancedSearch  bool    `json:"advanced_search"`
	RawSearch       bool    `json:"raw_search"`
	ComboListSearch bool    `json:"combolist_search"`
	DarkWebSearch   bool    `json:"dark_web_search"`
	DailyRequestCap int     `json:"daily_request_cap"`
}

// Profile mirrors GET /profile — verified directly against a real response
// (see example_api_call.txt), not the summarised OpenAPI spec.
type Profile struct {
	ID                  int       `json:"id"`
	FirstName           string    `json:"first_name"`
	LastName            string    `json:"last_name"`
	Organization        string    `json:"organization"`
	Email               string    `json:"email"`
	SubscriptionActive  bool      `json:"subscription_active"`
	SubscriptionPoints  int       `json:"subscription_points"`
	SubscriptionGB      int64     `json:"subscription_gb"`
	ExtraPoints         int       `json:"extra_points"`
	ExtraGB             int64     `json:"extra_gb"`
	SubscriptionEndDate string    `json:"subscription_end_date"`
	Plan                Plan      `json:"plan"`
	Invoices            []Invoice `json:"invoices,omitempty"`
	EmailConfirmed      bool      `json:"email_confirmed"`
	Banned              bool      `json:"banned"`
}

// UsageMetric is one plan dimension's current consumption vs its limit.
// Limit == nil means unlimited; Used == nil means the counter couldn't be
// read right now (show "unavailable", not 0).
type UsageMetric struct {
	Used  *int `json:"used"`
	Limit *int `json:"limit"`
}

// PlanUsageResponse is GET /profile/usage — live plan quota consumption,
// distinct from the points balance in Profile (points fund unlocks; these
// are separate rate/concurrency caps).
type PlanUsageResponse struct {
	PlanName                string      `json:"plan_name,omitempty"`
	IsPaid                  bool        `json:"is_paid"`
	CapsEnforced            bool        `json:"caps_enforced"`
	RequestsDaily           UsageMetric `json:"requests_daily"`
	RequestsDailyResetEpoch *int64      `json:"requests_daily_reset_epoch,omitempty"`
	ConcurrentRaw           UsageMetric `json:"concurrent_raw"`
	ConcurrentAdvanced      UsageMetric `json:"concurrent_advanced"`
	UnlockedLists           UsageMetric `json:"unlocked_lists"`
}

// CombolistCredentialDetails is one credential from the dedicated combolist
// search surface — a separate, larger dataset from the main leak corpus
// (parsed combolists rather than stealer-log/database-breach records). It
// has no URL/leak-name fields at all, only an identifier + password pair.
type CombolistCredentialDetails struct {
	ID               string `json:"id,omitempty"`
	Username         string `json:"username,omitempty"`
	UsernameMasked   string `json:"username_masked,omitempty"`
	Password         string `json:"password,omitempty"`
	PasswordStrength int    `json:"password_strength,omitempty"`
	IsEmail          bool   `json:"is_email"`
	EmailDomain      string `json:"email_domain,omitempty"`
	AddedAt          string `json:"added_at"`
	Unlocked         bool   `json:"unlocked"`
	Status           string `json:"status,omitempty"`
}

// CombolistSearchResponse is returned by every combolist search endpoint
// (email, username, domain list, advanced).
type CombolistSearchResponse struct {
	Items                 []CombolistCredentialDetails `json:"items"`
	Total                 int                          `json:"total"`
	TotalUnlocked         int                          `json:"total_unlocked"`
	Page                  int                          `json:"page"`
	PageSize              int                          `json:"page_size"`
	BlacklistedValue      string                       `json:"blacklisted_value,omitempty"`
	AutoUnlockPointsSpent int                          `json:"auto_unlock_points_consumed,omitempty"`
}

// CombolistEmailSearchRequest is the body for POST /search/combolist/email.
type CombolistEmailSearchRequest struct {
	Email  string `json:"email"`
	Search string `json:"search,omitempty"`
}

// CombolistUsernameSearchRequest is the body for POST /search/combolist/username.
type CombolistUsernameSearchRequest struct {
	Username string `json:"username"`
	Search   string `json:"search,omitempty"`
}

// CombolistAdvancedSearchRequest is a strict subset of LeakSearchFilters —
// the combolist dataset has no URL data at all, so none of the
// url/url_domain/url_host/url_scheme/url_port/url_tld/email_host/email_tld
// fields exist here. The server rejects unknown fields on this request.
type CombolistAdvancedSearchRequest struct {
	Username             []string       `json:"username,omitempty"`
	UsernameNot          []string       `json:"username_not,omitempty"`
	UsernameMatchType    *TextMatchType `json:"username_match_type,omitempty"`
	UsernameNotMatchType *TextMatchType `json:"username_not_match_type,omitempty"`

	Password             []string       `json:"password,omitempty"`
	PasswordNot          []string       `json:"password_not,omitempty"`
	PasswordMatchType    *TextMatchType `json:"password_match_type,omitempty"`
	PasswordNotMatchType *TextMatchType `json:"password_not_match_type,omitempty"`

	EmailDomain             []string       `json:"email_domain,omitempty"`
	EmailDomainNot          []string       `json:"email_domain_not,omitempty"`
	EmailDomainMatchType    *TextMatchType `json:"email_domain_match_type,omitempty"`
	EmailDomainNotMatchType *TextMatchType `json:"email_domain_not_match_type,omitempty"`

	UsernameHash []string `json:"username_hash,omitempty"`
	PasswordHash []string `json:"password_hash,omitempty"`

	IsEmail          *bool                     `json:"is_email,omitempty"`
	PasswordStrength *PasswordStrengthCategory `json:"password_strength,omitempty"`

	AddedFrom string `json:"added_from,omitempty"` // RFC3339 datetime
	AddedTo   string `json:"added_to,omitempty"`   // RFC3339 datetime
	ForceAnd  bool   `json:"force_and,omitempty"`
}

// CombolistPasswordStrengthReport is the weak/medium/strong breakdown used
// in combolist domain reports (plain counts, unlike the main dataset's
// PasswordStrengthBucket which also carries a percentage).
type CombolistPasswordStrengthReport struct {
	TooWeak int `json:"too_weak"`
	Weak    int `json:"weak"`
	Medium  int `json:"medium"`
	Strong  int `json:"strong"`
}

// CombolistDomainReportResponse is GET /search/combolist/domain/{domain}/report.
type CombolistDomainReportResponse struct {
	Domain           string                          `json:"domain"`
	TotalCredentials int                             `json:"total_credentials"`
	UniqueEmails     int                             `json:"unique_emails"`
	PasswordStrength CombolistPasswordStrengthReport `json:"password_strength"`
	FirstSeen        string                          `json:"first_seen,omitempty"`
	LastSeen         string                          `json:"last_seen,omitempty"`
}

// CombolistScopedRequest is the stable search contract used by
// /search/combolist/unlock/task and /search/combolist/export — it replays
// an email/username/domain/advanced search server-side rather than the
// client supplying explicit record IDs (locked combolist rows never expose
// an ID, so that's the only way to bulk-unlock/export by search criteria).
// Scope is one of "email"/"username"/"domain"/"advanced": the first three
// use Value (+ optional Search); "advanced" uses Filters instead.
type CombolistScopedRequest struct {
	Scope   string                          `json:"scope"`
	Value   string                          `json:"value,omitempty"`
	Search  string                          `json:"search,omitempty"`
	Filters *CombolistAdvancedSearchRequest `json:"filters,omitempty"`
}

// RawMatchType is how a raw file-name/folder-name value is matched. Kept
// separate from TextMatchType on purpose — raw has an extra "exact" mode
// the other datasets don't.
type RawMatchType string

const (
	RawMatchContains   RawMatchType = "contains"
	RawMatchStartsWith RawMatchType = "starts_with"
	RawMatchEndsWith   RawMatchType = "ends_with"
	RawMatchExact      RawMatchType = "exact"
)

// RawSearchRequest is the body for /search/raw, /search/raw/count,
// /search/raw/unlock/task and /search/raw/export — the same filter set
// drives search, cost estimation, unlock and export for the raw dataset
// (full-text search across unparsed breach file content, not the
// structured email/domain/advanced or combolist datasets).
type RawSearchRequest struct {
	Q           string `json:"q,omitempty"`
	ContainerID *int   `json:"container_id,omitempty"`

	Exts          []string `json:"exts,omitempty"`
	ExtsNot       []string `json:"exts_not,omitempty"`
	Categories    []string `json:"categories,omitempty"`
	CategoriesNot []string `json:"categories_not,omitempty"`

	FileName             []string      `json:"file_name,omitempty"`
	FileNameNot          []string      `json:"file_name_not,omitempty"`
	FileNameMatchType    *RawMatchType `json:"file_name_match_type,omitempty"`
	FileNameNotMatchType *RawMatchType `json:"file_name_not_match_type,omitempty"`

	FolderName             []string      `json:"folder_name,omitempty"`
	FolderNameNot          []string      `json:"folder_name_not,omitempty"`
	FolderNameMatchType    *RawMatchType `json:"folder_name_match_type,omitempty"`
	FolderNameNotMatchType *RawMatchType `json:"folder_name_not_match_type,omitempty"`

	ForceAnd *bool `json:"force_and,omitempty"`

	QExact       *bool    `json:"q_exact,omitempty"`
	IncludeTerms []string `json:"include_terms,omitempty"`
	ExcludeTerms []string `json:"exclude_terms,omitempty"`

	IngestedAtMin string `json:"ingested_at_min,omitempty"` // RFC3339 datetime
	IngestedAtMax string `json:"ingested_at_max,omitempty"` // RFC3339 datetime
}

// RawSearchItem is one matching block from the raw (unparsed) dataset —
// a location inside a source file, not a structured credential.
type RawSearchItem struct {
	ContainerID      string   `json:"container_id"`
	EntryPath        string   `json:"entry_path"`
	EntryName        string   `json:"entry_name,omitempty"`
	Ext              string   `json:"ext,omitempty"`
	Seq              *int     `json:"seq,omitempty"`
	Offset           *int64   `json:"offset,omitempty"`
	IngestedAt       string   `json:"ingested_at,omitempty"`
	SHA256Original   string   `json:"sha256_original,omitempty"`
	OriginalFileName string   `json:"original_file_name,omitempty"`
	Category         string   `json:"category,omitempty"` // stealer-logs | database | combolist
	DisplayName      string   `json:"display_name,omitempty"`
	Snippet          string   `json:"snippet,omitempty"` // censored while locked
	AlreadyUnlocked  bool     `json:"already_unlocked"`
	MatchingLines    []string `json:"matching_lines,omitempty"` // export only
}

// RawSearchResponse is returned by POST /search/raw. Supports both
// page-based (Page/PageSize) and cursor-based (HasMore/NextCursor)
// pagination; Total is -1 when cursor mode is in use.
type RawSearchResponse struct {
	Items            []RawSearchItem `json:"items"`
	Total            int             `json:"total"`
	Page             int             `json:"page"`
	PageSize         int             `json:"page_size"`
	HasMore          *bool           `json:"has_more,omitempty"`
	NextCursor       string          `json:"next_cursor,omitempty"`
	BlacklistedValue string          `json:"blacklisted_value,omitempty"`
}

// RawCountResponse is returned by POST /search/raw/count — a cheap upper
// bound on unlock cost (1 point per part) for the same filters, before
// committing to an actual unlock.
type RawCountResponse struct {
	Total            int    `json:"total"`
	Exact            bool   `json:"exact"`
	Capped           bool   `json:"capped"`
	BlacklistedValue string `json:"blacklisted_value,omitempty"`
}

// APIError is the uniform {"detail": ...} error body. Detail can be either
// a plain string or an object with code/message, so it is decoded loosely.
type APIError struct {
	Detail interface{} `json:"detail"`
}

// Message returns a human-readable string regardless of which shape Detail took.
func (e *APIError) Message() string {
	switch v := e.Detail.(type) {
	case string:
		return v
	case map[string]interface{}:
		if msg, ok := v["message"].(string); ok {
			return msg
		}
	}
	return "unknown API error"
}

// Code returns the machine-readable error code when the API supplied one.
func (e *APIError) Code() string {
	if v, ok := e.Detail.(map[string]interface{}); ok {
		if code, ok := v["code"].(string); ok {
			return code
		}
	}
	return ""
}

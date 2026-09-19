package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c := New("test-key")
	c.BaseURL = srv.URL
	c.HTTPClient.Timeout = 5 * time.Second
	c.maxRetries = 2
	return c
}

func TestSearchEmail_Success(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization header = %q, want Bearer test-key", got)
		}
		if r.Method != http.MethodPost || r.URL.Path != "/search/email" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body EmailSearchRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding body: %v", err)
		}
		if body.Email != "test@example.com" {
			t.Fatalf("body.Email = %q", body.Email)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(PaginatedLeaksResponse{
			Items: []LeakDetails{{Username: "test@example.com", URL: "https://example.com/login"}},
			Total: 1, Page: 1, PageSize: 25,
		})
	})

	resp, err := c.SearchEmail(context.Background(), EmailSearchRequest{Email: "test@example.com"})
	if err != nil {
		t.Fatalf("SearchEmail: %v", err)
	}
	if resp.Total != 1 || len(resp.Items) != 1 || resp.Items[0].URL != "https://example.com/login" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestSearchEmail_Unauthorized(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(APIError{Detail: "invalid API key"})
	})

	_, err := c.SearchEmail(context.Background(), EmailSearchRequest{Email: "x@example.com"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T: %v", err, err)
	}
	if apiErr.StatusCode != 401 {
		t.Fatalf("StatusCode = %d, want 401", apiErr.StatusCode)
	}
	if apiErr.APIError.Message() != "invalid API key" {
		t.Fatalf("Message() = %q", apiErr.APIError.Message())
	}
}

func TestSearchEmail_RetriesOn429ThenSucceeds(t *testing.T) {
	attempts := 0
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(429)
			json.NewEncoder(w).Encode(APIError{Detail: "rate limited"})
			return
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(PaginatedLeaksResponse{Total: 0, Page: 1, PageSize: 25})
	})

	_, err := c.SearchEmail(context.Background(), EmailSearchRequest{Email: "x@example.com"})
	if err != nil {
		t.Fatalf("expected eventual success, got error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d, want 2", attempts)
	}
}

func TestSearchEmail_GivesUpAfterMaxRetries(t *testing.T) {
	attempts := 0
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(503)
		json.NewEncoder(w).Encode(APIError{Detail: map[string]interface{}{"code": "raw_unlock_busy", "message": "db locked"}})
	})

	_, err := c.SearchEmail(context.Background(), EmailSearchRequest{Email: "x@example.com"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error, got %T", err)
	}
	if apiErr.APIError.Code() != "raw_unlock_busy" {
		t.Fatalf("Code() = %q", apiErr.APIError.Code())
	}
	if attempts != c.maxRetries+1 {
		t.Fatalf("attempts = %d, want %d", attempts, c.maxRetries+1)
	}
}

func TestSearchEmail_MalformedBody(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not json"))
	})

	_, err := c.SearchEmail(context.Background(), EmailSearchRequest{Email: "x@example.com"})
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestReadLimitedRejectsOversizedBody(t *testing.T) {
	_, err := readLimited(bytes.NewReader(make([]byte, 1025)), 1024)
	if err == nil {
		t.Fatal("expected oversized body to be rejected")
	}
}

func TestDomainReportPDFStreamsResponse(t *testing.T) {
	release := make(chan struct{})
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-release
		_, _ = w.Write([]byte("pdf-body"))
	})

	type result struct {
		body io.ReadCloser
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		body, err := c.DomainReportPDF(context.Background(), "example.com")
		resultCh <- result{body: body, err: err}
	}()

	var got result
	select {
	case got = <-resultCh:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("download did not return after response headers; body was buffered")
	}
	close(release)
	if got.err != nil {
		t.Fatalf("DomainReportPDF: %v", got.err)
	}
	defer got.body.Close()
	data, err := io.ReadAll(got.body)
	if err != nil {
		t.Fatalf("reading streamed body: %v", err)
	}
	if string(data) != "pdf-body" {
		t.Fatalf("streamed body = %q", data)
	}
}

func TestDomainReport_QueryParams(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/domain/example.com" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("light") != "true" {
			t.Fatalf("light query param missing/wrong: %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(DomainSearchResponse{EmployeesCompromised: 3})
	})

	resp, err := c.DomainReport(context.Background(), "example.com", true, false)
	if err != nil {
		t.Fatalf("DomainReport: %v", err)
	}
	if resp.EmployeesCompromised != 3 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// TestDomainReport_RealFieldNames guards against regressing to paraphrased
// field names (e.g. "total_employees" instead of the real
// "employees_compromised") by decoding a fixed real-world-shaped payload,
// not a value round-tripped through our own struct.
func TestDomainReport_RealFieldNames(t *testing.T) {
	const raw = `{
		"employees_compromised": 2,
		"third_parties_compromised": 5,
		"customers_compromised": 99,
		"blacklisted_value": null,
		"searched_by_count": 3
	}`
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(raw))
	})
	resp, err := c.DomainReport(context.Background(), "example.com", false, true)
	if err != nil {
		t.Fatalf("DomainReport: %v", err)
	}
	if resp.EmployeesCompromised != 2 || resp.ThirdPartiesCompromised != 5 || resp.CustomersCompromised != 99 || resp.SearchedByCount != 3 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// TestEmailsLockedExists_UnwrapsResultsEnvelope guards against regressing to
// treating the response as a bare array: the real API wraps it in {"results": [...]}.
func TestEmailsLockedExists_UnwrapsResultsEnvelope(t *testing.T) {
	const raw = `{"results":[{"email":"john.doe@example.com","any":true,"locked":3,"total":5,"unlocked":2}]}`
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/emails/locked-exists" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Write([]byte(raw))
	})
	results, err := c.EmailsLockedExists(context.Background(), []string{"john.doe@example.com"}, true)
	if err != nil {
		t.Fatalf("EmailsLockedExists: %v", err)
	}
	if len(results) != 1 || results[0].Email != "john.doe@example.com" || results[0].Total == nil || *results[0].Total != 5 {
		t.Fatalf("unexpected results: %+v", results)
	}
}

// TestSearchAdvanced_FiltersUseRealSchema guards against regressing to the
// invented flat username/email/domain/leak_id filter shape: the real API
// only has url_domain/email_domain/url_host/email_host etc, no bare "domain".
func TestSearchAdvanced_FiltersUseRealSchema(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding body: %v", err)
		}
		if _, ok := body["domain"]; ok {
			t.Fatalf("body contains invented 'domain' field, real schema has no such field: %+v", body)
		}
		if _, ok := body["url_domain"]; !ok {
			t.Fatalf("body missing real 'url_domain' field: %+v", body)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(PaginatedLeaksResponse{Page: 1, PageSize: 25})
	})
	strength := StrengthWeak
	_, err := c.SearchAdvanced(context.Background(), LeakSearchFilters{
		URLDomain:        []string{"example.com"},
		PasswordStrength: &strength,
	})
	if err != nil {
		t.Fatalf("SearchAdvanced: %v", err)
	}
}

func TestDomainReport_EscapesPathSegment(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/search/domain/evil%2F..%2Fadmin" {
			t.Fatalf("expected escaped path, got %s", r.URL.EscapedPath())
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(DomainSearchResponse{})
	})
	_, err := c.DomainReport(context.Background(), "evil/../admin", false, false)
	if err != nil {
		t.Fatalf("DomainReport: %v", err)
	}
}

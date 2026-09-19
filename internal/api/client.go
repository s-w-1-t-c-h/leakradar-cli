// Package api is a typed client for the LeakRadar REST API
// (https://api.leakradar.io), covering email/domain/advanced/dark-web
// search, password-range lookups, batch checks, unlocks, exports and
// account profile endpoints.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"leakradar-cli/internal/ratelimit"
)

const DefaultBaseURL = "https://api.leakradar.io"

const (
	maxJSONResponseBytes  = 16 << 20 // 16 MiB is ample for paginated API responses.
	maxErrorResponseBytes = 1 << 20  // Error bodies only need enough data for diagnostics.
)

// Client talks to the LeakRadar API. It is safe for concurrent use.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	UserAgent  string

	edgeLimiter     *ratelimit.Bucket // global 30 req/s
	advancedLimiter *ratelimit.Bucket // 5 req/s for /search/advanced
	maxRetries      int
}

// New builds a Client. apiKey must not be empty for authenticated endpoints.
func New(apiKey string) *Client {
	return &Client{
		BaseURL:         DefaultBaseURL,
		APIKey:          apiKey,
		HTTPClient:      &http.Client{Timeout: 30 * time.Second},
		UserAgent:       "leakradar-cli",
		edgeLimiter:     ratelimit.NewBucket(28), // shade under the documented 30/s edge cap
		advancedLimiter: ratelimit.NewBucket(4.5),
		maxRetries:      4,
	}
}

// Error is returned for non-2xx responses, wrapping the parsed API error body.
type Error struct {
	StatusCode int
	APIError   *APIError
	RetryAfter time.Duration
	Raw        string
}

func (e *Error) Error() string {
	code := e.APIError.Code()
	msg := e.APIError.Message()
	if code != "" {
		return fmt.Sprintf("leakradar-cli API error (HTTP %d, code=%s): %s", e.StatusCode, code, msg)
	}
	return fmt.Sprintf("leakradar-cli API error (HTTP %d): %s", e.StatusCode, msg)
}

type requestOpts struct {
	method      string
	path        string
	query       url.Values
	body        interface{}
	useAdvanced bool // route through the tighter advanced-search limiter
	rawResponse bool // caller wants the raw body (e.g. PDF/file download)
}

// doJSON performs an authenticated request and decodes a JSON response into out.
func (c *Client) doJSON(ctx context.Context, opts requestOpts, out interface{}) error {
	body, raw, err := c.do(ctx, opts)
	if err != nil {
		return err
	}
	defer body.Close()
	if out == nil {
		return nil
	}
	dec := json.NewDecoder(body)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decoding response: %w (raw prefix: %.200s)", err, raw)
	}
	return nil
}

// doRaw performs an authenticated request and returns the raw response body
// for the caller to stream (e.g. PDF report / export download).
func (c *Client) doRaw(ctx context.Context, opts requestOpts) (io.ReadCloser, error) {
	body, _, err := c.do(ctx, opts)
	return body, err
}

// Note: the API has no per-export-ID GET or download-by-URL endpoint (the
// /exports list returns job status/filename only, never a fetchable URL).
// Queued CSV/TXT/JSON exports (email/domain/advanced) are retrieved from
// the LeakRadar member portal's Exports page, not this API. The synchronous
// streamed exports (DomainSubdomainsExport, DomainURLsExport, and the PDF
// report) are the only file downloads this client can do directly.

func (c *Client) do(ctx context.Context, opts requestOpts) (io.ReadCloser, string, error) {
	limiter := c.edgeLimiter
	if opts.useAdvanced {
		limiter = c.advancedLimiter
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if err := limiter.Wait(ctx); err != nil {
			return nil, "", err
		}
		if limiter != c.edgeLimiter {
			// Advanced-search calls also count against the global edge rate.
			if err := c.edgeLimiter.Wait(ctx); err != nil {
				return nil, "", err
			}
		}

		resp, err := c.attempt(ctx, opts)
		if err != nil {
			return nil, "", err
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if opts.rawResponse {
				// Downloads must remain a stream: the caller owns and closes the
				// response body, and http.Client.Timeout continues to cover reads.
				return resp.Body, "", nil
			}
			raw, err := readLimited(resp.Body, maxJSONResponseBytes)
			resp.Body.Close()
			if err != nil {
				return nil, "", fmt.Errorf("reading response: %w", err)
			}
			return io.NopCloser(bytes.NewReader(raw)), string(raw), nil
		}

		raw, readErr := readLimited(resp.Body, maxErrorResponseBytes)
		resp.Body.Close()
		if readErr != nil {
			return nil, "", fmt.Errorf("reading error response: %w", readErr)
		}

		apiErr := &APIError{}
		_ = json.Unmarshal(raw, apiErr)

		retryAfter := parseRetryAfter(resp.Header)
		cliErr := &Error{StatusCode: resp.StatusCode, APIError: apiErr, RetryAfter: retryAfter, Raw: string(raw)}

		if !shouldRetry(resp.StatusCode) || attempt == c.maxRetries {
			return nil, "", cliErr
		}

		lastErr = cliErr
		wait := retryAfter
		if wait <= 0 {
			wait = backoff(attempt)
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, "", ctx.Err()
		case <-timer.C:
		}
	}
	return nil, "", lastErr
}

func (c *Client) attempt(ctx context.Context, opts requestOpts) (*http.Response, error) {
	u := strings.TrimRight(c.BaseURL, "/") + opts.path
	if len(opts.query) > 0 {
		u += "?" + opts.query.Encode()
	}

	var bodyReader io.Reader
	if opts.body != nil {
		b, err := json.Marshal(opts.body)
		if err != nil {
			return nil, fmt.Errorf("encoding request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, opts.method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	if c.APIKey == "" {
		return nil, fmt.Errorf("no API key configured; run 'leakradar-cli auth set'")
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("User-Agent", c.UserAgent)
	if opts.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	return resp, nil
}

func readLimited(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("body exceeds %d-byte limit", limit)
	}
	return data, nil
}

func shouldRetry(status int) bool {
	return status == 429 || status == 503
}

func parseRetryAfter(h http.Header) time.Duration {
	if v := h.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	if v := h.Get("X-RateLimit-Reset"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 0
}

// backoff returns an exponential delay with jitter, capped at 20s.
func backoff(attempt int) time.Duration {
	base := math.Min(20, math.Pow(2, float64(attempt)))
	jitter := rand.Float64() * base * 0.3
	return time.Duration((base+jitter)*1000) * time.Millisecond
}

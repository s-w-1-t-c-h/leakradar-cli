package commands

import (
	"strings"
	"testing"
	"time"

	"leakradar-cli/internal/api"
)

func TestWrapRawFilterError_AddsHintForEmptyFilterError(t *testing.T) {
	err := &api.Error{
		StatusCode: 400,
		APIError:   &api.APIError{Detail: "Empty raw search is not allowed. Provide q or at least one filter (...)."},
	}
	got := wrapRawFilterError(err)
	if !strings.Contains(got.Error(), "did you forget --q") {
		t.Fatalf("expected a --q hint, got: %s", got.Error())
	}
	if !strings.Contains(got.Error(), "Empty raw search is not allowed") {
		t.Fatalf("expected the original API message to survive, got: %s", got.Error())
	}
}

func TestWrapRawFilterError_LeavesOtherErrorsUntouched(t *testing.T) {
	cases := []error{
		&api.Error{StatusCode: 401, APIError: &api.APIError{Detail: "invalid API key"}},
		&api.Error{StatusCode: 400, APIError: &api.APIError{Detail: "some unrelated validation error"}},
		&api.Error{StatusCode: 429, RetryAfter: time.Second, APIError: &api.APIError{Detail: "rate limited"}},
		nil,
	}
	for _, c := range cases {
		got := wrapRawFilterError(c)
		if c == nil {
			if got != nil {
				t.Fatalf("expected nil to stay nil, got: %v", got)
			}
			continue
		}
		if strings.Contains(got.Error(), "did you forget --q") {
			t.Fatalf("unexpected --q hint added to unrelated error: %s", got.Error())
		}
		if got.Error() != c.Error() {
			t.Fatalf("expected unrelated error to be returned unchanged, got %q want %q", got.Error(), c.Error())
		}
	}
}

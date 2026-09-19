package api

import (
	"context"
	"net/http"
	"testing"
)

// TestCountRaw_RealFieldNames decodes a fixed payload matching what the
// live API actually returned during development (a real /search/raw/count
// call for "bom.gov.au"), not a value round-tripped through our own struct.
func TestCountRaw_RealFieldNames(t *testing.T) {
	const raw = `{"blacklisted_value":null,"capped":false,"exact":true,"total":65985}`
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/raw/count" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Write([]byte(raw))
	})
	resp, err := c.CountRaw(context.Background(), RawSearchRequest{Q: "bom.gov.au"}, false)
	if err != nil {
		t.Fatalf("CountRaw: %v", err)
	}
	if resp.Total != 65985 || !resp.Exact || resp.Capped {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

// TestSearchRaw_CursorModeHasNegativeTotal decodes a fixed payload for
// cursor-mode pagination, where the API documents Total as -1.
func TestSearchRaw_CursorModeHasNegativeTotal(t *testing.T) {
	const raw = `{
		"items": [],
		"total": -1,
		"page": 1,
		"page_size": 10,
		"has_more": true,
		"next_cursor": "abc123"
	}`
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(raw))
	})
	resp, err := c.SearchRaw(context.Background(), RawSearchRequest{Q: "test"}, 1, 10, "", false)
	if err != nil {
		t.Fatalf("SearchRaw: %v", err)
	}
	if resp.Total != -1 {
		t.Fatalf("Total = %d, want -1", resp.Total)
	}
	if resp.NextCursor != "abc123" || resp.HasMore == nil || !*resp.HasMore {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRawExport_RequiresExportModeQueryParam(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("export") != "parts" {
			t.Fatalf("expected export=parts query param, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"queued","message":"ok","export_id":1}`))
	})
	_, err := c.RawExport(context.Background(), RawSearchRequest{Q: "test"}, "parts", false)
	if err != nil {
		t.Fatalf("RawExport: %v", err)
	}
}

func TestRawUnlockTask_NoListIDParam(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/raw/unlock/task" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("max") != "3" {
			t.Fatalf("expected max=3, got %s", r.URL.RawQuery)
		}
		if r.URL.Query().Get("list_id") != "" {
			t.Fatalf("raw unlock/task has no list_id support, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"task_id":"xyz:UNLOCK"}`))
	})
	task, err := c.RawUnlockTask(context.Background(), RawSearchRequest{Q: "test"}, 3, false)
	if err != nil {
		t.Fatalf("RawUnlockTask: %v", err)
	}
	if task.TaskID() != "xyz:UNLOCK" {
		t.Fatalf("unexpected task: %+v", task)
	}
}

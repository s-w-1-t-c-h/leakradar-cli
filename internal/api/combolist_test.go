package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// TestComboDomainReport_RealFieldNames decodes a fixed payload matching what
// the live API actually returned during development (captured via a
// throwaway probe against bom.gov.au), not a value round-tripped through
// our own struct — guards against the same class of bug as
// TestDomainReport_RealFieldNames in client_test.go.
func TestComboDomainReport_RealFieldNames(t *testing.T) {
	const raw = `{
		"dataset": "combolist",
		"domain": "bom.gov.au",
		"first_seen": "2026-09-03T23:52:44.664000Z",
		"last_seen": "2026-09-18T15:31:29.794000Z",
		"password_strength": {
			"medium": 261,
			"strong": 165,
			"too_weak": 128,
			"weak": 208
		},
		"total_credentials": 762,
		"unique_emails": 426
	}`
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/combolist/domain/bom.gov.au/report" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.WriteHeader(200)
		w.Write([]byte(raw))
	})
	resp, err := c.ComboDomainReport(context.Background(), "bom.gov.au")
	if err != nil {
		t.Fatalf("ComboDomainReport: %v", err)
	}
	if resp.TotalCredentials != 762 || resp.UniqueEmails != 426 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.PasswordStrength.TooWeak != 128 || resp.PasswordStrength.Weak != 208 || resp.PasswordStrength.Medium != 261 || resp.PasswordStrength.Strong != 165 {
		t.Fatalf("unexpected password strength: %+v", resp.PasswordStrength)
	}
}

// TestSearchComboEmail_LockedRowHasNullIdentifiers decodes a fixed payload
// with a locked row (id/username/password/status all null) matching what
// the live API actually returns — guards the assumption the rest of the
// combolist unlock design depends on: locked rows never expose an ID.
func TestSearchComboEmail_LockedRowHasNullIdentifiers(t *testing.T) {
	const raw = `{
		"auto_unlock_points_consumed": 0,
		"blacklisted_value": null,
		"items": [
			{
				"added_at": "2026-09-18T15:31:29.794000Z",
				"dataset": "combolist",
				"email_domain": "bom.gov.au",
				"id": null,
				"is_email": true,
				"password": null,
				"password_strength": 10,
				"status": null,
				"unlocked": false,
				"username": null,
				"username_masked": "lu●●●@bom.gov.au"
			}
		],
		"page": 1,
		"page_size": 5,
		"total": 762,
		"total_unlocked": 0
	}`
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(raw))
	})
	resp, err := c.SearchComboEmail(context.Background(), CombolistEmailSearchRequest{Email: "x@bom.gov.au"}, 1, 5, false)
	if err != nil {
		t.Fatalf("SearchComboEmail: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Items))
	}
	it := resp.Items[0]
	if it.ID != "" || it.Username != "" || it.Password != "" {
		t.Fatalf("expected empty ID/Username/Password for a locked row, got %+v", it)
	}
	if it.UsernameMasked == "" || it.Unlocked {
		t.Fatalf("expected a masked identifier and Unlocked=false, got %+v", it)
	}
}

// TestCombolistScopedRequest_OmitsUnusedFieldsByScope locks in the request
// shape sent for each scope: value-based scopes never send filters, and the
// advanced scope never sends value/search (the server rejects unknown/
// misplaced fields on CombolistAdvancedSearchRequest-derived bodies).
func TestCombolistScopedRequest_OmitsUnusedFieldsByScope(t *testing.T) {
	domainScoped := CombolistScopedRequest{Scope: "domain", Value: "example.com", Search: "admin"}
	b, err := json.Marshal(domainScoped)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["filters"]; ok {
		t.Fatalf("domain-scoped request should omit filters, got %s", b)
	}
	if m["value"] != "example.com" || m["search"] != "admin" {
		t.Fatalf("unexpected domain-scoped request: %s", b)
	}

	strength := StrengthWeak
	advancedScoped := CombolistScopedRequest{Scope: "advanced", Filters: &CombolistAdvancedSearchRequest{PasswordStrength: &strength}}
	b, err = json.Marshal(advancedScoped)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	m = nil
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["value"]; ok {
		t.Fatalf("advanced-scoped request should omit value, got %s", b)
	}
	if _, ok := m["search"]; ok {
		t.Fatalf("advanced-scoped request should omit search, got %s", b)
	}
	if _, ok := m["filters"]; !ok {
		t.Fatalf("advanced-scoped request should include filters, got %s", b)
	}
}

func TestComboUnlockTask_UsesTaskStatusShape(t *testing.T) {
	const raw = `{"task_id":"abc123:UNLOCK","running":true,"completed":false,"total":5,"updated":0}`
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/combolist/unlock/task" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("max") != "5" {
			t.Fatalf("expected max=5 query param, got %s", r.URL.RawQuery)
		}
		w.WriteHeader(200)
		w.Write([]byte(raw))
	})
	task, err := c.ComboUnlockTask(context.Background(), CombolistScopedRequest{Scope: "domain", Value: "example.com"}, 5, 0)
	if err != nil {
		t.Fatalf("ComboUnlockTask: %v", err)
	}
	if task.TaskID != "abc123:UNLOCK" || !task.Running || task.Completed {
		t.Fatalf("unexpected task: %+v", task)
	}
}

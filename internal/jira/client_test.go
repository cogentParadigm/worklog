package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetIssueIDSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/rest/api/3/issue/PROJ-123" {
			t.Errorf("expected path /rest/api/3/issue/PROJ-123, got %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("fields") != "id" {
			t.Errorf("expected fields=id, got %s", q.Get("fields"))
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", auth)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "10001"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token")
	id, err := client.GetIssueID("PROJ-123")
	if err != nil {
		t.Fatalf("GetIssueID: %v", err)
	}
	if id != "10001" {
		t.Errorf("id: got %q, want %q", id, "10001")
	}
}

func TestGetIssueIDBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Errorf("expected Authorization header")
		}
		// We just verify it starts with Basic and is non-empty
		if len(auth) < 6 || auth[:6] != "Basic " {
			t.Errorf("expected Basic auth, got %s", auth)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "20002"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "user@example.com:api-token")
	id, err := client.GetIssueID("FOO-1")
	if err != nil {
		t.Fatalf("GetIssueID: %v", err)
	}
	if id != "20002" {
		t.Errorf("id: got %q, want %q", id, "20002")
	}
}

func TestGetIssueIDError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"errorMessages": []string{"Issue does not exist."}})
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-token")
	_, err := client.GetIssueID("MISSING-1")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	expected := "Issue does not exist"
	if !contains(err.Error(), expected) {
		t.Errorf("error message should contain %q, got %q", expected, err.Error())
	}
}

func TestGetIssueIDNetworkError(t *testing.T) {
	client := NewClient("http://localhost:1", "token")
	_, err := client.GetIssueID("PROJ-1")
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestGetIssueIDMissingBaseURL(t *testing.T) {
	client := NewClient("", "token")
	_, err := client.GetIssueID("PROJ-1")
	if err == nil {
		t.Fatal("expected error for missing base URL")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

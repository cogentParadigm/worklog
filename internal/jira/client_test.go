package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
		if auth == "" {
			t.Errorf("expected Authorization header")
		}
		if len(auth) < 6 || auth[:6] != "Basic " {
			t.Errorf("expected Basic auth, got %s", auth)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "10001"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "test-token")
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

	client := NewClient(server.URL, "user@example.com", "api-token")
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

	client := NewClient(server.URL, "user", "bad-token")
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
	client := NewClient("http://localhost:1", "user", "token")
	_, err := client.GetIssueID("PROJ-1")
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestGetIssueIDMissingBaseURL(t *testing.T) {
	client := NewClient("", "user", "token")
	_, err := client.GetIssueID("PROJ-1")
	if err == nil {
		t.Fatal("expected error for missing base URL")
	}
}

func TestGetRemainingEstimateSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/rest/api/3/issue/PROJ-123" {
			t.Errorf("expected path /rest/api/3/issue/PROJ-123, got %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("fields") != "timetracking" {
			t.Errorf("expected fields=timetracking, got %s", q.Get("fields"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"fields": map[string]interface{}{
				"timetracking": map[string]interface{}{
					"remainingEstimateSeconds": 3600,
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "test-token")
	seconds, err := client.GetRemainingEstimate("PROJ-123")
	if err != nil {
		t.Fatalf("GetRemainingEstimate: %v", err)
	}
	if seconds != 3600 {
		t.Errorf("seconds: got %d, want 3600", seconds)
	}
}

func TestGetRemainingEstimateNullTimeTracking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"fields": map[string]interface{}{
				"timetracking": nil,
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "test-token")
	seconds, err := client.GetRemainingEstimate("PROJ-123")
	if err != nil {
		t.Fatalf("GetRemainingEstimate: %v", err)
	}
	if seconds != 0 {
		t.Errorf("seconds: got %d, want 0", seconds)
	}
}

func TestGetRemainingEstimateMissingTimeTracking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"fields": map[string]interface{}{},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "test-token")
	seconds, err := client.GetRemainingEstimate("PROJ-123")
	if err != nil {
		t.Fatalf("GetRemainingEstimate: %v", err)
	}
	if seconds != 0 {
		t.Errorf("seconds: got %d, want 0", seconds)
	}
}

func TestGetRemainingEstimateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"errorMessages": []string{"Issue does not exist."}})
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "bad-token")
	_, err := client.GetRemainingEstimate("MISSING-1")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	expected := "Issue does not exist"
	if !contains(err.Error(), expected) {
		t.Errorf("error message should contain %q, got %q", expected, err.Error())
	}
}

func TestGetRemainingEstimateNetworkError(t *testing.T) {
	client := NewClient("http://localhost:1", "user", "token")
	_, err := client.GetRemainingEstimate("PROJ-1")
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestGetRemainingEstimateMissingBaseURL(t *testing.T) {
	client := NewClient("", "user", "token")
	_, err := client.GetRemainingEstimate("PROJ-1")
	if err == nil {
		t.Fatal("expected error for missing base URL")
	}
}

func TestSearchIssuesSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/rest/api/3/search/jql" {
			t.Errorf("expected path /rest/api/3/search/jql, got %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("fields") != "summary" {
			t.Errorf("expected fields=summary, got %s", q.Get("fields"))
		}
		if q.Get("maxResults") != "20" {
			t.Errorf("expected maxResults=20, got %s", q.Get("maxResults"))
		}
		if !strings.Contains(q.Get("jql"), `text ~`) {
			t.Errorf("expected jql to contain text ~, got %s", q.Get("jql"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": []map[string]interface{}{
				{
					"key": "PROJ-123",
					"fields": map[string]interface{}{
						"summary": "Test issue",
					},
				},
				{
					"key": "PROJ-456",
					"fields": map[string]interface{}{
						"summary": "Another issue",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "token")
	results, err := client.SearchIssues(`text ~ "test"`)
	if err != nil {
		t.Fatalf("SearchIssues: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Key != "PROJ-123" {
		t.Errorf("key: got %q, want PROJ-123", results[0].Key)
	}
	if results[0].Summary != "Test issue" {
		t.Errorf("summary: got %q, want Test issue", results[0].Summary)
	}
}

func TestSearchIssuesJQLError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errorMessages": []string{"Field 'foo' does not exist"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "user", "token")
	_, err := client.SearchIssues(`foo = bar`)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "Field 'foo' does not exist") {
		t.Errorf("error message should contain 'Field 'foo' does not exist', got %q", err.Error())
	}
}

func TestSearchIssuesMissingBaseURL(t *testing.T) {
	client := NewClient("", "user", "token")
	_, err := client.SearchIssues(`text ~ "test"`)
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

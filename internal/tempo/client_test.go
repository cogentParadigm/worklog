package tempo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateWorklogSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/worklogs" {
			t.Errorf("expected path /worklogs, got %s", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", auth)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["issueId"] != "PROJ-123" {
			t.Errorf("issueId: got %v, want PROJ-123", body["issueId"])
		}
		if body["authorAccountId"] != "account-123" {
			t.Errorf("authorAccountId: got %v, want account-123", body["authorAccountId"])
		}
		if body["startTime"] != "09:00:00" {
			t.Errorf("startTime: got %v, want 09:00:00", body["startTime"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 42})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-token", "account-123")
	wl := Worklog{
		IssueId:          "PROJ-123",
		TimeSpentSeconds: 3600,
		StartDate:        "2026-05-06",
		StartTime:        "09:00:00",
		Description:      "Test worklog",
	}
	if err := client.CreateWorklog(wl); err != nil {
		t.Fatalf("CreateWorklog: %v", err)
	}
}

func TestCreateWorklogError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid token"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-token", "account-123")
	wl := Worklog{IssueId: "PROJ-123", TimeSpentSeconds: 3600}
	err := client.CreateWorklog(wl)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	expected := "invalid token"
	if !contains(err.Error(), expected) {
		t.Errorf("error message should contain %q, got %q", expected, err.Error())
	}
}

func TestCreateWorklogNetworkError(t *testing.T) {
	// Use an invalid URL to force a network error
	client := NewClient("http://localhost:1", "token", "account")
	wl := Worklog{IssueId: "PROJ-123", TimeSpentSeconds: 3600}
	err := client.CreateWorklog(wl)
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestNewClientDefaults(t *testing.T) {
	client := NewClient("", "token", "account")
	if client.baseURL != "https://api.tempo.io/4" {
		t.Errorf("default base URL: got %q, want %q", client.baseURL, "https://api.tempo.io/4")
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

func TestCreateWorklogPlainTextError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "Forbidden")
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", "account")
	wl := Worklog{IssueId: "PROJ-123", TimeSpentSeconds: 3600}
	err := client.CreateWorklog(wl)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

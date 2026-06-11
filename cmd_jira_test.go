package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cogentParadigm/worklog/internal/jira"
)

func TestBuildSearchJQL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"onboarding refactor", `text ~ "onboarding refactor"`},
		{"PROJ-123", `text ~ "PROJ-123"`},
		{"with \"quotes\" inside", `text ~ "with \"quotes\" inside"`},
		{"single 'quotes'", `text ~ "single 'quotes'"`},
		{"", `text ~ ""`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := buildSearchJQL(tt.input)
			if got != tt.expected {
				t.Errorf("buildSearchJQL(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func mockJiraServer(results []jira.SearchIssueResult) *httptest.Server {
	issues := make([]map[string]interface{}, len(results))
	for i, r := range results {
		issues[i] = map[string]interface{}{
			"key": r.Key,
			"fields": map[string]interface{}{
				"summary": r.Summary,
			},
		}
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"issues": issues,
		})
	}))
}

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func makeTestSkippedTask(name string) *Task {
	task := NewTask(name)
	return task
}

func TestResolveSkippedTasksAssign(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{
		{Key: "PROJ-123", Summary: "Test issue"},
	})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestSkippedTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}
	skipped := []skippedTask{{task: task, duration: 3600}}

	in := strings.NewReader("y\n\n1\n")
	out := &bytes.Buffer{}

	resolved, err := resolveSkippedTasks(skipped, client, shortUUIDs, in, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != 1 {
		t.Errorf("resolved = %d, want 1", resolved)
	}
	if task.IssueKey() != "PROJ-123" {
		t.Errorf("issue key = %q, want PROJ-123", task.IssueKey())
	}
}

func TestResolveSkippedTasksSkip(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{
		{Key: "PROJ-123", Summary: "Test issue"},
	})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestSkippedTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}
	skipped := []skippedTask{{task: task, duration: 3600}}

	in := strings.NewReader("y\n\ns\n")
	out := &bytes.Buffer{}

	resolved, err := resolveSkippedTasks(skipped, client, shortUUIDs, in, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != 0 {
		t.Errorf("resolved = %d, want 0", resolved)
	}
	if task.IssueKey() != "" {
		t.Errorf("issue key = %q, want empty", task.IssueKey())
	}
}

func TestResolveSkippedTasksQuit(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{
		{Key: "PROJ-123", Summary: "Test issue"},
	})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestSkippedTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}
	skipped := []skippedTask{{task: task, duration: 3600}}

	in := strings.NewReader("y\n\nq\n")
	out := &bytes.Buffer{}

	resolved, err := resolveSkippedTasks(skipped, client, shortUUIDs, in, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != 0 {
		t.Errorf("resolved = %d, want 0", resolved)
	}
	if task.IssueKey() != "" {
		t.Errorf("issue key = %q, want empty", task.IssueKey())
	}
}

func TestResolveSkippedTasksPartialQuit(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{
		{Key: "PROJ-123", Summary: "First issue"},
		{Key: "PROJ-456", Summary: "Second issue"},
	})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task1 := makeTestSkippedTask("Task One")
	task2 := makeTestSkippedTask("Task Two")
	shortUUIDs := map[string]string{task1.uuid: "a", task2.uuid: "b"}
	skipped := []skippedTask{
		{task: task1, duration: 3600},
		{task: task2, duration: 1800},
	}

	in := strings.NewReader("y\n\n1\n\nq\n")
	out := &bytes.Buffer{}

	resolved, err := resolveSkippedTasks(skipped, client, shortUUIDs, in, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != 1 {
		t.Errorf("resolved = %d, want 1", resolved)
	}
	if task1.IssueKey() != "PROJ-123" {
		t.Errorf("task1 issue key = %q, want PROJ-123", task1.IssueKey())
	}
	if task2.IssueKey() != "" {
		t.Errorf("task2 issue key = %q, want empty", task2.IssueKey())
	}
}

// The resolveSkippedTests above verify the in-memory mutation contract:
// resolveSkippedTasks mutates the Task objects in the skipped slice directly,
// and the caller (runJiraSync) is responsible for persisting those changes.
// This is why the tests inspect task.IssueKey() after the call and why
// runJiraSync calls worklog.Save when resolved > 0.

func TestResolveSkippedTasksDecline(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{
		{Key: "PROJ-123", Summary: "Test issue"},
	})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestSkippedTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}
	skipped := []skippedTask{{task: task, duration: 3600}}

	in := strings.NewReader("n\n")
	out := &bytes.Buffer{}

	resolved, err := resolveSkippedTasks(skipped, client, shortUUIDs, in, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != 0 {
		t.Errorf("resolved = %d, want 0", resolved)
	}
	if task.IssueKey() != "" {
		t.Errorf("issue key = %q, want empty", task.IssueKey())
	}
}

func TestResolveSkippedTasksInvalidThenValid(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{
		{Key: "PROJ-123", Summary: "Test issue"},
	})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestSkippedTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}
	skipped := []skippedTask{{task: task, duration: 3600}}

	in := strings.NewReader("y\n\nabc\n1\n")
	out := &bytes.Buffer{}

	resolved, err := resolveSkippedTasks(skipped, client, shortUUIDs, in, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != 1 {
		t.Errorf("resolved = %d, want 1", resolved)
	}
	if task.IssueKey() != "PROJ-123" {
		t.Errorf("issue key = %q, want PROJ-123", task.IssueKey())
	}
	outStr := out.String()
	if !strings.Contains(outStr, "Invalid selection.") {
		t.Errorf("output should contain 'Invalid selection.', got:\n%s", outStr)
	}
}

func TestResolveSkippedTasksMultipleTasks(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{
		{Key: "PROJ-123", Summary: "First issue"},
		{Key: "PROJ-456", Summary: "Second issue"},
	})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task1 := makeTestSkippedTask("Task One")
	task2 := makeTestSkippedTask("Task Two")
	shortUUIDs := map[string]string{task1.uuid: "a", task2.uuid: "b"}
	skipped := []skippedTask{
		{task: task1, duration: 3600},
		{task: task2, duration: 1800},
	}

	in := strings.NewReader("y\n\n1\n\n2\n")
	out := &bytes.Buffer{}

	resolved, err := resolveSkippedTasks(skipped, client, shortUUIDs, in, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != 2 {
		t.Errorf("resolved = %d, want 2", resolved)
	}
	if task1.IssueKey() != "PROJ-123" {
		t.Errorf("task1 issue key = %q, want PROJ-123", task1.IssueKey())
	}
	if task2.IssueKey() != "PROJ-456" {
		t.Errorf("task2 issue key = %q, want PROJ-456", task2.IssueKey())
	}
}

func TestResolveSkippedTasksNoResults(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestSkippedTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}
	skipped := []skippedTask{{task: task, duration: 3600}}

	in := strings.NewReader("y\n\n")
	out := &bytes.Buffer{}

	resolved, err := resolveSkippedTasks(skipped, client, shortUUIDs, in, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != 0 {
		t.Errorf("resolved = %d, want 0", resolved)
	}
	if !strings.Contains(out.String(), "No issues found.") {
		t.Errorf("output should contain 'No issues found.', got:\n%s", out.String())
	}
}

func TestResolveSkippedTasksSearchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errorMessages": []string{"bad jql"},
		})
	}))
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestSkippedTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}
	skipped := []skippedTask{{task: task, duration: 3600}}

	in := strings.NewReader("y\n\n")
	out := &bytes.Buffer{}

	_, err := resolveSkippedTasks(skipped, client, shortUUIDs, in, out)
	if err == nil {
		t.Fatal("expected error for bad JQL")
	}
	if !strings.Contains(err.Error(), "search for") {
		t.Errorf("error should mention search, got: %v", err)
	}
}

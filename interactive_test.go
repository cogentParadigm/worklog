package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cogentParadigm/worklog/internal/jira"
	"github.com/cogentParadigm/worklog/internal/tempo"
)

func makeTestTask(name string) *Task {
	task := NewTask(name)
	return task
}

func TestInteractiveSearchIssueKeyAssign(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{
		{Key: "PROJ-123", Summary: "Test issue"},
	})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// "?" to enter search, default query, pick 1
	in := strings.NewReader("?\n\n1\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditIssueKey(task, client, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	if task.IssueKey() != "PROJ-123" {
		t.Errorf("issue key = %q, want PROJ-123", task.IssueKey())
	}
}

func TestInteractiveSearchIssueKeySkip(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{
		{Key: "PROJ-123", Summary: "Test issue"},
	})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// "?" to enter search, default query, skip
	in := strings.NewReader("?\n\ns\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditIssueKey(task, client, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Errorf("changed = true, want false")
	}
	if task.IssueKey() != "" {
		t.Errorf("issue key = %q, want empty", task.IssueKey())
	}
}

func TestInteractiveSearchIssueKeyQuit(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{
		{Key: "PROJ-123", Summary: "Test issue"},
	})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// "?" to enter search, default query, quit
	in := strings.NewReader("?\n\nq\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditIssueKey(task, client, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Errorf("changed = true, want false")
	}
	if task.IssueKey() != "" {
		t.Errorf("issue key = %q, want empty", task.IssueKey())
	}
}

func TestInteractiveSearchIssueKeyDirectEntry(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// Direct entry: type the issue key directly
	in := strings.NewReader("PROJ-456\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditIssueKey(task, client, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	if task.IssueKey() != "PROJ-456" {
		t.Errorf("issue key = %q, want PROJ-456", task.IssueKey())
	}
}

func TestInteractiveSearchIssueKeyClear(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestTask("Test Task")
	task.SetIssueKey("PROJ-123")
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// Empty line to clear existing key
	in := strings.NewReader("\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditIssueKey(task, client, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	if task.IssueKey() != "" {
		t.Errorf("issue key = %q, want empty", task.IssueKey())
	}
}

func TestInteractiveSearchIssueKeyInvalidThenValid(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{
		{Key: "PROJ-123", Summary: "Test issue"},
	})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// "?" to enter search, default query, invalid input "abc", then valid "1"
	in := strings.NewReader("?\n\nabc\n1\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditIssueKey(task, client, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	if task.IssueKey() != "PROJ-123" {
		t.Errorf("issue key = %q, want PROJ-123", task.IssueKey())
	}
	outStr := out.String()
	if !strings.Contains(outStr, "Invalid selection.") {
		t.Errorf("output should contain 'Invalid selection.', got:\n%s", outStr)
	}
}

func TestInteractiveSearchIssueKeyNoResults(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// "?" to enter search, default query
	in := strings.NewReader("?\n\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditIssueKey(task, client, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Errorf("changed = true, want false")
	}
	if !strings.Contains(out.String(), "No issues found.") {
		t.Errorf("output should contain 'No issues found.', got:\n%s", out.String())
	}
}

func TestInteractiveSearchIssueKeySearchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errorMessages": []string{"bad jql"},
		})
	}))
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// "?" to enter search, default query
	in := strings.NewReader("?\n\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	_, err := interactiveEditIssueKey(task, client, shortUUIDs, reader, out)
	if err == nil {
		t.Fatal("expected error for bad JQL")
	}
	if !strings.Contains(err.Error(), "search for") {
		t.Errorf("error should mention search, got: %v", err)
	}
}

func TestInteractiveSearchIssueKeyNoJiraClient(t *testing.T) {
	task := makeTestTask("Test Task")
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// "?" to enter search but no jira client configured
	in := strings.NewReader("?\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditIssueKey(task, nil, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Errorf("changed = true, want false")
	}
	if !strings.Contains(out.String(), "Jira client not configured.") {
		t.Errorf("output should contain 'Jira client not configured.', got:\n%s", out.String())
	}
}

func TestInteractiveSelectTask(t *testing.T) {
	task1 := makeTestTask("Task One")
	task2 := makeTestTask("Task Two")
	tasks := []*Task{task1, task2}
	shortUUIDs := map[string]string{task1.uuid: "a", task2.uuid: "b"}

	in := strings.NewReader("2\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	selected, err := interactiveSelectTask(tasks, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected == nil {
		t.Fatal("selected = nil, want task2")
	}
	if selected.uuid != task2.uuid {
		t.Errorf("selected = %q, want task2", selected.name)
	}
}

func TestInteractiveSelectTaskQuit(t *testing.T) {
	task1 := makeTestTask("Task One")
	tasks := []*Task{task1}
	shortUUIDs := map[string]string{task1.uuid: "a"}

	in := strings.NewReader("q\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	selected, err := interactiveSelectTask(tasks, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected != nil {
		t.Errorf("selected = %v, want nil", selected)
	}
}

func TestInteractiveEditTaskName(t *testing.T) {
	task := makeTestTask("Old Name")
	worklog := &Worklog{tasks: []*Task{task}}
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// 1 = name, "New Name", 6 = done
	in := strings.NewReader("1\nNew Name\n6\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditTask(task, worklog, nil, nil, nil, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	if task.name != "New Name" {
		t.Errorf("name = %q, want 'New Name'", task.name)
	}
}

func TestInteractiveEditTaskDescription(t *testing.T) {
	task := makeTestTask("Test Task")
	worklog := &Worklog{tasks: []*Task{task}}
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// 2 = description, "New Description", 6 = done
	in := strings.NewReader("2\nNew Description\n6\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditTask(task, worklog, nil, nil, nil, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	if task.description != "New Description" {
		t.Errorf("description = %q, want 'New Description'", task.description)
	}
}

func TestInteractiveEditTaskClearDescription(t *testing.T) {
	task := makeTestTask("Test Task")
	task.description = "Old Description"
	worklog := &Worklog{tasks: []*Task{task}}
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// 2 = description, empty string to clear, 6 = done
	in := strings.NewReader("2\n\n6\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditTask(task, worklog, nil, nil, nil, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	if task.description != "" {
		t.Errorf("description = %q, want empty", task.description)
	}
}

func TestInteractiveEditTaskIssueKey(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{
		{Key: "PROJ-123", Summary: "Test issue"},
	})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestTask("Test Task")
	worklog := &Worklog{tasks: []*Task{task}}
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// 3 = issue key, "?" to search, default query, pick 1, 6 = done
	in := strings.NewReader("3\n?\n\n1\n6\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditTask(task, worklog, client, nil, nil, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	if task.IssueKey() != "PROJ-123" {
		t.Errorf("issue key = %q, want PROJ-123", task.IssueKey())
	}
}

func TestInteractiveEditTaskDirectIssueKey(t *testing.T) {
	task := makeTestTask("Test Task")
	worklog := &Worklog{tasks: []*Task{task}}
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// 3 = issue key, direct entry "PROJ-456", 6 = done
	in := strings.NewReader("3\nPROJ-456\n6\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditTask(task, worklog, nil, nil, nil, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	if task.IssueKey() != "PROJ-456" {
		t.Errorf("issue key = %q, want PROJ-456", task.IssueKey())
	}
}

func TestInteractiveEditTaskTempoAttributes(t *testing.T) {
	task := makeTestTask("Test Task")
	worklog := &Worklog{tasks: []*Task{task}}
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// 4 = tempo attrs, a = add, key = "_WorkType_", value = "Development", d = done, 6 = done
	in := strings.NewReader("4\na\n_WorkType_\nDevelopment\nd\n6\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditTask(task, worklog, nil, nil, nil, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	attrs := task.TempoAttributes()
	if attrs["_WorkType_"] != "Development" {
		t.Errorf("_WorkType_ = %q, want Development", attrs["_WorkType_"])
	}
}

func TestInteractiveEditTaskMultipleFields(t *testing.T) {
	server := mockJiraServer([]jira.SearchIssueResult{
		{Key: "PROJ-123", Summary: "Test issue"},
	})
	defer server.Close()

	client := jira.NewClient(server.URL, "user", "token")
	task := makeTestTask("Old Name")
	task.description = "Old Desc"
	worklog := &Worklog{tasks: []*Task{task}}
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// 1 = name, "New Name", 2 = description, "New Desc", 3 = issue key, "?", search, 1, 6 = done
	in := strings.NewReader("1\nNew Name\n2\nNew Desc\n3\n?\n\n1\n6\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditTask(task, worklog, client, nil, nil, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	if task.name != "New Name" {
		t.Errorf("name = %q, want 'New Name'", task.name)
	}
	if task.description != "New Desc" {
		t.Errorf("description = %q, want 'New Desc'", task.description)
	}
	if task.IssueKey() != "PROJ-123" {
		t.Errorf("issue key = %q, want PROJ-123", task.IssueKey())
	}
}

func TestInteractiveEditTaskNoChange(t *testing.T) {
	task := makeTestTask("Test Task")
	worklog := &Worklog{tasks: []*Task{task}}
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// 6 = done immediately
	in := strings.NewReader("6\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditTask(task, worklog, nil, nil, nil, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if changed {
		t.Errorf("changed = true, want false")
	}
}

func mockTempoServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"key":      "_WorkType_",
					"name":     "Work Type",
					"type":     "string",
					"required": false,
					"values":   []string{"Development", "Code Review", "Testing"},
					"names":    map[string]string{"Development": "Development", "Code Review": "Code Review", "Testing": "Testing"},
				},
			},
		})
	}))
}

func TestInteractiveEditTempoAttributesWithServer(t *testing.T) {
	server := mockTempoServer()
	defer server.Close()

	client := tempo.NewClient(server.URL, "token", "account")
	task := makeTestTask("Test Task")
	worklog := &Worklog{tasks: []*Task{task}}
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// 4 = tempo attrs, a = add, 1 = pick first attribute (_WorkType_), 2 = pick "Code Review", d = done, 6 = done
	in := strings.NewReader("4\na\n1\n2\nd\n6\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditTask(task, worklog, nil, client, nil, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	attrs := task.TempoAttributes()
	if attrs["_WorkType_"] != "Code Review" {
		t.Errorf("_WorkType_ = %q, want Code Review", attrs["_WorkType_"])
	}
}

func TestInteractiveEditTempoAttributesClear(t *testing.T) {
	task := makeTestTask("Test Task")
	task.SetTempoAttribute("_WorkType_", "Development")
	worklog := &Worklog{tasks: []*Task{task}}
	shortUUIDs := map[string]string{task.uuid: "abc"}

	// 4 = tempo attrs, c = clear all, d = done, 6 = done
	in := strings.NewReader("4\nc\nd\n6\n")
	out := &bytes.Buffer{}
	reader := bufio.NewReader(in)

	changed, err := interactiveEditTask(task, worklog, nil, nil, nil, shortUUIDs, reader, out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !changed {
		t.Errorf("changed = false, want true")
	}
	attrs := task.TempoAttributes()
	if len(attrs) != 0 {
		t.Errorf("attrs = %v, want empty", attrs)
	}
}

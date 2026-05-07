package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out)
}

func copyExampleToTemp(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.ics")
	data, err := os.ReadFile("testdata/example.ics")
	if err != nil {
		t.Fatalf("failed to read example.ics: %v", err)
	}
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return tmpFile
}

func TestRunNoCommand(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{})
		if err == nil {
			t.Error("expected error for no command")
		}
	})
	if out == "" {
		t.Error("expected usage output on missing command")
	}
}

func TestRunOldRootCommandsRejected(t *testing.T) {
	err := run([]string{"list"})
	if err == nil {
		t.Fatal("expected error for old root-level command")
	}
	if err.Error() != "unknown command 'list'" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunTaskNamespaceList(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"task", "list", "-file", "testdata/example.ics"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
	if out == "" {
		t.Error("expected list output")
	}
}

func TestRunTaskNamespaceMissingSubcommand(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"task"})
		if err == nil {
			t.Error("expected error for missing subcommand")
		}
	})
	if out == "" {
		t.Error("expected usage output on missing subcommand")
	}
}

func TestRunTaskUnknownSubcommand(t *testing.T) {
	err := run([]string{"task", "foo"})
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	if err.Error() != "unknown task subcommand 'foo'" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunTaskCreate(t *testing.T) {
	tmpFile := copyExampleToTemp(t)

	err := run([]string{"task", "create", "-file", tmpFile, "-name", "CLI Test Task"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	wl, err := NewWorklog(tmpFile)
	if err != nil {
		t.Fatalf("failed to reload worklog: %v", err)
	}
	found := false
	for _, task := range wl.tasks {
		if task.name == "CLI Test Task" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected created task to be found in reloaded worklog")
	}
}

func TestRunTaskUpdate(t *testing.T) {
	tmpFile := copyExampleToTemp(t)

	err := run([]string{"task", "update", "-file", tmpFile, "-uuid", "a96e0dd3-1321-4bad-a58d-e256e23d44d8", "-name", "Updated Name"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	wl, err := NewWorklog(tmpFile)
	if err != nil {
		t.Fatalf("failed to reload worklog: %v", err)
	}
	task := wl.FindTaskByUUID("a96e0dd3-1321-4bad-a58d-e256e23d44d8")
	if task == nil {
		t.Fatal("expected to find updated task")
	}
	if task.name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", task.name)
	}
}

func TestRunTaskDelete(t *testing.T) {
	tmpFile := copyExampleToTemp(t)

	err := run([]string{"task", "delete", "-file", tmpFile, "-uuid", "a96e0dd3-1321-4bad-a58d-e256e23d44d8", "-force"})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	wl, err := NewWorklog(tmpFile)
	if err != nil {
		t.Fatalf("failed to reload worklog: %v", err)
	}
	if wl.FindTaskByUUID("a96e0dd3-1321-4bad-a58d-e256e23d44d8") != nil {
		t.Error("expected task to be deleted")
	}
}

func TestHelpTaskList(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"task", "list", "-h"})
		if err != flag.ErrHelp {
			t.Errorf("expected flag.ErrHelp, got %v", err)
		}
	})
	if !strings.Contains(out, "Usage: worklog task list") {
		t.Errorf("expected usage output, got:\n%s", out)
	}
}

func TestHelpTaskCreate(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"task", "create", "--help"})
		if err != flag.ErrHelp {
			t.Errorf("expected flag.ErrHelp, got %v", err)
		}
	})
	if !strings.Contains(out, "Usage: worklog task create") {
		t.Errorf("expected usage output, got:\n%s", out)
	}
	if !strings.Contains(out, "Examples:") {
		t.Error("expected Examples section in usage output")
	}
}

func TestHelpTimeAdd(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"time", "add", "-h"})
		if err != flag.ErrHelp {
			t.Errorf("expected flag.ErrHelp, got %v", err)
		}
	})
	if !strings.Contains(out, "Usage: worklog time add") {
		t.Errorf("expected usage output, got:\n%s", out)
	}
}

func TestHelpReportTimesheet(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"report", "timesheet", "--help"})
		if err != flag.ErrHelp {
			t.Errorf("expected flag.ErrHelp, got %v", err)
		}
	})
	if !strings.Contains(out, "Usage: worklog report timesheet") {
		t.Errorf("expected usage output, got:\n%s", out)
	}
}

func TestHelpTopLevel(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"-h"})
		if err != flag.ErrHelp {
			t.Errorf("expected flag.ErrHelp, got %v", err)
		}
	})
	if !strings.Contains(out, "Usage: worklog <command>") {
		t.Errorf("expected top-level usage, got:\n%s", out)
	}
}

func TestHelpTopLevelDoubleDash(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"--help"})
		if err != flag.ErrHelp {
			t.Errorf("expected flag.ErrHelp, got %v", err)
		}
	})
	if !strings.Contains(out, "Available commands:") {
		t.Errorf("expected top-level usage, got:\n%s", out)
	}
}

func TestHelpTaskNamespace(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"task", "-h"})
		if err != flag.ErrHelp {
			t.Errorf("expected flag.ErrHelp, got %v", err)
		}
	})
	if !strings.Contains(out, "Usage: worklog task <subcommand>") {
		t.Errorf("expected task namespace usage, got:\n%s", out)
	}
}

func TestHelpTimeNamespace(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"time", "--help"})
		if err != flag.ErrHelp {
			t.Errorf("expected flag.ErrHelp, got %v", err)
		}
	})
	if !strings.Contains(out, "Usage: worklog time <subcommand>") {
		t.Errorf("expected time namespace usage, got:\n%s", out)
	}
}

func TestHelpReportNamespace(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"report", "-h"})
		if err != flag.ErrHelp {
			t.Errorf("expected flag.ErrHelp, got %v", err)
		}
	})
	if !strings.Contains(out, "Usage: worklog report <subcommand>") {
		t.Errorf("expected report namespace usage, got:\n%s", out)
	}
}

func TestRunTimeListDateRange(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"time", "list", "-file", "testdata/example.ics", "-from", "2023-08-14", "-to", "2023-08-17"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
	// Should show events from 08/14 and 08/17 (4 total)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least header and data rows, got:\n%s", out)
	}
	// Header is first line, data lines follow
	dataLines := lines[1:]
	if len(dataLines) != 4 {
		t.Errorf("expected 4 events in range, got %d\n%s", len(dataLines), out)
	}
}

func TestRunTimeListFromOnly(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"time", "list", "-file", "testdata/example.ics", "-from", "2023-08-18"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected output, got:\n%s", out)
	}
	dataLines := lines[1:]
	if len(dataLines) != 10 {
		t.Errorf("expected 10 events on/after 2023-08-18, got %d\n%s", len(dataLines), out)
	}
}

func TestRunTimeListInvalidDateRange(t *testing.T) {
	err := run([]string{"time", "list", "-file", "testdata/example.ics", "-from", "2023-08-18", "-to", "2023-08-14"})
	if err == nil {
		t.Fatal("expected error for invalid date range")
	}
	if !strings.Contains(err.Error(), "from date must not be after to date") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunTimeListSearchTaskName(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"time", "list", "-file", "testdata/example.ics", "-search", "stax"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected output, got:\n%s", out)
	}
	dataLines := lines[1:]
	if len(dataLines) != 6 {
		t.Errorf("expected 6 events matching 'stax', got %d\n%s", len(dataLines), out)
	}
	for _, line := range dataLines {
		if !strings.Contains(line, "stax") {
			t.Errorf("expected each line to contain 'stax', got: %s", line)
		}
	}
}

func TestRunTimeListSearchNoMatch(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"time", "list", "-file", "testdata/example.ics", "-search", "nonexistent"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Errorf("expected only header for no-match search, got %d lines:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], "UUID") {
		t.Errorf("expected header line, got: %s", lines[0])
	}
}

func TestRunTaskListSearch(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"task", "list", "-file", "testdata/example.ics", "-search", "stax"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected output, got:\n%s", out)
	}
	// Header + 3 matching tasks
	if len(lines) != 4 {
		t.Errorf("expected 3 matching tasks (+header = 4 lines), got %d\n%s", len(lines), out)
	}
	for _, line := range lines[1:] {
		if !strings.Contains(strings.ToLower(line), "stax") {
			t.Errorf("expected each data line to contain 'stax', got: %s", line)
		}
	}
}

func TestRunTaskListSearchNoMatch(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"task", "list", "-file", "testdata/example.ics", "-search", "zzzzzzz"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
	if !strings.Contains(out, "No tasks match the search criteria.") {
		t.Errorf("expected no-match message, got:\n%s", out)
	}
}

func TestRunTaskListParent(t *testing.T) {
	// a96e0dd3-1321-4bad-a58d-e256e23d44d8 is "08/18" which has 4 children
	out := captureStdout(func() {
		err := run([]string{"task", "list", "-file", "testdata/example.ics", "-parent", "a96e0dd3-1321-4bad-a58d-e256e23d44d8"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected output, got:\n%s", out)
	}
	// Header + 5 tasks (parent + 4 children)
	if len(lines) != 6 {
		t.Errorf("expected 5 tasks (+header = 6 lines), got %d\n%s", len(lines), out)
	}
}

func TestRunTaskListParentInvalidUUID(t *testing.T) {
	err := run([]string{"task", "list", "-file", "testdata/example.ics", "-parent", "invalid-uuid"})
	if err == nil {
		t.Fatal("expected error for invalid parent UUID")
	}
	if !strings.Contains(err.Error(), "parent task with UUID 'invalid-uuid' not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRunTaskListParentAndSearch(t *testing.T) {
	// Parent "08/18" with search "stax" -> should show "08/18 - josh - stax" and "08/18 - stax internal check in"
	out := captureStdout(func() {
		err := run([]string{"task", "list", "-file", "testdata/example.ics", "-parent", "a96e0dd3-1321-4bad-a58d-e256e23d44d8", "-search", "stax"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected output, got:\n%s", out)
	}
	// Header + 2 matching tasks within subtree
	if len(lines) != 3 {
		t.Errorf("expected 2 matching tasks (+header = 3 lines), got %d\n%s", len(lines), out)
	}
}

func TestRunTimeListCombinedFilters(t *testing.T) {
	out := captureStdout(func() {
		err := run([]string{"time", "list", "-file", "testdata/example.ics", "-from", "2023-08-14", "-to", "2023-08-17", "-search", "ACCPLAN"})
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected output, got:\n%s", out)
	}
	dataLines := lines[1:]
	// Events in 08/14-08/17 matching ACCPLAN: 2 events on 08/14 for task ace3355a (ACCPLAN-90)
	if len(dataLines) != 2 {
		t.Errorf("expected 2 events with combined filters, got %d\n%s", len(dataLines), out)
	}
	for _, line := range dataLines {
		if !strings.Contains(line, "ACCPLAN") {
			t.Errorf("expected each line to contain 'ACCPLAN', got: %s", line)
		}
	}
}

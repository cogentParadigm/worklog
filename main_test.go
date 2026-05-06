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

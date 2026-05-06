package main

import (
	"io"
	"os"
	"path/filepath"
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

package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ics "github.com/arran4/golang-ical"
)

func captureStderr(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	f()
	w.Close()
	os.Stderr = old
	out, _ := io.ReadAll(r)
	return string(out)
}

func TestSidecarRestoresStrippedTaskMetadata(t *testing.T) {
	dir := t.TempDir()
	icsPath := filepath.Join(dir, "test.ics")

	// Create initial .ics with worklog metadata
	cal := ics.NewCalendar()
	todo := ics.VTodo{}
	todo.SetProperty(ics.ComponentPropertyUniqueId, "task-1")
	todo.SetProperty(ics.ComponentPropertySummary, "Task One")
	todo.SetProperty("X-WORKLOG-ISSUE-ID", "1001")
	todo.SetProperty("X-WORKLOG-ISSUE-KEY", "DEMO-42")
	todo.Properties = append(todo.Properties, ics.IANAProperty{
		BaseProperty: ics.BaseProperty{
			IANAToken:      "X-WORKLOG-TEMPO-ATTR",
			ICalParameters: map[string][]string{"KEY": {"account"}},
			Value:          "engineering",
		},
	})
	cal.Components = append(cal.Components, &todo)

	if err := saveCalendar(icsPath, cal); err != nil {
		t.Fatal(err)
	}

	// Load worklog and save to create sidecar
	wl, err := NewWorklog(icsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := wl.Save(""); err != nil {
		t.Fatal(err)
	}

	// Verify sidecar exists
	if _, err := os.Stat(icsPath + ".worklog"); os.IsNotExist(err) {
		t.Fatal("sidecar not created")
	}

	// Simulate external tool stripping metadata: rewrite .ics without X-WORKLOG-*
	cal2 := ics.NewCalendar()
	todo2 := ics.VTodo{}
	todo2.SetProperty(ics.ComponentPropertyUniqueId, "task-1")
	todo2.SetProperty(ics.ComponentPropertySummary, "Task One")
	cal2.Components = append(cal2.Components, &todo2)

	if err := saveCalendar(icsPath, cal2); err != nil {
		t.Fatal(err)
	}

	// Reload: metadata should be restored from sidecar
	oldVerbose := verbose
	verbose = true
	defer func() { verbose = oldVerbose }()
	stderr := captureStderr(func() {
		wl2, err := NewWorklog(icsPath)
		if err != nil {
			t.Fatal(err)
		}

		task := wl2.FindTaskByUUID("task-1")
		if task == nil {
			t.Fatal("task not found")
		}
		if task.IssueID() != "1001" {
			t.Errorf("expected issue id 1001, got %s", task.IssueID())
		}
		if task.IssueKey() != "DEMO-42" {
			t.Errorf("expected issue key DEMO-42, got %s", task.IssueKey())
		}
		attrs := task.TempoAttributes()
		if attrs["account"] != "engineering" {
			t.Errorf("expected tempo attr account=engineering, got %v", attrs)
		}
	})

	if !strings.Contains(stderr, "Restored metadata for task \"Task One\"") {
		t.Errorf("expected stderr notice about restored metadata, got: %s", stderr)
	}
}

func TestSidecarRestoresStrippedEventMetadata(t *testing.T) {
	dir := t.TempDir()
	icsPath := filepath.Join(dir, "test.ics")

	// Create initial .ics with event metadata
	cal := ics.NewCalendar()
	ve := cal.AddEvent("event-1")
	start := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	ve.SetStartAt(start)
	ve.SetEndAt(end)
	ve.SetSummary("Work session")
	ve.SetProperty("RELATED-TO", "task-1")
	ve.SetProperty("X-WORKLOG-SYNCED-AT", "20240115T120000Z")
	ve.SetProperty("X-WORKLOG-SYNC-HASH", "payloadhash123")

	if err := saveCalendar(icsPath, cal); err != nil {
		t.Fatal(err)
	}

	// Load and save to create sidecar
	wl, err := NewWorklog(icsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := wl.Save(""); err != nil {
		t.Fatal(err)
	}

	// Simulate external tool stripping metadata
	cal2 := ics.NewCalendar()
	ve2 := cal2.AddEvent("event-1")
	ve2.SetStartAt(start)
	ve2.SetEndAt(end)
	ve2.SetSummary("Work session")
	ve2.SetProperty("RELATED-TO", "task-1")

	if err := saveCalendar(icsPath, cal2); err != nil {
		t.Fatal(err)
	}

	// Reload: metadata should be restored
	oldVerbose := verbose
	verbose = true
	defer func() { verbose = oldVerbose }()
	stderr := captureStderr(func() {
		wl2, err := NewWorklog(icsPath)
		if err != nil {
			t.Fatal(err)
		}

		event := wl2.FindEventByUUID("event-1")
		if event == nil {
			t.Fatal("event not found")
		}
		expected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		if !event.SyncedAt().Equal(expected) {
			t.Errorf("expected synced_at %v, got %v", expected, event.SyncedAt())
		}
		if event.SyncHash() != "payloadhash123" {
			t.Errorf("expected sync_hash restored, got %q", event.SyncHash())
		}
	})

	if !strings.Contains(stderr, "Restored metadata for event \"Work session\"") {
		t.Errorf("expected stderr notice about restored event metadata, got: %s", stderr)
	}
}

func TestSidecarDoesNotRestoreWhenPropertiesPresent(t *testing.T) {
	dir := t.TempDir()
	icsPath := filepath.Join(dir, "test.ics")

	// Create .ics with metadata
	cal := ics.NewCalendar()
	todo := ics.VTodo{}
	todo.SetProperty(ics.ComponentPropertyUniqueId, "task-1")
	todo.SetProperty(ics.ComponentPropertySummary, "Task One")
	todo.SetProperty("X-WORKLOG-ISSUE-ID", "1001")
	cal.Components = append(cal.Components, &todo)

	if err := saveCalendar(icsPath, cal); err != nil {
		t.Fatal(err)
	}

	// Load and save to create sidecar
	wl, err := NewWorklog(icsPath)
	if err != nil {
		t.Fatal(err)
	}
	wl.Save("")

	// Reload without stripping: should NOT print restore notice
	stderr := captureStderr(func() {
		_, err := NewWorklog(icsPath)
		if err != nil {
			t.Fatal(err)
		}
	})

	if strings.Contains(stderr, "Restored metadata") {
		t.Errorf("expected no restore notice when metadata is present, got: %s", stderr)
	}
}

func TestSidecarSummaryWhenNotVerbose(t *testing.T) {
	dir := t.TempDir()
	icsPath := filepath.Join(dir, "test.ics")

	// Create initial .ics with worklog metadata
	cal := ics.NewCalendar()
	todo := ics.VTodo{}
	todo.SetProperty(ics.ComponentPropertyUniqueId, "task-1")
	todo.SetProperty(ics.ComponentPropertySummary, "Task One")
	todo.SetProperty("X-WORKLOG-ISSUE-ID", "1001")
	cal.Components = append(cal.Components, &todo)

	if err := saveCalendar(icsPath, cal); err != nil {
		t.Fatal(err)
	}

	// Load and save to create sidecar
	wl, err := NewWorklog(icsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := wl.Save(""); err != nil {
		t.Fatal(err)
	}

	// Strip metadata
	cal2 := ics.NewCalendar()
	todo2 := ics.VTodo{}
	todo2.SetProperty(ics.ComponentPropertyUniqueId, "task-1")
	todo2.SetProperty(ics.ComponentPropertySummary, "Task One")
	cal2.Components = append(cal2.Components, &todo2)

	if err := saveCalendar(icsPath, cal2); err != nil {
		t.Fatal(err)
	}

	// Reload with verbose=false (default): should print concise summary only
	verbose = false
	stderr := captureStderr(func() {
		_, err := NewWorklog(icsPath)
		if err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(stderr, "Restored metadata for 1 task(s) from sidecar") {
		t.Errorf("expected concise summary, got: %s", stderr)
	}
	if strings.Contains(stderr, "Task One") {
		t.Errorf("expected no per-task detail when not verbose, got: %s", stderr)
	}
}

func TestSidecarRoundTripPreservesMetadata(t *testing.T) {
	dir := t.TempDir()
	icsPath := filepath.Join(dir, "test.ics")

	cal := ics.NewCalendar()
	todo := ics.VTodo{}
	todo.SetProperty(ics.ComponentPropertyUniqueId, "task-1")
	todo.SetProperty(ics.ComponentPropertySummary, "Task One")
	todo.SetProperty("X-WORKLOG-ISSUE-ID", "1001")
	cal.Components = append(cal.Components, &todo)

	if err := saveCalendar(icsPath, cal); err != nil {
		t.Fatal(err)
	}

	// First load/save creates sidecar
	wl, err := NewWorklog(icsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := wl.Save(""); err != nil {
		t.Fatal(err)
	}

	// Second load should not need restoration
	wl2, err := NewWorklog(icsPath)
	if err != nil {
		t.Fatal(err)
	}

	task := wl2.FindTaskByUUID("task-1")
	if task == nil {
		t.Fatal("task not found")
	}
	if task.IssueID() != "1001" {
		t.Errorf("expected issue id 1001 after round trip, got %s", task.IssueID())
	}
}

func TestSidecarSaveOnDifferentOutputPath(t *testing.T) {
	dir := t.TempDir()
	icsPath := filepath.Join(dir, "original.ics")
	outPath := filepath.Join(dir, "output.ics")

	cal := ics.NewCalendar()
	todo := ics.VTodo{}
	todo.SetProperty(ics.ComponentPropertyUniqueId, "task-1")
	todo.SetProperty(ics.ComponentPropertySummary, "Task One")
	cal.Components = append(cal.Components, &todo)

	if err := saveCalendar(icsPath, cal); err != nil {
		t.Fatal(err)
	}

	wl, err := NewWorklog(icsPath)
	if err != nil {
		t.Fatal(err)
	}
	wl.NewTask("New Task")
	if err := wl.Save(outPath); err != nil {
		t.Fatal(err)
	}

	// Sidecar should be next to output file
	if _, err := os.Stat(outPath + ".worklog"); os.IsNotExist(err) {
		t.Fatal("sidecar not created next to output file")
	}
	// Original should NOT have sidecar (we didn't save there)
	if _, err := os.Stat(icsPath + ".worklog"); !os.IsNotExist(err) {
		t.Error("unexpected sidecar next to original file")
	}
}

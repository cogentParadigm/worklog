package main

import (
	"testing"
	"time"

	ics "github.com/arran4/golang-ical"
)

func TestTaskIssueKeyExplicit(t *testing.T) {
	task := &Task{
		name: "Review PROJ-123 code",
		properties: []ics.IANAProperty{
			{BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-ISSUE-KEY", Value: "EXPLICIT-456"}},
		},
	}
	if got := task.IssueKey(); got != "EXPLICIT-456" {
		t.Errorf("explicit issue key: got %q, want %q", got, "EXPLICIT-456")
	}
}

func TestTaskIssueKeyAutoDetect(t *testing.T) {
	task := &Task{name: "Review PROJ-123 code"}
	if got := task.IssueKey(); got != "PROJ-123" {
		t.Errorf("auto-detected issue key: got %q, want %q", got, "PROJ-123")
	}
}

func TestTaskIssueKeyNoMatch(t *testing.T) {
	task := &Task{name: "General planning meeting"}
	if got := task.IssueKey(); got != "" {
		t.Errorf("no match: got %q, want empty", got)
	}
}

func TestTaskIssueKeyNumbersInProject(t *testing.T) {
	task := &Task{name: "Work on FOO2-456 migration"}
	if got := task.IssueKey(); got != "FOO2-456" {
		t.Errorf("numbers in project key: got %q, want %q", got, "FOO2-456")
	}
}

func TestEventSyncedAt(t *testing.T) {
	event := &Event{
		properties: []ics.IANAProperty{
			{BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-SYNCED-AT", Value: "20260506T120000Z"}},
		},
	}
	got := event.SyncedAt()
	want := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("SyncedAt: got %v, want %v", got, want)
	}
}

func TestEventSyncedAtEmpty(t *testing.T) {
	event := &Event{properties: []ics.IANAProperty{}}
	if !event.SyncedAt().IsZero() {
		t.Error("expected zero time for missing synced-at")
	}
}

func TestEventSetSyncedAt(t *testing.T) {
	event := &Event{properties: []ics.IANAProperty{}}
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	event.SetSyncedAt(now)
	if event.SyncedAt() != now {
		t.Errorf("after SetSyncedAt: got %v, want %v", event.SyncedAt(), now)
	}

	// Verify property exists
	found := false
	for _, prop := range event.properties {
		if prop.IANAToken == "X-WORKLOG-SYNCED-AT" && prop.Value == "20260506T120000Z" {
			found = true
			break
		}
	}
	if !found {
		t.Error("X-WORKLOG-SYNCED-AT property not found after SetSyncedAt")
	}
}

func TestEventSetSyncedAtUpdatesExisting(t *testing.T) {
	event := &Event{
		properties: []ics.IANAProperty{
			{BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-SYNCED-AT", Value: "20260505T100000Z"}},
		},
	}
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	event.SetSyncedAt(now)
	if len(event.properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(event.properties))
	}
	if event.properties[0].Value != "20260506T120000Z" {
		t.Errorf("updated value: got %q, want %q", event.properties[0].Value, "20260506T120000Z")
	}
}

func TestEventLastModified(t *testing.T) {
	event := &Event{
		properties: []ics.IANAProperty{
			{BaseProperty: ics.BaseProperty{IANAToken: "LAST-MODIFIED", Value: "20260506T120000Z"}},
		},
	}
	got := event.LastModified()
	want := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("LastModified: got %v, want %v", got, want)
	}
}

func TestBuildSyncEntriesBasic(t *testing.T) {
	task := NewTask("Implement PROJ-123 feature")
	task.uuid = "task-uuid-1"

	event := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 30, 0, 0, time.UTC),
		5400, task.name, "Worked on API")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event}}

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].issueKey != "PROJ-123" {
		t.Errorf("issue key: got %q, want %q", entries[0].issueKey, "PROJ-123")
	}
	if len(entries[0].events) != 1 {
		t.Errorf("expected 1 source event, got %d", len(entries[0].events))
	}
	if entries[0].events[0].uuid != event.uuid {
		t.Error("source event mismatch")
	}
}

func TestBuildSyncEntriesAggregatesSameTaskSameDay(t *testing.T) {
	task := NewTask("Matt - ABC-100")
	task.uuid = "task-matt"

	event1 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 9, 10, 0, 0, time.UTC),
		600, task.name, "Standup sync")
	event2 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 14, 10, 0, 0, time.UTC),
		600, task.name, "Follow-up discussion")
	event3 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 16, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 16, 10, 0, 0, time.UTC),
		600, task.name, "Follow-up discussion") // duplicate comment

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event1, event2, event3}}

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 aggregated entry, got %d", len(entries))
	}
	if entries[0].duration != 1800 {
		t.Errorf("duration: got %d, want 1800", entries[0].duration)
	}
	if entries[0].start.Hour() != 9 {
		t.Errorf("start time should be earliest event: got hour %d, want 9", entries[0].start.Hour())
	}
	wantComment := "Standup sync; Follow-up discussion"
	if entries[0].comment != wantComment {
		t.Errorf("comment: got %q, want %q", entries[0].comment, wantComment)
	}
	if len(entries[0].events) != 3 {
		t.Errorf("source events: got %d, want 3", len(entries[0].events))
	}
}

func TestBuildSyncEntriesSeparateTasksSameDay(t *testing.T) {
	taskMatt := NewTask("Matt - ABC-100")
	taskMatt.uuid = "task-matt"
	taskNathan := NewTask("Nathan - ABC-100")
	taskNathan.uuid = "task-nathan"

	eventMatt := NewEvent(taskMatt.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 9, 10, 0, 0, time.UTC),
		600, taskMatt.name, "Sprint planning")
	eventNathan1 := NewEvent(taskNathan.uuid,
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 10, 0, 0, time.UTC),
		600, taskNathan.name, "Code review")
	eventNathan2 := NewEvent(taskNathan.uuid,
		time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 14, 10, 0, 0, time.UTC),
		600, taskNathan.name, "Code review")
	eventNathan3 := NewEvent(taskNathan.uuid,
		time.Date(2026, 5, 6, 16, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 16, 10, 0, 0, time.UTC),
		600, taskNathan.name, "Architecture discussion")

	wl := &Worklog{
		tasks:  []*Task{taskMatt, taskNathan},
		events: []*Event{eventMatt, eventNathan1, eventNathan2, eventNathan3},
	}

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (one per task), got %d", len(entries))
	}

	// Sort by duration to have deterministic order
	if entries[0].duration > entries[1].duration {
		entries[0], entries[1] = entries[1], entries[0]
	}

	if entries[0].duration != 600 {
		t.Errorf("Matt duration: got %d, want 600", entries[0].duration)
	}
	if entries[0].comment != "Sprint planning" {
		t.Errorf("Matt comment: got %q, want %q", entries[0].comment, "Sprint planning")
	}
	if entries[1].duration != 1800 {
		t.Errorf("Nathan duration: got %d, want 1800", entries[1].duration)
	}
	wantNathanComment := "Code review; Architecture discussion"
	if entries[1].comment != wantNathanComment {
		t.Errorf("Nathan comment: got %q, want %q", entries[1].comment, wantNathanComment)
	}
}

func TestBuildSyncEntriesNoComments(t *testing.T) {
	task := NewTask("ABC-100")
	task.uuid = "task-1"

	event1 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 9, 10, 0, 0, time.UTC),
		600, task.name, "")
	event2 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 10, 0, 0, time.UTC),
		600, task.name, "")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event1, event2}}

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].comment != "" {
		t.Errorf("empty comment: got %q, want empty", entries[0].comment)
	}
	if entries[0].duration != 1200 {
		t.Errorf("duration: got %d, want 1200", entries[0].duration)
	}
}

func TestBuildSyncEntriesSkipsFullySyncedGroup(t *testing.T) {
	task := NewTask("Implement PROJ-123 feature")
	task.uuid = "task-uuid-2"

	event := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 30, 0, 0, time.UTC),
		5400, task.name, "Worked on API")
	event.SetSyncedAt(time.Now())

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event}}

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for fully synced group, got %d", len(entries))
	}
}

func TestBuildSyncEntriesPartialSyncTriggersReSend(t *testing.T) {
	task := NewTask("Implement PROJ-123 feature")
	task.uuid = "task-uuid-partial"

	event1 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		3600, task.name, "Morning work")
	event1.SetSyncedAt(time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC))

	event2 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 15, 0, 0, 0, time.UTC),
		3600, task.name, "Afternoon work")
	// event2 is not synced

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event1, event2}}

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (group re-sent), got %d", len(entries))
	}
	if entries[0].duration != 7200 {
		t.Errorf("duration: got %d, want 7200 (both events summed)", entries[0].duration)
	}
	if len(entries[0].events) != 2 {
		t.Errorf("source events: got %d, want 2", len(entries[0].events))
	}
}

func TestBuildSyncEntriesDateRange(t *testing.T) {
	task := NewTask("Implement PROJ-123 feature")
	task.uuid = "task-uuid-3"

	event := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 30, 0, 0, time.UTC),
		5400, task.name, "Worked on API")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event}}

	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)
	entries, err := buildSyncEntries(wl, "", from, to)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries outside range, got %d", len(entries))
	}

	from = time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	to = time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	entries, err = buildSyncEntries(wl, "", from, to)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry inside range, got %d", len(entries))
	}
}

func TestBuildSyncEntriesTaskFilter(t *testing.T) {
	task1 := NewTask("Task one PROJ-123")
	task1.uuid = "task-uuid-a"
	task2 := NewTask("Task two PROJ-456")
	task2.uuid = "task-uuid-b"

	event1 := NewEvent(task1.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		3600, task1.name, "Work 1")
	event2 := NewEvent(task2.uuid,
		time.Date(2026, 5, 6, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
		3600, task2.name, "Work 2")

	wl := &Worklog{tasks: []*Task{task1, task2}, events: []*Event{event1, event2}}

	entries, err := buildSyncEntries(wl, "task-uuid-b", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].task.uuid != "task-uuid-b" {
		t.Errorf("wrong task: got %q, want %q", entries[0].task.uuid, "task-uuid-b")
	}
}

func TestBuildSyncEntriesNoIssueKey(t *testing.T) {
	task := NewTask("General planning")
	task.uuid = "task-uuid-4"

	event := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		3600, task.name, "Planning")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event}}

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for task without issue key, got %d", len(entries))
	}
}

func TestBuildSyncEntriesReSyncAfterEdit(t *testing.T) {
	task := NewTask("Implement PROJ-123")
	task.uuid = "task-uuid-5"

	event := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		3600, task.name, "Initial work")

	// Simulate: synced yesterday, then edited today
	event.SetSyncedAt(time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC))
	// Update last-modified to today
	for i, prop := range event.properties {
		if prop.IANAToken == "LAST-MODIFIED" {
			event.properties[i].Value = "20260506T120000Z"
		}
	}

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event}}

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry for re-sync after edit, got %d", len(entries))
	}
}

func TestBuildSyncEntriesNoReSyncIfUnchanged(t *testing.T) {
	task := NewTask("Implement PROJ-123")
	task.uuid = "task-uuid-6"

	event := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		3600, task.name, "Initial work")

	// Synced after last edit
	event.SetSyncedAt(time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC))
	for i, prop := range event.properties {
		if prop.IANAToken == "LAST-MODIFIED" {
			event.properties[i].Value = "20260506T100000Z"
		}
	}

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event}}

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries when synced after last modified, got %d", len(entries))
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("short string: got %q", got)
	}
	long := "this is a very long string indeed"
	if got := truncate(long, 10); got != "this is..." {
		t.Errorf("truncated: got %q, want %q", got, "this is...")
	}
}

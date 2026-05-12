package main

import (
	"strings"
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

func TestTaskIssueID(t *testing.T) {
	task := &Task{
		name: "Review PROJ-123 code",
		properties: []ics.IANAProperty{
			{BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-ISSUE-ID", Value: "10001"}},
		},
	}
	if got := task.IssueID(); got != "10001" {
		t.Errorf("issue id: got %q, want %q", got, "10001")
	}
}

func TestTaskIssueIDEmpty(t *testing.T) {
	task := &Task{name: "General task"}
	if got := task.IssueID(); got != "" {
		t.Errorf("expected empty issue id, got %q", got)
	}
}

func TestTaskSetIssueID(t *testing.T) {
	task := &Task{name: "Test"}
	task.SetIssueID("20002")
	if got := task.IssueID(); got != "20002" {
		t.Errorf("after SetIssueID: got %q, want %q", got, "20002")
	}
}

func TestTaskSetIssueIDUpdatesExisting(t *testing.T) {
	task := &Task{
		name: "Test",
		properties: []ics.IANAProperty{
			{BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-ISSUE-ID", Value: "100"}},
		},
	}
	task.SetIssueID("200")
	if got := task.IssueID(); got != "200" {
		t.Errorf("after update: got %q, want %q", got, "200")
	}
	if len(task.properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(task.properties))
	}
}

func TestTaskClearIssueID(t *testing.T) {
	task := &Task{
		name: "Test",
		properties: []ics.IANAProperty{
			{BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-ISSUE-ID", Value: "100"}},
		},
	}
	task.ClearIssueID()
	if got := task.IssueID(); got != "" {
		t.Errorf("after ClearIssueID: got %q, want empty", got)
	}
	if len(task.properties) != 0 {
		t.Fatalf("expected 0 properties, got %d", len(task.properties))
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

func TestEventSyncHash(t *testing.T) {
	event := &Event{
		properties: []ics.IANAProperty{
			{BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-SYNC-HASH", Value: "abc123"}},
		},
	}
	if got := event.SyncHash(); got != "abc123" {
		t.Errorf("SyncHash: got %q, want %q", got, "abc123")
	}
}

func TestEventSyncHashEmpty(t *testing.T) {
	event := &Event{properties: []ics.IANAProperty{}}
	if event.SyncHash() != "" {
		t.Error("expected empty hash for missing sync-hash")
	}
}

func TestEventSetSyncHash(t *testing.T) {
	event := &Event{properties: []ics.IANAProperty{}}
	event.SetSyncHash("hash456")
	if event.SyncHash() != "hash456" {
		t.Errorf("after SetSyncHash: got %q, want %q", event.SyncHash(), "hash456")
	}
}

func TestEventSetSyncHashUpdatesExisting(t *testing.T) {
	event := &Event{
		properties: []ics.IANAProperty{
			{BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-SYNC-HASH", Value: "oldhash"}},
		},
	}
	event.SetSyncHash("newhash")
	if len(event.properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(event.properties))
	}
	if event.properties[0].Value != "newhash" {
		t.Errorf("updated value: got %q, want %q", event.properties[0].Value, "newhash")
	}
}

func TestTaskTempoAttributes(t *testing.T) {
	task := &Task{
		properties: []ics.IANAProperty{
			{BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-TEMPO-ATTR", ICalParameters: map[string][]string{"KEY": {"_WorkType_"}}, Value: "Development"}},
			{BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-TEMPO-ATTR", ICalParameters: map[string][]string{"KEY": {"_Billable_"}}, Value: "Yes"}},
			{BaseProperty: ics.BaseProperty{IANAToken: "X-OTHER", Value: "ignore"}},
		},
	}
	attrs := task.TempoAttributes()
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(attrs))
	}
	if attrs["_WorkType_"] != "Development" {
		t.Errorf("_WorkType_: got %q, want Development", attrs["_WorkType_"])
	}
	if attrs["_Billable_"] != "Yes" {
		t.Errorf("_Billable_: got %q, want Yes", attrs["_Billable_"])
	}
}

func TestTaskSetTempoAttribute(t *testing.T) {
	task := &Task{}
	task.SetTempoAttribute("_WorkType_", "Development")
	attrs := task.TempoAttributes()
	if attrs["_WorkType_"] != "Development" {
		t.Errorf("after SetTempoAttribute: got %q, want Development", attrs["_WorkType_"])
	}
}

func TestTaskSetTempoAttributeUpdatesExisting(t *testing.T) {
	task := &Task{
		properties: []ics.IANAProperty{
			{BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-TEMPO-ATTR", ICalParameters: map[string][]string{"KEY": {"_WorkType_"}}, Value: "Review"}},
		},
	}
	task.SetTempoAttribute("_WorkType_", "Development")
	if len(task.properties) != 1 {
		t.Fatalf("expected 1 property, got %d", len(task.properties))
	}
	if task.properties[0].Value != "Development" {
		t.Errorf("updated value: got %q, want Development", task.properties[0].Value)
	}
}

func TestTaskClearTempoAttribute(t *testing.T) {
	task := &Task{
		properties: []ics.IANAProperty{
			{BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-TEMPO-ATTR", ICalParameters: map[string][]string{"KEY": {"_WorkType_"}}, Value: "Development"}},
			{BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-TEMPO-ATTR", ICalParameters: map[string][]string{"KEY": {"_Billable_"}}, Value: "Yes"}},
		},
	}
	task.ClearTempoAttribute("_WorkType_")
	attrs := task.TempoAttributes()
	if _, ok := attrs["_WorkType_"]; ok {
		t.Error("expected _WorkType_ to be cleared")
	}
	if attrs["_Billable_"] != "Yes" {
		t.Errorf("_Billable_: got %q, want Yes", attrs["_Billable_"])
	}
}

func TestTaskTempoAttributesRoundTrip(t *testing.T) {
	task := NewTask("Review code")
	task.SetTempoAttribute("_WorkType_", "Development")
	task.SetTempoAttribute("_Billable_", "Yes")

	todo := makeTodoForTask(task)
	cal := ics.NewCalendar()
	cal.Components = append(cal.Components, &todo)

	serialized := cal.Serialize()
	parsed, err := ics.ParseCalendar(strings.NewReader(serialized))
	if err != nil {
		t.Fatalf("parse serialized calendar: %v", err)
	}

	todos := getTodos(parsed)
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}

	roundTrip, _ := makeTaskForTodo(todos[0])
	attrs := roundTrip.TempoAttributes()
	if len(attrs) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(attrs))
	}
	if attrs["_WorkType_"] != "Development" {
		t.Errorf("_WorkType_: got %q, want Development", attrs["_WorkType_"])
	}
	if attrs["_Billable_"] != "Yes" {
		t.Errorf("_Billable_: got %q, want Yes", attrs["_Billable_"])
	}
}

func TestHumanizeLabel(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"_WorkType_", "Work Type"},
		{"_Billable_", "Billable"},
		{"_SomeOtherAttr_", "Some Other Attr"},
		{"simple", "simple"},
		{"", ""},
		{"ProjectManagement", "Project Management"},
		{"Project Management", "Project Management"},
		{"Already Spaced", "Already Spaced"},
	}
	for _, c := range cases {
		got := humanizeLabel(c.input)
		if got != c.expected {
			t.Errorf("humanizeLabel(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}

func TestBuildHumanizedAttrKeys(t *testing.T) {
	labels := buildHumanizedAttrKeys([]string{"_WorkType_", "_Billable_"})
	if len(labels) != 2 {
		t.Fatalf("expected 2 labels, got %d", len(labels))
	}
	if labels[0] != "Work Type" {
		t.Errorf("label 0: got %q, want Work Type", labels[0])
	}
	if labels[1] != "Billable" {
		t.Errorf("label 1: got %q, want Billable", labels[1])
	}

	// duplicate humanized labels should be disambiguated
	labelsDup := buildHumanizedAttrKeys([]string{"_WorkType_", "WorkType"})
	if labelsDup[0] != "Work Type" {
		t.Errorf("label 0: got %q, want Work Type", labelsDup[0])
	}
	if labelsDup[1] != "Work Type (WorkType)" {
		t.Errorf("label 1: got %q, want 'Work Type (WorkType)'", labelsDup[1])
	}
}

func TestMergedTempoAttributes(t *testing.T) {
	task := NewTask("Test")
	task.SetTempoAttribute("_WorkType_", "Development")
	task.SetTempoAttribute("_Billable_", "Yes")

	cfg := &Config{
		Tempo: TempoConfig{
			Attributes: map[string]string{
				"_WorkType_": "Review",
				"_Location_": "Office",
			},
		},
	}

	attrs := mergedTempoAttributes(cfg, task)
	if len(attrs) != 3 {
		t.Fatalf("expected 3 attributes, got %d", len(attrs))
	}
	if attrs["_WorkType_"] != "Development" {
		t.Errorf("_WorkType_: got %q, want Development", attrs["_WorkType_"])
	}
	if attrs["_Location_"] != "Office" {
		t.Errorf("_Location_: got %q, want Office", attrs["_Location_"])
	}
	if attrs["_Billable_"] != "Yes" {
		t.Errorf("_Billable_: got %q, want Yes", attrs["_Billable_"])
	}

	// nil config should return only task attributes
	attrsNil := mergedTempoAttributes(nil, task)
	if len(attrsNil) != 2 {
		t.Fatalf("expected 2 attributes with nil config, got %d", len(attrsNil))
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

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
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
	task.description = "Daily standup tasks"

	event1 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 9, 10, 0, 0, time.UTC),
		600, task.name, "")
	event2 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 14, 10, 0, 0, time.UTC),
		600, task.name, "")
	event3 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 16, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 16, 10, 0, 0, time.UTC),
		600, task.name, "")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event1, event2, event3}}

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
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
	wantComment := "Daily standup tasks"
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
	taskMatt.description = "Sprint planning and reviews"
	taskNathan := NewTask("Nathan - ABC-100")
	taskNathan.uuid = "task-nathan"
	taskNathan.description = "Code review and architecture"

	eventMatt := NewEvent(taskMatt.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 9, 10, 0, 0, time.UTC),
		600, taskMatt.name, "")
	eventNathan1 := NewEvent(taskNathan.uuid,
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 10, 0, 0, time.UTC),
		600, taskNathan.name, "")
	eventNathan2 := NewEvent(taskNathan.uuid,
		time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 14, 10, 0, 0, time.UTC),
		600, taskNathan.name, "")
	eventNathan3 := NewEvent(taskNathan.uuid,
		time.Date(2026, 5, 6, 16, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 16, 10, 0, 0, time.UTC),
		600, taskNathan.name, "")

	wl := &Worklog{
		tasks:  []*Task{taskMatt, taskNathan},
		events: []*Event{eventMatt, eventNathan1, eventNathan2, eventNathan3},
	}

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
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
	if entries[0].comment != "Sprint planning and reviews" {
		t.Errorf("Matt comment: got %q, want %q", entries[0].comment, "Sprint planning and reviews")
	}
	if entries[1].duration != 1800 {
		t.Errorf("Nathan duration: got %d, want 1800", entries[1].duration)
	}
	wantNathanComment := "Code review and architecture"
	if entries[1].comment != wantNathanComment {
		t.Errorf("Nathan comment: got %q, want %q", entries[1].comment, wantNathanComment)
	}
}

func TestBuildSyncEntriesNoComments(t *testing.T) {
	task := NewTask("ABC-100")
	task.uuid = "task-1"
	task.description = "Task description fallback"

	event1 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 9, 10, 0, 0, time.UTC),
		600, task.name, "")
	event2 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 10, 0, 0, time.UTC),
		600, task.name, "")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event1, event2}}

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].comment != "Task description fallback" {
		t.Errorf("comment: got %q, want %q", entries[0].comment, "Task description fallback")
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

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event}}

	// Compute expected hash from initial build
	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for initial build, got %d", len(entries))
	}
	event.SetSyncHash(entries[0].syncHash)
	event.SetSyncedAt(time.Now())

	// Now it should be skipped
	entries, err = buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
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

	event2 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 15, 0, 0, 0, time.UTC),
		3600, task.name, "Afternoon work")

	// Sync event1 alone first
	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event1}}
	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for event1 alone, got %d", len(entries))
	}
	event1.SetSyncHash(entries[0].syncHash)
	event1.SetSyncedAt(time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC))

	// Add event2: group hash changes because duration and events differ
	wl.events = []*Event{event1, event2}
	entries, err = buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
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
	entries, err := buildSyncEntries(wl, "", from, to, nil, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries outside range, got %d", len(entries))
	}

	from = time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	to = time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	entries, err = buildSyncEntries(wl, "", from, to, nil, nil)
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

	entries, err := buildSyncEntries(wl, "task-uuid-b", time.Time{}, time.Time{}, nil, nil)
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

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
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
	task.description = "Initial work"

	event := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		3600, task.name, "")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event}}

	// Build and sync
	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for initial build, got %d", len(entries))
	}
	event.SetSyncHash(entries[0].syncHash)

	// Edit task description (changes comment/payload)
	task.description = "Updated work"

	entries, err = buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
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
	task.description = "Initial work"

	event := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		3600, task.name, "")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event}}

	// Build and set sync hash
	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for initial build, got %d", len(entries))
	}
	event.SetSyncHash(entries[0].syncHash)
	event.SetSyncedAt(time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC))

	// Rebuild — unchanged, should skip
	entries, err = buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries when unchanged, got %d", len(entries))
	}
}

func TestBuildSyncEntriesExternalAppBumpsLastModified(t *testing.T) {
	task := NewTask("Implement PROJ-123")
	task.uuid = "task-uuid-lm"
	task.description = "Some work"

	event := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		3600, task.name, "")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event}}

	// Build and set sync hash
	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for initial build, got %d", len(entries))
	}
	event.SetSyncHash(entries[0].syncHash)
	event.SetSyncedAt(time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC))

	// Simulate external app (e.g. KTimeTracker) bumping LAST-MODIFIED
	for i, prop := range event.properties {
		if prop.IANAToken == "LAST-MODIFIED" {
			event.properties[i].Value = "20260506T200000Z"
		}
	}

	// Should still skip — hash unchanged despite LAST-MODIFIED bump
	entries, err = buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after external LAST-MODIFIED bump, got %d", len(entries))
	}
}

func TestBuildSyncEntriesReSyncAfterIssueKeyChange(t *testing.T) {
	task := NewTask("Implement PROJ-123")
	task.uuid = "task-uuid-ik"
	task.description = "Work"

	event := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		3600, task.name, "")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event}}

	// Build and sync
	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for initial build, got %d", len(entries))
	}
	event.SetSyncHash(entries[0].syncHash)
	event.SetSyncedAt(time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC))

	// Change task name so issue key auto-detection changes
	task.name = "Implement PROJ-456"

	entries, err = buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry for re-sync after issue key change, got %d", len(entries))
	}
	if entries[0].issueKey != "PROJ-456" {
		t.Errorf("issue key: got %q, want PROJ-456", entries[0].issueKey)
	}
}

func TestBuildSyncEntriesReSyncAfterTempoAttributeChange(t *testing.T) {
	task := NewTask("Implement PROJ-123")
	task.uuid = "task-uuid-attr"
	task.description = "Work"

	event := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		3600, task.name, "")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event}}

	// Build and sync with default attribute from config
	cfg := &Config{
		Tempo: TempoConfig{
			Attributes: map[string]string{"_WorkType_": "Development"},
		},
	}
	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, cfg)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for initial build, got %d", len(entries))
	}
	event.SetSyncHash(entries[0].syncHash)
	event.SetSyncedAt(time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC))

	// Change task-level Tempo attribute
	task.SetTempoAttribute("_WorkType_", "Review")

	entries, err = buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, cfg)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry for re-sync after tempo attr change, got %d", len(entries))
	}
	if entries[0].task.TempoAttributes()["_WorkType_"] != "Review" {
		t.Errorf("tempo attr: got %q, want Review", entries[0].task.TempoAttributes()["_WorkType_"])
	}
}

func TestBuildTimesheetFromSyncEntries(t *testing.T) {
	task1 := NewTask("Task A PROJ-1")
	task1.uuid = "task-a"
	task2 := NewTask("Task B PROJ-2")
	task2.uuid = "task-b"

	entries := []syncEntry{
		{task: task1, start: time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC), duration: 3600},
		{task: task1, start: time.Date(2026, 5, 7, 10, 0, 0, 0, time.UTC), duration: 1800},
		{task: task2, start: time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC), duration: 7200},
	}

	ts := buildTimesheetFromSyncEntries(entries, time.Time{}, time.Time{})

	if len(ts.days) != 2 {
		t.Fatalf("expected 2 days, got %d", len(ts.days))
	}
	if !ts.days[0].Equal(time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("day 0: got %v", ts.days[0])
	}
	if !ts.days[1].Equal(time.Date(2026, 5, 7, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("day 1: got %v", ts.days[1])
	}

	if len(ts.rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(ts.rows))
	}
	if ts.rows[0].task.name != "Task A PROJ-1" {
		t.Errorf("row 0 task: got %s", ts.rows[0].task.name)
	}
	if ts.rows[0].durations[0] != 3600 || ts.rows[0].durations[1] != 1800 {
		t.Errorf("row 0 durations: got %v", ts.rows[0].durations)
	}
	if ts.rows[1].task.name != "Task B PROJ-2" {
		t.Errorf("row 1 task: got %s", ts.rows[1].task.name)
	}
	if ts.rows[1].durations[0] != 7200 || ts.rows[1].durations[1] != 0 {
		t.Errorf("row 1 durations: got %v", ts.rows[1].durations)
	}

	if ts.totals[0] != 10800 || ts.totals[1] != 1800 {
		t.Errorf("totals: got %v", ts.totals)
	}
}

func TestBuildTimesheetFromSyncEntriesWithRange(t *testing.T) {
	task := NewTask("Task A PROJ-1")
	task.uuid = "task-a"

	entries := []syncEntry{
		{task: task, start: time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC), duration: 3600},
	}

	from := time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	ts := buildTimesheetFromSyncEntries(entries, from, to)

	if len(ts.days) != 4 {
		t.Fatalf("expected 4 days, got %d", len(ts.days))
	}
	if ts.rows[0].durations[0] != 0 || ts.rows[0].durations[1] != 3600 {
		t.Errorf("durations: got %v", ts.rows[0].durations)
	}
}

func TestBuildTimesheetFromSyncEntriesAggregatesSameTaskSameDay(t *testing.T) {
	task := NewTask("Task A PROJ-1")
	task.uuid = "task-a"

	entries := []syncEntry{
		{task: task, start: time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC), duration: 3600},
		{task: task, start: time.Date(2026, 5, 6, 14, 0, 0, 0, time.UTC), duration: 1800},
	}

	ts := buildTimesheetFromSyncEntries(entries, time.Time{}, time.Time{})

	if len(ts.rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(ts.rows))
	}
	if ts.rows[0].durations[0] != 5400 {
		t.Errorf("expected 5400, got %d", ts.rows[0].durations[0])
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

func TestCollectAllTasks(t *testing.T) {
	root1 := NewTask("Root 1")
	child1 := NewTask("Child 1")
	grandchild1 := NewTask("Grandchild 1")
	root2 := NewTask("Root 2")

	root1.children = []*Task{child1}
	child1.parent = root1
	child1.children = []*Task{grandchild1}
	grandchild1.parent = child1

	tasks := []*Task{root1, root2}
	all := flattenTasks(tasks)

	if len(all) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(all))
	}

	want := []string{"Root 1", "Child 1", "Grandchild 1", "Root 2"}
	for i, task := range all {
		if task.name != want[i] {
			t.Errorf("task %d: got %q, want %q", i, task.name, want[i])
		}
	}
}

func TestParseRoundingSteps(t *testing.T) {
	tests := []struct {
		input   string
		want    []RoundingStep
		wantErr bool
	}{
		{"", nil, false},
		{"floor:1m", []RoundingStep{{Step: "floor", To: "1m"}}, false},
		{"floor:1m,ceil:5m", []RoundingStep{{Step: "floor", To: "1m"}, {Step: "ceil", To: "5m"}}, false},
		{"  floor : 1m , ceil : 5m  ", []RoundingStep{{Step: "floor", To: "1m"}, {Step: "ceil", To: "5m"}}, false},
		{"round:30s", []RoundingStep{{Step: "round", To: "30s"}}, false},
		{"invalid", nil, true},
		{"foo:1m", nil, true},
		{"floor:", nil, true},
	}
	for _, tt := range tests {
		got, err := parseRoundingSteps(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseRoundingSteps(%q) expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseRoundingSteps(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if len(got) != len(tt.want) {
			t.Errorf("parseRoundingSteps(%q) len=%d, want %d", tt.input, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i].Step != tt.want[i].Step || got[i].To != tt.want[i].To {
				t.Errorf("parseRoundingSteps(%q)[%d] = {%s, %s}, want {%s, %s}", tt.input, i, got[i].Step, got[i].To, tt.want[i].Step, tt.want[i].To)
			}
		}
	}
}

func TestApplyRounding(t *testing.T) {
	tests := []struct {
		name   string
		steps  []RoundingStep
		input  int
		want   int
		wantErr bool
	}{
		{"no rounding", nil, 359, 359, false},
		{"floor 1m exact", []RoundingStep{{Step: "floor", To: "1m"}}, 300, 300, false},
		{"floor 1m truncate", []RoundingStep{{Step: "floor", To: "1m"}}, 359, 300, false},
		{"ceil 5m exact", []RoundingStep{{Step: "ceil", To: "5m"}}, 300, 300, false},
		{"ceil 5m up", []RoundingStep{{Step: "ceil", To: "5m"}}, 301, 600, false},
		{"round 1m down", []RoundingStep{{Step: "round", To: "1m"}}, 29, 0, false},
		{"round 1m up", []RoundingStep{{Step: "round", To: "1m"}}, 31, 60, false},
		{"round 1m exact", []RoundingStep{{Step: "round", To: "1m"}}, 30, 60, false},
		{"floor 1m then ceil 5m - 5m59s", []RoundingStep{{Step: "floor", To: "1m"}, {Step: "ceil", To: "5m"}}, 359, 300, false},
		{"floor 1m then ceil 5m - 6m1s", []RoundingStep{{Step: "floor", To: "1m"}, {Step: "ceil", To: "5m"}}, 361, 600, false},
		{"floor 1m then ceil 5m - 10m exact", []RoundingStep{{Step: "floor", To: "1m"}, {Step: "ceil", To: "5m"}}, 600, 600, false},
		{"floor 1m then ceil 5m - 11m", []RoundingStep{{Step: "floor", To: "1m"}, {Step: "ceil", To: "5m"}}, 660, 900, false},
		{"invalid step", []RoundingStep{{Step: "invalid", To: "1m"}}, 100, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyRounding(tt.steps, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("applyRounding(%v, %d) = %d, want %d", tt.steps, tt.input, got, tt.want)
			}
		})
	}
}

func TestBuildSyncEntriesWithRounding(t *testing.T) {
	task := NewTask("ABC-100")
	task.uuid = "task-1"

	// 5m 59s
	event1 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 9, 5, 59, 0, time.UTC),
		359, task.name, "")
	// 6m 1s
	event2 := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 6, 1, 0, time.UTC),
		361, task.name, "")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event1, event2}}

	steps := []RoundingStep{{Step: "floor", To: "1m"}, {Step: "ceil", To: "5m"}}
	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, steps, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	// 359 -> floor to 300 -> ceil to 300 = 5m
	// 361 -> floor to 360 -> ceil to 600 = 10m
	// total = 900s = 15m
	if entries[0].duration != 900 {
		t.Errorf("duration: got %d, want 900", entries[0].duration)
	}
}

func TestBuildSyncEntriesRoundingToZero(t *testing.T) {
	task := NewTask("ABC-100")
	task.uuid = "task-1"

	event := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 9, 0, 29, 0, time.UTC),
		29, task.name, "")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{event}}

	steps := []RoundingStep{{Step: "floor", To: "1m"}}
	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, steps, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after rounding to zero, got %d", len(entries))
	}
}

func TestBuildSyncEntriesSkipsInProgressEvents(t *testing.T) {
	task := NewTask("Implement PROJ-123 feature")
	task.uuid = "task-uuid-inprogress"

	finished := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		3600, task.name, "Finished work")

	inProgress := NewEvent(task.uuid,
		time.Date(2026, 5, 6, 11, 0, 0, 0, time.UTC),
		time.Time{}, // no DTEND
		0, task.name, "In progress work")

	wl := &Worklog{tasks: []*Task{task}, events: []*Event{finished, inProgress}}

	entries, err := buildSyncEntries(wl, "", time.Time{}, time.Time{}, nil, nil)
	if err != nil {
		t.Fatalf("buildSyncEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry (finished only), got %d", len(entries))
	}
	if entries[0].duration != 3600 {
		t.Errorf("duration: got %d, want 3600", entries[0].duration)
	}
	if len(entries[0].events) != 1 {
		t.Errorf("expected 1 source event, got %d", len(entries[0].events))
	}
	if entries[0].events[0].uuid != finished.uuid {
		t.Error("expected finished event, got in-progress event")
	}
}

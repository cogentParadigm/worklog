package main

import (
	"strings"
	"testing"
)

func TestShortUUID(t *testing.T) {
	tests := []struct {
		name     string
		uuid     string
		all      []string
		expected string
	}{
		{
			name:     "single uuid",
			uuid:     "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			all:      []string{"a1b2c3d4-e5f6-7890-abcd-ef1234567890"},
			expected: "a1b2",
		},
		{
			name:     "unique after 7 chars",
			uuid:     "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			all:      []string{"a1b2c3d4-e5f6-7890-abcd-ef1234567890", "a1b2c3ff-aaaa-bbbb-cccc-dddddddddddd"},
			expected: "a1b2c3d",
		},
		{
			name:     "unique after 4 chars",
			uuid:     "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			all:      []string{"a1b2c3d4-e5f6-7890-abcd-ef1234567890", "b1b2c3ff-aaaa-bbbb-cccc-dddddddddddd"},
			expected: "a1b2",
		},
		{
			name:     "full uuid when identical",
			uuid:     "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			all:      []string{"a1b2c3d4-e5f6-7890-abcd-ef1234567890", "a1b2c3d4-e5f6-7890-abcd-ef1234567890"},
			expected: "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		},
		{
			name:     "uuid shorter than min len",
			uuid:     "abc",
			all:      []string{"abc"},
			expected: "abc",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := shortUUID(tc.uuid, tc.all)
			if result != tc.expected {
				t.Errorf("shortUUID() = %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestShortUUIDs(t *testing.T) {
	uuids := []string{
		"a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"a1b2c3ff-aaaa-bbbb-cccc-dddddddddddd",
		"b1b2c3d4-e5f6-7890-abcd-ef1234567890",
	}
	result := shortUUIDs(uuids)
	if result[uuids[0]] != "a1b2c3d" {
		t.Errorf("expected %q to be shortened to a1b2c3d, got %q", uuids[0], result[uuids[0]])
	}
	if result[uuids[1]] != "a1b2c3f" {
		t.Errorf("expected %q to be shortened to a1b2c3f, got %q", uuids[1], result[uuids[1]])
	}
	if result[uuids[2]] != "b1b2" {
		t.Errorf("expected %q to be shortened to b1b2, got %q", uuids[2], result[uuids[2]])
	}
}

func TestResolveTaskUUIDExact(t *testing.T) {
	w := createTestWorklog()
	task := NewTask("test task")
	w.tasks = append(w.tasks, task)

	found, err := resolveTaskUUID(w, task.uuid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.uuid != task.uuid {
		t.Errorf("expected %q, got %q", task.uuid, found.uuid)
	}
}

func TestResolveTaskUUIDPrefix(t *testing.T) {
	w := createTestWorklog()
	task := NewTask("test task")
	w.tasks = append(w.tasks, task)

	prefix := task.uuid[:8]
	found, err := resolveTaskUUID(w, prefix)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.uuid != task.uuid {
		t.Errorf("expected %q, got %q", task.uuid, found.uuid)
	}
}

func TestResolveTaskUUIDNotFound(t *testing.T) {
	w := createTestWorklog()
	_, err := resolveTaskUUID(w, "zzzz")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestResolveTaskUUIDAmbiguous(t *testing.T) {
	w := createTestWorklog()
	t1 := NewTask("task 1")
	t2 := NewTask("task 2")
	// Force UUIDs with same prefix
	t1.uuid = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	t2.uuid = "a1b2c3ff-aaaa-bbbb-cccc-dddddddddddd"
	w.tasks = append(w.tasks, t1, t2)

	_, err := resolveTaskUUID(w, "a1b2")
	if err == nil {
		t.Fatal("expected error for ambiguous prefix")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "task 1") || !strings.Contains(err.Error(), "task 2") {
		t.Errorf("expected candidate names in error, got: %v", err)
	}
}

func TestResolveEventUUIDExact(t *testing.T) {
	w := createTestWorklog()
	event := &Event{uuid: "e1b2c3d4-e5f6-7890-abcd-ef1234567890", relatedTo: "task-uuid"}
	w.events = append(w.events, event)

	found, err := resolveEventUUID(w, event.uuid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.uuid != event.uuid {
		t.Errorf("expected %q, got %q", event.uuid, found.uuid)
	}
}

func TestResolveEventUUIDNotFound(t *testing.T) {
	w := createTestWorklog()
	_, err := resolveEventUUID(w, "zzzz")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestResolveEventUUIDAmbiguous(t *testing.T) {
	w := createTestWorklog()
	e1 := &Event{uuid: "a1b2c3d4-e5f6-7890-abcd-ef1234567890", relatedTo: "task-uuid"}
	e2 := &Event{uuid: "a1b2c3ff-aaaa-bbbb-cccc-dddddddddddd", relatedTo: "task-uuid"}
	w.events = append(w.events, e1, e2)

	_, err := resolveEventUUID(w, "a1b2")
	if err == nil {
		t.Fatal("expected error for ambiguous prefix")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected 'ambiguous' in error, got: %v", err)
	}
}

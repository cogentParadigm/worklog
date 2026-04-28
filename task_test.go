package main

import (
	"testing"

	ics "github.com/arran4/golang-ical"
)

func TestPropertyPreservation(t *testing.T) {
	cal := ics.NewCalendar()
	todo := ics.VTodo{}
	todo.SetProperty(ics.ComponentPropertyUniqueId, "test-uuid-123")
	todo.SetProperty(ics.ComponentPropertySummary, "Test Task")
	todo.SetProperty(ics.ComponentPropertyDescription, "Test Description")
	todo.SetProperty("CREATED", "20240101T000000Z")
	todo.SetProperty("PERCENT-COMPLETE", "50")
	todo.SetProperty("X-KDE-ktimetracker-totalSessionTime", "3600")
	todo.SetProperty("X-KDE-ktimetracker-totalTaskTime", "7200")
	cal.Components = append(cal.Components, &todo)

	tasks := getTasksForTodos(cal)
	if len(tasks) != 1 {
		t.Fatalf("Expected 1 task, got %d", len(tasks))
	}
	task := tasks[0]
	if len(task.properties) != 4 {
		t.Fatalf("Expected 4 preserved properties, got %d", len(task.properties))
	}

	outCal := getCalendarForTasks(tasks)
	outTodos := getTodos(outCal)
	if len(outTodos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(outTodos))
	}
	outTodo := outTodos[0]

	expectedProps := map[string]string{
		"CREATED":                             "20240101T000000Z",
		"PERCENT-COMPLETE":                    "50",
		"X-KDE-ktimetracker-totalSessionTime": "3600",
		"X-KDE-ktimetracker-totalTaskTime":    "7200",
	}
	for token, expectedVal := range expectedProps {
		prop := outTodo.GetProperty(ics.ComponentProperty(token))
		if prop == nil {
			t.Errorf("Expected property %s to be preserved, but was missing", token)
			continue
		}
		if prop.Value != expectedVal {
			t.Errorf("Expected property %s to have value %s, got %s", token, expectedVal, prop.Value)
		}
	}
}

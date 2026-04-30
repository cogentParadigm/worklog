package main

import (
	"testing"

	ics "github.com/arran4/golang-ical"
)

func TestMakeTasksForTodosPreservesOrder(t *testing.T) {
	cal := ics.NewCalendar()

	todo1 := ics.VTodo{}
	todo1.SetProperty(ics.ComponentPropertyUniqueId, "uuid-1")
	todo1.SetProperty(ics.ComponentPropertySummary, "Task 1")

	todo2 := ics.VTodo{}
	todo2.SetProperty(ics.ComponentPropertyUniqueId, "uuid-2")
	todo2.SetProperty(ics.ComponentPropertySummary, "Task 2")
	todo2.SetProperty("RELATED-TO", "uuid-1")

	todo3 := ics.VTodo{}
	todo3.SetProperty(ics.ComponentPropertyUniqueId, "uuid-3")
	todo3.SetProperty(ics.ComponentPropertySummary, "Task 3")

	todo4 := ics.VTodo{}
	todo4.SetProperty(ics.ComponentPropertyUniqueId, "uuid-4")
	todo4.SetProperty(ics.ComponentPropertySummary, "Task 4")
	todo4.SetProperty("RELATED-TO", "uuid-1")

	cal.Components = append(cal.Components, &todo1, &todo2, &todo3, &todo4)
	todos := getTodos(cal)

	// Call multiple times to verify deterministic ordering
	var firstOrder []string
	for i := 0; i < 10; i++ {
		tasks := makeTasksForTodos(todos)

		var order []string
		var collect func([]*Task)
		collect = func(ts []*Task) {
			for _, task := range ts {
				order = append(order, task.uuid)
				collect(task.children)
			}
		}
		collect(tasks)

		if i == 0 {
			firstOrder = order
		} else {
			if len(order) != len(firstOrder) {
				t.Fatalf("Iteration %d: expected %d tasks, got %d", i, len(firstOrder), len(order))
			}
			for j, uid := range firstOrder {
				if order[j] != uid {
					t.Fatalf("Iteration %d: order mismatch at position %d: expected %s, got %s", i, j, firstOrder[j], order[j])
				}
			}
		}
	}

	// Verify order matches input-derived expectation:
	// root-1 (first in input), child-2 (second in input, child of 1),
	// child-4 (fourth in input, child of 1), root-3 (third in input)
	expected := []string{"uuid-1", "uuid-2", "uuid-4", "uuid-3"}
	if len(firstOrder) != len(expected) {
		t.Fatalf("Expected %d tasks in order, got %d", len(expected), len(firstOrder))
	}
	for i, uid := range expected {
		if firstOrder[i] != uid {
			t.Errorf("Position %d: expected %s, got %s", i, uid, firstOrder[i])
		}
	}
}

func TestRoundTripPreservesTodoOrder(t *testing.T) {
	cal := ics.NewCalendar()

	// Create todos in a deliberate interleaved order: child before parent
	todo1 := ics.VTodo{}
	todo1.SetProperty(ics.ComponentPropertyUniqueId, "uuid-child-1")
	todo1.SetProperty(ics.ComponentPropertySummary, "Child 1")
	todo1.SetProperty("RELATED-TO", "uuid-parent")

	todo2 := ics.VTodo{}
	todo2.SetProperty(ics.ComponentPropertyUniqueId, "uuid-parent")
	todo2.SetProperty(ics.ComponentPropertySummary, "Parent")

	todo3 := ics.VTodo{}
	todo3.SetProperty(ics.ComponentPropertyUniqueId, "uuid-child-2")
	todo3.SetProperty(ics.ComponentPropertySummary, "Child 2")
	todo3.SetProperty("RELATED-TO", "uuid-parent")

	cal.Components = append(cal.Components, &todo1, &todo2, &todo3)

	// Round-trip: calendar -> tasks -> calendar
	tasks := getTasksForTodos(cal)
	outCal := getCalendarForTasks(tasks, nil, nil)
	outTodos := getTodos(outCal)

	if len(outTodos) != 3 {
		t.Fatalf("Expected 3 todos, got %d", len(outTodos))
	}

	expectedOrder := []string{"uuid-child-1", "uuid-parent", "uuid-child-2"}
	for i, expected := range expectedOrder {
		actual := getProperty(outTodos[i], ics.ComponentPropertyUniqueId)
		if actual != expected {
			t.Errorf("Position %d: expected UID %s, got %s", i, expected, actual)
		}
	}
}

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
	// All original properties are kept (UID, SUMMARY, DESCRIPTION, CREATED, PERCENT-COMPLETE,
	// X-KDE-ktimetracker-totalSessionTime, X-KDE-ktimetracker-totalTaskTime)
	if len(task.properties) != 7 {
		t.Fatalf("Expected 7 preserved properties, got %d", len(task.properties))
	}

	outCal := getCalendarForTasks(tasks, nil, nil)
	outTodos := getTodos(outCal)
	if len(outTodos) != 1 {
		t.Fatalf("Expected 1 todo, got %d", len(outTodos))
	}
	outTodo := outTodos[0]

	// Verify property order is preserved
	expectedOrder := []string{
		"UID", "SUMMARY", "DESCRIPTION", "CREATED", "PERCENT-COMPLETE",
		"X-KDE-ktimetracker-totalSessionTime", "X-KDE-ktimetracker-totalTaskTime",
	}
	if len(outTodo.Properties) != len(expectedOrder) {
		t.Fatalf("Expected %d properties, got %d", len(expectedOrder), len(outTodo.Properties))
	}
	for i, expectedToken := range expectedOrder {
		actualToken := outTodo.Properties[i].IANAToken
		if actualToken != expectedToken {
			t.Errorf("Property %d: expected %s, got %s", i, expectedToken, actualToken)
		}
	}

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

func TestRoundTripPreservesVEvents(t *testing.T) {
	cal := ics.NewCalendar()

	todo1 := ics.VTodo{}
	todo1.SetProperty(ics.ComponentPropertyUniqueId, "uuid-todo-1")
	todo1.SetProperty(ics.ComponentPropertySummary, "Task 1")

	event1 := ics.VEvent{}
	event1.SetProperty(ics.ComponentPropertyUniqueId, "uuid-event-1")
	event1.SetProperty(ics.ComponentPropertySummary, "Event 1")
	event1.SetProperty("RELATED-TO", "uuid-todo-1")

	todo2 := ics.VTodo{}
	todo2.SetProperty(ics.ComponentPropertyUniqueId, "uuid-todo-2")
	todo2.SetProperty(ics.ComponentPropertySummary, "Task 2")

	event2 := ics.VEvent{}
	event2.SetProperty(ics.ComponentPropertyUniqueId, "uuid-event-2")
	event2.SetProperty(ics.ComponentPropertySummary, "Event 2")
	event2.SetProperty("RELATED-TO", "uuid-todo-2")

	// Interleave: todo, event, todo, event
	cal.Components = append(cal.Components, &todo1, &event1, &todo2, &event2)

	tasks := getTasksForTodos(cal)
	events := makeEventsForVEvents(getEvents(cal))
	outCal := getCalendarForTasks(tasks, events, cal)

	if len(outCal.Components) != 4 {
		t.Fatalf("Expected 4 components, got %d", len(outCal.Components))
	}
	if _, ok := outCal.Components[0].(*ics.VTodo); !ok {
		t.Errorf("Expected component 0 to be VTODO")
	}
	if _, ok := outCal.Components[1].(*ics.VEvent); !ok {
		t.Errorf("Expected component 1 to be VEVENT")
	}
	if _, ok := outCal.Components[2].(*ics.VTodo); !ok {
		t.Errorf("Expected component 2 to be VTODO")
	}
	if _, ok := outCal.Components[3].(*ics.VEvent); !ok {
		t.Errorf("Expected component 3 to be VEVENT")
	}

	outTodos := getTodos(outCal)
	if len(outTodos) != 2 {
		t.Fatalf("Expected 2 todos, got %d", len(outTodos))
	}
	if getProperty(outTodos[0], ics.ComponentPropertyUniqueId) != "uuid-todo-1" {
		t.Errorf("Expected first todo UID uuid-todo-1, got %s", getProperty(outTodos[0], ics.ComponentPropertyUniqueId))
	}

	outEvents := getEvents(outCal)
	if len(outEvents) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(outEvents))
	}
	if getEventProperty(outEvents[0], ics.ComponentPropertyUniqueId) != "uuid-event-1" {
		t.Errorf("Expected first event UID uuid-event-1, got %s", getEventProperty(outEvents[0], ics.ComponentPropertyUniqueId))
	}
}

func TestRoundTripPreservesCalendarProperties(t *testing.T) {
	cal := ics.NewCalendar()
	cal.SetProductId("-//K Desktop Environment//NONSGML libkcal 4.3//EN")
	cal.CalendarProperties = append(cal.CalendarProperties, ics.CalendarProperty{
		BaseProperty: ics.BaseProperty{
			IANAToken: "X-KDE-ICAL-IMPLEMENTATION-VERSION",
			Value:     "1.0",
		},
	})

	todo := ics.VTodo{}
	todo.SetProperty(ics.ComponentPropertyUniqueId, "uuid-todo-1")
	todo.SetProperty(ics.ComponentPropertySummary, "Task 1")
	cal.Components = append(cal.Components, &todo)

	tasks := getTasksForTodos(cal)
	outCal := getCalendarForTasks(tasks, nil, cal)

	var prodId, xKdeVersion string
	for _, prop := range outCal.CalendarProperties {
		switch prop.IANAToken {
		case "PRODID":
			prodId = prop.Value
		case "X-KDE-ICAL-IMPLEMENTATION-VERSION":
			xKdeVersion = prop.Value
		}
	}
	if prodId != "-//K Desktop Environment//NONSGML libkcal 4.3//EN" {
		t.Errorf("Expected PRODID preserved, got: %s", prodId)
	}
	if xKdeVersion != "1.0" {
		t.Errorf("Expected X-KDE-ICAL-IMPLEMENTATION-VERSION preserved, got: %s", xKdeVersion)
	}
}

func TestRoundTripAppendsNewTasks(t *testing.T) {
	cal := ics.NewCalendar()

	todo1 := ics.VTodo{}
	todo1.SetProperty(ics.ComponentPropertyUniqueId, "uuid-todo-1")
	todo1.SetProperty(ics.ComponentPropertySummary, "Task 1")
	cal.Components = append(cal.Components, &todo1)

	tasks := getTasksForTodos(cal)
	// Add a brand-new task
	newTask := NewTask("New Task")
	newTask.position = 1
	tasks = append(tasks, newTask)

	outCal := getCalendarForTasks(tasks, nil, cal)

	if len(outCal.Components) != 2 {
		t.Fatalf("Expected 2 components, got %d", len(outCal.Components))
	}

	outTodos := getTodos(outCal)
	if len(outTodos) != 2 {
		t.Fatalf("Expected 2 todos, got %d", len(outTodos))
	}
	if getProperty(outTodos[0], ics.ComponentPropertyUniqueId) != "uuid-todo-1" {
		t.Errorf("Expected first todo to be original, got %s", getProperty(outTodos[0], ics.ComponentPropertyUniqueId))
	}
	if getProperty(outTodos[1], ics.ComponentPropertyUniqueId) != newTask.uuid {
		t.Errorf("Expected second todo to be new task, got %s", getProperty(outTodos[1], ics.ComponentPropertyUniqueId))
	}
}



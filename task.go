package main

import (
	"sort"

	ics "github.com/arran4/golang-ical"
	"github.com/google/uuid"
)

type Task struct {
	uuid        string
	name        string
	description string
	parent      *Task
	children    []*Task
	properties  []ics.IANAProperty
	position    int
}

func (task *Task) getUUID() string {
	return task.uuid
}

// Generic property helpers --------------------------------------------------

func (task *Task) getProperty(token string) string {
	for _, prop := range task.properties {
		if prop.IANAToken == token && prop.Value != "" {
			return prop.Value
		}
	}
	return ""
}

func (task *Task) setProperty(token, value string) {
	for i, prop := range task.properties {
		if prop.IANAToken == token {
			task.properties[i].Value = value
			return
		}
	}
	task.properties = append(task.properties, ics.IANAProperty{
		BaseProperty: ics.BaseProperty{IANAToken: token, Value: value},
	})
}

func (task *Task) removeProperty(token string) {
	var newProps []ics.IANAProperty
	for _, prop := range task.properties {
		if prop.IANAToken != token {
			newProps = append(newProps, prop)
		}
	}
	task.properties = newProps
}

func (task *Task) getPropertyParam(token, paramKey, paramValue string) string {
	for _, prop := range task.properties {
		if prop.IANAToken == token && prop.Value != "" {
			if keys, ok := prop.ICalParameters[paramKey]; ok && len(keys) > 0 && keys[0] == paramValue {
				return prop.Value
			}
		}
	}
	return ""
}

func (task *Task) setPropertyParam(token, paramKey, paramValue, value string) {
	for i, prop := range task.properties {
		if prop.IANAToken == token {
			if keys, ok := prop.ICalParameters[paramKey]; ok && len(keys) > 0 && keys[0] == paramValue {
				task.properties[i].Value = value
				return
			}
		}
	}
	task.properties = append(task.properties, ics.IANAProperty{
		BaseProperty: ics.BaseProperty{
			IANAToken:      token,
			ICalParameters: map[string][]string{paramKey: {paramValue}},
			Value:          value,
		},
	})
}

func (task *Task) removePropertyParam(token, paramKey, paramValue string) {
	var newProps []ics.IANAProperty
	for _, prop := range task.properties {
		if prop.IANAToken == token {
			if keys, ok := prop.ICalParameters[paramKey]; ok && len(keys) > 0 && keys[0] == paramValue {
				continue
			}
		}
		newProps = append(newProps, prop)
	}
	task.properties = newProps
}

// ---------------------------------------------------------
// convert between Task and ics.VTodo
// ---------------------------------------------------------

func makeTaskForTodo(todo *ics.VTodo) (Task, string) {
	task := Task{
		uuid:        getProperty(todo, ics.ComponentPropertyUniqueId),
		name:        getProperty(todo, ics.ComponentPropertySummary),
		description: getProperty(todo, ics.ComponentPropertyDescription),
	}
	relatedTo := getProperty(todo, "RELATED-TO")
	task.properties = append([]ics.IANAProperty(nil), todo.Properties...)
	return task, relatedTo
}

func makeTodoForTask(task *Task) ics.VTodo {
	todo := ics.VTodo{}

	emitted := make(emittedSet)

	for _, prop := range task.properties {
		switch prop.IANAToken {
		case string(ics.ComponentPropertyUniqueId):
			todo.SetProperty(ics.ComponentPropertyUniqueId, task.uuid)
			emitted.mark("UID")
		case string(ics.ComponentPropertySummary):
			todo.SetProperty(ics.ComponentPropertySummary, task.name)
			emitted.mark("SUMMARY")
		case string(ics.ComponentPropertyDescription):
			todo.SetProperty(ics.ComponentPropertyDescription, task.description)
			emitted.mark("DESCRIPTION")
		case "RELATED-TO":
			if task.parent != nil {
				todo.SetProperty("RELATED-TO", task.parent.uuid)
				emitted.mark("RELATED-TO")
			}
		default:
			todo.Properties = append(todo.Properties, prop)
		}
	}

	if !emitted.has("UID") {
		todo.SetProperty(ics.ComponentPropertyUniqueId, task.uuid)
	}
	if !emitted.has("SUMMARY") {
		todo.SetProperty(ics.ComponentPropertySummary, task.name)
	}
	if !emitted.has("DESCRIPTION") && task.description != "" {
		todo.SetProperty(ics.ComponentPropertyDescription, task.description)
	}
	if !emitted.has("RELATED-TO") && task.parent != nil {
		todo.SetProperty("RELATED-TO", task.parent.uuid)
	}

	return todo
}

// ---------------------------------------------------------
// Convert between []Task (tree) and []ics.VTodo (flat)
// ---------------------------------------------------------

func makeTasksForTodos(todos []*ics.VTodo) []*Task {
	uidMap := make(map[string]*Task)
	relatedToMap := make(map[string]string)
	taskOrder := make([]*Task, 0, len(todos))
	roots := []*Task{}
	for i, todo := range todos {
		task, relatedTo := makeTaskForTodo(todo)
		task.position = i
		uidMap[task.uuid] = &task
		relatedToMap[task.uuid] = relatedTo
		taskOrder = append(taskOrder, &task)
	}
	for _, task := range taskOrder {
		if parentUUID := relatedToMap[task.uuid]; parentUUID == "" {
			roots = append(roots, task)
		} else {
			parent := uidMap[parentUUID]
			task.parent = parent
			parent.children = append(parent.children, task)
		}
	}
	return roots
}

func flattenTasks(tasks []*Task) []*Task {
	var result []*Task
	for _, task := range tasks {
		result = append(result, task)
		result = append(result, flattenTasks(task.children)...)
	}
	return result
}

func makeTodosForTasks(tasks []*Task) []*ics.VTodo {
	allTasks := flattenTasks(tasks)
	sort.Slice(allTasks, func(i, j int) bool {
		return allTasks[i].position < allTasks[j].position
	})
	todos := make([]*ics.VTodo, len(allTasks))
	for i, task := range allTasks {
		todo := makeTodoForTask(task)
		todos[i] = &todo
	}
	return todos
}

func getTasksForTodos(cal *ics.Calendar) []*Task {
	return makeTasksForTodos(getTodos(cal))
}

func getCalendarForTasks(tasks []*Task, events []*Event, original *ics.Calendar) *ics.Calendar {
	// Build map of current tasks by UUID
	allTasks := flattenTasks(tasks)
	taskMap := make(map[string]*Task, len(allTasks))
	for _, task := range allTasks {
		taskMap[task.uuid] = task
	}

	// Build map of current events by UUID
	eventMap := make(map[string]*Event, len(events))
	for _, event := range events {
		eventMap[event.uuid] = event
	}

	if original == nil {
		cal := ics.NewCalendar()
		todos := makeTodosForTasks(tasks)
		for _, todo := range todos {
			cal.Components = append(cal.Components, todo)
		}
		for _, event := range events {
			ve := makeVEventForEvent(event)
			cal.Components = append(cal.Components, &ve)
		}
		return cal
	}

	cal := ics.NewCalendar()
	// Preserve original calendar-level properties
	cal.CalendarProperties = append([]ics.CalendarProperty(nil), original.CalendarProperties...)

	// Iterate original components in order, replacing VTODOs and VEVENTs, keeping everything else
	for _, comp := range original.Components {
		switch c := comp.(type) {
		case *ics.VTodo:
			uid := getProperty(c, ics.ComponentPropertyUniqueId)
			if task, ok := taskMap[uid]; ok {
				todo := makeTodoForTask(task)
				cal.Components = append(cal.Components, &todo)
				delete(taskMap, uid)
			}
			// If task no longer exists, drop the VTODO (deletion)
		case *ics.VEvent:
			uid := getEventProperty(c, ics.ComponentPropertyUniqueId)
			if event, ok := eventMap[uid]; ok {
				ve := makeVEventForEvent(event)
				cal.Components = append(cal.Components, &ve)
				delete(eventMap, uid)
			}
			// If event no longer exists in our list, drop the VEVENT (orphaned / deleted)
		default:
			cal.Components = append(cal.Components, comp)
		}
	}

	// Append any brand-new tasks at the end, in position order
	if len(taskMap) > 0 {
		remaining := make([]*Task, 0, len(taskMap))
		for _, task := range taskMap {
			remaining = append(remaining, task)
		}
		sort.Slice(remaining, func(i, j int) bool {
			return remaining[i].position < remaining[j].position
		})
		for _, task := range remaining {
			todo := makeTodoForTask(task)
			cal.Components = append(cal.Components, &todo)
		}
	}

	// Append any brand-new events at the end, in UUID order for determinism
	if len(eventMap) > 0 {
		remaining := make([]*Event, 0, len(eventMap))
		for _, event := range eventMap {
			remaining = append(remaining, event)
		}
		sort.Slice(remaining, func(i, j int) bool {
			return remaining[i].uuid < remaining[j].uuid
		})
		for _, event := range remaining {
			ve := makeVEventForEvent(event)
			cal.Components = append(cal.Components, &ve)
		}
	}

	return cal
}

func countSubtasks(task *Task) int {
	count := 1 // count the task itself
	for _, child := range task.children {
		count += countSubtasks(child)
	}
	return count
}

func collectTaskUUIDs(task *Task, set map[string]bool) {
	set[task.uuid] = true
	for _, child := range task.children {
		collectTaskUUIDs(child, set)
	}
}

func NewTask(name string) *Task {
	return &Task{
		uuid: uuid.New().String(),
		name: name,
	}
}

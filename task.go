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
	for _, prop := range todo.Properties {
		switch prop.IANAToken {
		case string(ics.ComponentPropertyUniqueId), string(ics.ComponentPropertySummary), string(ics.ComponentPropertyDescription), "RELATED-TO":
			continue
		}
		task.properties = append(task.properties, prop)
	}
	return task, relatedTo
}

func makeTodoForTask(task *Task) ics.VTodo {
	todo := ics.VTodo{}
	todo.SetProperty(ics.ComponentPropertyUniqueId, task.uuid)
	todo.SetProperty(ics.ComponentPropertySummary, task.name)
	todo.SetProperty(ics.ComponentPropertyDescription, task.description)
	if task.parent != nil {
		todo.SetProperty("RELATED-TO", task.parent.uuid)
	}
	todo.Properties = append(todo.Properties, task.properties...)
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

func getCalendarForTasks(tasks []*Task, original *ics.Calendar) *ics.Calendar {
	if original == nil {
		cal := ics.NewCalendar()
		todos := makeTodosForTasks(tasks)
		for _, todo := range todos {
			cal.Components = append(cal.Components, todo)
		}
		return cal
	}

	// Build map of current tasks by UUID
	allTasks := flattenTasks(tasks)
	taskMap := make(map[string]*Task, len(allTasks))
	for _, task := range allTasks {
		taskMap[task.uuid] = task
	}

	cal := ics.NewCalendar()
	// Preserve original calendar-level properties
	cal.CalendarProperties = append([]ics.CalendarProperty(nil), original.CalendarProperties...)

	// Iterate original components in order, replacing VTODOs and keeping everything else
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

	return cal
}

func NewTask(name string) *Task {
	return &Task{
		uuid: uuid.New().String(),
		name: name,
	}
}

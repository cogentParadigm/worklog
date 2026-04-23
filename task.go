package main

import (
	ics "github.com/arran4/golang-ical"
	"github.com/google/uuid"
)

type Task struct {
	uuid        string
	name        string
	description string
	relatedTo   string
	parent      *Task
	children    []*Task
}

// ---------------------------------------------------------
// convert between Task and ics.VTodo
// ---------------------------------------------------------

func makeTaskForTodo(todo *ics.VTodo) Task {
	return Task{
		getProperty(todo, ics.ComponentPropertyUniqueId),
		getProperty(todo, ics.ComponentPropertySummary),
		getProperty(todo, ics.ComponentPropertyDescription),
		getProperty(todo, "RELATED-TO"),
		nil,
		nil,
	}
}

func makeTodoForTask(task *Task) ics.VTodo {
	todo := ics.VTodo{}
	todo.SetProperty(ics.ComponentPropertyUniqueId, task.uuid)
	todo.SetProperty(ics.ComponentPropertySummary, task.name)
	todo.SetProperty(ics.ComponentPropertyDescription, task.description)
	if task.parent != nil {
		todo.SetProperty("RELATED-TO", task.parent.uuid)
	}
	return todo
}

// ---------------------------------------------------------
// Convert between []Task (tree) and []ics.VTodo (flat)
// ---------------------------------------------------------

func makeTasksForTodos(todos []*ics.VTodo) []*Task {
	uidMap := make(map[string]*Task)
	roots := []*Task{}
	for _, todo := range todos {
		task := makeTaskForTodo(todo)
		uidMap[task.uuid] = &task
	}
	for _, task := range uidMap {
		if task.relatedTo == "" {
			roots = append(roots, task)
		} else {
			parent := uidMap[task.relatedTo]
			task.parent = parent
			parent.children = append(parent.children, task)
		}
	}
	return roots
}

func makeTodosForTasks(tasks []*Task) (todos []*ics.VTodo) {
	for _, task := range tasks {
		todo := makeTodoForTask(task)
		todos = append(todos, &todo)
		if len(task.children) > 0 {
			children := makeTodosForTasks(task.children)
			todos = append(todos, children...)
		}
	}
	return todos
}

func getTasksForTodos(cal *ics.Calendar) []*Task {
	return makeTasksForTodos(getTodos(cal))
}

func getCalendarForTasks(tasks []*Task) *ics.Calendar {
	cal := ics.NewCalendar()
	todos := makeTodosForTasks(tasks)
	for _, todo := range todos {
		cal.Components = append(cal.Components, todo)
	}
	return cal
}

func NewTask(name string) *Task {
	return &Task{
		uuid.New().String(),
		name,
		"",
		"",
		nil,
		nil,
	}
}

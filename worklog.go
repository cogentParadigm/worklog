package main

import "strings"

type Worklog struct {
	path  string
	tasks []*Task
}

func NewWorklog(path string) *Worklog {
	cal := openCalendar(path)
	tasks := getTasksForTodos(cal)
	return &Worklog{path, tasks}
}

func (worklog *Worklog) NewTask(name string) *Task {
	task := NewTask(name)
	worklog.tasks = append(worklog.tasks, task)
	return task
}

func (worklog *Worklog) Save() {
	cal := getCalendarForTasks(worklog.tasks)
	saveCalendar(strings.Replace(worklog.path, ".ics", "-output.ics", 1), cal)
}

package main

import (
	"fmt"
	"strings"
)

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

func (worklog *Worklog) FindTaskByUUID(uuid string) *Task {
	return findTaskByUUID(worklog.tasks, uuid)
}

func findTaskByUUID(tasks []*Task, uuid string) *Task {
	for _, task := range tasks {
		if task.uuid == uuid {
			return task
		}
		if found := findTaskByUUID(task.children, uuid); found != nil {
			return found
		}
	}
	return nil
}

func (worklog *Worklog) UpdateTask(uuid string, name string, description string, parentUUID string) error {
	task := worklog.FindTaskByUUID(uuid)
	if task == nil {
		return fmt.Errorf("task with UUID '%s' not found", uuid)
	}

	if name != "" {
		task.name = name
	}
	if description != "" {
		task.description = description
	}
	if parentUUID != "" {
		// Find the new parent task
		newParent := worklog.FindTaskByUUID(parentUUID)
		if newParent == nil {
			return fmt.Errorf("parent task with UUID '%s' not found", parentUUID)
		}

		// Remove task from old parent's children if it had a parent
		if task.parent != nil {
			oldParent := task.parent
			for i, child := range oldParent.children {
				if child.uuid == task.uuid {
					oldParent.children = append(oldParent.children[:i], oldParent.children[i+1:]...)
					break
				}
			}
		} else {
			// Task was a root task, remove from worklog.tasks
			for i, t := range worklog.tasks {
				if t.uuid == task.uuid {
					worklog.tasks = append(worklog.tasks[:i], worklog.tasks[i+1:]...)
					break
				}
			}
		}

		// Add task to new parent's children
		task.parent = newParent
		newParent.children = append(newParent.children, task)
		task.relatedTo = parentUUID
	}

	return nil
}

func (worklog *Worklog) Save() {
	cal := getCalendarForTasks(worklog.tasks)
	saveCalendar(strings.Replace(worklog.path, ".ics", "-output.ics", 1), cal)
}

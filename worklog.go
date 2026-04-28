package main

import (
	"fmt"
	"strings"

	ics "github.com/arran4/golang-ical"
)

type Worklog struct {
	path         string
	tasks        []*Task
	nextPosition int
	calendar     *ics.Calendar // original calendar for round-trip preservation
}

func NewWorklog(path string) (*Worklog, error) {
	cal, err := openCalendar(path)
	if err != nil {
		return nil, fmt.Errorf("open worklog: %w", err)
	}
	todos := getTodos(cal)
	tasks := makeTasksForTodos(todos)
	return &Worklog{path: path, tasks: tasks, nextPosition: len(todos), calendar: cal}, nil
}

func (worklog *Worklog) NewTask(name string) *Task {
	task := NewTask(name)
	task.position = worklog.nextPosition
	worklog.nextPosition++
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

// isDescendantOf checks if 'potentialDescendant' is a descendant of 'ancestor' in the task tree.
// This is used for cycle detection when moving tasks.
func isDescendantOf(ancestor, potentialDescendant *Task) bool {
	if ancestor == nil || potentialDescendant == nil {
		return false
	}
	for _, child := range ancestor.children {
		if child.uuid == potentialDescendant.uuid {
			return true
		}
		if isDescendantOf(child, potentialDescendant) {
			return true
		}
	}
	return false
}

// removeFromParent removes a task from its current parent's children list
// or from the root tasks list if it has no parent.
// Note: This does not clear task.parent - the caller is responsible for setting the new parent.
func (worklog *Worklog) removeFromParent(task *Task) {
	if task.parent != nil {
		oldParent := task.parent
		for i, child := range oldParent.children {
			if child.uuid == task.uuid {
				oldParent.children = append(oldParent.children[:i], oldParent.children[i+1:]...)
				break
			}
		}
		task.parent = nil
	} else {
		// Task was a root task, remove from worklog.tasks
		for i, t := range worklog.tasks {
			if t.uuid == task.uuid {
				worklog.tasks = append(worklog.tasks[:i], worklog.tasks[i+1:]...)
				break
			}
		}
	}
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
		// Check for self-parenting (immediate cycle)
		if parentUUID == uuid {
			return fmt.Errorf("cannot set task as its own parent (cycle detected)")
		}

		// Find the new parent task
		newParent := worklog.FindTaskByUUID(parentUUID)
		if newParent == nil {
			return fmt.Errorf("parent task with UUID '%s' not found", parentUUID)
		}

		// Check for deeper cycle: newParent must not be a descendant of task
		if isDescendantOf(task, newParent) {
			return fmt.Errorf("cannot move task to a descendant (cycle detected)")
		}

		// Remove task from its current parent (root or existing parent)
		worklog.removeFromParent(task)

		// Add task to new parent's children
		task.parent = newParent
		newParent.children = append(newParent.children, task)
		task.relatedTo = parentUUID
	}

	return nil
}

func (worklog *Worklog) GetEvents() []*ics.VEvent {
	if worklog.calendar == nil {
		return nil
	}
	return getEvents(worklog.calendar)
}

func (worklog *Worklog) Save() error {
	cal := getCalendarForTasks(worklog.tasks, worklog.calendar)
	outPath := strings.Replace(worklog.path, ".ics", "-output.ics", 1)
	if err := saveCalendar(outPath, cal); err != nil {
		return fmt.Errorf("save worklog: %w", err)
	}
	return nil
}

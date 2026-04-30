package main

import (
	"fmt"
	"strings"

	ics "github.com/arran4/golang-ical"
)

type Worklog struct {
	path         string
	tasks        []*Task
	events       []*Event
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
	vevents := getEvents(cal)
	events := makeEventsForVEvents(vevents)
	return &Worklog{path: path, tasks: tasks, events: events, nextPosition: len(todos), calendar: cal}, nil
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

type TaskUpdate struct {
	Name        *string
	Description *string
	ParentUUID  *string
}

func (worklog *Worklog) UpdateTask(uuid string, update TaskUpdate) error {
	task := worklog.FindTaskByUUID(uuid)
	if task == nil {
		return fmt.Errorf("task with UUID '%s' not found", uuid)
	}

	if update.Name != nil {
		if *update.Name == "" {
			return fmt.Errorf("cannot clear task name")
		}
		task.name = *update.Name
	}
	if update.Description != nil {
		task.description = *update.Description
	}
	if update.ParentUUID != nil {
		if *update.ParentUUID == "" {
			// Move to root
			worklog.removeFromParent(task)
			worklog.tasks = append(worklog.tasks, task)
			return nil
		}

		// Check for self-parenting (immediate cycle)
		if *update.ParentUUID == uuid {
			return fmt.Errorf("cannot set task as its own parent (cycle detected)")
		}

		// Find the new parent task
		newParent := worklog.FindTaskByUUID(*update.ParentUUID)
		if newParent == nil {
			return fmt.Errorf("parent task with UUID '%s' not found", *update.ParentUUID)
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
	}

	return nil
}

func (worklog *Worklog) DeleteTask(uuid string) (int, error) {
	task := worklog.FindTaskByUUID(uuid)
	if task == nil {
		return 0, fmt.Errorf("task with UUID '%s' not found", uuid)
	}

	count := countSubtasks(task)
	worklog.removeFromParent(task)

	// Collect all deleted task UUIDs
	deletedUUIDs := make(map[string]bool)
	collectTaskUUIDs(task, deletedUUIDs)

	// Filter out events linked to deleted tasks
	var remaining []*Event
	for _, event := range worklog.events {
		if !deletedUUIDs[event.relatedTo] {
			remaining = append(remaining, event)
		}
	}
	worklog.events = remaining

	return count, nil
}

func (worklog *Worklog) GetEvents() []*Event {
	return worklog.events
}

func (worklog *Worklog) Save() error {
	cal := getCalendarForTasks(worklog.tasks, worklog.events, worklog.calendar)
	outPath := strings.Replace(worklog.path, ".ics", "-output.ics", 1)
	if err := saveCalendar(outPath, cal); err != nil {
		return fmt.Errorf("save worklog: %w", err)
	}
	return nil
}

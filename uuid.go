package main

import (
	"fmt"
	"strings"
)

const minShortUUIDLen = 4

// shortUUID returns the shortest unique prefix of uuid against all other UUIDs in allUUIDs.
// The prefix is always at least minShortUUIDLen characters.
func shortUUID(uuid string, allUUIDs []string) string {
	if len(uuid) <= minShortUUIDLen {
		return uuid
	}
	// Build a list of other UUIDs, skipping exactly one occurrence of uuid.
	others := make([]string, 0, len(allUUIDs))
	skipped := false
	for _, u := range allUUIDs {
		if u == uuid && !skipped {
			skipped = true
			continue
		}
		others = append(others, u)
	}
	for l := minShortUUIDLen; l <= len(uuid); l++ {
		prefix := uuid[:l]
		unique := true
		for _, other := range others {
			if len(other) >= l && other[:l] == prefix {
				unique = false
				break
			}
		}
		if unique {
			return prefix
		}
	}
	return uuid
}

// shortUUIDs computes the shortest unique prefix for every UUID in the given slice.
func shortUUIDs(uuids []string) map[string]string {
	result := make(map[string]string, len(uuids))
	for _, uuid := range uuids {
		result[uuid] = shortUUID(uuid, uuids)
	}
	return result
}

// resolveTaskUUID finds a task by full or short UUID prefix.
// It tries exact match first, then prefix match against all tasks.
func resolveTaskUUID(worklog *Worklog, prefix string) (*Task, error) {
	if prefix == "" {
		return nil, fmt.Errorf("task UUID is empty")
	}

	// Exact match
	if task := worklog.FindTaskByUUID(prefix); task != nil {
		return task, nil
	}

	// Collect all UUIDs for short formatting and prefix matching
	allTasks := flattenTasks(worklog.tasks)
	var matches []*Task
	for _, task := range allTasks {
		if strings.HasPrefix(task.uuid, prefix) {
			matches = append(matches, task)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("task with UUID '%s' not found", prefix)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}

	// Ambiguous: list candidates
	allUUIDs := make([]string, len(allTasks))
	for i, task := range allTasks {
		allUUIDs[i] = task.uuid
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("UUID prefix '%s' is ambiguous; matches %d tasks:\n", prefix, len(matches)))
	for _, task := range matches {
		sb.WriteString(fmt.Sprintf("  %s - %s\n", shortUUID(task.uuid, allUUIDs), task.name))
	}
	return nil, fmt.Errorf(sb.String())
}

// resolveEventUUID finds an event by full or short UUID prefix.
// It tries exact match first, then prefix match against all events.
func resolveEventUUID(worklog *Worklog, prefix string) (*Event, error) {
	if prefix == "" {
		return nil, fmt.Errorf("time entry UUID is empty")
	}

	// Exact match
	if event := worklog.FindEventByUUID(prefix); event != nil {
		return event, nil
	}

	var matches []*Event
	for _, event := range worklog.events {
		if strings.HasPrefix(event.uuid, prefix) {
			matches = append(matches, event)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("time entry with UUID '%s' not found", prefix)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}

	// Ambiguous: list candidates
	allUUIDs := make([]string, len(worklog.events))
	for i, event := range worklog.events {
		allUUIDs[i] = event.uuid
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("UUID prefix '%s' is ambiguous; matches %d time entries:\n", prefix, len(matches)))
	for _, event := range matches {
		taskName := "(orphaned task)"
		if task := worklog.FindTaskByUUID(event.relatedTo); task != nil {
			taskName = task.name
		}
		sb.WriteString(fmt.Sprintf("  %s - %s\n", shortUUID(event.uuid, allUUIDs), taskName))
	}
	return nil, fmt.Errorf(sb.String())
}

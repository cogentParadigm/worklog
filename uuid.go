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

type uuided interface {
	getUUID() string
}

func resolveByPrefix[T uuided](prefix string, exact func(string) (T, bool), all []T, itemName string, itemNamePlural string, describe func(T) string) (T, error) {
	var zero T
	if prefix == "" {
		return zero, fmt.Errorf("%s UUID is empty", itemName)
	}

	if item, ok := exact(prefix); ok {
		return item, nil
	}

	allUUIDs := make([]string, len(all))
	for i, item := range all {
		allUUIDs[i] = item.getUUID()
	}

	var matches []T
	for _, item := range all {
		if strings.HasPrefix(item.getUUID(), prefix) {
			matches = append(matches, item)
		}
	}

	if len(matches) == 0 {
		return zero, fmt.Errorf("%s with UUID '%s' not found", itemName, prefix)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("UUID prefix '%s' is ambiguous; matches %d %s:\n", prefix, len(matches), itemNamePlural))
	for _, item := range matches {
		sb.WriteString(fmt.Sprintf("  %s - %s\n", shortUUID(item.getUUID(), allUUIDs), describe(item)))
	}
	return zero, fmt.Errorf(sb.String())
}

// resolveTaskUUID finds a task by full or short UUID prefix.
// It tries exact match first, then prefix match against all tasks.
func resolveTaskUUID(worklog *Worklog, prefix string) (*Task, error) {
	allTasks := flattenTasks(worklog.tasks)
	return resolveByPrefix(prefix, func(p string) (*Task, bool) {
		t := worklog.FindTaskByUUID(p)
		return t, t != nil
	}, allTasks, "task", "tasks", func(t *Task) string {
		return t.name
	})
}

// resolveEventUUID finds an event by full or short UUID prefix.
// It tries exact match first, then prefix match against all events.
func resolveEventUUID(worklog *Worklog, prefix string) (*Event, error) {
	return resolveByPrefix(prefix, func(p string) (*Event, bool) {
		e := worklog.FindEventByUUID(p)
		return e, e != nil
	}, worklog.events, "time entry", "time entries", func(e *Event) string {
		if task := worklog.FindTaskByUUID(e.relatedTo); task != nil {
			return task.name
		}
		return "(orphaned task)"
	})
}

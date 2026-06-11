package main

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/cogentParadigm/worklog/internal/jira"
	"github.com/cogentParadigm/worklog/internal/tempo"
)

func readLineTrim(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func interactiveSelectTask(tasks []*Task, shortUUIDs map[string]string, reader *bufio.Reader, out io.Writer) (*Task, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks to select")
	}

	for {
		fmt.Fprintln(out, "Select a task to edit:")
		for i, task := range tasks {
			fmt.Fprintf(out, "  %d) %s  %s\n", i+1, shortUUIDs[task.uuid], task.name)
		}
		fmt.Fprintln(out, "  q) Quit")
		fmt.Fprint(out, "Select: ")

		line, err := readLineTrim(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to read selection: %w", err)
		}

		if line == "q" || line == "Q" {
			return nil, nil
		}

		idx, err := strconv.Atoi(line)
		if err != nil || idx < 1 || idx > len(tasks) {
			fmt.Fprintln(out, "Invalid selection.")
			continue
		}

		return tasks[idx-1], nil
	}
}

func interactiveEditTask(task *Task, worklog *Worklog, jiraClient *jira.Client, tempoClient *tempo.Client, cfg *Config, shortUUIDs map[string]string, reader *bufio.Reader, out io.Writer) (bool, error) {
	changed := false

	for {
		fmt.Fprintf(out, "\nEditing task %q [%s]\n", task.name, shortUUIDs[task.uuid])
		fmt.Fprintf(out, "  1) Name: %s\n", task.name)

		desc := task.description
		if desc == "" {
			desc = "(empty)"
		}
		fmt.Fprintf(out, "  2) Description: %s\n", desc)

		issueKey := task.IssueKey()
		if issueKey == "" {
			issueKey = "(empty)"
		}
		fmt.Fprintf(out, "  3) Issue key: %s\n", issueKey)

		attrs := task.TempoAttributes()
		if len(attrs) == 0 {
			fmt.Fprintln(out, "  4) Tempo attributes: (none)")
		} else {
			var parts []string
			var keys []string
			for k := range attrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				parts = append(parts, fmt.Sprintf("%s=%s", k, attrs[k]))
			}
			fmt.Fprintf(out, "  4) Tempo attributes: %s\n", strings.Join(parts, ", "))
		}

		currentParent := ""
		if task.parent != nil {
			currentParent = shortUUIDs[task.parent.uuid]
		}
		if currentParent == "" {
			fmt.Fprintln(out, "  5) Parent: (root)")
		} else {
			fmt.Fprintf(out, "  5) Parent: %s\n", currentParent)
		}

		fmt.Fprintln(out, "  6) Done")
		fmt.Fprint(out, "Select field: ")

		line, err := readLineTrim(reader)
		if err != nil {
			return changed, fmt.Errorf("failed to read selection: %w", err)
		}

		switch line {
		case "1":
			fmt.Fprintf(out, "New name [%s]: ", task.name)
			newName, err := readLineTrim(reader)
			if err != nil {
				return changed, fmt.Errorf("failed to read name: %w", err)
			}
			if newName != "" && newName != task.name {
				task.name = newName
				changed = true
			}
		case "2":
			currentDesc := task.description
			if currentDesc == "" {
				fmt.Fprint(out, "New description: ")
			} else {
				fmt.Fprintf(out, "New description [%s]: ", currentDesc)
			}
			newDesc, err := readLineTrim(reader)
			if err != nil {
				return changed, fmt.Errorf("failed to read description: %w", err)
			}
			if newDesc != currentDesc {
				task.description = newDesc
				changed = true
			}
		case "3":
			c, err := interactiveEditIssueKey(task, jiraClient, shortUUIDs, reader, out)
			if err != nil {
				return changed, err
			}
			if c {
				changed = true
			}
		case "4":
			c, err := interactiveEditTempoAttributes(task, tempoClient, cfg, reader, out)
			if err != nil {
				return changed, err
			}
			if c {
				changed = true
			}
		case "5":
			if currentParent == "" {
				fmt.Fprint(out, "New parent UUID (empty = keep root): ")
			} else {
				fmt.Fprintf(out, "New parent UUID [%s] (empty = keep): ", currentParent)
			}
			newParentUUID, err := readLineTrim(reader)
			if err != nil {
				return changed, fmt.Errorf("failed to read parent: %w", err)
			}
			if newParentUUID == "" {
				// Keep current parent
				continue
			}
			parentTask, err := resolveTaskUUID(worklog, newParentUUID)
			if err != nil {
				fmt.Fprintf(out, "Invalid parent: %v\n", err)
				continue
			}
			if task.uuid == parentTask.uuid {
				fmt.Fprintln(out, "A task cannot be its own parent.")
				continue
			}
			if isDescendantOf(task, parentTask) {
				fmt.Fprintln(out, "Cannot move a task under one of its descendants.")
				continue
			}
			worklog.removeFromParent(task)
			task.parent = parentTask
			parentTask.children = append(parentTask.children, task)
			changed = true
		case "6", "q", "Q", "":
			return changed, nil
		default:
			fmt.Fprintln(out, "Invalid selection.")
		}
	}
}

func interactiveEditIssueKey(task *Task, jiraClient *jira.Client, shortUUIDs map[string]string, reader *bufio.Reader, out io.Writer) (bool, error) {
	currentKey := task.IssueKey()

	if currentKey == "" {
		fmt.Fprint(out, "New issue key (empty = clear, '?' = search Jira): ")
	} else {
		fmt.Fprintf(out, "New issue key [%s] (empty = clear, '?' = search Jira): ", currentKey)
	}

	line, err := readLineTrim(reader)
	if err != nil {
		return false, fmt.Errorf("failed to read issue key: %w", err)
	}

	if line == "" {
		if currentKey != "" {
			task.ClearIssueKey()
			return true, nil
		}
		return false, nil
	}

	if line != "?" {
		if line != currentKey {
			task.SetIssueKey(line)
			return true, nil
		}
		return false, nil
	}

	// Search mode
	if jiraClient == nil {
		fmt.Fprintln(out, "Jira client not configured.")
		return false, nil
	}

	fmt.Fprintf(out, "Search query [%s]: ", task.name)
	query, err := readLineTrim(reader)
	if err != nil {
		return false, fmt.Errorf("failed to read query: %w", err)
	}
	if query == "" {
		query = task.name
	}

	jql := buildSearchJQL(query)
	results, err := jiraClient.SearchIssues(jql)
	if err != nil {
		return false, fmt.Errorf("search for %q: %w", query, err)
	}

	if len(results) == 0 {
		fmt.Fprintln(out, "No issues found.")
		return false, nil
	}

	for i, r := range results {
		fmt.Fprintf(out, "  %d) %s — %s\n", i+1, r.Key, r.Summary)
	}
	fmt.Fprintln(out, "  s) Skip")
	fmt.Fprintln(out, "  q) Quit")

	for {
		fmt.Fprint(out, "Select: ")
		sel, err := readLineTrim(reader)
		if err != nil {
			return false, fmt.Errorf("failed to read selection: %w", err)
		}

		if sel == "q" || sel == "Q" || sel == "s" || sel == "S" {
			return false, nil
		}

		idx, err := strconv.Atoi(sel)
		if err != nil || idx < 1 || idx > len(results) {
			fmt.Fprintln(out, "Invalid selection.")
			continue
		}

		result := results[idx-1]
		task.SetIssueKey(result.Key)
		fmt.Fprintf(out, "Issue key %s assigned.\n", result.Key)
		return true, nil
	}
}

func interactiveEditTempoAttributes(task *Task, tempoClient *tempo.Client, cfg *Config, reader *bufio.Reader, out io.Writer) (bool, error) {
	changed := false

	for {
		attrs := task.TempoAttributes()
		if len(attrs) == 0 {
			fmt.Fprintln(out, "No tempo attributes set.")
		} else {
			fmt.Fprintln(out, "Current attributes:")
			var keys []string
			for k := range attrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(out, "  %s = %s\n", k, attrs[k])
			}
		}

		fmt.Fprintln(out, "  a) Add/edit attribute")
		fmt.Fprintln(out, "  c) Clear all")
		fmt.Fprintln(out, "  d) Done")
		fmt.Fprint(out, "Select: ")

		line, err := readLineTrim(reader)
		if err != nil {
			return changed, fmt.Errorf("failed to read selection: %w", err)
		}

		switch strings.ToLower(line) {
		case "a":
			c, err := interactiveAddEditTempoAttribute(task, tempoClient, reader, out)
			if err != nil {
				return changed, err
			}
			if c {
				changed = true
			}
		case "c":
			for k := range attrs {
				task.ClearTempoAttribute(k)
			}
			changed = true
		case "d", "q", "":
			return changed, nil
		default:
			fmt.Fprintln(out, "Invalid selection.")
		}
	}
}

func interactiveAddEditTempoAttribute(task *Task, tempoClient *tempo.Client, reader *bufio.Reader, out io.Writer) (bool, error) {
	attrs := task.TempoAttributes()
	var availableAttrs []tempo.WorkAttribute
	var fetchErr error

	if tempoClient != nil {
		availableAttrs, fetchErr = tempoClient.GetWorkAttributes()
	}

	var attrKey string

	if len(availableAttrs) > 0 {
		fmt.Fprintln(out, "Available attributes:")
		for i, a := range availableAttrs {
			req := ""
			if a.Required {
				req = " [required]"
			}
			fmt.Fprintf(out, "  %d) %s (%s)%s\n", i+1, a.Name, a.Key, req)
		}
		fmt.Fprintln(out, "  m) Manual entry")
		fmt.Fprint(out, "Select: ")
		sel, err := readLineTrim(reader)
		if err != nil {
			return false, fmt.Errorf("failed to read selection: %w", err)
		}
		if sel == "m" || sel == "M" {
			fmt.Fprint(out, "Key: ")
			key, err := readLineTrim(reader)
			if err != nil {
				return false, fmt.Errorf("failed to read key: %w", err)
			}
			attrKey = key
		} else {
			idx, err := strconv.Atoi(sel)
			if err != nil || idx < 1 || idx > len(availableAttrs) {
				fmt.Fprintln(out, "Invalid selection.")
				return false, nil
			}
			attrKey = availableAttrs[idx-1].Key
		}
	} else {
		if fetchErr != nil && verbose {
			fmt.Fprintf(out, "Could not fetch attributes: %v\n", fetchErr)
		}
		fmt.Fprint(out, "Key: ")
		key, err := readLineTrim(reader)
		if err != nil {
			return false, fmt.Errorf("failed to read key: %w", err)
		}
		attrKey = key
	}

	if attrKey == "" {
		fmt.Fprintln(out, "Key cannot be empty.")
		return false, nil
	}

	currentVal := attrs[attrKey]

	// Try to find the attribute definition for known values
	var attrDef *tempo.WorkAttribute
	for i := range availableAttrs {
		if availableAttrs[i].Key == attrKey {
			attrDef = &availableAttrs[i]
			break
		}
	}

	if attrDef != nil && len(attrDef.Values) > 0 {
		fmt.Fprintln(out, "Values:")
		for i, v := range attrDef.Values {
			label := v
			if attrDef.Names != nil {
				if name, ok := attrDef.Names[v]; ok && name != "" {
					label = name
				}
			}
			if label != v {
				fmt.Fprintf(out, "  %d) %s (%s)\n", i+1, label, v)
			} else {
				fmt.Fprintf(out, "  %d) %s\n", i+1, v)
			}
		}
		fmt.Fprintln(out, "  c) Clear")
		fmt.Fprintln(out, "  s) Skip")
		fmt.Fprint(out, "Select: ")
		sel, err := readLineTrim(reader)
		if err != nil {
			return false, fmt.Errorf("failed to read selection: %w", err)
		}
		sel = strings.ToLower(sel)
		if sel == "s" {
			return false, nil
		}
		if sel == "c" {
			if _, ok := attrs[attrKey]; ok {
				task.ClearTempoAttribute(attrKey)
				return true, nil
			}
			return false, nil
		}
		idx, err := strconv.Atoi(sel)
		if err != nil || idx < 1 || idx > len(attrDef.Values) {
			fmt.Fprintln(out, "Invalid selection.")
			return false, nil
		}
		val := attrDef.Values[idx-1]
		task.SetTempoAttribute(attrKey, val)
		fmt.Fprintf(out, "%s = %s\n", attrKey, val)
		return true, nil
	}

	// Free-form value
	if currentVal != "" {
		fmt.Fprintf(out, "Value [%s]: ", currentVal)
	} else {
		fmt.Fprint(out, "Value: ")
	}
	val, err := readLineTrim(reader)
	if err != nil {
		return false, fmt.Errorf("failed to read value: %w", err)
	}
	if val == "" && currentVal == "" {
		return false, nil
	}
	if val == "" {
		task.ClearTempoAttribute(attrKey)
		return true, nil
	}
	if val != currentVal {
		task.SetTempoAttribute(attrKey, val)
		return true, nil
	}
	return false, nil
}

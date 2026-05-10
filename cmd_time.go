package main

import (
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"
)

func runTime(args []string) error {
	if len(args) < 1 {
		printTimeUsage()
		return fmt.Errorf("no time subcommand provided")
	}
	if isHelpFlag(args[0]) {
		printTimeUsage()
		return flag.ErrHelp
	}

	commands := map[string]func([]string) error{
		"add":    runTimeAdd,
		"list":   runTimeList,
		"edit":   runTimeEdit,
		"delete": runTimeDelete,
	}

	handler, ok := commands[args[0]]
	if !ok {
		return fmt.Errorf("unknown time subcommand '%s'", args[0])
	}
	return handler(args[1:])
}

func runTimeAdd(args []string) error {
	timeAddCommand := flag.NewFlagSet("time add", flag.ContinueOnError)
	timeAddFile := timeAddCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	timeAddOutput := timeAddCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
	timeAddTask := timeAddCommand.String("task", "", "UUID (or short unique prefix) of the task to log time against (required)")
	timeAddDuration := timeAddCommand.String("duration", "", "Duration to log (e.g., 30m, 1h30m, 3600s) (required)")
	timeAddStart := timeAddCommand.String("start", "", "Start time (optional, defaults to now-duration)")
	timeAddComment := timeAddCommand.String("comment", "", "Comment for the time entry (optional)")
	configureFlagSet(timeAddCommand, "Add a manual time entry for a task. If -start is omitted, the start time is computed as now - duration.", "  worklog time add -task <short-uuid> -duration 30m\n  worklog time add -task <short-uuid> -duration 1h -start \"2023-08-14 09:00:00\"\n  worklog time add -task <short-uuid> -duration 3600s -comment \"Reviewed with team\"")
	if err := timeAddCommand.Parse(args); err != nil {
		return err
	}

	worklog, err := loadWorklog(*timeAddFile)
	if err != nil {
		return err
	}

	if *timeAddTask == "" {
		return fmt.Errorf("-task flag is required for time add")
	}
	if *timeAddDuration == "" {
		return fmt.Errorf("-duration flag is required for time add")
	}

	task, err := resolveTaskUUID(worklog, *timeAddTask)
	if err != nil {
		return err
	}

	duration, err := parseDurationFlag(*timeAddDuration)
	if err != nil {
		return err
	}

	var start time.Time
	if *timeAddStart != "" {
		start, err = parseTimeFlag(*timeAddStart)
		if err != nil {
			return err
		}
	}

	var end time.Time
	if !start.IsZero() {
		end = start.Add(time.Duration(duration) * time.Second)
	} else {
		end = time.Now()
		start = end.Add(-time.Duration(duration) * time.Second)
	}

	event := NewEvent(task.uuid, start, end, duration, task.name, *timeAddComment)
	worklog.AddEvent(event)
	if err := worklog.Save(*timeAddOutput); err != nil {
		return err
	}
	fmt.Printf("Added %s time entry for '%s'.\n", time.Duration(duration*int(time.Second)).String(), task.name)
	return nil
}

func runTimeList(args []string) error {
	timeListCommand := flag.NewFlagSet("time list", flag.ContinueOnError)
	timeListFile := timeListCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	timeListTask := timeListCommand.String("task", "", "Filter to a specific task UUID (or short unique prefix) (optional)")
	timeListFrom := timeListCommand.String("from", "", "Filter events starting on or after this date (YYYY-MM-DD)")
	timeListTo := timeListCommand.String("to", "", "Filter events starting on or before this date (YYYY-MM-DD)")
	timeListSearch := timeListCommand.String("search", "", "Filter by case-insensitive search in task name, description, or comment")
	configureFlagSet(timeListCommand, "List time entries sorted by start time (most recent first).", "  worklog time list\n  worklog time list -task <short-uuid>\n  worklog time list -from 2023-08-01 -to 2023-08-15\n  worklog time list -search meeting")
	if err := timeListCommand.Parse(args); err != nil {
		return err
	}

	worklog, err := loadWorklog(*timeListFile)
	if err != nil {
		return err
	}

	var fromDay, toDay time.Time
	if *timeListFrom != "" {
		fromDay, err = parseDateFlag(*timeListFrom)
		if err != nil {
			return err
		}
	}
	if *timeListTo != "" {
		toDay, err = parseDateFlag(*timeListTo)
		if err != nil {
			return err
		}
	}
	if !fromDay.IsZero() && !toDay.IsZero() && fromDay.After(toDay) {
		return fmt.Errorf("from date must not be after to date")
	}

	events := worklog.GetEvents()
	if *timeListTask != "" {
		task, err := resolveTaskUUID(worklog, *timeListTask)
		if err != nil {
			return err
		}
		var filtered []*Event
		for _, event := range events {
			if event.relatedTo == task.uuid {
				filtered = append(filtered, event)
			}
		}
		events = filtered
	}

	var filtered []*Event
	for _, event := range events {
		// Date range filter
		if !event.dtstart.IsZero() {
			eventDay := time.Date(event.dtstart.Year(), event.dtstart.Month(), event.dtstart.Day(), 0, 0, 0, 0, event.dtstart.Location())
			if !fromDay.IsZero() && eventDay.Before(fromDay) {
				continue
			}
			if !toDay.IsZero() && eventDay.After(toDay) {
				continue
			}
		} else if !fromDay.IsZero() || !toDay.IsZero() {
			continue
		}

		// Search filter
		if *timeListSearch != "" {
			match := false
			if task := worklog.FindTaskByUUID(event.relatedTo); task != nil {
				if matchesSearch(task.name, *timeListSearch) || matchesSearch(task.description, *timeListSearch) {
					match = true
				}
			}
			if !match && matchesSearch(event.comment, *timeListSearch) {
				match = true
			}
			if !match {
				continue
			}
		}

		filtered = append(filtered, event)
	}
	events = filtered

	sort.Slice(events, func(i, j int) bool {
		return events[i].dtstart.After(events[j].dtstart)
	})

	allEventUUIDs := worklog.allEventUUIDs()
	shortEventUUIDs := shortUUIDs(allEventUUIDs)
	maxShortLen := 4
	for _, su := range shortEventUUIDs {
		if len(su) > maxShortLen {
			maxShortLen = len(su)
		}
	}

	maxCommentLen := 7 // "Comment"
	for _, event := range events {
		truncated := truncate(event.comment, 30)
		if len(truncated) > maxCommentLen {
			maxCommentLen = len(truncated)
		}
	}

	fmt.Printf("%-*s %-30s %-20s %-20s %-10s %-*s\n", maxShortLen, "UUID", "Task", "Start", "End", "Duration", maxCommentLen, "Comment")
	for _, event := range events {
		taskName := "(orphaned task)"
		if task := worklog.FindTaskByUUID(event.relatedTo); task != nil {
			taskName = task.name
		}
		startStr := "-"
		if !event.dtstart.IsZero() {
			startStr = event.dtstart.Format("2006-01-02 15:04:05")
		}
		endStr := "-"
		if !event.dtend.IsZero() {
			endStr = event.dtend.Format("2006-01-02 15:04:05")
		}
		durStr := time.Duration(event.duration * int(time.Second)).String()
		comment := truncate(event.comment, 30)
		fmt.Printf("%-*s %-30s %-20s %-20s %-10s %-*s\n", maxShortLen, shortEventUUIDs[event.uuid], taskName, startStr, endStr, durStr, maxCommentLen, comment)
	}
	return nil
}

func runTimeEdit(args []string) error {
	timeEditCommand := flag.NewFlagSet("time edit", flag.ContinueOnError)
	timeEditFile := timeEditCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	timeEditOutput := timeEditCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
	timeEditUUID := timeEditCommand.String("uuid", "", "UUID (or short unique prefix) of the time entry to edit (required)")
	timeEditStart := timeEditCommand.String("start", "", "New start time")
	timeEditEnd := timeEditCommand.String("end", "", "New end time")
	timeEditDuration := timeEditCommand.String("duration", "", "New duration (e.g., 30m, 1h30m)")
	timeEditComment := timeEditCommand.String("comment", "", "New comment")
	configureFlagSet(timeEditCommand, "Edit an existing time entry. Only provided fields are changed. Duration is automatically recomputed when start or end is modified.", "  worklog time edit -uuid <short-uuid> -comment \"Updated\"\n  worklog time edit -uuid <short-uuid> -start \"2023-08-14 10:00:00\" -end \"2023-08-14 11:30:00\"")
	if err := timeEditCommand.Parse(args); err != nil {
		return err
	}

	worklog, err := loadWorklog(*timeEditFile)
	if err != nil {
		return err
	}

	if *timeEditUUID == "" {
		return fmt.Errorf("-uuid flag is required for time edit")
	}

	update := EventUpdate{}
	timeEditCommand.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "comment":
			update.Comment = timeEditComment
		}
	})

	if *timeEditStart != "" {
		t, err := parseTimeFlag(*timeEditStart)
		if err != nil {
			return err
		}
		update.Dtstart = &t
	}
	if *timeEditEnd != "" {
		t, err := parseTimeFlag(*timeEditEnd)
		if err != nil {
			return err
		}
		update.Dtend = &t
	}
	if *timeEditDuration != "" {
		d, err := parseDurationFlag(*timeEditDuration)
		if err != nil {
			return err
		}
		update.Duration = &d
	}

	event, err := resolveEventUUID(worklog, *timeEditUUID)
	if err != nil {
		return err
	}
	if err := worklog.UpdateEvent(event.uuid, update); err != nil {
		return err
	}
	if err := worklog.Save(*timeEditOutput); err != nil {
		return err
	}
	fmt.Println("Time entry updated.")
	return nil
}

func runTimeDelete(args []string) error {
	timeDeleteCommand := flag.NewFlagSet("time delete", flag.ContinueOnError)
	timeDeleteFile := timeDeleteCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	timeDeleteOutput := timeDeleteCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
	timeDeleteUUID := timeDeleteCommand.String("uuid", "", "UUID (or short unique prefix) of the time entry to delete (required)")
	timeDeleteForce := timeDeleteCommand.Bool("force", false, "Delete without confirmation")
	configureFlagSet(timeDeleteCommand, "Delete a time entry.", "  worklog time delete -uuid <short-uuid>\n  worklog time delete -uuid <short-uuid> -force")
	if err := timeDeleteCommand.Parse(args); err != nil {
		return err
	}

	worklog, err := loadWorklog(*timeDeleteFile)
	if err != nil {
		return err
	}

	if *timeDeleteUUID == "" {
		return fmt.Errorf("-uuid flag is required for time delete")
	}

	event, err := resolveEventUUID(worklog, *timeDeleteUUID)
	if err != nil {
		return err
	}

	if !*timeDeleteForce {
		taskName := "(orphaned task)"
		if task := worklog.FindTaskByUUID(event.relatedTo); task != nil {
			taskName = task.name
		}
		fmt.Printf("Delete time entry for '%s' (%s)? [y/N] ", taskName, time.Duration(event.duration*int(time.Second)).String())
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	if err := worklog.DeleteEvent(event.uuid); err != nil {
		return err
	}
	if err := worklog.Save(*timeDeleteOutput); err != nil {
		return err
	}
	fmt.Println("Time entry deleted.")
	return nil
}

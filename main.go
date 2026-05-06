package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func resolveFilePath(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if envPath := os.Getenv("WORKLOG_FILE"); envPath != "" {
		return envPath, nil
	}
	return "", fmt.Errorf("no file path specified; use -file flag or set WORKLOG_FILE environment variable")
}

func loadWorklog(flagValue string) (*Worklog, error) {
	filePath, err := resolveFilePath(flagValue)
	if err != nil {
		return nil, err
	}
	return NewWorklog(filePath)
}

func run(args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: worklog <command> [<args>]")
		fmt.Println("")
		fmt.Println("Available commands:")
		fmt.Println("  task    Manage tasks")
		fmt.Println("  time    Manage time entries")
		fmt.Println("  report  Generate reports")
		return fmt.Errorf("no command provided")
	}

	switch args[0] {
	case "task":
		return runTask(args[1:])
	case "time":
		if len(args) < 2 {
			fmt.Println("Usage: worklog time <subcommand> [<args>]")
			fmt.Println("")
			fmt.Println("Available time subcommands:")
			fmt.Println("  add     Add a time entry")
			fmt.Println("  list    List time entries")
			fmt.Println("  edit    Edit a time entry")
			fmt.Println("  delete  Delete a time entry")
			return fmt.Errorf("no time subcommand provided")
		}
		timeSubcommand := args[1]
		timeArgs := args[2:]

		switch timeSubcommand {
		case "add":
			timeAddCommand := flag.NewFlagSet("time add", flag.ExitOnError)
			timeAddFile := timeAddCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
			timeAddOutput := timeAddCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
			timeAddTask := timeAddCommand.String("task", "", "UUID of the task to log time against (required)")
			timeAddDuration := timeAddCommand.String("duration", "", "Duration to log (e.g., 30m, 1h30m, 3600s) (required)")
			timeAddStart := timeAddCommand.String("start", "", "Start time (optional, defaults to now-duration)")
			timeAddNote := timeAddCommand.String("note", "", "Note for the time entry (optional, defaults to task name)")
			timeAddCommand.Parse(timeArgs)

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

			task := worklog.FindTaskByUUID(*timeAddTask)
			if task == nil {
				return fmt.Errorf("task with UUID '%s' not found", *timeAddTask)
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

			note := *timeAddNote
			if note == "" {
				note = task.name
			}

			event := NewEvent(*timeAddTask, start, end, duration, note)
			worklog.AddEvent(event)
			if err := worklog.Save(*timeAddOutput); err != nil {
				return err
			}
			fmt.Printf("Added %s time entry for '%s'.\n", time.Duration(duration*int(time.Second)).String(), task.name)
		case "list":
			timeListCommand := flag.NewFlagSet("time list", flag.ExitOnError)
			timeListFile := timeListCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
			timeListTask := timeListCommand.String("task", "", "Filter to a specific task UUID (optional)")
			timeListCommand.Parse(timeArgs)

			worklog, err := loadWorklog(*timeListFile)
			if err != nil {
				return err
			}

			events := worklog.GetEvents()
			if *timeListTask != "" {
				var filtered []*Event
				for _, event := range events {
					if event.relatedTo == *timeListTask {
						filtered = append(filtered, event)
					}
				}
				events = filtered
			}

			sort.Slice(events, func(i, j int) bool {
				return events[i].dtstart.After(events[j].dtstart)
			})

			fmt.Printf("%-36s %-30s %-20s %-20s %-10s\n", "UUID", "Task", "Start", "End", "Duration")
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
				fmt.Printf("%-36s %-30s %-20s %-20s %-10s\n", event.uuid, taskName, startStr, endStr, durStr)
			}
		case "edit":
			timeEditCommand := flag.NewFlagSet("time edit", flag.ExitOnError)
			timeEditFile := timeEditCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
			timeEditOutput := timeEditCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
			timeEditUUID := timeEditCommand.String("uuid", "", "UUID of the time entry to edit (required)")
			timeEditStart := timeEditCommand.String("start", "", "New start time")
			timeEditEnd := timeEditCommand.String("end", "", "New end time")
			timeEditDuration := timeEditCommand.String("duration", "", "New duration (e.g., 30m, 1h30m)")
			timeEditNote := timeEditCommand.String("note", "", "New note")
			timeEditCommand.Parse(timeArgs)

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
				case "note":
					update.Summary = timeEditNote
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

			if err := worklog.UpdateEvent(*timeEditUUID, update); err != nil {
				return err
			}
			if err := worklog.Save(*timeEditOutput); err != nil {
				return err
			}
			fmt.Println("Time entry updated.")
		case "delete":
			timeDeleteCommand := flag.NewFlagSet("time delete", flag.ExitOnError)
			timeDeleteFile := timeDeleteCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
			timeDeleteOutput := timeDeleteCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
			timeDeleteUUID := timeDeleteCommand.String("uuid", "", "UUID of the time entry to delete (required)")
			timeDeleteForce := timeDeleteCommand.Bool("force", false, "Delete without confirmation")
			timeDeleteCommand.Parse(timeArgs)

			worklog, err := loadWorklog(*timeDeleteFile)
			if err != nil {
				return err
			}

			if *timeDeleteUUID == "" {
				return fmt.Errorf("-uuid flag is required for time delete")
			}

			event := worklog.FindEventByUUID(*timeDeleteUUID)
			if event == nil {
				return fmt.Errorf("time entry with UUID '%s' not found", *timeDeleteUUID)
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

			if err := worklog.DeleteEvent(*timeDeleteUUID); err != nil {
				return err
			}
			if err := worklog.Save(*timeDeleteOutput); err != nil {
				return err
			}
			fmt.Println("Time entry deleted.")
		default:
			return fmt.Errorf("unknown time subcommand '%s'", timeSubcommand)
		}
	case "report":
		if len(args) < 2 {
			fmt.Println("Usage: worklog report <subcommand> [<args>]")
			fmt.Println("")
			fmt.Println("Available report subcommands:")
			fmt.Println("  timesheet  Generate a timesheet report")
			return fmt.Errorf("no report subcommand provided")
		}
		reportSubcommand := args[1]
		reportArgs := args[2:]

		switch reportSubcommand {
		case "timesheet":
			timesheetCommand := flag.NewFlagSet("report timesheet", flag.ExitOnError)
			timesheetFile := timesheetCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
			timesheetFrom := timesheetCommand.String("from", "", "Start date (YYYY-MM-DD, defaults to Monday of current week)")
			timesheetTo := timesheetCommand.String("to", "", "End date (YYYY-MM-DD, defaults to Sunday of current week)")
			timesheetFormat := timesheetCommand.String("format", "table", "Output format: table or csv")
			timesheetDecimal := timesheetCommand.Bool("decimal", false, "Display hours in decimal format (e.g., 1.50)")
			timesheetAll := timesheetCommand.Bool("all", false, "Use the full date range of all events")
			timesheetHideEmpty := timesheetCommand.Bool("hide-empty", false, "Hide days with no time entries")
			timesheetCommand.Parse(reportArgs)

			worklog, err := loadWorklog(*timesheetFile)
			if err != nil {
				return err
			}

			isDefaultRange := true
			from, to := currentWeekRange(time.Now())

			if *timesheetAll {
				isDefaultRange = false
				min, max := eventDateRange(worklog)
				if !min.IsZero() {
					from = min
				}
				if !max.IsZero() {
					to = max
				}
			}
			if *timesheetFrom != "" {
				isDefaultRange = false
				parsed, err := parseDateFlag(*timesheetFrom)
				if err != nil {
					return err
				}
				from = parsed
			}
			if *timesheetTo != "" {
				isDefaultRange = false
				parsed, err := parseDateFlag(*timesheetTo)
				if err != nil {
					return err
				}
				to = parsed
			}
			if from.After(to) {
				return fmt.Errorf("from date must not be after to date")
			}

			ts := generateTimesheet(worklog, from, to)
			if *timesheetHideEmpty {
				ts = hideEmptyColumns(ts)
			}

			if len(ts.rows) == 0 {
				fmt.Println("No time entries in the selected date range.")
				if isDefaultRange {
					fmt.Println("Use --all to see all data, or specify --from and --to.")
				}
				return nil
			}

			switch *timesheetFormat {
			case "table":
				printTimesheetTable(os.Stdout, ts, *timesheetDecimal)
			case "csv":
				printTimesheetCSV(os.Stdout, ts, *timesheetDecimal)
			default:
				return fmt.Errorf("unknown format '%s', use 'table' or 'csv'", *timesheetFormat)
			}
		default:
			return fmt.Errorf("unknown report subcommand '%s'", reportSubcommand)
		}
	default:
		return fmt.Errorf("unknown command '%s'", args[0])
	}
	return nil
}

func runTask(args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: worklog task <subcommand> [<args>]")
		fmt.Println("")
		fmt.Println("Available task subcommands:")
		fmt.Println("  list    List tasks")
		fmt.Println("  create  Create a new task")
		fmt.Println("  update  Update a task")
		fmt.Println("  delete  Delete a task")
		return fmt.Errorf("no task subcommand provided")
	}

	switch args[0] {
	case "list":
		return runTaskList(args[1:])
	case "create":
		return runTaskCreate(args[1:])
	case "update":
		return runTaskUpdate(args[1:])
	case "delete":
		return runTaskDelete(args[1:])
	default:
		return fmt.Errorf("unknown task subcommand '%s'", args[0])
	}
}

func runTaskList(args []string) error {
	listCommand := flag.NewFlagSet("task list", flag.ExitOnError)
	listFile := listCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	listCommand.Parse(args)

	worklog, err := loadWorklog(*listFile)
	if err != nil {
		return err
	}

	direct, totals := worklog.ComputeTaskTotals()
	nameWidth := maxTaskLineWidth(worklog.tasks, "")
	if nameWidth < 4 {
		nameWidth = 4
	}
	uuidWidth := 36
	durWidth := 10
	fmt.Printf("%-*s %-*s %*s %*s\n", uuidWidth, "UUID", nameWidth, "Name", durWidth, "Duration", durWidth, "Total")
	printTasks(worklog.tasks, "", direct, totals, nameWidth)
	return nil
}

func runTaskCreate(args []string) error {
	createCommand := flag.NewFlagSet("task create", flag.ExitOnError)
	createFile := createCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	createOutput := createCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
	createName := createCommand.String("name", "", "The name of the task")
	createDescription := createCommand.String("description", "", "Additional or more detailed description")
	createParent := createCommand.String("parent", "", "Unique ID of parent task")
	createCommand.Parse(args)

	worklog, err := loadWorklog(*createFile)
	if err != nil {
		return err
	}

	task := worklog.NewTask(*createName)
	task.description = *createDescription
	if *createParent != "" {
		parentTask := worklog.FindTaskByUUID(*createParent)
		if parentTask == nil {
			return fmt.Errorf("parent task with UUID '%s' not found", *createParent)
		}
		worklog.removeFromParent(task)
		task.parent = parentTask
		parentTask.children = append(parentTask.children, task)
	}
	if err := worklog.Save(*createOutput); err != nil {
		return err
	}
	return nil
}

func runTaskUpdate(args []string) error {
	updateCommand := flag.NewFlagSet("task update", flag.ExitOnError)
	updateFile := updateCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	updateOutput := updateCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
	updateUUID := updateCommand.String("uuid", "", "The UUID of the task to update (required)")
	updateName := updateCommand.String("name", "", "New name for the task")
	updateDescription := updateCommand.String("description", "", "New description for the task")
	updateParent := updateCommand.String("parent", "", "New parent UUID for the task")
	updateCommand.Parse(args)

	worklog, err := loadWorklog(*updateFile)
	if err != nil {
		return err
	}

	if *updateUUID == "" {
		return fmt.Errorf("-uuid flag is required for update command")
	}

	update := TaskUpdate{}
	updateCommand.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "name":
			update.Name = updateName
		case "description":
			update.Description = updateDescription
		case "parent":
			update.ParentUUID = updateParent
		}
	})

	if err := worklog.UpdateTask(*updateUUID, update); err != nil {
		return err
	}
	if err := worklog.Save(*updateOutput); err != nil {
		return err
	}
	return nil
}

func runTaskDelete(args []string) error {
	deleteCommand := flag.NewFlagSet("task delete", flag.ExitOnError)
	deleteFile := deleteCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	deleteOutput := deleteCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
	deleteUUID := deleteCommand.String("uuid", "", "The UUID of the task to delete (required)")
	deleteForce := deleteCommand.Bool("force", false, "Delete without confirmation")
	deleteCommand.Parse(args)

	worklog, err := loadWorklog(*deleteFile)
	if err != nil {
		return err
	}

	if *deleteUUID == "" {
		return fmt.Errorf("-uuid flag is required for delete command")
	}

	task := worklog.FindTaskByUUID(*deleteUUID)
	if task == nil {
		return fmt.Errorf("task with UUID '%s' not found", *deleteUUID)
	}

	count := countSubtasks(task)
	if !*deleteForce {
		fmt.Printf("This will delete '%s' and %d subtask(s).\n", task.name, count-1)
		fmt.Print("Continue? [y/N] ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	deleted, err := worklog.DeleteTask(*deleteUUID)
	if err != nil {
		return err
	}
	if err := worklog.Save(*deleteOutput); err != nil {
		return err
	}
	fmt.Printf("Deleted '%s' and %d subtask(s).\n", task.name, deleted-1)
	return nil
}

func formatDuration(seconds int) string {
	if seconds <= 0 {
		return "-"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	var parts []string
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if m > 0 {
		parts = append(parts, fmt.Sprintf("%dm", m))
	}
	if s > 0 && h == 0 {
		parts = append(parts, fmt.Sprintf("%ds", s))
	}
	return strings.Join(parts, " ")
}

func maxTaskLineWidth(tasks []*Task, prefix string) int {
	width := 0
	for _, task := range tasks {
		separator := "- "
		if prefix != "" {
			separator = "| - "
		}
		lineLen := len(prefix) + len(separator) + len(task.name)
		if lineLen > width {
			width = lineLen
		}
		if len(task.children) > 0 {
			childWidth := maxTaskLineWidth(task.children, "  "+prefix)
			if childWidth > width {
				width = childWidth
			}
		}
	}
	return width
}

func printTasks(tasks []*Task, prefix string, direct map[string]int, totals map[string]int, nameWidth int) {
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].name < tasks[j].name
	})
	for _, task := range tasks {
		separator := "- "
		if prefix != "" {
			separator = "| - "
		}
		name := prefix + separator + task.name
		directDur := formatDuration(direct[task.uuid])
		totalDur := formatDuration(totals[task.uuid])
		fmt.Printf("%-*s %-*s %*s %*s\n", 36, task.uuid, nameWidth, name, 10, directDur, 10, totalDur)
		if len(task.children) > 0 {
			printTasks(task.children, "  "+prefix, direct, totals, nameWidth)
		}
	}
}

func parseTimeFlag(value string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04",
		"2006-01-02 15:04",
		"15:04:05",
		"15:04",
	}
	for _, layout := range layouts {
		if layout == "15:04:05" || layout == "15:04" {
			t, err := time.ParseInLocation(layout, value, time.Local)
			if err == nil {
				now := time.Now()
				return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local), nil
			}
		} else {
			t, err := time.ParseInLocation(layout, value, time.Local)
			if err == nil {
				return t, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf("invalid time format: %s", value)
}

func parseDurationFlag(value string) (int, error) {
	d, err := time.ParseDuration(value)
	if err == nil {
		return int(d.Seconds()), nil
	}
	seconds, err := strconv.Atoi(value)
	if err == nil {
		return seconds, nil
	}
	return 0, fmt.Errorf("invalid duration format: %s (use Go duration like 30m, 1h30m, or seconds)", value)
}

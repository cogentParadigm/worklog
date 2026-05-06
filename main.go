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

func run(args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: worklog <command> [<args>]")
		fmt.Println("")
		fmt.Println("Available commands:")
		fmt.Println("  list    List tasks")
		fmt.Println("  create  Create a new task")
		fmt.Println("  update  Update a task")
		fmt.Println("  delete  Delete a task")
		fmt.Println("  time    Manage time entries")
		return fmt.Errorf("no command provided")
	}

	createCommand := flag.NewFlagSet("create", flag.ExitOnError)
	createName := createCommand.String("name", "", "The name of the task")
	createDescription := createCommand.String("description", "", "Additional or more detailed description")
	createParent := createCommand.String("parent", "", "Unique ID of parent task")

	updateCommand := flag.NewFlagSet("update", flag.ExitOnError)
	updateUUID := updateCommand.String("uuid", "", "The UUID of the task to update (required)")
	updateName := updateCommand.String("name", "", "New name for the task")
	updateDescription := updateCommand.String("description", "", "New description for the task")
	updateParent := updateCommand.String("parent", "", "New parent UUID for the task")

	deleteCommand := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteUUID := deleteCommand.String("uuid", "", "The UUID of the task to delete (required)")
	deleteForce := deleteCommand.Bool("force", false, "Delete without confirmation")

	worklog, err := NewWorklog("testdata/example.ics")
	if err != nil {
		return err
	}

	switch args[0] {
	case "list":
		printTasks(worklog.tasks, "")
	case "create":
		createCommand.Parse(args[1:])
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
		if err := worklog.Save(); err != nil {
			return err
		}
	case "update":
		updateCommand.Parse(args[1:])
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
		if err := worklog.Save(); err != nil {
			return err
		}
	case "delete":
		deleteCommand.Parse(args[1:])
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
		if err := worklog.Save(); err != nil {
			return err
		}
		fmt.Printf("Deleted '%s' and %d subtask(s).\n", task.name, deleted-1)
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
			timeAddTask := timeAddCommand.String("task", "", "UUID of the task to log time against (required)")
			timeAddDuration := timeAddCommand.String("duration", "", "Duration to log (e.g., 30m, 1h30m, 3600s) (required)")
			timeAddStart := timeAddCommand.String("start", "", "Start time (optional, defaults to now-duration)")
			timeAddNote := timeAddCommand.String("note", "", "Note for the time entry (optional, defaults to task name)")
			timeAddCommand.Parse(timeArgs)

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
			if err := worklog.Save(); err != nil {
				return err
			}
			fmt.Printf("Added %s time entry for '%s'.\n", time.Duration(duration*int(time.Second)).String(), task.name)
		case "list":
			timeListCommand := flag.NewFlagSet("time list", flag.ExitOnError)
			timeListTask := timeListCommand.String("task", "", "Filter to a specific task UUID (optional)")
			timeListCommand.Parse(timeArgs)

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
			timeEditUUID := timeEditCommand.String("uuid", "", "UUID of the time entry to edit (required)")
			timeEditStart := timeEditCommand.String("start", "", "New start time")
			timeEditEnd := timeEditCommand.String("end", "", "New end time")
			timeEditDuration := timeEditCommand.String("duration", "", "New duration (e.g., 30m, 1h30m)")
			timeEditNote := timeEditCommand.String("note", "", "New note")
			timeEditCommand.Parse(timeArgs)

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
			if err := worklog.Save(); err != nil {
				return err
			}
			fmt.Println("Time entry updated.")
		case "delete":
			timeDeleteCommand := flag.NewFlagSet("time delete", flag.ExitOnError)
			timeDeleteUUID := timeDeleteCommand.String("uuid", "", "UUID of the time entry to delete (required)")
			timeDeleteForce := timeDeleteCommand.Bool("force", false, "Delete without confirmation")
			timeDeleteCommand.Parse(timeArgs)

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
			if err := worklog.Save(); err != nil {
				return err
			}
			fmt.Println("Time entry deleted.")
		default:
			return fmt.Errorf("unknown time subcommand '%s'", timeSubcommand)
		}
	default:
		return fmt.Errorf("unknown command '%s'", args[0])
	}
	return nil
}

func printTasks(tasks []*Task, prefix string) {
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].name < tasks[j].name
	})
	for _, task := range tasks {
		separator := "- "
		if prefix != "" {
			separator = "|" + separator
		}
		fmt.Printf("%v%v%v\n", prefix, separator, task.name)
		if len(task.children) > 0 {
			printTasks(task.children, "  "+prefix)
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

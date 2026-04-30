package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
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

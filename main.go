package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: worklog <command> [<args>]")
		fmt.Println("")
		fmt.Println("Available commands:")
		fmt.Println("  list    List tasks")
		fmt.Println("  create  Create a new task")
		fmt.Println("  update  Update a task")
		fmt.Println("  delete  Delete a task")
		os.Exit(1)
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

	worklog := NewWorklog("testdata/example.ics")

	switch os.Args[1] {
	case "list":
		printTasks(worklog.tasks, "")
	case "create":
		createCommand.Parse(os.Args[2:])
		task := worklog.NewTask(*createName)
		task.description = *createDescription
		if *createParent != "" {

		}
		worklog.Save()
	case "update":
		updateCommand.Parse(os.Args[2:])
		if *updateUUID == "" {
			fmt.Println("Error: -uuid flag is required for update command")
			os.Exit(1)
		}
		err := worklog.UpdateTask(*updateUUID, *updateName, *updateDescription, *updateParent)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		worklog.Save()
	default:
		fmt.Printf("Unknown command '%v'\n", os.Args[1])
	}
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

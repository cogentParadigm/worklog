package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func isHelpFlag(s string) bool {
	return s == "-h" || s == "--help"
}

func printTopLevelUsage() {
	fmt.Println("Usage: worklog <command> [<args>]")
	fmt.Println("")
	fmt.Println("Available commands:")
	fmt.Println("  task    Manage tasks")
	fmt.Println("  time    Manage time entries")
	fmt.Println("  report  Generate reports")
	fmt.Println("  jira    Sync time entries to Jira/Tempo")
	fmt.Println("  config  Manage configuration")
	fmt.Println("  init    Initialize configuration")
}

func printTaskUsage() {
	fmt.Println("Usage: worklog task <subcommand> [<args>]")
	fmt.Println("")
	fmt.Println("Available task subcommands:")
	fmt.Println("  list    List tasks")
	fmt.Println("  create  Create a new task")
	fmt.Println("  update  Update a task")
	fmt.Println("  delete  Delete a task")
}

func matchesSearch(text, query string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(query))
}

func printTimeUsage() {
	fmt.Println("Usage: worklog time <subcommand> [<args>]")
	fmt.Println("")
	fmt.Println("Available time subcommands:")
	fmt.Println("  add     Add a time entry")
	fmt.Println("  list    List time entries")
	fmt.Println("  edit    Edit a time entry")
	fmt.Println("  delete  Delete a time entry")
}

func printReportUsage() {
	fmt.Println("Usage: worklog report <subcommand> [<args>]")
	fmt.Println("")
	fmt.Println("Available report subcommands:")
	fmt.Println("  timesheet  Generate a timesheet report")
}

func printConfigUsage() {
	fmt.Println("Usage: worklog config <subcommand> [<args>]")
	fmt.Println("")
	fmt.Println("Available config subcommands:")
	fmt.Println("  path    Print config file path")
	fmt.Println("  get     Get a config value")
	fmt.Println("  set     Set a config value")
}

var verbose bool

func configureFlagSet(fs *flag.FlagSet, description, examples string) {
	fs.BoolVar(&verbose, "v", false, "Verbose output")
	fs.BoolVar(&verbose, "verbose", false, "Verbose output")
	fs.SetOutput(os.Stdout)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: worklog %s [flags]\n\n", fs.Name())
		fmt.Fprintf(fs.Output(), "%s\n\n", description)
		if examples != "" {
			fmt.Fprintf(fs.Output(), "Examples:\n%s\n\n", examples)
		}
		fmt.Fprintf(fs.Output(), "Flags:\n")
		fs.PrintDefaults()
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
		cfg, cfgErr := LoadConfig()
		if cfgErr == nil && cfg.WorklogFile != "" {
			filePath = expandTilde(cfg.WorklogFile)
		} else {
			return nil, err
		}
	}
	return NewWorklog(filePath)
}

func run(args []string) error {
	if len(args) < 1 {
		printTopLevelUsage()
		return fmt.Errorf("no command provided")
	}
	if isHelpFlag(args[0]) {
		printTopLevelUsage()
		return flag.ErrHelp
	}

	commands := map[string]func([]string) error{
		"task":   runTask,
		"time":   runTime,
		"jira":   runJira,
		"config": runConfig,
		"init":   runInit,
		"report": runReport,
	}

	handler, ok := commands[args[0]]
	if !ok {
		return fmt.Errorf("unknown command '%s'", args[0])
	}
	return handler(args[1:])
}

func runTask(args []string) error {
	if len(args) < 1 {
		printTaskUsage()
		return fmt.Errorf("no task subcommand provided")
	}
	if isHelpFlag(args[0]) {
		printTaskUsage()
		return flag.ErrHelp
	}

	commands := map[string]func([]string) error{
		"list":   runTaskList,
		"create": runTaskCreate,
		"update": runTaskUpdate,
		"delete": runTaskDelete,
	}

	handler, ok := commands[args[0]]
	if !ok {
		return fmt.Errorf("unknown task subcommand '%s'", args[0])
	}
	return handler(args[1:])
}

func runTaskList(args []string) error {
	listCommand := flag.NewFlagSet("task list", flag.ContinueOnError)
	listFile := listCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	listSearch := listCommand.String("search", "", "Filter tasks by case-insensitive search in name or description")
	listParent := listCommand.String("parent", "", "Show only the specified task and its descendants (UUID or short unique prefix)")
	configureFlagSet(listCommand, "List all tasks, showing the hierarchy, UUID, direct duration, and total duration.", "  worklog task list\n  worklog task list -file ~/tasks.ics\n  worklog task list -search planning\n  worklog task list -parent <short-uuid>")
	if err := listCommand.Parse(args); err != nil {
		return err
	}

	worklog, err := loadWorklog(*listFile)
	if err != nil {
		return err
	}

	tasks := worklog.tasks
	if *listParent != "" {
		parentTask, err := resolveTaskUUID(worklog, *listParent)
		if err != nil {
			return err
		}
		tasks = []*Task{parentTask}
	}

	direct, totals := worklog.ComputeTaskTotals()

	allTaskUUIDs := worklog.allTaskUUIDs()
	shortTaskUUIDs := shortUUIDs(allTaskUUIDs)
	maxShortLen := 4
	for _, su := range shortTaskUUIDs {
		if len(su) > maxShortLen {
			maxShortLen = len(su)
		}
	}

	if *listSearch != "" {
		// Flat list of matching tasks
		allTasks := flattenTasks(tasks)
		var matching []*Task
		for _, task := range allTasks {
			if matchesSearch(task.name, *listSearch) || matchesSearch(task.description, *listSearch) {
				matching = append(matching, task)
			}
		}
		if len(matching) == 0 {
			fmt.Println("No tasks match the search criteria.")
			return nil
		}
		nameWidth := 4
		for _, task := range matching {
			if len(task.name) > nameWidth {
				nameWidth = len(task.name)
			}
		}
		fmt.Printf("%-*s %-*s %*s %*s\n", maxShortLen, "UUID", nameWidth, "Name", 10, "Duration", 10, "Total")
		for _, task := range matching {
			directDur := formatDuration(direct[task.uuid])
			totalDur := formatDuration(totals[task.uuid])
			fmt.Printf("%-*s %-*s %*s %*s\n", maxShortLen, shortTaskUUIDs[task.uuid], nameWidth, task.name, 10, directDur, 10, totalDur)
		}
		return nil
	}

	nameWidth := maxTaskLineWidth(tasks, "")
	if nameWidth < 4 {
		nameWidth = 4
	}
	durWidth := 10
	fmt.Printf("%-*s %-*s %*s %*s\n", maxShortLen, "UUID", nameWidth, "Name", durWidth, "Duration", durWidth, "Total")
	printTasks(tasks, "", direct, totals, nameWidth, shortTaskUUIDs)
	return nil
}

func runTaskCreate(args []string) error {
	createCommand := flag.NewFlagSet("task create", flag.ContinueOnError)
	createFile := createCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	createOutput := createCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
	createName := createCommand.String("name", "", "The name of the task")
	createDescription := createCommand.String("description", "", "Additional or more detailed description")
	createParent := createCommand.String("parent", "", "Unique ID (or short unique prefix) of parent task")
	createIssueKey := createCommand.String("issue-key", "", "Explicit Jira issue key for this task")
	configureFlagSet(createCommand, "Create a new task with an auto-generated UUID.", "  worklog task create -name \"Project Setup\"\n  worklog task create -name \"Subtask\" -parent <short-uuid>\n  worklog task create -file ~/tasks.ics -output ~/backup.ics -name \"Backup\"")
	if err := createCommand.Parse(args); err != nil {
		return err
	}

	worklog, err := loadWorklog(*createFile)
	if err != nil {
		return err
	}

	task := worklog.NewTask(*createName)
	task.description = *createDescription
	if *createIssueKey != "" {
		task.properties = append(task.properties, ics.IANAProperty{
			BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-ISSUE-KEY", Value: *createIssueKey},
		})
	}
	if *createParent != "" {
		parentTask, err := resolveTaskUUID(worklog, *createParent)
		if err != nil {
			return err
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
	updateCommand := flag.NewFlagSet("task update", flag.ContinueOnError)
	updateFile := updateCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	updateOutput := updateCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
	updateUUID := updateCommand.String("uuid", "", "The UUID (or short unique prefix) of the task to update (required)")
	updateName := updateCommand.String("name", "", "New name for the task")
	updateDescription := updateCommand.String("description", "", "New description for the task")
	updateParent := updateCommand.String("parent", "", "New parent UUID (or short unique prefix) for the task")
	updateIssueKey := updateCommand.String("issue-key", "", "Explicit Jira issue key for this task")
	updateAttrs := updateCommand.String("attr", "", "Tempo work attributes as comma-separated key=value pairs (e.g. _WorkType_=Development)")
	configureFlagSet(updateCommand, "Update an existing task. Only provided fields are changed.", "  worklog task update -uuid <short-uuid> -name \"New Name\"\n  worklog task update -uuid <short-uuid> -parent \"\"\n  worklog task update -uuid <short-uuid> -description \"Details\"\n  worklog task update -uuid <short-uuid> -attr _WorkType_=Development")
	if err := updateCommand.Parse(args); err != nil {
		return err
	}

	worklog, err := loadWorklog(*updateFile)
	if err != nil {
		return err
	}

	if *updateUUID == "" {
		return fmt.Errorf("-uuid flag is required for update command")
	}

	resolvedTask, err := resolveTaskUUID(worklog, *updateUUID)
	if err != nil {
		return err
	}
	resolvedUUID := resolvedTask.uuid

	issueKeySet := false
	attrSet := false
	update := TaskUpdate{}
	updateCommand.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "name":
			update.Name = *updateName
			update.NameSet = true
		case "description":
			update.Description = *updateDescription
			update.DescriptionSet = true
		case "parent":
			update.ParentUUID = *updateParent
			update.ParentUUIDSet = true
		case "issue-key":
			issueKeySet = true
		case "attr":
			attrSet = true
		}
	})

	// Resolve parent UUID if provided and non-empty
	if update.ParentUUIDSet && update.ParentUUID != "" {
		parentTask, err := resolveTaskUUID(worklog, update.ParentUUID)
		if err != nil {
			return err
		}
		update.ParentUUID = parentTask.uuid
	}

	if err := worklog.UpdateTask(resolvedUUID, update); err != nil {
		return err
	}

	if issueKeySet {
		task := worklog.FindTaskByUUID(resolvedUUID)
		var newProps []ics.IANAProperty
		for _, prop := range task.properties {
			if prop.IANAToken != "X-WORKLOG-ISSUE-KEY" && prop.IANAToken != "X-WORKLOG-ISSUE-ID" {
				newProps = append(newProps, prop)
			}
		}
		if *updateIssueKey != "" {
			newProps = append(newProps, ics.IANAProperty{
				BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-ISSUE-KEY", Value: *updateIssueKey},
			})
		}
		task.properties = newProps
	}

	if attrSet {
		task := worklog.FindTaskByUUID(resolvedUUID)
		if task == nil {
			return fmt.Errorf("task with UUID '%s' not found", shortUUID(resolvedUUID, worklog.allTaskUUIDs()))
		}
		if *updateAttrs == "" {
			// Clear all tempo attributes
			var newProps []ics.IANAProperty
			for _, prop := range task.properties {
				if prop.IANAToken != "X-WORKLOG-TEMPO-ATTR" {
					newProps = append(newProps, prop)
				}
			}
			task.properties = newProps
		} else {
			attrs, err := parseTempoAttributes(*updateAttrs)
			if err != nil {
				return err
			}
			for k, v := range attrs {
				task.SetTempoAttribute(k, v)
			}
		}
	}

	if err := worklog.Save(*updateOutput); err != nil {
		return err
	}
	return nil
}

func runTaskDelete(args []string) error {
	deleteCommand := flag.NewFlagSet("task delete", flag.ContinueOnError)
	deleteFile := deleteCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	deleteOutput := deleteCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
	deleteUUID := deleteCommand.String("uuid", "", "The UUID (or short unique prefix) of the task to delete (required)")
	deleteForce := deleteCommand.Bool("force", false, "Delete without confirmation")
	configureFlagSet(deleteCommand, "Delete a task and all of its subtasks recursively.", "  worklog task delete -uuid <short-uuid>\n  worklog task delete -uuid <short-uuid> -force")
	if err := deleteCommand.Parse(args); err != nil {
		return err
	}

	worklog, err := loadWorklog(*deleteFile)
	if err != nil {
		return err
	}

	if *deleteUUID == "" {
		return fmt.Errorf("-uuid flag is required for delete command")
	}

	task, err := resolveTaskUUID(worklog, *deleteUUID)
	if err != nil {
		return err
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

	deleted, err := worklog.DeleteTask(task.uuid)
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

func printTasks(tasks []*Task, prefix string, direct map[string]int, totals map[string]int, nameWidth int, shortUUIDs map[string]string) {
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].name < tasks[j].name
	})
	maxShortLen := 4
	for _, su := range shortUUIDs {
		if len(su) > maxShortLen {
			maxShortLen = len(su)
		}
	}
	for _, task := range tasks {
		separator := "- "
		if prefix != "" {
			separator = "| - "
		}
		name := prefix + separator + task.name
		directDur := formatDuration(direct[task.uuid])
		totalDur := formatDuration(totals[task.uuid])
		fmt.Printf("%-*s %-*s %*s %*s\n", maxShortLen, shortUUIDs[task.uuid], nameWidth, name, 10, directDur, 10, totalDur)
		if len(task.children) > 0 {
			printTasks(task.children, "  "+prefix, direct, totals, nameWidth, shortUUIDs)
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

func runConfig(args []string) error {
	if len(args) < 1 {
		printConfigUsage()
		return fmt.Errorf("no config subcommand provided")
	}
	if isHelpFlag(args[0]) {
		printConfigUsage()
		return flag.ErrHelp
	}

	commands := map[string]func([]string) error{
		"path": runConfigPath,
		"get":  runConfigGet,
		"set":  runConfigSet,
	}

	handler, ok := commands[args[0]]
	if !ok {
		return fmt.Errorf("unknown config subcommand '%s'", args[0])
	}
	return handler(args[1:])
}

func runConfigPath(args []string) error {
	fmt.Println(configPath())
	return nil
}

func runConfigGet(args []string) error {
	getCommand := flag.NewFlagSet("config get", flag.ContinueOnError)
	show := getCommand.Bool("show", false, "Show unmasked secret values")
	configureFlagSet(getCommand, "Get a config value by key.", "  worklog config get worklog_file\n  worklog config get --show tempo.token")
	if err := getCommand.Parse(args); err != nil {
		return err
	}
	if getCommand.NArg() != 1 {
		return fmt.Errorf("expected exactly one key argument")
	}
	key := getCommand.Arg(0)
	if !isValidConfigKey(key) {
		return fmt.Errorf("unknown config key: %s", key)
	}

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	var value string
	if *show {
		value, err = cfg.GetUnmasked(key)
	} else {
		value, err = cfg.Get(key)
	}
	if err != nil {
		return err
	}
	fmt.Println(value)
	return nil
}

func runConfigSet(args []string) error {
	setCommand := flag.NewFlagSet("config set", flag.ContinueOnError)
	configureFlagSet(setCommand, "Set a config value by key.", "  worklog config set worklog_file ~/tasks.ics\n  worklog config set tempo.token pass:worklog/tempo-token")
	if err := setCommand.Parse(args); err != nil {
		return err
	}
	if setCommand.NArg() != 2 {
		return fmt.Errorf("expected key and value arguments")
	}
	key := setCommand.Arg(0)
	value := setCommand.Arg(1)
	if !isValidConfigKey(key) {
		return fmt.Errorf("unknown config key: %s", key)
	}

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	if err := cfg.Set(key, value); err != nil {
		return err
	}

	if key == "worklog_file" && value != "" {
		expanded := expandTilde(value)
		dir := filepath.Dir(expanded)
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("worklog_file parent directory %s is not writable: %w", dir, err)
			}
		}
	}

	if (key == "tempo.base_url" || key == "jira.base_url") && value != "" {
		if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") {
			return fmt.Errorf("%s must start with http:// or https://", key)
		}
	}

	if err := SaveConfig(cfg); err != nil {
		return err
	}

	if (key == "tempo.token" || key == "jira.token") && value != "" && !strings.HasPrefix(value, "pass:") {
		fmt.Fprintf(os.Stderr, "Tip: store this in pass and set to pass:worklog/%s for better security.\n", strings.Replace(key, ".", "-", 1))
	}

	fmt.Println("Config updated.")
	return nil
}

func detectExistingICSFiles() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	var candidates []string
	patterns := []string{
		filepath.Join(home, ".local", "share", "ktimetracker", "*.ics"),
		filepath.Join(home, ".kde", "share", "apps", "ktimetracker", "*.ics"),
		filepath.Join(home, "*.ics"),
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			found := false
			for _, c := range candidates {
				if c == m {
					found = true
					break
				}
			}
			if !found {
				candidates = append(candidates, m)
			}
		}
	}
	return candidates
}

func createSkeletonICS(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	cal := ics.NewCalendar()
	cal.SetProductId("-//Worklog//Worklog//EN")
	cal.SetVersion("2.0")
	return os.WriteFile(path, []byte(cal.Serialize()), 0644)
}

func runInit(args []string) error {
	initCommand := flag.NewFlagSet("init", flag.ContinueOnError)
	worklogFileFlag := initCommand.String("worklog-file", "", "Default worklog .ics file path")
	tempoBaseURL := initCommand.String("tempo-base-url", "https://api.tempo.io/4", "Tempo Cloud base URL")
	tempoAccountID := initCommand.String("tempo-account-id", "", "Atlassian account ID")
	tempoToken := initCommand.String("tempo-token", "", "Tempo API token")
	tempoRounding := initCommand.String("tempo-rounding", "", "Tempo rounding steps (e.g. floor:1m,ceil:5m)")
	jiraBaseURL := initCommand.String("jira-base-url", "", "Jira base URL")
	jiraUsername := initCommand.String("jira-username", "", "Jira username")
	jiraToken := initCommand.String("jira-token", "", "Jira API token")
	skipTempo := initCommand.Bool("skip-tempo", false, "Skip Tempo configuration")
	skipJira := initCommand.Bool("skip-jira", false, "Skip Jira configuration")
	force := initCommand.Bool("force", false, "Overwrite existing config")
	configureFlagSet(initCommand, "Initialize worklog configuration.", "  worklog init\n  worklog init --worklog-file ~/tasks.ics\n  worklog init --worklog-file ~/tasks.ics --tempo-token pass:worklog/token --jira-username user@example.com --jira-token pass:worklog/jira-token")
	if err := initCommand.Parse(args); err != nil {
		return err
	}

	path := configPath()
	if _, err := os.Stat(path); err == nil && !*force {
		return fmt.Errorf("config already exists at %s; use --force to overwrite", path)
	}

	cfg := &Config{}
	var filePath string

	if *worklogFileFlag != "" {
		// Non-interactive mode
		filePath = *worklogFileFlag
		if !*skipTempo {
			cfg.Tempo.BaseURL = *tempoBaseURL
			cfg.Tempo.AccountID = *tempoAccountID
			cfg.Tempo.Token = *tempoToken
			if *tempoRounding != "" {
				steps, err := parseRoundingSteps(*tempoRounding)
				if err != nil {
					return fmt.Errorf("--tempo-rounding: %w", err)
				}
				cfg.Tempo.Rounding = steps
			}
		}
		if !*skipJira {
			cfg.Jira.BaseURL = *jiraBaseURL
			cfg.Jira.Username = *jiraUsername
			cfg.Jira.Token = *jiraToken
		}
	} else {
		// Interactive mode
		detected := detectExistingICSFiles()
		defaultPath := ""
		if len(detected) > 0 {
			fmt.Println("Detected existing .ics files:")
			for i, f := range detected {
				fmt.Printf("  %d. %s\n", i+1, f)
			}
			fmt.Println()
			defaultPath = detected[0]
		} else {
			home, _ := os.UserHomeDir()
			defaultPath = filepath.Join(home, "worklog.ics")
		}

		fmt.Printf("Enter default worklog file path [%s]: ", defaultPath)
		var input string
		fmt.Scanln(&input)
		input = strings.TrimSpace(input)
		if input == "" {
			filePath = defaultPath
		} else if n, err := strconv.Atoi(input); err == nil && n >= 1 && n <= len(detected) {
			filePath = detected[n-1]
		} else {
			filePath = input
		}

		// Create file if needed
		expanded := expandTilde(filePath)
		if _, err := os.Stat(expanded); os.IsNotExist(err) {
			fmt.Printf("File does not exist. Create %s? [Y/n] ", filePath)
			var response string
			fmt.Scanln(&response)
			if strings.ToLower(strings.TrimSpace(response)) != "n" {
				if err := createSkeletonICS(expanded); err != nil {
					return fmt.Errorf("create skeleton .ics file: %w", err)
				}
				fmt.Printf("Created %s\n", filePath)
			}
		}

		fmt.Print("Configure Tempo integration? [y/N] ")
		var tempoResponse string
		fmt.Scanln(&tempoResponse)
		if strings.ToLower(strings.TrimSpace(tempoResponse)) == "y" {
			fmt.Printf("Tempo base URL [%s]: ", *tempoBaseURL)
			var urlInput string
			fmt.Scanln(&urlInput)
			if strings.TrimSpace(urlInput) != "" {
				*tempoBaseURL = strings.TrimSpace(urlInput)
			}
			fmt.Print("Atlassian account ID: ")
			fmt.Scanln(&cfg.Tempo.AccountID)
			fmt.Print("Tempo API token: ")
			fmt.Scanln(&cfg.Tempo.Token)
			cfg.Tempo.BaseURL = *tempoBaseURL
			if cfg.Tempo.Token != "" && !strings.HasPrefix(cfg.Tempo.Token, "pass:") {
				fmt.Fprintln(os.Stderr, "Tip: store this in pass and set to pass:worklog/tempo-token for better security.")
			}
		}

		fmt.Print("Configure Jira integration? [y/N] ")
		var jiraResponse string
		fmt.Scanln(&jiraResponse)
		if strings.ToLower(strings.TrimSpace(jiraResponse)) == "y" {
			fmt.Print("Jira base URL: ")
			fmt.Scanln(&cfg.Jira.BaseURL)
			fmt.Print("Jira username: ")
			fmt.Scanln(&cfg.Jira.Username)
			fmt.Print("Jira API token: ")
			fmt.Scanln(&cfg.Jira.Token)
			if cfg.Jira.Token != "" && !strings.HasPrefix(cfg.Jira.Token, "pass:") {
				fmt.Fprintln(os.Stderr, "Tip: store this in pass and set to pass:worklog/jira-token for better security.")
			}
		}
	}

	cfg.WorklogFile = filePath

	// Expand and create file in non-interactive mode if needed
	if *worklogFileFlag != "" {
		expanded := expandTilde(filePath)
		if _, err := os.Stat(expanded); os.IsNotExist(err) {
			if err := createSkeletonICS(expanded); err != nil {
				return fmt.Errorf("create skeleton .ics file: %w", err)
			}
			fmt.Printf("Created %s\n", filePath)
		}
	}

	if err := SaveConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("Config written to %s\n", path)
	return nil
}

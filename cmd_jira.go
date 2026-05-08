package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/cogentParadigm/worklog/internal/jira"
	"github.com/cogentParadigm/worklog/internal/tempo"
)

type syncEntry struct {
	task     *Task
	issueKey string
	start    time.Time
	duration int
	comment  string
	events   []*Event
}

func printJiraUsage() {
	fmt.Println("Usage: worklog jira <subcommand> [<args>]")
	fmt.Println("")
	fmt.Println("Available jira subcommands:")
	fmt.Println("  resolve    Resolve Jira issue keys to numeric IDs")
	fmt.Println("  sync       Sync time entries to Tempo")
}

func runJira(args []string) error {
	if len(args) < 1 {
		printJiraUsage()
		return fmt.Errorf("no jira subcommand provided")
	}
	if isHelpFlag(args[0]) {
		printJiraUsage()
		return flag.ErrHelp
	}

	switch args[0] {
	case "resolve":
		return runJiraResolve(args[1:])
	case "sync":
		return runJiraSync(args[1:])
	default:
		return fmt.Errorf("unknown jira subcommand '%s'", args[0])
	}
}

func collectAllTasks(tasks []*Task) []*Task {
	var result []*Task
	for _, task := range tasks {
		result = append(result, task)
		result = append(result, collectAllTasks(task.children)...)
	}
	return result
}

func runJiraResolve(args []string) error {
	resolveCommand := flag.NewFlagSet("jira resolve", flag.ContinueOnError)
	resolveFile := resolveCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	resolveOutput := resolveCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
	resolveTask := resolveCommand.String("task", "", "UUID of a specific task to resolve (optional)")
	configureFlagSet(resolveCommand, "Resolve Jira issue keys to numeric IDs and cache them on tasks.", "  worklog jira resolve\n  worklog jira resolve --task <uuid>")
	if err := resolveCommand.Parse(args); err != nil {
		return err
	}

	worklog, err := loadWorklog(*resolveFile)
	if err != nil {
		return err
	}

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	jiraToken, err := cfg.ResolveJiraToken()
	if err != nil {
		return fmt.Errorf("jira configuration: %w", err)
	}
	if cfg.Jira.BaseURL == "" {
		return fmt.Errorf("jira.base_url not configured")
	}
	jiraClient := jira.NewClient(cfg.Jira.BaseURL, jiraToken)

	var tasks []*Task
	if *resolveTask != "" {
		task := worklog.FindTaskByUUID(*resolveTask)
		if task == nil {
			return fmt.Errorf("task with UUID '%s' not found", *resolveTask)
		}
		tasks = []*Task{task}
	} else {
		tasks = collectAllTasks(worklog.tasks)
	}

	resolved := 0
	skipped := 0

	for _, task := range tasks {
		issueKey := task.IssueKey()
		if issueKey == "" {
			continue
		}
		if task.IssueID() != "" {
			skipped++
			continue
		}
		id, err := jiraClient.GetIssueID(issueKey)
		if err != nil {
			return fmt.Errorf("resolve issue key %s for task '%s': %w", issueKey, task.name, err)
		}
		task.SetIssueID(id)
		resolved++
	}

	if err := worklog.Save(*resolveOutput); err != nil {
		return fmt.Errorf("save worklog: %w", err)
	}

	fmt.Printf("Resolved %d issue key(s), skipped %d already cached.\n", resolved, skipped)
	return nil
}

func runJiraSync(args []string) error {
	syncCommand := flag.NewFlagSet("jira sync", flag.ContinueOnError)
	syncFile := syncCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	syncOutput := syncCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
	syncTask := syncCommand.String("task", "", "UUID of a specific task to sync (optional)")
	syncFrom := syncCommand.String("from", "", "Start date for sync range (YYYY-MM-DD)")
	syncTo := syncCommand.String("to", "", "End date for sync range (YYYY-MM-DD)")
	syncDryRun := syncCommand.Bool("dry-run", false, "Preview what would be synced without sending")
	syncForce := syncCommand.Bool("force", false, "Sync without confirmation prompt")
	syncFormat := syncCommand.String("format", "list", "Preview format: list or timesheet")
	syncHideEmpty := syncCommand.Bool("hide-empty", false, "Hide days with no time entries (timesheet format only)")
	syncDecimal := syncCommand.Bool("decimal", false, "Display hours in decimal format (timesheet format only)")
	configureFlagSet(syncCommand, "Sync time entries to Tempo Cloud. Entries are merged by task and date before sending.", "  worklog jira sync --dry-run\n  worklog jira sync --task <uuid>\n  worklog jira sync --from 2026-05-01 --to 2026-05-07\n  worklog jira sync --dry-run --format timesheet --hide-empty")
	if err := syncCommand.Parse(args); err != nil {
		return err
	}

	worklog, err := loadWorklog(*syncFile)
	if err != nil {
		return err
	}

	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	token, err := cfg.ResolveTempoToken()
	if err != nil {
		return fmt.Errorf("tempo configuration: %w", err)
	}
	if cfg.Tempo.AccountID == "" {
		return fmt.Errorf("tempo account_id not configured")
	}
	baseURL := cfg.Tempo.BaseURL
	if baseURL == "" {
		baseURL = "https://api.tempo.io/4"
	}

	client := tempo.NewClient(baseURL, token, cfg.Tempo.AccountID)

	jiraToken, err := cfg.ResolveJiraToken()
	if err != nil {
		return fmt.Errorf("jira configuration: %w", err)
	}
	var jiraClient *jira.Client
	if cfg.Jira.BaseURL != "" {
		jiraClient = jira.NewClient(cfg.Jira.BaseURL, jiraToken)
	}

	var fromDay, toDay time.Time
	if *syncFrom != "" {
		fromDay, err = parseDateFlag(*syncFrom)
		if err != nil {
			return err
		}
	}
	if *syncTo != "" {
		toDay, err = parseDateFlag(*syncTo)
		if err != nil {
			return err
		}
	}

	entries, err := buildSyncEntries(worklog, *syncTask, fromDay, toDay)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Println("No entries to sync.")
		return nil
	}

	if *syncFormat == "timesheet" {
		ts := buildTimesheetFromSyncEntries(entries, fromDay, toDay)
		if *syncHideEmpty {
			ts = hideEmptyColumns(ts)
		}
		printTimesheetTable(os.Stdout, ts, *syncDecimal)
		fmt.Printf("\nTotal: %d worklog(s) to send.\n", len(entries))
	} else {
		fmt.Printf("%-30s %-12s %-10s %-8s %s\n", "Task", "Issue Key", "Date", "Duration", "Comment")
		fmt.Println(strings.Repeat("-", 80))
		for _, e := range entries {
			durStr := formatDuration(e.duration)
			dateStr := e.start.Format("2006-01-02")
			name := truncate(e.task.name, 30)
			comment := truncate(e.comment, 30)
			fmt.Printf("%-30s %-12s %-10s %-8s %s\n", name, e.issueKey, dateStr, durStr, comment)
		}
		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("Total: %d worklog(s) to send.\n", len(entries))
	}

	if *syncDryRun {
		fmt.Println("Dry run complete. No entries were sent.")
		return nil
	}

	if !*syncForce {
		fmt.Print("Send to Tempo? [y/N] ")
		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			return fmt.Errorf("failed to read confirmation: %w", err)
		}
		if strings.ToLower(strings.TrimSpace(response)) != "y" {
			fmt.Println("Sync cancelled.")
			return nil
		}
	}

	for _, e := range entries {
		issueID := e.task.IssueID()
		if issueID == "" {
			if jiraClient == nil {
				return fmt.Errorf("jira.base_url and jira.token required to resolve issue key %s", e.issueKey)
			}
			id, err := jiraClient.GetIssueID(e.issueKey)
			if err != nil {
				return fmt.Errorf("resolve issue key %s: %w", e.issueKey, err)
			}
			e.task.SetIssueID(id)
			issueID = id
		}
		wl := tempo.Worklog{
			IssueId:          issueID,
			TimeSpentSeconds: e.duration,
			StartDate:        e.start.Format("2006-01-02"),
			StartTime:        e.start.Format("15:04:05"),
			Description:      e.comment,
		}
		if err := client.CreateWorklog(wl); err != nil {
			return fmt.Errorf("send worklog for '%s' (%s): %w", e.task.name, e.issueKey, err)
		}
		now := time.Now()
		for _, event := range e.events {
			event.SetSyncedAt(now)
		}
	}

	if err := worklog.Save(*syncOutput); err != nil {
		return fmt.Errorf("save worklog: %w", err)
	}

	fmt.Printf("Successfully synced %d worklog(s).\n", len(entries))
	return nil
}

func buildSyncEntries(worklog *Worklog, taskUUID string, fromDay, toDay time.Time) ([]syncEntry, error) {
	type groupKey struct {
		taskUUID string
		day      string
		issueKey string
	}
	groups := make(map[groupKey]*syncEntry)

	events := worklog.GetEvents()
	for _, event := range events {
		if event.dtstart.IsZero() {
			continue
		}

		day := time.Date(event.dtstart.Year(), event.dtstart.Month(), event.dtstart.Day(), 0, 0, 0, 0, event.dtstart.Location())
		if !fromDay.IsZero() && day.Before(fromDay) {
			continue
		}
		if !toDay.IsZero() && day.After(toDay) {
			continue
		}

		task := worklog.FindTaskByUUID(event.relatedTo)
		if task == nil {
			continue
		}
		if taskUUID != "" && task.uuid != taskUUID {
			continue
		}

		issueKey := task.IssueKey()
		if issueKey == "" {
			continue
		}

		key := groupKey{taskUUID: task.uuid, day: day.Format("2006-01-02"), issueKey: issueKey}
		g, ok := groups[key]
		if !ok {
			g = &syncEntry{
				task:     task,
				issueKey: issueKey,
				start:    event.dtstart,
			}
			groups[key] = g
		}

		g.duration += event.duration
		g.events = append(g.events, event)
		if event.dtstart.Before(g.start) {
			g.start = event.dtstart
		}
	}

	var entries []syncEntry
	for _, g := range groups {
		// Check if any event in the group needs syncing
		needsSync := false
		for _, event := range g.events {
			syncedAt := event.SyncedAt()
			lastModified := event.LastModified()
			if syncedAt.IsZero() || (!lastModified.IsZero() && syncedAt.Before(lastModified)) {
				needsSync = true
				break
			}
		}
		if !needsSync {
			continue
		}

		// Combine unique non-empty comments
		seen := make(map[string]bool)
		var parts []string
		for _, event := range g.events {
			c := strings.TrimSpace(event.comment)
			if c != "" && !seen[c] {
				seen[c] = true
				parts = append(parts, c)
			}
		}
		if len(parts) > 0 {
			g.comment = strings.Join(parts, "; ")
		}

		entries = append(entries, *g)
	}

	sort.Slice(entries, func(i, j int) bool {
		if !entries[i].start.Equal(entries[j].start) {
			return entries[i].start.Before(entries[j].start)
		}
		return entries[i].task.name < entries[j].task.name
	})

	return entries, nil
}

func buildTimesheetFromSyncEntries(entries []syncEntry, from, to time.Time) *Timesheet {
	var minDay, maxDay time.Time
	for _, e := range entries {
		day := time.Date(e.start.Year(), e.start.Month(), e.start.Day(), 0, 0, 0, 0, e.start.Location())
		if minDay.IsZero() || day.Before(minDay) {
			minDay = day
		}
		if maxDay.IsZero() || day.After(maxDay) {
			maxDay = day
		}
	}

	if !from.IsZero() {
		minDay = from
	}
	if !to.IsZero() {
		maxDay = to
	}

	var days []time.Time
	day := minDay
	endDay := maxDay
	for !day.After(endDay) {
		days = append(days, day)
		day = day.AddDate(0, 0, 1)
	}

	dayIndex := make(map[string]int)
	for i, d := range days {
		dayIndex[d.Format("2006-01-02")] = i
	}

	taskDurations := make(map[string][]int)
	taskMap := make(map[string]*Task)
	for _, e := range entries {
		dayStr := time.Date(e.start.Year(), e.start.Month(), e.start.Day(), 0, 0, 0, 0, e.start.Location()).Format("2006-01-02")
		idx, ok := dayIndex[dayStr]
		if !ok {
			continue
		}
		if _, ok := taskDurations[e.task.uuid]; !ok {
			taskDurations[e.task.uuid] = make([]int, len(days))
			taskMap[e.task.uuid] = e.task
		}
		taskDurations[e.task.uuid][idx] += e.duration
	}

	var rows []TimesheetRow
	for uuid, durations := range taskDurations {
		total := 0
		for _, d := range durations {
			total += d
		}
		rows = append(rows, TimesheetRow{
			task:      taskMap[uuid],
			durations: durations,
			total:     total,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].task.name < rows[j].task.name
	})

	totals := make([]int, len(days))
	for i := range days {
		for _, row := range rows {
			totals[i] += row.durations[i]
		}
	}

	return &Timesheet{
		days:   days,
		rows:   rows,
		totals: totals,
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

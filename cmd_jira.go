package main

import (
	"flag"
	"fmt"
	"sort"
	"strings"
	"time"

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
	fmt.Println("  sync    Sync time entries to Tempo")
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
	case "sync":
		return runJiraSync(args[1:])
	default:
		return fmt.Errorf("unknown jira subcommand '%s'", args[0])
	}
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
	configureFlagSet(syncCommand, "Sync time entries to Tempo Cloud. Entries are merged by task and date before sending.", "  worklog jira sync --dry-run\n  worklog jira sync --task <uuid>\n  worklog jira sync --from 2026-05-01 --to 2026-05-07")
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
		baseURL = "https://api.tempo.io/core/3"
	}

	client := tempo.NewClient(baseURL, token, cfg.Tempo.AccountID)

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
		wl := tempo.Worklog{
			IssueKey:         e.issueKey,
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

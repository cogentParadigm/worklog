package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
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
	syncHash string
}

func printJiraUsage() {
	fmt.Println("Usage: worklog jira <subcommand> [<args>]")
	fmt.Println("")
	fmt.Println("Available jira subcommands:")
	fmt.Println("  resolve     Resolve Jira issue keys to numeric IDs")
	fmt.Println("  sync        Sync time entries to Tempo")
	fmt.Println("  attributes  List available Tempo work attributes")
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
	case "attributes":
		return runJiraAttributes(args[1:])
	default:
		return fmt.Errorf("unknown jira subcommand '%s'", args[0])
	}
}

func runJiraResolve(args []string) error {
	resolveCommand := flag.NewFlagSet("jira resolve", flag.ContinueOnError)
	resolveFile := resolveCommand.String("file", "", "Path to .ics file (overrides WORKLOG_FILE)")
	resolveOutput := resolveCommand.String("output", "", "Output path for the updated .ics file (defaults to input file)")
	resolveTask := resolveCommand.String("task", "", "UUID (or short unique prefix) of a specific task to resolve (optional)")
	configureFlagSet(resolveCommand, "Resolve Jira issue keys to numeric IDs and cache them on tasks.", "  worklog jira resolve\n  worklog jira resolve --task <short-uuid>")
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
	if cfg.Jira.Username == "" {
		return fmt.Errorf("jira.username not configured")
	}
	jiraClient := jira.NewClient(cfg.Jira.BaseURL, cfg.Jira.Username, jiraToken)

	var tasks []*Task
	if *resolveTask != "" {
		task, err := resolveTaskUUID(worklog, *resolveTask)
		if err != nil {
			return err
		}
		tasks = []*Task{task}
	} else {
		tasks = flattenTasks(worklog.tasks)
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
	syncTask := syncCommand.String("task", "", "UUID (or short unique prefix) of a specific task to sync (optional)")
	syncFrom := syncCommand.String("from", "", "Start date for sync range (YYYY-MM-DD)")
	syncTo := syncCommand.String("to", "", "End date for sync range (YYYY-MM-DD)")
	syncDryRun := syncCommand.Bool("dry-run", false, "Preview what would be synced without sending")
	syncForce := syncCommand.Bool("force", false, "Sync without confirmation prompt")
	syncFormat := syncCommand.String("format", "list", "Preview format: list or timesheet")
	syncHideEmpty := syncCommand.Bool("hide-empty", false, "Hide days with no time entries (timesheet format only)")
	syncDecimal := syncCommand.Bool("decimal", false, "Display hours in decimal format (timesheet format only)")
	syncRounding := syncCommand.String("rounding", "", "Rounding steps: floor/ceil/round:to[,...] (default from config, or round:1m)")
	configureFlagSet(syncCommand, "Sync time entries to Tempo Cloud. Entries are merged by task and date before sending.", "  worklog jira sync --dry-run\n  worklog jira sync --task <short-uuid>\n  worklog jira sync --from 2026-05-01 --to 2026-05-07\n  worklog jira sync --dry-run --format timesheet --hide-empty")
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
		if cfg.Jira.Username == "" {
			return fmt.Errorf("jira.username not configured")
		}
		jiraClient = jira.NewClient(cfg.Jira.BaseURL, cfg.Jira.Username, jiraToken)
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

	var roundingSteps []RoundingStep
	if *syncRounding != "" {
		roundingSteps, err = parseRoundingSteps(*syncRounding)
		if err != nil {
			return fmt.Errorf("--rounding: %w", err)
		}
	} else if len(cfg.Tempo.Rounding) > 0 {
		roundingSteps = cfg.Tempo.Rounding
	} else {
		roundingSteps = []RoundingStep{{Step: "round", To: "1m"}}
	}

	syncTaskUUID := ""
	if *syncTask != "" {
		task, err := resolveTaskUUID(worklog, *syncTask)
		if err != nil {
			return err
		}
		syncTaskUUID = task.uuid
	}

	entries, err := buildSyncEntries(worklog, syncTaskUUID, fromDay, toDay, roundingSteps, cfg)
	if err != nil {
		return err
	}

	skipped := findSkippedTasks(worklog, syncTaskUUID, fromDay, toDay)
	allTaskUUIDs := worklog.allTaskUUIDs()
	shortTaskUUIDs := shortUUIDs(allTaskUUIDs)

	if len(entries) == 0 {
		fmt.Println("No entries to sync.")
		if len(skipped) > 0 {
			fmt.Println("")
			fmt.Println("Skipped tasks (no issue key):")
			printSkippedTasks(os.Stdout, skipped, shortTaskUUIDs)
		}
		return nil
	}

	if *syncFormat == "timesheet" {
		renderTimesheetPreview(os.Stdout, entries, fromDay, toDay, cfg, shortTaskUUIDs, *syncHideEmpty, *syncDecimal)
	} else {
		renderListPreview(os.Stdout, entries, cfg, shortTaskUUIDs)
	}

	renderPreviewFooter(os.Stdout, entries, skipped, cfg, shortTaskUUIDs)

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

	if err := sendWorklogs(entries, client, jiraClient, cfg); err != nil {
		return err
	}

	if err := worklog.Save(*syncOutput); err != nil {
		return fmt.Errorf("save worklog: %w", err)
	}

	fmt.Printf("Successfully synced %d worklog(s).\n", len(entries))
	return nil
}

func runJiraAttributes(args []string) error {
	attrCommand := flag.NewFlagSet("jira attributes", flag.ContinueOnError)
	configureFlagSet(attrCommand, "List available Tempo work attributes.", "  worklog jira attributes")
	if err := attrCommand.Parse(args); err != nil {
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
	baseURL := cfg.Tempo.BaseURL
	if baseURL == "" {
		baseURL = "https://api.tempo.io/4"
	}

	client := tempo.NewClient(baseURL, token, cfg.Tempo.AccountID)
	attrs, err := client.GetWorkAttributes()
	if err != nil {
		return fmt.Errorf("fetch work attributes: %w", err)
	}

	if len(attrs) == 0 {
		fmt.Println("No work attributes found.")
		return nil
	}

	for _, a := range attrs {
		header := fmt.Sprintf("%s (%s)", a.Name, a.Key)
		if a.Required {
			header += " [required]"
		}
		fmt.Println(header)
		for _, v := range a.Values {
			label := v
			if a.Names != nil {
				if name, ok := a.Names[v]; ok && name != "" {
					label = name
				}
			}
			if label != v {
				fmt.Printf("  %s (%s)\n", label, v)
			} else {
				fmt.Printf("  %s\n", v)
			}
		}
		fmt.Println()
	}
	return nil
}

type skippedTask struct {
	task     *Task
	duration int
}

func findSkippedTasks(worklog *Worklog, taskUUID string, fromDay, toDay time.Time) []skippedTask {
	durations := make(map[string]int)
	taskMap := make(map[string]*Task)

	for _, event := range worklog.GetEvents() {
		if !eventInDateRange(event, fromDay, toDay) {
			continue
		}
		task := worklog.FindTaskByUUID(event.relatedTo)
		if task == nil {
			continue
		}
		if taskUUID != "" && task.uuid != taskUUID {
			continue
		}
		if task.IssueKey() != "" {
			continue
		}
		durations[task.uuid] += event.duration
		taskMap[task.uuid] = task
	}

	var result []skippedTask
	for uuid, dur := range durations {
		result = append(result, skippedTask{task: taskMap[uuid], duration: dur})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].task.name < result[j].task.name
	})
	return result
}

func printSkippedTasks(w io.Writer, skipped []skippedTask, shortUUIDs map[string]string) {
	maxShortLen := 4
	for _, s := range skipped {
		if su := shortUUIDs[s.task.uuid]; len(su) > maxShortLen {
			maxShortLen = len(su)
		}
	}
	nameWidth := 30
	for _, s := range skipped {
		if len(s.task.name) > nameWidth {
			nameWidth = len(s.task.name)
		}
	}
	for _, s := range skipped {
		fmt.Fprintf(w, "  %-*s %-*s %s\n", maxShortLen, shortUUIDs[s.task.uuid], nameWidth, s.task.name, formatDuration(s.duration))
	}
}

func computeSyncHash(entry syncEntry, cfg *Config) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s\n%s\n%s\n%d\n%s\n", entry.issueKey, entry.start.Format("2006-01-02"), entry.start.Format("15:04:05"), entry.duration, entry.comment)
	attrs := mergedTempoAttributes(cfg, entry.task)
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(h, "%s=%s\n", k, attrs[k])
	}
	return hex.EncodeToString(h.Sum(nil))
}

func buildSyncEntries(worklog *Worklog, taskUUID string, fromDay, toDay time.Time, roundingSteps []RoundingStep, cfg *Config) ([]syncEntry, error) {
	type groupKey struct {
		taskUUID string
		day      string
		issueKey string
	}
	groups := make(map[groupKey]*syncEntry)

	events := worklog.GetEvents()
	for _, event := range events {
		if !eventInDateRange(event, fromDay, toDay) {
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

		day := time.Date(event.dtstart.Year(), event.dtstart.Month(), event.dtstart.Day(), 0, 0, 0, 0, event.dtstart.Location())
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
		rounded, err := applyRounding(roundingSteps, g.duration)
		if err != nil {
			return nil, err
		}
		g.duration = rounded
		if g.duration == 0 {
			continue
		}

		// Use task description as Tempo worklog comment
		g.comment = strings.TrimSpace(g.task.description)

		g.syncHash = computeSyncHash(*g, cfg)

		// Check if any event in the group needs syncing
		needsSync := false
		for _, event := range g.events {
			if event.SyncedAt().IsZero() || event.SyncHash() != g.syncHash {
				needsSync = true
				break
			}
		}
		if !needsSync {
			continue
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

	days := buildDayRange(minDay, maxDay)
	dayIndex := buildDayIndex(days)

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

func applyRounding(steps []RoundingStep, seconds int) (int, error) {
	d := seconds
	for _, step := range steps {
		toSec, err := parseDurationFlag(step.To)
		if err != nil {
			return 0, fmt.Errorf("rounding step %s %s: %w", step.Step, step.To, err)
		}
		if toSec <= 0 {
			return 0, fmt.Errorf("rounding step %s %s: duration must be positive", step.Step, step.To)
		}
		switch step.Step {
		case "floor":
			d = (d / toSec) * toSec
		case "ceil":
			if d%toSec != 0 {
				d = ((d / toSec) + 1) * toSec
			}
		case "round":
			d = ((d + toSec/2) / toSec) * toSec
		default:
			return 0, fmt.Errorf("rounding step %s %s: unknown step %q", step.Step, step.To, step.Step)
		}
	}
	return d, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func humanizeLabel(s string) string {
	s = strings.Trim(s, "_")
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			if len(result) > 0 {
				prev := result[len(result)-1]
				if prev >= 'a' && prev <= 'z' {
					result = append(result, ' ')
				}
			}
		}
		result = append(result, r)
	}
	return string(result)
}

// buildHumanizedAttrKeys converts raw Tempo attribute keys to human-readable
// column labels. If two different raw keys map to the same label, the raw key
// is appended in parentheses to disambiguate.
func buildHumanizedAttrKeys(attrKeys []string) []string {
	seen := make(map[string]bool)
	labels := make([]string, len(attrKeys))
	for i, k := range attrKeys {
		label := humanizeLabel(k)
		if seen[label] {
			label = fmt.Sprintf("%s (%s)", label, k)
		}
		seen[label] = true
		labels[i] = label
	}
	return labels
}

func renderTimesheetPreview(w io.Writer, entries []syncEntry, fromDay, toDay time.Time, cfg *Config, shortTaskUUIDs map[string]string, hideEmpty bool, decimal bool) {
	ts := buildTimesheetFromSyncEntries(entries, fromDay, toDay)
	ts.shortUUIDs = shortTaskUUIDs

	// Build Jira/Tempo-specific maps from entries.
	taskIssueKeys := make(map[string]string)
	taskHasComments := make(map[string][]bool)
	dayIndex := buildDayIndex(ts.days)
	for _, e := range entries {
		dayStr := time.Date(e.start.Year(), e.start.Month(), e.start.Day(), 0, 0, 0, 0, e.start.Location()).Format("2006-01-02")
		idx, ok := dayIndex[dayStr]
		if !ok {
			continue
		}
		if _, ok := taskHasComments[e.task.uuid]; !ok {
			taskHasComments[e.task.uuid] = make([]bool, len(ts.days))
		}
		if taskIssueKeys[e.task.uuid] == "" && e.issueKey != "" {
			taskIssueKeys[e.task.uuid] = e.issueKey
		}
		taskHasComments[e.task.uuid][idx] = taskHasComments[e.task.uuid][idx] || strings.TrimSpace(e.task.description) != ""
	}

	// Determine Tempo attribute columns.
	attrKeysSet := make(map[string]bool)
	for _, row := range ts.rows {
		attrs := mergedTempoAttributes(cfg, row.task)
		for k := range attrs {
			attrKeysSet[k] = true
		}
	}
	var attrKeys []string
	for k := range attrKeysSet {
		attrKeys = append(attrKeys, k)
	}
	sort.Strings(attrKeys)
	attrLabels := buildHumanizedAttrKeys(attrKeys)

	// Set extension headers: Issue Key + attribute columns.
	ts.extraHeaders = append([]string{"Issue Key"}, attrLabels...)

	// Populate per-row extension values and day annotations.
	for i := range ts.rows {
		row := &ts.rows[i]
		attrs := mergedTempoAttributes(cfg, row.task)

		extraValues := make([]string, len(ts.extraHeaders))
		extraValues[0] = taskIssueKeys[row.task.uuid]
		for j, k := range attrKeys {
			val := ""
			if v, ok := attrs[k]; ok {
				if cfg.Tempo.Attributes[k] != v {
					val = humanizeLabel(v)
				}
			}
			extraValues[j+1] = val
		}
		row.extraValues = extraValues

		if hasComments, ok := taskHasComments[row.task.uuid]; ok {
			row.dayAnnotations = make([]string, len(ts.days))
			for d, has := range hasComments {
				if has {
					row.dayAnnotations[d] = "*"
				}
			}
		}
	}

	if hideEmpty {
		ts = hideEmptyColumns(ts)
	}
	printTimesheetTable(w, ts, decimal)
}

func renderListPreview(w io.Writer, entries []syncEntry, cfg *Config, shortTaskUUIDs map[string]string) {
	mergedAttrs := make([]map[string]string, len(entries))
	allAttrKeys := make(map[string]bool)
	for i, e := range entries {
		attrs := mergedTempoAttributes(cfg, e.task)
		mergedAttrs[i] = attrs
		for k := range attrs {
			allAttrKeys[k] = true
		}
	}
	var attrKeys []string
	for k := range allAttrKeys {
		attrKeys = append(attrKeys, k)
	}
	sort.Strings(attrKeys)
	attrLabels := buildHumanizedAttrKeys(attrKeys)

	maxShortLen := 4
	for _, su := range shortTaskUUIDs {
		if len(su) > maxShortLen {
			maxShortLen = len(su)
		}
	}

	colNames := []string{"UUID", "Task", "Issue Key", "Date", "Duration", "Comment"}
	colWidths := []int{maxShortLen, 30, 12, 10, 8, 30}
	for _, label := range attrLabels {
		colNames = append(colNames, label)
		colWidths = append(colWidths, len(label))
	}

	for i, e := range entries {
		if len(shortTaskUUIDs[e.task.uuid]) > colWidths[0] {
			colWidths[0] = len(shortTaskUUIDs[e.task.uuid])
		}
		if len(e.issueKey) > colWidths[2] {
			colWidths[2] = len(e.issueKey)
		}
		durStr := formatDuration(e.duration)
		if len(durStr) > colWidths[4] {
			colWidths[4] = len(durStr)
		}
		for j, k := range attrKeys {
			val := ""
			if v, ok := mergedAttrs[i][k]; ok {
				if cfg.Tempo.Attributes[k] != v {
					val = humanizeLabel(v)
				}
			}
			if len(val) > colWidths[6+j] {
				colWidths[6+j] = len(val)
			}
		}
	}

	for i, name := range colNames {
		if i == 0 {
			fmt.Fprintf(w, "%-*s", colWidths[i], name)
		} else {
			fmt.Fprintf(w, " %-*s", colWidths[i], name)
		}
	}
	fmt.Fprintln(w)

	totalWidth := 0
	for i, w := range colWidths {
		if i > 0 {
			totalWidth++
		}
		totalWidth += w
	}
	fmt.Fprintln(w, strings.Repeat("-", totalWidth))

	for i, e := range entries {
		durStr := formatDuration(e.duration)
		dateStr := e.start.Format("2006-01-02")
		name := truncate(e.task.name, 30)
		comment := truncate(e.comment, 30)
		cells := []string{
			shortTaskUUIDs[e.task.uuid],
			name,
			e.issueKey,
			dateStr,
			durStr,
			comment,
		}
		for _, k := range attrKeys {
			val := ""
			if v, ok := mergedAttrs[i][k]; ok {
				if cfg.Tempo.Attributes[k] != v {
					val = humanizeLabel(v)
				}
			}
			cells = append(cells, val)
		}
		for j, cell := range cells {
			if j == 0 {
				fmt.Fprintf(w, "%-*s", colWidths[j], cell)
			} else {
				fmt.Fprintf(w, " %-*s", colWidths[j], cell)
			}
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, strings.Repeat("-", totalWidth))
}

func renderPreviewFooter(w io.Writer, entries []syncEntry, skipped []skippedTask, cfg *Config, shortTaskUUIDs map[string]string) {
	if len(cfg.Tempo.Attributes) > 0 {
		parts := make([]string, 0, len(cfg.Tempo.Attributes))
		for k, v := range cfg.Tempo.Attributes {
			parts = append(parts, fmt.Sprintf("%s=%s", humanizeLabel(k), humanizeLabel(v)))
		}
		sort.Strings(parts)
		fmt.Fprintf(w, "Default Tempo attributes: %s\n", strings.Join(parts, ", "))
	}
	fmt.Fprintf(w, "Total: %d worklog(s) to send.\n", len(entries))

	if len(skipped) > 0 {
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "Skipped tasks (no issue key):")
		printSkippedTasks(w, skipped, shortTaskUUIDs)
	}
}

func sendWorklogs(entries []syncEntry, client *tempo.Client, jiraClient *jira.Client, cfg *Config) error {
	// Pre-fetch remaining estimates from Jira so Tempo doesn't require us to
	// auto-reduce them. We only fetch for unique issue keys to limit API calls.
	remainingEstimates := make(map[string]int)
	if jiraClient != nil {
		uniqueKeys := make(map[string]bool)
		for _, e := range entries {
			uniqueKeys[e.issueKey] = true
		}
		for key := range uniqueKeys {
			seconds, err := jiraClient.GetRemainingEstimate(key)
			if err != nil {
				return fmt.Errorf("fetch remaining estimate for %s: %w", key, err)
			}
			remainingEstimates[key] = seconds
		}
	}

	for _, e := range entries {
		issueID := e.task.IssueID()
		if issueID == "" {
			if jiraClient == nil {
				return fmt.Errorf("jira.base_url, jira.username, and jira.token required to resolve issue key %s", e.issueKey)
			}
			id, err := jiraClient.GetIssueID(e.issueKey)
			if err != nil {
				return fmt.Errorf("resolve issue key %s: %w", e.issueKey, err)
			}
			e.task.SetIssueID(id)
			issueID = id
		}
		attrs := mergedTempoAttributes(cfg, e.task)
		wl := tempo.Worklog{
			IssueId:          issueID,
			TimeSpentSeconds: e.duration,
			StartDate:        e.start.Format("2006-01-02"),
			StartTime:        e.start.Format("15:04:05"),
			Description:      e.comment,
			Attributes:       attrs,
		}
		if est, ok := remainingEstimates[e.issueKey]; ok {
			wl.RemainingEstimateSeconds = &est
		}
		if err := client.CreateWorklog(wl); err != nil {
			return fmt.Errorf("send worklog for '%s' (%s): %w", e.task.name, e.issueKey, err)
		}
		now := time.Now()
		for _, event := range e.events {
			event.SetSyncedAt(now)
			event.SetSyncHash(e.syncHash)
		}
	}
	return nil
}

func mergedTempoAttributes(cfg *Config, task *Task) map[string]string {
	attrs := make(map[string]string)
	if cfg != nil {
		for k, v := range cfg.Tempo.Attributes {
			attrs[k] = v
		}
	}
	for k, v := range task.TempoAttributes() {
		attrs[k] = v
	}
	return attrs
}

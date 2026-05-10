package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type Timesheet struct {
	days        []time.Time
	rows        []TimesheetRow
	totals      []int // daily totals in seconds
	shortUUIDs  map[string]string
	defaultAttrs map[string]string // Tempo attribute defaults (sync preview only)
}

type TimesheetRow struct {
	task        *Task
	durations   []int // per day, in seconds
	total       int   // row total in seconds
	attributes  map[string]string // merged Tempo attributes (sync preview only)
	issueKey    string            // sync preview only
	hasComments []bool            // sync preview only, per day
}

func eventDateRange(worklog *Worklog) (time.Time, time.Time) {
	var min, max time.Time
	for _, event := range worklog.events {
		if event.dtstart.IsZero() {
			continue
		}
		day := time.Date(event.dtstart.Year(), event.dtstart.Month(), event.dtstart.Day(), 0, 0, 0, 0, event.dtstart.Location())
		if min.IsZero() || day.Before(min) {
			min = day
		}
		if max.IsZero() || day.After(max) {
			max = day
		}
	}
	return min, max
}

func hideEmptyColumns(ts *Timesheet) *Timesheet {
	var keptIndices []int
	for i, total := range ts.totals {
		if total > 0 {
			keptIndices = append(keptIndices, i)
		}
	}

	if len(keptIndices) == len(ts.days) {
		return ts
	}

	newDays := make([]time.Time, len(keptIndices))
	newTotals := make([]int, len(keptIndices))
	for i, idx := range keptIndices {
		newDays[i] = ts.days[idx]
		newTotals[i] = ts.totals[idx]
	}

	newRows := make([]TimesheetRow, len(ts.rows))
	for i, row := range ts.rows {
		newDurations := make([]int, len(keptIndices))
		newTotal := 0
		for j, idx := range keptIndices {
			newDurations[j] = row.durations[idx]
			newTotal += row.durations[idx]
		}
		newRows[i] = TimesheetRow{
			task:       row.task,
			durations:  newDurations,
			total:      newTotal,
			attributes: row.attributes,
		}
	}

	return &Timesheet{
		days:         newDays,
		rows:         newRows,
		totals:       newTotals,
		shortUUIDs:   ts.shortUUIDs,
		defaultAttrs: ts.defaultAttrs,
	}
}

func currentWeekRange(now time.Time) (time.Time, time.Time) {
	wd := int(now.Weekday())
	if wd == 0 { // Sunday
		wd = 7
	}
	monday := now.AddDate(0, 0, -wd+1)
	from := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, now.Location())
	to := from.AddDate(0, 0, 6)
	to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, now.Location())
	return from, to
}

func parseDateFlag(value string) (time.Time, error) {
	layout := "2006-01-02"
	t, err := time.ParseInLocation(layout, value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date format: %s (use YYYY-MM-DD)", value)
	}
	return t, nil
}

func buildDayRange(from, to time.Time) []time.Time {
	var days []time.Time
	day := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	endDay := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, to.Location())
	for !day.After(endDay) {
		days = append(days, day)
		day = day.AddDate(0, 0, 1)
	}
	return days
}

func buildDayIndex(days []time.Time) map[string]int {
	dayIndex := make(map[string]int)
	for i, d := range days {
		dayIndex[d.Format("2006-01-02")] = i
	}
	return dayIndex
}

func generateTimesheet(worklog *Worklog, from, to time.Time) *Timesheet {
	days := buildDayRange(from, to)

	// Collect events per task per day
	type taskDayKey struct {
		taskUUID string
		day      string
	}
	cellDurations := make(map[taskDayKey]int)
	taskHasTime := make(map[string]bool)

	for _, event := range worklog.events {
		if event.dtstart.IsZero() {
			continue
		}
		eventDay := time.Date(event.dtstart.Year(), event.dtstart.Month(), event.dtstart.Day(), 0, 0, 0, 0, event.dtstart.Location())
		if eventDay.Before(days[0]) || eventDay.After(days[len(days)-1]) {
			continue
		}
		dayStr := eventDay.Format("2006-01-02")
		key := taskDayKey{taskUUID: event.relatedTo, day: dayStr}
		cellDurations[key] += event.duration
		taskHasTime[event.relatedTo] = true
	}

	// Build rows: only tasks with direct time in range
	allTasks := flattenTasks(worklog.tasks)
	var rows []TimesheetRow
	for _, task := range allTasks {
		if !taskHasTime[task.uuid] {
			continue
		}
		durations := make([]int, len(days))
		rowTotal := 0
		for i, d := range days {
			key := taskDayKey{taskUUID: task.uuid, day: d.Format("2006-01-02")}
			if dur, ok := cellDurations[key]; ok {
				durations[i] = dur
				rowTotal += dur
			}
		}
		rows = append(rows, TimesheetRow{
			task:      task,
			durations: durations,
			total:     rowTotal,
		})
	}

	// Sort rows alphabetically by task name
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].task.name < rows[j].task.name
	})

	// Compute daily totals
	totals := make([]int, len(days))
	for i := range days {
		for _, row := range rows {
			totals[i] += row.durations[i]
		}
	}

	ts := &Timesheet{
		days:   days,
		rows:   rows,
		totals: totals,
	}
	ts.shortUUIDs = shortUUIDs(worklog.allTaskUUIDs())
	return ts
}

func formatDurationDecimal(seconds int) string {
	if seconds <= 0 {
		return ""
	}
	hours := float64(seconds) / 3600.0
	return fmt.Sprintf("%.2f", hours)
}

func formatCell(seconds int, decimal bool) string {
	if seconds <= 0 {
		return ""
	}
	if decimal {
		return formatDurationDecimal(seconds)
	}
	return formatDuration(seconds)
}

func printTimesheetTable(w io.Writer, ts *Timesheet, decimal bool) {
	if len(ts.rows) == 0 {
		fmt.Fprintln(w, "No time entries in the selected date range.")
		return
	}

	// Determine attribute columns (sync preview only)
	attrKeysSet := make(map[string]bool)
	for _, row := range ts.rows {
		for k := range row.attributes {
			attrKeysSet[k] = true
		}
	}
	var attrKeys []string
	for k := range attrKeysSet {
		attrKeys = append(attrKeys, k)
	}
	sort.Strings(attrKeys)
	attrLabels := buildHumanizedAttrKeys(attrKeys)
	attrColCount := len(attrKeys)

	isSyncPreview := ts.defaultAttrs != nil

	numBaseCols := len(ts.days) + 3 // uuid + task name + days + total
	if isSyncPreview {
		numBaseCols++ // + issue key
	}
	numCols := numBaseCols + attrColCount

	// Build header
	header := make([]string, numCols)
	header[0] = "UUID"
	header[1] = "Task"
	for i, d := range ts.days {
		header[i+2] = d.Format("01/02")
	}
	header[2+len(ts.days)] = "Total"
	if isSyncPreview {
		header[3+len(ts.days)] = "Issue Key"
	}
	for i, label := range attrLabels {
		header[numBaseCols+i] = label
	}

	// Compute column widths
	colWidths := make([]int, numCols)
	for i, h := range header {
		colWidths[i] = len(h)
	}
	for _, row := range ts.rows {
		uuid := ""
		if ts.shortUUIDs != nil {
			uuid = ts.shortUUIDs[row.task.uuid]
		}
		if len(uuid) > colWidths[0] {
			colWidths[0] = len(uuid)
		}
		if len(row.task.name) > colWidths[1] {
			colWidths[1] = len(row.task.name)
		}
		for i, dur := range row.durations {
			cell := formatCell(dur, decimal)
			if isSyncPreview && len(row.hasComments) > i && row.hasComments[i] {
				cell += "*"
			}
			if len(cell) > colWidths[i+2] {
				colWidths[i+2] = len(cell)
			}
		}
		cell := formatCell(row.total, decimal)
		if len(cell) > colWidths[2+len(ts.days)] {
			colWidths[2+len(ts.days)] = len(cell)
		}
		if isSyncPreview {
			if len(row.issueKey) > colWidths[3+len(ts.days)] {
				colWidths[3+len(ts.days)] = len(row.issueKey)
			}
		}
		for i, k := range attrKeys {
			val := ""
			if v, ok := row.attributes[k]; ok {
				if ts.defaultAttrs == nil || ts.defaultAttrs[k] != v {
					val = humanizeLabel(v)
				}
			}
			if len(val) > colWidths[numBaseCols+i] {
				colWidths[numBaseCols+i] = len(val)
			}
		}
	}
	// Consider totals row for widths (base columns only)
	for i, tot := range ts.totals {
		cell := formatCell(tot, decimal)
		if len(cell) > colWidths[i+2] {
			colWidths[i+2] = len(cell)
		}
	}
	cell := formatCell(sum(ts.totals), decimal)
	if len(cell) > colWidths[2+len(ts.days)] {
		colWidths[2+len(ts.days)] = len(cell)
	}

	// Helper to print a row
	printRow := func(cells []string) {
		for i, cell := range cells {
			if i == 0 {
				fmt.Fprintf(w, "%-*s", colWidths[i], cell)
			} else {
				fmt.Fprintf(w, " %*s", colWidths[i], cell)
			}
		}
		fmt.Fprintln(w)
	}

	// Header separator
	sep := make([]string, numCols)
	for i, w := range colWidths {
		sep[i] = strings.Repeat("-", w)
	}

	printRow(header)
	printRow(sep)

	for _, row := range ts.rows {
		cells := make([]string, numCols)
		if ts.shortUUIDs != nil {
			cells[0] = ts.shortUUIDs[row.task.uuid]
		}
		cells[1] = row.task.name
		for i, dur := range row.durations {
			cell := formatCell(dur, decimal)
			if isSyncPreview && len(row.hasComments) > i && row.hasComments[i] {
				cell += "*"
			}
			cells[i+2] = cell
		}
		cells[2+len(ts.days)] = formatCell(row.total, decimal)
		if isSyncPreview {
			cells[3+len(ts.days)] = row.issueKey
		}
		for i, k := range attrKeys {
			val := ""
			if v, ok := row.attributes[k]; ok {
				if ts.defaultAttrs == nil || ts.defaultAttrs[k] != v {
					val = humanizeLabel(v)
				}
			}
			cells[numBaseCols+i] = val
		}
		printRow(cells)
	}

	printRow(sep)

	// Totals row
	totalCells := make([]string, numCols)
	totalCells[0] = ""
	totalCells[1] = "Total"
	for i, tot := range ts.totals {
		totalCells[i+2] = formatCell(tot, decimal)
	}
	totalCells[2+len(ts.days)] = formatCell(sum(ts.totals), decimal)
	printRow(totalCells)
}

func printTimesheetCSV(w io.Writer, ts *Timesheet, decimal bool) {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header
	header := []string{"UUID", "Task"}
	for _, d := range ts.days {
		header = append(header, d.Format("2006-01-02"))
	}
	header = append(header, "Total")
	writer.Write(header)

	// Rows
	for _, row := range ts.rows {
		record := []string{}
		if ts.shortUUIDs != nil {
			record = append(record, ts.shortUUIDs[row.task.uuid])
		} else {
			record = append(record, "")
		}
		record = append(record, row.task.name)
		for _, dur := range row.durations {
			record = append(record, formatCell(dur, decimal))
		}
		record = append(record, formatCell(row.total, decimal))
		writer.Write(record)
	}

	// Totals row
	record := []string{"", "Total"}
	for _, tot := range ts.totals {
		record = append(record, formatCell(tot, decimal))
	}
	record = append(record, formatCell(sum(ts.totals), decimal))
	writer.Write(record)
}

func sum(values []int) int {
	s := 0
	for _, v := range values {
		s += v
	}
	return s
}

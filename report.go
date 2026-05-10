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
	days         []time.Time
	rows         []TimesheetRow
	totals       []int // daily totals in seconds
	shortUUIDs   map[string]string
	extraHeaders []string // extension column headers
}

type TimesheetRow struct {
	task           *Task
	durations      []int    // per day, in seconds
	total          int      // row total in seconds
	extraValues    []string // extension column values, aligned to extraHeaders
	dayAnnotations []string // per-day suffix, e.g. "*" for comment indicator
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
		newDayAnnotations := make([]string, len(keptIndices))
		newTotal := 0
		for j, idx := range keptIndices {
			newDurations[j] = row.durations[idx]
			if len(row.dayAnnotations) > idx {
				newDayAnnotations[j] = row.dayAnnotations[idx]
			}
			newTotal += row.durations[idx]
		}
		newRows[i] = TimesheetRow{
			task:           row.task,
			durations:      newDurations,
			total:          newTotal,
			extraValues:    row.extraValues,
			dayAnnotations: newDayAnnotations,
		}
	}

	return &Timesheet{
		days:         newDays,
		rows:         newRows,
		totals:       newTotals,
		shortUUIDs:   ts.shortUUIDs,
		extraHeaders: ts.extraHeaders,
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

func buildHeader(ts *Timesheet) []string {
	extraCount := len(ts.extraHeaders)
	numBaseCols := len(ts.days) + 3 // uuid + task name + days + total
	numCols := numBaseCols + extraCount

	header := make([]string, numCols)
	header[0] = "UUID"
	header[1] = "Task"
	for i, d := range ts.days {
		header[i+2] = d.Format("01/02")
	}
	header[2+len(ts.days)] = "Total"
	for i, h := range ts.extraHeaders {
		header[numBaseCols+i] = h
	}
	return header
}

func buildDataRowCells(ts *Timesheet, row TimesheetRow, decimal bool) []string {
	extraCount := len(ts.extraHeaders)
	numBaseCols := len(ts.days) + 3
	numCols := numBaseCols + extraCount

	cells := make([]string, numCols)
	if ts.shortUUIDs != nil {
		cells[0] = ts.shortUUIDs[row.task.uuid]
	}
	cells[1] = row.task.name
	for i, dur := range row.durations {
		cell := formatCell(dur, decimal)
		if len(row.dayAnnotations) > i && row.dayAnnotations[i] != "" {
			cell += row.dayAnnotations[i]
		}
		cells[i+2] = cell
	}
	cells[2+len(ts.days)] = formatCell(row.total, decimal)
	for i, val := range row.extraValues {
		cells[numBaseCols+i] = val
	}
	return cells
}

func buildTotalsRowCells(ts *Timesheet, decimal bool) []string {
	numBaseCols := len(ts.days) + 3
	numCols := numBaseCols + len(ts.extraHeaders)

	cells := make([]string, numCols)
	cells[0] = ""
	cells[1] = "Total"
	for i, tot := range ts.totals {
		cells[i+2] = formatCell(tot, decimal)
	}
	cells[2+len(ts.days)] = formatCell(sum(ts.totals), decimal)
	return cells
}

type tableFormatter struct {
	w         io.Writer
	colWidths []int
}

func newTableFormatter(w io.Writer, rows [][]string) *tableFormatter {
	if len(rows) == 0 {
		return &tableFormatter{w: w}
	}
	numCols := len(rows[0])
	colWidths := make([]int, numCols)
	for _, row := range rows {
		for i, cell := range row {
			if i >= numCols {
				break
			}
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}
	return &tableFormatter{w: w, colWidths: colWidths}
}

func (tf *tableFormatter) printRow(cells []string) {
	for i, cell := range cells {
		if i == 0 {
			fmt.Fprintf(tf.w, "%-*s", tf.colWidths[i], cell)
		} else {
			fmt.Fprintf(tf.w, " %*s", tf.colWidths[i], cell)
		}
	}
	fmt.Fprintln(tf.w)
}

func (tf *tableFormatter) printSeparator() {
	sep := make([]string, len(tf.colWidths))
	for i, w := range tf.colWidths {
		sep[i] = strings.Repeat("-", w)
	}
	tf.printRow(sep)
}

func printTimesheetTable(w io.Writer, ts *Timesheet, decimal bool) {
	if len(ts.rows) == 0 {
		fmt.Fprintln(w, "No time entries in the selected date range.")
		return
	}

	header := buildHeader(ts)

	dataRows := make([][]string, len(ts.rows))
	for i, row := range ts.rows {
		dataRows[i] = buildDataRowCells(ts, row, decimal)
	}

	totalsRow := buildTotalsRowCells(ts, decimal)

	allRows := make([][]string, 0, 2+len(ts.rows))
	allRows = append(allRows, header)
	allRows = append(allRows, dataRows...)
	allRows = append(allRows, totalsRow)

	tf := newTableFormatter(w, allRows)

	tf.printRow(header)
	tf.printSeparator()
	for _, cells := range dataRows {
		tf.printRow(cells)
	}
	tf.printSeparator()
	tf.printRow(totalsRow)
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

package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestCurrentWeekRange(t *testing.T) {
	// Test a Wednesday
	wed := time.Date(2023, 8, 16, 12, 0, 0, 0, time.UTC)
	from, to := currentWeekRange(wed)

	expectedFrom := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC) // Monday
	expectedTo := time.Date(2023, 8, 20, 23, 59, 59, 0, time.UTC) // Sunday

	if !from.Equal(expectedFrom) {
		t.Errorf("Expected from %v, got %v", expectedFrom, from)
	}
	if !to.Equal(expectedTo) {
		t.Errorf("Expected to %v, got %v", expectedTo, to)
	}
}

func TestCurrentWeekRangeSunday(t *testing.T) {
	// Test a Sunday
	sun := time.Date(2023, 8, 20, 12, 0, 0, 0, time.UTC)
	from, to := currentWeekRange(sun)

	expectedFrom := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC) // Monday
	expectedTo := time.Date(2023, 8, 20, 23, 59, 59, 0, time.UTC) // Sunday

	if !from.Equal(expectedFrom) {
		t.Errorf("Expected from %v, got %v", expectedFrom, from)
	}
	if !to.Equal(expectedTo) {
		t.Errorf("Expected to %v, got %v", expectedTo, to)
	}
}

func TestParseDateFlag(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
		wantErr  bool
	}{
		{"2023-08-14", time.Date(2023, 8, 14, 0, 0, 0, 0, time.Local), false},
		{"2023-12-31", time.Date(2023, 12, 31, 0, 0, 0, 0, time.Local), false},
		{"08-14-2023", time.Time{}, true},
		{"", time.Time{}, true},
	}

	for _, tt := range tests {
		result, err := parseDateFlag(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseDateFlag(%q) expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseDateFlag(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if !result.Equal(tt.expected) {
			t.Errorf("parseDateFlag(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestGenerateTimesheet(t *testing.T) {
	worklog := createTestWorklog()

	taskA := NewTask("Task A")
	taskB := NewTask("Task B")
	taskC := NewTask("Task C") // no events
	worklog.tasks = append(worklog.tasks, taskA, taskB, taskC)

	day1 := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2023, 8, 15, 0, 0, 0, 0, time.UTC)

	// Task A: 1h on day1, 30m on day2
	worklog.events = append(worklog.events, &Event{
		uuid:      "ev-a1",
		relatedTo: taskA.uuid,
		dtstart:   day1.Add(9 * time.Hour),
		dtend:     day1.Add(10 * time.Hour),
		duration:  3600,
	})
	worklog.events = append(worklog.events, &Event{
		uuid:      "ev-a2",
		relatedTo: taskA.uuid,
		dtstart:   day2.Add(9 * time.Hour),
		dtend:     day2.Add(9*time.Hour + 30*time.Minute),
		duration:  1800,
	})

	// Task B: 2h on day1
	worklog.events = append(worklog.events, &Event{
		uuid:      "ev-b1",
		relatedTo: taskB.uuid,
		dtstart:   day1.Add(14 * time.Hour),
		dtend:     day1.Add(16 * time.Hour),
		duration:  7200,
	})

	from := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 8, 15, 0, 0, 0, 0, time.UTC)

	ts := generateTimesheet(worklog, from, to)

	if len(ts.days) != 2 {
		t.Errorf("Expected 2 days, got %d", len(ts.days))
	}

	// Should only include tasks A and B (alphabetical order)
	if len(ts.rows) != 2 {
		t.Fatalf("Expected 2 rows, got %d", len(ts.rows))
	}

	if ts.rows[0].task.name != "Task A" {
		t.Errorf("Expected first row Task A, got %s", ts.rows[0].task.name)
	}
	if ts.rows[1].task.name != "Task B" {
		t.Errorf("Expected second row Task B, got %s", ts.rows[1].task.name)
	}

	// Task A durations: [3600, 1800]
	if ts.rows[0].durations[0] != 3600 {
		t.Errorf("Task A day1 expected 3600, got %d", ts.rows[0].durations[0])
	}
	if ts.rows[0].durations[1] != 1800 {
		t.Errorf("Task A day2 expected 1800, got %d", ts.rows[0].durations[1])
	}
	if ts.rows[0].total != 5400 {
		t.Errorf("Task A total expected 5400, got %d", ts.rows[0].total)
	}

	// Task B durations: [7200, 0]
	if ts.rows[1].durations[0] != 7200 {
		t.Errorf("Task B day1 expected 7200, got %d", ts.rows[1].durations[0])
	}
	if ts.rows[1].durations[1] != 0 {
		t.Errorf("Task B day2 expected 0, got %d", ts.rows[1].durations[1])
	}
	if ts.rows[1].total != 7200 {
		t.Errorf("Task B total expected 7200, got %d", ts.rows[1].total)
	}

	// Daily totals: [10800, 1800]
	if ts.totals[0] != 10800 {
		t.Errorf("Day1 total expected 10800, got %d", ts.totals[0])
	}
	if ts.totals[1] != 1800 {
		t.Errorf("Day2 total expected 1800, got %d", ts.totals[1])
	}
}

func TestGenerateTimesheetEmptyRange(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("Task")
	worklog.tasks = append(worklog.tasks, task)

	from := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 8, 15, 0, 0, 0, 0, time.UTC)

	ts := generateTimesheet(worklog, from, to)

	if len(ts.rows) != 0 {
		t.Errorf("Expected 0 rows, got %d", len(ts.rows))
	}
	if len(ts.totals) != 2 {
		t.Errorf("Expected 2 totals entries, got %d", len(ts.totals))
	}
	if ts.totals[0] != 0 || ts.totals[1] != 0 {
		t.Errorf("Expected zero totals, got %v", ts.totals)
	}
}

func TestGenerateTimesheetEventOutsideRange(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("Task")
	worklog.tasks = append(worklog.tasks, task)

	worklog.events = append(worklog.events, &Event{
		uuid:      "ev-1",
		relatedTo: task.uuid,
		dtstart:   time.Date(2023, 8, 10, 9, 0, 0, 0, time.UTC),
		duration:  3600,
	})

	from := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 8, 15, 0, 0, 0, 0, time.UTC)

	ts := generateTimesheet(worklog, from, to)

	if len(ts.rows) != 0 {
		t.Errorf("Expected 0 rows for out-of-range event, got %d", len(ts.rows))
	}
}

func TestGenerateTimesheetMultipleEventsSameDay(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("Task")
	worklog.tasks = append(worklog.tasks, task)

	day1 := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)

	// Two events on same day for same task
	worklog.events = append(worklog.events, &Event{
		uuid:      "ev-1",
		relatedTo: task.uuid,
		dtstart:   day1.Add(9 * time.Hour),
		duration:  3600,
	})
	worklog.events = append(worklog.events, &Event{
		uuid:      "ev-2",
		relatedTo: task.uuid,
		dtstart:   day1.Add(14 * time.Hour),
		duration:  1800,
	})

	from := day1
	to := day1

	ts := generateTimesheet(worklog, from, to)

	if len(ts.rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(ts.rows))
	}
	if ts.rows[0].durations[0] != 5400 {
		t.Errorf("Expected combined duration 5400, got %d", ts.rows[0].durations[0])
	}
	if ts.totals[0] != 5400 {
		t.Errorf("Expected daily total 5400, got %d", ts.totals[0])
	}
}

func TestGenerateTimesheetEventWithZeroDtstart(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("Task")
	worklog.tasks = append(worklog.tasks, task)

	worklog.events = append(worklog.events, &Event{
		uuid:      "ev-1",
		relatedTo: task.uuid,
		dtstart:   time.Time{}, // zero
		duration:  3600,
	})

	from := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)
	to := time.Date(2023, 8, 15, 0, 0, 0, 0, time.UTC)

	ts := generateTimesheet(worklog, from, to)

	if len(ts.rows) != 0 {
		t.Errorf("Expected 0 rows for zero-dtstart event, got %d", len(ts.rows))
	}
}

func TestFormatDurationDecimal(t *testing.T) {
	tests := []struct {
		seconds  int
		expected string
	}{
		{0, ""},
		{-1, ""},
		{3600, "1.00"},
		{5400, "1.50"},
		{900, "0.25"},
		{3660, "1.02"},
	}

	for _, tt := range tests {
		result := formatDurationDecimal(tt.seconds)
		if result != tt.expected {
			t.Errorf("formatDurationDecimal(%d) = %q, want %q", tt.seconds, result, tt.expected)
		}
	}
}

func TestFormatCell(t *testing.T) {
	if formatCell(0, false) != "" {
		t.Errorf("formatCell(0, false) expected empty string")
	}
	if formatCell(3600, false) != "1h" {
		t.Errorf("formatCell(3600, false) = %q, want '1h'", formatCell(3600, false))
	}
	if formatCell(3600, true) != "1.00" {
		t.Errorf("formatCell(3600, true) = %q, want '1.00'", formatCell(3600, true))
	}
}

func TestPrintTimesheetTable(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("Test Task")
	worklog.tasks = append(worklog.tasks, task)

	day := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)
	worklog.events = append(worklog.events, &Event{
		uuid:      "ev-1",
		relatedTo: task.uuid,
		dtstart:   day.Add(9 * time.Hour),
		duration:  3600,
	})

	ts := generateTimesheet(worklog, day, day)

	var buf bytes.Buffer
	printTimesheetTable(&buf, ts, false)
	output := buf.String()

	if !strings.Contains(output, "Test Task") {
		t.Errorf("Expected output to contain task name")
	}
	if !strings.Contains(output, "1h") {
		t.Errorf("Expected output to contain '1h'")
	}
	if !strings.Contains(output, "Total") {
		t.Errorf("Expected output to contain 'Total'")
	}
}

func TestPrintTimesheetTableEmpty(t *testing.T) {
	worklog := createTestWorklog()
	day := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)
	ts := generateTimesheet(worklog, day, day)

	var buf bytes.Buffer
	printTimesheetTable(&buf, ts, false)
	output := buf.String()

	if !strings.Contains(output, "No time entries") {
		t.Errorf("Expected empty message, got: %s", output)
	}
}

func TestPrintTimesheetCSV(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("Test Task")
	worklog.tasks = append(worklog.tasks, task)

	day := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)
	worklog.events = append(worklog.events, &Event{
		uuid:      "ev-1",
		relatedTo: task.uuid,
		dtstart:   day.Add(9 * time.Hour),
		duration:  3600,
	})

	ts := generateTimesheet(worklog, day, day)

	var buf bytes.Buffer
	printTimesheetCSV(&buf, ts, false)
	output := buf.String()

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 3 { // header + data row + totals row
		t.Fatalf("Expected 3 CSV lines, got %d: %s", len(lines), output)
	}

	if !strings.Contains(lines[0], "Task") {
		t.Errorf("Expected header to contain 'Task'")
	}
	if !strings.Contains(lines[1], "Test Task") {
		t.Errorf("Expected data row to contain 'Test Task'")
	}
	if !strings.Contains(lines[2], "Total") {
		t.Errorf("Expected totals row to contain 'Total'")
	}
}

func TestSum(t *testing.T) {
	if sum([]int{1, 2, 3}) != 6 {
		t.Errorf("sum([1,2,3]) expected 6")
	}
	if sum([]int{}) != 0 {
		t.Errorf("sum([]) expected 0")
	}
}

func TestEventDateRange(t *testing.T) {
	worklog := createTestWorklog()

	task := NewTask("Task")
	worklog.tasks = append(worklog.tasks, task)

	day1 := time.Date(2023, 8, 14, 9, 0, 0, 0, time.UTC)
	day2 := time.Date(2023, 8, 16, 9, 0, 0, 0, time.UTC)
	day3 := time.Date(2023, 8, 20, 9, 0, 0, 0, time.UTC)

	worklog.events = append(worklog.events, &Event{
		uuid: "ev-1", relatedTo: task.uuid, dtstart: day2, duration: 3600,
	})
	worklog.events = append(worklog.events, &Event{
		uuid: "ev-2", relatedTo: task.uuid, dtstart: day1, duration: 3600,
	})
	worklog.events = append(worklog.events, &Event{
		uuid: "ev-3", relatedTo: task.uuid, dtstart: day3, duration: 3600,
	})

	min, max := eventDateRange(worklog)
	expectedMin := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)
	expectedMax := time.Date(2023, 8, 20, 0, 0, 0, 0, time.UTC)

	if !min.Equal(expectedMin) {
		t.Errorf("Expected min %v, got %v", expectedMin, min)
	}
	if !max.Equal(expectedMax) {
		t.Errorf("Expected max %v, got %v", expectedMax, max)
	}
}

func TestEventDateRangeEmpty(t *testing.T) {
	worklog := createTestWorklog()
	min, max := eventDateRange(worklog)
	if !min.IsZero() {
		t.Errorf("Expected zero min for empty worklog, got %v", min)
	}
	if !max.IsZero() {
		t.Errorf("Expected zero max for empty worklog, got %v", max)
	}
}

func TestEventDateRangeIgnoresZeroDtstart(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("Task")
	worklog.tasks = append(worklog.tasks, task)

	worklog.events = append(worklog.events, &Event{
		uuid: "ev-1", relatedTo: task.uuid, dtstart: time.Time{}, duration: 3600,
	})

	min, max := eventDateRange(worklog)
	if !min.IsZero() || !max.IsZero() {
		t.Errorf("Expected zero range when all events have zero dtstart")
	}
}

func TestHideEmptyColumns(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("Task")
	worklog.tasks = append(worklog.tasks, task)

	day1 := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2023, 8, 15, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2023, 8, 16, 0, 0, 0, 0, time.UTC)

	// Only time on day2
	worklog.events = append(worklog.events, &Event{
		uuid: "ev-1", relatedTo: task.uuid,
		dtstart: day2.Add(9 * time.Hour), duration: 3600,
	})

	ts := generateTimesheet(worklog, day1, day3)
	if len(ts.days) != 3 {
		t.Fatalf("Expected 3 days before hiding, got %d", len(ts.days))
	}

	filtered := hideEmptyColumns(ts)
	if len(filtered.days) != 1 {
		t.Fatalf("Expected 1 day after hiding, got %d", len(filtered.days))
	}
	if !filtered.days[0].Equal(day2) {
		t.Errorf("Expected remaining day %v, got %v", day2, filtered.days[0])
	}
	if filtered.totals[0] != 3600 {
		t.Errorf("Expected total 3600, got %d", filtered.totals[0])
	}
	if len(filtered.rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(filtered.rows))
	}
	if filtered.rows[0].durations[0] != 3600 {
		t.Errorf("Expected row duration 3600, got %d", filtered.rows[0].durations[0])
	}
	if filtered.rows[0].total != 3600 {
		t.Errorf("Expected row total 3600, got %d", filtered.rows[0].total)
	}
}

func TestHideEmptyColumnsNoChange(t *testing.T) {
	worklog := createTestWorklog()
	task := NewTask("Task")
	worklog.tasks = append(worklog.tasks, task)

	day1 := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)
	worklog.events = append(worklog.events, &Event{
		uuid: "ev-1", relatedTo: task.uuid,
		dtstart: day1.Add(9 * time.Hour), duration: 3600,
	})

	ts := generateTimesheet(worklog, day1, day1)
	filtered := hideEmptyColumns(ts)

	if len(filtered.days) != 1 {
		t.Fatalf("Expected 1 day, got %d", len(filtered.days))
	}
	if !filtered.days[0].Equal(day1) {
		t.Errorf("Expected day %v, got %v", day1, filtered.days[0])
	}
}

func TestHideEmptyColumnsAllEmpty(t *testing.T) {
	worklog := createTestWorklog()
	day1 := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2023, 8, 15, 0, 0, 0, 0, time.UTC)

	ts := generateTimesheet(worklog, day1, day2)
	filtered := hideEmptyColumns(ts)

	if len(filtered.days) != 0 {
		t.Errorf("Expected 0 days after hiding all empty, got %d", len(filtered.days))
	}
	if len(filtered.rows) != 0 {
		t.Errorf("Expected 0 rows after hiding all empty, got %d", len(filtered.rows))
	}
}

func TestPrintTimesheetTableWithExtraColumns(t *testing.T) {
	task := NewTask("Test Task")
	day := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)

	ts := &Timesheet{
		days: []time.Time{day},
		rows: []TimesheetRow{
			{
				task:        task,
				durations:   []int{3600},
				total:       3600,
				extraValues: []string{"PROJ-123", "Project Management"},
			},
		},
		totals:       []int{3600},
		shortUUIDs:   map[string]string{task.uuid: "abc"},
		extraHeaders: []string{"Issue Key", "Work Type"},
	}

	var buf bytes.Buffer
	printTimesheetTable(&buf, ts, false)
	output := buf.String()

	if !strings.Contains(output, "Issue Key") {
		t.Errorf("Expected output to contain header 'Issue Key'")
	}
	if !strings.Contains(output, "Work Type") {
		t.Errorf("Expected output to contain header 'Work Type'")
	}
	if !strings.Contains(output, "PROJ-123") {
		t.Errorf("Expected output to contain extra value 'PROJ-123'")
	}
	if !strings.Contains(output, "Project Management") {
		t.Errorf("Expected output to contain extra value 'Project Management'")
	}
}

func TestHideEmptyColumnsPreservesExtras(t *testing.T) {
	task := NewTask("Test Task")
	day1 := time.Date(2023, 8, 14, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2023, 8, 15, 0, 0, 0, 0, time.UTC)

	ts := &Timesheet{
		days: []time.Time{day1, day2},
		rows: []TimesheetRow{
			{
				task:           task,
				durations:      []int{3600, 0},
				total:          3600,
				extraValues:    []string{"Review"},
				dayAnnotations: []string{"*", ""},
			},
		},
		totals:       []int{3600, 0},
		shortUUIDs:   map[string]string{task.uuid: "abc"},
		extraHeaders: []string{"Status"},
	}

	filtered := hideEmptyColumns(ts)
	if len(filtered.rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(filtered.rows))
	}
	if filtered.rows[0].extraValues[0] != "Review" {
		t.Errorf("Expected extra value to be preserved, got %q", filtered.rows[0].extraValues[0])
	}
	if len(filtered.rows[0].dayAnnotations) != 1 || filtered.rows[0].dayAnnotations[0] != "*" {
		t.Errorf("Expected day annotation to be preserved and aligned, got %v", filtered.rows[0].dayAnnotations)
	}
	if filtered.extraHeaders[0] != "Status" {
		t.Errorf("Expected extraHeaders to be preserved, got %v", filtered.extraHeaders)
	}
}

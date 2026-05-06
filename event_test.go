package main

import (
	"testing"
	"time"

	ics "github.com/arran4/golang-ical"
)

func TestNewEvent(t *testing.T) {
	start := time.Date(2023, 8, 14, 9, 0, 0, 0, time.Local)
	end := time.Date(2023, 8, 14, 10, 0, 0, 0, time.Local)
	event := NewEvent("task-uuid", start, end, 3600, "Worked on bug fix", "bug fix comment")

	if event.uuid == "" {
		t.Error("Expected UUID to be generated")
	}
	if event.relatedTo != "task-uuid" {
		t.Errorf("Expected relatedTo 'task-uuid', got '%s'", event.relatedTo)
	}
	if !event.dtstart.Equal(start) {
		t.Errorf("Expected dtstart %v, got %v", start, event.dtstart)
	}
	if !event.dtend.Equal(end) {
		t.Errorf("Expected dtend %v, got %v", end, event.dtend)
	}
	if event.duration != 3600 {
		t.Errorf("Expected duration 3600, got %d", event.duration)
	}
	if event.summary != "Worked on bug fix" {
		t.Errorf("Expected summary 'Worked on bug fix', got '%s'", event.summary)
	}
	if event.comment != "bug fix comment" {
		t.Errorf("Expected comment 'bug fix comment', got '%s'", event.comment)
	}

	hasDTSTAMP := false
	hasCREATED := false
	hasLASTMOD := false
	hasCATEGORIES := false
	hasTRANSP := false
	for _, prop := range event.properties {
		switch prop.IANAToken {
		case "DTSTAMP":
			hasDTSTAMP = true
		case "CREATED":
			hasCREATED = true
		case "LAST-MODIFIED":
			hasLASTMOD = true
		case "CATEGORIES":
			if prop.Value == "KTimeTracker" {
				hasCATEGORIES = true
			}
		case "TRANSP":
			if prop.Value == "OPAQUE" {
				hasTRANSP = true
			}
		}
	}
	if !hasDTSTAMP {
		t.Error("Expected DTSTAMP property")
	}
	if !hasCREATED {
		t.Error("Expected CREATED property")
	}
	if !hasLASTMOD {
		t.Error("Expected LAST-MODIFIED property")
	}
	if !hasCATEGORIES {
		t.Error("Expected CATEGORIES:KTimeTracker property")
	}
	if !hasTRANSP {
		t.Error("Expected TRANSP:OPAQUE property")
	}
}

func TestMakeVEventForEvent_NewEvent(t *testing.T) {
	start := time.Date(2023, 8, 14, 9, 0, 0, 0, time.Local)
	end := time.Date(2023, 8, 14, 10, 0, 0, 0, time.Local)
	event := NewEvent("task-uuid", start, end, 3600, "Worked on bug fix", "bug fix comment")

	ve := makeVEventForEvent(event)

	uid := ve.GetProperty(ics.ComponentPropertyUniqueId)
	if uid == nil || uid.Value != event.uuid {
		t.Error("Expected UID to match event UUID")
	}

	summary := ve.GetProperty(ics.ComponentPropertySummary)
	if summary == nil || summary.Value != "Worked on bug fix" {
		t.Error("Expected SUMMARY to be set")
	}

	comment := ve.GetProperty(ics.ComponentProperty(ics.PropertyComment))
	if comment == nil || comment.Value != "bug fix comment" {
		t.Error("Expected COMMENT to be set")
	}

	related := ve.GetProperty("RELATED-TO")
	if related == nil || related.Value != "task-uuid" {
		t.Error("Expected RELATED-TO to be set")
	}

	dtstart := ve.GetProperty(ics.ComponentPropertyDtStart)
	if dtstart == nil {
		t.Error("Expected DTSTART to be set")
	}

	dtend := ve.GetProperty(ics.ComponentPropertyDtEnd)
	if dtend == nil {
		t.Error("Expected DTEND to be set")
	}

	dur := ve.GetProperty("X-KDE-ktimetracker-duration")
	if dur == nil || dur.Value != "3600" {
		t.Error("Expected X-KDE-ktimetracker-duration to be 3600")
	}

	cat := ve.GetProperty("CATEGORIES")
	if cat == nil || cat.Value != "KTimeTracker" {
		t.Error("Expected CATEGORIES:KTimeTracker")
	}

	transp := ve.GetProperty("TRANSP")
	if transp == nil || transp.Value != "OPAQUE" {
		t.Error("Expected TRANSP:OPAQUE")
	}
}

func TestUpdateEventLastModified(t *testing.T) {
	worklog := createTestWorklog()
	start := time.Date(2023, 8, 14, 9, 0, 0, 0, time.Local)
	end := time.Date(2023, 8, 14, 10, 0, 0, 0, time.Local)
	event := NewEvent("task-uuid", start, end, 3600, "Test", "")
	worklog.AddEvent(event)

	// Set LAST-MODIFIED to a known old value to ensure it changes
	for i, prop := range event.properties {
		if prop.IANAToken == "LAST-MODIFIED" {
			event.properties[i].Value = "20230101T000000Z"
		}
	}

	newComment := "Updated"
	worklog.UpdateEvent(event.uuid, EventUpdate{Comment: &newComment})

	newLastMod := ""
	for _, prop := range event.properties {
		if prop.IANAToken == "LAST-MODIFIED" {
			newLastMod = prop.Value
		}
	}

	if newLastMod == "20230101T000000Z" {
		t.Error("Expected LAST-MODIFIED to be updated")
	}
}

func TestParseDurationFlag(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"30m", 1800},
		{"1h30m", 5400},
		{"3600s", 3600},
		{"3600", 3600},
		{"2h", 7200},
	}

	for _, tt := range tests {
		result, err := parseDurationFlag(tt.input)
		if err != nil {
			t.Errorf("parseDurationFlag(%q) error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("parseDurationFlag(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestParseDurationFlagInvalid(t *testing.T) {
	_, err := parseDurationFlag("not-a-duration")
	if err == nil {
		t.Error("Expected error for invalid duration")
	}
}

func TestParseTimeFlag(t *testing.T) {
	now := time.Now()
	tests := []struct {
		input    string
		expected time.Time
	}{
		{"2023-08-14T09:00:00", time.Date(2023, 8, 14, 9, 0, 0, 0, time.Local)},
		{"2023-08-14 09:00:00", time.Date(2023, 8, 14, 9, 0, 0, 0, time.Local)},
		{"2023-08-14T09:00", time.Date(2023, 8, 14, 9, 0, 0, 0, time.Local)},
		{"09:00:00", time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.Local)},
		{"09:00", time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, time.Local)},
	}

	for _, tt := range tests {
		result, err := parseTimeFlag(tt.input)
		if err != nil {
			t.Errorf("parseTimeFlag(%q) error: %v", tt.input, err)
			continue
		}
		if !result.Equal(tt.expected) {
			t.Errorf("parseTimeFlag(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseTimeFlagInvalid(t *testing.T) {
	_, err := parseTimeFlag("not-a-time")
	if err == nil {
		t.Error("Expected error for invalid time")
	}
}

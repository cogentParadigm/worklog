package main

import (
	"os"
	"testing"
	"time"

	ics "github.com/arran4/golang-ical"
)

func TestItCanOpenIcsFiles(t *testing.T) {
	cal, err := openCalendar("testdata/example.ics")
	if err != nil {
		t.Fatalf("Unexpected error opening calendar: %v", err)
	}
	expected := 32
	actual := len(cal.Components)
	if actual != expected {
		t.Errorf("'%d' does not match expected '%d'", actual, expected)
	}
}

func TestItCanSaveIcsFiles(t *testing.T) {
	cal := createTestIcsCalendar()

	if err := saveCalendar("testdata/actual.ics", cal); err != nil {
		t.Fatalf("Unexpected error saving calendar: %v", err)
	}

	actualBytes, _ := os.ReadFile("testdata/actual.ics")
	expectedBytes, _ := os.ReadFile("testdata/expected.ics")

	if string(actualBytes) != string(expectedBytes) {
		t.Error("Contents of actual.ics does not match expected.ics")
	}

}

func createTestIcsCalendar() *ics.Calendar {
	cal := ics.NewCalendar()
	todo1 := ics.VTodo{}
	todo1.SetProperty(ics.ComponentPropertyUniqueId, "71ee80e9-d292-4a75-ace4-2964c04daa6c")
	todo1.SetProperty(ics.ComponentPropertySummary, "Test Task 1")
	todo1.SetProperty(ics.ComponentPropertyDescription, "This is the first task")
	todo2 := ics.VTodo{}
	todo2.SetProperty(ics.ComponentPropertyUniqueId, "231b8414-cbdb-4544-a6d2-460193be850c")
	todo2.SetProperty(ics.ComponentPropertySummary, "Test Task 2")
	todo2.SetProperty(ics.ComponentPropertyDescription, "This is the second task")
	cal.Components = append(cal.Components, &todo1, &todo2)
	event1 := cal.AddEvent("0d93a2f2-7a60-413e-ab38-bebeb21c1cb9")
	startTime := time.Date(2023, time.August, 27, 17, 0, 0, 0, time.UTC)
	duration, _ := time.ParseDuration("30m")
	endTime := startTime.Add(duration)
	event1.SetStartAt(startTime)
	event1.SetEndAt(endTime)
	event1.SetSummary("Event 1")
	event1.SetProperty(ics.ComponentProperty(ics.PropertyComment), "This is the first event")
	event1.SetProperty("RELATED-TO", todo1.GetProperty(ics.ComponentPropertyUniqueId).Value)
	event2 := cal.AddEvent("fdde4f82-1109-4fc6-80e4-42b40e000076")
	event2.SetStartAt(startTime)
	event2.SetEndAt(endTime)
	event2.SetSummary("Event 2")
	event2.SetProperty(ics.ComponentProperty(ics.PropertyComment), "This is the second event")
	event2.SetProperty("RELATED-TO", todo2.GetProperty(ics.ComponentPropertyUniqueId).Value)
	return cal
}

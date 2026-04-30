package main

import (
	"fmt"
	"os"
	"strings"

	ics "github.com/arran4/golang-ical"
)

// ---------------------------------------------------------
// open and save functions to read and write from ics file
// ---------------------------------------------------------

func openCalendar(path string) (*ics.Calendar, error) {
	icsBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read ics file: %w", err)
	}
	cal, err := ics.ParseCalendar(strings.NewReader(string(icsBytes)))
	if err != nil {
		return nil, fmt.Errorf("parse ics file: %w", err)
	}
	return cal, nil
}

func saveCalendar(path string, cal *ics.Calendar) error {
	err := os.WriteFile(path, []byte(cal.Serialize()), 0644)
	if err != nil {
		return fmt.Errorf("write ics file: %w", err)
	}
	return nil
}

// ---------------------------------------------------------
// helper functions to extract data from the ics.Calendar
// and nested components (todos, events, properties, etc..)
// ---------------------------------------------------------

func getTodos(cal *ics.Calendar) (r []*ics.VTodo) {
	r = []*ics.VTodo{}
	for i := range cal.Components {
		switch todo := cal.Components[i].(type) {
		case *ics.VTodo:
			r = append(r, todo)
		}
	}
	return r
}

func getEvents(cal *ics.Calendar) (r []*ics.VEvent) {
	r = []*ics.VEvent{}
	for i := range cal.Components {
		switch event := cal.Components[i].(type) {
		case *ics.VEvent:
			r = append(r, event)
		}
	}
	return r
}

func getProperty(todo *ics.VTodo, componentProperty ics.ComponentProperty) string {
	property := todo.GetProperty(componentProperty)
	if property != nil {
		return property.Value
	}
	return ""
}

func getEventProperty(event *ics.VEvent, prop ics.ComponentProperty) string {
	property := event.GetProperty(prop)
	if property != nil {
		return property.Value
	}
	return ""
}

// ---------------------------------------------------------
// helper functions to print data from the ics.Calendar
// and nested components for debugging only.
// ---------------------------------------------------------

func getProperties(tasks []*ics.VTodo) {
	for _, task := range tasks {
		for _, property := range task.Properties {
			fmt.Printf("%v: %v\n", property.IANAToken, property.Value)
		}
	}
}

func getSummaries(tasks []*ics.VTodo) {
	for _, task := range tasks {
		fmt.Printf("- %v\n\n", task.GetProperty(ics.ComponentPropertySummary).Value)
	}
}

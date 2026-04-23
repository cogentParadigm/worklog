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

func openCalendar(path string) *ics.Calendar {
	icsBytes, err := os.ReadFile(path)
	handleError(err)
	cal, err := ics.ParseCalendar(strings.NewReader(string(icsBytes)))
	handleError(err)
	return cal
}

func saveCalendar(path string, cal *ics.Calendar) {
	err := os.WriteFile(path, []byte(cal.Serialize()), 0644)
	handleError(err)
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

func getProperty(todo *ics.VTodo, componentProperty ics.ComponentProperty) string {
	property := todo.GetProperty(componentProperty)
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

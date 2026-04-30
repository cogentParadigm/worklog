package main

import (
	"strconv"
	"time"

	ics "github.com/arran4/golang-ical"
)

type Event struct {
	uuid        string
	summary     string
	description string
	relatedTo   string
	dtstart     time.Time
	dtend       time.Time
	dtstartProp *ics.IANAProperty // preserves original DTSTART formatting/TZID
	dtendProp   *ics.IANAProperty // preserves original DTEND formatting/TZID
	duration    int
	properties  []ics.IANAProperty
}

func makeEventForVEvent(ve *ics.VEvent) Event {
	event := Event{
		uuid:        getEventProperty(ve, ics.ComponentPropertyUniqueId),
		summary:     getEventProperty(ve, ics.ComponentPropertySummary),
		description: getEventProperty(ve, ics.ComponentPropertyDescription),
		relatedTo:   getEventProperty(ve, "RELATED-TO"),
	}

	// Try to parse dtstart/dtend as time.Time for Phase 2/4 operations
	if dtstart, err := ve.GetStartAt(); err == nil {
		event.dtstart = dtstart
	}
	if dtend, err := ve.GetEndAt(); err == nil {
		event.dtend = dtend
	}

	// Capture original DTSTART/DTEND properties for round-trip preservation
	for i, prop := range ve.Properties {
		switch prop.IANAToken {
		case string(ics.ComponentPropertyDtStart):
			event.dtstartProp = &ve.Properties[i]
		case string(ics.ComponentPropertyDtEnd):
			event.dtendProp = &ve.Properties[i]
		}
	}

	// Parse X-KDE-ktimetracker-duration
	if durProp := ve.GetProperty("X-KDE-ktimetracker-duration"); durProp != nil {
		if d, err := strconv.Atoi(durProp.Value); err == nil {
			event.duration = d
		}
	}

	// Collect all unknown properties (skip the ones we've handled)
	for _, prop := range ve.Properties {
		switch prop.IANAToken {
		case string(ics.ComponentPropertyUniqueId),
			string(ics.ComponentPropertySummary),
			string(ics.ComponentPropertyDescription),
			"RELATED-TO",
			string(ics.ComponentPropertyDtStart),
			string(ics.ComponentPropertyDtEnd),
			"X-KDE-ktimetracker-duration":
			continue
		}
		event.properties = append(event.properties, prop)
	}

	return event
}

func makeVEventForEvent(event *Event) ics.VEvent {
	ve := ics.VEvent{}
	ve.SetProperty(ics.ComponentPropertyUniqueId, event.uuid)
	ve.SetProperty(ics.ComponentPropertySummary, event.summary)
	if event.description != "" {
		ve.SetProperty(ics.ComponentPropertyDescription, event.description)
	}
	if event.relatedTo != "" {
		ve.SetProperty("RELATED-TO", event.relatedTo)
	}

	// Restore original DTSTART/DTEND properties if available
	if event.dtstartProp != nil {
		ve.Properties = append(ve.Properties, *event.dtstartProp)
	} else if !event.dtstart.IsZero() {
		ve.SetStartAt(event.dtstart)
	}
	if event.dtendProp != nil {
		ve.Properties = append(ve.Properties, *event.dtendProp)
	} else if !event.dtend.IsZero() {
		ve.SetEndAt(event.dtend)
	}

	// Restore duration
	if event.duration > 0 {
		ve.SetProperty("X-KDE-ktimetracker-duration", strconv.Itoa(event.duration))
	}

	ve.Properties = append(ve.Properties, event.properties...)

	return ve
}

func makeEventsForVEvents(ves []*ics.VEvent) []*Event {
	events := make([]*Event, len(ves))
	for i, ve := range ves {
		event := makeEventForVEvent(ve)
		events[i] = &event
	}
	return events
}

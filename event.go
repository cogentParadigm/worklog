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

	// Capture all original properties and keep pointers to DTSTART/DTEND
	// within our own copy so round-trips preserve property order.
	event.properties = append([]ics.IANAProperty(nil), ve.Properties...)
	for i, prop := range event.properties {
		switch prop.IANAToken {
		case string(ics.ComponentPropertyDtStart):
			event.dtstartProp = &event.properties[i]
		case string(ics.ComponentPropertyDtEnd):
			event.dtendProp = &event.properties[i]
		}
	}

	// Parse X-KDE-ktimetracker-duration
	if durProp := ve.GetProperty("X-KDE-ktimetracker-duration"); durProp != nil {
		if d, err := strconv.Atoi(durProp.Value); err == nil {
			event.duration = d
		}
	}

	return event
}

func makeVEventForEvent(event *Event) ics.VEvent {
	ve := ics.VEvent{}

	emitted := make(map[string]bool)

	for _, prop := range event.properties {
		switch prop.IANAToken {
		case string(ics.ComponentPropertyUniqueId):
			ve.SetProperty(ics.ComponentPropertyUniqueId, event.uuid)
			emitted["UID"] = true
		case string(ics.ComponentPropertySummary):
			ve.SetProperty(ics.ComponentPropertySummary, event.summary)
			emitted["SUMMARY"] = true
		case string(ics.ComponentPropertyDescription):
			if event.description != "" {
				ve.SetProperty(ics.ComponentPropertyDescription, event.description)
				emitted["DESCRIPTION"] = true
			}
		case "RELATED-TO":
			if event.relatedTo != "" {
				ve.SetProperty("RELATED-TO", event.relatedTo)
				emitted["RELATED-TO"] = true
			}
		case string(ics.ComponentPropertyDtStart):
			if event.dtstartProp != nil {
				ve.Properties = append(ve.Properties, *event.dtstartProp)
			} else if !event.dtstart.IsZero() {
				ve.SetStartAt(event.dtstart)
			}
			emitted["DTSTART"] = true
		case string(ics.ComponentPropertyDtEnd):
			if event.dtendProp != nil {
				ve.Properties = append(ve.Properties, *event.dtendProp)
			} else if !event.dtend.IsZero() {
				ve.SetEndAt(event.dtend)
			}
			emitted["DTEND"] = true
		case "X-KDE-ktimetracker-duration":
			if event.duration > 0 {
				ve.SetProperty("X-KDE-ktimetracker-duration", strconv.Itoa(event.duration))
				emitted["X-KDE-ktimetracker-duration"] = true
			}
		default:
			ve.Properties = append(ve.Properties, prop)
		}
	}

	if !emitted["UID"] {
		ve.SetProperty(ics.ComponentPropertyUniqueId, event.uuid)
	}
	if !emitted["SUMMARY"] {
		ve.SetProperty(ics.ComponentPropertySummary, event.summary)
	}
	if !emitted["DESCRIPTION"] && event.description != "" {
		ve.SetProperty(ics.ComponentPropertyDescription, event.description)
	}
	if !emitted["RELATED-TO"] && event.relatedTo != "" {
		ve.SetProperty("RELATED-TO", event.relatedTo)
	}
	if !emitted["DTSTART"] && !event.dtstart.IsZero() {
		if event.dtstartProp != nil {
			ve.Properties = append(ve.Properties, *event.dtstartProp)
		} else {
			ve.SetStartAt(event.dtstart)
		}
	}
	if !emitted["DTEND"] && !event.dtend.IsZero() {
		if event.dtendProp != nil {
			ve.Properties = append(ve.Properties, *event.dtendProp)
		} else {
			ve.SetEndAt(event.dtend)
		}
	}
	if !emitted["X-KDE-ktimetracker-duration"] && event.duration > 0 {
		ve.SetProperty("X-KDE-ktimetracker-duration", strconv.Itoa(event.duration))
	}

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

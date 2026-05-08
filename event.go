package main

import (
	"strconv"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/google/uuid"
)

type Event struct {
	uuid       string
	summary    string
	comment    string
	relatedTo  string
	dtstart    time.Time
	dtend      time.Time
	dtstartProp *ics.IANAProperty // preserves original DTSTART formatting/TZID
	dtendProp   *ics.IANAProperty // preserves original DTEND formatting/TZID
	duration    int
	properties  []ics.IANAProperty
}

func NewEvent(taskUUID string, start, end time.Time, duration int, note, comment string) *Event {
	now := time.Now().UTC()
	return &Event{
		uuid:      uuid.New().String(),
		relatedTo: taskUUID,
		dtstart:   start,
		dtend:     end,
		duration:  duration,
		summary:   note,
		comment:   comment,
		properties: []ics.IANAProperty{
		{BaseProperty: ics.BaseProperty{IANAToken: "DTSTAMP", Value: now.Format("20060102T150405Z")}},
		{BaseProperty: ics.BaseProperty{IANAToken: "CREATED", Value: now.Format("20060102T150405Z")}},
		{BaseProperty: ics.BaseProperty{IANAToken: "LAST-MODIFIED", Value: now.Format("20060102T150405Z")}},
		{BaseProperty: ics.BaseProperty{IANAToken: "TRANSP", Value: "OPAQUE"}},
		},
	}
}

func (event *Event) updateLastModified() {
	now := time.Now().UTC().Format("20060102T150405Z")
	found := false
	for i, prop := range event.properties {
		if prop.IANAToken == "LAST-MODIFIED" {
			event.properties[i].Value = now
			found = true
			break
		}
	}
	if !found {
		event.properties = append(event.properties, ics.IANAProperty{
			BaseProperty: ics.BaseProperty{IANAToken: "LAST-MODIFIED", Value: now},
		})
	}
}

func makeEventForVEvent(ve *ics.VEvent) Event {
	event := Event{
		uuid:      getEventProperty(ve, ics.ComponentPropertyUniqueId),
		summary:   getEventProperty(ve, ics.ComponentPropertySummary),
		comment:   getEventProperty(ve, ics.ComponentProperty(ics.PropertyComment)),
		relatedTo: getEventProperty(ve, "RELATED-TO"),
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
		case string(ics.ComponentProperty(ics.PropertyComment)):
			if event.comment != "" {
				ve.SetProperty(ics.ComponentProperty(ics.PropertyComment), event.comment)
				emitted["COMMENT"] = true
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
	if !emitted["COMMENT"] && event.comment != "" {
		ve.SetProperty(ics.ComponentProperty(ics.PropertyComment), event.comment)
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

package main

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
	"github.com/google/uuid"
)

type Event struct {
	uuid        string
	summary     string
	comment     string
	relatedTo   string
	dtstart     time.Time
	dtend       time.Time
	dtstartProp *ics.IANAProperty // preserves original DTSTART formatting/TZID
	dtendProp   *ics.IANAProperty // preserves original DTEND formatting/TZID
	duration    int
	active      bool // DTSTART without DTEND or an explicit duration
	properties  []ics.IANAProperty
}

func (event *Event) getUUID() string {
	return event.uuid
}

func (event *Event) isComplete() bool {
	return !event.active
}

var iCalDurationRE = regexp.MustCompile(`^([+-])?P(?:(\d+)W|(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?)$`)

// parseICalDuration parses the duration value format defined by RFC 5545.
func parseICalDuration(value string) (int, bool) {
	matches := iCalDurationRE.FindStringSubmatch(strings.TrimSpace(value))
	if matches == nil {
		return 0, false
	}

	// Require at least one duration component; P and PT alone are invalid.
	hasComponent := false
	total := 0
	multipliers := []int{7 * 24 * 60 * 60, 24 * 60 * 60, 60 * 60, 60, 1}
	for i, multiplier := range multipliers {
		valueIndex := i + 2
		if matches[valueIndex] == "" {
			continue
		}
		hasComponent = true
		part, err := strconv.Atoi(matches[valueIndex])
		if err != nil {
			return 0, false
		}
		total += part * multiplier
	}
	if !hasComponent {
		return 0, false
	}
	if matches[1] == "-" {
		total = -total
	}
	return total, true
}

// eventInDateRange reports whether an event falls within [fromDay, toDay]
// inclusive. Events with no DTSTART are rejected.
func eventInDateRange(event *Event, fromDay, toDay time.Time) bool {
	if event.dtstart.IsZero() {
		return false
	}
	day := time.Date(event.dtstart.Year(), event.dtstart.Month(), event.dtstart.Day(), 0, 0, 0, 0, event.dtstart.Location())
	if !fromDay.IsZero() && day.Before(fromDay) {
		return false
	}
	if !toDay.IsZero() && day.After(toDay) {
		return false
	}
	return true
}

// eventInDateRangeOrUndated reports whether an event falls within [fromDay, toDay]
// inclusive. Events with no DTSTART are accepted only when no date bounds are set.
func eventInDateRangeOrUndated(event *Event, fromDay, toDay time.Time) bool {
	if event.dtstart.IsZero() {
		return fromDay.IsZero() && toDay.IsZero()
	}
	return eventInDateRange(event, fromDay, toDay)
}

// Generic property helpers --------------------------------------------------

func (event *Event) getProperty(token string) string {
	for _, prop := range event.properties {
		if prop.IANAToken == token && prop.Value != "" {
			return prop.Value
		}
	}
	return ""
}

func (event *Event) setProperty(token, value string) {
	for i, prop := range event.properties {
		if prop.IANAToken == token {
			event.properties[i].Value = value
			return
		}
	}
	event.properties = append(event.properties, ics.IANAProperty{
		BaseProperty: ics.BaseProperty{IANAToken: token, Value: value},
	})
}

func (event *Event) removeProperty(token string) {
	var newProps []ics.IANAProperty
	for _, prop := range event.properties {
		if prop.IANAToken != token {
			newProps = append(newProps, prop)
		}
	}
	event.properties = newProps
}

func NewEvent(taskUUID string, start, end time.Time, duration int, note, comment string) *Event {
	now := time.Now().UTC()
	return &Event{
		uuid:      uuid.New().String(),
		relatedTo: taskUUID,
		dtstart:   start,
		dtend:     end,
		duration:  duration,
		active:    end.IsZero(),
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
	event.setProperty("LAST-MODIFIED", time.Now().UTC().Format("20060102T150405Z"))
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

	// Prefer standard DTSTART/DTEND, then standard DURATION. KTimeTracker
	// represents duration corrections as DTSTART-only events with a signed
	// X-KDE-ktimetracker-duration value, so support that as a compatibility
	// fallback. A DTSTART-only event without an explicit duration is active.
	if !event.dtstart.IsZero() {
		if !event.dtend.IsZero() {
			event.duration = int(event.dtend.Sub(event.dtstart).Seconds())
		} else if duration, ok := parseICalDuration(getEventProperty(ve, ics.ComponentProperty(ics.PropertyDuration))); ok {
			event.duration = duration
		} else if duration, err := strconv.Atoi(strings.TrimSpace(getEventProperty(ve, "X-KDE-ktimetracker-duration"))); err == nil && getEventProperty(ve, "X-KDE-ktimetracker-duration") != "" {
			event.duration = duration
		} else {
			event.active = true
			event.duration = int(time.Since(event.dtstart).Seconds())
		}
	}

	return event
}

func makeVEventForEvent(event *Event) ics.VEvent {
	ve := ics.VEvent{}

	emitted := make(emittedSet)

	for _, prop := range event.properties {
		switch prop.IANAToken {
		case string(ics.ComponentPropertyUniqueId):
			ve.SetProperty(ics.ComponentPropertyUniqueId, event.uuid)
			emitted.mark("UID")
		case string(ics.ComponentPropertySummary):
			ve.SetProperty(ics.ComponentPropertySummary, event.summary)
			emitted.mark("SUMMARY")
		case string(ics.ComponentProperty(ics.PropertyComment)):
			if event.comment != "" {
				ve.SetProperty(ics.ComponentProperty(ics.PropertyComment), event.comment)
				emitted.mark("COMMENT")
			}
		case "RELATED-TO":
			if event.relatedTo != "" {
				ve.SetProperty("RELATED-TO", event.relatedTo)
				emitted.mark("RELATED-TO")
			}
		case string(ics.ComponentPropertyDtStart):
			if event.dtstartProp != nil {
				ve.Properties = append(ve.Properties, *event.dtstartProp)
			} else if !event.dtstart.IsZero() {
				ve.SetStartAt(event.dtstart)
			}
			emitted.mark("DTSTART")
		case string(ics.ComponentPropertyDtEnd):
			if event.dtendProp != nil {
				ve.Properties = append(ve.Properties, *event.dtendProp)
			} else if !event.dtend.IsZero() {
				ve.SetEndAt(event.dtend)
			}
			emitted.mark("DTEND")

		default:
			ve.Properties = append(ve.Properties, prop)
		}
	}

	if !emitted.has("UID") {
		ve.SetProperty(ics.ComponentPropertyUniqueId, event.uuid)
	}
	if !emitted.has("SUMMARY") {
		ve.SetProperty(ics.ComponentPropertySummary, event.summary)
	}
	if !emitted.has("COMMENT") && event.comment != "" {
		ve.SetProperty(ics.ComponentProperty(ics.PropertyComment), event.comment)
	}
	if !emitted.has("RELATED-TO") && event.relatedTo != "" {
		ve.SetProperty("RELATED-TO", event.relatedTo)
	}
	if !emitted.has("DTSTART") && !event.dtstart.IsZero() {
		if event.dtstartProp != nil {
			ve.Properties = append(ve.Properties, *event.dtstartProp)
		} else {
			ve.SetStartAt(event.dtstart)
		}
	}
	if !emitted.has("DTEND") && !event.dtend.IsZero() {
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

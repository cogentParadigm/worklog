package main

import (
	"regexp"
	"time"

	ics "github.com/arran4/golang-ical"
)

var issueKeyRegex = regexp.MustCompile(`([A-Z][A-Z0-9]+-\d+)`)

func (task *Task) IssueKey() string {
	for _, prop := range task.properties {
		if prop.IANAToken == "X-WORKLOG-ISSUE-KEY" && prop.Value != "" {
			return prop.Value
		}
	}
	if match := issueKeyRegex.FindString(task.name); match != "" {
		return match
	}
	return ""
}

func (event *Event) SyncedAt() time.Time {
	for _, prop := range event.properties {
		if prop.IANAToken == "X-WORKLOG-SYNCED-AT" {
			if t, err := time.Parse("20060102T150405Z", prop.Value); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

func (event *Event) SetSyncedAt(t time.Time) {
	value := t.UTC().Format("20060102T150405Z")
	for i, prop := range event.properties {
		if prop.IANAToken == "X-WORKLOG-SYNCED-AT" {
			event.properties[i].Value = value
			return
		}
	}
	event.properties = append(event.properties, ics.IANAProperty{
		BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-SYNCED-AT", Value: value},
	})
}

func (event *Event) LastModified() time.Time {
	for _, prop := range event.properties {
		if prop.IANAToken == "LAST-MODIFIED" {
			if t, err := time.Parse("20060102T150405Z", prop.Value); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

package main

import (
	"regexp"
	"strings"
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

func (task *Task) IssueID() string {
	for _, prop := range task.properties {
		if prop.IANAToken == "X-WORKLOG-ISSUE-ID" && prop.Value != "" {
			return prop.Value
		}
	}
	return ""
}

func (task *Task) SetIssueID(id string) {
	for i, prop := range task.properties {
		if prop.IANAToken == "X-WORKLOG-ISSUE-ID" {
			task.properties[i].Value = id
			return
		}
	}
	task.properties = append(task.properties, ics.IANAProperty{
		BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-ISSUE-ID", Value: id},
	})
}

func (task *Task) ClearIssueID() {
	var newProps []ics.IANAProperty
	for _, prop := range task.properties {
		if prop.IANAToken != "X-WORKLOG-ISSUE-ID" {
			newProps = append(newProps, prop)
		}
	}
	task.properties = newProps
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

func (task *Task) TempoAttributes() map[string]string {
	attrs := make(map[string]string)
	prefix := "X-WORKLOG-TEMPO-ATTR-"
	for _, prop := range task.properties {
		if strings.HasPrefix(prop.IANAToken, prefix) && prop.Value != "" {
			key := strings.TrimPrefix(prop.IANAToken, prefix)
			attrs[key] = prop.Value
		}
	}
	return attrs
}

func (task *Task) SetTempoAttribute(key, value string) {
	token := "X-WORKLOG-TEMPO-ATTR-" + key
	for i, prop := range task.properties {
		if prop.IANAToken == token {
			task.properties[i].Value = value
			return
		}
	}
	task.properties = append(task.properties, ics.IANAProperty{
		BaseProperty: ics.BaseProperty{IANAToken: token, Value: value},
	})
}

func (task *Task) ClearTempoAttribute(key string) {
	token := "X-WORKLOG-TEMPO-ATTR-" + key
	var newProps []ics.IANAProperty
	for _, prop := range task.properties {
		if prop.IANAToken != token {
			newProps = append(newProps, prop)
		}
	}
	task.properties = newProps
}

package main

import (
	"regexp"
	"time"
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
	return task.getProperty("X-WORKLOG-ISSUE-ID")
}

func (task *Task) SetIssueID(id string) {
	task.setProperty("X-WORKLOG-ISSUE-ID", id)
}

func (task *Task) ClearIssueID() {
	task.removeProperty("X-WORKLOG-ISSUE-ID")
}

func (event *Event) SyncedAt() time.Time {
	if v := event.getProperty("X-WORKLOG-SYNCED-AT"); v != "" {
		if t, err := time.Parse("20060102T150405Z", v); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (event *Event) SetSyncedAt(t time.Time) {
	event.setProperty("X-WORKLOG-SYNCED-AT", t.UTC().Format("20060102T150405Z"))
}

func (event *Event) LastModified() time.Time {
	if v := event.getProperty("LAST-MODIFIED"); v != "" {
		if t, err := time.Parse("20060102T150405Z", v); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (task *Task) TempoAttributes() map[string]string {
	attrs := make(map[string]string)
	for _, prop := range task.properties {
		if prop.IANAToken == "X-WORKLOG-TEMPO-ATTR" && prop.Value != "" {
			if keys, ok := prop.ICalParameters["KEY"]; ok && len(keys) > 0 {
				attrs[keys[0]] = prop.Value
			}
		}
	}
	return attrs
}

func (task *Task) SetTempoAttribute(key, value string) {
	task.setPropertyParam("X-WORKLOG-TEMPO-ATTR", "KEY", key, value)
}

func (task *Task) ClearTempoAttribute(key string) {
	task.removePropertyParam("X-WORKLOG-TEMPO-ATTR", "KEY", key)
}

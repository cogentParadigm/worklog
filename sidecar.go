package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	ics "github.com/arran4/golang-ical"
)

type Sidecar struct {
	Version    int                  `json:"version"`
	LastHash   string               `json:"last_ics_hash,omitempty"`
	ModifiedAt string               `json:"modified_at,omitempty"`
	Tasks      map[string]TaskMeta  `json:"tasks"`
	Events     map[string]EventMeta `json:"events"`
}

type TaskMeta struct {
	IssueID    string            `json:"issue_id,omitempty"`
	IssueKey   string            `json:"issue_key,omitempty"`
	TempoAttrs map[string]string `json:"tempo_attrs,omitempty"`
}

type EventMeta struct {
	SyncedAt string `json:"synced_at,omitempty"`
}

func sidecarPath(icsPath string) string {
	return icsPath + ".worklog"
}

func loadSidecar(icsPath string) (*Sidecar, error) {
	path := sidecarPath(icsPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Sidecar{
				Version: 1,
				Tasks:   make(map[string]TaskMeta),
				Events:  make(map[string]EventMeta),
			}, nil
		}
		return nil, fmt.Errorf("read sidecar %s: %w", path, err)
	}
	var sc Sidecar
	if err := json.Unmarshal(data, &sc); err != nil {
		return nil, fmt.Errorf("parse sidecar %s: %w", path, err)
	}
	if sc.Tasks == nil {
		sc.Tasks = make(map[string]TaskMeta)
	}
	if sc.Events == nil {
		sc.Events = make(map[string]EventMeta)
	}
	return &sc, nil
}

func saveSidecar(icsPath string, sc *Sidecar) error {
	path := sidecarPath(icsPath)
	sc.ModifiedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sidecar: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write sidecar %s: %w", path, err)
	}
	return nil
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func buildSidecar(tasks []*Task, events []*Event) *Sidecar {
	sc := &Sidecar{
		Version: 1,
		Tasks:   make(map[string]TaskMeta),
		Events:  make(map[string]EventMeta),
	}
	for _, task := range flattenTasks(tasks) {
		meta := TaskMeta{
			IssueID:    task.IssueID(),
			TempoAttrs: task.TempoAttributes(),
		}
		for _, prop := range task.properties {
			if prop.IANAToken == "X-WORKLOG-ISSUE-KEY" {
				meta.IssueKey = prop.Value
				break
			}
		}
		sc.Tasks[task.uuid] = meta
	}
	for _, event := range events {
		if !event.SyncedAt().IsZero() {
			sc.Events[event.uuid] = EventMeta{
				SyncedAt: event.SyncedAt().UTC().Format("20060102T150405Z"),
			}
		}
	}
	return sc
}

func restoreFromSidecar(tasks []*Task, events []*Event, sc *Sidecar) {
	for _, task := range flattenTasks(tasks) {
		meta, ok := sc.Tasks[task.uuid]
		if !ok {
			continue
		}
		restored := false

		if task.IssueID() == "" && meta.IssueID != "" {
			task.SetIssueID(meta.IssueID)
			restored = true
		}

		hasExplicitKey := false
		for _, prop := range task.properties {
			if prop.IANAToken == "X-WORKLOG-ISSUE-KEY" {
				hasExplicitKey = true
				break
			}
		}
		if !hasExplicitKey && meta.IssueKey != "" {
			task.properties = append(task.properties, ics.IANAProperty{
				BaseProperty: ics.BaseProperty{IANAToken: "X-WORKLOG-ISSUE-KEY", Value: meta.IssueKey},
			})
			restored = true
		}

		currentAttrs := task.TempoAttributes()
		for key, value := range meta.TempoAttrs {
			if _, ok := currentAttrs[key]; !ok {
				task.SetTempoAttribute(key, value)
				restored = true
			}
		}

		if restored {
			fmt.Fprintf(os.Stderr, "Restored metadata for task %q from sidecar (another application may have stripped worklog properties)\n", task.name)
		}
	}

	for _, event := range events {
		meta, ok := sc.Events[event.uuid]
		if !ok {
			continue
		}
		if event.SyncedAt().IsZero() && meta.SyncedAt != "" {
			if t, err := time.Parse("20060102T150405Z", meta.SyncedAt); err == nil {
				event.SetSyncedAt(t)
				fmt.Fprintf(os.Stderr, "Restored metadata for event %q from sidecar (another application may have stripped worklog properties)\n", event.summary)
			}
		}
	}
}

# Known Issues

This document tracks known issues identified during code review.

- **Tempo comment source**: Currently the Tempo worklog comment is built by aggregating unique time-entry comments. When time entries have no comments, the worklog is sent with an empty description. Consider whether the task description should be used as a fallback comment, or whether a per-task `X-WORKLOG-TEMPO-COMMENT` override property should be added.

- **`main.go` god function**: The `run()` function is 1235 lines containing inline flag setup, parsing, validation, and business logic for every subcommand. Extract each subcommand into a standalone function and use a simple command registry map.

- **`cmd_jira.go` mixed concerns**: `runJiraSync` alone is ~300 lines mixing CLI parsing, API client setup, data aggregation, preview formatting, user confirmation, and API calls. `buildTimesheetFromSyncEntries` also duplicates day-range and row-building logic already in `generateTimesheet`. Split into dedicated phases and reuse the existing timesheet generator for previews.

- **`collectAllTasks` duplicates `flattenTasks`**: `cmd_jira.go:56-63` is identical in behavior to `task.go:105-112`. Remove the dead duplication.

- **Config `Get` / `GetUnmasked` duplication**: `config.go:110-164` has two near-identical 30-line switch statements. Any new key requires editing both. Have `Get` delegate to `GetUnmasked` and mask after, or use a struct tag/reflection approach.

- **`resolveTaskUUID` and `resolveEventUUID` are ~90% identical**: `uuid.go:53-133` — same exact-match, prefix-scan, and ambiguous-reporting logic. Extract a generic `resolveByPrefix` helper.

- **Property manipulation methods repeat the same pattern**: `jira.go` and `event.go` contain ~10 methods (`SetIssueID`, `ClearIssueID`, `SetSyncedAt`, `SetTempoAttribute`, etc.) that all loop over `properties`, match by `IANAToken`, then update/delete/append. Add generic helpers: `setProperty`, `getProperty`, `removeProperty`.

- **`buildSidecar` reimplements `IssueKey()` logic**: `sidecar.go:95-99` manually scans `task.properties` for `X-WORKLOG-ISSUE-KEY` instead of calling `task.IssueKey()`.

- **`detectExistingICSFiles` hardcodes KTimeTracker paths**: `main.go:1050-1079` searches `~/.local/share/ktimetracker/` and `~/.kde/share/apps/ktimetracker/`, violating the AGENTS.md principle of iCalendar neutrality.

- **`TimesheetRow` mixes general and sync-preview concerns**: `report.go:20-27` — `attributes`, `issueKey`, and `hasComments` are only populated/used by the Jira sync preview path but exist in the general report struct, coupling reporting to Jira sync.

- **`printTimesheetTable` is ~200 lines of dense formatting logic**: `report.go:212-375` computes column widths, builds headers, prints separators, and renders rows all in one function, with sync-preview-specific branches adding further complexity.

- **Dead debug code in `ical.go`**: `getProperties()` and `getSummaries()` (lines 83-95) are debug helpers with `fmt.Printf`, never called in production or tests.

- **Date range filtering is copy-pasted in 3+ places**: `main.go:250-262` (time list), `cmd_jira.go:516-543` (findSkippedTasks), and `cmd_jira.go:573-624` (buildSyncEntries) all iterate events, check `dtstart.IsZero()`, strip to day, and compare `fromDay`/`toDay`. Extract a shared helper.

- **`TaskUpdate` pointer fields add noise**: `worklog.go:119-123` uses `*string` fields to distinguish "not provided" from "set to empty", forcing callers to use `strPtr()` everywhere. Consider a small builder or explicit flags.

- **`emitted` map pattern in `makeTodoForTask` / `makeVEventForEvent` is slightly opaque**: `task.go:38-74` and `event.go:101-171` use an `emitted` map to track which standard properties have been handled. A small named type or comment would improve readability.

- **`humanizeLabel` does complex rune-by-rune scanning**: `cmd_jira.go:798-813` is ~15 lines to insert spaces before capitals. Could be simplified with a regex or string replacement.

---

*Last reviewed: 2026-05-09*

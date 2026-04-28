# Agent Notes for worklog

## Project Structure

- `main.go` — CLI entry point and command dispatch
- `worklog.go` — `Worklog` type: in-memory task tree, file I/O, updates
- `task.go` — `Task` type and conversion to/from `ics.VTodo`
- `ical.go` — Low-level iCalendar parsing helpers (`openCalendar`, `getTodos`, `getEvents`, etc.)
- `*_test.go` — Tests

## Key Architectural Decisions

### Round-Trip iCalendar Preservation

`Worklog` stores the original `*ics.Calendar` parsed from disk in its `calendar` field. On `Save()`, `getCalendarForTasks` rebuilds the calendar, replacing VTODOs with updated task state while preserving all other components and calendar-level properties:

1. It walks the original calendar's `Components` in order.
2. VTODOs are replaced with updated versions from the current task tree (matched by UID).
3. VEVENTs and all other non-VTODO components are kept as-is.
4. Original calendar-level properties (`PRODID`, `VERSION`, `X-KDE-*`, etc.) are preserved.
5. Brand-new tasks (UIDs not in the original) are appended at the end.

This prevents data loss for KTimeTracker timer-session VEVENTs and any future unknown components.

### Task Tree Model

- `Task` has `parent`, `children`, `uuid`, `name`, `description`, `relatedTo`, `position`, and `properties`.
- `position` is set during load to the original VTODO's index and used to maintain deterministic ordering.
- Parent/child relationships are built from `RELATED-TO` during load and emitted back during save.

### Testing

- Run tests with `go test ./...`
- `testdata/example.ics` contains 12 VTODOs and 14 VEVENTs from a real KTimeTracker export.
- `testdata/expected.ics` is used for basic save/load assertions.

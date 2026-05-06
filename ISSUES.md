# Known Issues

This document tracks known issues identified during code review.

### Hardcoded ICS file paths
**Priority:** High
**Impact:** The compiled binary is unusable for real data without recompilation.

- `main.go` hardcodes `testdata/example.ics` as the input file.
- `Save()` constructs output path with `strings.Replace(path, ".ics", "-output.ics", 1)`, producing `example-output.ics`.
- A normal workflow should write back to the same file (in-place save), but we need a strategy for how the user specifies the input file: CLI flag, env var, config file, default path, or KTimeTracker file discovery.

### No way to discover task UUIDs from CLI
**Priority:** High
**Impact:** Users cannot `update`, `delete`, `create -parent`, or `time add` without manually reading the raw `.ics` file.

- `worklog list` shows `Name` and rolled-up `Total` only.
- `worklog time list` shows UUIDs, but a user should not have to log time to see task identifiers.
- Add `UUID` and **direct** `Duration` columns to `worklog list` output so it is uniform with `time list`.

### Inconsistent CLI command hierarchy
**Priority:** Medium
**Impact:** Mental model is fragmented; `time` and `report` are namespaces but task CRUD is at root.

- Current: `worklog list`, `worklog create`, `worklog update`, `worklog delete`
- Should be: `worklog task list`, `worklog task create`, `worklog task update`, `worklog task delete`
- This is a **breaking change**.

### No `-h` / `--help` on subcommands
**Priority:** Medium
**Impact:** Poor CLI discoverability.

- All subcommands use `flag.NewFlagSet(..., flag.ExitOnError)`, so `worklog create -h` exits with code 2 and prints default Go flag help to stderr.
- Users must run a command with missing arguments to see usage text.

### `time edit` cannot set event description
**Priority:** Low
**Impact:** `EventUpdate` has a `Description` field, but the CLI only exposes `-note` which maps to `Summary`.

- Add a `-description` flag to `time edit` (and possibly `time add`) to populate `Event.Description`.

### `Save()` has no integration test coverage
**Priority:** Medium
**Impact:** The entire load → modify → save → reload → verify round-trip is untested.

- `ical_test.go` tests `saveCalendar` with a fresh calendar, but no test exercises `Worklog.Save()` end-to-end.
- Should add a test that loads `example.ics`, mutates a task, saves, reloads, and asserts state preservation.

### `worklog list` destroys original VTODO position order
**Priority:** Low
**Impact:** KTimeTracker and other tools may rely on VTODO ordering.

- `printTasks()` sorts tasks alphabetically by name at every tree level.
- The original `position` field (set during load from the ICS component index) is ignored during display.
- Sorting should either respect `position` or be an optional flag.

### Unused TUI dependencies bloating binary
**Priority:** Low
**Impact:** Larger binary size and unnecessary supply-chain surface.

- `go.mod` includes `github.com/rivo/tview`, `github.com/gdamore/tcell/v2`, `github.com/lucasb-eyer/go-colorful`, etc.
- Zero code references these packages; they appear to be a vestige of Phase 4 TUI work.
- Remove from `go.mod` until TUI development actually begins.

---

*Last reviewed: 2026-05-06*

# Known Issues

This document tracks known issues identified during code review.

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

# Known Issues

This document tracks known issues identified during code review.


- **Date range filtering is copy-pasted in 3+ places**: `main.go:250-262` (time list), `cmd_jira.go:516-543` (findSkippedTasks), and `cmd_jira.go:573-624` (buildSyncEntries) all iterate events, check `dtstart.IsZero()`, strip to day, and compare `fromDay`/`toDay`. Extract a shared helper.

- **`TaskUpdate` pointer fields add noise**: `worklog.go:119-123` uses `*string` fields to distinguish "not provided" from "set to empty", forcing callers to use `strPtr()` everywhere. Consider a small builder or explicit flags.

- **`emitted` map pattern in `makeTodoForTask` / `makeVEventForEvent` is slightly opaque**: `task.go:38-74` and `event.go:101-171` use an `emitted` map to track which standard properties have been handled. A small named type or comment would improve readability.

- **`humanizeLabel` does complex rune-by-rune scanning**: `cmd_jira.go:798-813` is ~15 lines to insert spaces before capitals. Could be simplified with a regex or string replacement.

---

*Last reviewed: 2026-05-10* 

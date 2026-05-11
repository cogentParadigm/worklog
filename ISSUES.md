# Known Issues

This document tracks known issues identified during code review.


- **`TaskUpdate` pointer fields add noise**: `worklog.go:119-123` uses `*string` fields to distinguish "not provided" from "set to empty", forcing callers to use `strPtr()` everywhere. Consider a small builder or explicit flags.

- **`emitted` map pattern in `makeTodoForTask` / `makeVEventForEvent` is slightly opaque**: `task.go:38-74` and `event.go:101-171` use an `emitted` map to track which standard properties have been handled. A small named type or comment would improve readability.

- **`humanizeLabel` does complex rune-by-rune scanning**: `cmd_jira.go:798-813` is ~15 lines to insert spaces before capitals. Could be simplified with a regex or string replacement.

---

*Last reviewed: 2026-05-10* 

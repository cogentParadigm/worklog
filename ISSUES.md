# Known Issues

This document tracks known issues identified during code review.

## Issues

### `delete` Command Advertised but Unimplemented `[Medium]`
**Location:** `main.go`

**Description:** `delete` is listed in the usage banner and ROADMAP marks it incomplete, but there's no `case "delete"` in the switch statement.

**Impact:** Users see "delete" as an available command but get "Unknown command" error when trying to use it.

**Suggested Fix:**
- Either implement the delete command
- Or remove it from the usage banner until implemented

---

### Brittle Path Munging in `Save()` `[Low]`
**Location:** `worklog.go` - `Save()` function

**Description:** `strings.Replace(path, ".ics", "-output.ics", 1)` only replaces first occurrence. Paths like `backup.ics.old.ics` produce unexpected results.

**Impact:** Edge case path handling issues.

**Suggested Fix:**
- Use `strings.TrimSuffix` with `filepath.Ext()`
- Or use `filepath` package properly: `strings.TrimSuffix(path, filepath.Ext(path)) + "-output.ics"`

---

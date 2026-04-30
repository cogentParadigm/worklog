# Known Issues

This document tracks known issues identified during code review.

## Issues

### Brittle Path Munging in `Save()` `[Low]`
**Location:** `worklog.go` - `Save()` function

**Description:** `strings.Replace(path, ".ics", "-output.ics", 1)` only replaces first occurrence. Paths like `backup.ics.old.ics` produce unexpected results.

**Impact:** Edge case path handling issues.

**Suggested Fix:**
- Use `strings.TrimSuffix` with `filepath.Ext()`
- Or use `filepath` package properly: `strings.TrimSuffix(path, filepath.Ext(path)) + "-output.ics"`

---

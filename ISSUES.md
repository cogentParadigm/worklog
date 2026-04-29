# Known Issues

This document tracks known issues identified during code review.

## Issues

### `relatedTo` vs `parent` Inconsistency `[Medium]`
**Location:** `worklog.go`, `task.go`

**Description:** `UpdateTask` sets both `task.parent = newParent` and `task.relatedTo = parentUUID`. However, `makeTodoForTask` only looks at `task.parent` when emitting `RELATED-TO`, making the `relatedTo` write potentially dead code. During load, `makeTasksForTodos` treats `relatedTo` as the source of truth.

**Impact:** Potential confusion about which field is authoritative; risk of data inconsistency if `parent` and `relatedTo` diverge.

**Suggested Fix:**
- Clarify the relationship between these fields
- Consider removing `relatedTo` field and deriving it from `parent` during serialization
- Or ensure both fields stay synchronized

---

### Cannot Clear/Unset Fields `[Medium]`
**Location:** `worklog.go` - `UpdateTask()` function

**Description:** The empty-string-check pattern (`if name != ""`, `if parentUUID != ""`) means:
- Cannot unset a name or description
- Cannot move a task back to root once it has a parent

**Impact:** Limited flexibility in task updates; users may need workarounds.

**Suggested Fix:**
- Consider explicit "clear" sentinel value (e.g., `-`)
- Or use a separate flag to indicate "clear this field"
- Document this implicit contract in CLI help

---

### `delete` Command Advertised but Unimplemented `[Medium]`
**Location:** `main.go`

**Description:** `delete` is listed in the usage banner and ROADMAP marks it incomplete, but there's no `case "delete"` in the switch statement.

**Impact:** Users see "delete" as an available command but get "Unknown command" error when trying to use it.

**Suggested Fix:**
- Either implement the delete command
- Or remove it from the usage banner until implemented

---

### `UpdateTask` API Design `[Low]`
**Location:** `worklog.go` - `UpdateTask()` signature

**Description:** Four positional string arguments is hard to read at call sites and doesn't scale well.

**Impact:** Code readability, future extensibility.

**Suggested Fix:**
- Use an options struct (e.g., `TaskUpdate{ Name: "...", ParentUUID: "..." }`)
- Or use functional options pattern

---

### Brittle Path Munging in `Save()` `[Low]`
**Location:** `worklog.go` - `Save()` function

**Description:** `strings.Replace(path, ".ics", "-output.ics", 1)` only replaces first occurrence. Paths like `backup.ics.old.ics` produce unexpected results.

**Impact:** Edge case path handling issues.

**Suggested Fix:**
- Use `strings.TrimSuffix` with `filepath.Ext()`
- Or use `filepath` package properly: `strings.TrimSuffix(path, filepath.Ext(path)) + "-output.ics"`

---

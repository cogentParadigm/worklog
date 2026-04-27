# Known Issues

This document tracks known issues identified during code review, organized by priority.

## High Priority

### H1: Cycle Risk in Parent Reassignment
**Location:** `worklog.go` - `UpdateTask()` function

**Description:** `UpdateTask` does not guard against setting a task's parent to itself (`parentUUID == uuid`), nor does it prevent creating deeper cycles (e.g., making a task the child of its own descendant). Both would corrupt the tree and break `FindTaskByUUID` (infinite recursion) and serialization.

**Impact:** Application crash or infinite loop when processing corrupted task trees.

**Suggested Fix:** 
- Add check for `parentUUID == uuid` (self-parenting)
- Add cycle detection (ensure new parent is not a descendant of the task being moved)
- Consider extracting tree removal logic into a helper with cycle detection built-in

---

### H2: Inconsistent Error Handling Styles
**Location:** Across codebase (`worklog.go`, `main.go`, `errors.go`, `ical.go`)

**Description:** Mixed error handling patterns:
- `UpdateTask` returns errors to caller (good)
- `NewTask`, `Save`, calendar I/O use `handleError()` which prints but doesn't exit
- `update` branch in `main.go` prints "Error: ..." and exits with `1`
- `default` branch prints "Unknown command ..." but doesn't prefix with "Error:"

**Impact:** Unpredictable failure modes, some errors may be silently swallowed or handled inconsistently.

**Suggested Fix:** 
- Create unified error-print-and-exit helper
- Ensure all CLI errors follow consistent format and exit codes

---

### H3: `create` Command Parent Flag is No-Op
**Location:** `main.go` - `create` command case

**Description:** The `-parent` flag is parsed for the `create` command but the block is empty:
```go
if *createParent != "" {
    // Nothing happens here
}
```

**Impact:** Users can pass `-parent` but it has no effect, leading to confusion.

**Suggested Fix:** Implement parent assignment logic (similar to `UpdateTask` parent handling) or remove the flag until implemented.

---

### H4: KTimeTracker Metadata Stripping (KTimeTracker Compatibility Broken)
**Location:** `task.go` - `makeTodoForTask()` and `makeTaskForTodo()` functions

**Description:** The serialization round-trip strips KTimeTracker-specific metadata from the source `.ics` file:
- `CREATED`, `DTSTAMP`, `LAST-MODIFIED`
- `PERCENT-COMPLETE`
- `X-KDE-ktimetracker-totalSessionTime`
- `X-KDE-ktimetracker-totalTaskTime`

This breaks KTimeTracker compatibility — if you open the output `.ics` in KTimeTracker, you'd lose creation dates, completion status, and tracked time.

**Impact:** Loss of historical time tracking data; incompatible with KTimeTracker workflows.

**Suggested Fix:** 
- Extend `Task` struct to include metadata fields
- Preserve all properties in `makeTaskForTodo()`
- Emit all properties in `makeTodoForTask()`
- Consider using a `map[string]string` for arbitrary property preservation

---

### H5: Task Ordering Instability (Non-Deterministic Output)
**Location:** `task.go` - `makeTasksForTodos()` function

**Description:** The function iterates over a Go map (`uidMap`) in randomized order, causing tasks to appear in non-deterministic order in the output. Combined with `RELATED-TO` links changing during iteration, this produces different file layouts on every run even when nothing meaningful changes.

**Impact:** Noisy diffs; makes it impossible to use file comparison for regression testing; confusing for users expecting stable output.

**Suggested Fix:** 
- Sort tasks by UID (or other stable key) before iterating
- Or maintain original file order by tracking sequence during parse
- Consider stable tree serialization (e.g., depth-first pre-order with stable child ordering)

---

## Medium Priority

### M1: `relatedTo` vs `parent` Inconsistency
**Location:** `worklog.go`, `task.go`

**Description:** `UpdateTask` sets both `task.parent = newParent` and `task.relatedTo = parentUUID`. However, `makeTodoForTask` only looks at `task.parent` when emitting `RELATED-TO`, making the `relatedTo` write potentially dead code. During load, `makeTasksForTodos` treats `relatedTo` as the source of truth.

**Impact:** Potential confusion about which field is authoritative; risk of data inconsistency if `parent` and `relatedTo` diverge.

**Suggested Fix:** 
- Clarify the relationship between these fields
- Consider removing `relatedTo` field and deriving it from `parent` during serialization
- Or ensure both fields stay synchronized

---

### M2: Cannot Clear/Unset Fields
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

### M3: No Tests for New Code
**Location:** `worklog.go`

**Description:** `FindTaskByUUID` and `UpdateTask` have zero unit tests. Only existing test is `TestItCanSaveIcsFiles` which tests round-tripping, not tree manipulation or CLI commands.

**Impact:** Regression risk; no safety net for refactoring.

**Suggested Fix:** 
- Add unit tests for `FindTaskByUUID` (flat and nested tasks)
- Add unit tests for `UpdateTask` (name, description, parent changes)
- Add tests for edge cases: moving to root, moving to invalid parent, etc.

---

### M4: `delete` Command Advertised but Unimplemented
**Location:** `main.go`

**Description:** `delete` is listed in the usage banner and ROADMAP marks it incomplete, but there's no `case "delete"` in the switch statement.

**Impact:** Users see "delete" as an available command but get "Unknown command" error when trying to use it.

**Suggested Fix:** 
- Either implement the delete command
- Or remove it from the usage banner until implemented

---

## Low Priority

### L1: `UpdateTask` API Design
**Location:** `worklog.go` - `UpdateTask()` signature

**Description:** Four positional string arguments is hard to read at call sites and doesn't scale well.

**Impact:** Code readability, future extensibility.

**Suggested Fix:** 
- Use an options struct (e.g., `TaskUpdate{ Name: "...", ParentUUID: "..." }`)
- Or use functional options pattern

---

### L2: Duplicate Tree Removal Logic
**Location:** `worklog.go` - `UpdateTask()` function

**Description:** The slice-splicing logic for removing a task appears twice:
- Lines 63-66: removing from old parent's children
- Lines 72-75: removing from root list

**Impact:** Code duplication; harder to maintain and add cycle detection.

**Suggested Fix:** 
- Extract to `removeChild(parent, task)` helper
- Or add `Task.RemoveFromParent()` method

---

### L3: Brittle Path Munging in `Save()`
**Location:** `worklog.go` - `Save()` function

**Description:** `strings.Replace(path, ".ics", "-output.ics", 1)` only replaces first occurrence. Paths like `backup.ics.old.ics` produce unexpected results.

**Impact:** Edge case path handling issues.

**Suggested Fix:** 
- Use `strings.TrimSuffix` with `filepath.Ext()`
- Or use `filepath` package properly: `strings.TrimSuffix(path, filepath.Ext(path)) + "-output.ics"`

---

### L4: Test Data File Churn (Resolved: Gitignore)
**Location:** `testdata/example-output.ics`

**Description:** ~~The diff shows regenerated UUIDs, reordered tasks, a missing `My New Task`, and shifted `RELATED-TO` links. If this is meant to be committed as reference output, the instability is concerning. If it's an artifact of running the CLI, it shouldn't be committed.~~

**Resolution:** This file is a generated artifact and should not be tracked. Added to `.gitignore` and removed from git.

**Underlying causes (separate issues tracked as H4 and H5):**
- Metadata stripping (H4)
- Task ordering instability (H5)

**Action Taken:** 
- Added `testdata/example-output.ics` to `.gitignore`
- Removed from git tracking with `git rm --cached`

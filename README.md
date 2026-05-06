# Worklog

A CLI worklog and time tracking tool compatible with [KTimeTracker](https://apps.kde.org/ktt/), with an extensible architecture for integrations.

> ⚠️ **Work in Progress**: This tool is actively being developed. Core functionality works, but some features are incomplete.

## Overview

Worklog reads and writes standard iCalendar (`.ics`) files, making it compatible with KTimeTracker and other calendar applications. It provides a simple command-line interface for managing tasks and time entries, with plans for integrations to external systems like Jira.

## Features

- **KTimeTracker Compatible**: Reads and writes standard `.ics` files with VTODO (tasks) and VEVENT (time entries)
- **Hierarchical Tasks**: Supports parent-child task relationships
- **Simple CLI**: List, create, update, and delete tasks from the command line
- **Time Entry Management**: Add, list, edit, and delete manual time entries with duration auto-recompute
- **Timesheet Reports**: Generate daily timesheets by task with table and CSV output
- **Extensible**: Architecture supports plugins/integrations for external time tracking systems

## Installation

Requires Go 1.20 or later.

```bash
go install github.com/cogentParadigm/worklog@latest
```

Or clone and build:

```bash
git clone https://github.com/cogentParadigm/worklog.git
cd worklog
go build .
```

## Quick Start

```bash
# List all tasks (sorted alphabetically, showing hierarchy)
worklog list

# Create a new task
worklog create -name "Project Setup" -description "Initial configuration"

# Create a subtask
worklog create -name "Configure database" -parent <parent-uuid>
```

## Usage

### `list`

Lists all tasks sorted alphabetically by name, displaying the parent/child hierarchy.

```bash
worklog list
```

### `create`

Creates a new task with an auto-generated UUID.

**Flags:**
- `-name` — The name/summary of the task.
- `-description` — Additional detailed description.
- `-parent` — UUID of the parent task under which to create this task (optional).

**Examples:**
```bash
worklog create -name "Project Setup" -description "Initial configuration"
worklog create -name "Configure database" -parent <parent-uuid>
```

### `update`

Updates an existing task. Only the fields you provide are changed.

**Flags:**
- `-uuid` — The UUID of the task to update (required).
- `-name` — New name for the task.
- `-description` — New description for the task.
- `-parent` — New parent UUID for the task. Set to an empty string to move the task to the root level.

Cycle detection prevents a task from being set as its own parent or moved under one of its descendants.

**Examples:**
```bash
worklog update -uuid <uuid> -name "Updated Name"
worklog update -uuid <uuid> -description "New details"
worklog update -uuid <uuid> -parent ""
```

### `delete`

Deletes a task and **all of its subtasks recursively**.

**Flags:**
- `-uuid` — The UUID of the task to delete (required).
- `-force` — Delete without interactive confirmation.

Without `-force`, you will be prompted to confirm the deletion and shown the total number of tasks (including subtasks) that will be removed.

**Examples:**
```bash
worklog delete -uuid <uuid>
worklog delete -uuid <uuid> -force
```

### `time add`

Adds a manual time entry for a task. If `-start` is omitted, the start time is computed as `now - duration`, matching KTimeTracker behavior.

**Flags:**
- `-task` — UUID of the task to log time against (required).
- `-duration` — Duration to log. Accepts Go duration strings (`30m`, `1h30m`, `3600s`) or raw seconds (required).
- `-start` — Start datetime. Optional formats: `2023-08-14T09:00:00`, `2023-08-14 09:00:00`, `09:00:00`, `09:00` (defaults to now - duration).
- `-note` — Note for the time entry (optional, defaults to the task's name).

**Examples:**
```bash
# Log 30 minutes ending now
worklog time add -task <uuid> -duration 30m

# Log 1 hour starting at a specific time
worklog time add -task <uuid> -duration 1h -start "2023-08-14 09:00:00"

# Log with a custom note
worklog time add -task <uuid> -duration 3600s -note "Fixed authentication bug"
```

### `time list`

Lists time entries sorted by start time (most recent first). Optionally filter to a specific task.

**Flags:**
- `-task` — Filter to a specific task UUID (optional).

**Example:**
```bash
worklog time list
worklog time list -task <uuid>
```

### `time edit`

Edits an existing time entry. Only provided fields are changed. Duration is automatically recomputed when start or end is modified.

**Flags:**
- `-uuid` — UUID of the time entry to edit (required).
- `-start` — New start time.
- `-end` — New end time.
- `-duration` — New duration (e.g., `30m`, `1h30m`).
- `-note` — New note.

**Examples:**
```bash
worklog time edit -uuid <event-uuid> -note "Updated description"
worklog time edit -uuid <event-uuid> -start "2023-08-14 10:00:00" -end "2023-08-14 11:30:00"
```

### `time delete`

Deletes a time entry.

**Flags:**
- `-uuid` — UUID of the time entry to delete (required).
- `-force` — Delete without interactive confirmation.

**Example:**
```bash
worklog time delete -uuid <event-uuid>
worklog time delete -uuid <event-uuid> -force
```

### `report timesheet`

Generates a timesheet showing time logged per task per day. Defaults to the current week (Monday–Sunday). Only tasks with direct time entries in the selected range are shown.

**Flags:**
- `-from` — Start date (`YYYY-MM-DD`, defaults to Monday of current week).
- `-to` — End date (`YYYY-MM-DD`, defaults to Sunday of current week).
- `-all` — Use the full date range of all events in the file (ignores `-from`/`-to`).
- `-hide-empty` — Hide day columns that have no time entries.
- `-format` — Output format: `table` (default) or `csv`.
- `-decimal` — Display hours in decimal format (e.g., `1.50`) instead of `1h30m`.

**Examples:**
```bash
# Current week in table format
worklog report timesheet

# Custom date range
worklog report timesheet -from 2023-08-01 -to 2023-08-15

# CSV export with decimal hours
worklog report timesheet -format csv -decimal

# View all data, skipping empty days
worklog report timesheet -all -hide-empty
```

## File I/O

The tool currently reads from `testdata/example.ics` and writes to `testdata/example-output.ics`. This path is temporary and will become configurable in a future release.

All existing VEVENT components (KTimeTracker timer sessions), calendar-level properties, and any unknown iCalendar components are preserved exactly across saves.

## KTimeTracker Compatibility

Worklog uses the standard iCalendar format:

- **VTODO components** represent tasks with SUMMARY, DESCRIPTION, UID, and RELATED-TO (for parent relationships)
- **VEVENT components** represent time entries with DTSTART/DTEND, RELATED-TO (linking to tasks), and KTimeTracker-specific extensions

Your existing KTimeTracker data will be preserved and readable by this tool.

## Integrations

Worklog is designed with an extensible integration system. Planned integrations include:

- **Jira**: Log time entries as worklogs with preview/confirmation
- Additional integrations can be added via the plugin architecture

See [ROADMAP.md](ROADMAP.md) for details on planned features.

## Project Structure

- `main.go` - CLI entry point and command routing
- `worklog.go` - Core Worklog struct and persistence
- `task.go` - Task domain model and iCalendar conversion
- `event.go` - Event/time entry model and iCalendar conversion
- `report.go` - Report generation and formatting
- `ical.go` - iCalendar file I/O utilities
- `*_test.go` - Unit tests
- `testdata/` - Sample iCalendar files used for development and testing

## License

This project is licensed under the GNU General Public License v3.0 — see the [LICENSE](LICENSE) file for details.

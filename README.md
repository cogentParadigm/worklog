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
# Set the default file path via environment variable
export WORKLOG_FILE=~/my-tasks.ics

# List all tasks (sorted alphabetically, showing hierarchy)
worklog task list

# Create a new task (writes back to the same file by default)
worklog task create -name "Project Setup" -description "Initial configuration"

# Create a subtask
worklog task create -name "Configure database" -parent <parent-uuid>

# Use a specific file for a single command
worklog task list -file ~/other-tasks.ics

# Save changes to a different file
worklog task create -file ~/source.ics -output ~/backup.ics -name "Backup task"
```

## Usage

### `task list`

Lists all tasks sorted alphabetically by name, displaying the parent/child hierarchy, task UUID, direct duration, and rolled-up total duration.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE` environment variable).

```bash
worklog task list
worklog task list -file ~/my-tasks.ics
```

### `task create`

Creates a new task with an auto-generated UUID.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE` environment variable).
- `-output` — Output path for the updated `.ics` file. If omitted, writes back to the input file.
- `-name` — The name/summary of the task.
- `-description` — Additional detailed description.
- `-parent` — UUID of the parent task under which to create this task (optional).

**Examples:**
```bash
worklog task create -name "Project Setup" -description "Initial configuration"
worklog task create -file ~/tasks.ics -name "Configure database" -parent <parent-uuid>
worklog task create -file ~/source.ics -output ~/backup.ics -name "Backup task"
```

### `task update`

Updates an existing task. Only the fields you provide are changed.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE` environment variable).
- `-output` — Output path for the updated `.ics` file. If omitted, writes back to the input file.
- `-uuid` — The UUID of the task to update (required).
- `-name` — New name for the task.
- `-description` — New description for the task.
- `-parent` — New parent UUID for the task. Set to an empty string to move the task to the root level.

Cycle detection prevents a task from being set as its own parent or moved under one of its descendants.

**Examples:**
```bash
worklog task update -uuid <uuid> -name "Updated Name"
worklog task update -file ~/tasks.ics -uuid <uuid> -description "New details"
worklog task update -uuid <uuid> -parent ""
```

### `task delete`

Deletes a task and **all of its subtasks recursively**.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE` environment variable).
- `-output` — Output path for the updated `.ics` file. If omitted, writes back to the input file.
- `-uuid` — The UUID of the task to delete (required).
- `-force` — Delete without interactive confirmation.

Without `-force`, you will be prompted to confirm the deletion and shown the total number of tasks (including subtasks) that will be removed.

**Examples:**
```bash
worklog task delete -uuid <uuid>
worklog task delete -file ~/tasks.ics -uuid <uuid> -force
```

### `time add`

Adds a manual time entry for a task. If `-start` is omitted, the start time is computed as `now - duration`, matching KTimeTracker behavior.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE` environment variable).
- `-output` — Output path for the updated `.ics` file. If omitted, writes back to the input file.
- `-task` — UUID of the task to log time against (required).
- `-duration` — Duration to log. Accepts Go duration strings (`30m`, `1h30m`, `3600s`) or raw seconds (required).
- `-start` — Start datetime. Optional formats: `2023-08-14T09:00:00`, `2023-08-14 09:00:00`, `09:00:00`, `09:00` (defaults to now - duration).
- `-comment` — Comment for the time entry (optional).

**Examples:**
```bash
# Log 30 minutes ending now
worklog time add -task <uuid> -duration 30m

# Log 1 hour starting at a specific time
worklog time add -file ~/tasks.ics -task <uuid> -duration 1h -start "2023-08-14 09:00:00"

# Log with a comment
worklog time add -task <uuid> -duration 3600s -comment "Reviewed with team"
```

### `time list`

Lists time entries sorted by start time (most recent first). Optionally filter to a specific task.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE` environment variable).
- `-task` — Filter to a specific task UUID (optional).

**Example:**
```bash
worklog time list
worklog time list -file ~/tasks.ics -task <uuid>
```

### `time edit`

Edits an existing time entry. Only provided fields are changed. Duration is automatically recomputed when start or end is modified.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE` environment variable).
- `-output` — Output path for the updated `.ics` file. If omitted, writes back to the input file.
- `-uuid` — UUID of the time entry to edit (required).
- `-start` — New start time.
- `-end` — New end time.
- `-duration` — New duration (e.g., `30m`, `1h30m`).
- `-comment` — New comment.

**Examples:**
```bash
worklog time edit -uuid <event-uuid> -comment "Updated comment"
worklog time edit -file ~/tasks.ics -uuid <event-uuid> -start "2023-08-14 10:00:00" -end "2023-08-14 11:30:00"
```

### `time delete`

Deletes a time entry.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE` environment variable).
- `-output` — Output path for the updated `.ics` file. If omitted, writes back to the input file.
- `-uuid` — UUID of the time entry to delete (required).
- `-force` — Delete without interactive confirmation.

**Example:**
```bash
worklog time delete -uuid <event-uuid>
worklog time delete -file ~/tasks.ics -uuid <event-uuid> -force
```

### `report timesheet`

Generates a timesheet showing time logged per task per day. Defaults to the current week (Monday–Sunday). Only tasks with direct time entries in the selected range are shown.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE` environment variable).
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
worklog report timesheet -file ~/tasks.ics -from 2023-08-01 -to 2023-08-15

# CSV export with decimal hours
worklog report timesheet -format csv -decimal

# View all data, skipping empty days
worklog report timesheet -all -hide-empty
```

## File I/O

Worklog reads and writes standard `.ics` files. By default, modifying commands save **in-place** back to the input file. Use the `-output` flag to write to a different path instead.

The input file is resolved in this order:
1. `-file` flag on the command.
2. `WORKLOG_FILE` environment variable.

If neither is set, the command exits with an error telling you to use one of the two options.

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

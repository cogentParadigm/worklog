# Worklog

A CLI worklog and time tracking tool compatible with [KTimeTracker](https://apps.kde.org/ktt/), with an extensible architecture for integrations.

> ⚠️ **Work in Progress**: This tool is actively being developed. Core functionality works, but some features are incomplete.

## Overview

Worklog reads and writes standard iCalendar (`.ics`) files, making it compatible with KTimeTracker and other calendar applications. It provides a simple command-line interface for managing tasks and time entries, with plans for integrations to external systems like Jira.

## Features

- **KTimeTracker Compatible**: Reads and writes standard `.ics` files with VTODO (tasks) and VEVENT (time entries)
- **Hierarchical Tasks**: Supports parent-child task relationships
- **Simple CLI**: List, create, update, and delete tasks from the command line
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
- `ical.go` - iCalendar file I/O utilities
- `*_test.go` - Unit tests
- `testdata/` - Sample iCalendar files used for development and testing

## License

This project is licensed under the GNU General Public License v3.0 — see the [LICENSE](LICENSE) file for details.

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

## License

[Your License Here]

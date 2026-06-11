# Worklog

A CLI worklog and time tracking tool using standard iCalendar (`.ics`) files, with a Jira/Tempo integration and a plugin architecture in progress.

> ⚠️ **Work in Progress**: This tool is actively being developed. Core functionality works, but some features are incomplete.

## Overview

Worklog reads and writes standard iCalendar (`.ics`) files, making it compatible with any calendar application that supports the format (including KTimeTracker). It provides a simple command-line interface for managing tasks and time entries, with plans for integrations to external systems like Jira.

## Features

- **Standard iCalendar Format**: Reads and writes standard `.ics` files with VTODO (tasks) and VEVENT (time entries)
- **Hierarchical Tasks**: Supports parent-child task relationships
- **Simple CLI**: List, create, update, and delete tasks from the command line
- **Time Entry Management**: Add, list, edit, and delete manual time entries with duration auto-recompute
- **Timesheet Reports**: Generate daily timesheets by task with table and CSV output
- **Jira/Tempo Sync**: Send time entries to Tempo Cloud with preview, dry-run, and confirmation
- **Plugin architecture (in progress)**: Jira/Tempo integration works today; a generic plugin framework is planned (see [ROADMAP.md](ROADMAP.md))

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
# Initialize configuration (creates ~/.config/worklog/config.yaml)
worklog init

# Or set the default file path via environment variable
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

All subcommands support the following flag:
- `-v` / `--verbose` — Enable verbose output (e.g., per-item sidecar restoration notices).

Example:
```bash
worklog task list --verbose
```

### `init`

Initializes the worklog configuration interactively or via flags. Detects existing `.ics` files (including common KTimeTracker locations) and suggests them as the default worklog file. Creates a skeleton `.ics` file if one does not exist.

**Flags:**
- `--worklog-file` — Default worklog `.ics` file path (skips interactive prompts when provided).
- `--tempo-base-url` — Tempo Cloud base URL (default: `https://api.tempo.io/4`).
- `--tempo-account-id` — Atlassian account ID.
- `--tempo-token` — Tempo API token.
- `--tempo-rounding` — Rounding steps for sync (e.g. `floor:1m,ceil:5m`).
- `--jira-base-url` — Jira base URL.
- `--jira-username` — Jira username.
- `--jira-token` — Jira API token.
- `--skip-tempo` — Skip Tempo configuration.
- `--skip-jira` — Skip Jira configuration.
- `--force` — Overwrite an existing config file.

**Examples:**
```bash
# Interactive wizard
worklog init

# Non-interactive
worklog init --worklog-file ~/tasks.ics
worklog init --worklog-file ~/tasks.ics --tempo-token pass:worklog/tempo-token --jira-username user@example.com --jira-token pass:worklog/jira-token
```

### `config path`

Prints the resolved path to `config.yaml`.

```bash
worklog config path
```

### `config get`

Retrieves a config value by dot-notation key. Secrets are masked by default.

**Flags:**
- `--show` — Reveal unmasked secret values (e.g., `tempo.token`).

```bash
worklog config get worklog_file
worklog config get tempo.token
worklog config get --show tempo.token
```

### `config set`

Updates a config value by dot-notation key and writes the file. Validates URLs and paths automatically.

```bash
worklog config set worklog_file ~/tasks.ics
worklog config set tempo.token pass:worklog/tempo-token
worklog config set tempo.account_id abc-123
```

### `task list`

Lists all tasks sorted alphabetically by name, displaying the parent/child hierarchy, task UUID, direct duration, and rolled-up total duration.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE` environment variable).
- `-search` — Filter tasks by case-insensitive search in name or description (shows matching tasks as a flat list).
- `-parent` — Show only the specified task and its descendants.

```bash
worklog task list
worklog task list -file ~/my-tasks.ics
worklog task list -search planning
worklog task list -parent <parent-uuid>
```

### `task create`

Creates a new task with an auto-generated UUID.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE` environment variable).
- `-output` — Output path for the updated `.ics` file. If omitted, writes back to the input file.
- `-name` — The name/summary of the task.
- `-description` — Additional detailed description.
- `-parent` — UUID of the parent task under which to create this task (optional).
- `-issue-key` — Explicit Jira issue key for this task (optional, overrides auto-detection).

**Examples:**
```bash
worklog task create -name "Project Setup" -description "Initial configuration"
worklog task create -file ~/tasks.ics -name "Configure database" -parent <parent-uuid>
worklog task create -file ~/source.ics -output ~/backup.ics -name "Backup task"
worklog task create -name "Review PR" -issue-key PROJ-123
```

### `task update`

Updates an existing task. Only the fields you provide are changed.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE` environment variable).
- `-output` — Output path for the updated `.ics` file. If omitted, writes back to the input file.
- `-uuid` — The UUID of the task to update (required unless `-i`).
- `-name` — New name for the task.
- `-description` — New description for the task.
- `-parent` — New parent UUID for the task. Set to an empty string to move the task to the root level.
- `-issue-key` — Explicit Jira issue key for this task. Set to an empty string to clear it.
- `-attr` — Tempo work attributes as comma-separated `key=value` pairs (e.g., `_WorkType_=Development`). Set to an empty string to clear all attributes.
- `-i` — Interactive mode: select a task from a numbered list and edit its name, description, issue key, Tempo attributes, and parent.
- `-search` — Filter tasks by case-insensitive search (interactive mode only).

Cycle detection prevents a task from being set as its own parent or moved under one of its descendants.

**Examples:**
```bash
worklog task update -uuid <uuid> -name "Updated Name"
worklog task update -file ~/tasks.ics -uuid <uuid> -description "New details"
worklog task update -uuid <uuid> -parent ""
worklog task update -uuid <uuid> -issue-key PROJ-456
worklog task update -uuid <uuid> -attr _WorkType_=Development
worklog task update -uuid <uuid> -attr _WorkType_=Development,_Billable_=Yes

# Interactive mode
worklog task update -i
worklog task update -i -search onboarding
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

Adds a manual time entry for a task. If `-start` is omitted, the start time is computed as `now - duration`.

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

Lists time entries sorted by start time (most recent first). Shows task, start/end times, duration, and comment. Supports filtering by task, date range, or search query.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE` environment variable).
- `-task` — Filter to a specific task UUID (optional).
- `-from` — Filter events starting on or after this date (`YYYY-MM-DD`, optional).
- `-to` — Filter events starting on or before this date (`YYYY-MM-DD`, optional).
- `-search` — Filter by case-insensitive search in task name, description, or comment (optional).

**Examples:**
```bash
worklog time list
worklog time list -file ~/tasks.ics -task <uuid>
worklog time list -from 2023-08-01 -to 2023-08-15
worklog time list -search meeting
worklog time list -task <uuid> -from 2023-08-01 -search review
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

Generates a timesheet showing time logged per task per day. Defaults to the current week (Monday–Sunday). Only tasks with direct time entries in the selected range are shown. The output includes a short task UUID column for easy cross-referencing with other commands.

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

### `jira search`

Search Jira issues using JQL. By default, keywords are automatically wrapped in a `text ~ "..."` JQL query, which searches summary, description, and comments. Use `--jql` to provide a raw JQL query instead.

**Flags:**
- `--jql` — Raw JQL query (overrides keyword search).

**Examples:**
```bash
# Keyword search (searches summary, description, and comments)
worklog jira search "onboarding refactor"

# Search by issue key (text search)
worklog jira search PROJ-123

# Raw JQL query
worklog jira search --jql 'project = PROJ AND status = "In Progress"'
```

### `jira resolve`

Resolves Jira issue keys to numeric IDs and caches them on tasks. This is useful for progressive testing of Jira connectivity before running a full sync. Only Jira configuration is required — Tempo credentials are not needed.

Issue keys are read from task names (auto-detected via regex) or the explicit `X-WORKLOG-ISSUE-KEY` property. Tasks that already have a cached `X-WORKLOG-ISSUE-ID` are skipped.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE`).
- `-output` — Output path for the updated `.ics` file. If omitted, writes back to the input file.
- `-task` — Resolve only a specific task UUID (optional).

**Examples:**
```bash
# Resolve all issue keys across all tasks
worklog jira resolve

# Resolve a single task
worklog jira resolve --task <uuid>
```

### `jira attributes`

Lists available Tempo work attributes and their allowed values. This is useful for discovering which attribute keys and values your Tempo instance expects.

**Examples:**
```bash
worklog jira attributes
```

### `jira sync`

Syncs time entries to Tempo Cloud (Jira). Entries are **merged by task and date** before sending: multiple small entries on the same day for the same task are summed into a single worklog, using the task description as the worklog comment. By default, only unsynced entries are sent.

Issue keys are auto-detected from task names (e.g., `PROJ-123`) or set explicitly via `-issue-key`. Before sending, issue keys are resolved to numeric Jira issue IDs via the Jira REST API. Resolved IDs are cached on the task as `X-WORKLOG-ISSUE-ID` so subsequent syncs skip the lookup.

Previews (dry-run output in both `list` and `timesheet` formats) include short task UUIDs for easy cross-referencing with other commands. The timesheet preview also shows the mapped **Issue Key** after the Total column, and appends `*` to duration cells when the task has a non-empty description (indicating the worklog will include a comment). Tasks with time entries but no resolvable issue key are listed under a "Skipped tasks" summary at the end of the preview.

**Tempo work attributes:** Work attributes (such as `_WorkType_`) can be sent with each worklog. Attributes are set per-task using `worklog task update -attr key=value`. They are also merged with any `tempo.attributes` defined in `config.yaml`, with task-level values taking precedence.

**Flags:**
- `-file` — Path to the `.ics` file (overrides `WORKLOG_FILE`).
- `-output` — Output path for the updated `.ics` file. If omitted, writes back to the input file.
- `-task` — Sync only a specific task UUID (optional).
- `-from` — Start date for sync range (`YYYY-MM-DD`, optional).
- `-to` — End date for sync range (`YYYY-MM-DD`, optional).
- `-dry-run` — Preview what would be synced without sending anything.
- `-force` — Sync without interactive confirmation.
- `-format` — Preview format: `list` (default) or `timesheet`.
- `-hide-empty` — Hide days with no time entries (`timesheet` format only).
- `-decimal` — Display hours in decimal format (`timesheet` format only).
- `-rounding` — Rounding steps: `floor/ceil/round:to[,...]` (overrides config; default is `round:1m`).
- `-i` — Interactive mode: before syncing, review each task that will be sent and edit its name, description, issue key, Tempo attributes, and parent. Edits are saved as you go.

**Examples:**
```bash
# Preview what would be sent (shows merged entries)
worklog jira sync --dry-run

# Sync all unsynced entries for a single task
worklog jira sync --task <uuid>

# Sync a specific date range
worklog jira sync --from 2026-05-01 --to 2026-05-07

# Sync without prompting
worklog jira sync --force

# Preview in timesheet grid format
worklog jira sync --dry-run --format timesheet

# Preview in timesheet format, hiding empty days and showing decimal hours
worklog jira sync --dry-run --format timesheet --hide-empty --decimal

# Interactive mode: review and edit tasks before syncing
worklog jira sync -i

# Sync with custom rounding (floor to minutes, then ceil to 5 minutes)
worklog jira sync --rounding "floor:1m,ceil:5m"
```

## Configuration

Worklog reads an optional YAML config file from the standard config directory:
- Linux: `~/.config/worklog/config.yaml`
- macOS: `~/Library/Application Support/worklog/config.yaml`

Override the directory with the `XDG_CONFIG_HOME` environment variable.

Use `worklog init` to create the initial config interactively, or `worklog config` to read and update values without opening an editor.

**Example `config.yaml`:**
```yaml
worklog_file: ~/tasks.ics

tempo:
  base_url: https://api.tempo.io/4
  token: my-api-token
  account_id: your-atlassian-account-id
  rounding:
    - step: floor
      to: 1m
    - step: ceil
      to: 5m
  attributes:
    _WorkType_: Development

jira:
  base_url: https://mycompany.atlassian.net
  username: user@example.com
  token: my-jira-token
```

**Credential security:** The `token` fields support a `pass:` prefix to read secrets from the [`pass`](https://www.passwordstore.org/) password store:
```yaml
tempo:
  token: "pass:worklog/tempo-token"
jira:
  token: "pass:worklog/jira-token"
```

Jira authentication uses Basic auth with `base64(username:token)`. Set `jira.username` to your Atlassian account email and `jira.token` to your Jira API token.

**Time rounding:** By default, durations sent to Tempo are rounded to the nearest minute (`round:1m`). You can customize this in `config.yaml` under `tempo.rounding` as a sequence of `floor`, `ceil`, or `round` steps, each with a duration like `1m`, `5m`, or `30s`. Rounding is applied after events are merged by task and date. Set `-rounding ""` on the command line or an empty array in config to disable rounding entirely.

## File I/O

Worklog reads and writes standard `.ics` files. By default, modifying commands save **in-place** back to the input file. Use the `-output` flag to write to a different path instead.

The input file is resolved in this order:
1. `-file` flag on the command.
2. `WORKLOG_FILE` environment variable.
3. `worklog_file` in `config.yaml`.

If none are set, the command exits with an error.

All existing VEVENT components (timer sessions from other apps), calendar-level properties, and any unknown iCalendar components are preserved exactly across saves.

**Sidecar file:** Worklog creates a `.worklog` sidecar file next to each `.ics` it saves (e.g., `tasks.ics.worklog`). This JSON file stores worklog-specific metadata such as cached Jira issue IDs, Tempo attributes, and sync fingerprints. If another application strips these properties when it saves the `.ics`, worklog automatically restores them from the sidecar on the next load and prints a notice to stderr.

## iCalendar Compatibility

Worklog uses the standard iCalendar format:

- **VTODO components** represent tasks with SUMMARY, DESCRIPTION, UID, and RELATED-TO (for parent relationships)
- **VEVENT components** represent time entries with DTSTART/DTEND, RELATED-TO (linking to tasks), and duration

Existing data from KTimeTracker and other iCalendar applications will be preserved and readable by this tool.

## Integrations

- **Jira/Tempo**: The `worklog jira sync` command sends time entries to Tempo Cloud. Issue keys are auto-detected from task names (e.g., `PROJ-123`) or set explicitly with the `-issue-key` flag.

The long-term goal is a generic plugin framework so that additional integrations (Toggl, Harvest, GitHub, etc.) can be added without modifying core packages. This is a work in progress — see [doc/plugin-architecture.md](doc/plugin-architecture.md) for the design and [ROADMAP.md](ROADMAP.md) for the timeline.

## Project Structure

- `main.go` - CLI entry point and command routing
- `worklog.go` - Core Worklog struct and persistence
- `task.go` - Task domain model and iCalendar conversion
- `event.go` - Event/time entry model and iCalendar conversion
- `report.go` - Report generation and formatting
- `ical.go` - iCalendar file I/O utilities
- `config.go` - Configuration file loading (`config.yaml`, `pass:` credential support)
- `jira.go` - Jira/Tempo domain helpers (issue key mapping, sync state)
- `cmd_jira.go` - Jira/Tempo CLI command implementation
- `internal/tempo/client.go` - Tempo Cloud REST API client
- `*_test.go` - Unit tests
- `testdata/` - Sample iCalendar files used for development and testing

## License

This project is licensed under the GNU General Public License v3.0 — see the [LICENSE](LICENSE) file for details.

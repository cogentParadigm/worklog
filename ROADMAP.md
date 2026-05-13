# Roadmap

## Phase 1: Core CLI (Current → Near-term)

Foundation for basic task management.

- [x] Read/write standard iCalendar (`.ics`) files
- [x] List tasks (tree view, sorted alphabetically)
- [x] Create tasks
- [x] Complete `update` command implementation
- [x] Complete `delete` command implementation
- [x] Clean up orphaned VEVENTs on task deletion
  - `delete` now drops both VTODOs and related VEVENT timer sessions
  - `Event` model with `time.Time` support added for Phase 2/4 timer functionality
  - `Save()` filters VEVENTs whose `RELATED-TO` matches deleted task UIDs
- [x] Fix parent flag handling in `create` command

## Phase 2: Time Management

Manual time entry and reporting before automation.

- [x] Add/edit time entries manually
  - `time add` with `-task`, `-duration`, `-start`, `-comment`
  - `time list` with optional `-task` filter
  - `time edit` with partial updates and auto-recompute
  - `time delete` with confirmation
  - Supports Go duration strings (`30m`, `1h30m`) and raw seconds
  - Duration auto-recompute on start/end changes
    - Standard iCalendar VEVENT serialization (DTSTAMP, CATEGORIES, TRANSP)
- [x] Generate time reports and summaries (daily, weekly, by task)
  - `report timesheet` command with daily task/duration matrix
  - Defaults to current week (Mon–Sun), supports custom `--from` / `--to` ranges
  - Flat task list (parents and children shown independently with direct time only)
- [x] Export reports to various formats (CSV, JSON, text)
  - `--format table` (default) and `--format csv`
  - `--decimal` flag for decimal hour display (e.g., 1.50)
- [x] Filter and query time entries by date range or task
  - `time list` with `-from`, `-to`, `-search`, and `-task` filters
  - `task list` with `-search` and `-parent` filters

## Phase 3: Integration System

Extensible architecture for external system integrations.

- [x] **Jira/Tempo Integration** (functional but not yet plugin-isolated)
  - Map tasks to Jira issues via auto-detection (`PROJ-123` from name) or explicit `X-WORKLOG-ISSUE-KEY` property
  - Send single task's time entries to Tempo Cloud worklogs (`worklog jira sync --task <uuid>`)
  - Batch send multiple tasks' time entries by date range (`worklog jira sync --from ... --to ...`)
  - **Confirmation workflow** before sending:
    - Interactive prompt showing summary of what will be sent
    - `--dry-run` flag to preview without actually sending
  - Jira API configuration via `~/.config/worklog/config.yaml` (URL, token with optional `pass:` prefix, account ID)
- [ ] **True Plugin Architecture**
  - Define `Plugin` interface with lifecycle hooks (commands, config, metadata)
  - Generic domain model: remove Jira/Tempo-specific methods from `Task`/`Event`
  - Generic sidecar extensions: plugin-namespaced metadata
  - Generic config registry: plugin-registered config keys
  - Command registration via plugin interface
  - Isolate Jira-specific CLI flags from core `task` commands
  - Isolate Jira/Tempo prompts from `init` command
- [ ] Framework documentation for community integrations

## Phase 4: Standalone Mode

Standalone timer and background tracking for users without a desktop time-tracking application.

- [ ] Timer functionality (start/stop tracking for active task)
- [ ] Background/ daemon mode for automatic time tracking
- [ ] Notification/reminder system
- [ ] Full timer parity with desktop time-tracking applications

## Phase 5: Future Considerations

Potential enhancements based on usage and feedback.

- [ ] Team collaboration features (shared worklogs)
- [ ] Additional integrations (GitHub, GitLab, Toggl, Harvest, etc.)
- [ ] Web or desktop GUI (beyond the current CLI focus)
- [ ] Mobile companion app

---

## Notes

- **Priority**: Phase 1 completion is required before Phase 3 begins
- **Compatibility**: All changes must maintain standard iCalendar (`.ics`) compatibility
- **Extensibility**: The integration system is designed to be generic, not Jira-specific
- **Architecture**: `Worklog` now stores the original parsed `*ics.Calendar` and rebuilds it on save, preserving VEVENTs, calendar properties, and any unknown components during round-trips. The `GetEvents()` method exposes preserved VEVENTs for Phase 2/4 timer functionality.

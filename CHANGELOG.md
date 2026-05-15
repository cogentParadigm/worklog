## [0.27.1](https://github.com/cogentParadigm/worklog/compare/v0.27.0...v0.27.1) (2026-05-15)


### Bug Fixes

* move emittedSet out of debug-only section ([0f944bc](https://github.com/cogentParadigm/worklog/commit/0f944bc3e1008155f4e397fd6790c2aca3de6f26))

# [0.27.0](https://github.com/cogentParadigm/worklog/compare/v0.26.0...v0.27.0) (2026-05-14)


### Bug Fixes

* print saved count after resolving skipped tasks ([edc2cb1](https://github.com/cogentParadigm/worklog/commit/edc2cb1db864faba354b19fa2fbb2631bf9910d5))
* print saved count only after successful save ([7f62e6e](https://github.com/cogentParadigm/worklog/commit/7f62e6e0d50a4fe18bf93f698bd3a4c894f1e886))
* switch SearchIssues from Issue Picker to JQL endpoint ([584550f](https://github.com/cogentParadigm/worklog/commit/584550f2548a9604d1f0e615c3b071c89c239804))


### Features

* add jira search command and resolve-skipped flag ([1bf4ff1](https://github.com/cogentParadigm/worklog/commit/1bf4ff170f550a12d0de9294af79efc558367d8e))
* add SearchIssues method to jira client ([266670c](https://github.com/cogentParadigm/worklog/commit/266670c6bfa28e985744f525de5cd9b0c98cdd97))
* add SetIssueKey and ClearIssueKey helpers ([15866d3](https://github.com/cogentParadigm/worklog/commit/15866d3010c481fa9e6f0283c6a6ff8e8c9834b7))

# [0.26.0](https://github.com/cogentParadigm/worklog/compare/v0.25.1...v0.26.0) (2026-05-13)


### Features

* add verbose flag for sidecar restore output ([7135347](https://github.com/cogentParadigm/worklog/commit/71353479c057f7b0d839f43b651eb2b6b4ffd285))

## [0.25.1](https://github.com/cogentParadigm/worklog/compare/v0.25.0...v0.25.1) (2026-05-12)


### Bug Fixes

* **jira:** compute duration from DTSTART/DTEND instead of KDE property ([cdb180a](https://github.com/cogentParadigm/worklog/commit/cdb180aca1a5b8da4e51a091d705cd029e6eaa5f))

# [0.25.0](https://github.com/cogentParadigm/worklog/compare/v0.24.0...v0.25.0) (2026-05-11)


### Features

* **jira:** use payload fingerprint for sync detection instead of LAST-MODIFIED ([b798278](https://github.com/cogentParadigm/worklog/commit/b7982788589ab01b8e57bc2059ba787ba1d786b1))

# [0.24.0](https://github.com/cogentParadigm/worklog/compare/v0.23.0...v0.24.0) (2026-05-09)


### Features

* **jira:** use task description as tempo worklog comment ([1215e48](https://github.com/cogentParadigm/worklog/commit/1215e482c85611631052c75a28a88e06b8fed056))

# [0.23.0](https://github.com/cogentParadigm/worklog/compare/v0.22.0...v0.23.0) (2026-05-09)


### Features

* improve jira sync preview and time list UX ([0ec6586](https://github.com/cogentParadigm/worklog/commit/0ec658610ce21336f860a49a48652a0daa806b5b))

# [0.22.0](https://github.com/cogentParadigm/worklog/compare/v0.21.0...v0.22.0) (2026-05-09)


### Features

* **jira:** show tempo attributes in sync preview ([14d65fc](https://github.com/cogentParadigm/worklog/commit/14d65fc589de7c37fe88f24f79db987d93dfb470))

# [0.21.0](https://github.com/cogentParadigm/worklog/compare/v0.20.0...v0.21.0) (2026-05-09)


### Features

* display short task UUIDs in report timesheet and jira sync previews ([ad738b5](https://github.com/cogentParadigm/worklog/commit/ad738b5b66bb8b18e204e320a7017e6a899efbd8))

# [0.20.0](https://github.com/cogentParadigm/worklog/compare/v0.19.0...v0.20.0) (2026-05-08)


### Features

* add short unique UUID prefix resolution and display ([8b4baec](https://github.com/cogentParadigm/worklog/commit/8b4baec17ed9d578929ca6023ec852ff2bb3a68c))

# [0.19.0](https://github.com/cogentParadigm/worklog/compare/v0.18.1...v0.19.0) (2026-05-08)


### Features

* add sidecar metadata file for KTimeTracker co-existence ([85de7df](https://github.com/cogentParadigm/worklog/commit/85de7dfb5461c267da2fb9be91b49e3db80095b4))

## [0.18.1](https://github.com/cogentParadigm/worklog/compare/v0.18.0...v0.18.1) (2026-05-08)


### Bug Fixes

* use KEY parameter for tempo attributes instead of dynamic property names ([35f0bf1](https://github.com/cogentParadigm/worklog/commit/35f0bf1a7d68b7c614492ce1ba26d1d8e4a81ba8))

# [0.18.0](https://github.com/cogentParadigm/worklog/compare/v0.17.0...v0.18.0) (2026-05-08)


### Features

* add tempo work attributes support ([236f097](https://github.com/cogentParadigm/worklog/commit/236f0971e833491c2fb17952a256b687cdc51c6a))

# [0.17.0](https://github.com/cogentParadigm/worklog/compare/v0.16.0...v0.17.0) (2026-05-08)


### Bug Fixes

* **jira:** require separate username and always use Basic auth ([f04c91b](https://github.com/cogentParadigm/worklog/commit/f04c91bee6f06012712ec2e4063d1698ae6e3386))


### Features

* add configurable rounding for jira sync ([22e8009](https://github.com/cogentParadigm/worklog/commit/22e800906eaa510835452906d6eb9e576fd78f48))
* add jira resolve subcommand ([06fdb2b](https://github.com/cogentParadigm/worklog/commit/06fdb2b53f86f5dbdc3f262125222f89fdf1c11f))
* **jira:** fetch and echo remaining estimates in tempo sync ([9db4f97](https://github.com/cogentParadigm/worklog/commit/9db4f9750e056676e1d7049752421565c302d5e7))
* resolve jira issue keys to numeric ids before sending to tempo ([3b73954](https://github.com/cogentParadigm/worklog/commit/3b73954c7d9f1bc4b8b718597cc1483785c7033c))

# [0.16.0](https://github.com/cogentParadigm/worklog/compare/v0.15.0...v0.16.0) (2026-05-07)


### Features

* update Tempo API client for v4 base path and payload format ([6b2bdc4](https://github.com/cogentParadigm/worklog/commit/6b2bdc4690ef8e2c556691572ff56c0b0be38507))

# [0.15.0](https://github.com/cogentParadigm/worklog/compare/v0.14.0...v0.15.0) (2026-05-07)


### Features

* add config management commands ([6e7b7de](https://github.com/cogentParadigm/worklog/commit/6e7b7dece4cb8adb96efbdc6fb7135de075f1800))

# [0.14.0](https://github.com/cogentParadigm/worklog/compare/v0.13.0...v0.14.0) (2026-05-07)


### Features

* add timesheet format option to jira sync preview ([256c983](https://github.com/cogentParadigm/worklog/commit/256c9836a74c81a9ba0bad4612e79a0e1a852ace))

# [0.13.0](https://github.com/cogentParadigm/worklog/compare/v0.12.0...v0.13.0) (2026-05-07)


### Features

* add Jira/Tempo sync with per-task per-day aggregation ([65fcffb](https://github.com/cogentParadigm/worklog/commit/65fcffbcee204731cccb7349ad8cddbacfbfbb71))
* add XDG config loading with optional pass: prefix support ([9c2963c](https://github.com/cogentParadigm/worklog/commit/9c2963c2e4a5ba5f32c62fdc2e5e1ac5553b8dfe))

# [0.12.0](https://github.com/cogentParadigm/worklog/compare/v0.11.1...v0.12.0) (2026-05-07)


### Features

* add filtering flags to time list and task list commands ([3adb3ef](https://github.com/cogentParadigm/worklog/commit/3adb3ef719d527ae090c140c0912dce328915f96))

## [0.11.1](https://github.com/cogentParadigm/worklog/compare/v0.11.0...v0.11.1) (2026-05-06)


### Bug Fixes

* use COMMENT property for VEVENT comment and expose -comment flag ([9ee7268](https://github.com/cogentParadigm/worklog/commit/9ee726811e18f76b8eff41a4b2e8d7a071f8ed94))

# [0.11.0](https://github.com/cogentParadigm/worklog/compare/v0.10.0...v0.11.0) (2026-05-06)


### Features

* add enhanced -h and --help support for all CLI commands ([3063ff4](https://github.com/cogentParadigm/worklog/commit/3063ff4a3aa96e7751be5efd770d9845892d26a7))

# [0.10.0](https://github.com/cogentParadigm/worklog/compare/v0.9.0...v0.10.0) (2026-05-06)


### Features

* move task CRUD commands under task namespace ([648a7a5](https://github.com/cogentParadigm/worklog/commit/648a7a50470a6f5dcd8d24fc4cf5c4f534008a56))

# [0.9.0](https://github.com/cogentParadigm/worklog/compare/v0.8.0...v0.9.0) (2026-05-06)


### Features

* add UUID and Duration columns to list command ([7b20d22](https://github.com/cogentParadigm/worklog/commit/7b20d225fabf5951a8afdbb8349cf61ee4a4e18d))

# [0.8.0](https://github.com/cogentParadigm/worklog/compare/v0.7.0...v0.8.0) (2026-05-06)


### Features

* add -file and -output flags with WORKLOG_FILE env var ([4f01b5c](https://github.com/cogentParadigm/worklog/commit/4f01b5c8e1004bdf5a84e960defae978a1280360))

# [0.7.0](https://github.com/cogentParadigm/worklog/compare/v0.6.0...v0.7.0) (2026-05-06)


### Features

* add timesheet report command ([6388816](https://github.com/cogentParadigm/worklog/commit/6388816d927871c9a55096e5aee43d0155568abb))

# [0.6.0](https://github.com/cogentParadigm/worklog/compare/v0.5.0...v0.6.0) (2026-05-06)


### Features

* add rolled-up time totals to task listing ([46ee9da](https://github.com/cogentParadigm/worklog/commit/46ee9dade733c9742e49c326be3ef5fb1bd894de))

# [0.5.0](https://github.com/cogentParadigm/worklog/compare/v0.4.2...v0.5.0) (2026-05-06)


### Features

* add time entry management commands ([536b0cd](https://github.com/cogentParadigm/worklog/commit/536b0cd4380bfa92605314479ec4a24c8da289af))

## [0.4.2](https://github.com/cogentParadigm/worklog/compare/v0.4.1...v0.4.2) (2026-04-30)


### Bug Fixes

* remove new task from root before attaching to parent in create ([7273169](https://github.com/cogentParadigm/worklog/commit/72731691ef2b88fe7aa6239bba2f1733c51662ba))

## [0.4.1](https://github.com/cogentParadigm/worklog/compare/v0.4.0...v0.4.1) (2026-04-30)


### Bug Fixes

* preserve property order in ICS component round-trip ([c2f1bcf](https://github.com/cogentParadigm/worklog/commit/c2f1bcf5cdb2d9ab102f09b251ad6b5f0671f9c0))

# [0.4.0](https://github.com/cogentParadigm/worklog/compare/v0.3.0...v0.4.0) (2026-04-30)


### Features

* add Event model and clean up orphaned VEVENTs on task deletion ([a6a52e9](https://github.com/cogentParadigm/worklog/commit/a6a52e9fd8a0aac59db9b6f2d2f2d07194c80ca1))

# [0.3.0](https://github.com/cogentParadigm/worklog/compare/v0.2.0...v0.3.0) (2026-04-30)


### Features

* implement delete command with interactive confirmation ([00aba5a](https://github.com/cogentParadigm/worklog/commit/00aba5a8cacc756c26dfe6ab4538eff89a438b9a))

# [0.2.0](https://github.com/cogentParadigm/worklog/compare/v0.1.1...v0.2.0) (2026-04-30)


### Features

* add TaskUpdate struct and support clearing fields in UpdateTask ([2a85ee7](https://github.com/cogentParadigm/worklog/commit/2a85ee7f9d77e2f4b8fe7866d8ee3c3a37400115))

## [0.1.1](https://github.com/cogentParadigm/worklog/compare/v0.1.0...v0.1.1) (2026-04-29)


### Bug Fixes

* restore git credentials for semantic-release push ([51d59a5](https://github.com/cogentParadigm/worklog/commit/51d59a57639f47b9ed93b5c09bcf3dd6256cbcae))
* use UTC in test to avoid timezone drift in CI ([840fe28](https://github.com/cogentParadigm/worklog/commit/840fe2817d981ef7e4e73a941a0f6e9435e42ba3))

# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2023-08-27

### Security

- **Cycle Risk in Parent Reassignment** - Added cycle detection to `UpdateTask()` to prevent:
  - Self-parenting (setting a task as its own parent)
  - Moving a task to become a child of its own descendant

  Previously, these operations would corrupt the task tree, causing infinite recursion in `FindTaskByUUID()` and breaking serialization. The function now returns descriptive errors when cycle attempts are detected.

### Changed

- **Extracted tree removal logic** - Duplicated logic for removing a task from its parent (previously repeated in `UpdateTask()`) is now consolidated into the `removeFromParent()` helper method. This improves maintainability and simplifies future enhancements.

### Added

- CLI for managing KTimeTracker-compatible `.ics` files
- `list` command with tree view, sorted alphabetically
- `create` command for adding new tasks
- `update` command for modifying task name, description, and parent
- Round-trip iCalendar preservation (preserves VEVENTs, calendar properties, and VTODO order)
- Comprehensive unit tests for `UpdateTask()` covering:
  - Name and description updates
  - Valid parent reassignment
  - Self-parenting rejection (cycle detection)
  - Moving to descendant rejection (cycle detection)
  - Error handling for non-existent tasks and parents
  - State consistency verification after failed updates
- Unit tests for `FindTaskByUUID()` covering flat and nested task lookup
- Unit tests for `isDescendantOf()` helper function
- Unit tests for `removeFromParent()` helper function

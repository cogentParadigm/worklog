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

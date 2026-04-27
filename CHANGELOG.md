# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Security

- **H1: Cycle Risk in Parent Reassignment** - Added cycle detection to `UpdateTask()` to prevent:
  - Self-parenting (setting a task as its own parent)
  - Moving a task to become a child of its own descendant
  
  Previously, these operations would corrupt the task tree, causing infinite recursion in `FindTaskByUUID()` and breaking serialization. The function now returns descriptive errors when cycle attempts are detected.

### Changed

- **L2: Extracted tree removal logic** - Duplicated logic for removing a task from its parent (previously repeated in `UpdateTask()`) is now consolidated into the `removeFromParent()` helper method. This improves maintainability and simplifies future enhancements.

### Added

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


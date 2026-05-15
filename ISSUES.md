# Known Issues

This document tracks known issues identified during code review.



- **`humanizeLabel` does complex rune-by-rune scanning**: `cmd_jira.go:798-813` is ~15 lines to insert spaces before capitals. Could be simplified with a regex or string replacement.

- **Core domain model polluted with Jira/Tempo-specific methods**: `jira.go` adds `IssueKey()`, `IssueID()`, `TempoAttributes()`, `SyncedAt()`, `SyncHash()`, etc. as methods on `Task` and `Event`. A 3rd-party plugin cannot add metadata to core types without modifying the main package. These should become generic property accessors or standalone helper functions in the plugin's own package.

- **Sidecar format hardcoded to Jira/Tempo metadata**: `sidecar.go` defines `TaskMeta.IssueID`, `TaskMeta.IssueKey`, `TaskMeta.TempoAttrs`, and `EventMeta.SyncedAt`/`SyncHash` as literal fields. There is no generic extension bucket where a plugin could stash its own metadata. Should be refactored to plugin-namespaced metadata (e.g., `Extensions map[string]json.RawMessage`).

- **Config system hardcoded to specific integrations**: `config.go` has literal `Tempo` and `Jira` structs on `Config`, and `Get`/`Set`/`isValidConfigKey` use hardcoded switch statements for `tempo.*` and `jira.*`. A new plugin cannot register its own config keys without editing the core config package.

- **CLI command dispatch hardcoded in main.go**: The top-level command map in `main.go` (`commands := map[string]func...`) and sub-command maps (`runTask`, `runJira`, etc.) are statically defined. There is no registration hook for a 3rd-party plugin to add new top-level commands or subcommands.

- **Core task commands and init wizard polluted with Jira/Tempo concepts**: `task create` and `task update` expose `--issue-key` and `--attr` flags. `init` has `--tempo-*`/`--jira-*` flags and interactive prompts. Core commands should remain iCalendar-neutral; plugin-specific flags and prompts should live in plugin commands or hooks.

---

*Last reviewed: 2026-05-13* 

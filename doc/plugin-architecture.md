# Plugin Architecture Design Document

## Goals

The Jira/Tempo integration should behave like an **optional plugin** that is:

1. **Optionally enabled** — worklog functions fully without it.
2. **Isolated from the core application** — the domain model, config system, sidecar, and CLI dispatcher know nothing about Jira or Tempo.
3. **Extensible by 3rd parties** — a developer can write a new integration (e.g., GitHub, Toggl, Harvest) as a standalone Go module and include it in a custom Worklog build by importing the module and registering its plugin in `main.go`. Core packages are never modified.

## Current State

| Area | Status | Notes |
|---|---|---|
| HTTP clients | ✅ Isolated | `internal/jira` and `internal/tempo` are clean, standalone packages. |
| Runtime optional | ✅ Works without config | Core `task`/`time`/`report` flows never call Jira/Tempo APIs. |
| Domain model | ❌ Polluted | `Task.IssueKey()`, `Event.SyncedAt()`, etc. live in `jira.go` (main package). |
| Sidecar | ❌ Hardcoded | `TaskMeta` has literal `IssueID`/`IssueKey`/`TempoAttrs` fields. |
| Config | ❌ Hardcoded | `Config` struct embeds `Tempo` and `Jira`; `Get`/`Set` use hardcoded switches. |
| CLI commands | ❌ Hardcoded | `runJira` is wired directly into `main.go`'s command map. |
| Task CLI | ❌ Polluted | `task create`/`update` expose `--issue-key` and `--attr` flags. |
| Init wizard | ❌ Polluted | `init` has `--tempo-*`/`--jira-*` flags and interactive prompts. |

## Target Architecture

### 1. Plugin Interface

Introduce a `pkg/plugin` package that defines the contract every integration must implement.

```go
package plugin

import (
	"encoding/json"

	"github.com/cogentParadigm/worklog/pkg/core"
)

type Plugin interface {
    // Unique plugin name (used as namespace in config and sidecar).
    Name() string

    // Register CLI commands under this plugin's namespace.
    // e.g., a plugin named "jira" adds the "jira" top-level command
    // with subcommands "sync", "resolve", "attributes".
    Commands() []Command

    // Register config keys this plugin reads/writes.
    // The registry validates and stores values generically.
    RegisterConfig(r *ConfigRegistry)

    // Called when a Worklog is loaded so the plugin can attach
    // metadata helpers or hooks to tasks/events.
    // (Optional; most plugins can read raw properties directly.)
    OnLoad(ctx *LoadContext) error
}

type Command struct {
    Name        string
    Description string
    Handler     func(args []string) error
}

// SidecarProvider is an optional sub-interface for plugins that store
// metadata in the sidecar file. Plugins that don't need sidecar persistence
// can omit this.
type SidecarProvider interface {
    BuildSidecarTaskMeta(task *core.Task) (json.RawMessage, error)
    RestoreSidecarTaskMeta(task *core.Task, data json.RawMessage) error
    BuildSidecarEventMeta(event *core.Event) (json.RawMessage, error)
    RestoreSidecarEventMeta(event *core.Event, data json.RawMessage) error
}
```

Plugins are registered at compile time in `main.go`:

```go
import (
    "github.com/cogentParadigm/worklog/pkg/plugin"
    "github.com/cogentParadigm/worklog/internal/jira"
)

func main() {
    registry := plugin.NewRegistry()
    registry.Register(jira.NewPlugin()) // single registration line
    // ...
}
```

### 2. Generic Domain Model

Remove all Jira/Tempo-specific methods from `Task` and `Event`. The core types expose **only** generic iCalendar property accessors:

```go
// In core task/event packages
func (t *Task) GetProperty(token string) string
func (t *Task) SetProperty(token, value string)
func (t *Task) RemoveProperty(token string)
```

The Jira plugin defines its own helpers in its own package:

```go
// In internal/jira/helpers.go
func IssueKey(task *core.Task) string {
    if v := task.GetProperty("X-WORKLOG-ISSUE-KEY"); v != "" {
        return v
    }
    return extractFromName(task.Name())
}

func SetIssueID(task *core.Task, id string) {
    task.SetProperty("X-WORKLOG-ISSUE-ID", id)
}
```

This pattern lets 3rd parties add arbitrary `X-WORKLOG-*` (or even standard) properties without touching core code.

### 3. Generic Sidecar Extensions

Refactor `TaskMeta` and `EventMeta` to include a plugin-namespaced extension bucket:

```go
type Sidecar struct {
    Version    int                    `json:"version"`
    LastHash   string                 `json:"last_ics_hash,omitempty"`
    ModifiedAt string                 `json:"modified_at,omitempty"`
    Tasks      map[string]TaskMeta    `json:"tasks"`
    Events     map[string]EventMeta   `json:"events"`
}

type TaskMeta struct {
    Extensions map[string]json.RawMessage `json:"extensions,omitempty"`
}

type EventMeta struct {
    Extensions map[string]json.RawMessage `json:"extensions,omitempty"`
}
```

A plugin serializes its own metadata subset into its namespace:

```go
func (p *JiraPlugin) BuildSidecarTaskMeta(task *core.Task) (json.RawMessage, error) {
    m := jiraTaskMeta{
        IssueID:  IssueID(task),
        IssueKey: IssueKey(task),
    }
    return json.Marshal(m)
}
```

`buildSidecar` iterates registered plugins and collects their metadata per task/event. `restoreFromSidecar` does the reverse, letting each plugin hydrate its own metadata from its namespace.

### 4. Generic Config Registry

Keep the **top-level YAML shape** for backward compatibility (existing `jira:` and `tempo:` blocks continue to work), but replace the hardcoded `Config` struct with a generic registry:

1. **Core config** — only `worklog_file` and other truly universal settings remain as typed struct fields.
2. **Plugin config** — a `map[string]interface{}` that preserves the current YAML layout. Plugins register their keys with the registry, which handles validation, defaults, and sensitive-key masking.

```go
type Config struct {
    WorklogFile string                 `yaml:"worklog_file"`
    Plugins     map[string]interface{} `yaml:",inline"` // preserves jira:, tempo:, etc.
}
```

Plugins register their schema and defaults:

```go
func (p *JiraPlugin) RegisterConfig(r *plugin.ConfigRegistry) {
    r.RegisterString("jira.base_url", "")
    r.RegisterString("jira.username", "")
    r.RegisterSecret("jira.token", "")
}
```

`worklog config get jira.base_url` and `worklog config set jira.base_url ...` work generically via the registry, not through a hardcoded switch. Existing user configs remain valid with no migration required.

### 5. CLI Registration

`main.go` iterates registered plugins to build the command map dynamically:

```go
commands := map[string]func([]string) error{
    "task":   runTask,
    "time":   runTime,
    "report": runReport,
    "config": runConfig,
    "init":   runInit,
}
for _, p := range registry.Plugins() {
    for _, cmd := range p.Commands() {
        commands[cmd.Name] = cmd.Handler
    }
}
```

Plugin commands live in the plugin's own package (e.g., `internal/jira/cmd.go`).

### 6. Clean Up Core Commands

- **`task create`/`update`**: Remove `--issue-key` and `--attr`. Add a generic `--property` flag instead:
  ```
  worklog task create --name "Foo" --property X-WORKLOG-ISSUE-KEY=PROJ-123
  ```
  Alternatively, move key assignment to a plugin subcommand:
  ```
  worklog jira assign-key --task <uuid> --key PROJ-123
  ```

- **`init`**: Remove `--tempo-*` and `--jira-*` flags and interactive prompts. Provide a minimal, generic init. Plugins can expose their own init helpers:
  ```
  worklog jira init
  worklog tempo init
  ```

## Migration Sequence

The refactor should happen in this order to keep the application working at every commit:

1. **Extract core types into an importable package** (e.g., `pkg/core` or `pkg/worklog`).
   - Move `Task`, `Event`, `Worklog`, and generic property helpers out of `package main`.
   - Export fields and methods that plugins need (`UUID()`, `Name()`, `GetProperty()`, `SetProperty()`, etc.).
   - Update all internal references to use the new package.
2. **Introduce the `pkg/plugin` interface** (no-op at first).
3. **Create a `PluginRegistry`** and register the Jira integration as the first plugin. Keep `main.go` working by delegating to the registry while still keeping the hardcoded map temporarily.
4. **Genericize the domain model**:
   - Add `GetProperty`/`SetProperty` to `Task`/`Event` (or export the existing helpers).
   - Convert `jira.go` methods to standalone helpers in `internal/jira/helpers.go`.
   - Update all call sites in `cmd_jira.go`.
5. **Genericize the sidecar**:
    - Add `Extensions` map to `TaskMeta` and `EventMeta`.
    - Move Jira metadata into `taskMeta.Extensions["jira"]` / `eventMeta.Extensions["jira"]`.
    - Update `buildSidecar`/`restoreFromSidecar` to iterate plugins.
6. **Genericize the config system**:
    - Introduce `ConfigRegistry`.
    - Move `tempo.*` and `jira.*` keys into plugin registration.
    - Update `config get`/`set` to use the registry.
7. **Remove hardcoded CLI wiring**:
    - Move `runJira*` functions into `internal/jira/cmd.go`.
    - Delete the static command entries in `main.go`.
8. **Clean core commands**:
    - Remove `--issue-key`/`--attr` from `task create`/`update`.
    - Remove Jira/Tempo prompts from `init`.
9. **Update documentation** and mark Phase 3 complete in `ROADMAP.md`.

## Open Questions

1. **Compile-time vs. runtime plugins**
   - *Decision*: Start with **compile-time registration** (plugins import themselves into `main.go`).
    - *Rationale*: Go's `plugin` package (`.so` files) is platform-specific and brittle. Sidecar executables add IPC complexity. Compile-time registration lets 3rd parties write standalone plugin packages and distribute them as forked binaries. A custom build imports the plugin module and calls `registry.Register(...)` in `main.go`. Core packages are never edited.

2. **Should plugins hook into `task create`/`update` for custom flags?**
   - *Option A*: Generic `--property` flag only (simplest, keeps core neutral).
   - *Option B*: Plugin hook that lets a plugin inject flags into core commands (more ergonomic but couples core and plugin CLI parsing).
   - *Recommendation*: Start with Option A. Revisit Option B if user feedback demands it.

3. **Sidecar versioning per plugin**
   - Should each plugin manage its own sidecar schema version? Or is a single `Sidecar.Version` sufficient?
   - *Recommendation*: Start with a global version. Plugins handle their own internal migration via `json.RawMessage` unmarshaling.

## Appendix: Example Plugin Skeleton

```go
package toggl

import (
    "github.com/cogentParadigm/worklog/pkg/core"
    "github.com/cogentParadigm/worklog/pkg/plugin"
)

type Plugin struct{}

func NewPlugin() *Plugin { return &Plugin{} }

func (p *Plugin) Name() string { return "toggl" }

func (p *Plugin) Commands() []plugin.Command {
    return []plugin.Command{
        {
            Name:        "toggl",
            Description: "Sync time entries to Toggl Track",
            Handler:     runToggl,
        },
    }
}

func (p *Plugin) RegisterConfig(r *plugin.ConfigRegistry) {
    r.RegisterString("toggl.api_token", "")
    r.RegisterString("toggl.workspace_id", "")
}

func runToggl(args []string) error {
    // ... implementation ...
    return nil
}

// Helpers (isolated in this package, operating on generic core types)
func TogglProjectID(task *core.Task) string {
    return task.GetProperty("X-WORKLOG-TOGGL-PROJECT")
}
```

A user would install it by adding one import and one registration line to `main.go`:

```go
import togglplugin "github.com/example/worklog-toggl-plugin"

// in main():
registry.Register(togglplugin.NewPlugin())
```

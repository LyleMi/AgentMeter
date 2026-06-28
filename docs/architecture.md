# Architecture

## Decision

AgentMeter uses a Go application core with SQLite storage, shared query
services, and two local interfaces:

- Web UI: the Vue 3 + Vite + TypeScript browser dashboard served by the Go HTTP
  backend.
- TUI: a terminal interface over the same `app.App`, database, indexing
  pipeline, pricing rules, and query semantics.

The current product is local-first. It reads local agent JSONL files, indexes
normalized records into SQLite, and exposes private local views over that data.

## Runtime Shape

```text
local agent JSONL files
  -> source discovery and source identity
  -> JSONL parser
  -> ingestion and offline audit
  -> SQLite
  -> shared query service
  -> Web HTTP API / TUI
```

During Web development, Vite serves the frontend and proxies `/api` to the Go
backend. For local production use, the Go server serves built `frontend/dist`
assets from disk.

The HTTP API remains the Web mode boundary. TUI mode does not require a
separate ingestion path, pricing calculator, database schema, or persistence
layer.

## Implemented Components

```text
AgentMeter
  interfaces
    web dashboard
    terminal UI

  frontend
    Vue views
    charts and tables
    filters
    settings
    audit views
    agent privacy controls

  backend
    app service and HTTP routes
    startup asset preparation
    source discovery and identity
    agent source adapters
    JSONL parser
    ingestion pipeline
    offline audit rules
    SQLite repository and migrations
    pricing registry and calculator
    privacy config adapters
    shared query service
    shared view-model helpers

  storage
    normalized SQLite database
    app_config key/value settings
    seeded pricing registry
```

Export is not an implemented backend package today. CSV and JSON export remain
future work in the roadmap.

## Backend Package Shape

Current Go package responsibilities:

- `app`: application lifecycle, settings, indexing coordination, privacy
  actions, audit queries, and HTTP route registration.
- `agent`: source-root classification for Codex, Claude Code, CodeBuddy,
  WorkBuddy, and generic JSONL directories.
- `audit`: offline command-risk, privacy, egress, file, and secret findings
  derived from parsed local events.
- `db`: SQLite connection, migrations, repositories, app config, and pricing
  seeding.
- `ingest`: scan, hash, deduplicate, parse, audit, and index source files.
- `model`: shared domain and API structs.
- `platform`: OS-specific database and default source path discovery.
- `pricing`: seeded pricing registry, model normalization, and cost
  calculation.
- `privacy`: user-level external-agent privacy config adapters.
- `query`: read models for UI screens and API responses.
- `sessionjsonl`: supported JSONL event-shape parsing and normalization.
- `startup`: frontend dependency/build checks and browser startup helpers.
- `tui`: terminal UI mode over `app.App`.
- `viewmodel`: shared display formatting and presenter helpers for UI parity.

## Source Of Truth

- Source session JSONL files are the source of truth for raw agent history.
- SQLite is the local normalized cache and query store.
- `internal/db/db.go` is the schema source of truth.
- `internal/model/types.go` defines API/read-model shapes.
- `internal/query` defines shared read-model semantics consumed by Web and TUI.
- `internal/pricing/pricing.go` is the pricing registry source of truth; see
  [Pricing Sources](pricing-sources.md) for source links and assumptions.
- [Validation](validation.md) is the smoke and verification source of truth.

## Data Flow

```text
discover configured source roots
  -> classify source instance, agent family, and sessions path
  -> scan JSONL files recursively
  -> compare path, size, modified time, and content hash
  -> parse raw events into normalized sessions, usage, model calls, and tools
  -> run offline audit over parsed local data
  -> upsert records into SQLite
  -> query from Web API or TUI
```

Indexing is incremental by default. Rebuild indexing clears indexed files for
enabled sources and parses them again.

Source identity is instance-aware. A source instance is one configured or
discovered root with `sourceId`, `sourceKey`, label, root path, and sessions
path. An agent family is the parser family stored as `agentKind`/`agentName`,
such as `codex` or `claude`. Multiple source instances can share one family.
Startup auto-adds detected default agent homes while the Settings list is still
auto-managed, and configured roots with known child directories are classified
as family variants where possible. Manual source labels are display metadata and
do not change parsing or indexing.

## UI Shape

UI contributors should follow the practical design guidance in
[`docs/ui-design.md`](ui-design.md) for Web layout, visual quality, dense data
tables, component consistency, and UI-state validation.

The implemented Web product areas are:

- Overview
- Sessions
- Session Detail
- Tools
- Audit
- Agent Privacy
- Settings

The implemented TUI product areas are:

- Overview
- Sessions
- Session Detail
- Tools
- Agent Privacy status and profile apply
- Settings

Overview, Sessions, Session Detail, and Settings should be source-aware in both
interfaces. Compact tables should prefer source labels when present and keep
family/path context available so repeated Codex, Claude, CodeBuddy, or
WorkBuddy instances do not collapse into one ambiguous family label.

The TUI Agent Privacy screen shows all supported targets in a compact summary,
lets users select a target, and can apply supported privacy profiles only after
an explicit confirmation step. Web Agent Privacy supports the richer per-setting
editing flow for Codex `config.toml` and Gemini CLI, Claude Code, and CodeBuddy
Code/IDE `settings.json`.

Agent Privacy is target-based rather than source-instance-based. Privacy
targets are supported external agent config files, not indexed source labels.
Applying a profile to `codex`, for example, changes the supported user-level
Codex config and is not scoped to one Codex source instance.

## Interface Synchronization

Web and TUI modes stay synchronized by design:

- Token totals, cost estimates, durations, model normalization, and status
  labels come from shared backend logic.
- Overview, Sessions, Session Detail, Tools, Settings, Pricing, and implemented
  Agent Privacy status data use shared query or app-service semantics.
- Source filters use `source:<id>` for one source instance; family filters use
  values such as `codex` or `claude` when all sources of that family should
  match. Existing API fields named `agent` may carry either form.
- Filtering and sorting rules should not be reimplemented with different
  behavior in each UI.
- New shared user-visible behavior should update both interface expectations in
  the same change.
- Documentation for commands, capabilities, and validation should be updated
  with the implementation state.

## Command Line

Recommended local Web command:

```sh
go run . -start
```

Interface selector:

```sh
go run . -start
go run . -ui web -http 127.0.0.1:34115
go run . -ui web -static frontend/dist
go run . -ui tui
```

Behavior:

- `web` is the default mode for MVP compatibility.
- `-start` prepares built frontend assets, starts Web mode, and opens the
  browser for normal local use.
- `-http` and `-static` apply to Web mode.
- TUI mode runs in the terminal and does not start an HTTP listener.

## Runtime Rules

- AgentMeter must not modify source session files.
- AgentMeter must not upload data.
- The HTTP server should bind to `127.0.0.1` by default, not a public
  interface.
- Indexing should be incremental unless the user explicitly chooses rebuild.
- Raw parse errors should be visible but non-fatal.

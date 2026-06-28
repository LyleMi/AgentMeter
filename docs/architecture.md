# Architecture

## Decision

Use a Go application core with SQLite storage, shared query services, and two
local interfaces:

- Web UI: the current Vue 3 + Vite + TypeScript browser dashboard served by the
  Go HTTP backend.
- TUI: a terminal interface over the same app services, database, pricing rules,
  and query semantics.

## Why Go HTTP + Vite

AgentMeter is primarily a local Go data application with a browser-based
dashboard. A single local HTTP API keeps the runtime model explicit:

```text
Vue/Vite UI -> local HTTP API -> Go services -> SQLite
```

During development, Vite serves the frontend and proxies `/api` to the Go
backend. For local production use, the Go server can serve the built
`frontend/dist` assets from disk.

The HTTP API remains the Web mode boundary. TUI mode should not require a
separate ingestion path or a second persistence layer.

## High-level Components

```text
AgentMeter
  interfaces
    web dashboard
    terminal UI

  frontend
    Vue views
    charts
    filters
    settings
    agent privacy controls

  backend
    source discovery
    agent source adapters
    JSONL parser
    ingestion pipeline
    SQLite repository
    pricing service
    shared query service
    export service

  storage
    normalized SQLite database
    schema migrations
    pricing registry
```

## Data Flow

```text
discover configured agent source roots
  -> scan session JSONL files
  -> parse raw events
  -> normalize to internal event model
  -> upsert into SQLite
  -> query from UI
```

## Backend Package Shape

Proposed Go layout:

```text
internal/
  app/
  agent/
  sessionjsonl/
  db/
  ingest/
  model/
  pricing/
  query/
  tui/
  viewmodel/
  export/
  platform/
```

Responsibilities:

- `agent`: detect source roots such as Codex, Claude Code, CodeBuddy, WorkBuddy, or generic JSONL directories.
- `sessionjsonl`: understand supported JSONL event shapes and convert them to normalized records.
- `ingest`: scan, hash, deduplicate, and index files.
- `db`: SQLite connection, migrations, repositories.
- `model`: normalized domain structs.
- `pricing`: model aliases, pricing table, cost calculation.
- `query`: read models for UI screens.
- `viewmodel`: shared display formatting and presenter helpers for UI parity.
- `tui`: terminal UI mode over `app.App`.
- `export`: JSON and CSV export.
- `platform`: OS-specific database and default source path discovery.

## UI Shape

The Web and TUI interfaces should cover the same product areas:

- Overview
- Sessions
- Session Detail
- Tools
- Agent Privacy
- Settings

Overview should show:

- total sessions;
- total input/output/cached/reasoning tokens;
- estimated cost;
- total wall duration;
- total active duration;
- total tool calls;
- recent daily trend.

Session Detail should show:

- session metadata;
- token and cost summary;
- wall/model/tool/idle time;
- timeline;
- model calls;
- tool calls;
- raw source path.

Agent Privacy should show external-agent privacy configuration status and edit
supported user-level controls with explicit set/unset changes. Strict values are
available as a privacy-first preset per option, but users can save custom values
or remove configured values to return to tool defaults. The current
implementation targets Codex user-level `config.toml` and Gemini CLI user-level
`settings.json`.

The Web UI can use charts, wide layouts, and browser affordances. The TUI can
use tables, panes, keyboard navigation, and compact summaries. Differences in
presentation are acceptable; differences in totals, filters, status labels, or
drill-down semantics are not.

## Interface Synchronization

Web and TUI modes should stay synchronized by design:

- Token totals, cost estimates, durations, model normalization, and status
  labels come from shared backend logic.
- Overview, Sessions, Session Detail, Tools, Settings, and Pricing data use
  shared query semantics.
- Filtering and sorting rules should not be reimplemented with different
  behavior in each UI.
- New shared user-visible behavior should update both interface expectations in
  the same change.
- Documentation for command examples and UI capabilities should be updated with
  the implementation state.

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
- The HTTP server should bind to `127.0.0.1` by default, not a public interface.
- Indexing should be incremental.
- Raw parse errors should be visible but non-fatal.

# Architecture

## Decision

Use Wails with a Go backend, SQLite storage, and Vue 3 + Vite + TypeScript for
the frontend.

## Why Wails

AgentMeter is primarily a local Go data application with a desktop UI. Wails
keeps the architecture simple:

```text
Vue/Vite UI -> Wails bridge -> Go services -> SQLite
```

Tauri would be a stronger fit for a Rust-first desktop app. With a Go backend,
Tauri would likely require a Go sidecar or an extra local server:

```text
Vue/Vite UI -> Tauri/Rust shell -> Go sidecar/server -> SQLite
```

That adds process management, packaging, lifecycle, and logging complexity that
does not help the MVP.

## High-level Components

```text
AgentMeter
  frontend
    Vue views
    charts
    filters
    settings

  backend
    source discovery
    agent source adapters
    JSONL parser
    ingestion pipeline
    SQLite repository
    pricing service
    query service
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
- `export`: JSON and CSV export.
- `platform`: OS-specific database and default source path discovery.

## UI Shape

MVP screens:

- Overview
- Sessions
- Session Detail
- Tools
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

## Runtime Rules

- AgentMeter must not modify source session files.
- AgentMeter must not upload data.
- UI should bind to local desktop runtime, not a public interface.
- Indexing should be incremental.
- Raw parse errors should be visible but non-fatal.

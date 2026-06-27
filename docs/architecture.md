# Architecture

## Decision

Use a Go HTTP backend, SQLite storage, and a Vue 3 + Vite + TypeScript frontend.

## Why Go HTTP + Vite

AgentMeter is primarily a local Go data application with a browser-based
dashboard. A single local HTTP API keeps the runtime model explicit:

```text
Vue/Vite UI -> local HTTP API -> Go services -> SQLite
```

During development, Vite serves the frontend and proxies `/api` to the Go
backend. For local production use, the Go server can serve the built
`frontend/dist` assets from disk.

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
- The HTTP server should bind to `127.0.0.1` by default, not a public interface.
- Indexing should be incremental.
- Raw parse errors should be visible but non-fatal.

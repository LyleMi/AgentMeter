# AgentMeter

AgentMeter is a local-first observability dashboard for coding agent sessions.

The first target is Codex on Windows. AgentMeter reads local Codex session data,
normalizes it into SQLite, and presents usage, timing, and tool-call statistics
through a local desktop UI.

## Product Direction

- Local-first: read local session files only.
- No proxy: do not sit between the user and model providers.
- Private by default: no telemetry, no upload, no cloud dependency.
- Codex first: support Codex session JSONL before adding other agents.
- Windows first: make Windows paths, packaging, and local usage reliable first.
- SQLite backed: scan once, query fast, support incremental indexing.

## Core Metrics

AgentMeter should answer:

- How many sessions were run?
- How many tokens were consumed?
- What did those tokens cost?
- How long did sessions take?
- How many tool calls happened?
- Which tools were called most often?
- Which sessions, projects, days, or models consumed the most?

## Chosen Stack

- Desktop shell: Wails
- Backend: Go
- Database: SQLite
- Frontend: Vue 3 + Vite + TypeScript
- First platform: Windows

Wails is preferred over Tauri because the project is Go-backend-first. The
frontend can call Go services directly, and the session scanner, parser,
pricing logic, and SQLite access can stay in one Go process.

## Current Repository State

This repository now contains the Wails-oriented AgentMeter MVP:

- Go backend with SQLite migrations and local app configuration.
- Codex JSONL discovery, incremental hashing, parsing, and indexing.
- Normalized sessions, events, token usage, model calls, and tool calls.
- Vue 3 + TypeScript frontend using Ant Design Vue and ECharts.
- Local HTTP mode for development or use without the Wails CLI.

## Run Locally

Install frontend dependencies and build the embedded UI:

```powershell
cd frontend
npm install
npm run build
cd ..
```

Start local HTTP mode:

```powershell
go run . -http :34115
```

Open:

```text
http://127.0.0.1:34115
```

If Wails CLI is installed, the same backend and frontend can be launched through
Wails:

```powershell
wails dev
```

The SQLite database is created under:

```text
%LOCALAPPDATA%\AgentMeter\agentmeter.sqlite
```

The default Codex source path is:

```text
%USERPROFILE%\.codex\sessions
```

## Implemented MVP Screens

- Overview: totals, token and cost estimates, daily trend, model usage, recent sessions.
- Sessions: searchable local session table with model, token, cost, timing, and parse status.
- Session Detail: metadata, timeline, model calls, tool calls, raw source traceability.
- Tools: calls by tool, success/failure counts, total and average duration.
- Settings: Codex source path, database path, index/rebuild actions, pricing registry.

## Notes

Cost is displayed as a local estimate from the built-in pricing registry. Unknown
models remain indexed and are shown as `unpriced`.

## Documents

- [Project Brief](docs/project-brief.md)
- [Architecture](docs/architecture.md)
- [Data Model](docs/data-model.md)
- [Roadmap](docs/roadmap.md)

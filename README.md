# AgentMeter

<p align="center">
  <img src="frontend/src/assets/agentmeter-logo.png" alt="AgentMeter logo" width="180">
</p>

![Status: MVP](https://img.shields.io/badge/status-MVP-f2c94c)
![Local first](https://img.shields.io/badge/local--first-yes-2f855a)
![Platform: Cross-platform](https://img.shields.io/badge/platform-cross--platform-0078d4)
![Backend: Go](https://img.shields.io/badge/backend-Go-00ADD8)
![Frontend: Vue 3](https://img.shields.io/badge/frontend-Vue%203-42b883)
![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue)

AgentMeter is a local-first dashboard for understanding coding-agent session usage:
tokens, estimated cost, timing, session history, and tool-call behavior.

It reads local agent JSONL session files, indexes them into SQLite, and shows the data
in private local Web and terminal interfaces. No proxy, no cloud service, no telemetry.

The Web UI is the default MVP interface. The TUI is available as a terminal MVP
over the same local database, indexing pipeline, pricing rules, and query
semantics.

## Why AgentMeter

Coding agents can generate a lot of useful local session data, but that data is
hard to inspect directly. AgentMeter turns local JSONL sessions into answers you
can actually use:

- How many sessions did I run?
- How many tokens did they consume?
- What did those tokens roughly cost?
- Which models, days, projects, or sessions used the most?
- Which tools were called most often?
- How long did sessions and tool calls take?

## Features

- Overview dashboard with sessions, tokens, estimated cost, daily usage, and model usage.
- Multiple local source roots, with Codex, Claude Code, CodeBuddy, WorkBuddy, and generic JSONL detection.
- Agent-level usage grouping for developers who run several coding agents side by side.
- Searchable session history with parse status and raw source traceability.
- Session detail timeline with model calls, tool calls, metadata, and source paths.
- Tool-call analytics with call counts, success/failure counts, and durations.
- Incremental indexing based on path, size, modified time, and content hash.
- Built-in pricing registry with unknown models clearly marked as `unpriced`.
- Local Go HTTP server with a Vue 3 + Vite frontend.
- TUI mode for terminal workflows, kept in sync with the Web UI through shared
  backend/query behavior.

## Privacy Model

AgentMeter is designed to stay local:

- Reads local session files only.
- Does not proxy model traffic.
- Does not upload session data.
- Does not require a cloud account.
- Stores normalized data in a local SQLite database.

The default database path follows the host OS:

```text
Windows: %LOCALAPPDATA%\AgentMeter\agentmeter.sqlite
macOS:   ~/Library/Application Support/AgentMeter/agentmeter.sqlite
Linux:   $XDG_DATA_HOME/AgentMeter/agentmeter.sqlite or ~/.local/share/AgentMeter/agentmeter.sqlite
```

Default source roots are detected from local agent homes when they exist:

```text
~/.codex
~/.claude
~/.codebuddy
~/.workbuddy
```

## Quick Start

Requirements:

- Go matching the version in `go.mod`
- Node.js and npm

Recommended local start:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\start.ps1
```

The script installs frontend dependencies when needed, rebuilds the UI when
source files change, starts AgentMeter, and opens:

```text
http://127.0.0.1:34115
```

On first launch, click **Index Now** in the app. AgentMeter defaults to detected
local agent homes such as `~/.codex` and `~/.claude`. In **Settings**, enter one
source root per line when you use multiple agents or keep session logs elsewhere.

Build the frontend and start local HTTP mode manually:

```powershell
cd frontend
npm ci
npm run build
cd ..

go run . -http 127.0.0.1:34115
```

Open:

```text
http://127.0.0.1:34115
```

### UI Modes

Web mode remains the default:

```powershell
go run . -http 127.0.0.1:34115
```

You can select a mode explicitly:

```powershell
go run . -ui web -http 127.0.0.1:34115
go run . -ui web -static frontend/dist
go run . -ui tui
```

Behavior:

- `-ui web` starts the local HTTP API and browser dashboard.
- `-ui tui` starts the terminal UI without opening a browser or HTTP listener.
- Web remains the default for compatibility with the MVP.
- `-http` and `-static` apply to Web mode.
- Both modes must use the same SQLite data, pricing rules, filters, and usage
  calculations.

TUI keys:

```text
1-4 / tab  switch screens
up/down    select or scroll
enter      open selected session detail
b / esc    back from detail
r          refresh current screen
i / I      index now / rebuild index
q          quit
```

For frontend development with Vite hot module reload, run the backend and
frontend dev server in separate terminals:

```powershell
go run . -http 127.0.0.1:34115
```

```powershell
cd frontend
npm run dev
```

Open `http://127.0.0.1:5173`. Vite proxies `/api` requests to the Go backend.

For Go backend auto-restart, install Air and run this instead of `go run`:

```powershell
air -c .air.toml
```

## How It Works

```text
Agent JSONL -> scanner/parser -> SQLite -> Go query service -> Vue dashboard
```

When the source path is a Codex home directory, AgentMeter scans `sessions\`
first and then `archived_sessions\`, keeping the active copy when both contain
the same relative JSONL path. When the source path is a Claude Code, CodeBuddy,
or WorkBuddy home, AgentMeter scans `projects/`. A direct directory of saved
JSONL output is also supported through the generic JSONL adapter.

## Current Status

AgentMeter is currently an MVP for local coding-agent JSONL usage. The repository
already includes:

- Go backend with SQLite migrations and local app configuration.
- Codex, Claude Code, CodeBuddy, WorkBuddy, and generic JSONL discovery, parsing, normalization, and incremental indexing.
- Normalized sessions, events, token usage, model calls, and tool calls.
- Vue 3 + TypeScript frontend using Ant Design Vue and ECharts.
- MVP screens for Overview, Sessions, Session Detail, Tools, and Settings.
- TUI mode with Overview, Sessions, Session Detail, Tools, Settings, refresh,
  and index actions over the same application services.

## Development Checks

Run the shared backend checks:

```powershell
go test ./...
```

Run the Web build check:

```powershell
cd frontend
npm ci
npm run build
cd ..
```

For TUI changes, run the backend tests and a terminal smoke check covering
startup, keyboard navigation, resize behavior, indexing, and parity for Overview
totals, Session Detail values, and Tools aggregates against Web mode for the
same database.

When shared query behavior changes, update both UI expectations in the same
change: Web views, TUI screens, README command examples, and
`docs/ui-modes.md`.

## Roadmap

Planned directions include richer TUI filters, packaged builds, more
coding-agent adapters, export formats, project grouping, custom pricing, and
richer timeline views.

See [Roadmap](docs/roadmap.md) for details.

## Documentation

- [Project Brief](docs/project-brief.md)
- [Architecture](docs/architecture.md)
- [UI Modes](docs/ui-modes.md)
- [Data Model](docs/data-model.md)
- [Codex Session Format](docs/codex-session-format.md)
- [Roadmap](docs/roadmap.md)

## Contributing

Issues and pull requests are welcome, especially for parser edge cases, pricing
updates, packaging, and adapters for other coding agents.

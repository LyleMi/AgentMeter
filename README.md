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
in a private desktop UI. No proxy, no cloud service, no telemetry.

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
- Multiple local source roots, with Codex, Claude Code, CodeBuddy, and generic JSONL detection.
- Agent-level usage grouping for developers who run several coding agents side by side.
- Searchable session history with parse status and raw source traceability.
- Session detail timeline with model calls, tool calls, metadata, and source paths.
- Tool-call analytics with call counts, success/failure counts, and durations.
- Incremental indexing based on path, size, modified time, and content hash.
- Built-in pricing registry with unknown models clearly marked as `unpriced`.
- Local HTTP mode for development or use without the Wails CLI.

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
```

## Quick Start

Requirements:

- Go matching the version in `go.mod`
- Node.js and npm
- Wails CLI is optional

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

go run . -http :34115
```

Open:

```text
http://127.0.0.1:34115
```

If the Wails CLI is installed, you can also run the desktop app in development
mode:

```powershell
wails dev
```

## How It Works

```text
Agent JSONL -> scanner/parser -> SQLite -> Go query service -> Vue dashboard
```

When the source path is a Codex home directory, AgentMeter scans `sessions\`
first and then `archived_sessions\`, keeping the active copy when both contain
the same relative JSONL path. When the source path is a Claude Code or CodeBuddy
home, AgentMeter scans `projects/`. A direct directory of saved JSONL output is
also supported through the generic JSONL adapter.

## Current Status

AgentMeter is currently an MVP for local coding-agent JSONL usage. The repository
already includes:

- Go backend with SQLite migrations and local app configuration.
- Codex, Claude Code, CodeBuddy, and generic JSONL discovery, parsing, normalization, and incremental indexing.
- Normalized sessions, events, token usage, model calls, and tool calls.
- Vue 3 + TypeScript frontend using Ant Design Vue and ECharts.
- MVP screens for Overview, Sessions, Session Detail, Tools, and Settings.

## Development Checks

```powershell
cd frontend
npm ci
npm run build
cd ..

go test ./...
```

## Roadmap

Planned directions include packaged builds, more coding-agent adapters, export formats, project grouping, custom pricing, and
richer timeline views.

See [Roadmap](docs/roadmap.md) for details.

## Documentation

- [Project Brief](docs/project-brief.md)
- [Architecture](docs/architecture.md)
- [Data Model](docs/data-model.md)
- [Codex Session Format](docs/codex-session-format.md)
- [Roadmap](docs/roadmap.md)

## Contributing

Issues and pull requests are welcome, especially for parser edge cases, pricing
updates, packaging, and adapters for other coding agents.

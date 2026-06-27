# Getting Started

## Requirements

- Go matching the version in `go.mod`
- Node.js and npm

## Recommended Local Start

Use the repository start script for normal local use:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\start.ps1
```

The script installs frontend dependencies when needed, rebuilds the UI when
built assets are missing or stale, starts AgentMeter, and opens:

```text
http://127.0.0.1:34115
```

`scripts/start.ps1` serves built frontend assets through the Go server. It is
not the Vite hot module reload workflow.

On first launch, click **Index Now** in the app. AgentMeter defaults to detected
local agent homes such as `~/.codex` and `~/.claude`. In **Settings**, enter one
source root per line when you use multiple agents or keep session logs elsewhere.

## Data Locations

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

## Manual Web Startup

Build the frontend and start local HTTP mode manually:

```powershell
cd frontend
npm ci
npm run build
cd ..

go run . -ui web -http 127.0.0.1:34115
```

Open:

```text
http://127.0.0.1:34115
```

You can also pass a built asset directory explicitly:

```powershell
go run . -ui web -static frontend/dist
```

## Frontend HMR

For frontend development with Vite hot module reload, run the backend and
frontend dev server in separate terminals.

Backend:

```powershell
go run . -http 127.0.0.1:34115
```

Frontend:

```powershell
cd frontend
npm run dev
```

Open:

```text
http://127.0.0.1:5173
```

Vite proxies `/api` requests to the Go backend.

For Go backend auto-restart, install Air and run this instead of `go run`:

```powershell
air -c .air.toml
```

## TUI Mode

Start the terminal UI with:

```powershell
go run . -ui tui
```

For mode behavior, command flags, and TUI keyboard bindings, see
[UI Modes](ui-modes.md).

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
change: Web views, TUI screens, README command examples, and `docs/ui-modes.md`.

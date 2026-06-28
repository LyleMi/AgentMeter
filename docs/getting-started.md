# Getting Started

## Requirements

- Go matching the version in `go.mod`
- Node.js and npm

## Recommended Local Start

Use the Go start mode for normal local use:

```sh
go run . -start
```

Start mode installs frontend dependencies when needed, rebuilds the UI when
built assets are missing or stale, starts AgentMeter, and opens:

```text
http://127.0.0.1:34115
```

`go run . -start` serves built frontend assets through the Go server. It is not
the Vite hot module reload workflow.

On first launch, click **Update Index** in the app. AgentMeter defaults to detected
local agent homes such as `~/.codex` and `~/.claude`. In **Settings**, enter one
source root per line when you use multiple agents or keep session logs elsewhere.
**Update Index** skips unchanged JSONL files; **Rebuild Index** clears indexed
files for enabled sources and parses every JSONL file again.

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

```sh
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

```sh
go run . -ui web -static frontend/dist
```

## Frontend HMR

For frontend development with Vite hot module reload, run the backend and
frontend dev server in separate terminals.

Backend:

```sh
go run . -http 127.0.0.1:34115
```

Frontend:

```sh
cd frontend
npm run dev
```

Open:

```text
http://127.0.0.1:5173
```

Vite proxies `/api` requests to the Go backend.

For Go backend auto-restart, run Air through the Go tool directive:

```sh
go tool air -c .air.toml
```

## TUI Mode

Start the terminal UI with:

```sh
go run . -ui tui
```

For mode behavior, command flags, and TUI keyboard bindings, see
[UI Modes](ui-modes.md).

## Privacy Config CLI

AgentMeter can inspect and edit supported user-level privacy config files
without opening the Web UI:

```sh
go run . privacy status
go run . privacy settings codex
go run . privacy apply codex
go run . privacy apply all recommended
go run . privacy apply gemini strict
```

`privacy apply <target>` defaults to the `recommended` profile. Use `strict` to
write every managed hardening setting, or `default` to unset AgentMeter-managed
keys and return to vendor defaults. Supported targets are `codex`, `gemini`,
`claude`, and `codebuddy`. Existing config files are backed up before writes.

## Development Checks

The complete validation and smoke contract lives in
[Validation](validation.md). Common commands:

```sh
go test ./...
```

```sh
cd frontend
npm ci
npm run build
cd ..
```

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke-api.ps1 -BaseUrl http://127.0.0.1:34115
```

```sh
cd frontend
npm run test:smoke
cd ..
```

Routine smoke checks should be read-only unless the task explicitly asks to
validate indexing, settings writes, or privacy config changes.

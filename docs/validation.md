# Validation

This file is the source of truth for AgentMeter validation commands and smoke
contracts. Keep routine smoke checks read-only unless the task explicitly asks
to validate a state-changing flow.

## General Rules

- Use existing dev services when they are already running.
- Do not kill or restart processes unless asked or unless you started them.
- Do not click **Update Index**, **Rebuild Index**, save settings, or change
  agent privacy settings during routine smoke checks unless the task explicitly
  requires that state change.
- Browser smoke uses hash-router URLs such as `/#/overview/summary` and
  `/#/time`.
- Web HMR smoke should target Vite at `http://127.0.0.1:5173`, with Vite
  proxying `/api` to the Go backend.
- `go run . -start` is for built-asset local use, not the Vite HMR smoke path.

## Backend

Run Go unit tests:

```sh
go test ./...
```

Use this for backend parsing, indexing, pricing, privacy, audit, startup,
query, viewmodel, and TUI-support changes.

## Frontend Build

Run the Web build check:

```sh
cd frontend
npm ci
npm run build
cd ..
```

This runs `vue-tsc` and builds the Vite app.

## API Smoke

Against an already running backend:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke-api.ps1 -BaseUrl http://127.0.0.1:34115
```

This smoke should remain read-only. It should verify API shape and availability
without triggering indexing, rebuilds, settings writes, or privacy config
changes.

## Browser Smoke

Start backend and frontend HMR in separate terminals when you own those
processes.

Backend:

```sh
go run . -http 127.0.0.1:34115
```

Frontend:

```sh
cd frontend
npm run dev
```

Run Playwright smoke:

```sh
cd frontend
npm run test:smoke
cd ..
```

Override the browser smoke target with `AGENTMETER_WEB_URL` only when needed.
Routine browser smoke should not perform state-changing app actions.

## TUI Smoke

For TUI changes, run backend tests and a terminal smoke path:

```sh
go test ./...
go run . -ui tui
```

Manually cover:

- startup and quit;
- screen navigation;
- session detail open/back flow;
- refresh key behavior;
- narrow and wide terminal resize behavior;
- visible parse/index/pricing status labels;
- Overview totals, Session Detail values, and Tools aggregates compared against
  Web mode for the same database.

Index and rebuild keys are state-changing. Use them only when the task
explicitly requires validating indexing from TUI.

## Shared Query And UI Contract

When shared query behavior or read-model shapes change, validate both Web and
TUI expectations where relevant:

- totals and filters;
- session identity and source path display;
- parse-status labels;
- pricing and `unpriced` behavior;
- audit summary and finding filters;
- Settings source entries and last index status;
- README command examples;
- [UI Modes](ui-modes.md);
- [Architecture](architecture.md);
- [Roadmap](roadmap.md).

## Documentation-only Changes

For documentation-only changes, text review is enough. Use `rg` to check old
links or stale terminology when files are renamed or concepts move:

```sh
rg "old-link-or-term" README.md README.zh-CN.md AGENTS.md docs
```

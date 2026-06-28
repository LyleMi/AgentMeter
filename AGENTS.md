# Repository Guidelines

## Project Structure & Module Organization

AgentMeter is a local-first Go and Vue application. The Go entry point is
`main.go`; backend packages live under `internal/`, grouped by responsibility
such as `app`, `ingest`, `query`, `pricing`, `privacy`, and `tui`. Frontend
source is in `frontend/src`, with Vue views in `frontend/src/views`,
reusable components in `frontend/src/components`, and assets in
`frontend/src/assets`. Playwright smoke tests live in `frontend/tests`. Project
docs are in `docs/`, scripts in `scripts/`, and static build assets are
generated under `frontend/dist`.

## Build, Test, and Development Commands

- `go run . -start`: installs/builds frontend assets if needed, starts the web
  app on `127.0.0.1:34115`, and serves built assets.
- `go run . -http 127.0.0.1:34115`: starts the backend for frontend HMR work.
- `cd frontend; npm run dev`: starts Vite HMR on `127.0.0.1:5173`.
- `go test ./...`: runs all Go unit tests.
- `cd frontend; npm ci; npm run build`: installs frontend dependencies, runs
  `vue-tsc`, and builds the Vite app.
- `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke-api.ps1 -BaseUrl http://127.0.0.1:34115`:
  read-only API smoke test against a running backend.
- `cd frontend; npm run test:smoke`: Playwright browser smoke tests against
  frontend HMR.

## Coding Style & Naming Conventions

Use `gofmt` for Go and keep packages small, lowercase, and purpose-specific.
Go tests should sit beside the package under test and use `_test.go` suffixes.
Frontend code uses Vue 3, TypeScript, two-space indentation, single quotes, and
no semicolons. Name Vue views and components in PascalCase, such as
`OverviewSummary.vue`, and keep shared presentation helpers in focused `.ts`
modules.

## Testing Guidelines

Prefer focused unit tests for backend parsing, indexing, pricing, privacy, and
viewmodel behavior. For shared query or data-shape changes, verify both Web and
TUI expectations where relevant. Browser smoke uses hash-router URLs such as
`/#/overview/summary`; keep routine smoke validation read-only.

## Commit & Pull Request Guidelines

Use Conventional Commits for every commit subject:
`type(scope): imperative subject`. Allowed types are `feat`, `fix`, `docs`,
`test`, `refactor`, `style`, `chore`, and `ci`. Keep scopes lowercase and
optional, keep the subject under 72 characters, start it lowercase, and do not
end it with punctuation. Examples: `docs: add repository contributor guide`,
`fix(privacy): correct agent privacy routes`, and
`test(smoke): add API and browser checks`. Keep PRs focused, describe behavior
changes and validation performed, link related issues when applicable, and add
screenshots for visible Web UI changes.

## Agent-Specific Instructions

Use existing dev services when they are already running. Do not kill or restart
processes unless asked or unless you started them. Do not click **Update Index**,
**Rebuild Index**, save settings, or change agent privacy settings during
routine smoke checks unless the task explicitly requires that state change.

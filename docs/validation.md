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

API smoke covers overview, token analytics, model signals, day/project usage
breakdowns, sessions, tools, audit, pricing, settings, and privacy status.
Cache-related shape checks include `totalInputTokens`, `totalCachedInputTokens`,
`cacheUtilizationRate`, `cacheHitTrend`, `dailyUsage.cachedInputTokens`, and
usage-breakdown bucket `projectPath`/`cachedInputTokens`/
`cacheUtilizationRate`. Model Signals checks include the top-level raw
operational metrics, `trend`, `modelBreakdown`, and `anomalySessions` arrays,
the operational efficiency `dailyMetrics` and `projectMetrics` arrays, plus the
Model Health `healthSummary` object and `cohorts`, `matrix`, and
`projectHotspots` arrays. These arrays are compatible with empty data and only
lightly validate row shape when rows exist.

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

## Static Pages Preview

The public GitHub Pages preview builds the real Vue frontend in static demo
mode and uploads `frontend/dist`. It does not require the Go backend at runtime.
For preview changes, validate:

```powershell
cd frontend
$env:VITE_AGENTMETER_STATIC_DEMO='true'
npm run build
cd ..
```

- `frontend/dist/index.html` renders the Vue app under
  `https://lylemi.github.io/AgentMeter/`;
- built JS, CSS, favicon, and other Vite asset URLs are rooted at
  `/AgentMeter/`, not `/`;
- hash-router navigation works for routes such as `/#/overview/summary`,
  `/#/sessions`, `/#/tools`, `/#/audit/summary`, and `/#/agent-privacy`;
- desktop and mobile viewports have no page-level horizontal overflow;
- `robots.txt`, `sitemap.xml`, `llms.txt`, canonical, and Open Graph metadata
  remain aligned with `https://lylemi.github.io/AgentMeter/`;
- no real prompts, file paths, session IDs, secrets, or audit evidence appear
  in public assets.

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
- source-aware Overview top agents, including source label and family/path
  context when multiple source instances exist;
- session rows showing source label or agent name, not only family kind;
- Session Detail showing source label, family kind, source root, sessions path,
  and raw JSONL file path;
- Settings source entries showing configured labels when labels are present;
- Agent Privacy staying target-based while clearly showing backend status
  warnings for the selected target;
- Overview totals, Session Detail values, Tools aggregates, and recent tool-call
  rows/details compared against Web mode for the same database.

Index and rebuild keys are state-changing. Use them only when the task
explicitly requires validating indexing from TUI.

## Shared Query And UI Contract

When shared query behavior or read-model shapes change, validate both Web and
TUI expectations where relevant:

- totals and filters;
- shared cache fields, including `dailyUsage.cachedInputTokens`,
  `dailyUsage.cacheUtilizationRate`, `cacheHitTrend`, and usage-breakdown
  `projectPath` for project grouping;
- Model Signals fields and filters, including raw signal fields, `trend`,
  `modelBreakdown`, `anomalySessions`, `dailyMetrics`, `projectMetrics`,
  `healthSummary`, `cohorts`, `matrix`, `projectHotspots`, and empty arrays
  returned as `[]` rather than `null`;
- Model Signals health/drift interpretation, including current window as the
  latest observed 24h in scope, baseline as the preceding 30d when available,
  low-sample confidence states, and missing baseline not being labeled as
  regression;
- Model Signals daily/project efficiency interpretation, including cost,
  cost-per-session, cost-per-active-hour, cost-per-1k-tokens, cache savings,
  p50/p90 latency, p50/p10 throughput, retry pressure, failure pressure,
  model quality risk, preceding 7-calendar-day daily drift, project
  current-versus-baseline drift, and missing pricing or low sample being labeled
  as confidence/completeness risk rather than failure;
- Model Signals Web presentation, including single-axis chart metric controls,
  horizontal source/model comparison, a standalone model-quality-risk page with
  explanations, current-versus-baseline project comparison where available, and
  missing chart values not being rendered as zero;
- project-scoped analytics filters using the `project` query parameter;
- session identity and source path display;
- source instance filters using `source:<id>` and family filters using values
  such as `codex` or `claude`;
- parse-status labels;
- pricing and `unpriced` behavior;
- audit summary and finding filters;
- Settings source entries and last index status;
- privacy status warnings and the target-based scope of privacy writes;
- README command examples;
- [UI Modes](ui-modes.md);
- [Architecture](architecture.md);
- [Roadmap](roadmap.md).

For backend read-model changes, run `go test ./...`. If visible Web UI changed,
also run the frontend build (`cd frontend; npm ci; npm run build`) and browser
smoke (`cd frontend; npm run test:smoke`) against the appropriate local
backend/frontend services.

## Documentation-only Changes

For documentation-only changes, text review is enough. Use `rg` to check old
links or stale terminology when files are renamed or concepts move:

```sh
rg "old-link-or-term" README.md README.zh-CN.md AGENTS.md docs
```

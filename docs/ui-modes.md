# UI Modes

AgentMeter is a dual-interface local application:

- Web UI for the current browser dashboard.
- TUI for terminal-first and remote-shell workflows.

Both interfaces must use the same local data model, SQLite database, pricing
rules, indexing pipeline, and query semantics. The Web UI remains the default
MVP interface. The TUI is implemented as a terminal MVP and should not fork
product behavior.

## Goals

- Keep AgentMeter useful in both a browser and a terminal.
- Preserve one source of truth for usage numbers, pricing, filters, and session
  identity.
- Let packaged builds expose a predictable interface choice.
- Make feature parity checks part of normal development rather than a separate
  cleanup task.

## Web Mode

Web mode runs the local Go HTTP server and serves the Vue dashboard.

Current command:

```powershell
go run . -http 127.0.0.1:34115
```

Current flags:

- `-http`: HTTP listen address. The default is `127.0.0.1:34115`.
- `-static`: directory containing built frontend assets. The default is
  `frontend/dist`.

Runtime rules:

- Bind to loopback by default.
- Serve `/api` from the Go backend.
- Serve built frontend assets from `-static` when available.
- Keep source session files read-only.

## TUI Mode

TUI mode runs inside the terminal and uses the same application services as Web
mode. It does not introduce a second parser, database schema, pricing
calculator, or set of usage formulas.

Implemented MVP TUI screens:

- Overview
- Sessions
- Session Detail
- Tools
- Settings

Implemented TUI actions:

- refresh the current screen;
- trigger incremental indexing;
- trigger rebuild indexing;
- open a selected session detail from the session list;
- navigate back from detail to the session list.

The TUI may present data differently from the Web UI. For example, charts can be
replaced by tables or compact sparklines. The underlying numbers, filters,
labels, parse-status meanings, and drill-down paths should remain aligned.

## Command Line

The implemented interface selector is:

```text
go run . -ui web
go run . -ui tui
```

Examples:

```powershell
go run . -ui web -http 127.0.0.1:34115
go run . -ui web -static frontend/dist
go run . -ui tui
```

Flag behavior:

- `-ui web` starts the local HTTP server and browser-oriented UI.
- `-ui tui` starts the terminal UI without opening a browser or HTTP listener.
- Default remains `web` for compatibility with the current MVP.
- `-http` applies to Web mode.
- `-static` applies to Web mode.
- TUI mode should not start a public HTTP listener by default.

## TUI Keyboard

```text
1 / o      Overview
2 / s      Sessions
3 / t      Tools
4 / g      Settings
tab/right  next screen
left       previous screen
up/down    select or scroll
j / k      select or scroll
pgup/pgdn  page through scrollable content
home/end   jump within lists
enter      open selected session detail
b / esc    return from detail
r          refresh current screen
i          index now
I          rebuild index
q / ctrl-c quit
```

## Synchronization Principles

- Shared calculations: token totals, cost estimates, durations, status labels,
  and model normalization must come from shared backend logic.
- Shared query contracts: Web and TUI should consume the same read-model
  semantics for Overview, Sessions, Session Detail, Tools, Settings, and
  Pricing data.
- No UI-only business rules: filtering, sorting defaults, pricing visibility,
  parse-status handling, and source-root behavior should not be reimplemented
  differently in each interface.
- Intentional presentation differences are allowed: Web can use charts and rich
  layouts; TUI can use tables, panes, keyboard navigation, and compact summaries.
- Feature changes should name both interface impacts in the pull request or
  change note when the change affects shared user-visible behavior.
- Documentation updates should keep README usage examples, this file, and the
  roadmap aligned when UI mode behavior changes.

## Development Checks

For shared backend or query changes:

```powershell
go test ./...
```

For Web UI changes:

```powershell
cd frontend
npm ci
npm run build
cd ..
```

For TUI changes:

- Run the TUI smoke path from a clean terminal.
- Verify resize behavior for narrow and wide terminal widths.
- Verify keyboard navigation for each implemented screen.
- Compare Overview totals, Session Detail values, and Tools
  aggregates against Web mode for the same database.
- When filter entry lands, compare Sessions filters against Web mode for the
  same database.

Before merging a feature that changes shared product behavior, check:

- README command examples still match implemented flags.
- `docs/architecture.md` still describes the actual interface boundaries.
- `docs/roadmap.md` still reflects whether the TUI work is planned, active, or
  delivered.

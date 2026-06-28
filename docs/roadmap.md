# Roadmap

## Phase 0: Discovery

Goal: understand local coding-agent session data well enough to design the parser safely.

Tasks:

- Locate Codex and Claude Code session directories across supported OSes.
- Collect representative local JSONL samples.
- Document event types found in real sessions.
- Identify where token usage appears.
- Identify how tool calls appear.
- Identify whether model-call durations can be derived.
- Capture parse edge cases: interrupted sessions, failed tool calls, empty
  files, malformed lines, partial writes.

Deliverables:

- `docs/codex-session-format.md`
- parser fixtures with redacted JSONL samples
- initial parser tests

## Phase 1: Local Web Scaffold

Goal: replace the exploratory prototype with the intended application shape.

Tasks:

- Create Vue + TypeScript frontend with Vite.
- Create Go HTTP backend.
- Set Go module name to AgentMeter.
- Add SQLite dependency.
- Add migration runner.
- Add app configuration storage.
- Add cross-platform database and default source path discovery.

Deliverables:

- runnable local web app;
- empty shell UI;
- local SQLite database creation.

## Phase 2: Agent Indexer

Goal: index local coding-agent sessions into SQLite.

Tasks:

- Implement source discovery.
- Implement JSONL scanner.
- Implement Codex, Claude Code, CodeBuddy, WorkBuddy, and generic JSONL parsing.
- Normalize sessions, events, token usage, and tool calls.
- Add incremental indexing based on path, size, modified time, and hash.
- Add parse warnings.

Deliverables:

- Update index action;
- session rows in SQLite;
- parser test coverage.

## Phase 3: MVP Web UI

Goal: make the indexed data useful in the browser.

Screens:

- Overview
- Sessions
- Session Detail
- Tools
- Agent Privacy
- Settings

Tasks:

- Overview totals and daily trend.
- Session table with filters.
- Session detail timeline.
- Tool-call breakdown.
- Editable external-agent privacy controls for supported Codex, Gemini CLI, and Claude Code config settings.
- Pricing registry display.
- Rebuild index action.

Deliverables:

- usable local Web dashboard for local coding-agent sessions.

## Phase 4: TUI Mode

Goal: keep the terminal interface useful without splitting AgentMeter into two
products.

Interface contract:

- Web mode remains the default MVP path.
- TUI mode uses the same SQLite database, pricing rules, indexing pipeline, and
  query semantics as Web mode.
- UI differences are presentational only: terminal tables and panes can replace
  charts, but totals, filters, statuses, and drill-down meaning must match Web.

Implemented command line:

```text
go run . -start
go run . -ui web -http 127.0.0.1:34115
go run . -ui web -static frontend/dist
go run . -ui tui
```

Delivered:

- Add `-ui web|tui` mode selection while keeping Web as the default.
- Define shared display helpers for formatting, status classification, Overview
  derived metrics, and Tools summary.
- Implement TUI navigation, table browsing, and session detail panes.
- Support an index trigger and visible indexing/parse status in TUI mode.
- Add terminal resize and narrow-width behavior.
- Document TUI keyboard behavior and README examples.

Remaining:

- Add search/filter entry in TUI Sessions.
- Add parity checks comparing Web and TUI values for the same database.
- Improve compact visual treatment for pricing, parse status, and long paths.
- Add terminal smoke checks to release validation.

## Phase 5: Packaging

Goal: make cross-platform usage easy.

Tasks:

- Windows portable build.
- macOS portable build.
- Linux portable build.
- Installer or portable zip.
- Package both Web and TUI modes from the same binary when TUI mode is ready.
- Local database path decision.
- Log file location.
- Basic crash/error reporting to local logs only.

Deliverables:

- Windows release artifact.

## Phase 6: Beyond MVP

Possible additions:

- More complete Claude Code, CodeBuddy, and WorkBuddy adapter coverage.
- Gemini CLI adapter.
- OpenCode adapter.
- CSV export.
- JSON export.
- estimated token mode.
- richer model-call timeline.
- project grouping.
- custom pricing UI.
- TUI command palette and saved filter shortcuts.
- dark/light theme.

# Roadmap

## Phase 0: Discovery

Goal: understand Codex session data well enough to design the parser safely.

Tasks:

- Locate Codex session directories on Windows.
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

## Phase 1: Wails Scaffold

Goal: replace the exploratory prototype with the intended application shape.

Tasks:

- Create Wails project using Vue + TypeScript.
- Set Go module name to AgentMeter.
- Add SQLite dependency.
- Add migration runner.
- Add app configuration storage.
- Add Windows Codex path discovery.

Deliverables:

- runnable Wails app;
- empty shell UI;
- local SQLite database creation.

## Phase 2: Codex Indexer

Goal: index Codex sessions into SQLite.

Tasks:

- Implement source discovery.
- Implement JSONL scanner.
- Implement Codex parser.
- Normalize sessions, events, token usage, and tool calls.
- Add incremental indexing based on path, size, modified time, and hash.
- Add parse warnings.

Deliverables:

- `Index Now` action;
- session rows in SQLite;
- parser test coverage.

## Phase 3: MVP UI

Goal: make the indexed data useful.

Screens:

- Overview
- Sessions
- Session Detail
- Tools
- Settings

Tasks:

- Overview totals and daily trend.
- Session table with filters.
- Session detail timeline.
- Tool-call breakdown.
- Pricing registry display.
- Rebuild index action.

Deliverables:

- usable local dashboard for Codex sessions.

## Phase 4: Packaging

Goal: make Windows usage easy.

Tasks:

- Windows app build.
- Installer or portable zip.
- Local database path decision.
- Log file location.
- Basic crash/error reporting to local logs only.

Deliverables:

- Windows release artifact.

## Phase 5: Beyond MVP

Possible additions:

- macOS and Linux support.
- Claude Code adapter.
- Gemini CLI adapter.
- OpenCode adapter.
- CSV export.
- JSON export.
- estimated token mode.
- richer model-call timeline.
- project grouping.
- custom pricing UI.
- dark/light theme.

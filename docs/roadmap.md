# Roadmap

This roadmap separates delivered MVP behavior from remaining and future work.
For current package boundaries, see [Architecture](architecture.md). For the
verification contract, see [Validation](validation.md).

## Current Status

AgentMeter is an MVP for local coding-agent JSONL usage analysis.

Implemented today:

- Local Web dashboard over a Go HTTP API.
- Terminal UI MVP over the same application core.
- SQLite database creation, migrations, and normalized query storage.
- App configuration storage for source entries and last index status.
- Codex, Claude Code, CodeBuddy, WorkBuddy, and generic JSONL source detection
  and parsing.
- Multiple source roots with enabled/disabled entries.
- Incremental indexing and rebuild indexing.
- Overview, Sessions, Session Detail, Tools, Audit, Agent Privacy, and Settings
  in Web mode.
- Overview, Sessions, Session Detail, Tools, Agent Privacy status/profile apply,
  and Settings in TUI mode.
- Offline audit findings for indexed local session data, exposed through Web UI
  and `/api/audit/*`.
- Pricing registry seeding, model normalization, and `unpriced` handling.
- Cross-platform portable release automation for Windows, macOS, and Linux on
  amd64 and arm64.

Not implemented today:

- CSV export.
- JSON export.
- Installer packages, signing, and notarization.
- TUI session search/filter entry.
- Automated terminal smoke checks in release validation.

## Delivered Phases

### Phase 0: Discovery

Delivered:

- Locate Codex and Claude Code session directories across supported OSes.
- Collect representative JSONL samples through parser tests and fixtures.
- Document observed event types and parsing assumptions.
- Identify token usage and tool-call shapes.
- Capture parser edge cases such as malformed lines, empty files, partial
  sessions, missing timestamps, and pending tool calls.

Primary document:

- [Session Formats](session-formats.md)

### Phase 1: Local Web Scaffold

Delivered:

- Vue + TypeScript frontend with Vite.
- Go HTTP backend.
- SQLite dependency and migration runner.
- App configuration storage.
- Cross-platform database path and default source path discovery.
- Local Web startup through `go run . -start`.

### Phase 2: Agent Indexer

Delivered:

- Source discovery for Codex, Claude Code, CodeBuddy, WorkBuddy, and generic
  JSONL directories.
- Recursive JSONL scanner.
- Normalization for sessions, events, token usage, model calls, and tool calls.
- Incremental indexing based on path, size, modified time, and content hash.
- Parse warnings and scan status.
- Offline audit generation during indexing.

### Phase 3: MVP Web UI

Delivered Web screens:

- Overview
- Sessions
- Session Detail
- Tools
- Audit
- Agent Privacy
- Settings

Delivered Web behavior:

- Overview totals, daily trend, model usage, and agent usage.
- Session table with search/model/agent filters.
- Session detail timeline, model calls, tool calls, and raw source path.
- Tool-call breakdown and tool-call list.
- Audit summary and findings list.
- Editable external-agent privacy controls for supported Codex, Gemini CLI,
  Claude Code, and CodeBuddy config settings.
- Pricing registry display.
- Update Index and Rebuild Index actions.

### Phase 4: TUI Mode

Delivered:

- `-ui web|tui` mode selection while keeping Web as default.
- TUI mode over the same SQLite database, pricing rules, indexing pipeline, and
  query semantics as Web mode.
- Shared display helpers for formatting, status classification, Overview
  derived metrics, and Tools summary.
- TUI navigation, table browsing, and session detail panes.
- TUI index trigger and visible indexing/parse status.
- TUI Agent Privacy status screen for supported targets, including confirmed
  profile application.
- Terminal resize and narrow-width behavior.
- Documented TUI keyboard behavior and README examples.

Remaining for TUI:

- Add search/filter entry in TUI Sessions.
- Add parity checks comparing Web and TUI values for the same database.
- Improve compact visual treatment for pricing, parse status, and long paths.
- Add automated terminal smoke checks to release validation.

## Future Phases

### Phase 5: Packaging

Goal: make cross-platform usage easy.

Planned:

- Installer packages, signing, and notarization.
- Package Web and TUI modes from the same binary.
- Confirm local database path behavior for packaged builds.
- Define log file location.
- Keep crash/error reporting local-only.

Deliverable:

- Signed installer or documented portable-package distribution.

### Phase 6: Beyond MVP

Possible additions:

- More complete Claude Code, CodeBuddy, and WorkBuddy adapter coverage.
- Gemini CLI session adapter.
- OpenCode adapter.
- CSV export.
- JSON export.
- Estimated token mode.
- Richer model-call timeline.
- Project grouping.
- Custom pricing UI.
- TUI command palette and saved filter shortcuts.
- Dark/light theme.

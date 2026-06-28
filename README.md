# AgentMeter

[中文](README.zh-CN.md) | [English](README.md)

<p align="center">
  <img src="frontend/src/assets/agentmeter-logo.png" alt="AgentMeter logo" width="180">
</p>

<p align="center">
  <strong>Local-first coding-agent usage analytics for tokens, cost, timing, sessions, and tool calls.</strong>
</p>

<p align="center">
  <a href="https://lylemi.github.io/AgentMeter/">Live demo</a>
  | <a href="https://github.com/LyleMi/AgentMeter/releases">Downloads</a>
  | <a href="docs/assets/screenshots/overview.png">Screenshot</a>
  | <a href="docs/install.md">Install</a>
  | <a href="docs/privacy.md">Privacy</a>
</p>

![Status: MVP](https://img.shields.io/badge/status-MVP-f2c94c)
![Local first](https://img.shields.io/badge/local--first-yes-2f855a)
![Platform: Cross-platform](https://img.shields.io/badge/platform-cross--platform-0078d4)
![Backend: Go](https://img.shields.io/badge/backend-Go-00ADD8)
![Frontend: Vue 3](https://img.shields.io/badge/frontend-Vue%203-42b883)
![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue)

AgentMeter is an open-source Go + Vue dashboard for understanding local
coding-agent session usage. It reads local agent JSONL session files, indexes
them into SQLite, and shows tokens, estimated cost, timing, session history,
models, projects, cache reuse, and tool-call behavior in private local Web and
terminal interfaces.

No proxy, no cloud service, no telemetry.

![AgentMeter overview screenshot](docs/assets/screenshots/overview.png)

## At A Glance

- **Supported agents:** Codex, Claude Code, CodeBuddy, WorkBuddy, and generic
  JSONL directories.
- **Privacy model:** session data stays on your machine in a local SQLite
  database; AgentMeter does not proxy traffic or upload telemetry.
- **Primary views:** sessions, daily usage, models, projects, cache reuse,
  estimated cost, tool-call analytics, and offline command/privacy audit
  findings.
- **Interfaces:** local Web dashboard by default, plus a terminal UI over the
  same database and query behavior.
- **Release assets:** cross-platform archives are published on
  [GitHub Releases](https://github.com/LyleMi/AgentMeter/releases) as
  `AgentMeter-<platform>-<arch>` builds.

## Why AgentMeter

Coding agents leave useful local session data behind, but raw JSONL is hard to
inspect directly. AgentMeter turns that data into answers you can use:

- How many sessions did I run?
- How many tokens did they consume?
- What did those tokens roughly cost?
- Which models, days, projects, or sessions used the most?
- Where are cached input tokens being reused by day or project?
- Which tools were called most often?
- How long did sessions and tool calls take?

## Features

- Local Web dashboard for sessions, tokens, estimated cost, daily usage, model
  usage, project usage, cache reuse by day/project, and tool-call analytics.
- Offline audit view for command-risk and privacy/secret findings derived from
  indexed local session data.
- Terminal UI mode over the same database, indexing pipeline, pricing rules, and
  query behavior.
- Codex, Claude Code, CodeBuddy, WorkBuddy, and generic JSONL source detection.
- Multiple labeled source instances for developers running several local coding
  agents or several roots from the same agent family.
- Incremental SQLite indexing with source path traceability and parse status.
- Built-in pricing registry with unknown models clearly marked as `unpriced`.

## Quick Start

For packaged builds, download the matching `AgentMeter-<platform>-<arch>` asset
from [Releases](https://github.com/LyleMi/AgentMeter/releases). For local source
startup, use the commands below.

Requirements:

- Go matching the version in `go.mod`
- Node.js and npm

Recommended local start:

```sh
go run . -start
# same as:
go run . start
```

Open:

```text
http://127.0.0.1:34115
```

On first launch, click **Update Index** in the app. AgentMeter defaults to
detected local agent homes such as `~/.codex` and `~/.claude`; you can add more
source roots in **Settings** and label them when paths alone are ambiguous. A
source instance is one local root, while the agent family (`codex`, `claude`,
and so on) controls parser behavior and family-level filters. **Update Index**
scans only new or changed JSONL files; **Rebuild Index** clears indexed files
for enabled sources and parses them all again.

For manual startup, frontend HMR, TUI mode, data locations, and development
checks, see [Getting Started](docs/getting-started.md).

Terminal UI shortcut:

```sh
go run . tui
# or:
go run . cli
```

Privacy config CLI:

```sh
go run . privacy status
go run . privacy settings codex
go run . privacy apply codex
go run . privacy apply all recommended
go run . privacy apply gemini strict
```

`privacy apply <target>` uses the recommended profile by default. Supported
targets are `codex`, `gemini`, `claude`, and `codebuddy`; existing config files
are backed up before AgentMeter writes changes. Privacy targets are user-level
agent configs, not indexed source instances, so a write is not scoped to one
manual source label.

## Privacy Model

AgentMeter is designed to stay local:

- Reads local session files only.
- Does not proxy model traffic.
- Does not upload session data.
- Does not require a cloud account.
- Stores normalized data in a local SQLite database.
- Audit findings may store raw local evidence so command and privacy issues can
  be inspected without leaving the machine.

## Current Status

AgentMeter is an MVP for local coding-agent JSONL usage. The Web UI is the
default interface; the TUI is available as a terminal MVP over the same
application core.

See [Roadmap](docs/roadmap.md) for planned work.

## Documentation

- [Install](docs/install.md)
- [Supported Agents](docs/supported-agents.md)
- [Privacy](docs/privacy.md)
- [Comparison](docs/comparison.md)
- [Release Distribution](docs/release-distribution.md)
- [Getting Started](docs/getting-started.md)
- [Project Brief](docs/project-brief.md)
- [Architecture](docs/architecture.md)
- [UI Modes](docs/ui-modes.md)
- [Data Model](docs/data-model.md)
- [Session Formats](docs/session-formats.md)
- [Pricing Sources](docs/pricing-sources.md)
- [Validation](docs/validation.md)
- [Roadmap](docs/roadmap.md)

## Contributing

Issues and pull requests are welcome, especially for parser edge cases, pricing
updates, packaging, and adapters for other coding agents.

# Supported Agents And JSONL Sources

AgentMeter indexes local coding-agent JSONL session files. It normalizes
supported events into one local SQLite schema for sessions, token usage, model
calls, tool calls, and offline audit findings.

For event-shape details, see [Session Formats](session-formats.md). For the data
model, see [Data Model](data-model.md).

## Source Instance Vs Agent Family

AgentMeter separates two related ideas:

- A source instance is one configured or discovered local root, with its own
  source id, label, root path, and sessions path.
- An agent family is the parser family, such as `codex`, `claude`,
  `codebuddy`, `workbuddy`, or `jsonl`.

This matters when you use the same agent in more than one place. Two Codex
homes should appear as two source instances while both still use the `codex`
parser family.

Use source labels such as `Work Codex` and `Personal Claude` when path names are
not enough to identify the local instance.

## Support Matrix

| Source | Default root | Family value | Support level | Notes |
| --- | --- | --- | --- | --- |
| Codex | `~/.codex` | `codex` | Deepest documented support | Reads `sessions` and `archived_sessions`; supports observed token-count and tool-call events. |
| Claude Code | `~/.claude` | `claude` | Observed local support | Supports observed assistant message usage and tool-use/tool-result content. |
| CodeBuddy | `~/.codebuddy` | `codebuddy` | Observed local support | Supports observed usage fields and function-call records under common project/session layouts. |
| WorkBuddy | `~/.workbuddy` | `workbuddy` | Observed local support | Uses the same observed family of CodeBuddy/WorkBuddy JSONL shapes where present. |
| Generic JSONL directory | Any configured directory | `jsonl` | Best effort | Recursively scans JSONL and extracts known usage/event shapes when metadata is available. |

Support level means AgentMeter can parse known local JSONL shapes used by the
current code and tests. It does not mean every future vendor event field is
known. Unknown or partial records should be retained as raw events when
possible, and missing fields are treated as unknown rather than guessed.

## Discovery Rules

AgentMeter auto-detects default homes when they exist:

```text
~/.codex
~/.claude
~/.codebuddy
~/.workbuddy
```

Configured roots are also classified when they contain a known family structure,
such as Codex `sessions` or `archived_sessions`, Claude `projects`, or
CodeBuddy/WorkBuddy `projects` or `sessions`.

If a configured path is a direct JSONL directory rather than a known agent home,
AgentMeter scans it recursively as `jsonl`.

## What AgentMeter Extracts

Depending on the source data, AgentMeter can extract:

- session identity and project path;
- started and ended timestamps;
- wall, active, model, tool, and idle duration estimates;
- input, cached input, output, reasoning, and total token usage;
- model names and provider metadata when present;
- tool-call names, status, summaries, and durations;
- parse status and non-fatal parse warnings;
- local offline audit findings.

AgentMeter prefers actual usage found in the source JSONL. When a field is not
present, the UI should show unknown or unpriced status instead of inventing a
number.

## Privacy Targets Are Separate

The Agent Privacy screen and privacy CLI can inspect or apply supported
user-level privacy config profiles for external agent tools. That target list
is not the same as the session indexing support matrix.

For example, a tool can have privacy config support before it has a session
parser. Privacy CLI writes are target-based. The Web UI can scope supported
privacy writes to a selected indexed source root when matching source instances
exist for that target.

See [Privacy](privacy.md) and [Getting Started](getting-started.md) for the
privacy CLI commands.

## Adding Or Improving Support

Parser changes should update:

- parser tests and fixtures;
- [Session Formats](session-formats.md);
- [Data Model](data-model.md), when read-model fields or semantics change;
- [Validation](validation.md), when smoke expectations change.

Keep real user prompts, secrets, private paths, and proprietary session data out
of issues and test fixtures.

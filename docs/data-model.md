# Data Model

The database stores normalized records derived from local coding-agent JSONL. Raw
source files remain the source of truth.

## Core Entities

### sources

Tracks configured local data sources.

Fields:

- `id`
- `kind` such as `codex`, `claude`, `codebuddy`, `workbuddy`, or `jsonl`
- `name`
- `root_path`
- `sessions_path`
- `platform`
- `created_at`
- `updated_at`

### source_files

Tracks scanned JSONL files.

Fields:

- `id`
- `source_id`
- `path`
- `size_bytes`
- `modified_at`
- `content_hash`
- `last_scanned_at`
- `scan_status`
- `error`

### sessions

One normalized agent session.

Fields:

- `id`
- `source_id`
- `source_file_id`
- `session_key`
- `codex_session_id`
- `project_path`
- `model`
- `model_provider`
- `originator`
- `thread_source`
- `agent_nickname`
- `agent_role`
- `started_at`
- `ended_at`
- `wall_duration_ms`
- `active_duration_ms`
- `model_duration_ms`
- `tool_duration_ms`
- `idle_duration_ms`
- `event_count`
- `parse_status`

`session_key` is the generic identity shown in the UI. `codex_session_id` is
kept for backward compatibility with older databases and API clients.

### events

Normalized timeline events.

Fields:

- `id`
- `session_id`
- `source_file_id`
- `source_line`
- `timestamp`
- `kind`
- `raw_type`
- `summary`
- `raw_json`

### token_usage

Token totals for sessions, model calls, or events.

Fields:

- `id`
- `owner_kind`
- `owner_id`
- `model`
- `input_tokens`
- `cached_input_tokens`
- `output_tokens`
- `reasoning_output_tokens`
- `total_tokens`
- `source`

`source` values:

- `actual`
- `estimated`
- `unknown`

MVP should only store `actual` or `unknown`. Estimation can come later.

### model_calls

Model invocation records when they can be derived from session data.

Fields:

- `id`
- `session_id`
- `started_at`
- `ended_at`
- `duration_ms`
- `model`
- `provider`
- `status`
- `input_tokens`
- `cached_input_tokens`
- `output_tokens`
- `reasoning_output_tokens`
- `total_tokens`
- `cost_usd`

### tool_calls

Tool invocation records derived from local events.

Fields:

- `id`
- `session_id`
- `started_at`
- `ended_at`
- `duration_ms`
- `tool_name`
- `status`
- `input_summary`
- `output_summary`
- `error`
- `call_id`
- `raw_event_id`
- `raw_start_event_id`
- `raw_end_event_id`

MVP statistics:

- total calls;
- calls by tool;
- success/failure count;
- total duration;
- average duration.

### audit_runs

Tracks that a source file has gone through offline audit, even when no findings
were produced. This lets incremental indexing backfill audit data for existing
databases without repeatedly reparsing clean files.

Fields:

- `id`
- `source_file_id`
- `session_id`
- `source`
- `status`
- `finding_count`
- `audited_at`

### audit_findings

Offline command, egress, file, and privacy findings derived from indexed local
session events. Findings are local-only and may keep raw evidence for inspection.

Fields:

- `id`
- `session_id`
- `tool_call_id`
- `source_file_id`
- `raw_event_id`
- `source_line`
- `timestamp`
- `source`
- `event_type`
- `category`
- `severity`
- `rule_id`
- `title`
- `description`
- `evidence`
- `command`
- `shell_family`
- `platform`
- `decision`
- `created_at`

`decision` is currently `observed` for offline indexing. It is reserved for
future live audit sources that may record `allowed`, `blocked`, or
`needs_approval`.

### pricing_models

Local pricing registry.

Fields:

- `id`
- `model`
- `normalized_model`
- `input_per_1m`
- `cached_input_per_1m`
- `output_per_1m`
- `source`
- `effective_from`

### app_config

Local application key/value settings that are not source session data.

Fields:

- `key`
- `value`
- `updated_at`

Known keys:

- `source_entries`: JSON array of configured source entries. Each entry has a
  `path` and `enabled` flag. This is the Settings source of truth for which
  local agent roots are indexed.
- `source_entries_auto_defaults`: JSON array tracking default source roots that
  were auto-added by AgentMeter. This lets startup merge newly discovered
  default roots without treating every user-edited source as auto-managed.
- `last_index_result`: JSON-encoded `IndexResult` from the latest successful
  index run. It is shown through Settings as local status metadata and is not a
  replacement for the normalized indexed tables.

`app_config` values are local AgentMeter state. They are not derived source
records and should not be interpreted as agent session history.

## Timing Definitions

### Wall Duration

`ended_at - started_at`, based on the first and last valid timestamps in the
session.

### Model Duration

Sum of model-call durations when start and end can be identified.

### Tool Duration

Sum of tool-call durations when start and end can be identified.

### Active Duration

`model_duration_ms + tool_duration_ms`.

### Idle Duration

`wall_duration_ms - active_duration_ms`, clamped at zero.

This is only an approximation. It includes user thinking time, UI delay, and
any session gaps that are not represented as model or tool activity.

## Cost Rules

Use local token usage from session data first.

Cost formula:

```text
(input_tokens - cached_input_tokens) * input_rate
+ cached_input_tokens * cached_input_rate
+ output_tokens * output_rate
```

Rates are USD per 1M tokens.

Unknown pricing should not block indexing. UI should show `unpriced` for those
sessions.

## Read-model Contract

`internal/query` is the shared read-model layer for Web API responses and TUI
screens. UI code should consume these semantics rather than recomputing business
rules locally.

Current read models:

- Overview: session totals, token totals, estimated cost, unpriced session
  count, wall/active duration totals, tool-call total, daily usage, model usage,
  agent usage, and recent sessions.
- Sessions: filtered list by search, model, agent, limit, and offset, ordered by
  newest `started_at` first.
- Session Detail: one session with normalized events, model calls, and tool
  calls.
- Tools: aggregate tool stats by name with success/failure counts, total
  duration, and average duration.
- Tool Calls: filtered list by tool, agent, start range, sort, limit, and offset.
- Audit Summary: finding counts by severity/category and recent findings.
- Audit Findings: filtered list by category, severity, shell family, search,
  limit, and offset.
- Pricing Models: seeded pricing rows returned from the local registry.

Contract rules:

- Web and TUI should show the same totals, filters, status labels, cost
  semantics, and session identity for the same database.
- UI-specific presentation may differ, but token, cost, duration, parse-status,
  pricing-status, and audit-status definitions must remain shared.
- `internal/model/types.go` defines the JSON/API field shape. Documentation
  should be updated when those shapes change.
- `source_entries` and `last_index_result` are surfaced through Settings, but
  normalized usage analytics come from the indexed source tables.

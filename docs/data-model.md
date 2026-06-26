# Data Model

The database stores normalized records derived from local coding-agent JSONL. Raw
source files remain the source of truth.

## Core Entities

### sources

Tracks configured local data sources.

Fields:

- `id`
- `kind` such as `codex`, `claude`, or `jsonl`
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
- `raw_event_id`

MVP statistics:

- total calls;
- calls by tool;
- success/failure count;
- total duration;
- average duration.

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

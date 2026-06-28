# Data Model

The database stores normalized records derived from local coding-agent JSONL. Raw
source files remain the source of truth.

## Source Identity Terms

A source instance is one configured or discovered local agent root or sessions
directory. It is identified in read models by `sourceId` and `sourceKey`
(`source:<id>`), and carries display/path context through `sourceLabel`,
`sourceRootPath`, and `sourceSessionsPath`.

An agent family is the parser family for that source, exposed as `agentKind`
and `agentName`. Family values include `codex`, `claude`, `codebuddy`,
`workbuddy`, and `jsonl`. Multiple source instances can share one family.

Use source identity when the user needs to distinguish local installations or
work/personal roots. Use family identity when the behavior is parser-specific or
when a filter intentionally spans every source of the same family.

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

`name` is the detected family display name for the source instance. Manual
labels live in `app_config.source_entries` and are surfaced through read models
as `sourceLabel` where available.

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
- `source_key` in read models
- `source_label` in read models
- `source_root_path` in read models
- `source_sessions_path` in read models
- `source_file_id`
- `raw_source_path` in read models
- `session_key`
- `codex_session_id`
- `agent_kind`
- `agent_name`
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

`session_key` is the stable session identity shown in the UI. `codex_session_id`
is kept for backward compatibility with older databases and API clients.
Session rows should use the source label or agent name for compact source
identity, not only the family kind. Session detail should expose the source
root, sessions path, and raw source file path for traceability.

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

`output_tokens` is normalized as generated output for cost and throughput.
For providers that report visible output and thinking/reasoning separately,
AgentMeter stores their billable/generated sum in `output_tokens` and keeps the
thinking/reasoning portion in `reasoning_output_tokens` as a sub-share.

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
- `source_id` in read models
- `source_key` in read models
- `source_label` in read models
- `source_root_path` in read models
- `source_sessions_path` in read models
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
- `source_id` in read models
- `source_key` in read models
- `source_label` in read models
- `source_root_path` in read models
- `source_sessions_path` in read models
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
- `is_custom`

### app_config

Local application key/value settings that are not source session data.

Fields:

- `key`
- `value`
- `updated_at`

Known keys:

- `source_entries`: JSON array of configured source entries. Each entry has a
  `path`, `enabled` flag, and optional `label`. This is the Settings source of
  truth for which local agent roots are indexed and which manual display labels
  should be used.
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

When older or generic records appear to report visible output and reasoning
separately, pricing adds the reasoning side to billable output only when the
model/total-token shape makes that separation explicit. This avoids
double-counting providers whose reported output already includes reasoning.

Unknown pricing should not block indexing. UI should show `unpriced` for those
sessions.

## Read-model Contract

`internal/query` is the shared read-model layer for Web API responses and TUI
screens. UI code should consume these semantics rather than recomputing business
rules locally.

Current read models:

- Overview: session totals, token totals, estimated cost, unpriced session
  count, wall/active duration totals, tool-call total, daily usage, cache-hit
  trend, model usage, source-aware agent usage, time attribution, slow sessions,
  and recent sessions. Overview can be scoped by agent/source, model, project,
  and started-at range.
- Token Analytics: token totals, cache utilization, cache-hit trend, estimated
  cost, model usage, source-aware agent usage, recent sessions, and high-token
  sessions. Token analytics can be scoped by agent/source, model, project, and
  started-at range.
- Usage Breakdown: token, cache utilization, session count, pricing, and
  identity buckets grouped by source (`agent`), model, source plus model
  (`agent,model`), day, or project, with the same agent/source, model, and
  project and started-at range filters.
- Model Signals: operational signals and health/drift read models for observed
  provider/model behavior, including model call density, output expansion,
  reasoning-token share, cache-miss rate, model throughput, tool dependency,
  tool failures, per-model breakdown, trend rows, anomaly sessions, daily
  operational efficiency metrics, project operational efficiency metrics,
  health summary, cohort health rows, matrix rows, and project hotspots. Model
  Signals can be scoped by agent/source, model, project, and started-at range.
- Sessions: filtered list by search, model, agent/source, limit, and offset,
  ordered by newest `started_at` first.
- Session Detail: one session with normalized events, model calls, and tool
  calls.
- Tools: aggregate tool stats by name with success/failure counts, total
  duration, and average duration.
- Tool Calls: filtered list by tool, agent/source, start range, sort, limit,
  and offset.
- Audit Summary: finding counts by severity/category and recent findings.
- Audit Findings: filtered list by category, severity, shell family,
  agent/source, search, limit, and offset.
- Pricing Models: seeded pricing rows returned from the local registry.

Contract rules:

- Web and TUI should show the same totals, filters, status labels, cost
  semantics, and session identity for the same database.
- Source filters use `source:<id>` when the intent is one source instance.
  Family filters use the family kind, such as `codex`, when the intent is every
  source in that parser family. Existing API fields named `agent` may carry
  either value until the API surface is renamed.
- Analytics APIs use `project`, `from`, and `to` query parameters for project
  path and inclusive `started_at` bounds. They should be passed as the same path
  and timestamp string formats used elsewhere in API filters.
- UI-specific presentation may differ, but token, cost, duration, parse-status,
  pricing-status, and audit-status definitions must remain shared.
- `internal/model/types.go` defines the JSON/API field shape. Documentation
  should be updated when those shapes change.
- `source_entries` and `last_index_result` are surfaced through Settings, but
  normalized usage analytics come from the indexed source tables.

Read-model shape notes:

- Overview `dailyUsage` rows include token totals for each day, including
  `cachedInputTokens` and `cacheUtilizationRate`, so day-level cache reuse is
  shared by Web and TUI instead of recomputed in presentation code.
- Overview and Token Analytics include `cacheHitTrend` rows for charting daily
  cache reuse. Each row contains daily input and cached input tokens, the daily
  cache utilization rate, a 7-day rolling cache utilization rate weighted by
  input tokens, and low-input-volume metadata so UI can distinguish low-sample
  volatility from broader model/provider behavior.
- `/api/model-signals` returns a Model Signals read model with the same
  analytics filters as Overview and Token Analytics: `agent`, `model`,
  `project`, `from`, and `to`. Existing top-level raw signal fields include
  `totalSessions`, `totalModelCalls`, `totalToolCalls`, `failedToolCalls`,
  `toolFailureRate`, `toolDependencyRate`, `avgModelCallsPerSession`,
  `outputExpansionRate`, `reasoningTokenShare`, `reasoningOverheadRate`,
  `visibleOutputTokens`, `billableOutputTokens`, `cacheMissRate`,
  `modelThroughputTokensPerSecond`,
  `modelThroughputOutputTokensPerSecond`, `trend`, `modelBreakdown`, and
  `anomalySessions`.
- `/api/model-signals` keeps the raw signal fields and Model Health layer:
  `healthSummary`, `cohorts`, `matrix`, and `projectHotspots`. The core
  grouping is provider/model + agent/source + project. The current health
  window is the latest observed 24 hours in the filtered scope; the baseline is
  the preceding 30 days when enough matching history is available. Missing
  baseline and low sample data should be surfaced as low confidence or
  unavailable history, not as regression.
- `/api/model-signals` also includes `dailyMetrics` for day-level operational
  efficiency. Rows should expose enough date, sample, cost, cost-per-session,
  cost-per-active-hour, cost-per-1k-tokens, cache-savings, latency percentile,
  throughput percentile, failure-pressure, model-quality-risk score,
  retry-pressure/model-calls-per-session, low-sample, and rolling
  7-calendar-day drift metadata for Web and TUI clients to explain the metric
  without recomputing it. Latency and throughput percentiles should use
  model-call token/duration samples when available and fall back to
  session-level samples when per-call token counts are missing.
- `/api/model-signals` also includes `projectMetrics` for project-level
  operational efficiency. Rows should expose enough project identity, sample,
  project cost burn, cache-savings, cost-per-session, cost-per-active-hour,
  cost-per-1k-tokens, dominant-model, model-mix, retry-pressure,
  failure-pressure, model-quality-risk score, confidence, and
  current-versus-baseline drift metadata for clients to present the same
  semantics.
- Model Signals `trend` and `modelBreakdown` rows expose count, token, duration,
  and rate fields so Web and TUI clients can present the same numerator and
  denominator semantics. Reasoning fields keep `reasoningTokenShare` for API
  compatibility and also expose `visibleOutputTokens`, `billableOutputTokens`,
  and `reasoningOverheadRate` so clients can show reasoning as observability
  and cost shape instead of assuming lower is always better. Empty collection
  fields must be JSON arrays (`[]`), not `null`.
- Model Health rows should expose enough cohort identity and sample/confidence
  metadata for clients to explain a health label without recomputing it. Strong
  signals are latency per 1k output tokens, throughput, model-call status/error
  data when available, and token/cost shape. The model-quality-risk score is a
  composite triage metric for stacked service symptoms, including relay,
  gateway, provider-side throttling, or weaker-model suspicion, but it is not
  proof of substitution or token padding. Tool failure, model calls per session,
  output expansion, cache miss, and reasoning overhead are weaker symptoms that
  require session context and baseline comparison.
- Model Signals efficiency fields are operational proxies for local behavior,
  not universal model capability scores. Missing pricing, unavailable
  cache-token data, missing baseline history, and low sample sizes should be
  documented and surfaced as confidence or completeness risk rather than
  treated as model failures.
- `/api/usage/breakdown` returns usage buckets selected by `groupBy`. Project
  buckets use `groupBy=project` and carry `projectPath`; project bucket keys use
  the same path normalization and platform case semantics as source paths. All
  bucket shapes, including day and project, include `cachedInputTokens` and
  `cacheUtilizationRate` for cache-hit visibility.

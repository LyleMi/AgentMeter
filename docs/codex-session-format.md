# Codex Session JSONL Format

AgentMeter currently targets local Codex session files under:

```text
%USERPROFILE%\.codex\sessions\YYYY\MM\DD\*.jsonl
```

Each line is a JSON object with these common top-level fields:

- `timestamp`
- `type`
- `payload`

Observed top-level `type` values:

- `session_meta`
- `turn_context`
- `event_msg`
- `response_item`

## Session Metadata

`session_meta` carries the stable session identity and environment metadata:

- `payload.session_id`
- `payload.cwd`
- `payload.originator`
- `payload.thread_source`
- `payload.model_provider`

`turn_context` carries the active turn environment:

- `payload.cwd`
- `payload.model`
- `payload.current_date`
- `payload.timezone`
- `payload.approval_policy`
- `payload.sandbox_policy`
- `payload.workspace_roots`

AgentMeter uses `session_meta.session_id` as the Codex session ID, falling back
to the JSONL filename when it is missing.

## Token Usage

Token usage appears in `payload.type = "token_count"` events.

The total session usage is in:

```text
payload.info.total_token_usage
```

The latest model-call delta is in:

```text
payload.info.last_token_usage
```

Both objects can contain:

- `input_tokens`
- `cached_input_tokens`
- `output_tokens`
- `reasoning_output_tokens`
- `total_tokens`

AgentMeter stores the latest `total_token_usage` as actual session usage. It
also creates approximate model-call rows from `last_token_usage` events.

## Tool Calls

Tool calls are represented by paired start/output events. The parser matches
them by `payload.call_id` or `payload.id`.

Observed call start payload types:

- `function_call`
- `custom_tool_call`
- `web_search_call`
- `tool_search_call`

Observed output/end payload types:

- `function_call_output`
- `custom_tool_call_output`
- `web_search_end`
- `web_search_output`
- `tool_search_output`
- `patch_apply_end`

AgentMeter stores the tool name, status, input/output previews, error preview,
and duration when both start and end timestamps are available.

## Timing

Session wall duration is the distance between the first and last valid event
timestamp in the file.

Model duration is approximate. Codex JSONL does not always expose exact model
call boundaries, so AgentMeter uses `task_started`, previous token count events,
and tool completion timestamps as available boundaries.

Tool duration is the sum of matched tool start/end timestamp differences.

## Edge Cases

The parser treats these as non-fatal warnings:

- malformed JSONL lines;
- missing timestamps;
- missing `session_meta`;
- empty files;
- pending tool calls without an output event.

The source JSONL files remain read-only source data. AgentMeter stores raw event
JSON in SQLite for local traceability.

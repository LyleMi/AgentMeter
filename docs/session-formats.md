# Session JSONL Formats

AgentMeter indexes local JSONL session files from multiple coding agents. The
parser normalizes supported shapes into the same session, event, usage, model
call, and tool-call records. Unsupported or partially understood fields are
kept local and treated as non-fatal parse warnings where possible.

The source JSONL files remain read-only source data. SQLite stores normalized
records and selected raw event JSON for local traceability.

## Supported Source Kinds

Current source kinds:

- `codex`
- `claude`
- `codebuddy`
- `workbuddy`
- `jsonl`

Generic `jsonl` sources are recursively scanned directories. The parser applies
the same observed event-shape support described below, but source-specific
metadata may be missing.

## Source Discovery

Codex session files are detected under a Codex home:

```text
%USERPROFILE%\.codex\sessions\YYYY\MM\DD\*.jsonl
%USERPROFILE%\.codex\archived_sessions\YYYY\MM\DD\*.jsonl
```

Other default agent homes:

```text
~/.claude
~/.codebuddy
~/.workbuddy
```

If the configured source path is a direct JSONL directory rather than a known
agent home, AgentMeter scans that directory recursively as `jsonl`.

## Codex Shape

Each line is a JSON object with these common top-level fields:

- `timestamp`
- `type`
- `payload`

Observed top-level `type` values:

- `session_meta`
- `turn_context`
- `event_msg`
- `response_item`

`session_meta` carries stable session identity and environment metadata:

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

### Codex Token Usage

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

AgentMeter stores actual session usage by summing each token event delta.
`last_token_usage` is used directly when present; otherwise AgentMeter subtracts
the previous `total_token_usage` from the current cumulative total. It also
creates approximate model-call rows from those per-event usage deltas.

### Codex Tool Calls

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

AgentMeter stores the tool name, call id, status, input/output previews, error
preview, start/end raw event links, and duration when both start and end
timestamps are available.

## Claude Code Shape

Claude Code JSONL support is based on observed local session events with common
top-level fields such as:

- `type`
- `sessionId`
- `cwd`
- `timestamp`
- `message`

The parser recognizes assistant message usage and tool-use content:

- `message.model`
- `message.usage.input_tokens`
- `message.usage.cache_creation_input_tokens`
- `message.usage.cache_read_input_tokens`
- `message.usage.output_tokens`
- `message.content[].type = "tool_use"`
- `message.content[].type = "tool_result"`

Cache creation tokens are counted as input tokens. Cache read tokens are counted
as cached input tokens.

## CodeBuddy And WorkBuddy Shape

CodeBuddy and WorkBuddy support is based on observed JSONL records with common
top-level fields such as:

- `id`
- `parentId`
- `timestamp`
- `type`
- `providerData`
- `sessionId`
- `cwd`

Recognized usage locations include:

- `providerData.usage`
- `message.usage`

Recognized usage field variants include camelCase and snake_case names:

- `inputTokens` / `input_tokens`
- `cachedTokens` / `cached_tokens`
- `cacheReadInputTokens` / `cache_read_input_tokens`
- `outputTokens` / `output_tokens`
- `reasoningTokens` / `reasoning_tokens`
- `totalTokens` / `total_tokens`

Recognized tool-call records include:

- `type = "function_call"`
- `type = "function_call_result"`

The parser matches tool calls by `callId` where available and stores input and
output summaries from arguments, argument display text, output text, or
providerData tool-result fields.

## Generic JSONL Shape

Generic JSONL directories do not imply a specific agent. The parser can still
extract useful records when lines use supported Codex-like, Claude-like,
CodeBuddy-like, WorkBuddy-like, or common usage shapes.

Observed generic usage shapes include:

- `usage.input_tokens`
- `usage.cached_input_tokens`
- `usage.output_tokens`
- `usage.reasoning_output_tokens`
- `usage.total_tokens`
- `usage.prompt_tokens`
- `usage.cached_tokens`
- `usage.completion_tokens`

Generic sources may have less reliable session identity, project path, agent
name, and provider metadata. AgentMeter falls back to the JSONL filename and
available event fields when source metadata is missing.

## Timing

Session wall duration is the distance between the first and last valid event
timestamp in the file.

Model duration is approximate. Local JSONL does not always expose exact model
call boundaries, so AgentMeter uses explicit usage events, previous token count
events, and tool completion timestamps as available boundaries.

Tool duration is the sum of matched tool start/end timestamp differences.

## Support Level

Support is intentionally pragmatic:

- Codex has the deepest documented event-shape support.
- Claude Code, CodeBuddy, and WorkBuddy support covers observed local message,
  usage, and tool-call shapes used by tests and current parsing logic.
- Generic JSONL support is best-effort over known usage and event fields.
- Unknown event types are retained as raw events when possible, but they may not
  contribute to usage, model-call, or tool-call aggregates.

When adding parser support, update this document, parser tests, and the shared
validation contract in [Validation](validation.md).

## Edge Cases

The parser treats these as non-fatal warnings:

- malformed JSONL lines;
- missing timestamps;
- missing session metadata;
- empty files;
- pending tool calls without an output event.

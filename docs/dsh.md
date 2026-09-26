# DeepSeek Harness (dsh)

AgentMeter reads local [DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness)
session history. The source family is `dsh`, displayed as **DeepSeek Harness**
in the shared Web/TUI source lists and session details.

## Source directories

The default source is `~/.dsh`, with logs under `~/.dsh/sessions`. `DSH_HOME`
is also discovered. Add the home or its `sessions` directory in Settings, then
run Update Index. Existing saved source lists are not silently changed.
Nested project/session directories under that root are also recognized.

The scanner reads `session.jsonl` and `session.vN.jsonl`, including their
`.jsonl.zstd` equivalents. It chooses the highest generation per session
directory; compressed files win when both encodings of that generation exist.
Migration replaces an older indexed generation only after the newer one parses
successfully. No source files are modified.

## Accounting

- Session headers supply identity, project path, creation time, and subagent role.
- Request headers and assistant message sources supply model/provider identity.
- Final assistant messages supply per-call usage. Stream snapshots do not add
  a second copy of those tokens. Settled attempts without messages use the last
  embedded usage snapshot when available.
- DSH's disjoint input, cache-read, and cache-write counters are normalized to
  AgentMeter's inclusive input count. Reasoning tokens remain an output subset.
  Cache writes use ordinary input pricing under the shared estimate contract.
- Inherited fork history and surface replacements do not add billable usage.
  Tool calls/results are paired by call ID, with timestamps and failure status.
- Unknown models remain unpriced. DeepSeek estimates automatically select peak/off-peak
  rates per call, then sum the session cost. Incomplete call histories fall back
  to session start time. See [Pricing Sources](pricing-sources.md) for boundaries,
  holiday coverage, and missing-time assumptions.

This adapter was checked against upstream commit
[`477b4f4`](https://github.com/deepseek-ai/deepseek-harness/tree/477b4f420553e8a52c2fbccc464d7561b239c443)
on 2026-09-26. Released JSONL formats 0–4 are accepted; newer format versions
are reported as indexing errors instead of silently producing incomplete costs.
This integration covers usage/history; it does not add dsh privacy-setting or
resource-management controls.

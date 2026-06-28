# Model Signals

Model Signals summarizes operational behavior that can be inferred from local
coding-agent sessions. These signals are operational measurements, not a model
capability leaderboard, and they do not assign a universal grade. They describe
how a provider/model behaved in real local usage for the selected agent/source
and project scope, using the available local token, timing, model-call, cost,
and tool-call data.

Use Model Signals to find patterns worth investigating, such as models that
produce unusually long outputs, show slower observed throughput, miss cache more
often, or start failing calls in a specific project. Do not use a single signal
as proof that one model is better or worse than another.

The feature has two layers:

- **Raw operational signals** stay close to the observed data: tokens, duration,
  throughput, cache shape, tool calls, model calls, and anomaly sessions.
- **Model Health** compares current behavior with the same cohort's own recent
  baseline and surfaces service-health, behavior-drift, and operational
  symptoms that deserve review.

Model Health is designed for both single long-running users who mostly use one
agent and one model, and teams or power users who compare many
provider/model x agent/source x project cohorts. The primary comparison is
self-comparison over time. Peer comparison can provide context, but it is not
proof that a model is generally better.

## Health And Drift

The core health cohort is:

```text
provider/model + agent/source + project
```

When filters are applied, the current window is the latest observed 24 hours in
the filtered scope. The baseline window is the preceding 30 days when enough
matching data is available. If the filtered scope only has recent data, the API
should mark the baseline as missing or low confidence instead of reporting a
regression.

Health interpretation should:

- Compare each cohort against its own baseline first.
- Show low sample and low confidence states explicitly.
- Treat missing baseline data as "not enough history", not as a regression.
- Use peer cohorts only as supporting context for triage.
- Preserve raw signal values so users can inspect the numerator, denominator,
  and source scope behind a health label.

## Signals

Stronger service-health signals:

- **Latency per 1k output tokens:** model-call duration normalized by output
  size. This is more useful than raw latency when output lengths vary.
- **Throughput:** observed token throughput over model-call duration, including
  output-token throughput where available.
- **Model call status and errors:** failed, retried, or errored model-call data
  when the source exposes it.
- **Token and cost shape:** shifts in input, cached input, output, reasoning,
  and cost mix for the same cohort.

Weaker operational symptoms:

- **Tool failure rate:** failed tool calls divided by total tool calls. Failures
  may reflect invalid commands, missing files, permission issues, parser status,
  or normal exploratory attempts. Review sessions before treating this as a
  model or provider issue.
- **Model calls per session:** useful for spotting repeated repair loops, but
  strongly affected by task size and agent strategy.
- **Output expansion rate:** output tokens divided by input tokens. Higher
  values usually mean the model generated more text relative to the prompt and
  context it received. This can indicate useful synthesis, verbose answers, or
  repeated repair loops depending on the session.
- **Cache miss rate:** uncached input tokens divided by total input tokens. A
  higher rate means less observed prompt/context reuse. It can be affected by
  prompt shape, project churn, cache eligibility, model/provider behavior, and
  whether the session data reports cached input tokens.
- **Reasoning token share:** reasoning output tokens divided by total output
  tokens when the source exposes reasoning tokens. Higher shares can suggest
  more hidden reasoning effort, but availability and accounting differ by agent,
  model, and provider.
- **Tool dependency rate:** sessions with at least one tool call divided by
  total sessions in the selected row or scope. Higher dependency can mean the
  agent is doing more repository inspection, command execution, or file work
  rather than only chatting.
- **Anomaly sessions:** sessions that cross fixed review thresholds for signals
  such as high reasoning share, high output/input ratio, slow observed model
  throughput, failed tool calls, or high cache miss. They are triage pointers
  for session review, not automatic defects.

## Denominators And Caveats

Rates depend on their denominator. A cache miss rate over very few input tokens,
a tool failure rate over one or two tool calls, or throughput from one short
model call can move sharply and should be treated as low confidence.

Small samples are volatile. Compare cohorts over similar projects, task types,
date ranges, and agent families before drawing conclusions. A model used for a
large refactor will naturally have different tool and token behavior than a
model used for short Q&A.

Some data is only as complete as the local session format. Missing token usage,
missing cached-token fields, unknown model-call boundaries, or incomplete
duration markers can lower confidence in the derived signal. AgentMeter keeps
these as operational signals because they are still useful for spotting local
workflow changes and outliers.

## Recommended Interpretation

Read the signals together:

- Prefer trends over isolated sessions.
- Start with the same cohort's current-versus-baseline movement.
- Compare peer models only under the same filters for agent/source, project,
  and date range.
- Treat high anomaly counts as a prompt to inspect session details.
- Pair tool dependency and tool failure rates with actual tool-call history.
- Treat low-sample rows as directional only.

The `/api/model-signals` endpoint supports the same analytics filters as
overview and token analytics: `agent`, `model`, `project`, `from`, and `to`.
Existing collection fields such as `trend`, `modelBreakdown`, and
`anomalySessions` remain raw operational signal views. The Model Health layer
adds:

- `healthSummary`: scope-level current window, baseline availability, sample,
  confidence, and strongest observed health/drift indicators.
- `cohorts`: provider/model + agent/source + project health rows.
- `matrix`: a cross-cohort view for scanning provider/model, agent/source, and
  project combinations.
- `projectHotspots`: projects whose current cohort behavior most deserves
  review.

Collection fields should return empty arrays (`[]`) when no rows match. Object
fields such as `healthSummary` should still be present and should report
missing baseline or low confidence explicitly when data is sparse.

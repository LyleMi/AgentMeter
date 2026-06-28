# Model Signals

Model Signals summarizes operational behavior that can be inferred from local
coding-agent sessions. These signals are operational measurements, not a model
capability leaderboard, and they do not assign a universal grade. They describe
how a provider/model behaved in real local usage for the selected agent/source,
project, and day scope, using the available local token, timing, model-call,
cost, cache, retry, failure, and tool-call data.

Use Model Signals to find patterns worth investigating, such as models that
produce unusually long outputs, show slower observed throughput, miss cache more
often, burn more cost per session or active hour, or start failing calls in a
specific project. Do not use a single signal as proof that one model is better
or worse than another.

The feature has three related views:

- **Raw operational signals** stay close to the observed data: tokens, duration,
  throughput, cache shape, tool calls, model calls, and anomaly sessions.
- **Daily and project efficiency metrics** summarize cost, cache savings,
  latency, throughput, failure pressure, retry pressure, sample confidence, and
  drift at day and project granularity.
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

Daily metrics use day-level rows and compare each day against the preceding
7 calendar days when enough history exists. Project metrics compare the current
project behavior against available baseline behavior for the same filtered
scope. In both cases, drift is a local operational change indicator, not a
general model capability score.

## Presentation Standard

Model Signals should lead with configurable dynamic charts, not raw numeric
tables. The primary view should let users switch between operational lenses
such as P90 latency, P10 throughput, cost burn, cost per active hour, cost per
1k tokens, cache savings, failure pressure, retry pressure, cache miss rate,
reasoning share, and output expansion.

Tables remain useful as inspectable detail, but they should sit after the chart
for traceability. Chart controls should preserve the active agent/source, model,
project, and date filters, show current versus baseline values where available,
and label low-sample or unavailable-price states directly.

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

Operational efficiency metrics:

- **Cost burn:** total observed estimated cost for a day or project.
- **Cost per session:** estimated cost divided by sessions in the same row.
- **Cost per active hour:** estimated cost divided by measured active model and
  tool time when active duration is available.
- **Cost per 1k tokens:** estimated cost divided by total token volume. This is
  the preferred normalized cost lens when comparing days or projects with very
  different session counts.
- **Cache savings:** estimated avoided cost from cached input tokens when both
  cached-token data and pricing are available.
- **Latency percentiles:** p50 and p90 latency from observed model-call
  token/duration rows when available, with session-level fallbacks when the
  source does not expose per-call token counts.
- **Throughput percentiles:** p50 and p10 observed throughput using the same
  model-call-first, session-fallback sample rule, so slow-tail throughput
  remains visible.
- **Failure pressure:** failed or errored model-call and tool-call pressure,
  depending on what the source exposes.
- **Retry pressure:** repeated model-call pressure, including model calls per
  session as a proxy for repair loops or larger tasks.

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

Missing pricing is a confidence and completeness limitation, not an indexing or
model failure. Cost and cache-savings metrics should clearly show unavailable or
partial pricing rather than silently treating unknown prices as zero. Low sample
rows should also be explained as risk or low confidence, not as failed behavior.

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
`anomalySessions` remain raw operational signal views. The endpoint also
returns day-level and project-level efficiency views:

- `dailyMetrics`: day rows for operational efficiency, including cost, cost per
  session, cost per active hour, cost per 1k tokens, cache savings, p50/p90
  latency, p50/p10 throughput, failure pressure, retry pressure or model calls
  per session, low sample flags, and drift against the preceding 7 calendar
  days.
- `projectMetrics`: project rows for operational efficiency, including project
  cost burn, cache savings, cost per session, cost per active hour, cost per 1k
  tokens, dominant model, model mix, retry pressure, failure pressure,
  confidence, and current-versus-baseline drift.

The Model Health layer adds:

- `healthSummary`: scope-level current window, baseline availability, sample,
  confidence, and strongest observed health/drift indicators.
- `cohorts`: provider/model + agent/source + project health rows.
- `matrix`: a cross-cohort view for scanning provider/model, agent/source, and
  project combinations.
- `projectHotspots`: projects whose current cohort behavior most deserves
  review.

Collection fields should return empty arrays (`[]`) when no rows match. Object
fields such as `healthSummary` should still be present and should report
missing baseline or low confidence explicitly when data is sparse. Missing
pricing, missing cache-token fields, or low samples should be represented as
confidence and completeness limits rather than as failures.

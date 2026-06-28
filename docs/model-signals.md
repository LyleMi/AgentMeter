# Model Signals

Model Signals summarizes operational behavior that can be inferred from local
coding-agent sessions. These signals are proxy measurements, not definitive
model quality scores. They describe how a model behaved in the observed
sessions, with the available local token, timing, and tool-call data.

Use Model Signals to find patterns worth investigating, such as models that
produce unusually long outputs, rely heavily on tools, miss cache more often, or
show slower observed throughput. Do not use a single signal as proof that one
model is better or worse than another.

## Signals

- **Output expansion rate:** output tokens divided by input tokens. Higher
  values usually mean the model generated more text relative to the prompt and
  context it received. This can indicate useful synthesis, verbose answers, or
  repeated repair loops depending on the session.
- **Reasoning token share:** reasoning output tokens divided by total output
  tokens when the source exposes reasoning tokens. Higher shares can suggest
  more hidden reasoning effort, but availability and accounting differ by agent,
  model, and provider.
- **Cache miss rate:** uncached input tokens divided by total input tokens. A
  higher rate means less observed prompt/context reuse. It can be affected by
  prompt shape, project churn, cache eligibility, model/provider behavior, and
  whether the session data reports cached input tokens.
- **Model throughput:** observed model tokens per second over model-call
  duration. AgentMeter also tracks output-token throughput. These are local
  observed rates, not provider benchmarks; network latency, streaming behavior,
  local machine load, retries, and incomplete timing markers can all affect
  them.
- **Tool dependency rate:** sessions with at least one tool call divided by
  total sessions in the selected row or scope. Higher dependency can mean the
  agent is doing more repository inspection, command execution, or file work
  rather than only chatting.
- **Tool failure rate:** failed tool calls divided by total tool calls. Failures
  may reflect invalid commands, missing files, permission issues, parser status,
  or normal exploratory attempts. Review sessions before treating this as a
  model problem.
- **Anomaly sessions:** sessions that cross fixed review thresholds for signals
  such as high reasoning share, high output/input ratio, slow observed model
  throughput, failed tool calls, or high cache miss. They are triage pointers
  for session review, not automatic defects.

## Denominators And Caveats

Rates depend on their denominator. A cache miss rate over very few input tokens,
a tool failure rate over one or two tool calls, or throughput from one short
model call can move sharply and should be treated as low confidence.

Small samples are volatile. Compare models over similar projects, task types,
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
- Compare models under the same filters for agent/source, project, and date
  range.
- Treat high anomaly counts as a prompt to inspect session details.
- Pair tool dependency and tool failure rates with actual tool-call history.
- Treat low-sample rows as directional only.

The `/api/model-signals` endpoint supports the same analytics filters as
overview and token analytics: `agent`, `model`, `project`, `from`, and `to`.
Collection fields such as `trend`, `modelBreakdown`, and `anomalySessions`
should return empty arrays (`[]`) when no rows match.

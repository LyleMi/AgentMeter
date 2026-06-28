# AgentMeter Compared With Other LLM Observability Tools

AgentMeter is not a replacement for every LLM observability product. It focuses
on a narrower problem: understanding local coding-agent sessions that already
exist as JSONL files on your machine.

For product scope, see [Project Brief](project-brief.md) and
[Roadmap](roadmap.md).

## Comparison Matrix

| Tool type | Primary data source | Best fit | Tradeoffs |
| --- | --- | --- | --- |
| AgentMeter | Local coding-agent JSONL files | Inspecting local Codex, Claude Code, CodeBuddy, WorkBuddy, and generic JSONL session history without a proxy or cloud service. | Limited to supported local file shapes; cost is an estimate from local usage and local pricing rows. |
| Proxy or gateway usage tools | Live API traffic passing through the proxy | Measuring requests that your app or workflow routes through a controlled endpoint. | Sessions that already happened outside the proxy may be invisible. Setup can require changing client configuration. |
| Cloud observability dashboards | Uploaded traces, spans, prompts, completions, or SDK events | Team dashboards, collaboration, hosted retention, alerting, and cross-service monitoring. | Data usually leaves the local machine by design, and setup depends on each product's ingestion path. |
| App instrumentation and OpenTelemetry-style tracing | Instrumented application code | Production app debugging, latency tracing, and service-level monitoring. | Local coding-agent history is not captured unless the agent workflow is instrumented. |
| Provider billing pages | Provider-side account usage | Official billing reconciliation for a provider account. | May not show local project/session context, tool calls, source files, or multi-agent local views. |

## When AgentMeter Fits

Use AgentMeter when you want to:

- inspect local coding-agent sessions after they happened;
- keep prompts, tool output, and audit evidence on your machine;
- compare usage across multiple local source roots;
- distinguish source instances from agent families;
- see token usage, cost estimates, durations, tool calls, and audit findings in
  one local dashboard;
- use a Web UI or TUI over the same local SQLite database.

## When Another Tool May Fit Better

A different tool may be a better fit when you need:

- hosted team dashboards or shared retention;
- real-time production alerting;
- gateway-level policy enforcement;
- request tracing across a distributed application;
- official provider billing totals for accounting.

AgentMeter can complement those tools by covering local session history that was
not routed through a proxy, SDK, or cloud tracing pipeline.

## Cost And Billing Caveat

AgentMeter cost numbers are estimates for local usage analysis. Pricing rows are
USD per 1M tokens and come from the local pricing registry. Subscription usage
inside coding agents and provider account billing may not map one-to-one to API
list prices.

See [Pricing Sources](pricing-sources.md) for current assumptions.

## Privacy Positioning

AgentMeter's privacy boundary is simple: read local files, store local SQLite,
and do not upload by default. Audit findings may store local raw evidence so
users can inspect why a finding appeared.

See [Privacy](privacy.md) for the full local data behavior.

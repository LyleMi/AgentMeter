# GitHub Repository Settings

This page gives maintainers exact repository metadata for discoverability and
trust. Keep it aligned with the current README, project brief, release workflow,
and website status.

## About Section

Recommended description:

```text
Local-first dashboard for Codex, Claude Code, CodeBuddy, WorkBuddy, and JSONL coding-agent session usage.
```

Recommended homepage:

```text
https://blog.lyle.ac.cn/AgentMeter/
```

This points to the mock-data GitHub Pages preview. Keep the URL aligned with
the Pages workflow and README links.

## Topics

Use these GitHub topics:

```text
ai
agent
agentic-coding
coding-agents
codex
claude-code
codebuddy
workbuddy
jsonl
sqlite
go
vue
local-first
developer-tools
llm-observability
token-usage
usage-analytics
privacy
tui
dashboard
```

## Social Preview

Use this repository asset:

```text
docs/assets/social-preview.png
```

Social preview guidance:

- Use a 1280x640 PNG.
- Show the AgentMeter name, logo, and a clean dashboard or terminal screenshot.
- Use synthetic data only.
- Do not show real prompts, file paths, repository names, session IDs, or audit
  evidence.
- Keep text short enough to remain readable in small link previews.
- Prefer a light/dark balanced image that still works when cropped by social
  platforms.

## Short Repository Pitch

Use this copy for listings that allow a slightly longer summary:

```text
AgentMeter is a local-first Go and Vue dashboard for indexing local coding-agent JSONL session files into SQLite, then inspecting token usage, estimated cost, durations, tool calls, and offline audit findings without a proxy or cloud service.
```

## Metadata Review Checklist

- Description matches the current supported source list.
- Homepage points to the canonical public preview.
- Topics stay under GitHub's topic limit.
- Social preview uses synthetic data.
- README, docs, and release notes do not claim installer signing or cloud sync.

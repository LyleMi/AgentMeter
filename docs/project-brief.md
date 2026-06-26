# AgentMeter Project Brief

## One-line Description

AgentMeter is an open-source, cross-platform-ready local dashboard for measuring
coding agent session usage, starting with Codex on Windows.

## Problem

Existing LLM observability tools are strong when they are placed in front of API
traffic or instrumented inside an application. They are less suitable when the
goal is to inspect local coding-agent sessions that already happened.

AgentMeter focuses on that gap:

- no proxy;
- no cloud service;
- no provider-side integration;
- no required code instrumentation;
- just local session files, parsed and displayed clearly.

## Initial Scope

The first version supports:

- Codex local sessions on Windows;
- token usage from local session data;
- session duration from local JSONL timestamps;
- tool-call statistics from local session events;
- SQLite indexing;
- local desktop UI.

## Non-goals For MVP

- Multi-agent support beyond Codex.
- Proxy or gateway mode.
- Cloud sync.
- Team dashboards.
- Authentication.
- Remote database.
- Automatic uploads or telemetry.
- Complex eval workflows.

## User

The first user is a developer who uses Codex locally and wants to understand:

- token burn;
- estimated cost;
- session history;
- active time versus wall time;
- model usage;
- tool-call behavior;
- project-level usage.

## Product Principles

- Read-only against Codex data.
- Prefer actual usage data over estimation.
- Clearly label unknown, missing, and estimated values.
- Keep data local unless the user explicitly exports it.
- Store normalized data so UI queries do not repeatedly rescan raw JSONL.
- Make raw source traceability possible for debugging.

## First Usable Version

The first useful build should let a user open AgentMeter, point it at a Codex
data directory if needed, index sessions, and inspect:

- overview totals;
- daily usage;
- session list;
- session detail timeline;
- model usage;
- tool-call counts and durations.

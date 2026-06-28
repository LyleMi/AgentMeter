# Privacy And Local Data

AgentMeter is designed as a local-first usage dashboard. It reads local
coding-agent session files, indexes normalized records into a local SQLite
database, and serves private local views over that data.

For architecture details, see [Architecture](architecture.md). For stored
fields, see [Data Model](data-model.md).

## What AgentMeter Reads

For session analysis, AgentMeter reads enabled local source roots such as:

```text
~/.codex
~/.claude
~/.codebuddy
~/.workbuddy
```

It also reads any custom JSONL directories you add in **Settings**.

For Agent Privacy features, AgentMeter can inspect supported user-level config
files for external agent tools. Privacy config targets are separate from indexed
source instances.

## What AgentMeter Stores

AgentMeter stores local application data in SQLite:

- configured source entries and labels;
- scanned JSONL file metadata, including path, size, modified time, and content
  hash;
- normalized sessions, events, usage, model calls, and tool calls;
- selected raw event JSON for traceability;
- local pricing registry rows and cost estimates;
- offline audit runs and findings.

Audit findings are local-only. They may include raw evidence, command previews,
file paths, URLs, or other text copied from local session events so that a user
can inspect why a finding appeared.

Default database path:

```text
Windows: %LOCALAPPDATA%\AgentMeter\agentmeter.sqlite
macOS:   ~/Library/Application Support/AgentMeter/agentmeter.sqlite
Linux:   $XDG_DATA_HOME/AgentMeter/agentmeter.sqlite or ~/.local/share/AgentMeter/agentmeter.sqlite
```

## What AgentMeter Does Not Upload

AgentMeter does not:

- upload session files, prompts, tool outputs, audit evidence, or database
  records;
- send telemetry;
- run as a proxy or gateway for model traffic;
- require a cloud service;
- call provider APIs to calculate routine local usage;
- sync your data to a remote database.

Normal indexing and local dashboard use should work without network access
after dependencies or release assets are already present.

## What AgentMeter Does Not Modify

AgentMeter does not modify source JSONL session files.

Indexing reads source files and writes AgentMeter's local SQLite cache. Removing
a source from **Settings** stops future indexing for that source but does not
delete the original session files.

Agent Privacy features are different: when you explicitly save or apply a
supported privacy profile, AgentMeter may write supported user-level config
files for the selected external agent target. Existing config files are backed
up before writes.

Routine smoke checks should not save settings, change privacy settings, click
**Update Index**, or click **Rebuild Index** unless the task explicitly requires
that state change. See [Validation](validation.md).

## Local HTTP Boundary

The Web UI binds to loopback by default:

```text
127.0.0.1:34115
```

Changing the HTTP bind address can expose the local dashboard beyond your
machine. Keep the default loopback address unless you have a specific reason and
understand the network boundary.

## Clearing Local AgentMeter Data

To clear AgentMeter's indexed cache and local app settings, stop AgentMeter and
delete the SQLite database listed above. This does not remove source JSONL files
or external agent config files.

If you use Agent Privacy writes, restore the backup files created next to the
changed external-agent config files or apply that target's default profile where
supported.

## Sharing Debug Information

When filing issues or security reports, redact:

- prompts and model responses;
- secrets and tokens;
- private file paths;
- repository names that should not be public;
- raw JSONL lines that include proprietary content.

Small synthetic JSONL examples are preferred when reporting parser bugs.

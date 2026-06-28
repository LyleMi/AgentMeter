# Security Policy

AgentMeter is local-first software. It reads local coding-agent session files,
stores a local SQLite database, and does not upload session data or telemetry by
design.

For privacy details, see [Privacy](docs/privacy.md).

## Supported Versions

Security fixes target the current main branch and the latest published release.
Older releases are handled on a best-effort basis.

## Reporting A Vulnerability

Do not open a public issue for a vulnerability that could expose private data,
enable code execution, bypass local trust boundaries, or corrupt user files.

Use GitHub private vulnerability reporting if it is enabled for the repository.
If it is not available, contact the maintainers through the repository's listed
security contact.

Include:

- affected version, commit, or release archive;
- operating system and install method;
- clear reproduction steps;
- expected and actual impact;
- whether the issue requires malicious local files, a local user account, or a
  network-exposed HTTP bind address.

Do not include real prompts, secrets, API keys, proprietary logs, or unredacted
JSONL session files. Synthetic examples are preferred.

## Scope

In scope:

- unintended upload or disclosure of local session data;
- modification or deletion of source JSONL session files;
- unsafe handling of local audit evidence;
- local HTTP exposure beyond the configured bind address;
- privacy config writes that occur without explicit user action;
- parser or importer behavior that can corrupt the AgentMeter database.

Out of scope:

- vulnerabilities in external coding-agent vendors;
- billing discrepancies between provider invoices and AgentMeter estimates;
- findings that require a user to intentionally expose the local HTTP server to
  an untrusted network;
- issues caused only by already compromised local accounts or file systems.

## Disclosure Expectations

Maintainers should acknowledge sensitive reports privately, investigate without
requesting private session logs, and publish fixes with clear release notes when
the issue affects users.

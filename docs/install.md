# Install And Run AgentMeter

AgentMeter is a local-first app for inspecting coding-agent JSONL session usage.
It runs on your machine, stores a local SQLite database, and serves the Web UI
on loopback by default.

For developer setup details, see [Getting Started](getting-started.md). For the
full validation contract, see [Validation](validation.md).

## Option 1: Portable Release

When a GitHub release is available, download the archive for your operating
system and CPU:

```text
AgentMeter-windows-amd64.zip
AgentMeter-windows-arm64.zip
AgentMeter-linux-amd64.tar.gz
AgentMeter-linux-arm64.tar.gz
AgentMeter-darwin-amd64.tar.gz
AgentMeter-darwin-arm64.tar.gz
```

Unpack the archive. The package includes the AgentMeter executable and built Web
assets.

Start the Web UI:

```powershell
.\AgentMeter.exe -start
```

On macOS or Linux:

```sh
./AgentMeter -start
```

Open the local dashboard if your browser does not open automatically:

```text
http://127.0.0.1:34115
```

Portable releases are not installer packages. Signing and notarization are
future packaging work, so your operating system may ask you to confirm that you
want to run the downloaded binary.

## Option 2: Go Toolchain Install

Requirements:

- Go matching the version in `go.mod`
- Node.js and pnpm 11.1.3 for Web startup when built Web assets are not already present

Install the command:

```sh
go install github.com/LyleMi/AgentMeter@latest
```

Make sure your Go binary directory is on `PATH`, then start the Web UI:

```sh
AgentMeter -start
```

On Windows, the installed command may be invoked as:

```powershell
AgentMeter.exe -start
```

You can also run without keeping an installed binary:

```sh
go run github.com/LyleMi/AgentMeter@latest start
```

Modern Go versions do not use `go get` to install command binaries. Use
`go install ...@latest` for persistent installs, or `go run ...@latest` for a
one-shot run.

## Option 3: Run From Source

Requirements:

- Go matching the version in `go.mod`
- Node.js and pnpm 11.1.3

From the repository root:

```sh
go run . -start
```

Start mode installs frontend dependencies when needed, rebuilds the Web assets
when they are missing or stale, starts the local Web app, and opens:

```text
http://127.0.0.1:34115
```

## First Launch

On first launch, use **Update Index** to index local session files. AgentMeter
detects common local agent homes when they exist:

```text
~/.codex
~/.claude
~/.codebuddy
~/.workbuddy
```

If your sessions live somewhere else, open **Settings** and add one source root
per line. Use a manual source label when the same agent family has more than
one local instance, such as `Work Codex` and `Personal Codex`.

For parser support and source identity details, see
[Supported Agents](supported-agents.md) and
[Session Formats](session-formats.md).

## Local Data Location

AgentMeter stores its SQLite database in the standard per-user application data
location for your OS:

```text
Windows: %LOCALAPPDATA%\AgentMeter\agentmeter.sqlite
macOS:   ~/Library/Application Support/AgentMeter/agentmeter.sqlite
Linux:   $XDG_DATA_HOME/AgentMeter/agentmeter.sqlite or ~/.local/share/AgentMeter/agentmeter.sqlite
```

Deleting this database removes AgentMeter's indexed cache and local app
settings. It does not delete your source JSONL session files.

## Terminal UI

AgentMeter also has a terminal UI over the same database and indexing pipeline.

From a portable release:

```sh
./AgentMeter -ui tui
```

From source:

```sh
go run . -ui tui
```

From a Go toolchain install:

```sh
AgentMeter -ui tui
```

See [UI Modes](ui-modes.md) for mode behavior and TUI keyboard shortcuts.

## Common Run Options

Use a different local HTTP address:

```sh
go run . -ui web -http 127.0.0.1:34116
```

Serve already built Web assets:

```sh
go run . -ui web -static frontend/dist
```

Skip opening the browser during start mode:

```sh
go run . -start -skip-browser
```

## Troubleshooting

- No sessions appear: add the correct source root in **Settings**, then run
  **Update Index**.
- The port is already in use: pass a different loopback address with `-http`.
- Costs show `unpriced`: AgentMeter indexed the usage, but the local pricing
  registry has no matching model rate.
- A JSONL file has parse warnings: AgentMeter keeps indexing non-fatal records
  where possible. See [Session Formats](session-formats.md) for supported
  shapes.

For privacy boundaries and local data behavior, see [Privacy](privacy.md).

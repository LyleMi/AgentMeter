# Changelog

All notable changes to AgentMeter are documented in this file.

## [v0.2.0] - 2026-06-28

### Added

- Added local Web startup through `go run . -start`, including frontend asset preparation and browser launch.
- Added terminal UI mode with synchronized overview, session, tool, audit, and settings views.
- Added multi-source agent support for Codex, CodeBuddy, WorkBuddy, Gemini CLI, Claude Code, and custom session sources.
- Added agent privacy controls for Codex, Gemini CLI, Claude Code, and CodeBuddy, with shared Web and TUI coverage.
- Added offline audit findings for risky shell commands, secret exposure, network installers, privilege changes, and persistence patterns.
- Added tool-call detail views, tool filters, duration sorting, and linked filtering from agent selections.
- Added overview subpages for summary, trends, breakdowns, and recent activity.
- Added frontend localization support and a Chinese README.
- Added API and browser smoke test coverage plus release validation documentation.

### Changed

- Replaced the Wails desktop runtime with a local-first Go HTTP server and Vue frontend bundle.
- Split settings, tools, and overview screens into focused subroutes.
- Refined the AgentMeter UI with shared page chrome, logo assets, responsive layout updates, and clearer summary metrics.
- Expanded the pricing model registry and documented pricing sources.
- Split frontend API, viewmodel, privacy editor, service, adapter, query, and session parsing boundaries for maintainability.
- Updated architecture, getting started, UI mode, data model, roadmap, and validation documentation.

### Fixed

- Corrected agent privacy API routes and edit/apply behavior.
- Fixed overview pricing coverage for model variants.
- Split chart runtime chunks to repair frontend build behavior.
- Repaired README logo and CI frontend build issues.

### Release Notes

- Windows release assets are packaged as `AgentMeter-windows-amd64.zip`.
- The zip includes the executable and built Web assets under `frontend/dist`; run `AgentMeter.exe` from the extracted directory.
- The Web UI listens on `127.0.0.1:34115` by default. Use `-ui tui` for terminal mode.

## [v0.1.0] - 2026-06-26

### Added

- Initial MVP release with the redesigned AgentMeter UI.

[v0.2.0]: https://github.com/LyleMi/AgentMeter/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/LyleMi/AgentMeter/releases/tag/v0.1.0

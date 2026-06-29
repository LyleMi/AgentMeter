# Changelog

All notable changes to AgentMeter are documented in this file.

## [Unreleased]

## [v0.4.0] - 2026-06-29

### Added

- Added Model Signals analytics for performance, health drift, degradation risk, model quality risk, multi-metric charts, and source/model comparisons.
- Added token cache-hit analytics with trend, day, project, and source comparison views.
- Added TUI pages and navigation for model signals, model risk, token analytics, time analytics, and tool-call drilldowns.
- Added custom model pricing support and startup behavior that works better with `go install` builds.
- Added read-only API smoke coverage for overview, tokens, model signals, sessions, tools, audit, pricing, settings, and privacy status.

### Changed

- Split large frontend views, model-signal chart logic, token analytics, route query handling, and backend API helper code into focused modules.
- Consolidated privacy adapters, shared usage-scope helpers, display formatting, and Web/TUI analytics presentation behavior.
- Updated install, architecture, pricing, model-signal, validation, session-format, roadmap, and UI documentation for the expanded analytics surface.

### Fixed

- Fixed routed time-detail pages and improved responsive layout behavior.
- Corrected reasoning-token accounting, token mix ratios, cache-input handling, and pricing suffix fallback behavior.
- Improved Model Signals chart ordering and label overflow handling.

### Release Notes

- Release assets are packaged as `AgentMeter-windows-amd64.zip`, `AgentMeter-windows-arm64.zip`, `AgentMeter-linux-amd64.tar.gz`, `AgentMeter-linux-arm64.tar.gz`, `AgentMeter-darwin-amd64.tar.gz`, and `AgentMeter-darwin-arm64.tar.gz`.
- Each release includes `checksums.txt`; archives include the executable, built Web assets, README files, license, and changelog.

## [v0.3.0] - 2026-06-28

### Added

- Added a dedicated time attribution page with session duration, time composition, slow-session, and tool-duration views.
- Added a dedicated tokens page for token usage and estimated-cost analysis.
- Added scoped usage breakdowns and source-instance identity so usage can be filtered by agent family, source instance, and project.
- Added Cursor source detection and expanded privacy profile workflows across the Web UI, TUI, and CLI.
- Added audit token and shell-command views, including shell-command filtering for tool analytics.
- Added a static GitHub Pages demo plus repository screenshot and social preview assets for public project pages.
- Added multi-platform release archives for Windows, Linux, and macOS on amd64 and arm64, with SHA256 checksums.

### Changed

- Split audit and tool analytics into focused summary, detail, findings, tool-call, and shell-command routes.
- Consolidated Web UI page shell patterns, display preferences, compact number formatting, and responsive dashboard styling.
- Refined query aggregation for scoped usage, estimated costs, orphan token records, and source filters.
- Updated installation, supported-agent, privacy, release-distribution, repository setup, and validation documentation.

### Fixed

- Fixed GitHub Pages deployment and demo links so the live preview uses the default Pages URL.
- Fixed estimated-cost aggregation and scope filter range behavior.
- Ignored orphan token usage costs so detached token records do not skew totals.

### Release Notes

- Release assets are packaged as `AgentMeter-windows-amd64.zip`, `AgentMeter-windows-arm64.zip`, `AgentMeter-linux-amd64.tar.gz`, `AgentMeter-linux-arm64.tar.gz`, `AgentMeter-darwin-amd64.tar.gz`, and `AgentMeter-darwin-arm64.tar.gz`.
- Each release includes `checksums.txt`; archives include the executable, built Web assets, README files, license, and changelog.

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

[Unreleased]: https://github.com/LyleMi/AgentMeter/compare/v0.4.0...HEAD
[v0.4.0]: https://github.com/LyleMi/AgentMeter/compare/v0.3.0...v0.4.0
[v0.3.0]: https://github.com/LyleMi/AgentMeter/compare/v0.2.0...v0.3.0
[v0.2.0]: https://github.com/LyleMi/AgentMeter/compare/v0.1.0...v0.2.0
[v0.1.0]: https://github.com/LyleMi/AgentMeter/releases/tag/v0.1.0

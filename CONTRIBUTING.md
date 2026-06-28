# Contributing To AgentMeter

Thanks for helping improve AgentMeter. The project is a local-first Go and Vue
app that indexes local coding-agent JSONL session files into SQLite.

## Before You Start

- Read [Project Brief](docs/project-brief.md) for scope and non-goals.
- Read [Architecture](docs/architecture.md) for package boundaries.
- Read [Validation](docs/validation.md) before choosing checks.
- Keep source session files read-only.
- Do not include private prompts, secrets, raw user session logs, or proprietary
  repository data in issues, tests, fixtures, screenshots, or pull requests.

## Development Setup

For normal local use:

```sh
go run . -start
```

For manual Web startup:

```sh
cd frontend
npm ci
npm run build
cd ..
go run . -ui web -http 127.0.0.1:34115
```

For TUI mode:

```sh
go run . -ui tui
```

See [Getting Started](docs/getting-started.md) for additional commands.

## Coding Guidelines

- Use `gofmt` for Go.
- Keep Go packages small, lowercase, and purpose-specific.
- Put Go tests beside the package under test with `_test.go` filenames.
- Use Vue 3 and TypeScript conventions already present in `frontend/src`.
- Keep frontend indentation at two spaces, use single quotes, and omit
  semicolons.
- Prefer shared backend query/view-model semantics over UI-only business rules.
- Keep Web and TUI behavior aligned when changing shared user-visible concepts.

## Validation

Use the narrowest validation that covers your change. Common checks:

```sh
go test ./...
```

```sh
cd frontend
npm ci
npm run build
cd ..
```

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke-api.ps1 -BaseUrl http://127.0.0.1:34115
```

```sh
cd frontend
npm run test:smoke
cd ..
```

Routine smoke checks should remain read-only unless your task explicitly
requires indexing, rebuilding, settings writes, or privacy config changes.

For documentation-only changes, text review plus targeted `rg` checks is
usually enough.

## Pull Requests

Use Conventional Commits for commit subjects:

```text
type(scope): imperative subject
```

Allowed types are `feat`, `fix`, `docs`, `test`, `refactor`, `style`, `chore`,
and `ci`.

PRs should include:

- a concise behavior summary;
- validation performed;
- screenshots for visible Web UI changes;
- notes about Web/TUI parity when shared behavior changes;
- links to related issues when relevant.

Keep PRs focused. Avoid unrelated formatting churn, generated asset changes, or
large fixture updates unless they are necessary for the change.

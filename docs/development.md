# Development

## Repository layout

```text
cmd/ottrta        CLI entrypoint
internal/agent    Agent integrations and agent-specific tests
internal/cli      Cobra commands and command wiring
internal/event    Event types
internal/session  Session lifecycle, persistence, shells, and PTY handling
internal/tui      Bubble Tea model, update loop, views, styles, and key bindings
```

## Architecture overview

- `cmd/ottrta/main.go` starts the Cobra-based CLI.
- `internal/cli` exposes the public commands such as `tui`, `run`, `shell`, `agent`, `update`, and `version`.
- `internal/session` manages session state, process execution, PTY shells, and persisted session metadata.
- `internal/tui` renders the interactive interface and handles keyboard-driven workflows.

## Local validation

Run the project test suite:

```sh
go test ./...
```

Build the executable:

```sh
go build ./cmd/ottrta
```

## Contribution notes

- Keep changes terminal-first and aligned with the current Cobra + Bubble Tea architecture.
- Prefer small, focused changes that preserve existing session and agent workflows.
- Update the root README and relevant files in `docs/` when user-facing behavior changes.

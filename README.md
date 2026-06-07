# OTTRTA

**One TUI To Rule Them All** is a Go-based terminal workspace for launching, monitoring, and attaching to multiple shell and AI-agent sessions from one place.

## Why OTTRTA

OTTRTA is built for workflows where several terminal-driven agents or shell processes need to run side by side. It combines a focused TUI with lightweight CLI commands for starting sessions, launching agents, and working with interactive PTY shells.

## Highlights

- Terminal-first workflow with a Bubble Tea powered TUI
- Two-column session dashboard with live status and logs
- Foreground command execution via `ottrta run`
- Interactive PTY shell support via `ottrta shell`
- Agent orchestration for `omp`-based sessions
- Session persistence across TUI restarts
- Cross-platform installation scripts for Linux, macOS, and Windows

## Installation

### macOS / Linux

```sh
curl -fsSL https://raw.githubusercontent.com/thilob97/ottrta/main/install.sh | sh
```

### Windows (PowerShell)

```powershell
powershell -c "irm https://raw.githubusercontent.com/thilob97/ottrta/main/install.ps1 | iex"
```

The installer downloads the latest GitHub release and installs both `ottrta` and the `rta` alias.

More details: [docs/installation.md](docs/installation.md)

## Quick start

```sh
ottrta
ottrta run --name go-version --command go --args version
ottrta shell
ottrta agent start omp --count 2
```

## Commands

| Command | Purpose |
| --- | --- |
| `ottrta` / `ottrta tui` | Start the terminal UI |
| `ottrta run` | Run a foreground process without PTY emulation |
| `ottrta shell` | Start an interactive PTY shell session |
| `ottrta agent start omp` | Start one or more `omp` agent sessions |
| `ottrta update` | Print the platform installer URL and command to run explicitly |
| `ottrta version` | Print the current OTTRTA version |

Usage details: [docs/usage.md](docs/usage.md)

## TUI essentials

### Navigation

- `j` / `k` or arrow keys: move in the session grid
- `h` / `l` or arrow keys: move between columns
- `enter`: attach to a running session or focus logs
- `esc`: leave logs or detach from an attached session
- `q` / `ctrl+c`: quit monitor mode

### Session actions

- `space`: start or stop the selected session
- `n`: create a new agent session
- `r`: rename the selected session
- `x`: remove the selected session

## Persistence

When the TUI exits cleanly, OTTRTA saves session definitions and reloads them on the next start. Persisted data includes identifiers, command metadata, working directory, and agent kind, but not terminal logs. Malformed persisted definitions are rejected instead of being loaded silently.

## Documentation

- [docs/README.md](docs/README.md) — documentation index
- [docs/installation.md](docs/installation.md) — install and update flows
- [docs/usage.md](docs/usage.md) — commands, workflows, and TUI controls
- [docs/development.md](docs/development.md) — architecture and local development

## Development

```sh
go fmt ./...
go test ./...
go build ./cmd/ottrta
```

Additional notes: [docs/development.md](docs/development.md)

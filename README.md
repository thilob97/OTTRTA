# OTTRTA

One Terminal To Rule Them All — a small Go TUI for orchestrating, monitoring, and attaching to multiple CLI-based AI agents and shell sessions.

## Installation

### macOS / Linux

```sh
curl -fsSL https://raw.githubusercontent.com/thilob97/ottrta/main/install.sh | sh
```

The script automatically detects your OS and architecture, downloads the latest release binary, and installs it to `/usr/local/bin` (or `~/.local/bin` if root permissions are not available).

### Windows (PowerShell)

```powershell
powershell -c "irm https://raw.githubusercontent.com/thilob97/ottrta/main/install.ps1 | iex"
```

The PowerShell script downloads the latest zip archive, extracts `ottrta.exe` to `~/.ottrta/bin`, and appends the directory to your user's `Path` environment variable.

## Commands

```sh
go run ./cmd/ottrta
go run ./cmd/ottrta tui
go run ./cmd/ottrta run --name test --command go --args version
go run ./cmd/ottrta shell
go run ./cmd/ottrta shell --command pwsh
go run ./cmd/ottrta version
```

## TUI keys

- `j` / `k` (or arrow keys `up` / `down`): move vertically in the 2-column session grid
- `h` / `l` (or arrow keys `left` / `right`): move horizontally in the 2-column session grid
- `space`: start/stop the selected session
- `enter`: attach immediately if session is running; otherwise focus the log panel
- `esc`: from log panel, returns focus to the session list; from attach mode, detaches
- `n`: add a new agent session. Asks for a **command** (e.g. `omp`, `claude`, `etc.`) and then a **working directory** (use `tab` for directory completion).
- `r`: rename the selected session; `enter` saves and `esc` cancels
- `x`: remove the selected session
- `q` / `ctrl+c`: quit in monitor mode

## TUI layout

- A stylish custom ASCII-art OTTRTA brand banner is displayed in the top-left area.
- Sessions render as a two-column grid of spacious, colorized imp cards showing the avatar, name, session ID, and status side-by-side.
- The session panel stays only wide enough for two cards; extra space goes to the log panel.
- Running sessions animate their imp banner; stopped sessions show `zZzZ`.
- Newly created agent sessions get short AI-slop-themed imp display names; stable session IDs are still persisted and used internally.
- The TUI runs in Bubble Tea's alternate screen, so it does not grow terminal scrollback while running.

## Persistence

- `ottrta tui` saves session definitions on clean exit and reloads them on the next start.
- Persisted data is limited to session ID, name, kind, command, args, workdir, agent kind, and task ID. Logs are not persisted.

## v0.3 PTY sessions

- Fake sessions remain available in the TUI.
- The TUI includes `real-go-version`, a stopped process session that runs `go version`.
- The TUI includes `shell-1`, a stopped PTY/ConPTY session using the default host shell.
- `ottrta run` executes a command in the foreground and streams stdout/stderr without PTY.
- `ottrta shell` starts an interactive PTY/ConPTY shell; use `--command` and repeated `--args` to override it.
- Attach mode forwards keyboard input to the active PTY session and shows PTY output in the log panel.
- Rendering is intentionally simple; ANSI/VT control sequences are stripped instead of emulated.

# OTTRTA

One Terminal To Rule Them All — a small Go TUI for fake sessions, foreground process sessions, and basic PTY/ConPTY shell sessions.

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

- `j` / `k`: move between sessions
- `h` / `l`: switch panel focus
- `space`: start/stop the selected session
- `enter`: attach to a running PTY session, otherwise focus the log panel
- `esc`: detach from an attached PTY session
- `n`: add a new stopped OMP agent session; enter a working directory path, use `tab` to complete directories, then `enter` creates or `esc` cancels
- `r`: rename the selected session; `enter` saves and `esc` cancels
- `x`: remove the selected session
- `q` / `ctrl+c`: quit in monitor mode

## TUI layout

- Sessions render as a two-column grid of taller, colorized imp cards.
- The session panel stays only wide enough for two cards; extra space goes to the log panel.
- Running sessions animate their imp banner; stopped sessions show `zZzZ`.
- Newly created agent sessions get short AI-slop-themed imp display names; stable session IDs are still persisted and used internally.
- The TUI runs in Bubble Tea's alternate screen, so it does not grow terminal scrollback while running.

## Persistence

- `ottrta tui` saves session definitions on clean exit and reloads them on the next start.
- Persisted data is limited to session ID, name, kind, command, args, workdir, agent kind, and task ID. Logs and task metadata are not persisted.

## v0.3 PTY sessions

- Fake sessions remain available in the TUI.
- The TUI includes `real-go-version`, a stopped process session that runs `go version`.
- The TUI includes `shell-1`, a stopped PTY/ConPTY session using the default host shell.
- `ottrta run` executes a command in the foreground and streams stdout/stderr without PTY.
- `ottrta shell` starts an interactive PTY/ConPTY shell; use `--command` and repeated `--args` to override it.
- Attach mode forwards keyboard input to the active PTY session and shows PTY output in the log panel.
- Rendering is intentionally simple; ANSI/VT control sequences are stripped instead of emulated.

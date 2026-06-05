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
- `q`: quit in monitor mode

## v0.3 PTY sessions

- Fake sessions remain available in the TUI.
- The TUI includes `real-go-version`, a stopped process session that runs `go version`.
- The TUI includes `shell-1`, a stopped PTY/ConPTY session using the default host shell.
- `ottrta run` executes a command in the foreground and streams stdout/stderr without PTY.
- `ottrta shell` starts an interactive PTY/ConPTY shell; use `--command` and repeated `--args` to override it.
- Attach mode forwards keyboard input to the active PTY session and shows PTY output in the log panel.
- Rendering is intentionally simple; ANSI/VT control sequences are stripped instead of emulated.

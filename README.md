# OTTRTA

One Terminal To Rule Them All — a small Go TUI for fake sessions and simple foreground process sessions.

## Commands

```sh
go run ./cmd/ottrta
go run ./cmd/ottrta tui
go run ./cmd/ottrta run --name test --command go --args version
go run ./cmd/ottrta version
```

## TUI keys

- `j` / `k`: move between sessions
- `h` / `l`: switch panel focus
- `space`: start/stop the selected session
- `enter`: focus the log panel
- `q`: quit

## v0.2 process sessions

- Fake sessions remain available in the TUI.
- The TUI includes `real-go-version`, a stopped process session that runs `go version`.
- `ottrta run` executes a command in the foreground and streams stdout/stderr.
- No PTY/ConPTY, terminal emulation, attach mode, or background session registry yet.

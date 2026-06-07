# Usage

OTTRTA offers both an interactive TUI and focused CLI commands for terminal sessions, shells, agents, and tasks.

## Start the TUI

```sh
ottrta
```

or

```sh
ottrta tui
```

The TUI opens in the terminal alternate screen and shows a two-column session grid with a log panel.

## Core commands

### Run a foreground command

```sh
ottrta run --name go-version --command go --args version
```

Use repeated `--args` flags to pass multiple arguments. Add `--workdir` to run the command in a specific directory.

### Open an interactive shell

```sh
ottrta shell
```

You can override the default shell:

```sh
ottrta shell --command pwsh
```

### Start agent sessions

```sh
ottrta agent start omp --count 2 --workdir /path/to/project
```

At the moment, `omp` is the supported agent kind.

### Start a task

```sh
ottrta task start "Compare approaches" --agents omp:3 --workdir /path/to/project
```

Tasks create grouped agent sessions. The default agent request is `omp:1`.

### List tasks

```sh
ottrta task list
```

### Show the version

```sh
ottrta version
```

### Update the installation

```sh
ottrta update
```

## TUI key bindings

### Navigation

- `j` / `k` or arrow keys: move vertically
- `h` / `l` or arrow keys: move horizontally
- `enter`: attach to a running session or focus the log panel
- `esc`: return from log focus or detach from attach mode
- `q` / `ctrl+c`: quit

### Session management

- `space`: start or stop the selected session
- `n`: add a new agent session
- `r`: rename the selected session
- `x`: remove the selected session

## Persistence behavior

The TUI stores session definitions on clean exit and reloads them on the next launch. Stored data includes:

- session ID and display name
- session kind
- command and arguments
- working directory
- agent kind
- task ID

Terminal output is not persisted.

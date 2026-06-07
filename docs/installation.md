# Installation

OTTRTA is distributed through GitHub releases and ships with install scripts for Unix-like systems and Windows.

## Supported platforms

- Linux (`amd64`, `arm64`)
- macOS (`amd64`, `arm64`)
- Windows (`amd64`, `arm64`)

## Install on macOS or Linux

```sh
curl -fsSL https://raw.githubusercontent.com/thilob97/ottrta/main/install.sh | sh
```

The shell installer:

- detects the current OS and architecture
- resolves the latest release tag from GitHub
- downloads the release archive
- installs `ottrta` and `rta`
- prefers `/usr/local/bin` and falls back to `~/.local/bin` if needed

## Install on Windows

```powershell
powershell -c "irm https://raw.githubusercontent.com/thilob97/ottrta/main/install.ps1 | iex"
```

The PowerShell installer:

- resolves the latest release from GitHub
- downloads the Windows zip archive
- installs `ottrta.exe` and `rta.exe` into `~/.ottrta/bin`
- adds the install directory to the user `Path` if required

## Update to the latest release

After installation, run:

```sh
ottrta update
```

The command prints the installer URL and the platform-specific command to run. It does not download and execute remote scripts automatically; review the installer first, then run the printed command explicitly if you trust it.

## Build from source

Prerequisite: Go 1.25 or newer.

```sh
go build ./cmd/ottrta
```

To run directly from source during development:

```sh
go run ./cmd/ottrta
```

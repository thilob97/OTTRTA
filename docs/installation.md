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

### Recommended (PowerShell session already open)

```powershell
irm https://raw.githubusercontent.com/thilob97/ottrta/main/install.ps1 | iex
```

Run this in an already opened PowerShell session (`powershell`, `pwsh`, or Windows Terminal PowerShell profile).

The PowerShell installer runs without admin privileges and:

- resolves the latest release from GitHub
- downloads the Windows zip archive
- installs `ottrta.exe` and `rta.exe` into `~/.ottrta/bin`
- adds the install directory to the user `Path` if required

### Without remote-script pipe (manual, user scope)

Use this when your environment blocks `irm ... | iex`:

1. Open the latest release page: `https://github.com/thilob97/ottrta/releases/latest`
2. Download the archive matching your system:
   - `ottrta_<version>_windows_amd64.zip` or
   - `ottrta_<version>_windows_arm64.zip`
3. Extract it into a user-writable install directory, for example:
   - `%USERPROFILE%\.ottrta\bin`
4. Ensure both `ottrta.exe` and `rta.exe` are present in that directory.
5. Add the directory to your user `Path` if it is not already there.
6. Open a new terminal and verify:

```powershell
ottrta version
```

### Manual fallback for restrictive company policies

If direct GitHub downloads are blocked, download the same release ZIP through your approved internal artifact mirror, then follow the same extraction and user-`Path` steps above.

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

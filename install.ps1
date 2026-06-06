$owner = "thilob97"
$repo = "ottrta"

# Detect Arch
$arch = $env:PROCESSOR_ARCHITECTURE
if ($arch -eq "AMD64") {
    $arch = "amd64"
} elseif ($arch -eq "ARM64") {
    $arch = "arm64"
} else {
    Write-Error "Unsupported architecture: $arch"
    exit 1
}

# Resolve latest release from GitHub API
Write-Host "Checking latest release of $owner/$repo..."
$latestReleaseUrl = "https://api.github.com/repos/$owner/$repo/releases/latest"

# Get Tag Name
try {
    # Set security protocol to TLS 1.2
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    $response = Invoke-RestMethod -Uri $latestReleaseUrl -UseBasicParsing
    $tag = $response.tag_name
} catch {
    # Fallback to redirect URL query
    $webResponse = Invoke-WebRequest -Uri "https://github.com/$owner/$repo/releases/latest" -MaximumRedirection 0 -ErrorAction SilentlyContinue
    $redirectUrl = $webResponse.Headers.Location
    $tag = $redirectUrl.Split('/')[-1]
}

if (-not $tag) {
    Write-Error "Could not detect latest release version."
    exit 1
}

$version = $tag.TrimStart('v')
Write-Host "Downloading ottrta $tag for Windows/$arch..."

# Construct download details
$fileName = "ottrta_${version}_windows_${arch}.zip"
$downloadUrl = "https://github.com/$owner/$repo/releases/download/$tag/$fileName"

# Create temp path
$tempDir = [System.IO.Path]::GetTempFileName()
Remove-Item $tempDir
New-Item -ItemType Directory -Path $tempDir | Out-Null

$zipPath = Join-Path $tempDir $fileName
$exePath = Join-Path $tempDir "ottrta.exe"

# Download the zip
Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing

# Extract
Expand-Archive -Path $zipPath -DestinationPath $tempDir -Force

if (-not (Test-Path $exePath)) {
    Write-Error "Could not find ottrta.exe in the extracted archive."
    exit 1
}

# Install directory: $HOME\.ottrta\bin
$installDir = Join-Path $HOME ".ottrta\bin"
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

$targetPath = Join-Path $installDir "ottrta.exe"
Copy-Item -Path $exePath -Destination $targetPath -Force

# Clean up temp
Remove-Item -Recurse -Force $tempDir

# Add to User PATH if not present
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$pathList = $userPath -split ";"
$alreadyInPath = $false

foreach ($p in $pathList) {
    if ($p -eq $installDir -or $p -eq "$installDir\") {
        $alreadyInPath = $true
        break
    }
}

if (-not $alreadyInPath) {
    Write-Host "Adding $installDir to User PATH..."
    $newUserPath = $userPath + ";" + $installDir
    [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
    Write-Host "Please restart your terminal to apply the PATH changes."
}

Write-Host "Successfully installed ottrta!"

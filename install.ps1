# MultiWANBond Windows Installer
# One-click installation script for Windows

param(
    [switch]$AutoYes = $false
)

$ErrorActionPreference = "Stop"

# Colors for output
function Write-Success { Write-Host $args -ForegroundColor Green }
function Write-Info { Write-Host $args -ForegroundColor Cyan }
function Write-Warn { Write-Host $args -ForegroundColor Yellow }
function Write-Err { Write-Host $args -ForegroundColor Red }

function Test-Administrator {
    $currentUser = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    return $currentUser.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

Write-Host ""
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host "       MultiWANBond Windows Installer" -ForegroundColor Cyan
Write-Host "================================================================" -ForegroundColor Cyan
Write-Host ""

# Check if running as administrator
if (-not (Test-Administrator)) {
    Write-Warn "This installer needs to run as Administrator."
    Write-Warn "Please right-click and select 'Run as Administrator'"
    Write-Host ""
    pause
    exit 1
}

# 1. Check Go installation
Write-Info "[1/7] Checking Go installation..."
$goInstalled = $false
$goVersion = ""

try {
    $goVersion = & go version 2>$null
    if ($goVersion -match "go(\d+)\.(\d+)") {
        $major = [int]$matches[1]
        $minor = [int]$matches[2]

        if ($major -ge 1 -and $minor -ge 21) {
            Write-Success "  [OK] Go $major.$minor is installed"
            $goInstalled = $true
        }
        else {
            Write-Warn "  [WARN] Go $major.$minor is installed but version 1.21+ is required"
        }
    }
}
catch {
    Write-Warn "  [WARN] Go is not installed"
}

if (-not $goInstalled) {
    Write-Host ""
    Write-Info "Go 1.21 or later is required to run MultiWANBond."

    if ($AutoYes) {
        $install = "Y"
    }
    else {
        $install = Read-Host "Would you like to download Go now? (Y/N)"
    }

    if ($install -eq "Y" -or $install -eq "y") {
        Write-Info "Opening Go download page..."
        Start-Process "https://go.dev/dl/"
        Write-Host ""
        Write-Warn "Please install Go and run this installer again."
        Write-Host ""
        pause
        exit 1
    }
    else {
        Write-Err "Go is required. Exiting installer."
        exit 1
    }
}

# 2. Check Git installation
Write-Info "[2/7] Checking Git installation..."
$gitInstalled = $false

try {
    $gitVersion = & git --version 2>$null
    if ($gitVersion) {
        Write-Success "  [OK] Git is installed"
        $gitInstalled = $true
    }
}
catch {
    Write-Warn "  [WARN] Git is not installed"
}

if (-not $gitInstalled) {
    Write-Host ""
    Write-Info "Git is recommended for easy updates."

    if ($AutoYes) {
        $install = "Y"
    }
    else {
        $install = Read-Host "Would you like to download Git now? (Y/N)"
    }

    if ($install -eq "Y" -or $install -eq "y") {
        Write-Info "Opening Git download page..."
        Start-Process "https://git-scm.com/download/win"
        Write-Host ""
        Write-Warn "Please install Git if you want easy updates."
        Write-Host ""
    }
}

# 3. Set up Go environment
Write-Info "[3/7] Setting up Go environment..."

$goPath = "c:\go-work"
$goModCache = "$goPath\pkg\mod"

# Set environment variables for current session
$env:GOPATH = $goPath
$env:GOMODCACHE = $goModCache
$env:GO111MODULE = "on"

# Set permanently for user
try {
    [Environment]::SetEnvironmentVariable("GOPATH", $goPath, "User")
    [Environment]::SetEnvironmentVariable("GO111MODULE", "on", "User")
    Write-Success "  [OK] Go environment configured"
}
catch {
    Write-Warn "  [WARN] Could not set environment variables permanently"
}

# Create Go directories
if (-not (Test-Path $goPath)) {
    New-Item -ItemType Directory -Path $goPath -Force | Out-Null
}
if (-not (Test-Path $goModCache)) {
    New-Item -ItemType Directory -Path $goModCache -Force | Out-Null
}

# 4. Download/Update MultiWANBond
Write-Info "[4/7] Installing MultiWANBond..."

$installDir = "$env:ProgramFiles\MultiWANBond"
$configDir = "$env:ProgramData\MultiWANBond"

if (Test-Path $installDir) {
    Write-Warn "  [WARN] MultiWANBond is already installed at $installDir"

    if ($AutoYes) {
        $update = "Y"
    }
    else {
        $update = Read-Host "Would you like to update it? (Y/N)"
    }

    if ($update -eq "Y" -or $update -eq "y") {
        Write-Info "  Updating MultiWANBond..."
        Set-Location $installDir
        if ($gitInstalled) {
            & git pull
        }
        else {
            Write-Warn "  Git not installed, skipping update"
        }
    }
}
else {
    Write-Info "  Downloading MultiWANBond..."

    if ($gitInstalled) {
        & git clone https://github.com/thelastdreamer/MultiWANBond.git $installDir
    }
    else {
        Write-Info "  Downloading ZIP (Git not available)..."
        $zipUrl = "https://github.com/thelastdreamer/MultiWANBond/archive/refs/heads/main.zip"
        $zipFile = "$env:TEMP\MultiWANBond.zip"

        Invoke-WebRequest -Uri $zipUrl -OutFile $zipFile
        Expand-Archive -Path $zipFile -DestinationPath "$env:TEMP\MultiWANBond-temp" -Force
        Move-Item "$env:TEMP\MultiWANBond-temp\MultiWANBond-main" $installDir -Force
        Remove-Item $zipFile
        Remove-Item "$env:TEMP\MultiWANBond-temp" -Recurse -Force
    }

    Write-Success "  [OK] MultiWANBond downloaded to $installDir"
}

# 5. Download and verify dependencies
Write-Info "[5/7] Downloading dependencies..."
Set-Location $installDir

try {
    & go mod tidy
    Write-Success "  [OK] Dependencies downloaded and verified"
}
catch {
    Write-Err "  [ERROR] Failed to download dependencies"
    Write-Err "  Error: $_"
    exit 1
}

# 6. Build MultiWANBond
Write-Info "[6/7] Building MultiWANBond..."

try {
    $env:CGO_ENABLED = "0"
    & go build -ldflags "-s -w" -o "$installDir\multiwanbond.exe" .\cmd\server\main.go
    Write-Success "  [OK] MultiWANBond built successfully"
}
catch {
    Write-Err "  [ERROR] Build failed"
    Write-Err "  Error: $_"
    exit 1
}

# 7. Create config directory and run setup wizard
Write-Info "[7/7] Running setup wizard..."

if (-not (Test-Path $configDir)) {
    New-Item -ItemType Directory -Path $configDir -Force | Out-Null
}

Write-Host ""
Write-Host "================================================================" -ForegroundColor Green
Write-Host "       Installation Complete!" -ForegroundColor Green
Write-Host "================================================================" -ForegroundColor Green
Write-Host ""
Write-Success "MultiWANBond has been installed to: $installDir"
Write-Success "Configuration will be stored in: $configDir"
Write-Host ""

# Run setup wizard
Write-Info "Starting setup wizard..."
Write-Host ""

& "$installDir\multiwanbond.exe" setup --config "$configDir\config.json"

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "================================================================" -ForegroundColor Green
    Write-Host ""
    Write-Success "Setup complete! MultiWANBond is ready to use."
    Write-Host ""
    Write-Info "To start MultiWANBond:"
    Write-Host "  cd `"$installDir`""
    Write-Host "  .\multiwanbond.exe --config `"$configDir\config.json`""
    Write-Host ""
    Write-Info "To manage configuration:"
    Write-Host "  .\multiwanbond.exe config --help"
    Write-Host ""
    Write-Info "To add/remove WAN interfaces later:"
    Write-Host "  .\multiwanbond.exe wan add"
    Write-Host "  .\multiwanbond.exe wan remove"
    Write-Host "  .\multiwanbond.exe wan list"
    Write-Host ""
    Write-Host "================================================================" -ForegroundColor Green
}
else {
    Write-Warn "Setup wizard was cancelled or encountered an error."
    Write-Info "You can run it again with:"
    Write-Host "  cd `"$installDir`""
    Write-Host "  .\multiwanbond.exe setup"
}

Write-Host ""
pause

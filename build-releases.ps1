# MultiWANBond - Multi-platform Build Script (PowerShell)
# Builds releases for Linux, Windows, and macOS

param(
    [string]$Version = "1.0.0"
)

$ErrorActionPreference = "Stop"

$BUILD_DIR = "build"
$RELEASE_DIR = "releases"

Write-Host "========================================"
Write-Host "MultiWANBond Build Script"
Write-Host "Version: $Version"
Write-Host "========================================"
Write-Host ""

# Clean previous builds
Write-Host "Cleaning previous builds..."
if (Test-Path $BUILD_DIR) { Remove-Item -Path $BUILD_DIR -Recurse -Force }
if (Test-Path $RELEASE_DIR) { Remove-Item -Path $RELEASE_DIR -Recurse -Force }
New-Item -ItemType Directory -Path $BUILD_DIR | Out-Null
New-Item -ItemType Directory -Path $RELEASE_DIR | Out-Null

# Build flags
$LDFLAGS = "-s -w -X main.version=$Version -X main.buildTime=$(Get-Date -Format 'yyyy-MM-dd_HH:mm:ss')"

# Build function
function Build-Binary {
    param(
        [string]$OS,
        [string]$ARCH,
        [string]$OUTPUT
    )

    Write-Host "Building for $OS/$ARCH..."

    $env:GOOS = $OS
    $env:GOARCH = $ARCH
    $env:CGO_ENABLED = "0"

    & go build -ldflags $LDFLAGS -o "$BUILD_DIR\$OUTPUT" .\cmd\server\main.go

    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ Successfully built $OUTPUT" -ForegroundColor Green
    } else {
        Write-Host "✗ Failed to build $OUTPUT" -ForegroundColor Red
        exit 1
    }
}

# Linux builds
Write-Host ""
Write-Host "Building Linux binaries..."
Write-Host "----------------------------"
Build-Binary "linux" "amd64" "multiwanbond-linux-amd64"
Build-Binary "linux" "arm64" "multiwanbond-linux-arm64"
Build-Binary "linux" "arm" "multiwanbond-linux-arm"

# Windows builds
Write-Host ""
Write-Host "Building Windows binaries..."
Write-Host "----------------------------"
Build-Binary "windows" "amd64" "multiwanbond-windows-amd64.exe"
Build-Binary "windows" "arm64" "multiwanbond-windows-arm64.exe"

# macOS builds
Write-Host ""
Write-Host "Building macOS binaries..."
Write-Host "----------------------------"
Build-Binary "darwin" "amd64" "multiwanbond-darwin-amd64"
Build-Binary "darwin" "arm64" "multiwanbond-darwin-arm64"

# Create release packages
Write-Host ""
Write-Host "Creating release packages..."
Write-Host "----------------------------"

# Helper function for creating archives
function Create-Archive {
    param(
        [string]$SourceFile,
        [string]$ArchiveName
    )

    $source = Join-Path $BUILD_DIR $SourceFile
    $destination = Join-Path $RELEASE_DIR $ArchiveName

    if ($ArchiveName.EndsWith(".zip")) {
        Compress-Archive -Path $source -DestinationPath $destination -Force
    } else {
        # For tar.gz, use tar if available (Windows 10 1803+)
        if (Get-Command tar -ErrorAction SilentlyContinue) {
            tar -czf $destination -C $BUILD_DIR $SourceFile
        } else {
            Write-Host "Warning: tar not found, creating .zip instead" -ForegroundColor Yellow
            $zipName = $ArchiveName -replace ".tar.gz$", ".zip"
            Compress-Archive -Path $source -DestinationPath (Join-Path $RELEASE_DIR $zipName) -Force
        }
    }

    Write-Host "✓ Created $ArchiveName" -ForegroundColor Green
}

# Linux packages
Create-Archive "multiwanbond-linux-amd64" "multiwanbond-$Version-linux-amd64.tar.gz"
Create-Archive "multiwanbond-linux-arm64" "multiwanbond-$Version-linux-arm64.tar.gz"
Create-Archive "multiwanbond-linux-arm" "multiwanbond-$Version-linux-arm.tar.gz"

# Windows packages
Create-Archive "multiwanbond-windows-amd64.exe" "multiwanbond-$Version-windows-amd64.zip"
Create-Archive "multiwanbond-windows-arm64.exe" "multiwanbond-$Version-windows-arm64.zip"

# macOS packages
Create-Archive "multiwanbond-darwin-amd64" "multiwanbond-$Version-darwin-amd64.tar.gz"
Create-Archive "multiwanbond-darwin-arm64" "multiwanbond-$Version-darwin-arm64.tar.gz"

# Generate checksums
Write-Host ""
Write-Host "Generating checksums..."
Write-Host "----------------------------"

$checksums = @()
Get-ChildItem -Path $RELEASE_DIR -Filter "multiwanbond-*" | ForEach-Object {
    $hash = (Get-FileHash -Path $_.FullName -Algorithm SHA256).Hash.ToLower()
    $checksums += "$hash  $($_.Name)"
}

$checksums | Out-File -FilePath (Join-Path $RELEASE_DIR "SHA256SUMS") -Encoding ASCII
Write-Host "✓ Created SHA256SUMS" -ForegroundColor Green

# Summary
Write-Host ""
Write-Host "========================================"
Write-Host "Build Summary"
Write-Host "========================================"
Write-Host "Version: $Version"
Write-Host "Output directory: $RELEASE_DIR\"
Write-Host ""
Write-Host "Release packages:"
Get-ChildItem -Path $RELEASE_DIR | Format-Table Name, Length
Write-Host ""
Write-Host "✓ Build completed successfully!" -ForegroundColor Green
Write-Host "========================================"

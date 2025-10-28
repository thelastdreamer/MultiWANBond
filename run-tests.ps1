# MultiWANBond Test Runner (PowerShell)
# This script sets up the correct Go environment and runs tests

# Set Go environment variables
$env:GOPATH = "c:\go-work"
$env:GOMODCACHE = "c:\go-work\pkg\mod"
$env:GOCACHE = "$env:LOCALAPPDATA\go-build"
$env:CGO_ENABLED = "0"

# Ensure cache directories exist
if (-not (Test-Path "c:\go-work\pkg\mod")) {
    New-Item -ItemType Directory -Path "c:\go-work\pkg\mod" -Force | Out-Null
}
if (-not (Test-Path "$env:LOCALAPPDATA\go-build")) {
    New-Item -ItemType Directory -Path "$env:LOCALAPPDATA\go-build" -Force | Out-Null
}

Write-Host "========================================"
Write-Host "MultiWANBond Test Suite" -ForegroundColor Cyan
Write-Host "========================================"
Write-Host ""
Write-Host "Environment:"
Write-Host "  GOPATH:      $env:GOPATH"
Write-Host "  GOMODCACHE:  $env:GOMODCACHE"
Write-Host "  GOCACHE:     $env:GOCACHE"
Write-Host ""

function Show-Menu {
    Write-Host "Select test to run:" -ForegroundColor Yellow
    Write-Host "  1. Network Detection Test"
    Write-Host "  2. Health Checker Test"
    Write-Host "  3. Final Integration Test"
    Write-Host "  4. All Tests"
    Write-Host "  5. Exit"
    Write-Host ""
}

function Run-NetworkTest {
    Write-Host ""
    Write-Host "Running Network Detection Test..." -ForegroundColor Green
    Write-Host "========================================"
    & "C:\Program Files\Go\bin\go.exe" run cmd\test\network_detect.go
    Write-Host ""
    Write-Host "Press any key to continue..."
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
}

function Run-HealthTest {
    Write-Host ""
    Write-Host "Running Health Checker Test..." -ForegroundColor Green
    Write-Host "========================================"
    & "C:\Program Files\Go\bin\go.exe" run cmd\test\health_checker.go
    Write-Host ""
    Write-Host "Press any key to continue..."
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
}

function Run-IntegrationTest {
    Write-Host ""
    Write-Host "Running Final Integration Test..." -ForegroundColor Green
    Write-Host "========================================"
    & "C:\Program Files\Go\bin\go.exe" run cmd\test\final_integration.go
    Write-Host ""
    Write-Host "Press any key to continue..."
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
}

function Run-AllTests {
    Write-Host ""
    Write-Host "Running All Tests..." -ForegroundColor Green
    Write-Host "========================================"

    Write-Host ""
    Write-Host "[1/3] Network Detection Test" -ForegroundColor Cyan
    Write-Host "----------------------------------------"
    & "C:\Program Files\Go\bin\go.exe" run cmd\test\network_detect.go

    Write-Host ""
    Write-Host "[2/3] Health Checker Test" -ForegroundColor Cyan
    Write-Host "----------------------------------------"
    & "C:\Program Files\Go\bin\go.exe" run cmd\test\health_checker.go

    Write-Host ""
    Write-Host "[3/3] Final Integration Test" -ForegroundColor Cyan
    Write-Host "----------------------------------------"
    & "C:\Program Files\Go\bin\go.exe" run cmd\test\final_integration.go

    Write-Host ""
    Write-Host "========================================" -ForegroundColor Green
    Write-Host "All tests completed!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Press any key to continue..."
    $null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
}

# Main loop
do {
    Clear-Host
    Write-Host "========================================"
    Write-Host "MultiWANBond Test Suite" -ForegroundColor Cyan
    Write-Host "========================================"
    Write-Host ""
    Show-Menu

    $choice = Read-Host "Enter choice (1-5)"

    switch ($choice) {
        "1" { Run-NetworkTest }
        "2" { Run-HealthTest }
        "3" { Run-IntegrationTest }
        "4" { Run-AllTests }
        "5" {
            Write-Host ""
            Write-Host "Exiting..." -ForegroundColor Yellow
            exit
        }
        default {
            Write-Host ""
            Write-Host "Invalid choice. Please try again." -ForegroundColor Red
            Start-Sleep -Seconds 2
        }
    }
} while ($true)

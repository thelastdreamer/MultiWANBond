@echo off
REM Fix Go Environment Variables
REM This script permanently sets the Go environment variables for your user account

echo ========================================
echo Go Environment Fix
echo ========================================
echo.
echo This script will set the following environment variables:
echo   GOPATH=c:\go-work
echo   GO111MODULE=on
echo.
echo These will be set permanently for your user account.
echo.

set /p confirm="Continue? (Y/N): "
if /i not "%confirm%"=="Y" (
    echo Cancelled.
    pause
    exit /b
)

echo.
echo Setting environment variables...

REM Set GOPATH for current user
setx GOPATH "c:\go-work"
if %errorlevel% equ 0 (
    echo [OK] GOPATH set to c:\go-work
) else (
    echo [ERROR] Failed to set GOPATH
)

REM Enable Go modules
setx GO111MODULE "on"
if %errorlevel% equ 0 (
    echo [OK] GO111MODULE set to on
) else (
    echo [ERROR] Failed to set GO111MODULE
)

REM Create required directories
echo.
echo Creating Go directories...
if not exist "c:\go-work" mkdir "c:\go-work"
if not exist "c:\go-work\pkg" mkdir "c:\go-work\pkg"
if not exist "c:\go-work\pkg\mod" mkdir "c:\go-work\pkg\mod"
if not exist "c:\go-work\bin" mkdir "c:\go-work\bin"
echo [OK] Directories created

echo.
echo ========================================
echo Environment variables updated!
echo ========================================
echo.
echo IMPORTANT: You need to:
echo   1. Close this command prompt
echo   2. Open a NEW command prompt
echo   3. Run your Go commands in the new window
echo.
echo The new environment will be active in new terminals.
echo.
pause

@echo off
setlocal

cd /d "%~dp0"

echo Building MultiWANBond...
echo.

set CGO_ENABLED=0
"C:\Program Files\Go\bin\go.exe" build -v -o multiwanbond.exe cmd/server/main.go

if %ERRORLEVEL% EQU 0 (
    echo.
    echo [OK] Build successful!
    echo Binary: %CD%\multiwanbond.exe
) else (
    echo.
    echo [ERROR] Build failed!
    exit /b 1
)

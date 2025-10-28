@echo off
REM MultiWANBond Test Runner
REM This script sets up the correct Go environment and runs tests

REM Set Go environment variables
set GOPATH=c:\go-work
set GOMODCACHE=c:\go-work\pkg\mod
set GOCACHE=%LOCALAPPDATA%\go-build
set CGO_ENABLED=0

REM Ensure cache directories exist
if not exist "c:\go-work\pkg\mod" mkdir "c:\go-work\pkg\mod"
if not exist "%LOCALAPPDATA%\go-build" mkdir "%LOCALAPPDATA%\go-build"

echo ========================================
echo MultiWANBond Test Suite
echo ========================================
echo.
echo Environment:
echo   GOPATH:      %GOPATH%
echo   GOMODCACHE:  %GOMODCACHE%
echo   GOCACHE:     %GOCACHE%
echo.

:menu
echo Select test to run:
echo   1. Network Detection Test
echo   2. Health Checker Test
echo   3. Final Integration Test
echo   4. All Tests
echo   5. Exit
echo.
set /p choice="Enter choice (1-5): "

if "%choice%"=="1" goto network_test
if "%choice%"=="2" goto health_test
if "%choice%"=="3" goto integration_test
if "%choice%"=="4" goto all_tests
if "%choice%"=="5" goto end

echo Invalid choice. Please try again.
echo.
goto menu

:network_test
echo.
echo Running Network Detection Test...
echo ========================================
"C:\Program Files\Go\bin\go.exe" run cmd\test\network_detect.go
echo.
pause
goto menu

:health_test
echo.
echo Running Health Checker Test...
echo ========================================
"C:\Program Files\Go\bin\go.exe" run cmd\test\health_checker.go
echo.
pause
goto menu

:integration_test
echo.
echo Running Final Integration Test...
echo ========================================
"C:\Program Files\Go\bin\go.exe" run cmd\test\final_integration.go
echo.
pause
goto menu

:all_tests
echo.
echo Running All Tests...
echo ========================================
echo.
echo [1/3] Network Detection Test
echo ----------------------------------------
"C:\Program Files\Go\bin\go.exe" run cmd\test\network_detect.go
echo.
echo [2/3] Health Checker Test
echo ----------------------------------------
"C:\Program Files\Go\bin\go.exe" run cmd\test\health_checker.go
echo.
echo [3/3] Final Integration Test
echo ----------------------------------------
"C:\Program Files\Go\bin\go.exe" run cmd\test\final_integration.go
echo.
echo ========================================
echo All tests completed!
echo ========================================
pause
goto menu

:end
echo.
echo Exiting...

# Go Environment Fix Guide

## Problem

When running Go commands, you get this error:
```
go: could not create module cache: mkdir C:\Program Files\Go\bin\go.exe: The system cannot find the path specified.
```

## Root Cause

Your Windows environment doesn't have the Go environment variables (`GOPATH`, `GOMODCACHE`) configured correctly. When you run `go run` from the Windows command prompt, Go doesn't know where to create its module cache.

## Quick Fix (Recommended)

Use the provided test runner scripts that automatically set the correct environment:

### Option 1: Batch File (Windows CMD)
```cmd
cd "C:\Users\Panagiotis\OneDrive - numoierapetra.com\Έγγραφα\GitHub\MultiWANBond"
run-tests.bat
```

### Option 2: PowerShell Script
```powershell
cd "C:\Users\Panagiotis\OneDrive - numoierapetra.com\Έγγραφα\GitHub\MultiWANBond"
.\run-tests.ps1
```

The test runner will:
- Set the correct Go environment variables
- Create necessary cache directories
- Provide a menu to run different tests
- Handle all environment setup automatically

## Permanent Fix

If you want to fix the Go environment permanently:

### Method 1: Run the Fix Script
```cmd
cd "C:\Users\Panagiotis\OneDrive - numoierapetra.com\Έγγραφα\GitHub\MultiWANBond"
fix-go-env.bat
```

This will:
1. Set `GOPATH=c:\go-work` for your user account
2. Enable Go modules (`GO111MODULE=on`)
3. Create required directories
4. **Important**: You must close and reopen your terminal after running this

### Method 2: Manual Fix (Windows)

1. **Open Environment Variables**:
   - Press `Win + R`, type `sysdm.cpl`, press Enter
   - Click "Advanced" tab → "Environment Variables"

2. **Add User Variables**:
   - Click "New" under "User variables"
   - Variable name: `GOPATH`
   - Variable value: `c:\go-work`
   - Click OK

3. **Create Directories**:
   - Open Command Prompt as Administrator
   - Run: `mkdir c:\go-work\pkg\mod`

4. **Restart Terminal**:
   - Close all command prompts
   - Open a new command prompt
   - Test with: `go env GOPATH`

## Verify Fix

After applying the fix, verify it works:

```cmd
go env GOPATH
```

Should output: `c:\go-work`

Then try running a test:
```cmd
go run cmd\test\health_checker.go
```

## Alternative: Use the Test Runner Scripts

The easiest way to avoid environment issues is to always use the provided test runner scripts:

- **`run-tests.bat`** - For Windows Command Prompt
- **`run-tests.ps1`** - For PowerShell

These scripts automatically set the correct environment every time you run them, so you don't need to worry about permanent environment configuration.

## Files Created to Help You

1. **`run-tests.bat`** - Test runner for Windows CMD with interactive menu
2. **`run-tests.ps1`** - Test runner for PowerShell with interactive menu
3. **`fix-go-env.bat`** - Permanently fixes Go environment variables
4. **This file (`GO_ENVIRONMENT_FIX.md`)** - This guide

## Summary

**Quick Solution**: Just use `run-tests.bat` or `run-tests.ps1` - they handle everything automatically!

**Permanent Solution**: Run `fix-go-env.bat`, then close and reopen your terminal.

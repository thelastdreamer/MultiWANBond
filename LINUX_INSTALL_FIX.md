# Linux Installation Fix

## Problem

When running `install.sh` on Linux, you encountered:
```
pkg/network/detector_linux.go:15:2: missing go.sum entry for module providing package github.com/vishvananda/netlink
```

## Root Cause

The `go.sum` file was missing entries for transitive dependencies, specifically:
- `github.com/vishvananda/netlink` (required for Linux network detection)
- `github.com/vishvananda/netns` (transitive dependency of netlink)

## Fix Applied

1. **Added go.sum** - Contains all dependency checksums
2. **Updated install.sh** - Changed from `go mod download` to `go mod tidy`
3. **Updated install.ps1** - Same change for Windows consistency

## How to Apply Fix on Your Linux Server

### Option 1: Pull Latest Changes (Recommended)

```bash
cd ~/MultiWANBond
git pull origin main
bash install.sh
```

### Option 2: Manual Fix (If git pull doesn't work)

```bash
cd ~/MultiWANBond

# Update dependencies
go mod tidy

# Try building again
export CGO_ENABLED=0
go build -ldflags "-s -w" -o multiwanbond ./cmd/server/main.go

# If successful, run setup
./multiwanbond setup
```

### Option 3: Fresh Installation

```bash
# Remove old installation
rm -rf ~/.local/share/multiwanbond

# Clone latest version
cd ~
rm -rf MultiWANBond
git clone https://github.com/thelastdreamer/MultiWANBond.git
cd MultiWANBond

# Run updated installer
bash install.sh
```

## What Changed

### install.sh (Line 244)
**Before:**
```bash
if go mod download; then
    echo -e "${GREEN}  ✓ Dependencies downloaded${NC}"
```

**After:**
```bash
if go mod tidy; then
    echo -e "${GREEN}  ✓ Dependencies downloaded and verified${NC}"
```

### New go.sum File
The file now includes:
```
github.com/vishvananda/netlink v1.1.0 h1:...
github.com/vishvananda/netlink v1.1.0/go.mod h1:...
github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df h1:...
github.com/vishvananda/netns v0.0.0-20191106174202-0a2b9b5464df/go.mod h1:...
```

## Expected Output (After Fix)

```bash
$ bash install.sh

================================================================
       MultiWANBond Installer for linux
================================================================

[1/7] Checking Go installation...
  ✓ Go 1.25 is installed
[2/7] Checking Git installation...
  ✓ Git is installed
[3/7] Setting up Go environment...
  ✓ Go environment configured
[4/7] Installing MultiWANBond...
  ✓ MultiWANBond downloaded to /home/minoanson/.local/share/multiwanbond
[5/7] Downloading dependencies...
  ✓ Dependencies downloaded and verified
[6/7] Building MultiWANBond...
  ✓ MultiWANBond built successfully
[7/7] Running setup wizard...

================================================================
       Installation Complete!
================================================================
```

## Verification

After the fix, verify the build:
```bash
cd ~/MultiWANBond
go build -o multiwanbond cmd/server/main.go
./multiwanbond version
```

Should output:
```
MultiWANBond v1.0.0
```

## Why This Happened

The initial development was done on Windows, where some Linux-specific dependencies like `vishvananda/netlink` weren't tested. The `go.sum` file tracks checksums for all dependencies to ensure reproducible builds. Without it, Go refuses to build on a different machine.

The switch from `go mod download` to `go mod tidy`:
- **`go mod download`** - Only downloads direct dependencies
- **`go mod tidy`** - Downloads all dependencies (including transitive) AND updates go.sum

## Prevention

Going forward, always commit `go.sum` when adding new dependencies:
```bash
git add go.sum
git commit -m "Update dependencies"
```

The updated installers now use `go mod tidy` which prevents this issue automatically.

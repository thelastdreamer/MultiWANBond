# MultiWANBond System Update Guide

## Overview

This guide walks you through updating your MultiWANBond client and server systems with the new features:
- ✅ Auto-generated Web UI passwords
- ✅ Full routing policy management
- ✅ Enhanced security and authentication

---

## Pre-Update Checklist

Before updating, complete these steps:

### 1. Backup Current Configuration

```powershell
# Backup config file
Copy-Item "C:\ProgramData\MultiWANBond\config.json" "C:\ProgramData\MultiWANBond\config.json.backup"

# Or if using custom location:
Copy-Item "path\to\config.json" "path\to\config.json.backup"
```

### 2. Note Current Credentials

If you manually set Web UI credentials before, note them down. The update is backward compatible, so existing credentials will continue to work.

### 3. Stop Running Instances

```powershell
# If running in terminal: Press Ctrl+C

# If running as service:
Stop-Service MultiWANBond

# Or with NSSM:
nssm stop MultiWANBond
```

---

## Update Process

### Option A: Fresh Installation (Recommended for Testing)

This approach lets you test the new setup wizard with password generation.

#### Step 1: Build New Executable

```powershell
# In your MultiWANBond directory
cd "C:\Users\Panagiotis\OneDrive - numoierapetra.com\Έγγραφα\GitHub\MultiWANBond"

# Build
& "C:\Program Files\Go\bin\go.exe" build -v -o multiwanbond-new.exe cmd/server/main.go
```

#### Step 2: Run Setup Wizard

```powershell
# Run setup to generate new config with password
.\multiwanbond-new.exe setup

# Follow the wizard:
# 1. Select mode (Client/Server/Standalone)
# 2. Select network interfaces
# 3. Configure WANs
# 4. Configure server settings
# 5. Security settings
# 6. **NEW** Web UI Security - Password will be auto-generated!
# 7. Review and confirm
```

**Important**: At Step 6, you'll see something like:

```
═══════════════════════════════════════════════════════════
  ⚠️  IMPORTANT: Web UI Credentials - SAVE THESE SECURELY!
═══════════════════════════════════════════════════════════

  Web UI URL:  http://localhost:8080
  Username:    admin
  Password:    3kF9mN2pX7qR5wL1

  ⚠️  Write this password down NOW!
  ⚠️  You'll need it to access the Web UI dashboard.
  ⚠️  This password will be saved to your config file.

═══════════════════════════════════════════════════════════
```

**SAVE THIS PASSWORD!**

#### Step 3: Replace Old Executable

```powershell
# Stop old instance if running
Stop-Process -Name "multiwanbond" -Force -ErrorAction SilentlyContinue

# Replace executable
Move-Item "multiwanbond.exe" "multiwanbond-old.exe" -Force
Move-Item "multiwanbond-new.exe" "multiwanbond.exe" -Force
```

#### Step 4: Start Updated System

```powershell
# Start server
.\multiwanbond.exe --config "path\to\new\config.json"

# You should see:
# "Web UI authentication enabled"
# "Web UI available at: http://localhost:8080 (Username: admin)"
```

---

### Option B: In-Place Update (Keep Existing Config)

This approach preserves your existing configuration and adds the new features.

#### Step 1: Build New Executable

```powershell
cd "C:\Users\Panagiotis\OneDrive - numoierapetra.com\Έγγραφα\GitHub\MultiWANBond"

& "C:\Program Files\Go\bin\go.exe" build -v -o multiwanbond-new.exe cmd/server/main.go
```

#### Step 2: Add WebUI Section to Existing Config

Open your config file and add the `webui` section:

```json
{
  "session": { ... },
  "wans": [ ... ],
  "routing": { ... },
  "fec": { ... },
  "monitoring": { ... },

  "webui": {
    "username": "admin",
    "password": "YourSecurePassword123",
    "enabled": true
  }
}
```

**Choose a strong password or generate one:**

```powershell
# Generate random password (PowerShell)
-join ((48..57) + (65..90) + (97..122) | Get-Random -Count 16 | ForEach-Object {[char]$_})
```

#### Step 3: Optional - Add Routing Policies Section

If you want to use routing policies, add to the `routing` section:

```json
{
  "routing": {
    "mode": "adaptive",
    "bandwidth_reset_interval": "1m",
    "policies": []
  }
}
```

#### Step 4: Replace Executable and Restart

```powershell
# Stop old instance
Stop-Process -Name "multiwanbond" -Force -ErrorAction SilentlyContinue

# Replace
Move-Item "multiwanbond.exe" "multiwanbond-old.exe" -Force
Move-Item "multiwanbond-new.exe" "multiwanbond.exe" -Force

# Restart
.\multiwanbond.exe --config "C:\ProgramData\MultiWANBond\config.json"
```

---

## Updating Client System

Your client system connects to the server. Follow these steps:

### Step 1: Update Local Client

```powershell
# On client machine
cd "path\to\MultiWANBond"

# Pull latest code
git pull origin main

# Build
& "C:\Program Files\Go\bin\go.exe" build -v -o multiwanbond.exe cmd/server/main.go
```

### Step 2: Check Config File

Your client config at `C:\ProgramData\MultiWANBond\config.json` should look like:

```json
{
  "session": {
    "local_endpoint": "0.0.0.0:0",
    "remote_endpoint": "45.32.232.36:9000",
    ...
  },
  "wans": [
    {
      "id": 1,
      "name": "WAN1-Ethernet",
      "local_addr": "192.168.82.20",
      "remote_addr": "45.32.232.36:9000",
      ...
    },
    {
      "id": 2,
      "name": "WAN2-Wi-Fi",
      "local_addr": "192.168.200.150",
      "remote_addr": "45.32.232.36:9000",
      ...
    }
  ],
  "webui": {
    "username": "admin",
    "password": "ClientPassword123",
    "enabled": true
  }
}
```

### Step 3: Restart Client

```powershell
# Stop current instance (Ctrl+C if in terminal)

# Or kill process
Stop-Process -Name "multiwanbond" -Force

# Restart
.\multiwanbond.exe --config "C:\ProgramData\MultiWANBond\config.json"
```

### Step 4: Access Client Web UI

```
Open browser: http://localhost:8080
Username: admin
Password: [your client password]
```

---

## Updating Server System

Your server at `45.32.232.36:9000` needs to be updated too.

### Step 1: Connect to Server

```powershell
# SSH or RDP to your server
ssh user@45.32.232.36
```

### Step 2: Update Server Code

```bash
# On server
cd /path/to/MultiWANBond

# Pull latest changes
git pull origin main

# Build for Linux (if server is Linux)
go build -v -o multiwanbond cmd/server/main.go

# Or for Windows Server
go build -v -o multiwanbond.exe cmd/server/main.go
```

### Step 3: Update Server Config

Add webui section to server config:

```json
{
  "session": {
    "local_endpoint": "0.0.0.0:9000",
    "remote_endpoint": "",
    ...
  },
  "wans": [ ... ],
  "webui": {
    "username": "admin",
    "password": "ServerPassword456",
    "enabled": true
  }
}
```

### Step 4: Restart Server

```bash
# Stop server (Ctrl+C or kill process)
pkill multiwanbond

# Or if using systemd:
sudo systemctl stop multiwanbond

# Restart
./multiwanbond --config /path/to/config.json

# Or with systemd:
sudo systemctl start multiwanbond
```

### Step 5: Access Server Web UI

```
# If server allows port 8080, or set up SSH tunnel:
ssh -L 8080:localhost:8080 user@45.32.232.36

# Then open: http://localhost:8080
Username: admin
Password: [your server password]
```

---

## Verification Steps

After updating both client and server:

### 1. Check Client Connection

```powershell
# Client should show:
# "Mode: Client - Connected to server at 45.32.232.36:9000"
```

### 2. Test Web UI Authentication

```powershell
# Try accessing without credentials - should prompt for login
# Enter username/password - should grant access
```

### 3. Test Routing Policies (Optional)

```powershell
# Via Web UI:
# 1. Go to Configuration → Routing Policies tab
# 2. Add a test policy
# 3. Check config file - should contain the policy

# Via API:
curl -u admin:YourPassword http://localhost:8080/api/routing
```

### 4. Verify Both Systems Running

```powershell
# Client
Get-Process multiwanbond
# Should show running process

# Server (via SSH)
ps aux | grep multiwanbond
# Should show running process
```

---

## Troubleshooting

### Issue: "Access is denied" when saving config

**Solution**: Follow [WINDOWS_PERMISSIONS.md](WINDOWS_PERMISSIONS.md)

Quick fix for testing:
```powershell
# Use user directory
$configDir = "$env:USERPROFILE\MultiWANBond"
New-Item -ItemType Directory -Path $configDir -Force
Copy-Item "C:\ProgramData\MultiWANBond\config.json" "$configDir\config.json"
.\multiwanbond.exe --config "$configDir\config.json"
```

---

### Issue: Web UI shows "No authentication" warning

**Cause**: Config missing `webui` section

**Solution**: Add webui section to config file (see Option B above)

---

### Issue: Can't access Web UI after update

**Check**:
1. Is server running? `Get-Process multiwanbond`
2. Is port 8080 open? `Test-NetConnection localhost -Port 8080`
3. Check logs for errors

**Solution**:
```powershell
# Restart with verbose logging
.\multiwanbond.exe --config config.json
# Check output for errors
```

---

### Issue: Client can't connect to server

**Check**:
1. Is server running?
2. Can you ping server? `ping 45.32.232.36`
3. Is port 9000 open? `Test-NetConnection 45.32.232.36 -Port 9000`

**Solution**:
```powershell
# On server, check firewall:
# Linux:
sudo ufw allow 9000/udp

# Windows Server:
New-NetFirewallRule -DisplayName "MultiWANBond" -Direction Inbound -LocalPort 9000 -Protocol UDP -Action Allow
```

---

### Issue: Lost Web UI password

**Solution**:
1. Stop multiwanbond
2. Edit config file: `C:\ProgramData\MultiWANBond\config.json`
3. Change `webui.password` to new password
4. Restart multiwanbond

Or regenerate config with setup wizard:
```powershell
.\multiwanbond.exe setup
# New password will be generated
```

---

## Rolling Back

If you need to rollback:

```powershell
# Stop new version
Stop-Process -Name "multiwanbond" -Force

# Restore old executable
Move-Item "multiwanbond-old.exe" "multiwanbond.exe" -Force

# Restore old config
Copy-Item "C:\ProgramData\MultiWANBond\config.json.backup" "C:\ProgramData\MultiWANBond\config.json" -Force

# Restart
.\multiwanbond.exe --config "C:\ProgramData\MultiWANBond\config.json"
```

---

## Production Deployment

For production systems, consider:

### 1. Install as Windows Service

See [WINDOWS_PERMISSIONS.md](WINDOWS_PERMISSIONS.md) for complete guide using NSSM.

Quick setup:
```powershell
# Download NSSM from https://nssm.cc/download

# Install service
.\nssm.exe install MultiWANBond "C:\Program Files\MultiWANBond\multiwanbond.exe"
.\nssm.exe set MultiWANBond AppParameters --config "C:\ProgramData\MultiWANBond\config.json"
.\nssm.exe set MultiWANBond Start SERVICE_AUTO_START

# Start
.\nssm.exe start MultiWANBond
```

### 2. Enable HTTPS for Web UI

See [SETUP_AUTHENTICATION.md](SETUP_AUTHENTICATION.md) for HTTPS setup.

### 3. Configure Firewall

```powershell
# Allow Web UI (localhost only)
New-NetFirewallRule -DisplayName "MultiWANBond WebUI" `
    -Direction Inbound -LocalPort 8080 -Protocol TCP `
    -Action Allow -RemoteAddress 127.0.0.1

# Allow tunnel traffic
New-NetFirewallRule -DisplayName "MultiWANBond Tunnel" `
    -Direction Inbound -LocalPort 9000 -Protocol UDP `
    -Action Allow
```

---

## Quick Reference Commands

### Build
```powershell
& "C:\Program Files\Go\bin\go.exe" build -v -o multiwanbond.exe cmd/server/main.go
```

### Start Client
```powershell
.\multiwanbond.exe --config "C:\ProgramData\MultiWANBond\config.json"
```

### Check Status
```powershell
Get-Process multiwanbond
Test-NetConnection localhost -Port 8080
```

### View Logs
```powershell
# Logs are printed to console by default
# To save to file:
.\multiwanbond.exe --config config.json > multiwanbond.log 2>&1
```

### Generate Password
```powershell
# PowerShell
-join ((48..57) + (65..90) + (97..122) | Get-Random -Count 16 | ForEach-Object {[char]$_})
```

---

## Summary

**Update Steps:**
1. ✅ Backup configuration
2. ✅ Build new executable
3. ✅ Add `webui` section to config (or run setup wizard)
4. ✅ Replace executable
5. ✅ Restart system
6. ✅ Test Web UI with credentials
7. ✅ Verify client-server connection

**New Features Available:**
- 🔐 Web UI authentication enabled by default
- 📋 Routing policy management via Web UI
- 🛠️ Complete Windows permissions documentation

**Important Files:**
- Config: `C:\ProgramData\MultiWANBond\config.json`
- Executable: `multiwanbond.exe`
- Documentation: `WINDOWS_PERMISSIONS.md`, `IMPLEMENTATION_SUMMARY.md`

For detailed information on new features, see [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md).

For Windows permissions issues, see [WINDOWS_PERMISSIONS.md](WINDOWS_PERMISSIONS.md).

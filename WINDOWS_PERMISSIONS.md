# Windows Permissions & Configuration File Access

## The Issue

When you try to modify the configuration file at `C:\ProgramData\MultiWANBond\config.json` from the Web UI, you may encounter an "Access is denied" error. This is **not a bug** - it's Windows security working as intended.

## Why This Happens

### Windows Security Model

Windows has a security model that protects system directories from unauthorized modification:

1. **ProgramData Directory** (`C:\ProgramData\`):
   - System-wide configuration storage
   - Requires administrative privileges to write
   - Protected by User Account Control (UAC)

2. **User Permissions**:
   - Normal users: Read-only access
   - Administrators: Full access (when elevated)
   - SYSTEM account: Full access

3. **User Account Control (UAC)**:
   - Even administrators run with limited privileges by default
   - Must explicitly elevate to gain full administrative rights

## Solutions

You have three options to solve this problem:

---

## Solution 1: Run MultiWANBond as Administrator (Recommended for Testing)

### Option A: Right-Click Method

1. Find `multiwanbond.exe`
2. Right-click → **"Run as administrator"**
3. Click **"Yes"** in the UAC prompt
4. The Web UI will now have write access to `C:\ProgramData\MultiWANBond\config.json`

### Option B: Always Run as Administrator

1. Right-click `multiwanbond.exe`
2. Select **"Properties"**
3. Go to **"Compatibility"** tab
4. Check **"Run this program as an administrator"**
5. Click **"Apply"** → **"OK"**

Now double-clicking the executable will always prompt for elevation.

### Option C: Task Scheduler (Run on Startup)

1. Open **Task Scheduler** (search in Start Menu)
2. Click **"Create Task"** (not Basic Task)
3. **General** tab:
   - Name: `MultiWANBond Service`
   - Check **"Run with highest privileges"**
   - Check **"Run whether user is logged on or not"**
4. **Triggers** tab:
   - New → **At startup**
5. **Actions** tab:
   - New → **Start a program**
   - Program: `C:\Path\To\multiwanbond.exe`
   - Add arguments: `--config "C:\ProgramData\MultiWANBond\config.json"`
6. Click **"OK"**

---

## Solution 2: Use User-Writable Configuration Location (Recommended for Development)

Instead of `C:\ProgramData\MultiWANBond\`, use a location in your user directory:

### Step 1: Create User Configuration Directory

```powershell
# PowerShell
$configDir = "$env:USERPROFILE\MultiWANBond"
New-Item -ItemType Directory -Path $configDir -Force
```

Or manually create: `C:\Users\YourUsername\MultiWANBond\`

### Step 2: Copy Existing Configuration

```powershell
# PowerShell
Copy-Item "C:\ProgramData\MultiWANBond\config.json" "$env:USERPROFILE\MultiWANBond\config.json"
```

### Step 3: Start MultiWANBond with User Config

```powershell
# PowerShell
.\multiwanbond.exe --config "$env:USERPROFILE\MultiWANBond\config.json"
```

### Step 4: Access Web UI

Open http://localhost:8080 - configuration changes will now work without admin rights!

---

## Solution 3: Grant Permissions to ProgramData (Production)

Make `C:\ProgramData\MultiWANBond\` writable by the user running MultiWANBond.

### Using PowerShell (Recommended)

```powershell
# Run PowerShell as Administrator

# Grant full control to the current user
$path = "C:\ProgramData\MultiWANBond"
$acl = Get-Acl $path
$username = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
$accessRule = New-Object System.Security.AccessControl.FileSystemAccessRule($username, "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow")
$acl.SetAccessRule($accessRule)
Set-Acl $path $acl

Write-Host "Permissions granted to $username on $path"
```

### Using GUI (Windows Explorer)

1. Open **File Explorer**
2. Navigate to `C:\ProgramData\` (you may need to show hidden files)
3. Right-click `MultiWANBond` folder → **Properties**
4. Go to **Security** tab
5. Click **Edit**
6. Click **Add**
7. Enter your Windows username → **Check Names** → **OK**
8. Select your username in the list
9. Check **Full control** in the Allow column
10. Click **Apply** → **OK**

### Verify Permissions

```powershell
# PowerShell
icacls "C:\ProgramData\MultiWANBond"
```

You should see your username with `(F)` (Full control).

---

## Solution 4: Windows Service Installation (Production)

Run MultiWANBond as a Windows Service running under SYSTEM account.

### Using NSSM (Non-Sucking Service Manager)

1. **Download NSSM**:
   - https://nssm.cc/download
   - Extract `nssm.exe` (use 64-bit version for 64-bit Windows)

2. **Install Service** (PowerShell as Administrator):

```powershell
# Navigate to NSSM directory
cd C:\Path\To\nssm\win64

# Install service
.\nssm.exe install MultiWANBond "C:\Path\To\multiwanbond.exe"

# Set arguments
.\nssm.exe set MultiWANBond AppParameters --config "C:\ProgramData\MultiWANBond\config.json"

# Set working directory
.\nssm.exe set MultiWANBond AppDirectory "C:\Path\To\MultiWANBond"

# Set startup type
.\nssm.exe set MultiWANBond Start SERVICE_AUTO_START

# Start service
.\nssm.exe start MultiWANBond
```

3. **Verify Service**:

```powershell
Get-Service MultiWANBond
```

4. **Manage Service**:

```powershell
# Stop
.\nssm.exe stop MultiWANBond

# Start
.\nssm.exe start MultiWANBond

# Restart
.\nssm.exe restart MultiWANBond

# Uninstall
.\nssm.exe remove MultiWANBond confirm
```

---

## Comparison of Solutions

| Solution | Pros | Cons | Best For |
|----------|------|------|----------|
| **Run as Admin** | Simple, no setup | UAC prompts, manual start | Testing |
| **User Directory** | No admin needed, simple | Config not system-wide | Development |
| **Grant Permissions** | Works without elevation | Security risk if misconfigured | Personal machines |
| **Windows Service** | Automatic start, proper security | More complex setup | Production |

---

## Security Best Practices

### DO:
- Use Windows Service for production deployments
- Keep configuration file permissions restrictive
- Enable Web UI authentication (automatic after setup wizard)
- Use HTTPS for Web UI in production
- Regularly backup configuration files

### DON'T:
- Grant "Everyone" full control to ProgramData
- Disable UAC to avoid prompts
- Store passwords in plain text outside config
- Run as SYSTEM unnecessarily
- Expose Web UI to the internet without authentication

---

## Troubleshooting

### Issue: "Access is denied" when saving config

**Check:**
```powershell
# Test write access
Test-Path "C:\ProgramData\MultiWANBond\config.json" -PathType Leaf
icacls "C:\ProgramData\MultiWANBond\config.json"
```

**Solution:** Use one of the solutions above.

---

### Issue: UAC prompts every time

**Options:**
1. Use Solution 2 (User Directory)
2. Use Solution 3 (Grant Permissions)
3. Install as Windows Service

---

### Issue: Service won't start

**Check Event Viewer:**
1. Windows Key + R → `eventvwr.msc`
2. Windows Logs → Application
3. Look for MultiWANBond errors

**Common Causes:**
- Config file path incorrect
- Config file malformed JSON
- Port 8080 already in use
- Missing dependencies

---

### Issue: Configuration changes not persisting

**Verify:**
```powershell
# Check if file is read-only
Get-ItemProperty "C:\ProgramData\MultiWANBond\config.json" | Select-Object IsReadOnly

# Remove read-only attribute if needed
Set-ItemProperty "C:\ProgramData\MultiWANBond\config.json" -Name IsReadOnly -Value $false
```

---

## Development vs Production

### Development Setup
- Use **Solution 2** (User Directory)
- Keep config in source control (without sensitive data)
- Easy to modify and test

### Production Setup
- Use **Solution 4** (Windows Service)
- Config in `C:\ProgramData\MultiWANBond\`
- Automatic startup
- Proper logging and monitoring

---

## Example: Complete Production Setup

### Step 1: Create Directory Structure

```powershell
# Run as Administrator
New-Item -ItemType Directory -Path "C:\ProgramData\MultiWANBond" -Force
New-Item -ItemType Directory -Path "C:\ProgramData\MultiWANBond\logs" -Force
New-Item -ItemType Directory -Path "C:\ProgramData\MultiWANBond\webui" -Force
```

### Step 2: Copy Files

```powershell
# Copy executable
Copy-Item ".\multiwanbond.exe" "C:\Program Files\MultiWANBond\" -Force

# Copy Web UI
Copy-Item ".\webui\*" "C:\ProgramData\MultiWANBond\webui\" -Recurse -Force

# Copy config
Copy-Item ".\config.json" "C:\ProgramData\MultiWANBond\" -Force
```

### Step 3: Install as Service

```powershell
.\nssm.exe install MultiWANBond "C:\Program Files\MultiWANBond\multiwanbond.exe"
.\nssm.exe set MultiWANBond AppParameters --config "C:\ProgramData\MultiWANBond\config.json"
.\nssm.exe set MultiWANBond AppDirectory "C:\Program Files\MultiWANBond"
.\nssm.exe set MultiWANBond DisplayName "MultiWANBond Service"
.\nssm.exe set MultiWANBond Description "Multi-WAN Bonding and Load Balancing"
.\nssm.exe set MultiWANBond Start SERVICE_AUTO_START
```

### Step 4: Configure Firewall

```powershell
# Allow Web UI (localhost only)
New-NetFirewallRule -DisplayName "MultiWANBond WebUI" `
    -Direction Inbound `
    -LocalPort 8080 `
    -Protocol TCP `
    -Action Allow `
    -RemoteAddress 127.0.0.1

# Allow bonding traffic (adjust ports as needed)
New-NetFirewallRule -DisplayName "MultiWANBond Tunnel" `
    -Direction Inbound `
    -LocalPort 9000 `
    -Protocol UDP `
    -Action Allow
```

### Step 5: Start Service

```powershell
Start-Service MultiWANBond
Get-Service MultiWANBond
```

---

## Quick Reference

### Check Permissions
```powershell
icacls "C:\ProgramData\MultiWANBond"
```

### Grant User Permissions
```powershell
icacls "C:\ProgramData\MultiWANBond" /grant "$env:USERNAME:(OI)(CI)F" /T
```

### Test Write Access
```powershell
Test-Path "C:\ProgramData\MultiWANBond" -IsValid
[System.IO.File]::WriteAllText("C:\ProgramData\MultiWANBond\test.txt", "test")
```

### View Service Status
```powershell
Get-Service MultiWANBond | Format-List *
```

### View Service Logs (if using NSSM)
```powershell
Get-Content "C:\ProgramData\MultiWANBond\logs\service.log" -Tail 50
```

---

## Additional Resources

- **Windows Services**: https://docs.microsoft.com/windows/win32/services/services
- **NSSM Documentation**: https://nssm.cc/usage
- **ICACLS Reference**: https://docs.microsoft.com/windows-server/administration/windows-commands/icacls
- **UAC Guide**: https://docs.microsoft.com/windows/security/identity-protection/user-account-control/

---

## Summary

The "Access is denied" error is **normal Windows behavior** protecting system directories. Choose the solution that best fits your use case:

- **Testing/Development**: Use User Directory (Solution 2)
- **Personal Use**: Run as Administrator or Grant Permissions
- **Production**: Install as Windows Service (Solution 4)

All solutions are valid - pick what works best for your scenario!

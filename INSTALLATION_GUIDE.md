# MultiWANBond Installation Guide

Complete installation instructions for all platforms with one-click installers.

## Table of Contents

1. [Windows Installation](#windows-installation)
2. [Linux Installation](#linux-installation)
3. [macOS Installation](#macos-installation)
4. [Android Installation](#android-installation)
5. [iOS Installation](#ios-installation)
6. [First-Time Setup](#first-time-setup)
7. [Managing WAN Interfaces](#managing-wan-interfaces)

---

## Windows Installation

### One-Click Install

1. **Download the installer**:
   ```powershell
   # Download install script
   Invoke-WebRequest -Uri "https://raw.githubusercontent.com/thelastdreamer/MultiWANBond/main/install.ps1" -OutFile "install.ps1"
   ```

2. **Run the installer** (Right-click PowerShell → Run as Administrator):
   ```powershell
   .\install.ps1
   ```

3. **The installer will automatically**:
   - ✓ Check if Go is installed (install if needed)
   - ✓ Check if Git is installed (install if needed)
   - ✓ Download MultiWANBond
   - ✓ Download all dependencies
   - ✓ Build the application
   - ✓ Run the setup wizard

### Manual Installation

```powershell
# 1. Install Go 1.21+ from https://go.dev/dl/

# 2. Clone repository
git clone https://github.com/thelastdreamer/MultiWANBond.git
cd MultiWANBond

# 3. Download dependencies
go mod download

# 4. Build
go build -o multiwanbond.exe ./cmd/server

# 5. Run setup
.\multiwanbond.exe setup
```

---

## Linux Installation

### One-Click Install

```bash
# Download and run installer
curl -fsSL https://raw.githubusercontent.com/thelastdreamer/MultiWANBond/main/install.sh | bash
```

Or download and run manually:

```bash
# Download
wget https://raw.githubusercontent.com/thelastdreamer/MultiWANBond/main/install.sh

# Make executable
chmod +x install.sh

# Run
./install.sh
```

### Distribution-Specific

#### Ubuntu/Debian

```bash
# Install Go
sudo apt update
sudo apt install golang-go git

# Clone and setup
git clone https://github.com/thelastdreamer/MultiWANBond.git
cd MultiWANBond
go mod download
go build -o multiwanbond ./cmd/server

# Run setup
./multiwanbond setup
```

#### CentOS/RHEL/Fedora

```bash
# Install Go
sudo dnf install golang git

# Clone and setup
git clone https://github.com/thelastdreamer/MultiWANBond.git
cd MultiWANBond
go mod download
go build -o multiwanbond ./cmd/server

# Run setup
./multiwanbond setup
```

#### Arch Linux

```bash
# Install Go
sudo pacman -S go git

# Clone and setup
git clone https://github.com/thelastdreamer/MultiWANBond.git
cd MultiWANBond
go mod download
go build -o multiwanbond ./cmd/server

# Run setup
./multiwanbond setup
```

---

## macOS Installation

### One-Click Install

```bash
# Download and run installer
curl -fsSL https://raw.githubusercontent.com/thelastdreamer/MultiWANBond/main/install.sh | bash
```

### Using Homebrew (Coming Soon)

```bash
brew tap thelastdreamer/multiwanbond
brew install multiwanbond
```

### Manual Installation

```bash
# Install Homebrew (if not installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install Go
brew install go git

# Clone and setup
git clone https://github.com/thelastdreamer/MultiWANBond.git
cd MultiWANBond
go mod download
go build -o multiwanbond ./cmd/server

# Run setup
./multiwanbond setup
```

---

## Android Installation

### APK Installation (Coming Soon)

1. Download the APK from [Releases](https://github.com/thelastdreamer/MultiWANBond/releases)
2. Enable "Install from Unknown Sources" in Settings
3. Install the APK
4. Open MultiWANBond app
5. Follow the setup wizard

### Building from Source

```bash
# Install gomobile
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init

# Build for Android
cd MultiWANBond
gomobile bind -target=android/arm64 -o multiwanbond.aar ./pkg/...

# Import the .aar file into your Android Studio project
```

---

## iOS Installation

### TestFlight (Coming Soon)

1. Install TestFlight from App Store
2. Open TestFlight invite link
3. Install MultiWANBond beta
4. Open app and follow setup wizard

### Building from Source

```bash
# Install gomobile
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init

# Build for iOS
cd MultiWANBond
gomobile bind -target=ios/arm64 -o MultiWANBond.xcframework ./pkg/...

# Import the .xcframework file into your Xcode project
```

---

## First-Time Setup

After installation, run the interactive setup wizard:

```bash
multiwanbond setup
```

### Setup Wizard Steps

#### 1. Select Operation Mode

```
1. Standalone   - Run on a single machine (testing/development)
2. Client       - Connect to a remote server
3. Server       - Accept connections from clients
```

**Choose Standalone** if:
- Testing on a single machine
- Don't have a remote server yet
- Want to test locally first

**Choose Client** if:
- Connecting to a remote MultiWANBond server
- Want to bond multiple connections to a central point

**Choose Server** if:
- Setting up the central bonding server
- Other clients will connect to you

#### 2. Select Network Interfaces

The wizard will detect all network interfaces:

```
Available network interfaces:

  1. Wi-Fi
     Status: UP | Type: physical
     IPv4: 192.168.1.100
     Speed: 300 Mbps

  2. Ethernet
     Status: UP | Type: physical
     IPv4: 192.168.2.100
     Speed: 1000 Mbps

  3. NordLynx (VPN)
     Status: UP | Type: tunnel
     IPv4: 10.5.0.2

Select interfaces to use for WAN bonding.
Enter numbers separated by commas (e.g., 1,2,3)

Select interfaces: 1,2
```

**Tips**:
- Select all connections you want to bond
- You can add/remove interfaces later
- Virtual adapters (VPN tunnels) can also be used

#### 3. Configure WAN Interfaces

For each selected interface, configure:

```
Configuring WAN 1: Wi-Fi

  Friendly name [WAN1-Wi-Fi]: Home WiFi
  Weight (1-1000) [100]: 50
```

**Weight** determines traffic distribution:
- Higher weight = more traffic
- Use bandwidth as a guideline (1 Gbps = 1000, 100 Mbps = 100)

#### 4. Server Configuration

**If you selected "Client" mode**:

```
Configure remote server address:

  Leave empty to configure later.

  Server address (e.g., server.example.com:9000):
```

You can leave this empty and configure it later when you're ready.

**If you selected "Server" mode**:

```
Configure server listening address:

  Listen address [0.0.0.0]: 0.0.0.0
  Listen port [9000]: 9000
```

#### 5. Security Settings

```
Enable encryption? (recommended) [Y/n]: Y

  1. ChaCha20-Poly1305 (fast, recommended)
  2. AES-256-GCM (hardware accelerated)

Select encryption [1-2]: 1

A pre-shared key is required for encryption.
This must be the same on client and server.

  Enter pre-shared key (leave empty to generate):

  Generated key: aB3dK9pL2mN7qR5tV8xZ1cF4gH6jK0sW

  ⚠ Save this key! You'll need it for the other side.
```

**Important**: Save the generated key! You'll need to enter the same key on both client and server.

#### 6. Review and Save

```
Configuration Summary

Mode:          client
WAN Count:     2

WAN Interfaces:
  - Home WiFi (Wi-Fi) - weight: 50 - enabled
  - Office LAN (Ethernet) - weight: 100 - enabled

Remote Server: (not configured)

Encryption:    chacha20poly1305

Save this configuration? [Y/n]: Y

Configuration saved to: /home/user/.config/multiwanbond/config.json
```

---

## Managing WAN Interfaces

After initial setup, you can manage WAN interfaces using CLI commands.

### List WAN Interfaces

```bash
multiwanbond wan list
```

Output:
```
Configured WAN Interfaces:

  [1] Home WiFi
      Interface: wlan0
      Weight: 50
      Status: enabled

  [2] Office LAN
      Interface: eth0
      Weight: 100
      Status: enabled
```

### Add New WAN Interface

```bash
multiwanbond wan add
```

This runs the interface selection wizard again for adding new WANs.

### Remove WAN Interface

```bash
multiwanbond wan remove 2
```

Removes WAN with ID 2.

### Enable/Disable WAN

```bash
# Disable a WAN temporarily
multiwanbond wan disable 1

# Re-enable it
multiwanbond wan enable 1
```

### View Configuration

```bash
# Show current configuration
multiwanbond config show

# Validate configuration
multiwanbond config validate
```

### Edit Configuration Manually

Configuration is stored in JSON format:

**Linux/macOS**: `~/.config/multiwanbond/config.json`
**Windows**: `%APPDATA%\multiwanbond\config.json`

Example configuration:

```json
{
  "version": "1.0",
  "mode": "client",
  "wans": [
    {
      "id": 1,
      "name": "Home WiFi",
      "interface": "wlan0",
      "enabled": true,
      "weight": 50
    },
    {
      "id": 2,
      "name": "Office LAN",
      "interface": "eth0",
      "enabled": true,
      "weight": 100
    }
  ],
  "server": {
    "remote_address": "server.example.com:9000"
  },
  "security": {
    "encryption_enabled": true,
    "encryption_type": "chacha20poly1305",
    "pre_shared_key": "aB3dK9pL2mN7qR5tV8xZ1cF4gH6jK0sW"
  }
}
```

---

## Starting MultiWANBond

After setup, start the service:

```bash
# Start in foreground
multiwanbond start

# With custom config
multiwanbond start --config /path/to/config.json
```

### Run as System Service

#### Linux (systemd)

Create `/etc/systemd/system/multiwanbond.service`:

```ini
[Unit]
Description=MultiWANBond Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/multiwanbond start --config /etc/multiwanbond/config.json
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable multiwanbond
sudo systemctl start multiwanbond
```

#### Windows (Service)

```powershell
# Install as Windows Service (requires admin)
sc.exe create MultiWANBond binPath= "C:\Program Files\MultiWANBond\multiwanbond.exe start"
sc.exe start MultiWANBond
```

#### macOS (launchd)

Create `~/Library/LaunchAgents/com.multiwanbond.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.multiwanbond</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/multiwanbond</string>
        <string>start</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

Load and start:

```bash
launchctl load ~/Library/LaunchAgents/com.multiwanbond.plist
```

---

## Troubleshooting

### Installation Issues

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for comprehensive troubleshooting guide.

### Quick Fixes

**Go not found**:
```bash
# Install Go from https://go.dev/dl/
# Or use package manager (apt, dnf, brew)
```

**Permission denied**:
```bash
# Linux/macOS: Use sudo for system-wide installation
sudo ./install.sh

# Windows: Run PowerShell as Administrator
```

**Network interfaces not detected**:
```bash
# Linux: Ensure you have required permissions
sudo multiwanbond setup

# Check interfaces manually
ip addr show        # Linux
ifconfig           # macOS
ipconfig /all      # Windows
```

---

## Next Steps

After installation and setup:

1. **Test the connection**: Start MultiWANBond and verify WANs are detected
2. **Configure client/server**: If using client/server mode, set up both sides
3. **Monitor performance**: Use built-in metrics or web UI (coming soon)
4. **Optimize settings**: Adjust weights, enable/disable WANs as needed

For more information:
- [README.md](README.md) - Full documentation
- [QUICKSTART.md](QUICKSTART.md) - Quick start guide
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Troubleshooting
- [GitHub Issues](https://github.com/thelastdreamer/MultiWANBond/issues) - Report bugs

# How to Build and Run MultiWANBond

Complete guide to building, configuring, and running MultiWANBond on any platform.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Building from Source](#building-from-source)
3. [Building Releases](#building-releases)
4. [Running Tests](#running-tests)
5. [Configuration](#configuration)
6. [Running the Server](#running-the-server)
7. [Monitoring](#monitoring)
8. [Next Steps](#next-steps)

---

## Prerequisites

### All Platforms
- **Go 1.21+**: Download from [golang.org](https://golang.org/dl/)
- **Git**: For cloning the repository

### Linux (for full features)
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install build-essential iproute2 iptables

# RHEL/CentOS/Fedora
sudo dnf install gcc iproute iptables

# Arch Linux
sudo pacman -S base-devel iproute2 iptables
```

### Windows
- Run PowerShell as Administrator for network operations
- Optional: [Npcap](https://npcap.com/) for advanced packet capture

### macOS
```bash
# Install Xcode Command Line Tools
xcode-select --install

# Install Homebrew (if not already installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

---

## Building from Source

### 1. Clone the Repository
```bash
git clone https://github.com/thelastdreamer/MultiWANBond.git
cd MultiWANBond
```

### 2. Download Dependencies
```bash
go mod download
go mod tidy
```

### 3. Build for Your Platform

#### Quick Build (Current Platform)
```bash
go build -o multiwanbond cmd/server/main.go
```

#### Optimized Build (Smaller Binary)
```bash
# Linux/macOS
go build -ldflags="-s -w" -o multiwanbond cmd/server/main.go

# Windows
go build -ldflags="-s -w" -o multiwanbond.exe cmd/server/main.go
```

#### Build with Version Info
```bash
VERSION="1.0.0"
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

go build \
  -ldflags="-s -w -X main.version=$VERSION -X main.buildTime=$BUILD_TIME" \
  -o multiwanbond \
  cmd/server/main.go
```

---

## Building Releases

Build binaries for all platforms at once:

### Linux/macOS
```bash
chmod +x build-releases.sh
./build-releases.sh
```

### Windows PowerShell
```powershell
.\build-releases.ps1
```

### Custom Version
```bash
# Linux/macOS
VERSION=1.2.3 ./build-releases.sh

# Windows
.\build-releases.ps1 -Version "1.2.3"
```

### Output
Releases will be created in the `releases/` directory:
```
releases/
├── multiwanbond-1.0.0-linux-amd64.tar.gz
├── multiwanbond-1.0.0-linux-arm64.tar.gz
├── multiwanbond-1.0.0-linux-arm.tar.gz
├── multiwanbond-1.0.0-windows-amd64.zip
├── multiwanbond-1.0.0-windows-arm64.zip
├── multiwanbond-1.0.0-darwin-amd64.tar.gz
├── multiwanbond-1.0.0-darwin-arm64.tar.gz
└── SHA256SUMS
```

---

## Running Tests

### Test All Components (Recommended First Step)
```bash
# Network detection
go run cmd/test/network_detect.go

# Health checking
go run cmd/test/health_checker.go

# NAT traversal
go run cmd/test/nat_traversal.go

# Routing
go run cmd/test/routing_demo.go

# DPI
go run cmd/test/dpi_demo.go

# Web UI
go run cmd/test/webui_demo.go

# Metrics
go run cmd/test/metrics_demo.go

# Security
go run cmd/test/security_demo.go

# Full integration test
go run cmd/test/final_integration.go
```

### Expected Output
Each test should show:
- ✓ Component initialization successful
- ✓ Feature tests passing
- Test score (e.g., "Tests Passed: 10/10 (100%)")

### If Tests Fail
1. Check prerequisites are installed
2. Verify network connectivity
3. Run with elevated privileges (sudo/Administrator)
4. See [TROUBLESHOOTING.md](TROUBLESHOOTING.md)

---

## Configuration

### 1. Detect Your Networks
```bash
go run cmd/test/network_detect.go
```

Note the interface names that are **up** and have connectivity.

### 2. Create Configuration File

Create `config.yaml`:

```yaml
# Server Configuration
server:
  listen_addr: "0.0.0.0"
  listen_port: 9000

# WAN Interfaces
wans:
  - id: 1
    interface: "eth0"
    enabled: true
    weight: 100
    description: "Primary Cable Connection"

  - id: 2
    interface: "wlan0"
    enabled: true
    weight: 80
    description: "Backup WiFi"

# Health Monitoring
health:
  check_interval: "5s"
  timeout: "3s"
  retry_count: 3
  method: "smart"  # auto, icmp, http, dns, tcp
  check_hosts:
    - "8.8.8.8"
    - "1.1.1.1"

# Security
security:
  encryption_enabled: true
  encryption_type: "chacha20poly1305"  # or "aes256gcm"
  auth_enabled: true
  auth_type: "psk"  # psk, token, certificate
  pre_shared_key: "CHANGE-THIS-SECRET-KEY"
  key_rotation_interval: "24h"

# Web UI
webui:
  enabled: true
  listen_addr: "0.0.0.0"
  listen_port: 8080
  enable_cors: true
  enable_auth: false  # Set true for production

# Metrics
metrics:
  enabled: true
  prometheus_enabled: true
  prometheus_port: 9090
  collection_interval: "10s"
  retention_period: "7d"

# NAT Traversal
nat:
  enabled: true
  stun_server: "stun.l.google.com:19302"
  refresh_interval: "25s"

# DPI (Deep Packet Inspection)
dpi:
  enabled: true
  max_flows: 100000

# Routing
routing:
  policy_based: true
  table_id_start: 100
  mark_base: 100

# FEC (Forward Error Correction)
fec:
  enabled: true
  redundancy: 0.2  # 20% overhead

# Logging
logging:
  level: "info"  # debug, info, warn, error
  file: "/var/log/multiwanbond.log"
  max_size_mb: 100
  max_backups: 3
  compress: true
```

### Configuration Examples

#### Simple Home Setup (2 WANs)
See [examples/home-simple.yaml](examples/home-simple.yaml)

#### High Availability (3+ WANs with failover)
See [examples/high-availability.yaml](examples/high-availability.yaml)

#### Gaming/Low Latency
See [examples/gaming.yaml](examples/gaming.yaml)

#### Maximum Bandwidth
See [examples/max-bandwidth.yaml](examples/max-bandwidth.yaml)

---

## Running the Server

### Development Mode (with live config)

#### Linux
```bash
sudo go run cmd/server/main.go --config config.yaml
```

#### Windows (PowerShell as Administrator)
```powershell
go run cmd\server\main.go --config config.yaml
```

#### macOS
```bash
sudo go run cmd/server/main.go --config config.yaml
```

### Production Mode (with compiled binary)

#### Linux
```bash
# Run directly
sudo ./multiwanbond --config /etc/multiwanbond/config.yaml

# Or install as systemd service
sudo cp multiwanbond /usr/local/bin/
sudo cp examples/multiwanbond.service /etc/systemd/system/
sudo systemctl enable multiwanbond
sudo systemctl start multiwanbond
sudo systemctl status multiwanbond

# View logs
sudo journalctl -u multiwanbond -f
```

#### Windows
```powershell
# Run directly
.\multiwanbond.exe --config config.yaml

# Or install as Windows Service
sc.exe create MultiWANBond binPath= "C:\path\to\multiwanbond.exe --config C:\path\to\config.yaml"
sc.exe start MultiWANBond

# View logs
Get-Content -Wait C:\path\to\multiwanbond.log
```

#### macOS
```bash
# Run directly
sudo ./multiwanbond --config config.yaml

# Or install as launchd service
sudo cp multiwanbond /usr/local/bin/
sudo cp examples/com.multiwanbond.plist /Library/LaunchDaemons/
sudo launchctl load /Library/LaunchDaemons/com.multiwanbond.plist
sudo launchctl start com.multiwanbond

# View logs
tail -f /var/log/multiwanbond.log
```

### Command-Line Options

```bash
./multiwanbond --help

Usage: multiwanbond [options]

Options:
  --config string
        Configuration file path (default "config.yaml")

  --log-level string
        Log level: debug, info, warn, error (default "info")

  --version
        Print version and exit

  --test-config
        Test configuration and exit

Examples:
  multiwanbond --config /etc/multiwanbond/config.yaml
  multiwanbond --log-level debug
  multiwanbond --test-config
```

---

## Monitoring

### Web Dashboard
Open in your browser:
```
http://localhost:8080
```

### API Endpoints

#### Dashboard
```bash
curl http://localhost:8080/api/dashboard
```

#### WAN Status
```bash
curl http://localhost:8080/api/wans/status | jq
```

#### Health Checks
```bash
curl http://localhost:8080/api/health | jq
```

#### Active Flows
```bash
curl http://localhost:8080/api/flows | jq
```

#### Metrics
```bash
curl http://localhost:8080/api/metrics | jq
```

### Prometheus Metrics
```bash
curl http://localhost:9090/metrics
```

### Grafana Dashboard
1. Install Grafana
2. Add Prometheus data source: `http://localhost:9090`
3. Import dashboard from `examples/grafana-dashboard.json`

### Real-Time Monitoring with WebSocket
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('Event:', data.type, data.data);
};
```

---

## Next Steps

### 1. Verify Everything is Working
```bash
# Check all WANs are up
curl http://localhost:8080/api/wans/status | jq '.[] | {name, status, health}'

# Check metrics are being collected
curl http://localhost:9090/metrics | grep multiwanbond_active_wans

# Monitor logs
tail -f /var/log/multiwanbond.log
```

### 2. Performance Testing
```bash
# Bandwidth test
iperf3 -c test-server.com -t 60 -P 4

# Latency test
ping -c 100 8.8.8.8 | tail -n 2

# Full system test
go run cmd/test/final_integration.go
```

### 3. Production Hardening
- [ ] Change default pre-shared key
- [ ] Enable Web UI authentication
- [ ] Configure firewall rules
- [ ] Set up log rotation
- [ ] Configure automatic backups
- [ ] Set up monitoring alerts
- [ ] Review security settings

### 4. Optional Enhancements
- [ ] Set up bandwidth quotas
- [ ] Configure DPI rules
- [ ] Set up failover policies
- [ ] Configure QoS rules
- [ ] Enable advanced routing

---

## Troubleshooting

If you encounter issues, see:
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** - Comprehensive troubleshooting guide
- **[QUICKSTART.md](QUICKSTART.md)** - Quick start with minimal configuration
- **GitHub Issues** - https://github.com/thelastdreamer/MultiWANBond/issues

---

## Summary of Commands

```bash
# Quick Start Commands
git clone https://github.com/thelastdreamer/MultiWANBond.git
cd MultiWANBond
go mod download
go run cmd/test/network_detect.go
go build -o multiwanbond cmd/server/main.go
sudo ./multiwanbond --config config.yaml

# Testing Commands
go run cmd/test/network_detect.go
go run cmd/test/health_checker.go
go run cmd/test/security_demo.go
go run cmd/test/final_integration.go

# Building Commands
go build -o multiwanbond cmd/server/main.go
./build-releases.sh               # All platforms
VERSION=1.0.0 ./build-releases.sh # Custom version

# Running Commands
sudo ./multiwanbond --config config.yaml
./multiwanbond --test-config
./multiwanbond --version

# Monitoring Commands
curl http://localhost:8080/api/dashboard
curl http://localhost:9090/metrics
tail -f /var/log/multiwanbond.log
```

---

**You're now ready to run MultiWANBond!** 🚀

For questions or issues, open an issue on GitHub:
https://github.com/thelastdreamer/MultiWANBond/issues

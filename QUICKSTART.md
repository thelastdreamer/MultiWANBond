# MultiWANBond - Quick Start Guide

This guide will help you get MultiWANBond up and running in minutes.

## Step 1: Detect Your Network Interfaces

First, let's discover what network interfaces are available:

```bash
# Run the network detection test
go run cmd/test/network_detect.go
```

This will show you all available interfaces. Look for interfaces that are **up** and have **internet connectivity**.

Example output:
```
Found 14 network interfaces:

1. eth0 (physical)
   Admin State:  up
   Has Internet: true

2. wlan0 (physical)
   Admin State:  up
   Has Internet: true
```

Make note of the interface names you want to use (e.g., `eth0`, `wlan0`).

## Step 2: Create Your Configuration

Create a file named `config.json` in your MultiWANBond directory:

```json
{
  "listen_port": 9000,
  "wans": [
    {
      "id": 1,
      "interface_name": "eth0",
      "enabled": true,
      "weight": 100
    },
    {
      "id": 2,
      "interface_name": "wlan0",
      "enabled": true,
      "weight": 80
    }
  ],
  "health": {
    "check_interval_ms": 5000,
    "timeout_ms": 3000,
    "retry_count": 3,
    "check_hosts": ["8.8.8.8", "1.1.1.1"]
  },
  "security": {
    "encryption_enabled": true,
    "encryption_type": "chacha20poly1305",
    "pre_shared_key": "CHANGE-THIS-TO-A-STRONG-SECRET-KEY"
  }
}
```

**Important**: Replace the `pre_shared_key` with your own secure random string!

## Step 3: Test Individual Components

Before running the full system, test each component:

### Test Health Checking
```bash
go run cmd/test/health_checker.go
```

Expected output:
```
✓ Health checker created
✓ ICMP ping successful
✓ HTTP check successful
```

### Test Security
```bash
go run cmd/test/security_demo.go
```

Expected output:
```
✓ AES-256-GCM Encryption: PASS
✓ ChaCha20-Poly1305 Encryption: PASS
✓ PSK Authentication: PASS
```

### Test Metrics
```bash
go run cmd/test/metrics_demo.go
```

## Step 4: Run the Server

### On Linux
```bash
# Requires root for network configuration
sudo go run cmd/server/main.go --config config.json
```

### On Windows
```powershell
# Run PowerShell as Administrator
go run cmd/server/main.go --config config.json
```

### On macOS
```bash
# Requires sudo for network configuration
sudo go run cmd/server/main.go --config config.json
```

## Step 5: Verify It's Working

### Check Web Interface
Open your browser and go to:
```
http://localhost:8080
```

### Check Prometheus Metrics
```bash
curl http://localhost:9090/metrics
```

### Check API Status
```bash
curl http://localhost:8080/api/dashboard
```

Example response:
```json
{
  "uptime": 120,
  "active_wans": 2,
  "total_packets": 12345,
  "current_pps": 150
}
```

## Common Configuration Examples

### Example 1: Home Office Setup
Two internet connections for redundancy:

```json
{
  "wans": [
    {
      "id": 1,
      "interface_name": "eth0",
      "enabled": true,
      "weight": 100,
      "description": "Primary Cable"
    },
    {
      "id": 2,
      "interface_name": "wlan0",
      "enabled": true,
      "weight": 50,
      "description": "Backup WiFi"
    }
  ],
  "failover": {
    "enabled": true,
    "failover_threshold_ms": 1000
  }
}
```

### Example 2: Maximum Performance
Aggregate multiple high-speed connections:

```json
{
  "wans": [
    {
      "id": 1,
      "interface_name": "eth0",
      "enabled": true,
      "weight": 100
    },
    {
      "id": 2,
      "interface_name": "eth1",
      "enabled": true,
      "weight": 100
    },
    {
      "id": 3,
      "interface_name": "wlan0",
      "enabled": true,
      "weight": 80
    }
  ],
  "load_balancing": {
    "mode": "round_robin"
  }
}
```

### Example 3: Gaming/Low Latency
Optimized for lowest latency:

```json
{
  "wans": [
    {
      "id": 1,
      "interface_name": "eth0",
      "enabled": true,
      "weight": 100
    }
  ],
  "routing": {
    "latency_priority": true,
    "sticky_sessions": true
  },
  "health": {
    "check_interval_ms": 1000
  }
}
```

## Troubleshooting

### Issue: "Permission denied" error
**Solution**: Run with sudo/administrator privileges
```bash
sudo go run cmd/server/main.go --config config.json
```

### Issue: Interfaces not detected
**Solution**: Check interface names
```bash
# Linux
ip link show

# Windows
ipconfig

# macOS
ifconfig
```

### Issue: No internet after starting
**Solution**: Check WAN health status
```bash
curl http://localhost:8080/api/wans/status
```

### Issue: High CPU usage
**Solutions**:
1. Increase health check intervals in config
2. Disable DPI if not needed
3. Reduce metrics collection frequency

## Next Steps

1. **Read the full documentation**: See [README.md](README.md)
2. **Configure advanced features**: See [CONFIGURATION.md](docs/CONFIGURATION.md)
3. **Set up monitoring**: See [MONITORING.md](docs/MONITORING.md)
4. **Optimize performance**: See [PERFORMANCE.md](docs/PERFORMANCE.md)
5. **Secure your installation**: See [SECURITY.md](docs/SECURITY.md)

## Getting Help

- **GitHub Issues**: https://github.com/thelastdreamer/MultiWANBond/issues
- **Discussions**: https://github.com/thelastdreamer/MultiWANBond/discussions
- **Documentation**: [docs/](docs/)

## Quick Command Reference

```bash
# Network detection
go run cmd/test/network_detect.go

# Test all components
go run cmd/test/final_integration.go

# Build for your platform
go build -o multiwanbond cmd/server/main.go

# Build all platforms
./build-releases.sh              # Linux/macOS
.\build-releases.ps1             # Windows

# Run server
./multiwanbond --config config.json

# View logs
tail -f /var/log/multiwanbond.log  # Linux
Get-Content -Wait multiwanbond.log # Windows

# Check metrics
curl http://localhost:9090/metrics

# Check API
curl http://localhost:8080/api/dashboard
```

---

**You're now ready to use MultiWANBond!** 🎉

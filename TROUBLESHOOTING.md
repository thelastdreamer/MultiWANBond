# MultiWANBond - Troubleshooting Guide

This guide covers common issues and their solutions.

## Table of Contents

- [Installation Issues](#installation-issues)
- [Permission Issues](#permission-issues)
- [Network Detection Issues](#network-detection-issues)
- [Connection Issues](#connection-issues)
- [Performance Issues](#performance-issues)
- [Security Issues](#security-issues)
- [Platform-Specific Issues](#platform-specific-issues)
- [Debugging Tools](#debugging-tools)

---

## Installation Issues

### Issue: Go version too old
**Error**: `go: directive requires go1.21 or later`

**Solution**:
```bash
# Check your Go version
go version

# Install latest Go from https://golang.org/dl/
# Or use your package manager:

# Ubuntu/Debian
sudo add-apt-repository ppa:longsleep/golang-backports
sudo apt-get update
sudo apt-get install golang-go

# macOS
brew install go

# Windows
# Download installer from golang.org
```

### Issue: Missing dependencies
**Error**: `cannot find package`

**Solution**:
```bash
# Download all dependencies
go mod download

# Verify go.mod and go.sum
go mod tidy

# If using vendor directory
go mod vendor
```

### Issue: Build fails on Windows
**Error**: `undefined: syscall.SYS_*`

**Solution**: Some features are Linux-only. Build with appropriate tags:
```powershell
go build -tags "!linux" -o multiwanbond.exe cmd/server/main.go
```

---

## Permission Issues

### Issue: "Permission denied" when starting server

**Linux/macOS Solution**:
```bash
# Option 1: Run with sudo (recommended for testing)
sudo ./multiwanbond --config config.json

# Option 2: Add capabilities (better for production)
sudo setcap cap_net_admin,cap_net_raw=+ep ./multiwanbond

# Option 3: Run as systemd service (best for production)
sudo cp multiwanbond.service /etc/systemd/system/
sudo systemctl enable multiwanbond
sudo systemctl start multiwanbond
```

**Windows Solution**:
```powershell
# Right-click PowerShell/CMD and select "Run as Administrator"
# Or use RunAs:
runas /user:Administrator "multiwanbond.exe --config config.json"
```

### Issue: Cannot bind to port 80 or 443

**Solution**:
```bash
# Linux: Use authbind
sudo apt-get install authbind
authbind --deep ./multiwanbond --config config.json

# Or use iptables redirect
sudo iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8080
```

---

## Network Detection Issues

### Issue: No interfaces detected

**Diagnosis**:
```bash
# Run network detection test
go run cmd/test/network_detect.go

# Check system network interfaces
# Linux
ip link show
ip addr show

# Windows
ipconfig /all

# macOS
ifconfig
networksetup -listallhardwareports
```

**Solutions**:

1. **Interface is down**:
```bash
# Linux
sudo ip link set eth0 up

# Windows
netsh interface set interface "Ethernet" admin=enabled

# macOS
sudo ifconfig en0 up
```

2. **Interface has no IP**:
```bash
# Linux
sudo dhclient eth0
# or
sudo ip addr add 192.168.1.100/24 dev eth0

# Windows
netsh interface ip set address "Ethernet" dhcp

# macOS
sudo ifconfig en0 inet 192.168.1.100 netmask 255.255.255.0
```

3. **Driver issues (Linux)**:
```bash
# Check for driver errors
dmesg | grep -i network
dmesg | grep -i eth

# Reload network driver
sudo modprobe -r <driver_name>
sudo modprobe <driver_name>
```

### Issue: Interface detected but "No Internet"

**Diagnosis**:
```bash
# Test connectivity
ping -c 4 8.8.8.8
ping -c 4 1.1.1.1

# Check routing
ip route show
```

**Solutions**:

1. **No default gateway**:
```bash
# Linux
sudo ip route add default via 192.168.1.1 dev eth0

# Windows
route add 0.0.0.0 mask 0.0.0.0 192.168.1.1

# macOS
sudo route add default 192.168.1.1
```

2. **DNS issues**:
```bash
# Test DNS
nslookup google.com
dig google.com

# Fix DNS (Linux)
echo "nameserver 8.8.8.8" | sudo tee /etc/resolv.conf
```

---

## Connection Issues

### Issue: WANs showing as "Down"

**Diagnosis**:
```bash
# Check WAN status via API
curl http://localhost:8080/api/wans/status

# Check health checker
go run cmd/test/health_checker.go

# Manual ping test
ping -c 10 8.8.8.8
```

**Solutions**:

1. **Firewall blocking ICMP**:
```json
{
  "health": {
    "method": "http",
    "check_url": "http://www.google.com/generate_204"
  }
}
```

2. **High latency/packet loss**:
```json
{
  "health": {
    "max_latency_ms": 500,
    "max_packet_loss_percent": 10.0,
    "retry_count": 5
  }
}
```

3. **Check interval too aggressive**:
```json
{
  "health": {
    "check_interval_ms": 10000
  }
}
```

### Issue: Frequent failovers

**Diagnosis**:
```bash
# Check logs for failover events
tail -f /var/log/multiwanbond.log | grep "failover"

# Check WAN metrics
curl http://localhost:8080/api/wans/status | jq
```

**Solutions**:

1. **Increase failure threshold**:
```json
{
  "health": {
    "failure_threshold": 5,
    "check_interval_ms": 5000
  }
}
```

2. **Adjust health check parameters**:
```json
{
  "health": {
    "timeout_ms": 5000,
    "retry_count": 3,
    "retry_interval_ms": 1000
  }
}
```

### Issue: NAT traversal failing

**Diagnosis**:
```bash
# Test NAT detection
go run cmd/test/nat_traversal.go

# Check STUN connectivity
telnet stun.l.google.com 19302
```

**Solutions**:

1. **Firewall blocking STUN**:
```bash
# Allow UDP port 19302
sudo ufw allow 19302/udp
sudo firewall-cmd --add-port=19302/udp --permanent
```

2. **Use different STUN servers**:
```json
{
  "nat": {
    "stun_servers": [
      "stun.l.google.com:19302",
      "stun1.l.google.com:19302",
      "stun2.l.google.com:19302"
    ]
  }
}
```

3. **CGNAT detected**:
```json
{
  "nat": {
    "force_relay": true
  }
}
```

---

## Performance Issues

### Issue: High CPU usage

**Diagnosis**:
```bash
# Check CPU usage
top -p $(pgrep multiwanbond)
ps aux | grep multiwanbond

# Profile CPU
go tool pprof http://localhost:8080/debug/pprof/profile
```

**Solutions**:

1. **Disable DPI if not needed**:
```json
{
  "dpi": {
    "enabled": false
  }
}
```

2. **Reduce health check frequency**:
```json
{
  "health": {
    "check_interval_ms": 10000
  }
}
```

3. **Reduce metrics collection**:
```json
{
  "metrics": {
    "collection_interval_ms": 30000,
    "max_data_points": 1000
  }
}
```

4. **Disable FEC for low packet loss**:
```json
{
  "fec": {
    "enabled": false
  }
}
```

### Issue: High memory usage

**Diagnosis**:
```bash
# Check memory usage
ps aux | grep multiwanbond

# Profile memory
go tool pprof http://localhost:8080/debug/pprof/heap
```

**Solutions**:

1. **Reduce buffer sizes**:
```json
{
  "packet": {
    "buffer_size": 500,
    "max_flows": 10000
  }
}
```

2. **Reduce metrics retention**:
```json
{
  "metrics": {
    "retention_period": "24h",
    "max_data_points": 5000
  }
}
```

3. **Enable memory limits**:
```bash
# Linux: Use cgroups
sudo cgcreate -g memory:/multiwanbond
echo 512M | sudo tee /sys/fs/cgroup/memory/multiwanbond/memory.limit_in_bytes
sudo cgexec -g memory:/multiwanbond ./multiwanbond
```

### Issue: High latency

**Causes and Solutions**:

1. **FEC overhead** - Disable FEC
2. **DPI processing** - Disable DPI or reduce signatures
3. **Packet reordering** - Reduce reorder buffer
4. **Encryption** - Use ChaCha20 instead of AES on ARM

```json
{
  "fec": {
    "enabled": false
  },
  "dpi": {
    "enabled": false
  },
  "packet": {
    "reorder_buffer_size": 100,
    "reorder_timeout_ms": 50
  },
  "security": {
    "encryption_type": "chacha20poly1305"
  }
}
```

### Issue: Low throughput

**Diagnosis**:
```bash
# Test individual WANs
iperf3 -c test-server.com -B eth0-ip
iperf3 -c test-server.com -B wlan0-ip

# Test combined
iperf3 -c test-server.com -P 4
```

**Solutions**:

1. **Increase packet buffer**:
```json
{
  "packet": {
    "buffer_size": 5000,
    "queue_size": 10000
  }
}
```

2. **Optimize FEC**:
```json
{
  "fec": {
    "enabled": true,
    "redundancy": 0.1
  }
}
```

3. **Use better load balancing**:
```json
{
  "routing": {
    "mode": "least_used",
    "per_flow_routing": true
  }
}
```

---

## Security Issues

### Issue: Authentication failing

**Diagnosis**:
```bash
# Check security events
curl http://localhost:8080/api/events | jq '.[] | select(.type == "auth_failure")'

# Test authentication
go run cmd/test/security_demo.go
```

**Solutions**:

1. **PSK mismatch**:
```json
{
  "security": {
    "pre_shared_key": "EXACT-SAME-KEY-ON-BOTH-SIDES"
  }
}
```

2. **Clock skew (tokens)**:
```bash
# Sync time
sudo ntpdate time.nist.gov
# or
sudo timedatectl set-ntp true
```

3. **Certificate issues**:
```bash
# Verify certificate
openssl x509 -in cert.pem -text -noout

# Check expiration
openssl x509 -in cert.pem -noout -dates
```

### Issue: Encryption errors

**Diagnosis**:
```bash
# Check encryption events
curl http://localhost:8080/api/events | jq '.[] | select(.type == "encryption_error")'
```

**Solutions**:

1. **Use compatible encryption**:
```json
{
  "security": {
    "encryption_type": "chacha20poly1305"
  }
}
```

2. **Disable encryption for testing**:
```json
{
  "security": {
    "encryption_enabled": false
  }
}
```

### Issue: Rate limiting blocking legitimate traffic

**Solution**:
```json
{
  "security": {
    "rate_limit": {
      "enabled": true,
      "max_connections_per_ip": 1000,
      "window_ms": 60000
    }
  }
}
```

---

## Platform-Specific Issues

### Linux Issues

**Issue: Routing table conflicts**
```bash
# Check routing tables
ip route show table all

# Clear MultiWANBond tables
sudo ip route flush table 100
sudo ip route flush table 101

# Reset rules
sudo ip rule flush
sudo ip rule add from all lookup main
```

**Issue: Netlink errors**
```bash
# Install/reinstall netlink library
go get -u github.com/vishvananda/netlink

# Check kernel support
uname -r
# Requires kernel 4.9+
```

### Windows Issues

**Issue: WinPcap/Npcap required**
```powershell
# Install Npcap (WinPcap replacement)
# Download from https://npcap.com/

# Or use Chocolatey
choco install npcap
```

**Issue: Windows Firewall blocking**
```powershell
# Allow MultiWANBond through firewall
New-NetFirewallRule -DisplayName "MultiWANBond" -Direction Inbound -Program "C:\path\to\multiwanbond.exe" -Action Allow
New-NetFirewallRule -DisplayName "MultiWANBond" -Direction Outbound -Program "C:\path\to\multiwanbond.exe" -Action Allow

# Or disable for testing (NOT RECOMMENDED FOR PRODUCTION)
Set-NetFirewallProfile -Profile Domain,Public,Private -Enabled False
```

### macOS Issues

**Issue: Network extension required**
```bash
# Grant network extension permission
# System Preferences → Security & Privacy → Privacy → Network

# Or disable SIP (NOT RECOMMENDED)
csrutil disable
```

**Issue: Packet filter conflicts**
```bash
# Check pf rules
sudo pfctl -s rules

# Flush pf rules (careful!)
sudo pfctl -F all
```

---

## Debugging Tools

### Enable Debug Logging
```bash
# Start with debug logging
./multiwanbond --config config.json --log-level debug

# Or in config:
{
  "logging": {
    "level": "debug",
    "file": "/var/log/multiwanbond.log"
  }
}
```

### Packet Capture
```bash
# Capture on specific interface
sudo tcpdump -i eth0 -w capture.pcap

# Capture MultiWANBond traffic
sudo tcpdump -i any port 9000 -w multiwanbond.pcap

# Analyze with Wireshark
wireshark capture.pcap
```

### API Debugging
```bash
# Get all WANs
curl -v http://localhost:8080/api/wans

# Get system metrics
curl http://localhost:8080/api/dashboard | jq

# Get health status
curl http://localhost:8080/api/health | jq

# Get security events
curl http://localhost:8080/api/events | jq
```

### Performance Profiling
```bash
# CPU profile (30 seconds)
go tool pprof -http=:8081 http://localhost:8080/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof -http=:8081 http://localhost:8080/debug/pprof/heap

# Goroutine profile
go tool pprof -http=:8081 http://localhost:8080/debug/pprof/goroutine
```

### Network Tests
```bash
# Test bandwidth
iperf3 -c test-server.com -t 60

# Test latency
mtr --report --report-cycles 100 test-server.com

# Test packet loss
ping -c 1000 test-server.com | grep loss

# Test DNS
dig @8.8.8.8 google.com
nslookup google.com

# Test NAT type
go run cmd/test/nat_traversal.go
```

---

## Getting More Help

If you're still experiencing issues:

1. **Check the logs**: Look for ERROR or WARN messages
2. **Run all test demos**: Execute all `cmd/test/*_demo.go` files
3. **Search existing issues**: https://github.com/thelastdreamer/MultiWANBond/issues
4. **Create a new issue**: Include:
   - OS and version
   - Go version
   - Configuration file (remove sensitive data!)
   - Log output (last 50 lines)
   - Output of `go run cmd/test/network_detect.go`
   - Steps to reproduce

---

**Still stuck? Open an issue on GitHub!**
https://github.com/thelastdreamer/MultiWANBond/issues/new

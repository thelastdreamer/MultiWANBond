# MultiWANBond Performance Tuning Guide

**Complete guide for optimizing MultiWANBond performance**

**Version**: 1.1
**Last Updated**: November 2, 2025

---

## Table of Contents

- [Performance Benchmarks](#performance-benchmarks)
- [System Tuning](#system-tuning)
- [Configuration Optimization](#configuration-optimization)
- [Network Optimization](#network-optimization)
- [Monitoring Performance](#monitoring-performance)
- [Troubleshooting Performance Issues](#troubleshooting-performance-issues)

---

## Performance Benchmarks

### Baseline Performance

**Test Environment**:
- CPU: Intel i7-10700 (8 cores @ 2.9 GHz)
- RAM: 32 GB DDR4
- OS: Windows 11 / Ubuntu 22.04
- WANs: 4x 1 Gbps Ethernet

**Results**:

| Metric | Value | Notes |
|--------|-------|-------|
| **Throughput (single WAN)** | 950 Mbps | Single core, with encryption |
| **Throughput (aggregate)** | 3.5 Gbps | 4 WANs, multi-core |
| **Latency overhead** | +0.5 ms | Average additional latency |
| **CPU usage** | ~5% per 100 Mbps | With ChaCha20-Poly1305 |
| **Memory usage** | 200 MB base | +50 KB per WAN, +200 bytes per flow |
| **Max flows** | 100,000 | Configurable, depends on RAM |
| **Health check overhead** | <0.1% CPU | Per WAN, 5-second interval |

### Scalability

**Small Deployment** (1-3 WANs, <500 Mbps total):
- CPU: 2 cores
- RAM: 2 GB
- Expected performance: Near line-rate

**Medium Deployment** (4-8 WANs, <2 Gbps total):
- CPU: 4 cores
- RAM: 4 GB
- Expected performance: >90% of line-rate

**Large Deployment** (8+ WANs, >2 Gbps total):
- CPU: 8+ cores
- RAM: 8+ GB
- Expected performance: >85% of line-rate

---

## System Tuning

### Linux Kernel Tuning

**Increase UDP buffer sizes** (`/etc/sysctl.conf`):
```bash
# Increase UDP receive buffer
net.core.rmem_max = 26214400
net.core.rmem_default = 26214400

# Increase UDP send buffer
net.core.wmem_max = 26214400
net.core.wmem_default = 26214400

# Increase network device backlog
net.core.netdev_max_backlog = 5000

# Apply changes
sudo sysctl -p
```

**TCP BBR congestion control**:
```bash
# Enable TCP BBR (better for long-distance connections)
net.core.default_qdisc = fq
net.ipv4.tcp_congestion_control = bbr

# Apply
sudo sysctl -p
```

**Increase file descriptors**:
```bash
# /etc/security/limits.conf
* soft nofile 65536
* hard nofile 65536

# Verify
ulimit -n
# Should show: 65536
```

**Disable unnecessary services**:
```bash
# Disable firewalld if using iptables
sudo systemctl disable firewalld

# Disable SELinux (or set to permissive) if not needed
sudo setenforce 0
```

### Windows Tuning

**Increase UDP buffer sizes** (PowerShell as Admin):
```powershell
# Set UDP receive buffer
netsh int udp set global recvbuffsize=2621440

# Set UDP send buffer
netsh int udp set global sendbuffsize=2621440
```

**Disable Windows Defender real-time scanning** (for testing):
```powershell
Set-MpPreference -DisableRealtimeMonitoring $true
```

**Enable RSS (Receive Side Scaling)**:
```powershell
Enable-NetAdapterRss -Name "Ethernet"
```

**Increase network adapter buffers**:
- Device Manager → Network Adapters → Properties → Advanced
- Increase "Receive Buffers" to maximum
- Increase "Transmit Buffers" to maximum

### CPU Affinity

**Pin MultiWANBond to specific cores** (Linux):
```bash
# Start with taskset
taskset -c 0,1,2,3 multiwanbond --config config.json

# Or with systemd
# /etc/systemd/system/multiwanbond.service
[Service]
CPUAffinity=0-3
```

**Windows**:
```powershell
# PowerShell
$Process = Get-Process multiwanbond
$Process.ProcessorAffinity = 0x0F  # Cores 0-3
```

---

## Configuration Optimization

### Load Balancing Mode

**Performance comparison**:

| Mode | CPU Usage | Latency | Throughput | Use Case |
|------|-----------|---------|------------|----------|
| **Round-Robin** | Lowest | Medium | High | Equal WANs, max throughput |
| **Weighted** | Low | Medium | High | Unequal WANs, max throughput |
| **Least-Used** | Medium | Medium | Very High | Variable traffic |
| **Least-Latency** | Medium | Lowest | Medium | Latency-sensitive apps |
| **Per-Flow** | Low | Low | High | Maintain packet order |
| **Adaptive** | High | Low | Very High | Best overall (recommended) |

**Recommendation for Performance**: **Weighted** or **Per-Flow**

**Configuration**:
```json
{
  "routing": {
    "mode": "weighted"  // Fast and predictable
  }
}
```

### FEC (Forward Error Correction)

**Trade-off**: Recovery vs Overhead

**Optimal Settings**:
```json
{
  "fec": {
    "enabled": true,
    "data_shards": 10,     // Original packets
    "parity_shards": 2,    // Redundancy (20% overhead)
    "shard_size": 1400     // Match MTU - headers
  }
}
```

**Recommendations**:
- **Low loss (<1%)**: Use data:parity ratio of 10:1 (10% overhead)
- **Medium loss (1-5%)**: Use data:parity ratio of 10:2 (20% overhead)
- **High loss (>5%)**: Use data:parity ratio of 10:3 (30% overhead)

**Disable FEC** if packet loss is consistently 0% to reduce CPU usage.

### Health Check Interval

**Trade-off**: Faster failover vs CPU/Network overhead

**Recommendations**:

| Scenario | Interval | Timeout | Retry Count |
|----------|----------|---------|-------------|
| **Low latency WANs** (<20ms) | 1000ms | 500ms | 3 |
| **Standard WANs** (20-50ms) | 5000ms | 3000ms | 3 |
| **High latency WANs** (>50ms) | 10000ms | 5000ms | 3 |

**Configuration**:
```json
{
  "health": {
    "check_interval_ms": 5000,
    "timeout_ms": 3000,
    "retry_count": 3,
    "check_hosts": ["8.8.8.8", "1.1.1.1"]
  }
}
```

### Encryption

**Performance Impact**:

| Algorithm | CPU Usage (per 100 Mbps) | Hardware Accel | Recommendation |
|-----------|--------------------------|----------------|----------------|
| **ChaCha20-Poly1305** | ~5% | No | ARM, RISC-V |
| **AES-256-GCM** (AES-NI) | ~2% | Yes (x86) | Intel/AMD with AES-NI |
| **AES-256-GCM** (no AES-NI) | ~15% | No | Avoid |

**Check for AES-NI support** (Linux):
```bash
grep aes /proc/cpuinfo
# If output, AES-NI is supported
```

**Optimal Configuration**:
```json
{
  "security": {
    "encryption_enabled": true,
    "encryption_type": "aes-256-gcm"  // If AES-NI available
    // OR
    "encryption_type": "chacha20poly1305"  // If no AES-NI
  }
}
```

**Disable Encryption** (for max performance, not recommended):
```json
{
  "security": {
    "encryption_enabled": false
  }
}
```

### Packet Reordering

**Trade-off**: In-order delivery vs Latency

**Configuration**:
```json
{
  "reordering": {
    "enabled": true,
    "timeout_ms": 100,  // Lower = less latency, more out-of-order
    "buffer_size": 1000 // Higher = more reordering tolerance
  }
}
```

**Recommendations**:
- **Low latency apps** (gaming, VoIP): 50ms timeout
- **Standard**: 100ms timeout (default)
- **High latency WANs**: 500ms timeout

**Disable Reordering** for max throughput (some apps tolerate out-of-order):
```json
{
  "reordering": {
    "enabled": false
  }
}
```

### DPI (Deep Packet Inspection)

**Performance Impact**: ~1% CPU for classification

**Optimization**:
```json
{
  "dpi": {
    "enabled": true,
    "max_flows": 10000,        // Reduce if memory constrained
    "flow_timeout_sec": 60,    // Cleanup idle flows faster
    "classify_first_n_packets": 3  // Only inspect first 3 packets
  }
}
```

**Disable DPI** if not using policy routing:
```json
{
  "dpi": {
    "enabled": false
  }
}
```

---

## Network Optimization

### MTU (Maximum Transmission Unit)

**Optimal MTU**: Largest size without fragmentation

**Calculate optimal MTU**:
```bash
# Linux/macOS
ping -M do -s 1472 8.8.8.8  # 1472 + 28 (headers) = 1500
# If successful, try larger:
ping -M do -s 1473 8.8.8.8
# If fails, reduce by 1 until successful

# Windows
ping -f -l 1472 8.8.8.8
```

**Set MTU** (Linux):
```bash
# Temporary
sudo ip link set dev eth0 mtu 1500

# Permanent (Ubuntu/Debian)
# /etc/network/interfaces
auto eth0
iface eth0 inet dhcp
    mtu 1500
```

**Set MTU** (Windows):
```powershell
netsh interface ipv4 set subinterface "Ethernet" mtu=1500 store=persistent
```

**Jumbo Frames** (if all WANs support):
```bash
# Set MTU to 9000 (jumbo frames)
sudo ip link set dev eth0 mtu 9000
```

**Benefits**: Fewer packets to process, lower CPU usage

### TX/RX Queue Sizes

**Increase network card queue sizes** (Linux):
```bash
# Check current size
ethtool -g eth0

# Increase RX queue
sudo ethtool -G eth0 rx 4096

# Increase TX queue
sudo ethtool -G eth0 tx 4096
```

### Interrupt Coalescing

**Reduce interrupts** (trades latency for throughput):
```bash
# Increase coalescing time
sudo ethtool -C eth0 rx-usecs 100
sudo ethtool -C eth0 tx-usecs 100
```

**Note**: Higher values = lower CPU, higher latency

### RSS (Receive Side Scaling)

**Enable RSS** to distribute packet processing across multiple cores (Linux):
```bash
# Check if supported
ethtool -l eth0

# Enable all queues
sudo ethtool -L eth0 combined 4
```

---

## Monitoring Performance

### Real-Time Monitoring

**CPU Usage**:
```bash
# Linux
top -p $(pidof multiwanbond)

# Windows
Get-Process multiwanbond | Format-Table -Property CPU, WorkingSet
```

**Network Usage**:
```bash
# Linux (iftop)
sudo iftop -i eth0

# Linux (nload)
sudo nload eth0

# Windows (Performance Monitor)
perfmon
# Add counters: Network Interface → Bytes Total/sec
```

**Memory Usage**:
```bash
# Linux
ps aux | grep multiwanbond

# Windows
Get-Process multiwanbond | Format-Table -Property WS, PM
```

### Performance Metrics

**Key metrics to monitor**:

1. **Throughput**: Mbps per WAN
2. **CPU Usage**: % per core
3. **Memory**: MB used
4. **Latency**: ms overhead
5. **Packet Loss**: % per WAN
6. **Goroutines**: Count (Go runtime metric)

**Collect via Prometheus** (future feature):
```bash
curl http://localhost:8080/metrics
```

### Profiling

**CPU profiling** (during development):
```bash
# Start with profiling enabled
multiwanbond --config config.json --profile

# Capture profile (30 seconds)
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Analyze
(pprof) top10
(pprof) list <function>
```

**Memory profiling**:
```bash
go tool pprof http://localhost:6060/debug/pprof/heap
```

---

## Troubleshooting Performance Issues

### Symptom: Low Throughput

**Possible Causes**:
1. **CPU bottleneck**: High CPU usage (>80%)
2. **WAN bottleneck**: Individual WAN saturated
3. **Encryption overhead**: Heavy encryption CPU usage
4. **MTU issues**: Packet fragmentation
5. **Buffer sizes**: Small UDP buffers

**Solutions**:
1. **CPU**: Upgrade hardware, reduce encryption overhead, disable FEC
2. **WAN**: Add more WANs, increase WAN bandwidth
3. **Encryption**: Use AES-GCM with AES-NI, or disable encryption (testing only)
4. **MTU**: Optimize MTU (see Network Optimization)
5. **Buffers**: Increase UDP buffer sizes (see System Tuning)

**Debugging**:
```bash
# Check CPU usage
top -p $(pidof multiwanbond)

# Check network saturation
iftop -i eth0

# Profile CPU
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

### Symptom: High Latency

**Possible Causes**:
1. **Reordering buffer**: High timeout
2. **FEC overhead**: Encoding/decoding delay
3. **WAN latency**: Inherent WAN delay
4. **CPU scheduling**: Process not getting CPU time

**Solutions**:
1. **Reordering**: Reduce timeout to 50ms
2. **FEC**: Reduce parity shards or disable
3. **WAN**: Use lower-latency WANs, optimize routing
4. **CPU**: Increase process priority

**Debugging**:
```bash
# Measure end-to-end latency
ping <destination>

# Check per-WAN latency in Web UI
# Dashboard → WAN cards

# Trace route
traceroute <destination>
```

### Symptom: High CPU Usage

**Possible Causes**:
1. **Encryption**: Heavy CPU usage for encryption
2. **DPI**: Classifying too many flows
3. **Health checks**: Too frequent checks
4. **Adaptive load balancing**: Complex calculations

**Solutions**:
1. **Encryption**: Use hardware-accelerated AES-GCM
2. **DPI**: Reduce max_flows, disable if not needed
3. **Health checks**: Increase interval to 10 seconds
4. **Load balancing**: Use "weighted" instead of "adaptive"

**Debugging**:
```bash
# Profile CPU hotspots
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
(pprof) top10
```

### Symptom: High Memory Usage

**Possible Causes**:
1. **Too many flows**: DPI tracking millions of flows
2. **Metrics retention**: Long retention period
3. **Reordering buffer**: Large buffer size
4. **Memory leak**: Bug in code

**Solutions**:
1. **Flows**: Reduce max_flows, reduce flow_timeout
2. **Metrics**: Reduce retention to 24 hours
3. **Reordering**: Reduce buffer_size
4. **Memory leak**: Report bug, restart service

**Debugging**:
```bash
# Check memory usage
ps aux | grep multiwanbond

# Profile memory
go tool pprof http://localhost:6060/debug/pprof/heap
(pprof) top10
```

### Symptom: Packet Loss

**Possible Causes**:
1. **WAN quality**: Inherent packet loss on WAN
2. **Buffer overflow**: UDP buffers too small
3. **CPU overload**: Dropped packets due to processing delay

**Solutions**:
1. **WAN**: Contact ISP, use better WAN, enable FEC
2. **Buffers**: Increase UDP buffer sizes
3. **CPU**: Upgrade hardware, reduce load

**Debugging**:
```bash
# Check per-WAN packet loss in Web UI
# Dashboard → WAN cards → Packet Loss

# Check system packet drops
netstat -s | grep -i drop

# Check interface errors
ethtool -S eth0 | grep -i error
```

---

## Best Practices

### Production Configuration

**Balanced Performance & Reliability**:
```json
{
  "routing": {
    "mode": "adaptive"
  },
  "health": {
    "check_interval_ms": 5000,
    "timeout_ms": 3000,
    "retry_count": 3
  },
  "security": {
    "encryption_enabled": true,
    "encryption_type": "aes-256-gcm"
  },
  "fec": {
    "enabled": true,
    "data_shards": 10,
    "parity_shards": 2
  },
  "dpi": {
    "enabled": true,
    "max_flows": 100000
  },
  "reordering": {
    "enabled": true,
    "timeout_ms": 100
  }
}
```

**Maximum Performance (Testing)**:
```json
{
  "routing": {
    "mode": "weighted"  // Faster than adaptive
  },
  "health": {
    "check_interval_ms": 10000  // Less frequent
  },
  "security": {
    "encryption_enabled": false  // No encryption overhead
  },
  "fec": {
    "enabled": false  // No FEC overhead
  },
  "dpi": {
    "enabled": false  // No DPI overhead
  },
  "reordering": {
    "enabled": false  // No reordering overhead
  }
}
```

### Capacity Planning

**Formula for required CPU cores**:
```
Cores = (Total_Throughput_Gbps × 0.5) + 2
```

**Examples**:
- 1 Gbps total: 2.5 cores → 4 cores recommended
- 5 Gbps total: 4.5 cores → 8 cores recommended
- 10 Gbps total: 7 cores → 8-16 cores recommended

**Formula for required RAM**:
```
RAM_MB = 200 + (WANs × 50) + (Max_Flows × 0.0002)
```

**Examples**:
- 3 WANs, 10K flows: 200 + 150 + 2 = 352 MB
- 10 WANs, 100K flows: 200 + 500 + 20 = 720 MB

---

## Hardware Recommendations

### CPU

**Priority**: Clock speed > Core count (for most workloads)

**Minimum**: Intel i3 or AMD Ryzen 3 (2 cores)

**Recommended**: Intel i5/i7 or AMD Ryzen 5/7 (4-8 cores)

**High Performance**: Intel Xeon or AMD EPYC (8+ cores)

**Features to Look For**:
- AES-NI (Intel/AMD): Hardware-accelerated AES encryption
- AVX2: Faster FEC encoding
- High single-thread performance: Faster packet processing

### Network Interface Cards

**Minimum**: 1 Gbps Ethernet

**Recommended**: 10 Gbps Ethernet (for high-throughput deployments)

**Features to Look For**:
- RSS (Receive Side Scaling): Multi-queue support
- Hardware offloading: TSO, GSO, LRO
- Large MTU support: Jumbo frames (9000 MTU)

**Recommended Brands**:
- Intel (best Linux support)
- Mellanox/NVIDIA
- Broadcom

### Storage

**Not critical**: MultiWANBond uses minimal disk I/O

**Recommendation**: SSD for OS and logs (faster boot, log writes)

---

## Additional Resources

- [ARCHITECTURE.md](ARCHITECTURE.md) - Performance architecture details
- [DEPLOYMENT.md](DEPLOYMENT.md) - Production deployment
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - General troubleshooting

---

**Last Updated**: November 2, 2025
**Version**: 1.1
**MultiWANBond Version**: 1.1

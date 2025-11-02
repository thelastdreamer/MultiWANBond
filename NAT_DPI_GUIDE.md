# MultiWANBond NAT & DPI Technical Guide

**Complete technical reference for NAT Traversal and Deep Packet Inspection**

**Version**: 1.2
**Last Updated**: November 3, 2025

---

## Table of Contents

- [Overview](#overview)
- [NAT Traversal](#nat-traversal)
- [Deep Packet Inspection (DPI)](#deep-packet-inspection-dpi)
- [API Reference](#api-reference)
- [Integration Guide](#integration-guide)
- [Use Cases](#use-cases)
- [Troubleshooting](#troubleshooting)
- [Performance Considerations](#performance-considerations)

---

## Overview

MultiWANBond provides two powerful network intelligence features:

1. **NAT Traversal**: Automatic detection and handling of NAT configurations
2. **Deep Packet Inspection**: Real-time traffic classification and monitoring

Both features are **enabled by default** and provide real-time data through the Web UI and API.

---

## NAT Traversal

### What is NAT Traversal?

NAT (Network Address Translation) Traversal allows MultiWANBond to:
- Detect your NAT configuration automatically
- Determine if direct peer-to-peer connections are possible
- Identify CGNAT (Carrier-Grade NAT) environments
- Choose optimal connection strategies

### NAT Detection Process

**Initialization** (on startup):
1. Bind to local UDP port
2. Contact STUN servers (stun.l.google.com:19302)
3. Discover public address and port mapping
4. Perform NAT type classification
5. Test for CGNAT presence

**Update Cycle**:
- Refresh every 5 minutes (configurable)
- Re-detect on network changes
- Update public address mapping

### NAT Types Explained

#### 1. **Open (No NAT)**
```
[ Client ] ←→ [ Internet ]
```
- **Description**: Direct public IP, no NAT
- **P2P Capability**: ✓✓✓ Excellent
- **Use Cases**: Servers, datacenters
- **Connection Method**: Direct

#### 2. **Full Cone NAT**
```
[ Client ] ←→ [ NAT (Full Cone) ] ←→ [ Internet ]
         Internal: 192.168.1.100
         Public:   203.0.113.45:12345
```
- **Description**: One-to-one port mapping, accepts from any external address
- **P2P Capability**: ✓✓ Very Good
- **Use Cases**: Home routers (common)
- **Connection Method**: Direct after initial outbound packet

**Behavior**:
- Maps internal port → fixed public port
- Any external host can send to public port
- Simple hole-punching works

#### 3. **Restricted Cone NAT**
```
[ Client ] ←→ [ NAT (Restricted) ] ←→ [ Internet ]

Only accepts packets from IPs the client has sent to
```
- **Description**: Filters by IP, allows any port from known IPs
- **P2P Capability**: ✓ Good (requires simultaneous open)
- **Use Cases**: Corporate firewalls
- **Connection Method**: Simultaneous hole-punching

**Behavior**:
- Must send packet to destination IP first
- Destination can use any port to reply
- Requires coordinated connection timing

#### 4. **Port-Restricted Cone NAT**
```
[ Client ] ←→ [ NAT (Port-Restricted) ] ←→ [ Internet ]

Only accepts packets from (IP:Port) combinations the client has sent to
```
- **Description**: Filters by IP and Port
- **P2P Capability**: ~ Fair (difficult hole-punching)
- **Use Cases**: Stricter firewalls
- **Connection Method**: Precise hole-punching with signaling

**Behavior**:
- Must send packet to exact (IP, Port) first
- Only that (IP, Port) can reply
- Requires precise coordination

#### 5. **Symmetric NAT**
```
[ Client ] ←→ [ NAT (Symmetric) ] ←→ [ Internet ]

Different public port for each destination!
192.168.1.100:5000 → 203.0.113.45:12345 (to Server A)
192.168.1.100:5000 → 203.0.113.45:23456 (to Server B)
```
- **Description**: Creates unique mapping per destination
- **P2P Capability**: ✗ Poor (relay required)
- **Use Cases**: ISP-grade equipment, high-security networks
- **Connection Method**: Relay server (TURN)

**Behavior**:
- Source port changes based on destination
- Makes hole-punching nearly impossible
- Requires relay servers for P2P

#### 6. **Blocked**
```
[ Client ] ←→ [ Firewall ] ✗ [ Internet ]

UDP completely blocked
```
- **Description**: UDP traffic filtered
- **P2P Capability**: ✗ None
- **Use Cases**: Restrictive networks
- **Connection Method**: TCP fallback or VPN

### CGNAT Detection

**What is CGNAT?**
- Carrier-Grade NAT (CGNAT) is a second layer of NAT
- ISP places multiple customers behind shared public IPs
- Common in mobile networks, oversubscribed ISPs

```
[ Client ] ←→ [ Home Router NAT ] ←→ [ ISP CGNAT ] ←→ [ Internet ]
  10.0.0.x         100.64.0.x            203.0.113.45
```

**Detection Methods**:
1. **RFC 6598 Range Check**: Public IP in 100.64.0.0/10 range
2. **Port Mapping Persistence**: Mappings change frequently
3. **Multiple Public IPs**: STUN reports different IPs over time

**CGNAT Indicators**:
- Public address starts with `100.64.x.x`
- Port mappings expire quickly (< 30 seconds)
- High port randomization
- Multiple clients share same public IP

**Impact**:
- ✗ Direct P2P connections unlikely
- ⚠️ Relay servers required
- ⚠️ Port forwarding ineffective
- ⚠️ May affect gaming/VoIP quality

### API: `/api/nat`

**Endpoint**: `GET /api/nat`

**Response**:
```json
{
  "success": true,
  "data": {
    "local_addr": "0.0.0.0:64293",
    "public_addr": "203.0.113.45:64293",
    "nat_type": "Full Cone NAT",
    "cgnat_detected": false,
    "can_direct_connect": true,
    "needs_relay": false,
    "relay_available": false
  }
}
```

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `local_addr` | string | Local bind address (IP:Port) |
| `public_addr` | string | Public address as seen externally |
| `nat_type` | string | Detected NAT type (see above) |
| `cgnat_detected` | boolean | True if behind CGNAT |
| `can_direct_connect` | boolean | True if P2P possible without relay |
| `needs_relay` | boolean | True if relay server required |
| `relay_available` | boolean | True if relay is configured and online |

**Update Frequency**: Every 5 minutes or on network change

**Use Cases**:
1. **Connection Strategy Selection**: Choose direct vs relay
2. **User Notifications**: Warn about CGNAT limitations
3. **Monitoring Dashboards**: Display network topology
4. **Debugging**: Understand connection failures

---

## Deep Packet Inspection (DPI)

### What is DPI?

Deep Packet Inspection analyzes network traffic to identify:
- Application protocols (HTTP, DNS, SSH, etc.)
- Specific applications (YouTube, Netflix, Zoom, etc.)
- Traffic categories (Streaming, Gaming, VoIP, etc.)

### Classification Process

**Packet Analysis**:
1. Capture packet metadata (IPs, ports, protocol)
2. Examine first N bytes of payload (configurable, default 512)
3. Match against signature database (40+ protocols)
4. Assign protocol and category
5. Track flow statistics

**Flow Tracking**:
- Each unique 5-tuple (SrcIP, DstIP, SrcPort, DstPort, Protocol) = 1 flow
- Flows timeout after 5 minutes of inactivity (configurable)
- Statistics: packets, bytes, duration, first/last seen

### Supported Protocols (40+)

#### Web Protocols
- HTTP, HTTPS, HTTP/2, HTTP/3
- WebSocket

#### Streaming Services
- YouTube
- Netflix
- Twitch
- Spotify
- Apple Music

#### Social Media
- Facebook
- Instagram
- Twitter (X)
- TikTok
- WhatsApp

#### Gaming
- Steam
- Epic Games
- Minecraft
- League of Legends
- Fortnite

#### Communication
- Zoom
- Microsoft Teams
- Skype
- Discord
- Slack

#### File Transfer
- FTP, SFTP, SCP
- BitTorrent
- Dropbox

#### Email
- SMTP, IMAP, POP3

#### DNS
- DNS (standard)
- DNS over HTTPS (DoH)
- DNS over TLS (DoT)

#### VPN Protocols
- OpenVPN
- WireGuard
- IPSec
- L2TP

#### Other
- SSH, Telnet, RDP, VNC
- NTP, DHCP

### Traffic Categories

| Category | Description | QoS Priority | Examples |
|----------|-------------|--------------|----------|
| **Real-Time** | Voice, video conferencing | Highest | Zoom, Teams, VoIP |
| **Interactive** | Gaming, remote desktop | High | Minecraft, RDP, SSH |
| **Streaming** | Video/audio streaming | Medium | YouTube, Netflix, Spotify |
| **Bulk** | Large file transfers | Low | FTP, backups |
| **Background** | System updates, P2P | Lowest | Torrents, system updates |

### Detection Methods

#### 1. **Port-Based Detection**
```
Port 80 → HTTP
Port 443 → HTTPS
Port 22 → SSH
```
- Fast, low CPU usage
- Unreliable (port forwarding, tunneling)
- Used as initial hint

#### 2. **Signature Matching**
```
Payload starts with "GET / HTTP/1.1" → HTTP
Payload contains "\x16\x03\x01" → TLS/SSL
```
- Examine first N bytes of payload
- Pattern matching (literal or regex)
- High accuracy for unencrypted traffic

#### 3. **Statistical Analysis**
```
Packet size distribution, timing patterns
```
- For encrypted traffic
- Machine learning-based (future)
- VPN/Tor detection

#### 4. **Domain/SNI Analysis**
```
TLS SNI extension: "www.youtube.com" → YouTube
DNS query: "www.netflix.com" → Netflix
```
- Works with encrypted HTTPS
- Requires DNS inspection or SNI extraction
- Bypassed by encrypted DNS (DoH/DoT)

### Configuration

**Default Settings**:
```go
DPIConfig{
    Enabled:             true,
    MaxFlows:            100000,
    FlowTimeout:         5 * time.Minute,
    SamplingRate:        1.0,  // Inspect 100% of packets
    MaxPayloadInspect:   512,  // Bytes
    EnableStatistics:    true,
    EnablePolicies:      true,
}
```

**Tuning for Performance**:
- High-traffic servers: Reduce `SamplingRate` to 0.1 (10%)
- Memory-constrained: Reduce `MaxFlows` to 10000
- Encrypted-only networks: Reduce `MaxPayloadInspect` to 64

### API: `/api/flows`

**Endpoint**: `GET /api/flows`

**Response**:
```json
{
  "success": true,
  "data": [
    {
      "src_ip": "192.168.1.100",
      "dst_ip": "142.250.185.46",
      "src_port": 52341,
      "dst_port": 443,
      "protocol": "HTTPS",
      "application": "YouTube",
      "category": "Streaming",
      "wan_id": 1,
      "packets": 1523,
      "bytes": 2147483,
      "duration_ms": 45200,
      "first_seen": "2025-11-03T01:30:15Z",
      "last_seen": "2025-11-03T01:30:60Z"
    },
    {
      "src_ip": "192.168.1.100",
      "dst_ip": "8.8.8.8",
      "src_port": 52342,
      "dst_port": 53,
      "protocol": "DNS",
      "application": "DNS",
      "category": "Infrastructure",
      "wan_id": 2,
      "packets": 2,
      "bytes": 128,
      "duration_ms": 45,
      "first_seen": "2025-11-03T01:30:15Z",
      "last_seen": "2025-11-03T01:30:15Z"
    }
  ]
}
```

**Fields**:

| Field | Type | Description |
|-------|------|-------------|
| `src_ip` | string | Source IP address |
| `dst_ip` | string | Destination IP address |
| `src_port` | uint16 | Source port |
| `dst_port` | uint16 | Destination port |
| `protocol` | string | Transport/application protocol |
| `application` | string | Detected application name |
| `category` | string | Traffic category |
| `wan_id` | uint8 | WAN interface carrying this flow (0 = unassigned) |
| `packets` | uint64 | Total packets in flow |
| `bytes` | uint64 | Total bytes in flow |
| `duration_ms` | int64 | Flow duration in milliseconds |
| `first_seen` | timestamp | When flow was first observed |
| `last_seen` | timestamp | Most recent packet in flow |

**Update Frequency**: Real-time (updates as packets arrive)

**Filtering** (future):
- By protocol: `/api/flows?protocol=YouTube`
- By category: `/api/flows?category=Streaming`
- By WAN: `/api/flows?wan_id=1`
- Top N: `/api/flows?top=10`

---

## Integration Guide

### Polling NAT Status

**JavaScript Example**:
```javascript
async function checkNATStatus() {
    const response = await fetch('/api/nat');
    const result = await response.json();

    if (!result.success) {
        console.error('Failed to fetch NAT info');
        return;
    }

    const nat = result.data;

    // Display NAT type
    document.getElementById('nat-type').innerText = nat.nat_type;

    // Display public address
    document.getElementById('public-addr').innerText = nat.public_addr;

    // Show CGNAT warning
    if (nat.cgnat_detected) {
        showWarning('You are behind CGNAT - P2P connections may not work');
    }

    // Check connection capability
    if (!nat.can_direct_connect && !nat.relay_available) {
        showError('No relay available - connections will fail');
    }
}

// Poll every 30 seconds
setInterval(checkNATStatus, 30000);
```

### Monitoring Active Flows

**JavaScript Example**:
```javascript
async function fetchActiveFlows() {
    const response = await fetch('/api/flows');
    const result = await response.json();

    if (!result.success) {
        console.error('Failed to fetch flows');
        return;
    }

    const flows = result.data;

    // Group by category
    const byCategory = {};
    flows.forEach(flow => {
        if (!byCategory[flow.category]) {
            byCategory[flow.category] = [];
        }
        byCategory[flow.category].push(flow);
    });

    // Display top categories
    console.log('Streaming:', byCategory['Streaming']?.length || 0);
    console.log('Gaming:', byCategory['Interactive']?.length || 0);
    console.log('Web:', byCategory['Web']?.length || 0);

    // Find bandwidth hogs
    const topFlows = flows
        .sort((a, b) => b.bytes - a.bytes)
        .slice(0, 10);

    displayTopFlows(topFlows);
}

// Update every 2 seconds
setInterval(fetchActiveFlows, 2000);
```

### WebSocket Real-Time Updates

**JavaScript Example**:
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
    const message = JSON.parse(event.data);

    switch(message.type) {
        case 'flow_created':
            console.log('New flow:', message.data);
            updateFlowTable(message.data);
            break;

        case 'flow_closed':
            console.log('Flow ended:', message.data);
            removeFlowFromTable(message.data);
            break;

        case 'system_alert':
            if (message.data.type === 'cgnat_detected') {
                showCGNATWarning();
            }
            break;
    }
};
```

---

## Use Cases

### 1. Connection Strategy Selection

```go
func selectConnectionStrategy(natInfo *NATInfo) ConnectionStrategy {
    if natInfo.CanDirectConnect {
        // NAT type allows direct P2P
        return StrategyDirect
    }

    if natInfo.RelayAvailable {
        // Fall back to relay server
        return StrategyRelay
    }

    // No viable connection method
    return StrategyFailed
}
```

### 2. Bandwidth Monitoring by Application

```javascript
async function getBandwidthByApp() {
    const flows = await fetchFlows();
    const stats = {};

    flows.forEach(flow => {
        if (!stats[flow.application]) {
            stats[flow.application] = {
                bytes: 0,
                flows: 0
            };
        }
        stats[flow.application].bytes += flow.bytes;
        stats[flow.application].flows += 1;
    });

    // Sort by bytes
    return Object.entries(stats)
        .sort(([,a], [,b]) => b.bytes - a.bytes)
        .slice(0, 10);
}
```

### 3. Automatic Policy Routing

```go
// Route video streaming through high-bandwidth WAN
func routeByProtocol(flow *dpi.Flow, wans []*WAN) *WAN {
    switch flow.Protocol {
    case dpi.ProtocolYouTube, dpi.ProtocolNetflix:
        // Route through fiber (highest bandwidth)
        return findWANByType(wans, "fiber")

    case dpi.ProtocolZoom, dpi.ProtocolTeams:
        // Route through most stable (lowest latency/jitter)
        return findMostStableWAN(wans)

    default:
        // Load balance
        return selectByWeight(wans)
    }
}
```

### 4. CGNAT Detection Alert

```javascript
function checkCGNAT() {
    fetch('/api/nat')
        .then(r => r.json())
        .then(result => {
            if (result.data.cgnat_detected) {
                showNotification({
                    title: 'CGNAT Detected',
                    message: 'Your ISP uses Carrier-Grade NAT. ' +
                             'Port forwarding will not work. ' +
                             'Consider VPN or relay server.',
                    severity: 'warning',
                    actions: [
                        { label: 'Learn More', url: '/docs/cgnat' },
                        { label: 'Configure Relay', url: '/config' }
                    ]
                });
            }
        });
}
```

---

## Troubleshooting

### NAT Detection Issues

#### Problem: NAT type shows "Unknown"

**Causes**:
- STUN server unreachable
- UDP port blocked
- Network not ready

**Solution**:
```bash
# Check STUN connectivity
nc -u stun.l.google.com 19302

# Check firewall
sudo iptables -L -n | grep 19302

# Manually refresh detection
curl -X POST http://localhost:8080/api/nat/refresh
```

#### Problem: Public address is IPv6 but expected IPv4

**Causes**:
- Dual-stack network
- IPv6 preferred by system

**Solution**:
```json
// In config.json
{
  "nat": {
    "prefer_ipv4": true,
    "stun_servers": [
      "stun.l.google.com:19302",
      "stun1.l.google.com:19302"
    ]
  }
}
```

#### Problem: CGNAT detected but ISP claims no CGNAT

**Causes**:
- Public IP in RFC 6598 range (100.64.0.0/10)
- ISP using private range internally

**Solution**:
Check with ISP, may need to disable CGNAT detection:
```json
{
  "nat": {
    "disable_cgnat_detection": true
  }
}
```

### DPI Classification Issues

#### Problem: Encrypted traffic shows as "Unknown"

**Expected Behavior**: DPI can't see inside encrypted payloads

**Solutions**:
1. Enable DNS inspection (identifies domains)
2. Use TLS SNI extraction (see server names)
3. Statistical classification (future feature)

#### Problem: High memory usage

**Causes**:
- Too many concurrent flows
- Large flow timeout

**Solution**:
```json
{
  "dpi": {
    "max_flows": 10000,       // Reduce from default 100000
    "flow_timeout": "2m",     // Reduce from default 5m
    "sampling_rate": 0.1      // Inspect only 10% of packets
  }
}
```

#### Problem: False positives in protocol detection

**Causes**:
- Port-based detection alone
- Ambiguous signatures

**Solution**:
Enable deeper payload inspection:
```json
{
  "dpi": {
    "max_payload_inspect": 1024,  // Increase from 512
    "use_port_hints": false        // Disable port-based hints
  }
}
```

---

## Performance Considerations

### NAT Traversal

**CPU Usage**: Minimal
- STUN requests: ~1ms every 5 minutes
- Type detection: ~10ms on startup

**Network Usage**: Negligible
- STUN request: ~200 bytes every 5 minutes
- STUN response: ~200 bytes

**Memory Usage**: Negligible
- ~1 KB per NAT manager instance

### DPI Classification

**CPU Usage**: Moderate to High
- Per-packet overhead: 0.1-1.0 µs (depends on payload size)
- 1 Gbps traffic: ~10-20% CPU usage (1-2 cores)

**Memory Usage**: Moderate
- ~500 bytes per active flow
- 100,000 flows: ~50 MB
- Flow table overhead: ~10 MB

**Optimization Tips**:

1. **Sampling** (for high traffic):
```json
{
  "dpi": {
    "sampling_rate": 0.1  // Inspect 10% of packets
  }
}
```
Reduces CPU by 90%, maintains ~85% accuracy

2. **Reduce Payload Inspection**:
```json
{
  "dpi": {
    "max_payload_inspect": 128  // From 512 bytes
  }
}
```
Reduces CPU by 50%, maintains ~90% accuracy

3. **Shorter Flow Timeout**:
```json
{
  "dpi": {
    "flow_timeout": "2m"  // From 5 minutes
  }
}
```
Reduces memory by 60%, may miss long-lived flows

4. **Limit Max Flows**:
```json
{
  "dpi": {
    "max_flows": 10000  // From 100,000
  }
}
```
Caps memory usage, oldest flows evicted when limit reached

---

## API Summary

| Endpoint | Method | Description | Update Frequency |
|----------|--------|-------------|------------------|
| `/api/nat` | GET | NAT traversal information | Every 5 minutes |
| `/api/nat/refresh` | POST | Force NAT re-detection | On-demand |
| `/api/flows` | GET | Active network flows | Real-time |
| `/api/flows/:id` | GET | Single flow details | Real-time |
| `/api/dpi/stats` | GET | DPI statistics | Every 1 second |
| `/api/dpi/protocols` | GET | Protocol breakdown | Every 1 second |

**WebSocket Events**:
- `flow_created`: New flow detected
- `flow_closed`: Flow ended
- `nat_updated`: NAT information changed
- `cgnat_detected`: CGNAT detected

---

## Further Reading

- [RFC 5389](https://tools.ietf.org/html/rfc5389) - STUN Protocol
- [RFC 6598](https://tools.ietf.org/html/rfc6598) - CGNAT Address Space
- [RFC 5766](https://tools.ietf.org/html/rfc5766) - TURN Protocol
- [DPI on Wikipedia](https://en.wikipedia.org/wiki/Deep_packet_inspection)
- [NAT Traversal Techniques](https://tools.ietf.org/html/rfc5128)

---

**Version**: 1.2
**Last Updated**: November 3, 2025
**MultiWANBond Version**: 1.2

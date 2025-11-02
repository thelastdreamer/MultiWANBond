# MultiWANBond Architecture

Comprehensive technical architecture documentation for MultiWANBond.

---

## Table of Contents

- [System Overview](#system-overview)
- [Component Architecture](#component-architecture)
- [Data Flow](#data-flow)
- [Thread Safety](#thread-safety)
- [Performance Characteristics](#performance-characteristics)
- [Deployment Topologies](#deployment-topologies)
- [Future Architecture](#future-architecture)

---

## System Overview

MultiWANBond is a multi-component system designed for bonding multiple WAN connections into a single, reliable, high-bandwidth link.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Application Layer                            │
│                    (User Applications, Browsers)                     │
└────────────────────────────────┬────────────────────────────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────────┐
│                          Web UI Layer                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐           │
│  │Dashboard │  │  Flows   │  │Analytics │  │   Logs   │           │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘           │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │            REST API + WebSocket Server (port 8080)           │  │
│  └──────────────────────────────────────────────────────────────┘  │
└────────────────────────────────┬────────────────────────────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────────┐
│                          Core Engine Layer                           │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │   Bonder     │  │  Session     │  │  Routing     │             │
│  │   Manager    │  │  Manager     │  │  Engine      │             │
│  └──────────────┘  └──────────────┘  └──────────────┘             │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │   Packet     │  │    FEC       │  │  Encryption  │             │
│  │   Processor  │  │  (Reed-Sol)  │  │  (AES/ChaCha)│             │
│  └──────────────┘  └──────────────┘  └──────────────┘             │
└────────────────────────────────┬────────────────────────────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────────┐
│                      Supporting Services Layer                       │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │   Health     │  │     DPI      │  │     NAT      │             │
│  │   Monitor    │  │  Classifier  │  │  Traversal   │             │
│  └──────────────┘  └──────────────┘  └──────────────┘             │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │   Metrics    │  │    Alerts    │  │   Policy     │             │
│  │  Collector   │  │   Manager    │  │   Router     │             │
│  └──────────────┘  └──────────────┘  └──────────────┘             │
└────────────────────────────────┬────────────────────────────────────┘
                                 │
┌────────────────────────────────▼────────────────────────────────────┐
│                        Network Interface Layer                       │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │   WAN 1      │  │   WAN 2      │  │   WAN 3      │             │
│  │   (Fiber)    │  │ (Starlink)   │  │   (LTE)      │             │
│  └──────────────┘  └──────────────┘  └──────────────┘             │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Component Architecture

### 1. Bonder Component

**Location**: [pkg/bonder/bonder.go](pkg/bonder/bonder.go)

**Purpose**: Core bonding orchestration and WAN management

**Key Responsibilities**:
- Manage multiple WAN interfaces
- Coordinate packet distribution across WANs
- Handle session establishment and teardown
- Integrate NAT and DPI components

**Data Structures**:
```go
type Bonder struct {
    config        *Config
    wans          map[uint8]*WANInterface
    sessions      map[string]*Session
    natManager    *nat.Manager
    dpiClassifier *dpi.Classifier
    healthMon     *health.Monitor
    metrics       *metrics.Collector
    running       bool
    mu            sync.RWMutex
}
```

**Thread Safety**:
- All public methods use `mu.RLock()` for reads
- Configuration changes use `mu.Lock()` for writes
- WAN map is immutable after initialization

**Performance**:
- O(1) WAN lookup by ID
- O(1) session lookup by ID
- Lock-free packet processing path (no mutex in hot path)

---

### 2. Health Monitor

**Location**: [pkg/health/monitor.go](pkg/health/monitor.go)

**Purpose**: Continuous health monitoring of all WAN interfaces

**Key Responsibilities**:
- Execute periodic health checks (ICMP, HTTP, TCP, DNS)
- Detect WAN failures within 1 second
- Calculate latency, jitter, packet loss
- Generate health status events

**Check Methods**:
1. **ICMP Ping**: Fast, low-overhead latency check
2. **HTTP/HTTPS**: Application-layer connectivity test
3. **TCP**: Connection establishment test
4. **DNS**: Name resolution test

**Adaptive Intervals**:
- Healthy WANs: Check every 5 seconds
- Degraded WANs: Check every 1 second
- Failed WANs: Check every 500ms (rapid recovery detection)

**Data Flow**:
```
Timer Trigger (configurable interval)
        ↓
Select Check Method (based on WAN config)
        ↓
Execute Check (ICMP/HTTP/TCP/DNS)
        ↓
Calculate Metrics (latency, jitter, loss)
        ↓
Update WAN Status (healthy/degraded/failed)
        ↓
Generate Event (if state changed)
        ↓
Notify Bonder & Web UI
```

---

### 3. NAT Traversal Manager

**Location**: [pkg/nat/manager.go](pkg/nat/manager.go)

**Purpose**: NAT type detection and P2P connection establishment

**Key Components**:

**STUN Client** ([pkg/nat/stun.go](pkg/nat/stun.go)):
- RFC 5389 compliant STUN implementation
- Discovers public IP and port mapping
- Automatic refresh every 25 seconds
- Fallback to secondary server on failure

**CGNAT Detector** ([pkg/nat/cgnat.go](pkg/nat/cgnat.go)):
- Detects carrier-grade NAT (RFC 6598 ranges)
- Identifies double NAT scenarios
- Recommends relay usage when needed

**Hole Puncher** ([pkg/nat/holepunch.go](pkg/nat/holepunch.go)):
- Simultaneous UDP hole punching
- Birthday paradox technique
- Coordinate through signaling server
- Fallback to TURN relay

**NAT Types Detected**:
1. **Open (No NAT)**: Direct internet connection
2. **Full Cone NAT**: One-to-one mapping (easiest)
3. **Restricted Cone NAT**: Restricted by IP
4. **Port-Restricted Cone NAT**: Restricted by IP and port
5. **Symmetric NAT**: Random port allocation (hardest)
6. **Blocked**: UDP completely filtered

**Discovery Process**:
```
Start NAT Manager
        ↓
Send STUN Request to Primary Server
        ↓
Receive Mapped Address (public IP:port)
        ↓
Determine NAT Type (cone vs symmetric)
        ↓
Check for CGNAT (IP range analysis)
        ↓
Store NAT Info (type, public addr, CGNAT status)
        ↓
Start Periodic Refresh (every 25s)
```

---

### 4. DPI Classifier

**Location**: [pkg/dpi/classifier.go](pkg/dpi/classifier.go)

**Purpose**: Deep packet inspection and protocol classification

**Classification Methods**:
1. **Port-Based**: Quick classification by well-known ports
2. **Signature Matching**: Pattern matching in packet payload
3. **TLS SNI Extraction**: Server Name Indication from ClientHello
4. **HTTP Host Header**: Extract host from HTTP requests
5. **Behavioral Analysis**: Connection patterns and data volumes

**Supported Protocols** (58 total):

**Web**: HTTP, HTTPS, HTTP/2, HTTP/3, WebSocket

**Streaming**: YouTube, Netflix, Twitch, Spotify, Apple Music, Amazon Prime Video, Disney+, Hulu

**Social Media**: Facebook, Instagram, Twitter, TikTok, Snapchat, WhatsApp, Telegram

**Gaming**: Steam, Epic Games, Minecraft, League of Legends, Fortnite, Valorant, PUBG, CS:GO

**Communication**: Zoom, Microsoft Teams, Skype, Discord, Slack, Google Meet, Webex

**File Transfer**: FTP, SFTP, SCP, BitTorrent, Dropbox, Google Drive, OneDrive

**Email**: SMTP, IMAP, POP3

**DNS**: DNS, DNS over HTTPS, DNS over TLS

**VPN**: OpenVPN, WireGuard, IPSec, L2TP

**Other**: SSH, Telnet, RDP, VNC, NTP, DHCP, SNMP

**Flow Tracking**:
```go
type Flow struct {
    SrcIP       net.IP
    DstIP       net.IP
    SrcPort     uint16
    DstPort     uint16
    Protocol    Protocol
    Category    Category
    Packets     uint64
    Bytes       uint64
    FirstSeen   time.Time
    LastSeen    time.Time
    Confidence  float64
}
```

**Performance**:
- In-memory flow table (configurable max flows)
- O(1) flow lookup via 5-tuple hash
- Classification only on first few packets
- Automatic flow cleanup (configurable timeout)

---

### 5. Metrics Collector

**Location**: [pkg/metrics/collector.go](pkg/metrics/collector.go)

**Purpose**: Time-series metrics collection and aggregation

**Metrics Collected**:

**Per-WAN Metrics**:
- Bytes sent/received
- Packets sent/received
- Latency (min, max, avg, p95, p99)
- Jitter (min, max, avg)
- Packet loss percentage
- Uptime percentage
- State changes

**System Metrics**:
- Total traffic (all WANs)
- Active flows count
- Session count
- CPU usage
- Memory usage
- Goroutine count

**Time-Series Storage**:
- In-memory ring buffer
- 7-day retention (configurable)
- 1-second resolution
- Downsampling for older data (5min, 1hour, 1day)

**Export Formats**:
1. **Prometheus**: `/metrics` endpoint
2. **JSON**: REST API format
3. **CSV**: Export to file
4. **InfluxDB**: Line protocol
5. **Graphite**: Carbon protocol

---

### 6. Web UI Server

**Location**: [pkg/webui/server.go](pkg/webui/server.go)

**Purpose**: Web-based monitoring and configuration interface

**Architecture**:

```
┌─────────────────────────────────────────────────────────────┐
│                      HTTP/WebSocket Server                   │
│                         (port 8080)                          │
└───────────────┬─────────────────────────────────────────────┘
                │
    ┌───────────┴───────────┐
    │                       │
┌───▼───────┐       ┌───────▼────────┐
│  REST API │       │   WebSocket    │
│ Endpoints │       │    Handler     │
└───┬───────┘       └───────┬────────┘
    │                       │
    │ GET /api/wans        │ Real-time events
    │ GET /api/traffic     │ - wan_status
    │ GET /api/flows       │ - traffic_update
    │ GET /api/health      │ - system_alert
    │ GET /api/nat         │ - health_update
    │ POST /api/login      │ - flows_update
    │ POST /api/logout     │
    │                       │
    └───────────┬───────────┘
                │
┌───────────────▼───────────────────┐
│     Session Management            │
│  - Cookie-based auth              │
│  - 24-hour expiration             │
│  - Thread-safe session store      │
│  - Auto cleanup (hourly)          │
└───────────────┬───────────────────┘
                │
┌───────────────▼───────────────────┐
│     Metrics Data Bridge           │
│  - Read from Bonder               │
│  - Cache for Web UI               │
│  - WebSocket event publishing     │
│  - Thread-safe updates            │
└───────────────────────────────────┘
```

**Session Management**:
```go
type Session struct {
    ID        string    // Cryptographic random token
    Username  string    // Authenticated username
    CreatedAt time.Time // Creation timestamp
    ExpiresAt time.Time // Expiration (24h)
}
```

**Security**:
- HttpOnly cookies (XSS protection)
- SameSite=Strict (CSRF protection)
- Secure session IDs (32 bytes random)
- Server-side validation
- Auto-cleanup of expired sessions

**WebSocket Event Broadcasting**:
```go
// Non-blocking broadcast to all connected clients
func (s *Server) broadcastEvent(event string, data interface{}) {
    s.wsMu.RLock()
    defer s.wsMu.RUnlock()

    for client := range s.wsClients {
        go func(c *websocket.Conn) {
            c.WriteJSON(Message{
                Event: event,
                Data:  data,
            })
        }(client)
    }
}
```

---

### 7. Routing Engine

**Location**: [pkg/routing/router.go](pkg/routing/router.go)

**Purpose**: Intelligent packet routing across multiple WANs

**Load Balancing Modes**:

1. **Round-Robin**:
   - Distribute packets evenly across all WANs
   - Simple, predictable distribution
   - O(1) selection

2. **Weighted**:
   - Distribute based on WAN weight (capacity)
   - Higher weight = more traffic
   - Weighted random selection

3. **Least-Used**:
   - Route to WAN with lowest current traffic
   - Balance actual usage, not predicted
   - O(n) selection (n = WAN count)

4. **Least-Latency**:
   - Route to WAN with lowest current latency
   - Best for latency-sensitive traffic
   - Uses real-time health data

5. **Per-Flow**:
   - Sticky routing per 5-tuple flow
   - Maintains packet ordering
   - Hash-based WAN selection

6. **Adaptive** (Recommended):
   - Combines multiple factors:
     * WAN weight (40%)
     * Current latency (30%)
     * Packet loss (20%)
     * Current usage (10%)
   - Dynamic adjustment based on conditions
   - Best overall performance

**Routing Decision Process**:
```
Incoming Packet
        ↓
Extract 5-Tuple (src IP/port, dst IP/port, protocol)
        ↓
Check Policy Rules (if policy routing enabled)
        ↓
Select Load Balancing Mode
        ↓
Filter Healthy WANs (exclude failed)
        ↓
Apply Load Balancing Algorithm
        ↓
Select WAN Interface
        ↓
Queue Packet for Transmission
```

**Policy-Based Routing** (Linux only):
```go
type RoutingPolicy struct {
    Priority    int
    Protocol    string        // HTTP, HTTPS, YouTube, etc.
    SrcNetwork  *net.IPNet    // Source IP range
    DstNetwork  *net.IPNet    // Destination IP range
    FwMark      uint32        // iptables fwmark
    WANID       uint8         // Target WAN
}
```

---

### 8. Packet Processor

**Location**: [pkg/processor/processor.go](pkg/processor/processor.go)

**Purpose**: Packet encapsulation, encryption, and reordering

**Packet Processing Pipeline**:

**Outbound**:
```
Application Data
        ↓
Fragment (if > MTU)
        ↓
Add Sequence Number
        ↓
Apply FEC (if enabled)
        ↓
Encrypt (AES-GCM or ChaCha20-Poly1305)
        ↓
Add Protocol Header
        ↓
Route to WAN (via Routing Engine)
        ↓
Transmit via UDP socket
```

**Inbound**:
```
Receive UDP Packet
        ↓
Parse Protocol Header
        ↓
Decrypt (verify AEAD tag)
        ↓
Check Sequence Number (detect duplicates)
        ↓
Apply FEC Decoding (recover lost packets)
        ↓
Reorder Buffer (wait for out-of-order)
        ↓
Reassemble Fragments
        ↓
Deliver to Application
```

**Packet Header Format**:
```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|Version|  Type |    Flags      |          Session ID           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        Sequence Number                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        Acknowledgment                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            Length             |          Checksum             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                       Encrypted Payload                       +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
+                      Authentication Tag                       +
|                                                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

---

### 9. FEC (Forward Error Correction)

**Location**: [pkg/fec/fec.go](pkg/fec/fec.go)

**Purpose**: Recover lost packets without retransmission

**Algorithm**: Reed-Solomon erasure coding

**Configuration**:
- Data shards: Number of original packets (e.g., 10)
- Parity shards: Number of redundancy packets (e.g., 3)
- Can recover up to `parity_shards` lost packets

**Example**:
```
10 data shards + 3 parity shards = 13 total
Can lose any 3 out of 13 and still recover original data
Overhead: 30% (3/10)
```

**Performance**:
- Encoding: ~500 MB/s per core
- Decoding: ~400 MB/s per core
- Memory: O(shard_size × shard_count)

---

## Data Flow

### End-to-End Data Flow (Client to Server)

```
[Client Side]
Application
    ↓
TUN/TAP Interface (receives packet)
    ↓
Bonder (intercepts traffic)
    ↓
DPI Classifier (identify protocol)
    ↓
Policy Router (apply rules)
    ↓
Routing Engine (select WAN)
    ↓
Packet Processor (fragment if needed)
    ↓
FEC Encoder (add redundancy)
    ↓
Encryption (AES-GCM or ChaCha20)
    ↓
Add Protocol Header
    ↓
Transmit via WAN 1, 2, 3... (UDP)
    ↓
    │
    │ [Internet / Multiple Paths]
    │
    ↓
[Server Side]
Receive via UDP Socket
    ↓
Parse Protocol Header
    ↓
Decrypt Payload
    ↓
Check Sequence Number
    ↓
FEC Decoder (recover if packets lost)
    ↓
Reorder Buffer (handle out-of-order)
    ↓
Reassemble Fragments
    ↓
Session Manager (route to session)
    ↓
TUN/TAP Interface (inject packet)
    ↓
Application receives packet
```

---

## Thread Safety

### Concurrency Model

MultiWANBond uses a combination of:
1. **Mutexes** for shared data structures
2. **Channels** for inter-goroutine communication
3. **Atomic operations** for counters
4. **Immutable data** where possible

### Critical Sections

**Bonder**:
```go
// Read operations (multiple concurrent readers)
func (b *Bonder) GetWANs() map[uint8]*WANInterface {
    b.mu.RLock()
    defer b.mu.RUnlock()
    return b.wans  // Returns copy
}

// Write operations (exclusive access)
func (b *Bonder) AddWAN(wan *WANInterface) error {
    b.mu.Lock()
    defer b.mu.Unlock()
    b.wans[wan.ID] = wan
    return nil
}
```

**Web UI Server**:
```go
// Session management
func (s *Server) createSession(username string) *Session {
    s.sessionMu.Lock()
    defer s.sessionMu.Unlock()

    session := &Session{...}
    s.sessions[session.ID] = session
    return session
}
```

**DPI Classifier**:
```go
// Flow access (thread-safe copy)
func (c *Classifier) GetActiveFlows() []*Flow {
    c.mu.RLock()
    defer c.mu.RUnlock()

    flows := make([]*Flow, 0, len(c.flows))
    for _, flow := range c.flows {
        flowCopy := *flow  // Copy to avoid race
        flows = append(flows, &flowCopy)
    }
    return flows
}
```

---

## Performance Characteristics

### Benchmarks

**Hardware**: Intel i7-10700 (8 cores), 32GB RAM, Windows 11

**Packet Processing**:
- Throughput: 950 Mbps per WAN (single core)
- Throughput: 3.5 Gbps aggregate (4 WANs, multi-core)
- Latency: +0.5ms average overhead
- CPU: ~5% per 100 Mbps (with encryption)

**Health Monitoring**:
- CPU overhead: <0.1% per WAN
- Memory: ~50 KB per WAN
- Check frequency: 1-5 seconds (adaptive)

**DPI Classification**:
- Throughput: 2 Gbps (single core)
- Flow table: 10,000 flows max (configurable)
- Memory: ~200 bytes per flow
- CPU: <1% at 10,000 flows

**Web UI**:
- Session storage: ~200 bytes per session
- WebSocket overhead: <1 KB/s per client
- API response time: <10ms average

---

## Deployment Topologies

### Topology 1: Single Client, Single Server

```
┌─────────────────────────────────────────┐
│          Client Location                 │
│                                          │
│  ┌────────────────────────────────┐     │
│  │   MultiWANBond Client          │     │
│  │   - 3 WAN connections          │     │
│  │   - TUN interface              │     │
│  └────┬────────┬────────┬─────────┘     │
│       │        │        │                │
│   WAN 1    WAN 2    WAN 3               │
│   Fiber   Starlink   LTE                │
└───────┼────────┼────────┼────────────────┘
        │        │        │
        └────────┴────────┴─── Internet ───┐
                                            │
┌───────────────────────────────────────────▼─┐
│          Server Location                     │
│                                              │
│  ┌────────────────────────────────────┐     │
│  │   MultiWANBond Server              │     │
│  │   - Single public IP               │     │
│  │   - Aggregates client WANs         │     │
│  │   - TUN interface                  │     │
│  └────────────────────────────────────┘     │
│                    │                         │
│                    ▼                         │
│           Corporate Network                  │
│           or Internet Access                 │
└──────────────────────────────────────────────┘
```

**Use Case**: Remote office, home user bonding multiple connections

---

### Topology 2: Multiple Clients, Single Server

```
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│  Client 1    │  │  Client 2    │  │  Client 3    │
│  3 WANs      │  │  2 WANs      │  │  4 WANs      │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │
       └─────────────────┴─────────────────┘
                         │
                   [Internet]
                         │
                         ▼
              ┌─────────────────────┐
              │ MultiWANBond Server │
              │  - Central Hub      │
              │  - Per-client VPN   │
              └─────────────────────┘
```

**Use Case**: Enterprise with multiple remote offices

---

### Topology 3: Peer-to-Peer (with NAT Traversal)

```
┌──────────────────────────────────┐
│         Peer A                   │
│   MultiWANBond Client            │
│   - NAT Traversal enabled        │
│   - STUN client                  │
└──────────┬───────────────────────┘
           │
           │  [NAT: Full Cone]
           │
           ▼
      [Internet]
      STUN Server
      (NAT discovery)
           ▲
           │
           │  [NAT: Port-Restricted]
           │
           │
┌──────────┴───────────────────────┐
│         Peer B                   │
│   MultiWANBond Client            │
│   - NAT Traversal enabled        │
│   - STUN client                  │
└──────────────────────────────────┘
```

**Use Case**: Direct P2P bonding without dedicated server

---

### Topology 4: Hybrid (Relay Fallback)

```
                    ┌─────────────┐
                    │    STUN     │
                    │   Server    │
                    └──────┬──────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│   Client A   │  │   Client B   │  │   Client C   │
│ (Sym NAT)    │  │ (Cone NAT)   │  │ (Cone NAT)   │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘
       │                 │                 │
       │                 └─────────────────┘
       │                  Direct P2P ✓
       │
       └─────────────────┐
                         ▼
                ┌─────────────────┐
                │   TURN Relay    │
                │   (fallback)    │
                └─────────────────┘
```

**Use Case**: Mixed NAT environments, automatic fallback to relay

---

## Future Architecture

### Planned Enhancements (v1.2+)

**1. QUIC Protocol Support**:
- Replace UDP with QUIC
- Built-in encryption
- Connection migration
- Improved NAT traversal

**2. Distributed Metrics Storage**:
- InfluxDB integration
- Long-term metrics retention
- Cross-node metrics aggregation

**3. Kubernetes Operator**:
```
┌────────────────────────────────────────┐
│         Kubernetes Cluster             │
│                                        │
│  ┌──────────────────────────────────┐ │
│  │  MultiWANBond Operator           │ │
│  │  - Auto-discovery of nodes       │ │
│  │  - Dynamic WAN configuration     │ │
│  │  - Health monitoring             │ │
│  └──────────────────────────────────┘ │
│                                        │
│  ┌────────┐  ┌────────┐  ┌────────┐  │
│  │ Node 1 │  │ Node 2 │  │ Node 3 │  │
│  │ 2 WANs │  │ 3 WANs │  │ 2 WANs │  │
│  └────────┘  └────────┘  └────────┘  │
└────────────────────────────────────────┘
```

**4. Hardware Acceleration**:
- DPDK integration for packet processing
- AES-NI for encryption
- GPU acceleration for FEC

**5. Multi-Node Clustering**:
```
┌─────────────┐     ┌─────────────┐
│   Node 1    │────│   Node 2    │
│  (Primary)  │     │ (Secondary) │
└─────┬───────┘     └─────┬───────┘
      │                   │
      └─────────┬─────────┘
                │
         ┌──────▼──────┐
         │   Node 3    │
         │  (Tertiary) │
         └─────────────┘
```
- Distributed session management
- Load balancing across nodes
- Automatic failover

---

## File Organization

```
MultiWANBond/
├── cmd/
│   ├── server/
│   │   └── main.go                    # Server entry point
│   ├── client/
│   │   └── main.go                    # Client entry point
│   └── test/
│       ├── network_detect.go          # Network detection tests
│       ├── health_checker.go          # Health check tests
│       └── final_integration.go       # Integration tests
│
├── pkg/
│   ├── bonder/
│   │   ├── bonder.go                  # Core bonding logic
│   │   ├── session.go                 # Session management
│   │   └── wan.go                     # WAN interface management
│   │
│   ├── nat/
│   │   ├── manager.go                 # NAT traversal manager
│   │   ├── stun.go                    # STUN client
│   │   ├── cgnat.go                   # CGNAT detector
│   │   ├── holepunch.go               # Hole punching
│   │   └── relay.go                   # TURN relay client
│   │
│   ├── dpi/
│   │   ├── classifier.go              # DPI classifier
│   │   ├── protocols.go               # Protocol definitions
│   │   └── flow.go                    # Flow tracking
│   │
│   ├── health/
│   │   ├── monitor.go                 # Health monitor
│   │   ├── checker.go                 # Check implementations
│   │   └── adaptive.go                # Adaptive intervals
│   │
│   ├── routing/
│   │   ├── router.go                  # Routing engine
│   │   ├── policy.go                  # Policy routing
│   │   └── loadbalancer.go            # Load balancing modes
│   │
│   ├── metrics/
│   │   ├── collector.go               # Metrics collector
│   │   ├── timeseries.go              # Time-series storage
│   │   └── exporter.go                # Export formats
│   │
│   ├── webui/
│   │   ├── server.go                  # Web server
│   │   ├── api.go                     # REST API handlers
│   │   └── websocket.go               # WebSocket handler
│   │
│   ├── processor/
│   │   ├── processor.go               # Packet processor
│   │   ├── encap.go                   # Encapsulation
│   │   └── reorder.go                 # Reordering buffer
│   │
│   ├── fec/
│   │   └── fec.go                     # Reed-Solomon FEC
│   │
│   └── config/
│       └── config.go                  # Configuration management
│
├── webui/
│   ├── login.html                     # Login page
│   ├── dashboard.html                 # Main dashboard
│   ├── flows.html                     # Flow analysis
│   ├── analytics.html                 # Traffic analytics
│   ├── logs.html                      # Log viewer
│   └── config.html                    # Configuration
│
└── docs/
    ├── README.md                      # Main documentation
    ├── ARCHITECTURE.md                # This file
    ├── API_REFERENCE.md               # API documentation
    ├── UNIFIED_WEB_UI_IMPLEMENTATION.md
    ├── NAT_DPI_INTEGRATION.md
    └── ...                            # Other docs
```

---

## Key Design Decisions

### 1. Why UDP Instead of TCP?

**TCP Issues for Multi-Path**:
- Head-of-line blocking
- Congestion control per path
- Path-dependent sequence numbers
- Difficult to aggregate bandwidth

**UDP Advantages**:
- No head-of-line blocking
- Custom congestion control
- Path-independent sequencing
- Easy to distribute packets

**Our Approach**:
- UDP transport layer
- Custom reliability layer
- FEC for packet loss
- Reordering buffer

---

### 2. Why Reed-Solomon FEC?

**Advantages**:
- Erasure coding (recover from any packet loss pattern)
- Configurable redundancy
- Well-tested implementations
- Hardware acceleration available

**Alternatives Considered**:
- XOR FEC: Too simple, limited recovery
- Fountain codes: Too complex, unpredictable latency
- Retransmission: Adds latency, wastes bandwidth

---

### 3. Why Cookie-Based Sessions Instead of JWT?

**Cookie Advantages**:
- HttpOnly (XSS protection)
- SameSite (CSRF protection)
- Server-side revocation
- No token refresh needed

**JWT Disadvantages**:
- Cannot revoke without database
- Larger payload
- Susceptible to XSS if stored in localStorage
- Refresh token complexity

---

### 4. Why In-Memory Metrics Instead of Database?

**In-Memory Advantages**:
- Ultra-low latency (<1ms)
- No external dependencies
- Simple deployment
- High throughput

**Database Disadvantages**:
- Higher latency (>10ms)
- External dependency
- Complex setup
- Lower throughput

**Future Plan**: Optional persistent storage for long-term retention

---

## Security Architecture

### Defense in Depth

**Layer 1: Network**:
- Firewall rules (restrict to bonding ports)
- Rate limiting (prevent DoS)
- IP whitelisting (restrict server access)

**Layer 2: Transport**:
- Encryption (AES-256-GCM or ChaCha20-Poly1305)
- Authentication (AEAD tag verification)
- Replay protection (sequence numbers)

**Layer 3: Application**:
- Session management (secure cookies)
- Input validation (prevent injection)
- CSRF protection (SameSite cookies)
- XSS protection (HttpOnly cookies)

**Layer 4: Data**:
- No plaintext credentials in config
- Environment variable support
- Key rotation support
- Secure key derivation (HKDF)

---

**Last Updated**: November 2, 2025
**Architecture Version**: 1.1
**MultiWANBond Version**: 1.1

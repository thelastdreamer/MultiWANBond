# MultiWANBond Architecture

## Overview

MultiWANBond is a high-performance, cross-platform network protocol designed to bond multiple WAN connections into a single, reliable, high-bandwidth link. This document describes the internal architecture and design decisions.

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                         │
│              (User Applications & Services)                  │
└────────────────────────┬────────────────────────────────────┘
                         │ API
┌────────────────────────▼────────────────────────────────────┐
│                      Bonder Core                            │
│  ┌──────────────────────────────────────────────────────┐  │
│  │          Session Management & Control                 │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │ Health Check │  │    Router    │  │ Packet Processor │  │
│  │   Manager    │  │   Engine     │  │   & Reordering   │  │
│  └──────────────┘  └──────────────┘  └─────────────────┘  │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │     FEC      │  │   Plugin     │  │      Config      │  │
│  │   Manager    │  │   System     │  │    Management    │  │
│  └──────────────┘  └──────────────┘  └─────────────────┘  │
└────────────────────────┬────────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┬──────────────┐
         │               │               │              │
┌────────▼───────┐ ┌────▼──────┐ ┌──────▼────┐ ┌──────▼────┐
│  WAN Interface │ │    WAN    │ │    WAN    │ │    WAN    │
│   Handler 1    │ │  Handler  │ │  Handler  │ │  Handler  │
│                │ │     2     │ │     3     │ │     N     │
└────────┬───────┘ └────┬──────┘ └──────┬────┘ └──────┬────┘
         │              │               │              │
         ▼              ▼               ▼              ▼
    Physical WAN   Physical WAN   Physical WAN   Physical WAN
```

## Core Components

### 1. Bonder Core

The central component that orchestrates all other modules.

**Responsibilities:**
- Session management
- Component lifecycle management
- Data flow coordination
- Event handling

**Key Files:**
- [pkg/bonder/bonder.go](../pkg/bonder/bonder.go)

### 2. Health Checker

Monitors the health of each WAN connection in real-time.

**Features:**
- Sub-second failure detection (<200ms default)
- Continuous latency monitoring
- Jitter calculation
- Packet loss detection
- Moving average calculations

**Algorithm:**
1. Send heartbeat probes at configured intervals (default: 200ms)
2. Measure round-trip time (RTT)
3. Calculate jitter as variance from average latency
4. Track consecutive failures
5. Transition state after threshold exceeded (default: 3 failures)

**State Machine:**
```
    ┌──────┐
    │ Down │◄───────┐
    └──┬───┘        │
       │            │
       ▼            │
  ┌──────────┐     │
  │ Starting │     │
  └─────┬────┘     │
        │          │
        ▼          │
    ┌──────┐       │
    │  Up  │       │ failure_threshold
    └──┬───┘       │ exceeded
       │           │
       ▼           │
  ┌───────────┐   │
  │ Degraded  │───┘
  └─────┬─────┘
        │
        ▼
  ┌────────────┐
  │ Recovering │──┐
  └────────────┘  │
        ▲         │
        └─────────┘
```

**Key Files:**
- [pkg/health/checker.go](../pkg/health/checker.go)

### 3. Router

Determines which WAN interface(s) to use for each packet.

**Routing Modes:**

#### Round Robin
- Distributes packets evenly across all WANs
- Simple, deterministic
- Best for: Testing, equal-quality connections

#### Weighted
- Routes based on connection quality score
- Score = weight × (100/latency) × (1-loss) × (1-utilization)
- Best for: Mixed connection types

#### Least Used
- Routes to WAN with lowest current utilization
- Balances load evenly
- Best for: Maximizing aggregate bandwidth

#### Least Latency
- Always routes to lowest-latency WAN
- Prioritizes speed over load balancing
- Best for: Real-time applications (VoIP, gaming)

#### Per-Flow
- Consistent routing per flow (5-tuple hash)
- Maintains packet ordering per flow
- Best for: TCP connections

#### Adaptive
- Dynamically selects mode based on packet priority:
  - High priority (>200): Least Latency
  - Low priority (<50): Least Used
  - Normal: Weighted
- Best for: Production environments

**Key Files:**
- [pkg/router/router.go](../pkg/router/router.go)

### 4. Packet Processor

Handles packet encoding, decoding, and reordering.

**Packet Format:**
```
┌──────────────────────────────────────────────────────────┐
│                       Packet Header                       │
├────────┬─────────┬──────────┬────────────┬───────────────┤
│ Byte 0 │  Byte 1 │ Bytes 2-3│ Bytes 4-11 │ Bytes 12-19   │
├────────┼─────────┼──────────┼────────────┼───────────────┤
│Version │  Type   │  Flags   │ Session ID │ Sequence ID   │
└────────┴─────────┴──────────┴────────────┴───────────────┘

┌───────────────┬────────┬──────────┬──────────┬──────────┐
│ Bytes 20-27   │ Byte 28│ Byte 29  │Bytes30-33│Bytes34-37│
├───────────────┼────────┼──────────┼──────────┼──────────┤
│  Timestamp    │ WAN ID │ Priority │ Data Len │ Checksum │
└───────────────┴────────┴──────────┴──────────┴──────────┘

┌─────────────────────────────────────────────────────────┐
│                      Payload Data                        │
│                    (Variable Length)                     │
└─────────────────────────────────────────────────────────┘
```

**Reordering Algorithm:**
1. Maintain expected sequence number
2. Buffer out-of-order packets
3. Deliver in-order when sequence matches
4. Force delivery if buffer full or timeout

**Key Files:**
- [pkg/packet/processor.go](../pkg/packet/processor.go)

### 5. FEC Manager

Implements Forward Error Correction using Reed-Solomon encoding.

**How FEC Works:**
```
Original Data: [D1] [D2] [D3] [D4]
                 │    │    │    │
                 └────┴────┴────┴──► Reed-Solomon Encoder
                                           │
         ┌─────────────┬────────┬─────────┴────────┬────────┐
         │             │        │                  │        │
    [D1] [D2] [D3] [D4] [P1] [P2]

Data Shards: D1, D2, D3, D4
Parity Shards: P1, P2

If any 2 shards are lost, data can be recovered from remaining 4.
```

**Configuration:**
- `redundancy`: Ratio of parity to data (0.2 = 20% overhead)
- `data_shards`: Number of data chunks (default: 4)
- `parity_shards`: Number of parity chunks (default: 2)

**Trade-offs:**
- Higher redundancy = better recovery, more bandwidth
- Lower redundancy = less overhead, faster transmission

**Key Files:**
- [pkg/fec/reedsolomon.go](../pkg/fec/reedsolomon.go)

### 6. Plugin System

Extensible plugin architecture for custom functionality.

**Plugin Types:**

#### Packet Filter
```go
type PacketFilter interface {
    FilterOutgoing(*Packet) (*Packet, error)
    FilterIncoming(*Packet) (*Packet, error)
    Priority() int
}
```

Use cases:
- Encryption/decryption
- Compression
- Rate limiting
- Traffic shaping
- Protocol translation

#### Metrics Collector
```go
type MetricsCollector interface {
    RecordPacket(wanID uint8, packet *Packet, sent bool)
    RecordMetrics(wanID uint8, metrics *WANMetrics)
    GetReport() (map[string]interface{}, error)
}
```

Use cases:
- Prometheus exporter
- InfluxDB integration
- Custom logging
- Performance analysis

#### Alert Manager
```go
type AlertManager interface {
    Alert(level AlertLevel, message string, details map[string]interface{}) error
    Subscribe() <-chan Alert
}
```

Use cases:
- Email notifications
- Slack/Discord webhooks
- SMS alerts
- SNMP traps

**Key Files:**
- [pkg/plugin/manager.go](../pkg/plugin/manager.go)

### 7. Configuration System

Hot-reloadable configuration with JSON format.

**Features:**
- JSON-based configuration
- Hot reload without restart
- Type-safe parsing
- Validation
- Defaults for all values

**Key Files:**
- [pkg/config/config.go](../pkg/config/config.go)

## Protocol Specification

### Packet Types

| Type | Value | Description |
|------|-------|-------------|
| Data | 0 | User data packet |
| Ack | 1 | Acknowledgment |
| Heartbeat | 2 | Health check probe |
| Control | 3 | Control message |
| Multicast | 4 | Multicast data |
| FEC | 5 | Forward error correction packet |

### Flags

| Flag | Bit | Description |
|------|-----|-------------|
| Duplicate | 0 | This is a duplicate packet |
| FEC | 1 | Contains FEC data |
| Compressed | 2 | Payload is compressed |
| Encrypted | 3 | Payload is encrypted |
| Fragment | 4 | Packet is fragmented |
| LastFrag | 5 | Last fragment in sequence |

### Connection Types

Supported WAN types with typical characteristics:

| Type | Typical Latency | Typical Bandwidth | Reliability |
|------|----------------|-------------------|-------------|
| Fiber | 1-20ms | 100 Mbps - 10 Gbps | Very High |
| Cable | 10-30ms | 50-500 Mbps | High |
| VDSL | 10-40ms | 10-100 Mbps | Medium-High |
| ADSL | 20-50ms | 1-24 Mbps | Medium |
| LTE | 20-80ms | 5-100 Mbps | Medium |
| 5G | 10-40ms | 100-1000 Mbps | Medium-High |
| Starlink | 20-80ms | 50-200 Mbps | Medium |
| Satellite | 500-700ms | 1-25 Mbps | Low-Medium |

## Performance Considerations

### Latency Optimization

1. **Minimize Reorder Buffer**: Smaller buffer = lower delay
2. **Use Least Latency Mode**: Direct packets to fastest path
3. **Disable FEC**: Avoid encoding overhead for low-latency
4. **Reduce Health Check Interval**: Faster failure detection

### Throughput Optimization

1. **Use Least Used Mode**: Balance load across all WANs
2. **Enable FEC**: Reduce retransmissions
3. **Increase Buffers**: Allow more in-flight packets
4. **Tune MTU**: Match largest common MTU

### Reliability Optimization

1. **Enable Packet Duplication**: Send on multiple paths
2. **Increase FEC Redundancy**: Higher recovery capability
3. **Lower Failure Threshold**: Faster failover
4. **Use Adaptive Mode**: Intelligent path selection

## Concurrency Model

MultiWANBond uses Go's goroutines for concurrent operation:

```
Main Goroutine
    │
    ├─► Health Checker Goroutine (per WAN)
    │   ├─► Probe sender
    │   └─► Metrics calculator
    │
    ├─► Sender Goroutine
    │   ├─► Packet encoding
    │   ├─► Routing decision
    │   └─► WAN transmission
    │
    ├─► Receiver Goroutine (per WAN)
    │   ├─► Packet reception
    │   ├─► Decoding
    │   └─► Reordering
    │
    └─► Health Event Handler
        └─► State transitions
```

**Synchronization:**
- Mutexes for shared state (WAN map, metrics)
- Channels for data flow (send/receive)
- Atomic operations for counters (sequence ID)

## Security Considerations

**Current Implementation:**
- CRC32 checksum for packet integrity
- No encryption (transport layer responsibility)

**Future Enhancements:**
- Built-in TLS/DTLS support
- Authentication tokens
- Rate limiting per source
- DoS protection

## Cross-Platform Support

### Platform-Specific Code

MultiWANBond is written in pure Go for maximum portability:

- **Linux**: Full support, optimal performance
- **Windows**: Full support, requires Admin for raw sockets
- **macOS**: Full support
- **Android**: Via gomobile, limited to userspace
- **iOS**: Via gomobile, limited to userspace

### Build Tags

Use build tags for platform-specific optimizations:

```go
// +build linux

package platform

// Linux-specific optimizations
```

## Testing Strategy

### Unit Tests
- Individual component testing
- Mock interfaces for isolation
- Coverage target: >80%

### Integration Tests
- Multi-component interaction
- Simulated network conditions
- Failure scenarios

### Performance Tests
- Throughput benchmarks
- Latency measurements
- Scalability tests

### Platform Tests
- Cross-compilation verification
- Platform-specific features
- Mobile integration

## Future Enhancements

1. **Hardware Acceleration**
   - DPDK support for kernel bypass
   - GPU acceleration for FEC

2. **Advanced Protocols**
   - QUIC support
   - WebRTC data channels

3. **Machine Learning**
   - Predictive path selection
   - Anomaly detection
   - Traffic classification

4. **Management**
   - Web UI
   - REST API
   - GraphQL endpoint

5. **Enterprise Features**
   - Active Directory integration
   - RADIUS authentication
   - SNMP support
   - Syslog integration

## Contributing

See the architecture guidelines in [CONTRIBUTING.md](../CONTRIBUTING.md) for information on:
- Code style
- Testing requirements
- Documentation standards
- Review process

## References

- [RFC 793](https://tools.ietf.org/html/rfc793) - TCP
- [RFC 768](https://tools.ietf.org/html/rfc768) - UDP
- [RFC 6824](https://tools.ietf.org/html/rfc6824) - MPTCP
- Reed-Solomon Error Correction
- Go Concurrency Patterns

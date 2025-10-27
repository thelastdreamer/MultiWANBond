# MultiWANBond

A high-performance, cross-platform network protocol for bonding multiple WAN connections to create an unbreakable, high-bandwidth, low-latency network link.

## Features

### Core Capabilities

- **Multi-WAN Bonding**: Combine unlimited WAN connections (ADSL, VDSL, Fiber, Starlink, Satellite, LTE, 5G, Cable)
- **Real-time Monitoring**: Track latency, jitter, packet loss, and bandwidth for each connection
- **Sub-second Failure Detection**: Detect connection failures in less than 1 second
- **Intelligent Packet Routing**: Multiple load-balancing strategies to optimize performance
- **Packet Reordering**: Ensure packets are delivered in the correct order
- **Forward Error Correction (FEC)**: Recover from packet loss without retransmission
- **Multicast Support**: Send and receive multicast packets across bonded connections
- **Packet Duplication**: Send critical packets on multiple paths for maximum reliability

### Advanced Features

- **Adaptive Routing**: Automatically select best path based on real-time conditions
- **Hot Configuration Reload**: Update configuration without restarting
- **Plugin Architecture**: Extend functionality with custom plugins
- **Cross-Platform**: Runs on Linux, Windows, macOS, Android, iOS, ARM devices
- **Zero Configuration**: Works with sensible defaults, configure only what you need

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Application Layer                       │
│            (Your apps using MultiWANBond)                   │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    MultiWANBond Core                        │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │ Health Check │  │    Router    │  │ Packet Processor │  │
│  │  <1s detect  │  │  Adaptive    │  │   Reordering     │  │
│  └──────────────┘  └──────────────┘  └─────────────────┘  │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │     FEC      │  │   Plugins    │  │   Config Mgmt   │  │
│  │ Reed-Solomon │  │  Extensible  │  │   Hot Reload    │  │
│  └──────────────┘  └──────────────┘  └─────────────────┘  │
└────────────────────────┬────────────────────────────────────┘
                         │
         ┌───────────────┼───────────────┬──────────────┐
         │               │               │              │
┌────────▼───────┐ ┌────▼──────┐ ┌──────▼────┐ ┌──────▼────┐
│  WAN 1: Fiber  │ │ WAN 2:    │ │ WAN 3:    │ │ WAN 4:    │
│  100 Mbps      │ │ Starlink  │ │  VDSL     │ │   LTE     │
│  10ms latency  │ │ 50 Mbps   │ │ 10 Mbps   │ │  5 Mbps   │
└────────────────┘ └───────────┘ └───────────┘ └───────────┘
```

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/thelastdreamer/MultiWANBond.git
cd MultiWANBond

# Build for your platform
go build -o multiwanbond ./cmd/server

# Or build for all platforms
make build-all
```

### Cross-Compilation

```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o multiwanbond-linux-amd64 ./cmd/server

# Linux ARM64 (Raspberry Pi, etc.)
GOOS=linux GOARCH=arm64 go build -o multiwanbond-linux-arm64 ./cmd/server

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o multiwanbond-windows-amd64.exe ./cmd/server

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o multiwanbond-darwin-arm64 ./cmd/server

# Android ARM64 (requires gomobile)
gomobile bind -target=android/arm64 ./pkg/...

# iOS ARM64 (requires gomobile)
gomobile bind -target=ios/arm64 ./pkg/...
```

## Quick Start

### 1. Create Configuration

Create a config file `config.json`:

```json
{
  "session": {
    "local_endpoint": "0.0.0.0:9000",
    "remote_endpoint": "server.example.com:9000"
  },
  "wans": [
    {
      "id": 1,
      "name": "Primary",
      "type": "fiber",
      "local_addr": "192.168.1.100",
      "remote_addr": "server.example.com:9000",
      "enabled": true
    },
    {
      "id": 2,
      "name": "Backup",
      "type": "lte",
      "local_addr": "192.168.2.100",
      "remote_addr": "server.example.com:9000",
      "enabled": true
    }
  ],
  "routing": {
    "mode": "adaptive"
  }
}
```

### 2. Run Server

```bash
./multiwanbond -config config.json -mode server
```

### 3. Run Client

```bash
./multiwanbond -config config.json -mode client
```

## Configuration Reference

### Session Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `local_endpoint` | string | `"0.0.0.0:9000"` | Local bind address |
| `remote_endpoint` | string | `""` | Remote server address |
| `duplicate_packets` | bool | `false` | Send packets on multiple WANs |
| `duplicate_mode` | string | `"fastest"` | How to handle duplicates: `first`, `fastest`, `best` |
| `reorder_buffer` | int | `1000` | Size of packet reorder buffer |
| `reorder_timeout` | string | `"500ms"` | Max wait time for out-of-order packets |
| `multicast_enabled` | bool | `false` | Enable multicast support |

### WAN Configuration

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | uint8 | Yes | Unique WAN identifier (1-255) |
| `name` | string | Yes | Human-readable name |
| `type` | string | Yes | Connection type: `adsl`, `vdsl`, `fiber`, `starlink`, `satellite`, `lte`, `5g`, `cable` |
| `local_addr` | string | Yes | Local IP address to bind |
| `remote_addr` | string | Yes | Remote endpoint address |
| `max_bandwidth` | uint64 | No | Maximum bandwidth in bytes/sec |
| `max_latency` | string | `"100ms"` | Maximum acceptable latency |
| `max_jitter` | string | `"20ms"` | Maximum acceptable jitter |
| `max_packet_loss` | float64 | `2.0` | Maximum acceptable packet loss % |
| `health_check_interval` | string | `"200ms"` | Health check frequency |
| `failure_threshold` | int | `3` | Consecutive failures before marking down |
| `weight` | int | `10` | Weight for load balancing |
| `enabled` | bool | `true` | Enable this WAN |

### Routing Modes

| Mode | Description | Best For |
|------|-------------|----------|
| `round_robin` | Simple round-robin distribution | Equal connections, testing |
| `weighted` | Weighted by bandwidth, latency, loss | Mixed connection types |
| `least_used` | Route to least utilized connection | Balanced load distribution |
| `least_latency` | Route to lowest latency connection | Latency-sensitive applications |
| `per_flow` | Consistent routing per flow (5-tuple) | TCP connections, gaming |
| `adaptive` | Dynamically adapt based on conditions | Production (recommended) |

### FEC Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `enabled` | bool | `false` | Enable Forward Error Correction |
| `redundancy` | float64 | `0.2` | Redundancy ratio (0.2 = 20% overhead) |
| `data_shards` | int | `4` | Number of data shards |
| `parity_shards` | int | `2` | Number of parity shards |

## Usage Examples

### Example 1: High Availability Setup

Combine fiber primary with LTE backup for zero-downtime connectivity:

```json
{
  "wans": [
    {
      "id": 1,
      "name": "Fiber Primary",
      "type": "fiber",
      "weight": 10,
      "enabled": true
    },
    {
      "id": 2,
      "name": "LTE Backup",
      "type": "lte",
      "weight": 1,
      "enabled": true
    }
  ],
  "routing": {
    "mode": "weighted"
  }
}
```

### Example 2: Maximum Bandwidth Aggregation

Combine all available connections for maximum throughput:

```json
{
  "wans": [
    {
      "id": 1,
      "name": "Fiber 1",
      "type": "fiber",
      "max_bandwidth": 104857600,
      "weight": 10
    },
    {
      "id": 2,
      "name": "Fiber 2",
      "type": "fiber",
      "max_bandwidth": 104857600,
      "weight": 10
    },
    {
      "id": 3,
      "name": "Starlink",
      "type": "starlink",
      "max_bandwidth": 52428800,
      "weight": 5
    }
  ],
  "routing": {
    "mode": "least_used"
  }
}
```

### Example 3: Ultra-Reliable with FEC and Duplication

For mission-critical applications where reliability is paramount:

```json
{
  "session": {
    "duplicate_packets": true,
    "duplicate_mode": "fastest"
  },
  "wans": [
    {
      "id": 1,
      "name": "Primary",
      "type": "fiber"
    },
    {
      "id": 2,
      "name": "Backup 1",
      "type": "starlink"
    },
    {
      "id": 3,
      "name": "Backup 2",
      "type": "lte"
    }
  ],
  "routing": {
    "mode": "adaptive"
  },
  "fec": {
    "enabled": true,
    "redundancy": 0.3,
    "data_shards": 4,
    "parity_shards": 2
  }
}
```

## Plugin Development

MultiWANBond supports custom plugins for extending functionality:

### Creating a Plugin

```go
package myplugin

import (
    "context"
    "github.com/thelastdreamer/MultiWANBond/pkg/plugin"
    "github.com/thelastdreamer/MultiWANBond/pkg/protocol"
)

type MyPlugin struct {
    *plugin.BasePlugin
}

func New() *MyPlugin {
    return &MyPlugin{
        BasePlugin: plugin.NewBasePlugin("myplugin", "1.0.0"),
    }
}

func (p *MyPlugin) Start(ctx context.Context) error {
    // Plugin initialization
    return nil
}

func (p *MyPlugin) Stop() error {
    // Cleanup
    return nil
}
```

### Plugin Types

1. **PacketFilter**: Inspect/modify/drop packets
2. **MetricsCollector**: Collect and export metrics
3. **AlertManager**: Handle alerts and notifications

## Performance Tuning

### Optimizing for Low Latency

```json
{
  "routing": {
    "mode": "least_latency"
  },
  "fec": {
    "enabled": false
  },
  "session": {
    "reorder_buffer": 100,
    "reorder_timeout": "50ms"
  }
}
```

### Optimizing for High Throughput

```json
{
  "routing": {
    "mode": "least_used"
  },
  "fec": {
    "enabled": true,
    "redundancy": 0.1
  },
  "session": {
    "reorder_buffer": 5000,
    "reorder_timeout": "1s"
  }
}
```

### Optimizing for Reliability

```json
{
  "routing": {
    "mode": "adaptive"
  },
  "fec": {
    "enabled": true,
    "redundancy": 0.3
  },
  "session": {
    "duplicate_packets": true,
    "duplicate_mode": "fastest",
    "reorder_buffer": 2000,
    "reorder_timeout": "800ms"
  }
}
```

## Monitoring & Metrics

MultiWANBond provides real-time metrics for each WAN connection:

- **Latency**: Round-trip time (RTT)
- **Jitter**: Variance in latency
- **Packet Loss**: Percentage of lost packets
- **Bandwidth**: Current throughput
- **State**: Connection health status

### Health States

- `Down`: Connection unavailable
- `Starting`: Connection initializing
- `Up`: Connection healthy
- `Degraded`: Connection experiencing issues
- `Recovering`: Connection recovering from failure

## API Integration

### Using MultiWANBond in Your Go Application

```go
package main

import (
    "context"
    "github.com/thelastdreamer/MultiWANBond/pkg/bonder"
    "github.com/thelastdreamer/MultiWANBond/pkg/config"
)

func main() {
    // Load configuration
    cfg, err := config.LoadBondConfig("config.json")
    if err != nil {
        panic(err)
    }

    // Create bonder
    b, err := bonder.New(cfg)
    if err != nil {
        panic(err)
    }

    // Start bonding
    ctx := context.Background()
    if err := b.Start(ctx); err != nil {
        panic(err)
    }
    defer b.Stop()

    // Send data
    data := []byte("Hello, MultiWANBond!")
    if err := b.Send(data); err != nil {
        panic(err)
    }

    // Receive data
    recvChan := b.Receive()
    for data := range recvChan {
        // Process received data
        println("Received:", string(data))
    }
}
```

## Platform-Specific Notes

### Android

Build with `gomobile`:

```bash
gomobile bind -target=android -o multiwanbond.aar ./pkg/...
```

### iOS

Build with `gomobile`:

```bash
gomobile bind -target=ios -o MultiWANBond.xcframework ./pkg/...
```

### Linux

Enable raw socket capabilities (optional, for better performance):

```bash
sudo setcap cap_net_raw+ep ./multiwanbond
```

### Windows

Run as Administrator for best performance.

## Troubleshooting

### Connection Won't Bond

1. Check firewall rules allow UDP port 9000 (or configured port)
2. Verify remote endpoint is reachable from all WANs
3. Check WAN interface configuration (IP addresses)
4. Review logs for error messages

### High Latency

1. Check individual WAN latencies
2. Reduce `reorder_buffer` size
3. Decrease `reorder_timeout`
4. Use `least_latency` routing mode

### Packet Loss

1. Enable FEC with appropriate redundancy
2. Enable packet duplication for critical traffic
3. Check WAN connection quality
4. Adjust `failure_threshold` and `health_check_interval`

## Roadmap

- [ ] Web-based management UI
- [ ] REST API for configuration
- [ ] Support for QUIC protocol
- [ ] Hardware acceleration (DPDK)
- [ ] Compression support
- [ ] Encryption (built-in TLS)
- [ ] Performance benchmarking tools
- [ ] Docker containerization
- [ ] Kubernetes operator

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License - see [LICENSE](LICENSE) file for details.

## Support

- GitHub Issues: https://github.com/thelastdreamer/MultiWANBond/issues
- Discussions: https://github.com/thelastdreamer/MultiWANBond/discussions
- Email: support@example.com

## Acknowledgments

MultiWANBond is inspired by various multi-path and link aggregation technologies including MPTCP, MLPPP, and modern SD-WAN solutions.

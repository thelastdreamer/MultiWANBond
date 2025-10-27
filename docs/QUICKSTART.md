# Quick Start Guide

Get MultiWANBond up and running in 5 minutes!

## Prerequisites

- Go 1.21 or later (for building from source)
- Multiple network interfaces configured on your system
- Network connectivity on each interface

## Installation

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/thelastdreamer/MultiWANBond.git
cd MultiWANBond

# Build
make build

# Binaries will be in build/
```

### Option 2: Download Pre-built Binary

Download the latest release for your platform from the [Releases page](https://github.com/thelastdreamer/MultiWANBond/releases).

## Basic Setup

### 1. Identify Your Network Interfaces

**Linux/macOS:**
```bash
ip addr show
# or
ifconfig
```

**Windows:**
```powershell
ipconfig
```

Note your interface IP addresses. For example:
- Ethernet: 192.168.1.100
- WiFi: 192.168.2.100
- LTE: 192.168.3.100

### 2. Create Configuration File

Create `my-config.json`:

```json
{
  "session": {
    "local_endpoint": "0.0.0.0:9000",
    "remote_endpoint": "server.example.com:9000"
  },
  "wans": [
    {
      "id": 1,
      "name": "Ethernet",
      "type": "fiber",
      "local_addr": "192.168.1.100",
      "remote_addr": "server.example.com:9000",
      "weight": 10,
      "enabled": true
    },
    {
      "id": 2,
      "name": "WiFi",
      "type": "wifi",
      "local_addr": "192.168.2.100",
      "remote_addr": "server.example.com:9000",
      "weight": 5,
      "enabled": true
    }
  ],
  "routing": {
    "mode": "adaptive"
  }
}
```

**Important:** Replace `server.example.com:9000` with your actual server address!

### 3. Start the Server

On your server machine:

```bash
./multiwanbond-server -config my-config.json
```

You should see:
```
Loading configuration from my-config.json
Creating MultiWANBond instance...
Starting MultiWANBond service...
Active WANs: 2
  - WAN 1 (Ethernet): fiber @ 192.168.1.100
  - WAN 2 (WiFi): wifi @ 192.168.2.100
MultiWANBond server is running. Press Ctrl+C to stop.
```

### 4. Start the Client

On your client machine:

```bash
./multiwanbond-client -config my-config.json
```

### 5. Send Test Messages

In interactive mode, type a message:

```
>> Hello, MultiWANBond!
Sent: Hello, MultiWANBond!

<< Received: ACK: Hello, MultiWANBond!
>>
```

Congratulations! Your multi-WAN bond is working!

## Common Scenarios

### Scenario 1: Simple Failover (Primary + Backup)

```json
{
  "wans": [
    {
      "id": 1,
      "name": "Primary",
      "type": "fiber",
      "local_addr": "192.168.1.100",
      "weight": 100,
      "enabled": true
    },
    {
      "id": 2,
      "name": "Backup",
      "type": "lte",
      "local_addr": "192.168.2.100",
      "weight": 1,
      "enabled": true
    }
  ],
  "routing": {
    "mode": "weighted"
  }
}
```

The backup WAN will only be used if the primary fails.

### Scenario 2: Load Balancing

```json
{
  "wans": [
    {
      "id": 1,
      "name": "WAN1",
      "local_addr": "192.168.1.100",
      "weight": 10,
      "enabled": true
    },
    {
      "id": 2,
      "name": "WAN2",
      "local_addr": "192.168.2.100",
      "weight": 10,
      "enabled": true
    }
  ],
  "routing": {
    "mode": "least_used"
  }
}
```

Traffic will be distributed evenly across both WANs.

### Scenario 3: High Reliability with FEC

```json
{
  "session": {
    "duplicate_packets": true
  },
  "wans": [
    {
      "id": 1,
      "name": "WAN1",
      "local_addr": "192.168.1.100",
      "enabled": true
    },
    {
      "id": 2,
      "name": "WAN2",
      "local_addr": "192.168.2.100",
      "enabled": true
    }
  ],
  "fec": {
    "enabled": true,
    "redundancy": 0.2
  }
}
```

Packets will be duplicated and FEC will provide additional recovery.

## Monitoring

### View Status

In the client, type commands:

```
>> /status
Connection Status:
  Session ID:       1730000000000000000
  Local Endpoint:   0.0.0.0:9000
  Remote Endpoint:  server.example.com:9000
  Active WANs:      2
  Uptime:           5m30s
```

### View Metrics

```
>> /metrics
Detailed Metrics:
  WAN 1: Ethernet (fiber)
    Latency:      10ms (avg: 12ms)
    Jitter:       2ms (avg: 3ms)
    Packet Loss:  0.00% (avg: 0.10%)
    Bandwidth:    100.50 Mbps
    Packets:      Sent: 1000, Recv: 998, Lost: 2
    Data:         Sent: 50.25 MB, Recv: 49.90 MB
```

### Server Statistics

The server automatically prints statistics every 10 seconds (configurable with `-stats-interval`).

## Testing Connection Failover

1. Start server and client
2. Send some messages to confirm connectivity
3. Disconnect one network interface (unplug cable, disable WiFi)
4. Watch the logs - failover should happen in <1 second
5. Continue sending messages - they should still go through!
6. Reconnect the interface - it will automatically rejoin

## Non-Interactive Mode

Send a single message:

```bash
./multiwanbond-client -config my-config.json -interactive=false -message "Test" -count 10
```

This sends "Test" 10 times and exits.

## Troubleshooting

### "Failed to bind: address already in use"

Another process is using port 9000. Either:
- Change the port in config: `"local_endpoint": "0.0.0.0:9001"`
- Stop the other process

### "No route to host"

Check:
1. Server is running
2. Remote endpoint address is correct
3. Firewall allows UDP traffic on port 9000
4. Network connectivity exists on specified interfaces

### "Permission denied"

On Linux/macOS, you may need elevated privileges:

```bash
sudo ./multiwanbond-server -config my-config.json
```

Or set capabilities:

```bash
sudo setcap cap_net_raw+ep ./multiwanbond-server
```

### High Latency

1. Check individual WAN latencies: `/metrics`
2. Reduce reorder buffer: `"reorder_buffer": 100`
3. Use least latency mode: `"mode": "least_latency"`

### Packet Loss

1. Enable FEC: `"fec": {"enabled": true, "redundancy": 0.2}`
2. Enable duplication: `"duplicate_packets": true`
3. Check WAN quality: `/metrics`

## Next Steps

- Read the full [README](../README.md) for all features
- Study [ARCHITECTURE.md](ARCHITECTURE.md) to understand internals
- Create custom plugins for your use case
- Join the community discussions

## Getting Help

- GitHub Issues: https://github.com/thelastdreamer/MultiWANBond/issues
- Discussions: https://github.com/thelastdreamer/MultiWANBond/discussions
- Documentation: https://github.com/thelastdreamer/MultiWANBond/wiki

Happy bonding! 🚀

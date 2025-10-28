# Web UI Integration & Fixes - Comprehensive Plan

## Current Problems

1. **Statistics Spam** - Same stats printed every 10s with no updates
2. **No Web UI** - Web UI code exists but not integrated into main executable
3. **Fake Connection Status** - Client says "connected" even when server is down
4. **No Real Metrics** - Health checks running but not updating displayed metrics
5. **Type Shows "Unknown"** - WAN type not being parsed correctly from config
6. **State Always 0 or 1** - States not updating based on actual health

## Root Causes

###  No Application Traffic
MultiWANBond is designed to tunnel application traffic between endpoints. Without traffic:
- Health checks have nothing to measure
- Metrics don't update
- Connection isn't really tested

### 2. Metrics Not Connected
The bonder collects metrics internally, but:
- main.go reads old static metrics
- Web UI has no data source
- PrintStats shows initial values only

### 3. Missing Integration
- Web UI server exists but never started
- No bridge between bonder and webui
- No HTML frontend files

## Solution Architecture

```
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│    Bonder    │─────▶│ Metrics      │─────▶│   Web UI     │
│  (Core)      │      │ Bridge       │      │   Server     │
└──────────────┘      └──────────────┘      └──────────────┘
       │                                            │
       │                                            │
       ▼                                            ▼
┌──────────────┐                          ┌──────────────┐
│ Health       │                          │  HTTP API    │
│ Checker      │                          │  :8080       │
└──────────────┘                          └──────────────┘
       │                                            │
       │                                            ▼
       ▼                                    ┌──────────────┐
┌──────────────┐                          │ HTML         │
│ Real-time    │                          │ Dashboard    │
│ Updates      │                          └──────────────┘
└──────────────┘
```

## Implementation Plan

### Phase 1: Fix Statistics Spam ✓
**File:** `cmd/server/main.go`

- Change `statsMonitor` to only print when metrics change
- Add comparison of previous vs current metrics
- Reduce verbosity, show summary only

**Changes:**
```go
func statsMonitor(b *bonder.Bonder, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    var prevMetrics map[uint8]*protocol.WANMetrics

    for range ticker.C {
        metrics := b.GetMetrics()

        // Only print if changed
        if !metricsEqual(prevMetrics, metrics) {
            printStats(b)
            prevMetrics = copyMetrics(metrics)
        }
    }
}
```

### Phase 2: Integrate Web UI Server ✓
**File:** `cmd/server/main.go`

Add Web UI startup:
```go
import "github.com/thelastdreamer/MultiWANBond/pkg/webui"

func runServer() {
    // ... existing code ...

    // Start Web UI
    webConfig := webui.DefaultConfig()
    webConfig.ListenPort = 8080
    webServer := webui.NewServer(webConfig)

    if err := webServer.Start(); err != nil {
        log.Printf("Warning: Failed to start Web UI: %v", err)
    } else {
        log.Printf("Web UI available at: http://localhost:8080")
    }

    // ... rest of code ...
}
```

### Phase 3: Create Metrics Bridge ✓
**New File:** `pkg/webui/bridge.go`

```go
package webui

import (
    "github.com/thelastdreamer/MultiWANBond/pkg/bonder"
    "github.com/thelastdreamer/MultiWANBond/pkg/protocol"
)

type MetricsBridge struct {
    bonder *bonder.Bonder
    server *Server
}

func NewMetricsBridge(b *bonder.Bonder, s *Server) *MetricsBridge {
    return &MetricsBridge{
        bonder: b,
        server: s,
    }
}

func (mb *MetricsBridge) Start() {
    go mb.updateLoop()
}

func (mb *MetricsBridge) updateLoop() {
    ticker := time.NewTicker(1 * time.Second)
    for range ticker.C {
        // Get metrics from bonder
        metrics := mb.bonder.GetMetrics()
        wans := mb.bonder.GetWANs()

        // Update Web UI stats
        mb.server.UpdateStats(metrics, wans)
    }
}
```

### Phase 4: Add Methods to Web UI Server ✓
**File:** `pkg/webui/server.go`

Add data update methods:
```go
// UpdateStats updates dashboard statistics from bonder metrics
func (s *Server) UpdateStats(metrics map[uint8]*protocol.WANMetrics, wans map[uint8]*protocol.WANInterface) {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.stats.TotalWANs = len(wans)
    s.stats.ActiveWANs = 0
    s.stats.HealthyWANs = 0
    s.stats.DegradedWANs = 0
    s.stats.DownWANs = 0

    for id, wan := range wans {
        switch wan.State {
        case protocol.WANStateUp:
            s.stats.ActiveWANs++
            s.stats.HealthyWANs++
        case protocol.WANStateDegraded:
            s.stats.ActiveWANs++
            s.stats.DegradedWANs++
        case protocol.WANStateDown:
            s.stats.DownWANs++
        }

        if m, ok := metrics[id]; ok {
            s.stats.TotalPackets += m.PacketsSent + m.PacketsRecv
            s.stats.TotalBytes += m.BytesSent + m.BytesReceived
        }
    }

    s.stats.Timestamp = time.Now()
}
```

### Phase 5: Create HTML Dashboard ✓
**New File:** `webui/index.html`

Simple responsive dashboard:
```html
<!DOCTYPE html>
<html>
<head>
    <title>MultiWANBond Dashboard</title>
    <style>
        body { font-family: Arial; margin: 0; padding: 20px; background: #f0f0f0; }
        .header { background: #2c3e50; color: white; padding: 20px; border-radius: 5px; }
        .stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 15px; margin: 20px 0; }
        .card { background: white; padding: 20px; border-radius: 5px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .card h3 { margin-top: 0; color: #2c3e50; }
        .wan { border-left: 4px solid #27ae60; }
        .wan.degraded { border-left-color: #f39c12; }
        .wan.down { border-left-color: #e74c3c; }
        .metric { display: flex; justify-content: space-between; margin: 10px 0; }
        .value { font-weight: bold; }
    </style>
</head>
<body>
    <div class="header">
        <h1>MultiWANBond Dashboard</h1>
        <p>Multi-WAN Bonding & Load Balancing System</p>
    </div>

    <div class="stats">
        <div class="card">
            <h3>System Status</h3>
            <div class="metric">
                <span>Uptime:</span>
                <span class="value" id="uptime">-</span>
            </div>
            <div class="metric">
                <span>Total WANs:</span>
                <span class="value" id="total-wans">-</span>
            </div>
            <div class="metric">
                <span>Active WANs:</span>
                <span class="value" id="active-wans">-</span>
            </div>
        </div>

        <div class="card">
            <h3>Traffic</h3>
            <div class="metric">
                <span>Total Packets:</span>
                <span class="value" id="total-packets">-</span>
            </div>
            <div class="metric">
                <span>Total Bytes:</span>
                <span class="value" id="total-bytes">-</span>
            </div>
        </div>
    </div>

    <h2>WAN Interfaces</h2>
    <div id="wans" class="stats"></div>

    <script>
        function formatBytes(bytes) {
            if (bytes < 1024) return bytes + ' B';
            if (bytes < 1024*1024) return (bytes/1024).toFixed(2) + ' KB';
            if (bytes < 1024*1024*1024) return (bytes/(1024*1024)).toFixed(2) + ' MB';
            return (bytes/(1024*1024*1024)).toFixed(2) + ' GB';
        }

        function formatDuration(ms) {
            const seconds = Math.floor(ms / 1000);
            const minutes = Math.floor(seconds / 60);
            const hours = Math.floor(minutes / 60);
            const days = Math.floor(hours / 24);

            if (days > 0) return days + 'd ' + (hours % 24) + 'h';
            if (hours > 0) return hours + 'h ' + (minutes % 60) + 'm';
            if (minutes > 0) return minutes + 'm ' + (seconds % 60) + 's';
            return seconds + 's';
        }

        function updateDashboard() {
            fetch('/api/dashboard')
                .then(r => r.json())
                .then(data => {
                    if (!data.success) return;
                    const stats = data.data;

                    document.getElementById('uptime').textContent = formatDuration(stats.uptime / 1000000);
                    document.getElementById('total-wans').textContent = stats.total_wans || 0;
                    document.getElementById('active-wans').textContent = stats.active_wans || 0;
                    document.getElementById('total-packets').textContent = (stats.total_packets || 0).toLocaleString();
                    document.getElementById('total-bytes').textContent = formatBytes(stats.total_bytes || 0);
                });

            fetch('/api/wans/status')
                .then(r => r.json())
                .then(data => {
                    if (!data.success) return;
                    const wans = data.data || [];

                    const container = document.getElementById('wans');
                    container.innerHTML = wans.map(wan => `
                        <div class="card wan ${wan.status}">
                            <h3>WAN ${wan.id}: ${wan.name}</h3>
                            <div class="metric">
                                <span>Status:</span>
                                <span class="value">${wan.status}</span>
                            </div>
                            <div class="metric">
                                <span>Latency:</span>
                                <span class="value">${wan.latency_ms}ms</span>
                            </div>
                            <div class="metric">
                                <span>Packet Loss:</span>
                                <span class="value">${wan.packet_loss.toFixed(2)}%</span>
                            </div>
                            <div class="metric">
                                <span>Sent:</span>
                                <span class="value">${formatBytes(wan.bytes_sent)}</span>
                            </div>
                            <div class="metric">
                                <span>Received:</span>
                                <span class="value">${formatBytes(wan.bytes_received)}</span>
                            </div>
                        </div>
                    `).join('');
                });
        }

        // Update every 2 seconds
        updateDashboard();
        setInterval(updateDashboard, 2000);
    </script>
</body>
</html>
```

### Phase 6: Fix Type Display ✓
**Issue:** Type shows "Unknown" instead of "ethernet"

**File:** `pkg/protocol/types.go` or `cmd/server/main.go`

Add proper type string conversion:
```go
func (t WANType) String() string {
    switch t {
    case WANTypeADSL:
        return "ADSL"
    case WANTypeVDSL:
        return "VDSL"
    case WANTypeFiber:
        return "Fiber"
    // ... etc
    default:
        return "Ethernet"
    }
}
```

### Phase 7: Test Traffic Generator ✓
**New File:** `cmd/test/traffic_test.go`

Simple UDP ping-pong for testing:
```go
package main

import (
    "fmt"
    "net"
    "time"
)

func main() {
    // Connect to local bonder
    conn, err := net.Dial("udp", "localhost:9000")
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    // Send test packets
    for i := 0; i < 100; i++ {
        msg := fmt.Sprintf("test packet %d", i)
        conn.Write([]byte(msg))
        time.Sleep(100 * time.Millisecond)
    }

    fmt.Println("Sent 100 test packets")
}
```

## Testing Plan

### Test 1: Web UI Access
```bash
# Start server
multiwanbond --config config.json

# Open browser
http://localhost:8080
```

**Expected:** Dashboard loads, shows system stats

### Test 2: Statistics Not Spamming
```bash
# Watch console output for 1 minute
```

**Expected:** Stats only print when values change

### Test 3: WAN Status Updates
```bash
# Disconnect one WAN (unplug network cable)
# Watch dashboard
```

**Expected:** WAN status changes from "up" to "down"

### Test 4: Server-Client Connection
```bash
# Terminal 1: Start server
multiwanbond --config server_config.json

# Terminal 2: Start client
multiwanbond --config client_config.json

# Check both Web UIs
http://server:8080
http://client:8080
```

**Expected:** Both show active connection and traffic

## Deployment Steps

### Step 1: Commit Changes
```bash
git add .
git commit -m "Integrate Web UI and fix statistics"
git push
```

### Step 2: Build
```bash
# Windows
go build -o multiwanbond.exe cmd/server/main.go

# Linux
go build -o multiwanbond cmd/server/main.go
```

### Step 3: Deploy
```bash
# Copy to installation directories
# Restart services
```

## Expected Results

After implementation:

1. ✅ Console shows stats only when changed
2. ✅ Web UI accessible at :8080
3. ✅ Dashboard shows real-time metrics
4. ✅ WAN states update correctly
5. ✅ Types display correctly ("Ethernet" not "Unknown")
6. ✅ Both server and client have working Web UI
7. ✅ Can manage configuration via Web UI
8. ✅ WebSocket updates push real-time changes

## File Modifications Summary

- **Modified:** `cmd/server/main.go` (add Web UI, fix stats)
- **Modified:** `pkg/webui/server.go` (add UpdateStats method)
- **Created:** `pkg/webui/bridge.go` (metrics bridge)
- **Created:** `webui/index.html` (dashboard HTML)
- **Modified:** `pkg/protocol/types.go` (fix String() methods)
- **Created:** `cmd/test/traffic_test.go` (testing tool)

## Time Estimate

- Phase 1-2: 30 minutes (integrate Web UI)
- Phase 3-4: 30 minutes (metrics bridge)
- Phase 5: 30 minutes (HTML dashboard)
- Phase 6-7: 20 minutes (fixes and testing)
- **Total: ~2 hours**

## Priority Order

1. **HIGH:** Fix statistics spam (immediate UX improvement)
2. **HIGH:** Integrate Web UI (core feature)
3. **MEDIUM:** Create metrics bridge (makes UI useful)
4. **MEDIUM:** HTML dashboard (user-facing)
5. **LOW:** Test traffic generator (nice to have)

Let's proceed with implementation!

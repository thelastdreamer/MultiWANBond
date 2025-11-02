# MultiWANBond Web UI User Guide

**Complete end-user guide for the MultiWANBond Web Interface**

**Version**: 1.2
**Last Updated**: November 2, 2025

---

## Table of Contents

- [Getting Started](#getting-started)
- [Login](#login)
- [Dashboard](#dashboard)
- [Flows Page](#flows-page)
- [Analytics Page](#analytics-page)
- [Logs Page](#logs-page)
- [Configuration Page](#configuration-page)
- [Common Tasks](#common-tasks)
- [Understanding Metrics](#understanding-metrics)
- [Troubleshooting](#troubleshooting)
- [Keyboard Shortcuts](#keyboard-shortcuts)

---

## Getting Started

### Accessing the Web UI

1. **Ensure MultiWANBond is running**:
   ```bash
   multiwanbond start
   ```

2. **Open your web browser** and navigate to:
   ```
   http://localhost:8080
   ```

   Or if accessing remotely:
   ```
   http://<server-ip-address>:8080
   ```

3. **You will be redirected to the login page**

### System Requirements

**Browser Requirements**:
- Modern web browser (Chrome 90+, Firefox 88+, Edge 90+, Safari 14+)
- JavaScript enabled
- WebSocket support (for real-time updates)
- Minimum screen resolution: 1280x720

**Network Requirements**:
- Access to MultiWANBond server on port 8080 (default)
- Stable connection for WebSocket updates

---

## Login

### First Time Login

![Login Page](webui/login.html)

**Default Credentials**:
- **Username**: `admin`
- **Password**: `MultiWAN2025Secure!`

**⚠️ Security Best Practice**: Change the default password immediately after first login!

### Login Process

1. Enter your **username** in the first field
2. Enter your **password** in the second field
3. Click **Login** button or press **Enter**

**Session Information**:
- Sessions last **24 hours** from login
- You'll be automatically logged out after 24 hours
- Your session is checked every 5 minutes
- Closing the browser **does not** log you out (session persists)

### Login Errors

**"Invalid credentials"**:
- Check username and password spelling
- Ensure Caps Lock is off
- Verify credentials in your `config.json` file

**"Connection failed"**:
- Ensure MultiWANBond server is running
- Check firewall settings (port 8080)
- Verify server address is correct

**"Session expired"**:
- Your 24-hour session has expired
- Simply log in again

---

## Dashboard

The **Dashboard** is your main monitoring screen, providing real-time overview of your entire MultiWANBond system.

### Dashboard Layout

```
┌────────────────────────────────────────────────────────────────┐
│  [Navigation Bar]                              [Alerts] [Logout]│
├────────────────────────────────────────────────────────────────┤
│  System Metrics Row                                            │
│  [Uptime: 24h] [Total Traffic: 500GB] [Current Speed: 125Mbps]│
├────────────────────────────────────────────────────────────────┤
│  WAN Interface Cards                                           │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                    │
│  │  WAN 1   │  │  WAN 2   │  │  WAN 3   │                    │
│  │  Fiber   │  │Starlink  │  │   LTE    │                    │
│  │ ✓HEALTHY │  │ ✓HEALTHY │  │ ✗ DOWN   │                    │
│  └──────────┘  └──────────┘  └──────────┘                    │
├────────────────────────────────────────────────────────────────┤
│  NAT Status                    │  Active Alerts               │
│  Type: Full Cone NAT           │  ⚠️ WAN 3 Down               │
│  Public IP: 203.0.113.45       │  ⚠️ High Latency (WAN 2)     │
│  CGNAT: No                     │                               │
├────────────────────────────────────────────────────────────────┤
│  Top 10 Active Flows                                           │
│  [Flow table with protocol classification]                    │
└────────────────────────────────────────────────────────────────┘
```

### System Metrics (Top Row)

**Uptime**:
- Shows how long MultiWANBond has been running
- Format: Days, Hours, Minutes (e.g., "2d 5h 30m")
- Resets when you restart MultiWANBond

**Total Traffic**:
- Combined traffic across all WANs (sent + received)
- Shows data transferred since MultiWANBond started
- Updates in real-time

**Current Speed**:
- Real-time aggregate bandwidth usage
- Measured in Mbps (megabits per second)
- Updates every second

### WAN Interface Cards

Each WAN interface is displayed in its own card with color-coded status:

**Healthy WAN** (Green Border):
```
┌────────────────────────┐
│ WAN 1: Fiber          │
│ ✓ HEALTHY             │
│                        │
│ Latency: 5.2ms        │
│ Jitter: 0.8ms         │
│ Loss: 0.01%           │
│                        │
│ ⬆ 2.5 GB              │
│ ⬇ 5.0 GB              │
│                        │
│ Weight: 100           │
│ Last Check: 2s ago    │
└────────────────────────┘
```

**Failed WAN** (Red Border):
```
┌────────────────────────┐
│ WAN 3: LTE            │
│ ✗ FAILED              │
│                        │
│ Latency: --           │
│ Jitter: --            │
│ Loss: 100%            │
│                        │
│ ⬆ 0 B                 │
│ ⬇ 0 B                 │
│                        │
│ Weight: 50            │
│ Last Check: 5s ago    │
└────────────────────────┘
```

**Card Information**:
- **Name**: WAN interface name (configured during setup)
- **Status**: HEALTHY (green), DEGRADED (yellow), FAILED (red)
- **Latency**: Round-trip time to health check target
- **Jitter**: Variation in latency (lower is better)
- **Packet Loss**: Percentage of lost packets (0% is ideal)
- **Uploaded/Downloaded**: Total bytes sent/received on this WAN
- **Weight**: Traffic distribution weight (higher = more traffic)
- **Last Check**: Time since last health check

### NAT Status Panel

Shows your NAT traversal information:

**NAT Type**:
- **Open (No NAT)**: Best - direct internet connection
- **Full Cone NAT**: Excellent - easy peer-to-peer
- **Restricted Cone NAT**: Good - moderate traversal difficulty
- **Port-Restricted Cone NAT**: Fair - more difficult traversal
- **Symmetric NAT**: Poor - requires relay for P2P
- **Blocked**: UDP completely filtered
- **Unknown**: Detection in progress

**Public Address**:
- Your public IP and port as seen from the internet
- Format: `IP:PORT` (e.g., 203.0.113.45:12345)

**CGNAT Detected**:
- **No**: Standard NAT, good for P2P
- **Yes**: Carrier-Grade NAT (double NAT), may need relay

**Connection Capability**:
- **Can Direct Connect**: Green checkmark if P2P possible
- **Needs Relay**: Yellow warning if relay required

### Active Alerts Panel

Shows recent system alerts and warnings:

**Alert Types**:
- 🔴 **Error**: Critical issues (WAN down, connection lost)
- ⚠️ **Warning**: Issues needing attention (high latency, packet loss)
- ℹ️ **Info**: Informational messages (WAN recovered, config changed)

**Example Alerts**:
```
⚠️ High Latency Detected
   WAN 2 (Starlink) latency increased to 150ms
   Threshold: 100ms | 2 minutes ago

🔴 WAN Down
   WAN 3 (LTE) is down - all health checks failing
   5 minutes ago

ℹ️ WAN Recovered
   WAN 1 (Fiber) has recovered and is now active
   10 minutes ago
```

**Alert Actions**:
- Click **View All** to see full alert history
- Click **Clear** to dismiss all alerts
- Alerts auto-clear when issue is resolved

### Top 10 Active Flows

Shows the most active network connections:

| Source | Destination | Protocol | Application | Bytes | WAN |
|--------|-------------|----------|-------------|-------|-----|
| 192.168.1.100:52341 | 142.250.185.46:443 | HTTPS | YouTube | 2.1 MB | 1 |
| 192.168.1.100:52342 | 8.8.8.8:53 | DNS | DNS | 128 B | 2 |

**Columns**:
- **Source**: Local IP and port
- **Destination**: Remote IP and port
- **Protocol**: Transport protocol (TCP/UDP) and application protocol
- **Application**: Classified application (YouTube, Netflix, etc.)
- **Bytes**: Total data transferred in this flow
- **WAN**: Which WAN interface is carrying this flow

**Flow Colors**:
- Streaming (YouTube, Netflix): Purple
- Web (HTTP, HTTPS): Blue
- Gaming: Green
- VoIP: Orange
- File Transfer: Red

### Real-Time Updates

The Dashboard **automatically updates** every **1 second** via WebSocket:
- WAN status changes instantly
- Traffic metrics update in real-time
- New alerts appear immediately
- No manual refresh needed

**Visual Indicators**:
- ✓ Green checkmark: Healthy
- ⚠️ Yellow warning: Degraded
- ✗ Red X: Failed
- 🔄 Spinning: Checking

---

## Flows Page

The **Flows** page provides detailed network flow analysis with Deep Packet Inspection (DPI) classification.

### Flow Statistics (Top Bar)

```
┌─────────────────────────────────────────────────────────┐
│ Total Flows: 142  │  Active: 38  │  Total Traffic: 2.5GB │
│ Top Protocol: HTTPS (45%)                               │
└─────────────────────────────────────────────────────────┘
```

**Metrics**:
- **Total Flows**: All flows seen since startup
- **Active Flows**: Currently active connections
- **Total Traffic**: Combined data across all flows
- **Top Protocol**: Most used protocol by bytes

### Filter Bar

Filter flows by multiple criteria:

**Search Box**:
- Search by IP address (e.g., "192.168.1.100")
- Search by port (e.g., "443")
- Search by hostname (e.g., "youtube.com")

**Protocol Filter**:
- Dropdown with all detected protocols
- Options: All, HTTP, HTTPS, YouTube, Netflix, DNS, etc.
- Updates based on actual traffic

**WAN Filter**:
- Filter by which WAN is carrying the flow
- Options: All WANs, WAN 1, WAN 2, WAN 3, etc.

**Refresh Controls**:
- **Auto-Refresh**: Toggle on/off (default: ON, every 5 seconds)
- **Refresh Now**: Manual refresh button

### Flow Table

Detailed 8-column table with all active flows:

| Src IP | Src Port | Dst IP | Dst Port | Protocol | App | Bytes | WAN | Duration | Status |
|--------|----------|--------|----------|----------|-----|-------|-----|----------|--------|
| 192.168.1.100 | 52341 | 142.250.185.46 | 443 | **HTTPS** | **YouTube** | ⬆ 1.2MB ⬇ 2.1MB | WAN 1 | 45s | Active |

**Column Details**:

1. **Src IP**: Source (local) IP address
2. **Src Port**: Source port number
3. **Dst IP**: Destination (remote) IP address
4. **Dst Port**: Destination port number
5. **Protocol**: Color-coded protocol badge
6. **Application**: Detected application (via DPI)
7. **Bytes**: Upload ⬆ and Download ⬇ separately
8. **WAN**: Which WAN interface
9. **Duration**: How long the flow has been active
10. **Status**: Active / Closed / Timeout

**Protocol Color Coding**:
- **HTTPS**: Blue badge
- **YouTube**: Red badge
- **Netflix**: Red badge
- **Gaming**: Green badge
- **VoIP**: Orange badge
- **File Transfer**: Purple badge
- **DNS**: Gray badge

### Understanding Flow Data

**What is a Flow?**
- A network connection between your device and a remote server
- Identified by: Source IP, Source Port, Destination IP, Destination Port, Protocol
- Example: Your browser connecting to YouTube

**How DPI Works**:
1. First few packets of flow are analyzed
2. Pattern matching identifies the protocol
3. Application is classified (YouTube, Netflix, etc.)
4. Category is assigned (Streaming, Web, Gaming, etc.)
5. Classification appears within milliseconds

**Supported Applications** (40+ protocols):
- **Web**: HTTP, HTTPS, WebSocket
- **Streaming**: YouTube, Netflix, Twitch, Spotify, etc.
- **Gaming**: Steam, Epic, Minecraft, League of Legends, etc.
- **Communication**: Zoom, Teams, Discord, Telegram, etc.
- **Social**: Facebook, Instagram, Twitter, TikTok, etc.
- **File Transfer**: BitTorrent, Dropbox, Google Drive, etc.

### Common Use Cases

**Finding Bandwidth Hogs**:
1. Sort by "Bytes" column (click header)
2. Top entries are using most bandwidth
3. Check protocol to see what type of traffic

**Monitoring Specific Applications**:
1. Use protocol filter dropdown
2. Select application (e.g., "YouTube")
3. See all YouTube traffic across all WANs

**Checking WAN Distribution**:
1. Look at WAN column
2. Verify traffic is balanced across WANs
3. If not, check load balancing mode in Configuration

**Troubleshooting Slow Connections**:
1. Find the slow connection in the flow table
2. Check which WAN it's using
3. Check that WAN's latency and packet loss
4. Consider policy routing if needed

---

## Analytics Page

The **Analytics** page provides visual insights into your network traffic with interactive charts.

### Time Range Selector

Choose the time period for analysis:
- **1H**: Last 1 hour (most detailed)
- **6H**: Last 6 hours
- **24H**: Last 24 hours (default)
- **7D**: Last 7 days
- **30D**: Last 30 days

**Note**: Longer time ranges show aggregated data (less detail).

### Key Metrics Cards

Four summary cards at the top:

```
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ 24h Traffic      │  │ Avg Latency      │  │ Packet Loss      │  │ Active Conns     │
│ 245.8 GB         │  │ 15.2 ms          │  │ 0.15%            │  │ 38               │
│ ⬆ 102.3 ⬇ 143.5  │  │ (across all WANs)│  │ (across all WANs)│  │ (current)        │
└──────────────────┘  └──────────────────┘  └──────────────────┘  └──────────────────┘
```

### Chart 1: Traffic Over Time (Line Chart)

**Shows**: Upload and download traffic trends

```
Traffic (Mbps)
150 │              ⬇ Download
    │         ┌──────┐
100 │    ┌────┘      └───┐
    │    │                │
 50 │────┘                └────
    │  ⬆ Upload
  0 └──────────────────────────
    0h    6h    12h   18h   24h
```

**How to Read**:
- **Blue line**: Download traffic
- **Green line**: Upload traffic
- **Peaks**: High usage periods
- **Valleys**: Low usage periods

**Insights**:
- Identify peak usage times
- Plan upgrades based on trends
- Spot unusual traffic spikes
- Verify bandwidth expectations

### Chart 2: Per-WAN Distribution (Doughnut Chart)

**Shows**: How traffic is distributed across WANs

```
        WAN 1 (Fiber)
           50%
      ┌─────────┐
WAN 3 │         │ WAN 2
 10%  │   PIE   │ (Starlink)
      │  CHART  │  40%
      └─────────┘
```

**How to Read**:
- Each slice represents one WAN
- Percentage = proportion of total traffic
- Hover over slice for exact bytes

**Expected Distribution**:
- Roughly matches WAN weights
- Example: Weight 100, 50, 30 = ~56%, 28%, 16%
- Failed WANs show 0%

**If Unbalanced**:
- Check WAN health (latency, packet loss)
- Review load balancing mode
- Verify weights in configuration
- Check for policy routing overrides

### Chart 3: WAN Latency Comparison (Bar Chart)

**Shows**: Average latency for each WAN

```
Latency (ms)
100 │
    │
 50 │    ██          ██
    │    ██    ██    ██
  0 └────██────██────██────
       WAN 1 WAN 2 WAN 3
        5ms   25ms  50ms
```

**How to Read**:
- **Green bars**: Healthy (<50ms)
- **Yellow bars**: Acceptable (50-100ms)
- **Red bars**: High (>100ms)
- Lower is better for responsiveness

**Good Latency**:
- Gaming: <30ms ideal
- Video calls: <50ms ideal
- Web browsing: <100ms acceptable
- File transfer: Latency less critical

### Chart 4: Protocol Breakdown (Doughnut Chart)

**Shows**: Traffic distribution by application protocol

```
        HTTPS
          45%
      ┌─────────┐
YouTube│        │ HTTP
  30%  │  PIE   │  15%
       │ CHART  │
       └─────────┘
         Other 10%
```

**How to Read**:
- Each slice is one protocol/application
- Helps understand what you're using bandwidth for
- Top 5 protocols shown, rest grouped as "Other"

**Common Patterns**:
- **Work from home**: High Zoom/Teams, moderate Web
- **Streaming**: High YouTube/Netflix
- **Gaming**: High Steam/Epic, moderate gaming protocols
- **General use**: High HTTPS (encrypted web traffic)

### Auto-Refresh

Analytics page **auto-refreshes every 10 seconds**:
- Charts update with latest data
- Smooth transitions between data points
- Can disable auto-refresh if needed

### Exporting Data

**Export Options** (planned for v1.2):
- Download charts as PNG images
- Export raw data as CSV
- Generate PDF report

---

## Logs Page

The **Logs** page provides a terminal-style system event viewer.

### Log Statistics Bar

```
┌─────────────────────────────────────────────────────────┐
│ Total: 1,523 │ Info: 1,245 │ Warnings: 245 │ Errors: 33 │
└─────────────────────────────────────────────────────────┘
```

### Filter Controls

**Log Level Filter**:
- **All**: Show all log levels
- **Debug**: Detailed debugging information
- **Info**: General informational messages
- **Warning**: Warning conditions
- **Error**: Error conditions

**Search Box**:
- Search for specific text in logs
- Case-insensitive
- Filters in real-time as you type

**Control Buttons**:
- **Clear**: Clear all logs from display
- **Export**: Download logs as .txt file
- **Refresh**: Manual refresh
- **Auto-Scroll**: Toggle auto-scroll to bottom (on by default)

### Log Display

Terminal-style dark theme with color-coded log levels:

```
[2025-11-02 14:30:00] INFO  WAN 1 health check successful (latency: 5.2ms)
[2025-11-02 14:29:55] WARN  WAN 2 latency increased to 150ms (threshold: 100ms)
[2025-11-02 14:29:50] ERROR WAN 3 health check failed (timeout)
[2025-11-02 14:29:45] DEBUG Packet processed: seq=12345 wan=1 size=1500
```

**Color Coding**:
- **DEBUG**: Gray text (detailed diagnostics)
- **INFO**: Green text (normal operations)
- **WARN**: Yellow text (warnings)
- **ERROR**: Red text (errors)

### Understanding Log Messages

**Common Log Patterns**:

**Health Check Success**:
```
[2025-11-02 14:30:00] INFO WAN 1 health check successful (latency: 5.2ms)
```
- Regular health check passed
- Shows current latency
- Appears every 5 seconds per WAN

**Health Check Failure**:
```
[2025-11-02 14:29:50] ERROR WAN 3 health check failed (timeout)
```
- Health check did not respond in time
- May indicate WAN is down
- Check network connection

**WAN State Change**:
```
[2025-11-02 14:29:45] WARN WAN 3 state changed: active -> down
```
- WAN failed after multiple failed health checks
- Traffic will be rerouted to healthy WANs
- Investigate WAN 3 connectivity

**High Latency Warning**:
```
[2025-11-02 14:29:40] WARN WAN 2 latency increased to 150ms (threshold: 100ms)
```
- Latency exceeded configured threshold
- WAN still functional but slower
- May affect user experience

**Packet Loss Warning**:
```
[2025-11-02 14:29:35] WARN WAN 1 packet loss: 5.2% (threshold: 5%)
```
- Packet loss above threshold
- May cause slow downloads
- Check network quality

**Configuration Change**:
```
[2025-11-02 14:29:30] INFO Configuration reloaded successfully
```
- Settings were changed and applied
- No restart required
- New settings now active

### Log Export

Click **Export** to download logs:
- Filename: `multiwanbond-logs-<timestamp>.txt`
- Contains all visible logs (respects current filters)
- Plain text format
- Can be viewed in any text editor

### Auto-Refresh

Logs page **auto-refreshes every 3 seconds**:
- New logs appear automatically
- Auto-scrolls to bottom if enabled
- No manual refresh needed

---

## Configuration Page

The **Configuration** page allows you to modify system settings through the Web UI.

⚠️ **Important**: Configuration changes require MultiWANBond restart to take effect.

### WAN Interfaces Section

Manage your WAN connections:

**WAN List**:
```
┌────────────────────────────────────────────────────┐
│ ID │ Name      │ Interface │ Enabled │ Weight │    │
├────┼───────────┼───────────┼─────────┼────────┼────┤
│ 1  │ Fiber     │ eth0      │    ✓    │  100   │[✎] │
│ 2  │ Starlink  │ wwan0     │    ✓    │   50   │[✎] │
│ 3  │ LTE       │ wwan1     │    ✗    │   30   │[✎] │
└────────────────────────────────────────────────────┘
[Add New WAN]
```

**Adding a WAN**:
1. Click **Add New WAN** button
2. Fill in WAN details:
   - **Name**: Friendly name (e.g., "Fiber", "LTE")
   - **Interface**: Network interface (e.g., "eth0", "wwan0")
   - **Weight**: Traffic distribution weight (1-1000)
   - **Enabled**: Check to enable immediately
3. Click **Save**
4. Restart MultiWANBond to apply

**Editing a WAN**:
1. Click **Edit** (pencil icon) next to WAN
2. Modify fields as needed
3. Click **Save**
4. Restart MultiWANBond to apply

**Deleting a WAN**:
1. Click **Edit** button
2. Click **Delete** at bottom of form
3. Confirm deletion
4. Restart MultiWANBond to apply

### Routing Policies Section

Manage policy-based routing to route specific traffic through specific WANs:

**Policy List**:
```
┌────────────────────────────────────────────────────────────────────┐
│ Name              │ Type        │ Match          │ WAN │ Priority │  │
├───────────────────┼─────────────┼────────────────┼─────┼──────────┼──┤
│ Video Streaming   │ destination │ 8.8.8.8/32     │  1  │   100    │[✗]│
│ Work VPN          │ source      │ 192.168.1.0/24 │  2  │   200    │[✗]│
│ Netflix           │ application │ Netflix        │  1  │   300    │[✗]│
└────────────────────────────────────────────────────────────────────┘
[Add Routing Policy]
```

**Policy Types**:

1. **Source-based**:
   - Routes traffic from specific source networks
   - Example: Route all traffic from `192.168.1.0/24` via WAN 2
   - Match field: Source IP or CIDR (e.g., `192.168.1.50/32`, `10.0.0.0/8`)

2. **Destination-based**:
   - Routes traffic to specific destinations
   - Example: Route all traffic to `8.8.8.8` via WAN 1
   - Match field: Destination IP or CIDR (e.g., `8.8.8.8/32`, `1.1.1.0/24`)

3. **Application-based**:
   - Routes traffic for specific applications (requires DPI)
   - Example: Route all Netflix traffic via WAN 1
   - Match field: Application name (e.g., `Netflix`, `YouTube`, `Zoom`)
   - ⚠️ Requires Deep Packet Inspection (DPI) enabled

**Adding a Routing Policy**:
1. Click **Add Routing Policy** button
2. Fill in policy details:
   - **Policy Name**: Descriptive name (e.g., "Video Streaming")
   - **Description**: Optional description
   - **Policy Type**: Select source, destination, or application
   - **Source/Destination/Application**: Depends on type selected
     - Source: Enter source IP or network (e.g., `192.168.1.0/24`)
     - Destination: Enter destination IP or network (e.g., `8.8.8.8/32`)
     - Application: Enter application name (e.g., `Netflix`, `YouTube`)
   - **Target WAN ID**: WAN to route matched traffic through (1-255)
   - **Priority**: Rule priority - lower numbers = higher priority (0-1000)
   - **Enabled**: Check to activate immediately
3. Click **Save Policy**
4. Restart MultiWANBond to apply changes

**Priority and Evaluation Order**:
- Policies are evaluated in **priority order** (lower number = evaluated first)
- First matching policy wins
- If no policy matches, traffic uses load balancing mode
- **Recommended priority spacing**: Use multiples of 100 (100, 200, 300...) to allow inserting policies later

**Example Use Cases**:

**Route work VPN through dedicated WAN**:
```
Name: Work VPN
Type: Source-based
Match: 192.168.1.0/24
Target WAN: 2
Priority: 100
```

**Route video streaming to low-latency WAN**:
```
Name: Video Streaming
Type: Destination-based
Match: 8.8.8.8/32
Target WAN: 1
Priority: 200
```

**Route Netflix through high-bandwidth WAN**:
```
Name: Netflix Traffic
Type: Application-based
Match: Netflix
Target WAN: 1
Priority: 300
(Requires DPI enabled)
```

**Deleting a Policy**:
1. Click **Delete** button next to policy
2. Confirm deletion
3. Restart MultiWANBond to apply

**Best Practices**:
- Use descriptive policy names
- Start priority at 100, increment by 100
- Test policies after adding
- Document your policies in the Description field
- Avoid overlapping policies with same priority
- Use source-based for client routing
- Use destination-based for server/service routing
- Use application-based for protocol-specific routing (requires DPI)

⚠️ **Important**: All policy changes require a MultiWANBond restart to take effect!

### Load Balancing Section

Choose how traffic is distributed:

**Load Balancing Modes**:

1. **Round-Robin**:
   - Simple rotation through WANs
   - Even distribution regardless of capacity
   - Best for: Equal WANs

2. **Weighted**:
   - Distributes based on WAN weight
   - Higher weight = more traffic
   - **Best for: WANs with different speeds** ✓ Recommended

3. **Least-Used**:
   - Routes to WAN with lowest current usage
   - Balances actual load
   - Best for: Highly variable traffic

4. **Least-Latency**:
   - Routes to WAN with lowest latency
   - Optimizes for responsiveness
   - Best for: Latency-sensitive applications

5. **Per-Flow**:
   - Sticky routing per connection
   - Maintains packet order
   - Best for: Applications requiring in-order delivery

6. **Adaptive**:
   - Combines weight, latency, loss, and usage
   - Dynamic adjustment
   - **Best for: Most scenarios** ✓ Recommended

### Health Check Section

Configure WAN health monitoring:

**Settings**:
- **Check Interval**: How often to check (1000-10000ms)
- **Timeout**: How long to wait for response (500-5000ms)
- **Retry Count**: Failures before marking WAN down (1-10)
- **Check Hosts**: IP addresses to ping (comma-separated)

**Recommended Values**:
```
Check Interval: 5000ms (5 seconds)
Timeout: 3000ms (3 seconds)
Retry Count: 3
Check Hosts: 8.8.8.8, 1.1.1.1
```

**Advanced Settings**:
- **Adaptive Intervals**: Automatically adjust check frequency based on stability
- **Check Method**: ICMP, HTTP, TCP, or DNS
- **Failure Threshold**: Packet loss % before warning (default: 5%)
- **Latency Threshold**: Latency before warning (default: 100ms)

### Security Section

Configure encryption and authentication:

**Encryption**:
- **Enabled**: Turn encryption on/off
- **Type**: ChaCha20-Poly1305 (recommended) or AES-256-GCM
- **Pre-Shared Key**: Secret key (minimum 16 characters)

**Web UI Security**:
- **Username**: Login username
- **Password**: Login password (minimum 8 characters)
- **Session Timeout**: Session duration in hours (default: 24)

⚠️ **Never commit secrets to Git or share your config file publicly!**

### Saving Configuration

1. Make your changes in any section
2. Click **Save Configuration** button at bottom
3. Confirmation message appears
4. **Restart MultiWANBond** for changes to take effect:
   ```bash
   multiwanbond restart
   ```

### Exporting/Importing Configuration

**Export**:
1. Click **Export Config** button
2. Save `config.json` file
3. Use for backup or sharing (remove secrets first!)

**Import**:
1. Click **Import Config** button
2. Select `config.json` file
3. Configuration is validated
4. Click **Apply** to use new config
5. Restart MultiWANBond

---

## Common Tasks

### Checking if All WANs are Healthy

1. Go to **Dashboard**
2. Look at WAN interface cards
3. All should show **✓ HEALTHY** in green
4. If any show **✗ FAILED** in red:
   - Check physical connection
   - Check ISP service status
   - View logs for error details

### Finding Out Why Traffic is Slow

1. **Check Dashboard** WAN cards:
   - High latency? (>100ms)
   - High jitter? (>50ms)
   - Packet loss? (>5%)

2. **Go to Analytics**:
   - Check latency comparison chart
   - Identify problematic WAN

3. **Go to Logs**:
   - Search for the WAN name
   - Look for errors or warnings

4. **Possible Solutions**:
   - Temporarily disable problematic WAN
   - Contact ISP if issue persists
   - Adjust WAN weights to favor better WANs

### Monitoring Specific Application Traffic

1. Go to **Flows** page
2. Use **Protocol Filter** dropdown
3. Select application (e.g., "YouTube")
4. View all flows for that application
5. Check which WANs are being used
6. Verify traffic distribution

### Adjusting Traffic Distribution

1. Go to **Configuration** page
2. Find **WAN Interfaces** section
3. Adjust **Weight** values:
   - Higher weight = more traffic
   - Example: 100, 50, 25 = 57%, 29%, 14%
4. Click **Save Configuration**
5. Restart MultiWANBond

### Temporarily Disabling a WAN

**Via Web UI**:
1. Go to **Configuration** page
2. Find the WAN in the list
3. Click **Edit** (pencil icon)
4. Uncheck **Enabled** checkbox
5. Click **Save**
6. Restart MultiWANBond

**Via CLI** (faster):
```bash
multiwanbond wan disable <wan-id>
```

### Clearing Alerts

1. Go to **Dashboard**
2. In the **Active Alerts** panel
3. Click **Clear All** button
4. Alerts are dismissed
5. New alerts will appear as issues occur

### Exporting Logs for Troubleshooting

1. Go to **Logs** page
2. Set **Log Level** to "All"
3. Optional: Set date/time range
4. Click **Export** button
5. Save `.txt` file
6. Share with support or review offline

---

## Understanding Metrics

### Latency

**What it is**: Round-trip time for a packet to reach destination and return

**Units**: Milliseconds (ms)

**Good Values**:
- Gaming: <30ms ideal, <50ms acceptable
- Video calls: <50ms ideal, <100ms acceptable
- Web browsing: <100ms ideal, <200ms acceptable
- File downloads: Less critical

**What Affects It**:
- Physical distance to server
- ISP routing
- Network congestion
- WAN technology (Fiber < Cable < DSL < Satellite)

### Jitter

**What it is**: Variation in latency over time

**Units**: Milliseconds (ms)

**Good Values**:
- Gaming: <10ms ideal
- Video calls: <30ms ideal
- General use: <50ms acceptable

**High Jitter Symptoms**:
- Choppy video calls
- Inconsistent gaming performance
- Buffering in streams

### Packet Loss

**What it is**: Percentage of packets that don't reach destination

**Units**: Percentage (%)

**Good Values**:
- Ideal: 0%
- Acceptable: <1%
- Poor: >5%
- Critical: >10%

**High Packet Loss Symptoms**:
- Slow downloads
- Video stuttering
- Connection timeouts
- Retransmissions

### Bandwidth

**What it is**: Amount of data transferred per second

**Units**: Mbps (megabits per second), GB (gigabytes)

**Understanding Values**:
- 1 Mbps = 125 KB/s
- 10 Mbps = 1.25 MB/s
- 100 Mbps = 12.5 MB/s
- 1000 Mbps = 125 MB/s

**Typical Usage**:
- Web browsing: 1-5 Mbps
- HD video (1080p): 5-8 Mbps
- 4K video: 25-50 Mbps
- Video calls: 2-4 Mbps
- Gaming: 1-3 Mbps (but latency matters more)

### Uptime

**What it is**: Percentage of time WAN has been available

**Calculation**: (time_healthy / total_time) × 100

**Target Values**:
- Consumer: >95%
- Business: >99%
- Enterprise: >99.9% ("three nines")

---

## Troubleshooting

### Dashboard Shows "Connection Lost"

**Cause**: WebSocket connection to server failed

**Solutions**:
1. Check if MultiWANBond is still running
2. Refresh the page (F5)
3. Check network connection
4. Check firewall/proxy settings
5. Verify server address is correct

### WAN Card Shows "Failed" But Internet Works

**Cause**: Health check target is unreachable, but WAN is functional

**Solutions**:
1. Check **health check hosts** in Configuration
2. Verify DNS is working (`nslookup 8.8.8.8`)
3. Check if health check hosts are blocked
4. Try different health check method (ICMP vs HTTP)

### No Traffic on a WAN Despite Being Healthy

**Causes**:
- Weight set to 0
- Load balancing mode not using this WAN
- Policy routing override

**Solutions**:
1. Check WAN weight in Configuration (should be >0)
2. Verify load balancing mode (try "Weighted" or "Adaptive")
3. Check for policy routing rules
4. Review logs for routing decisions

### Charts Not Updating

**Causes**:
- Auto-refresh disabled
- Browser issue
- WebSocket connection lost

**Solutions**:
1. Check auto-refresh is ON
2. Manually refresh (click Refresh button)
3. Hard refresh page (Ctrl+F5 or Cmd+Shift+R)
4. Check browser console for errors (F12)
5. Try different browser

### Session Keeps Expiring

**Causes**:
- Session timeout too short
- Server restarted
- Clock skew between client and server

**Solutions**:
1. Check session timeout in Configuration (default: 24h)
2. Verify server hasn't restarted (check uptime)
3. Synchronize clocks (use NTP)

### Cannot Save Configuration

**Causes**:
- Invalid values
- Missing required fields
- Permission issues

**Solutions**:
1. Check all fields have valid values
2. Look for error messages (red text)
3. Verify MultiWANBond has write permissions
4. Check disk space
5. Review server logs for errors

---

## Keyboard Shortcuts

### Global Shortcuts

| Key | Action |
|-----|--------|
| `Ctrl+/` or `Cmd+/` | Show keyboard shortcuts help |
| `Esc` | Close modals/dialogs |

### Navigation Shortcuts

| Key | Action |
|-----|--------|
| `1` | Go to Dashboard |
| `2` | Go to Flows |
| `3` | Go to Analytics |
| `4` | Go to Logs |
| `5` | Go to Configuration |
| `A` | View Alerts |
| `L` | Logout |

### Page-Specific Shortcuts

**Flows Page**:
| Key | Action |
|-----|--------|
| `F` | Focus search box |
| `R` | Refresh flows |
| `T` | Toggle auto-refresh |

**Logs Page**:
| Key | Action |
|-----|--------|
| `Ctrl+F` or `Cmd+F` | Focus search box |
| `C` | Clear logs |
| `E` | Export logs |
| `S` | Toggle auto-scroll |

**Analytics Page**:
| Key | Action |
|-----|--------|
| `1` | 1 hour time range |
| `6` | 6 hours time range |
| `2` | 24 hours time range |
| `7` | 7 days time range |
| `3` | 30 days time range |

---

## Tips and Best Practices

### Optimal Configuration

**For Home/Office**:
- Load Balancing: **Adaptive**
- Health Check Interval: **5000ms**
- Auto-Refresh: **ON**
- Log Level: **Info**

**For Servers/Production**:
- Load Balancing: **Adaptive**
- Health Check Interval: **1000ms** (faster failover)
- Encryption: **Enabled** (ChaCha20-Poly1305)
- Log Level: **Warning** (reduce noise)

### Monitoring Best Practices

1. **Check Dashboard Daily**:
   - Verify all WANs healthy
   - Review alerts
   - Check traffic patterns

2. **Review Analytics Weekly**:
   - Identify usage trends
   - Plan capacity upgrades
   - Optimize WAN weights

3. **Export Logs Monthly**:
   - Archive for historical analysis
   - Document issues and resolutions
   - Track uptime statistics

### Security Best Practices

1. **Change Default Password Immediately**
2. **Use Strong Passwords** (16+ characters, mixed case, numbers, symbols)
3. **Enable Encryption** (always in production)
4. **Restrict Web UI Access** (firewall rules)
5. **Regular Backups** (export config weekly)
6. **Update Regularly** (apply security patches)

### Performance Optimization

1. **Set Appropriate Weights**:
   - Match actual WAN speeds
   - Example: 1000Mbps fiber = 100, 100Mbps LTE = 10

2. **Use Adaptive Load Balancing**:
   - Best overall performance
   - Automatically adjusts to conditions

3. **Monitor and Tune**:
   - Watch per-WAN distribution in Analytics
   - Adjust if severely imbalanced
   - Consider policy routing for critical apps

4. **Regular Health Checks**:
   - Keep default 5-second interval
   - Use multiple check hosts for reliability
   - Enable adaptive intervals

---

## Getting Help

### In-App Help

- Hover over any setting for tooltip
- Click **?** icon for contextual help
- Error messages show specific fix suggestions

### Documentation

- [README.md](README.md) - Project overview
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Detailed troubleshooting
- [API_REFERENCE.md](API_REFERENCE.md) - API documentation
- [ARCHITECTURE.md](ARCHITECTURE.md) - Technical architecture

### Community Support

- [GitHub Issues](https://github.com/thelastdreamer/MultiWANBond/issues) - Report bugs
- [GitHub Discussions](https://github.com/thelastdreamer/MultiWANBond/discussions) - Ask questions

---

**Need More Help?** See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for comprehensive troubleshooting guide.

**Last Updated**: November 2, 2025
**Version**: 1.1
**MultiWANBond Version**: 1.1

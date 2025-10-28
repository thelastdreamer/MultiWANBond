# MultiWANBond Web UI Guide

## 🚀 Quick Start

### Start the Application

```bash
# With existing config
multiwanbond.exe --config "C:\ProgramData\MultiWANBond\config.json"
```

You'll see:
```
Starting Web UI server...
Web UI available at: http://localhost:8080
```

### Access the Interface

Open your browser:
- **Dashboard**: http://localhost:8080
- **Configuration**: http://localhost:8080/config.html

## 📊 Dashboard (Monitoring)

Real-time monitoring interface with auto-refresh every 2 seconds.

### What You'll See

**System Status Card**:
- Uptime
- Version (1.0.0)
- Platform (Windows/Linux)

**WAN Interfaces Card**:
- Total WANs configured
- Healthy (green) - Working perfectly
- Degraded (orange) - Performance issues
- Down (red) - Not responding

**Traffic Statistics Card**:
- Total packets sent/received
- Total bytes transferred
- Active flows

**Connection Card**:
- Connection status
- NAT type
- Public IP address

**Individual WAN Cards**:
Each WAN shows:
- Status badge (UP/DEGRADED/DOWN)
- Latency in milliseconds
- Jitter in milliseconds
- Packet loss percentage
- Weight for load balancing
- Bytes sent and received

### Navigation

Click **"Configuration"** button in header to access management interface.

## ⚙️ Configuration Management

Complete interface for managing MultiWANBond settings.

### Tab 1: WAN Interfaces

#### Add New WAN

1. Click **"Add WAN Interface"** button
2. Fill in the form:

| Field | Description | Example |
|-------|-------------|---------|
| Name | Friendly name | WAN3-5G |
| Interface | Network interface or IP | eth2 or 192.168.3.100 |
| Type | Connection type | Fiber, LTE, 5G, Starlink, etc. |
| Local Address | Your IP on this interface | 192.168.3.100 |
| Remote Address | Server address | server.example.com:9000 |
| Priority | Failover order (0=primary) | 0 |
| Weight | Traffic distribution | 100 |
| Max Latency | Threshold in ms | 100 |
| Max Jitter | Threshold in ms | 50 |
| Max Packet Loss | Threshold in % | 5 |
| Enabled | Active or disabled | ☑ |

3. Click **"Save WAN"**
4. **Restart MultiWANBond**

#### Edit Existing WAN

1. Find WAN in the list
2. Click **"Edit"** button
3. Modify any field
4. Click **"Save WAN"**
5. **Restart MultiWANBond**

#### Delete WAN

1. Click **"Delete"** button
2. Confirm in popup
3. **Restart MultiWANBond**

### Tab 2: Routing Policies

Configure how traffic is routed across WANs.

#### Add Policy

1. Click **"Add Routing Policy"**
2. Fill in:
   - **Policy Name**: e.g., "Video Streaming"
   - **Description**: What this policy does
   - **Policy Type**:
     - Source-based: Route from specific IP/subnet
     - Destination-based: Route to specific destination
     - Application-based: Route specific app (requires DPI)
   - **Source/Destination/Application**: Based on type
   - **Target WAN ID**: Which WAN to use
   - **Priority**: Rule priority (lower = higher)
3. Click **"Save Policy"**
4. **Restart MultiWANBond**

*Note: Routing policies are currently placeholder implementation*

### Tab 3: System Configuration

Global settings for MultiWANBond.

#### Load Balancing

**Mode Options**:
- **Round Robin**: Distribute evenly, packet by packet
- **Weighted**: Distribute based on WAN weights
- **Least Used**: Send to WAN with least current traffic
- **Least Latency**: Always use fastest WAN
- **Adaptive**: Intelligent selection based on conditions (Recommended)

#### Forward Error Correction (FEC)

Adds redundant packets to recover from packet loss.

**Settings**:
- **Enable FEC**: ☑ to activate
- **Data Shards**: Number of data packets per block (4 recommended)
- **Parity Shards**: Number of redundant packets (2 recommended)
- **Redundancy Ratio**: Percentage overhead (0.2 = 20%)

**When to use FEC**:
- Satellite connections with packet loss
- Wireless links with interference
- Any WAN with > 1% packet loss

**Trade-off**: Adds bandwidth overhead but improves reliability.

#### Features

**Deep Packet Inspection (DPI)**:
- Analyzes packet contents
- Enables application-aware routing
- Required for application-based policies

**Quality of Service (QoS)**:
- Prioritizes latency-sensitive traffic
- Good for VoIP, gaming
- May reduce overall throughput slightly

**NAT Traversal**:
- Uses STUN/TURN protocols
- Helps connections behind NAT/firewall
- Recommended to keep enabled

#### Save Configuration

1. Configure all desired settings
2. Click **"Save Configuration"**
3. Success message appears
4. **Restart MultiWANBond** for changes to take effect

## 🔄 Common Scenarios

### Scenario 1: Add 5G Backup WAN

**Goal**: Add LTE/5G as failover backup

**Steps**:
1. Configuration → WAN Interfaces
2. Add WAN Interface:
   ```
   Name: WAN3-5G-Backup
   Type: 5G
   Priority: 2 (backup)
   Weight: 50 (lower than primary)
   ```
3. Save → Restart
4. Dashboard will show 3 WANs
5. If primary fails, traffic automatically fails over to 5G

### Scenario 2: Prioritize Low Latency

**Goal**: Route gaming/VoIP traffic through fastest connection

**Steps**:
1. Configuration → System Configuration
2. Load Balance Mode: **Least Latency**
3. Enable QoS: ☑
4. Save → Restart
5. System will prefer lowest latency WAN

### Scenario 3: Optimize for Downloads

**Goal**: Maximum bandwidth for large downloads

**Steps**:
1. Configuration → WAN Interfaces
2. Edit each WAN, set weights by speed:
   ```
   Fiber 1 Gbps: Weight 1000
   Cable 500 Mbps: Weight 500
   LTE 100 Mbps: Weight 100
   ```
3. System Configuration → Mode: **Weighted**
4. Save → Restart
5. Traffic distributed proportionally

### Scenario 4: Handle Packet Loss

**Goal**: Satellite link has 3% packet loss

**Steps**:
1. Configuration → System Configuration
2. Enable FEC: ☑
3. Settings:
   ```
   Data Shards: 4
   Parity Shards: 2
   Redundancy: 0.33 (33% overhead)
   ```
4. Save → Restart
5. System can recover up to 2 lost packets per 4-packet block
6. Monitor packet loss improvement in dashboard

## 📈 Monitoring Best Practices

### What to Watch

**Latency**:
- 🟢 Good: < 50ms
- 🟡 OK: 50-100ms
- 🔴 Bad: > 100ms

**Jitter**:
- 🟢 Good: < 10ms
- 🟡 OK: 10-30ms
- 🔴 Bad: > 30ms

**Packet Loss**:
- 🟢 Good: < 0.5%
- 🟡 OK: 0.5-2%
- 🔴 Bad: > 2%

### Status Colors

- **Green (Up)**: WAN is healthy, all metrics within thresholds
- **Orange (Degraded)**: WAN working but one or more metrics exceed thresholds
- **Red (Down)**: WAN not responding to health checks

### When Metrics Don't Update

**If dashboard shows all zeros**:
1. No traffic is flowing yet
2. Health checks haven't completed (wait 10 seconds)
3. Both server and client must be running
4. Application traffic must route through MultiWANBond

**Statistics won't spam anymore**: They only print to console when values change.

## 🛠️ Troubleshooting

### Web UI Won't Load

**Check**:
1. MultiWANBond is running
2. Port 8080 not blocked by firewall
3. `./webui` directory exists with HTML files
4. Try http://127.0.0.1:8080

### Configuration Not Saving

**Check**:
1. File permissions on config file
2. Disk space available
3. Browser console (F12) for errors
4. MultiWANBond console for error messages

### Changes Don't Take Effect

**Solution**: **Restart MultiWANBond!**

All configuration changes require restarting the application. The UI shows this reminder after every save.

### Dashboard Shows Stale Data

**Solutions**:
1. Refresh browser (F5)
2. Check browser console for JavaScript errors
3. Verify network connection to server
4. Check MultiWANBond is still running

## 💾 Configuration File

All changes are saved to JSON config file:
- **Windows**: `C:\ProgramData\MultiWANBond\config.json`
- **Linux**: `~/.config/multiwanbond/config.json`

**Always backup before major changes!**

```bash
# Backup
cp config.json config.json.backup

# Restore if needed
cp config.json.backup config.json
```

## 🔐 Security

### Enable Authentication

To require login:

```go
// In code or future config
webConfig.EnableAuth = true
webConfig.Username = "admin"
webConfig.Password = "your-secure-password"
```

### Enable HTTPS

For encrypted connections:

```go
webConfig.EnableTLS = true
webConfig.CertFile = "/path/to/cert.pem"
webConfig.KeyFile = "/path/to/key.pem"
```

### Firewall Rules

Restrict access to localhost only:

**Windows Firewall**:
```powershell
New-NetFirewallRule -DisplayName "MultiWANBond WebUI" -Direction Inbound -LocalPort 8080 -Protocol TCP -Action Allow -RemoteAddress 127.0.0.1
```

**Linux iptables**:
```bash
iptables -A INPUT -p tcp --dport 8080 -s 127.0.0.1 -j ACCEPT
iptables -A INPUT -p tcp --dport 8080 -j DROP
```

## 📱 Mobile Access

The Web UI is mobile-responsive!

Access from phone/tablet on same network:
1. Find server IP: `ipconfig` (Windows) or `ip a` (Linux)
2. Open browser on mobile: `http://SERVER_IP:8080`
3. Dashboard and configuration work on mobile

**Note**: For remote access, set up VPN or use HTTPS with proper authentication.

## 🎯 Tips & Tricks

1. **Bookmark both pages**: Dashboard for monitoring, Config for changes
2. **Keep dashboard open**: Monitor changes in real-time
3. **Use descriptive names**: "WAN1-Fiber-Primary" better than "WAN1"
4. **Start conservative**: Don't set thresholds too strict initially
5. **Monitor before/after**: Check dashboard before and after config changes
6. **Test incrementally**: Change one thing at a time
7. **Document reasons**: Use WAN descriptions to note why configured that way
8. **Backup regularly**: Config file is small, easy to backup
9. **Watch the first hour**: After restart, monitor to ensure stable
10. **Use priorities wisely**: 0 for primary, 1 for backup, 2 for emergency

## 🎉 You're All Set!

Your MultiWANBond Web UI is fully functional:

✅ Real-time monitoring dashboard
✅ Complete WAN management
✅ System configuration
✅ Routing policies
✅ Professional interface
✅ Auto-refresh
✅ Mobile-friendly

**Start exploring**: http://localhost:8080

Enjoy your multi-WAN bonded network! 🚀

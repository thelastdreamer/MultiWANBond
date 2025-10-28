# MultiWANBond Web UI

Complete web-based monitoring and configuration interface for MultiWANBond.

## Features

### Dashboard (index.html)
- **Real-time Monitoring**: Auto-refreshes every 2 seconds
- **System Status**: Uptime, version, platform information
- **WAN Statistics**: Total, healthy, degraded, and down interfaces
- **Traffic Metrics**: Packets, bytes, active flows
- **Connection Info**: NAT type, public IP, connection status
- **Individual WAN Cards**: Per-interface metrics (latency, jitter, packet loss, traffic)

### Configuration Management (config.html)
- **WAN Interface Management**
  - Add new WAN interfaces
  - Edit existing configurations
  - Delete interfaces
  - Configure priority, weight, thresholds
  - Enable/disable individual WANs

- **System Configuration**
  - Load balance mode selection (Round Robin, Weighted, Least Used, Adaptive)
  - Forward Error Correction (FEC) settings
  - Feature toggles (DPI, QoS, NAT Traversal)

- **Routing Policies**
  - Source-based routing
  - Destination-based routing
  - Application-based routing
  - Priority management

## Access

Start MultiWANBond and open your browser:

- **Dashboard**: http://localhost:8080/index.html
- **Configuration**: http://localhost:8080/config.html

Or simply: http://localhost:8080/

## Requirements

- MultiWANBond server running with Web UI enabled (port 8080 by default)
- Modern web browser (Chrome, Firefox, Edge, Safari)
- JavaScript enabled

## Usage

### Viewing Status
1. Navigate to http://localhost:8080/
2. View real-time metrics on the dashboard
3. Click "Configuration" button to manage settings

### Adding a WAN Interface
1. Go to Configuration page
2. Click "Add WAN Interface"
3. Fill in the form:
   - Name: e.g., "WAN3-5G"
   - Interface: e.g., "eth2"
   - Type: Select from dropdown
   - Local Address: e.g., "192.168.3.100"
   - Remote Address: e.g., "server.example.com:9000"
   - Priority: 0 = highest
   - Weight: Traffic distribution weight
   - Health check thresholds
4. Click "Save WAN"
5. **Restart MultiWANBond** for changes to take effect

### Editing System Settings
1. Go to Configuration → System Configuration tab
2. Select load balance mode
3. Configure FEC if needed
4. Toggle features
5. Click "Save Configuration"
6. **Restart MultiWANBond** for changes to take effect

## API Endpoints

The Web UI communicates with the backend via REST API:

### Monitoring
- `GET /api/dashboard` - System statistics
- `GET /api/wans/status` - Real-time WAN status
- `GET /api/traffic` - Traffic statistics
- `GET /api/flows` - Active network flows
- `GET /api/health` - Health check results
- `GET /api/nat` - NAT traversal information

### Configuration
- `GET /api/wans` - List all WANs
- `POST /api/wans` - Add new WAN
- `PUT /api/wans` - Update WAN
- `DELETE /api/wans?id=X` - Delete WAN
- `GET /api/config` - Get system configuration
- `PUT /api/config` - Update system configuration
- `GET /api/routing` - List routing policies
- `POST /api/routing` - Add routing policy

### WebSocket
- `ws://localhost:8080/ws` - Real-time event updates

## Security

The Web UI supports optional authentication and TLS:

```go
webConfig := webui.DefaultConfig()
webConfig.EnableAuth = true
webConfig.Username = "admin"
webConfig.Password = "your-secure-password"
webConfig.EnableTLS = true
webConfig.CertFile = "/path/to/cert.pem"
webConfig.KeyFile = "/path/to/key.pem"
```

## Design

- **Framework**: Vanilla JavaScript (no dependencies)
- **Styling**: Embedded CSS with modern gradients
- **Layout**: Responsive CSS Grid
- **Colors**: Purple/blue gradient theme
- **Status Indicators**:
  - Green (#27ae60) = Up
  - Orange (#f39c12) = Degraded
  - Red (#e74c3c) = Down

## Files

- `index.html` - Dashboard (monitoring)
- `config.html` - Configuration management
- `README.md` - This file

## Browser Support

- Chrome/Edge 90+
- Firefox 88+
- Safari 14+

## Notes

1. **Restart Required**: Configuration changes require restarting MultiWANBond
2. **Auto-refresh**: Dashboard updates automatically every 2 seconds
3. **Form Validation**: Client-side validation for all inputs
4. **Error Handling**: User-friendly error messages
5. **Concurrent Access**: Last save wins if multiple users edit simultaneously

## Development

### Customizing Refresh Interval

In `index.html`, modify:
```javascript
// Update every 2 seconds
setInterval(updateAll, 2000);
```

### Customizing Port

In `cmd/server/main.go`:
```go
webConfig.ListenPort = 8080  // Change to desired port
```

### Adding Custom Styling

Both HTML files use embedded CSS. Modify the `<style>` section to customize appearance.

## Troubleshooting

**Problem**: Web UI not loading
- **Solution**: Check MultiWANBond is running and Web UI is enabled
- **Solution**: Verify `./webui` directory exists and contains HTML files
- **Solution**: Check firewall allows port 8080

**Problem**: Configuration changes not saving
- **Solution**: Check file permissions on config file
- **Solution**: Verify config file path is correct
- **Solution**: Check logs for error messages

**Problem**: Dashboard shows all zeros
- **Solution**: No traffic is flowing through the tunnel yet
- **Solution**: Make sure both server and client are running
- **Solution**: Route application traffic through MultiWANBond

**Problem**: Changes not taking effect
- **Solution**: Restart MultiWANBond after configuration changes

## License

Same as MultiWANBond project.

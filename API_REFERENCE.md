# MultiWANBond Web UI API Reference

Complete REST API documentation for the MultiWANBond Web UI.

**Base URL**: `http://localhost:8080` (or your server's address and configured port)

**Authentication**: Cookie-based sessions (except for login endpoints)

**Response Format**: JSON

---

## Table of Contents

- [Authentication Endpoints](#authentication-endpoints)
- [Dashboard Endpoints](#dashboard-endpoints)
- [WAN Management Endpoints](#wan-management-endpoints)
- [Health Monitoring Endpoints](#health-monitoring-endpoints)
- [Traffic & Flow Endpoints](#traffic--flow-endpoints)
- [NAT Information Endpoints](#nat-information-endpoints)
- [Configuration Endpoints](#configuration-endpoints)
- [Alerts & Logs Endpoints](#alerts--logs-endpoints)
- [WebSocket Events](#websocket-events)
- [Error Responses](#error-responses)

---

## Authentication Endpoints

### POST /api/login

**Description**: Authenticate user and create session

**Authentication**: None required

**Request Body**:
```json
{
  "username": "admin",
  "password": "MultiWAN2025Secure!"
}
```

**Success Response** (200 OK):
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "username": "admin",
    "expiresAt": "2025-11-03T15:30:00Z"
  }
}
```

**Sets Cookie**:
```
session_id=<random_token>; Path=/; HttpOnly; SameSite=Strict; Expires=<24h_from_now>
```

**Error Response** (401 Unauthorized):
```json
{
  "success": false,
  "message": "Invalid credentials"
}
```

---

### POST /api/logout

**Description**: Destroy session and clear cookie

**Authentication**: Required (session cookie)

**Request Body**: None

**Success Response** (200 OK):
```json
{
  "success": true,
  "message": "Logout successful"
}
```

**Clears Cookie**:
```
session_id=; Path=/; MaxAge=-1
```

---

### GET /api/session

**Description**: Check if current session is valid

**Authentication**: Required (session cookie)

**Success Response** (200 OK):
```json
{
  "success": true,
  "message": "Session active",
  "data": {
    "username": "admin",
    "expiresAt": "2025-11-03T15:30:00Z"
  }
}
```

**Error Response** (401 Unauthorized):
```json
{
  "success": false,
  "message": "No active session"
}
```

---

## Dashboard Endpoints

### GET /api/dashboard

**Description**: Get dashboard overview statistics

**Authentication**: Required

**Success Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "uptime_seconds": 86400,
    "total_wans": 3,
    "active_wans": 2,
    "total_bytes": 10737418240,
    "total_packets": 15000000,
    "current_mbps": 125.5,
    "active_flows": 42,
    "alerts_count": 1
  }
}
```

**Response Fields**:
- `uptime_seconds`: System uptime in seconds
- `total_wans`: Total configured WAN interfaces
- `active_wans`: Number of currently active WANs
- `total_bytes`: Total bytes transferred (all WANs)
- `total_packets`: Total packets transferred (all WANs)
- `current_mbps`: Current throughput in Mbps
- `active_flows`: Number of active network flows
- `alerts_count`: Number of unread alerts

---

## WAN Management Endpoints

### GET /api/wans

**Description**: Get list of all WAN interfaces with status

**Authentication**: Required

**Success Response** (200 OK):
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "name": "Fiber",
      "interface": "eth0",
      "enabled": true,
      "state": "active",
      "weight": 100,
      "latency_ms": 5.2,
      "jitter_ms": 0.8,
      "packet_loss": 0.01,
      "bytes_sent": 5368709120,
      "bytes_received": 10737418240,
      "last_check": "2025-11-02T14:30:00Z",
      "health_status": "healthy"
    },
    {
      "id": 2,
      "name": "LTE",
      "interface": "wwan0",
      "enabled": true,
      "state": "active",
      "weight": 50,
      "latency_ms": 25.5,
      "jitter_ms": 3.2,
      "packet_loss": 0.5,
      "bytes_sent": 2147483648,
      "bytes_received": 4294967296,
      "last_check": "2025-11-02T14:30:00Z",
      "health_status": "healthy"
    },
    {
      "id": 3,
      "name": "DSL",
      "interface": "eth1",
      "enabled": true,
      "state": "down",
      "weight": 30,
      "latency_ms": 0,
      "jitter_ms": 0,
      "packet_loss": 100,
      "bytes_sent": 0,
      "bytes_received": 0,
      "last_check": "2025-11-02T14:29:45Z",
      "health_status": "failed"
    }
  ]
}
```

**WAN States**:
- `active`: WAN is up and passing traffic
- `down`: WAN is down (health checks failing)
- `disabled`: WAN is administratively disabled
- `standby`: WAN is up but not actively used

**Health Status**:
- `healthy`: All health checks passing
- `degraded`: Some health checks failing
- `failed`: All health checks failing

---

## Health Monitoring Endpoints

### GET /api/health

**Description**: Get detailed health check results for all WANs

**Authentication**: Required

**Success Response** (200 OK):
```json
{
  "success": true,
  "data": [
    {
      "wan_id": 1,
      "wan_name": "Fiber",
      "check_time": "2025-11-02T14:30:00Z",
      "latency_ms": 5.2,
      "jitter_ms": 0.8,
      "packet_loss": 0.01,
      "check_method": "icmp",
      "check_target": "8.8.8.8",
      "status": "healthy",
      "consecutive_failures": 0,
      "uptime_percentage": 99.95
    },
    {
      "wan_id": 2,
      "wan_name": "LTE",
      "check_time": "2025-11-02T14:30:00Z",
      "latency_ms": 25.5,
      "jitter_ms": 3.2,
      "packet_loss": 0.5,
      "check_method": "http",
      "check_target": "http://www.google.com",
      "status": "healthy",
      "consecutive_failures": 0,
      "uptime_percentage": 98.5
    }
  ]
}
```

**Check Methods**:
- `icmp`: ICMP ping
- `http`: HTTP GET request
- `https`: HTTPS GET request
- `tcp`: TCP connection test
- `dns`: DNS query test

---

## Traffic & Flow Endpoints

### GET /api/traffic

**Description**: Get traffic statistics (overall and per-WAN)

**Authentication**: Required

**Query Parameters** (optional):
- `timeRange`: Time range for statistics (1h, 6h, 24h, 7d, 30d) - default: 24h

**Success Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "total_bytes": 10737418240,
    "total_packets": 15000000,
    "bytes_sent": 4294967296,
    "bytes_received": 6442450944,
    "current_upload_mbps": 45.2,
    "current_download_mbps": 80.3,
    "bytes_per_wan": {
      "1": 5368709120,
      "2": 2147483648,
      "3": 0
    },
    "packets_per_wan": {
      "1": 7500000,
      "2": 3750000,
      "3": 0
    },
    "top_protocols": [
      {
        "protocol": "HTTPS",
        "bytes": 5368709120,
        "percentage": 50.0
      },
      {
        "protocol": "HTTP",
        "bytes": 2147483648,
        "percentage": 20.0
      },
      {
        "protocol": "YouTube",
        "bytes": 1073741824,
        "percentage": 10.0
      }
    ]
  }
}
```

---

### GET /api/flows

**Description**: Get active network flows with DPI classification

**Authentication**: Required

**Query Parameters** (optional):
- `limit`: Maximum number of flows to return (default: 100)
- `protocol`: Filter by protocol (e.g., "HTTP", "HTTPS", "YouTube")
- `wan_id`: Filter by WAN ID

**Success Response** (200 OK):
```json
{
  "success": true,
  "data": [
    {
      "src_ip": "192.168.1.100",
      "src_port": 52341,
      "dst_ip": "142.250.185.46",
      "dst_port": 443,
      "protocol": "HTTPS",
      "application": "YouTube",
      "category": "Streaming",
      "packets": 1523,
      "bytes": 2097152,
      "duration_ms": 45000,
      "first_seen": "2025-11-02T14:29:15Z",
      "last_seen": "2025-11-02T14:30:00Z",
      "wan_id": 1,
      "state": "active"
    },
    {
      "src_ip": "192.168.1.100",
      "src_port": 52342,
      "dst_ip": "8.8.8.8",
      "dst_port": 53,
      "protocol": "DNS",
      "application": "DNS",
      "category": "System",
      "packets": 2,
      "bytes": 128,
      "duration_ms": 50,
      "first_seen": "2025-11-02T14:29:59Z",
      "last_seen": "2025-11-02T14:30:00Z",
      "wan_id": 2,
      "state": "active"
    }
  ]
}
```

**Flow States**:
- `active`: Flow currently active
- `closed`: Flow closed normally
- `timeout`: Flow timed out

**Categories**:
- Web
- Streaming
- Social Media
- Gaming
- Communication
- File Transfer
- Email
- DNS
- VPN
- System
- Unknown

---

## NAT Information Endpoints

### GET /api/nat

**Description**: Get NAT traversal information

**Authentication**: Required

**Success Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "nat_type": "Full Cone NAT",
    "public_addr": "203.0.113.45:12345",
    "local_addr": "192.168.1.100:9000",
    "cgnat_detected": false,
    "can_direct_connect": true,
    "needs_relay": false,
    "relay_available": false,
    "stun_server": "stun.l.google.com:19302",
    "last_check": "2025-11-02T14:30:00Z"
  }
}
```

**NAT Types**:
- `Open (No NAT)`: Direct internet connection
- `Full Cone NAT`: Easiest to traverse
- `Restricted Cone NAT`: Moderate difficulty
- `Port-Restricted Cone NAT`: Moderate difficulty
- `Symmetric NAT`: Hardest to traverse (needs relay)
- `Blocked`: UDP completely filtered
- `Unknown`: NAT type not yet detected

---

## Configuration Endpoints

### GET /api/config

**Description**: Get current configuration

**Authentication**: Required

**Success Response** (200 OK):
```json
{
  "success": true,
  "data": {
    "version": "1.0",
    "mode": "client",
    "wans": [
      {
        "id": 1,
        "name": "Fiber",
        "interface": "eth0",
        "enabled": true,
        "weight": 100
      }
    ],
    "routing": {
      "mode": "adaptive",
      "load_balancing": "weighted"
    },
    "health": {
      "check_interval_ms": 5000,
      "timeout_ms": 3000,
      "retry_count": 3,
      "check_hosts": ["8.8.8.8", "1.1.1.1"]
    },
    "security": {
      "encryption_enabled": true,
      "encryption_type": "chacha20poly1305"
    },
    "webui": {
      "enabled": true,
      "port": 8080,
      "username": "admin"
    }
  }
}
```

---

### POST /api/config

**Description**: Update configuration

**Authentication**: Required

**Request Body**: Same structure as GET response

**Success Response** (200 OK):
```json
{
  "success": true,
  "message": "Configuration updated successfully"
}
```

**Error Response** (400 Bad Request):
```json
{
  "success": false,
  "message": "Invalid configuration: missing required field 'mode'"
}
```

---

## Alerts & Logs Endpoints

### GET /api/alerts

**Description**: Get active alerts

**Authentication**: Required

**Success Response** (200 OK):
```json
{
  "success": true,
  "data": [
    {
      "id": "alert-001",
      "timestamp": "2025-11-02T14:25:00Z",
      "severity": "warning",
      "title": "High Latency Detected",
      "message": "WAN 2 (LTE) latency increased to 150ms (threshold: 100ms)",
      "wan_id": 2,
      "acknowledged": false
    },
    {
      "id": "alert-002",
      "timestamp": "2025-11-02T14:20:00Z",
      "severity": "error",
      "title": "WAN Down",
      "message": "WAN 3 (DSL) is down - all health checks failing",
      "wan_id": 3,
      "acknowledged": false
    }
  ]
}
```

**Severity Levels**:
- `info`: Informational message
- `warning`: Warning condition
- `error`: Error condition
- `critical`: Critical condition

---

### DELETE /api/alerts

**Description**: Clear all alerts (or specific alert)

**Authentication**: Required

**Query Parameters** (optional):
- `id`: Alert ID to clear (if omitted, clears all)

**Success Response** (200 OK):
```json
{
  "success": true,
  "message": "Alerts cleared"
}
```

---

### GET /api/logs

**Description**: Get system logs

**Authentication**: Required

**Query Parameters** (optional):
- `limit`: Maximum number of logs to return (default: 100)
- `level`: Filter by log level (debug, info, warn, error)
- `since`: ISO 8601 timestamp for logs since that time

**Success Response** (200 OK):
```json
{
  "success": true,
  "data": [
    {
      "timestamp": "2025-11-02T14:30:00Z",
      "level": "info",
      "message": "WAN 1 health check successful",
      "context": {
        "wan_id": 1,
        "latency_ms": 5.2
      }
    },
    {
      "timestamp": "2025-11-02T14:29:50Z",
      "level": "warn",
      "message": "WAN 2 latency increased to 150ms",
      "context": {
        "wan_id": 2,
        "latency_ms": 150.0,
        "threshold_ms": 100.0
      }
    }
  ]
}
```

**Log Levels**:
- `debug`: Detailed debug information
- `info`: Informational messages
- `warn`: Warning messages
- `error`: Error messages

---

## WebSocket Events

### Connection

**Endpoint**: `ws://localhost:8080/ws`

**Authentication**: Session cookie required

**Connection**:
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
```

---

### Event Types

#### 1. wan_status

**Triggered**: WAN state changes

**Payload**:
```json
{
  "event": "wan_status",
  "data": {
    "wan_id": 1,
    "state": "active",
    "health_status": "healthy"
  }
}
```

---

#### 2. system_alert

**Triggered**: New alert generated

**Payload**:
```json
{
  "event": "system_alert",
  "data": {
    "id": "alert-003",
    "timestamp": "2025-11-02T14:30:00Z",
    "severity": "warning",
    "title": "High Latency Detected",
    "message": "WAN 2 (LTE) latency increased to 150ms"
  }
}
```

---

#### 3. traffic_update

**Triggered**: Traffic statistics updated (every 1 second)

**Payload**:
```json
{
  "event": "traffic_update",
  "data": {
    "total_bytes": 10737418240,
    "current_mbps": 125.5,
    "bytes_per_wan": {
      "1": 5368709120,
      "2": 2147483648
    }
  }
}
```

---

#### 4. health_update

**Triggered**: Health check completed

**Payload**:
```json
{
  "event": "health_update",
  "data": {
    "wan_id": 1,
    "latency_ms": 5.2,
    "packet_loss": 0.01,
    "status": "healthy"
  }
}
```

---

#### 5. nat_info

**Triggered**: NAT information updated

**Payload**:
```json
{
  "event": "nat_info",
  "data": {
    "nat_type": "Full Cone NAT",
    "public_addr": "203.0.113.45:12345",
    "cgnat_detected": false
  }
}
```

---

#### 6. flows_update

**Triggered**: Active flows updated (every 1 second)

**Payload**:
```json
{
  "event": "flows_update",
  "data": {
    "total_flows": 42,
    "active_flows": 38,
    "top_flows": [...]
  }
}
```

---

## Error Responses

All endpoints use consistent error response format:

### 400 Bad Request

**Invalid request data**:
```json
{
  "success": false,
  "message": "Invalid request: missing required field 'username'"
}
```

---

### 401 Unauthorized

**Authentication required or invalid**:
```json
{
  "success": false,
  "message": "Authentication required"
}
```

---

### 403 Forbidden

**Insufficient permissions**:
```json
{
  "success": false,
  "message": "Insufficient permissions"
}
```

---

### 404 Not Found

**Resource not found**:
```json
{
  "success": false,
  "message": "Resource not found"
}
```

---

### 500 Internal Server Error

**Server-side error**:
```json
{
  "success": false,
  "message": "Internal server error",
  "details": "Connection to database failed"
}
```

---

## Rate Limiting

**Current Implementation**: No rate limiting

**Future Plans**: Rate limiting to prevent abuse (v1.2)

---

## CORS

**Current Configuration**: Same-origin only

**Custom Origins**: Configure in `config.json`:
```json
{
  "webui": {
    "cors_enabled": true,
    "cors_origins": ["http://localhost:3000", "https://dashboard.example.com"]
  }
}
```

---

## Authentication Details

### Session Cookie Format

```
session_id=<base64_random_token>; Path=/; HttpOnly; SameSite=Strict; Expires=<timestamp>
```

**Cookie Attributes**:
- `HttpOnly`: Prevents JavaScript access (XSS protection)
- `SameSite=Strict`: Prevents CSRF attacks
- `Secure`: Set when using HTTPS
- `Expires`: 24 hours from creation

### Session Lifecycle

1. **Login**: POST /api/login creates session, sets cookie
2. **Validation**: Every request checks session validity
3. **Refresh**: Session checked every 5 minutes (client-side)
4. **Expiration**: Session expires after 24 hours
5. **Logout**: POST /api/logout destroys session, clears cookie

---

## API Versioning

**Current Version**: v1

**Future Versioning**: API versioning will be introduced in v2.0 with prefix `/api/v2/`

---

## Example Usage

### JavaScript (Browser)

```javascript
// Login
const login = async (username, password) => {
  const response = await fetch('/api/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password })
  });
  return await response.json();
};

// Get WANs
const getWans = async () => {
  const response = await fetch('/api/wans');
  return await response.json();
};

// WebSocket connection
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Event:', data.event, 'Data:', data.data);
};
```

### cURL

```bash
# Login
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"MultiWAN2025Secure!"}' \
  -c cookies.txt

# Get WANs (with session cookie)
curl http://localhost:8080/api/wans \
  -b cookies.txt

# Get traffic stats
curl http://localhost:8080/api/traffic?timeRange=24h \
  -b cookies.txt

# Logout
curl -X POST http://localhost:8080/api/logout \
  -b cookies.txt
```

### Python

```python
import requests

# Login
session = requests.Session()
response = session.post('http://localhost:8080/api/login', json={
    'username': 'admin',
    'password': 'MultiWAN2025Secure!'
})

# Get WANs (session cookie automatically sent)
wans = session.get('http://localhost:8080/api/wans').json()
print(wans)

# Get flows
flows = session.get('http://localhost:8080/api/flows', params={'limit': 50}).json()
print(flows)

# Logout
session.post('http://localhost:8080/api/logout')
```

---

## Additional Resources

- **[UNIFIED_WEB_UI_IMPLEMENTATION.md](UNIFIED_WEB_UI_IMPLEMENTATION.md)** - Web UI architecture and implementation
- **[NAT_DPI_INTEGRATION.md](NAT_DPI_INTEGRATION.md)** - NAT and DPI technical details
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** - Troubleshooting API issues

---

**Last Updated**: November 2, 2025
**API Version**: 1.0
**MultiWANBond Version**: 1.1

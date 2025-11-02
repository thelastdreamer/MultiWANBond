# MultiWANBond Production Deployment Guide

**Complete guide for deploying MultiWANBond in production environments**

**Version**: 1.1
**Last Updated**: November 2, 2025

---

## Table of Contents

- [Production Checklist](#production-checklist)
- [System Requirements](#system-requirements)
- [Linux Deployment](#linux-deployment)
- [Windows Deployment](#windows-deployment)
- [Docker Deployment](#docker-deployment)
- [Monitoring](#monitoring)
- [Backup and Recovery](#backup-and-recovery)
- [High Availability](#high-availability)

---

## Production Checklist

Before deploying to production:

- [ ] **Change default credentials** (username/password)
- [ ] **Enable encryption** (ChaCha20-Poly1305 or AES-256-GCM)
- [ ] **Use strong passwords** (16+ characters, mixed case, numbers, symbols)
- [ ] **Configure firewall** (restrict access to necessary ports)
- [ ] **Set up monitoring** (health checks, metrics, alerts)
- [ ] **Configure backups** (config, logs)
- [ ] **Test failover** (simulate WAN failures)
- [ ] **Document configuration** (network topology, credentials, contacts)
- [ ] **Set up logging** (centralized logging if possible)
- [ ] **Plan maintenance windows** (for updates)

---

## System Requirements

### Minimum Requirements

**Server/Client**:
- **CPU**: 2 cores (4 cores recommended)
- **RAM**: 2 GB (4 GB recommended)
- **Disk**: 100 MB for binary, 10 GB for logs (adjust based on retention)
- **Network**: 1 Gbps NIC per WAN (minimum)

### Recommended Requirements

**Production Server**:
- **CPU**: 4+ cores (8 cores for high throughput)
- **RAM**: 8 GB (16 GB for large deployments)
- **Disk**: SSD for better performance
- **Network**: 10 Gbps NICs for high-bandwidth scenarios

### OS Requirements

**Supported**:
- Linux: Ubuntu 20.04+, Debian 11+, RHEL 8+, CentOS 8+
- Windows: Windows Server 2019+, Windows 10/11
- macOS: macOS 11 Big Sur+

**Kernel Requirements** (Linux):
- Kernel 4.19+ for netlink support
- TCP BBR congestion control (optional, for better performance)

---

## Linux Deployment

### systemd Service (Recommended)

**1. Install binary**:
```bash
sudo cp multiwanbond /usr/local/bin/
sudo chmod +x /usr/local/bin/multiwanbond
```

**2. Create config directory**:
```bash
sudo mkdir -p /etc/multiwanbond
sudo cp config.json /etc/multiwanbond/
sudo chmod 600 /etc/multiwanbond/config.json  # Protect secrets
```

**3. Create systemd service** (`/etc/systemd/system/multiwanbond.service`):
```ini
[Unit]
Description=MultiWANBond - Multi-WAN Link Bonding
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/multiwanbond --config /etc/multiwanbond/config.json
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/log/multiwanbond /etc/multiwanbond

# Resource limits
LimitNOFILE=65536
LimitNPROC=512

[Install]
WantedBy=multi-user.target
```

**4. Enable and start**:
```bash
sudo systemctl daemon-reload
sudo systemctl enable multiwanbond
sudo systemctl start multiwanbond
```

**5. Check status**:
```bash
sudo systemctl status multiwanbond
sudo journalctl -u multiwanbond -f  # Follow logs
```

### Running as Non-Root (Advanced)

**1. Create dedicated user**:
```bash
sudo useradd -r -s /bin/false multiwanbond
```

**2. Set capabilities** (allows binding to privileged ports without root):
```bash
sudo setcap 'cap_net_bind_service,cap_net_admin,cap_net_raw=+ep' /usr/local/bin/multiwanbond
```

**3. Update service file**:
```ini
[Service]
User=multiwanbond
Group=multiwanbond
```

**Note**: Some features (routing tables, netlink) may require root. Test thoroughly.

### Firewall Configuration

**ufw (Ubuntu/Debian)**:
```bash
# Allow Web UI (adjust source as needed)
sudo ufw allow from 192.168.1.0/24 to any port 8080 proto tcp

# Allow MultiWANBond bonding port
sudo ufw allow 9000/udp

# Enable firewall
sudo ufw enable
```

**firewalld (RHEL/CentOS)**:
```bash
sudo firewall-cmd --permanent --add-port=8080/tcp  # Web UI
sudo firewall-cmd --permanent --add-port=9000/udp  # Bonding
sudo firewall-cmd --reload
```

**iptables**:
```bash
# Allow Web UI from specific network
sudo iptables -A INPUT -p tcp --dport 8080 -s 192.168.1.0/24 -j ACCEPT

# Allow bonding traffic
sudo iptables -A INPUT -p udp --dport 9000 -j ACCEPT

# Save rules
sudo iptables-save > /etc/iptables/rules.v4
```

---

## Windows Deployment

### Windows Service

**1. Install binary**:
```powershell
# Create directory
New-Item -Path "C:\Program Files\MultiWANBond" -ItemType Directory

# Copy binary
Copy-Item multiwanbond.exe "C:\Program Files\MultiWANBond\"
Copy-Item config.json "C:\Program Files\MultiWANBond\"
```

**2. Create service using NSSM** (Non-Sucking Service Manager):

**Download NSSM**: https://nssm.cc/download

```powershell
# Install service
nssm install MultiWANBond "C:\Program Files\MultiWANBond\multiwanbond.exe"
nssm set MultiWANBond AppParameters "--config C:\Program Files\MultiWANBond\config.json"
nssm set MultiWANBond AppDirectory "C:\Program Files\MultiWANBond"
nssm set MultiWANBond DisplayName "MultiWANBond Service"
nssm set MultiWANBond Description "Multi-WAN Link Bonding Service"
nssm set MultiWANBond Start SERVICE_AUTO_START

# Configure failure actions (restart on failure)
nssm set MultiWANBond AppExit Default Restart
nssm set MultiWANBond AppThrottle 10000

# Start service
nssm start MultiWANBond
```

**3. Check status**:
```powershell
Get-Service MultiWANBond
Get-EventLog -LogName Application -Source MultiWANBond -Newest 20
```

### Alternative: sc.exe (Built-in)

```powershell
sc.exe create MultiWANBond binPath= "C:\Program Files\MultiWANBond\multiwanbond.exe --config C:\Program Files\MultiWANBond\config.json" start= auto
sc.exe start MultiWANBond
```

### Windows Firewall

```powershell
# Allow Web UI
New-NetFirewallRule -DisplayName "MultiWANBond Web UI" -Direction Inbound -LocalPort 8080 -Protocol TCP -Action Allow

# Allow bonding traffic
New-NetFirewallRule -DisplayName "MultiWANBond Bonding" -Direction Inbound -LocalPort 9000 -Protocol UDP -Action Allow
```

---

## Docker Deployment

### Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev linux-headers

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags="-s -w" -o multiwanbond cmd/server/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 multiwanbond && \
    adduser -D -u 1000 -G multiwanbond multiwanbond

# Copy binary from builder
COPY --from=builder /build/multiwanbond /usr/local/bin/

# Copy Web UI files
COPY webui /usr/local/share/multiwanbond/webui

# Set permissions
RUN chown -R multiwanbond:multiwanbond /usr/local/share/multiwanbond

# Create config directory
RUN mkdir -p /etc/multiwanbond && \
    chown multiwanbond:multiwanbond /etc/multiwanbond

# Switch to non-root user
USER multiwanbond

# Expose ports
EXPOSE 8080 9000/udp

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/api/session || exit 1

# Entry point
ENTRYPOINT ["/usr/local/bin/multiwanbond"]
CMD ["--config", "/etc/multiwanbond/config.json"]
```

### docker-compose.yml

```yaml
version: '3.8'

services:
  multiwanbond:
    build: .
    container_name: multiwanbond
    restart: unless-stopped
    network_mode: host  # Required for WAN access
    cap_add:
      - NET_ADMIN
      - NET_RAW
    volumes:
      - ./config.json:/etc/multiwanbond/config.json:ro
      - multiwanbond-logs:/var/log/multiwanbond
    environment:
      - TZ=UTC
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/api/session"]
      interval: 30s
      timeout: 10s
      retries: 3

volumes:
  multiwanbond-logs:
```

### Docker Commands

```bash
# Build image
docker build -t multiwanbond:1.1 .

# Run container
docker run -d \
  --name multiwanbond \
  --restart unless-stopped \
  --network host \
  --cap-add NET_ADMIN \
  --cap-add NET_RAW \
  -v $(pwd)/config.json:/etc/multiwanbond/config.json:ro \
  multiwanbond:1.1

# View logs
docker logs -f multiwanbond

# Stop container
docker stop multiwanbond

# Start with docker-compose
docker-compose up -d

# View logs with docker-compose
docker-compose logs -f
```

---

## Monitoring

### Health Checks

**HTTP Health Endpoint** (add to application):
```
GET /health
```

**Response**:
```json
{
  "status": "healthy",
  "uptime": 86400,
  "wans": {
    "total": 3,
    "healthy": 2,
    "failed": 1
  }
}
```

### Prometheus Metrics (Future Feature)

**Metrics endpoint**:
```
GET /metrics
```

**Example metrics**:
```prometheus
# HELP multiwanbond_wan_state WAN state (1=up, 0=down)
# TYPE multiwanbond_wan_state gauge
multiwanbond_wan_state{wan_id="1",wan_name="Fiber"} 1
multiwanbond_wan_state{wan_id="2",wan_name="LTE"} 0

# HELP multiwanbond_traffic_bytes Total traffic in bytes
# TYPE multiwanbond_traffic_bytes counter
multiwanbond_traffic_bytes{wan_id="1",direction="tx"} 5368709120
multiwanbond_traffic_bytes{wan_id="1",direction="rx"} 10737418240

# HELP multiwanbond_latency_ms WAN latency in milliseconds
# TYPE multiwanbond_latency_ms gauge
multiwanbond_latency_ms{wan_id="1"} 5.2
```

### Log Aggregation

**rsyslog configuration** (`/etc/rsyslog.d/multiwanbond.conf`):
```
if $programname == 'multiwanbond' then /var/log/multiwanbond/multiwanbond.log
& stop
```

**Logrotate** (`/etc/logrotate.d/multiwanbond`):
```
/var/log/multiwanbond/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 0640 multiwanbond multiwanbond
    sharedscripts
    postrotate
        systemctl reload multiwanbond
    endscript
}
```

### External Monitoring

**Nagios/Icinga**:
```bash
#!/bin/bash
# check_multiwanbond.sh

STATUS=$(curl -s http://localhost:8080/api/session | jq -r '.success')

if [ "$STATUS" == "true" ]; then
    echo "OK - MultiWANBond is running"
    exit 0
else
    echo "CRITICAL - MultiWANBond is down"
    exit 2
fi
```

**Uptime Robot**: Monitor `http://<server>:8080/` every 5 minutes

---

## Backup and Recovery

### What to Backup

**Essential**:
- `/etc/multiwanbond/config.json` (configuration)
- Encryption keys (if stored separately)
- Web UI credentials

**Optional**:
- `/var/log/multiwanbond/` (logs, if needed)
- Metrics data (if using persistent storage)

### Backup Script

```bash
#!/bin/bash
# backup-multiwanbond.sh

BACKUP_DIR="/backup/multiwanbond"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup configuration
tar -czf "$BACKUP_DIR/multiwanbond-config-$DATE.tar.gz" \
    /etc/multiwanbond/config.json

# Optional: Backup logs
tar -czf "$BACKUP_DIR/multiwanbond-logs-$DATE.tar.gz" \
    /var/log/multiwanbond/

# Keep only last 30 days
find "$BACKUP_DIR" -name "multiwanbond-*.tar.gz" -mtime +30 -delete

echo "Backup complete: $BACKUP_DIR/multiwanbond-config-$DATE.tar.gz"
```

### Automated Backups with Cron

```bash
# Edit crontab
crontab -e

# Add daily backup at 2 AM
0 2 * * * /usr/local/bin/backup-multiwanbond.sh
```

### Recovery Procedure

**1. Stop service**:
```bash
sudo systemctl stop multiwanbond
```

**2. Restore configuration**:
```bash
tar -xzf /backup/multiwanbond/multiwanbond-config-20251102_020000.tar.gz -C /
```

**3. Verify configuration**:
```bash
multiwanbond --config /etc/multiwanbond/config.json --validate
```

**4. Start service**:
```bash
sudo systemctl start multiwanbond
```

---

## High Availability

### Active-Passive Setup

**Architecture**:
```
┌─────────────┐         ┌─────────────┐
│  Primary    │◄───────►│  Secondary  │
│  Server     │  VRRP   │  Server     │
│  (MASTER)   │         │  (BACKUP)   │
└──────┬──────┘         └──────┬──────┘
       │                       │
       └───────────┬───────────┘
                   │
            Virtual IP (VIP)
                   │
           ┌───────▼───────┐
           │    Clients    │
           └───────────────┘
```

**Using Keepalived** (`/etc/keepalived/keepalived.conf`):

**Primary**:
```
vrrp_script check_multiwanbond {
    script "/usr/local/bin/check_multiwanbond.sh"
    interval 5
    weight 20
}

vrrp_instance VI_1 {
    state MASTER
    interface eth0
    virtual_router_id 51
    priority 100
    advert_int 1

    authentication {
        auth_type PASS
        auth_pass your_secure_password
    }

    virtual_ipaddress {
        192.168.1.100/24
    }

    track_script {
        check_multiwanbond
    }
}
```

**Secondary**:
```
# Same config, but:
state BACKUP
priority 90
```

### Load Balancing (Multiple Servers)

**Using HAProxy**:
```
frontend multiwanbond_web
    bind *:8080
    mode http
    default_backend multiwanbond_servers

backend multiwanbond_servers
    mode http
    balance roundrobin
    option httpchk GET /api/session
    server server1 192.168.1.101:8080 check
    server server2 192.168.1.102:8080 check
    server server3 192.168.1.103:8080 check
```

### Health Check Script

```bash
#!/bin/bash
# check_multiwanbond.sh

curl -f -s http://localhost:8080/api/session > /dev/null 2>&1
exit $?
```

---

## Best Practices

1. **Change default credentials** immediately after deployment
2. **Use strong passwords** (16+ characters, mixed case, numbers, symbols)
3. **Enable encryption** in production environments
4. **Restrict network access** (firewall, VPN, private networks)
5. **Monitor actively** (health checks, metrics, alerts)
6. **Backup regularly** (automate with cron)
7. **Test disaster recovery** (restore from backup annually)
8. **Keep software updated** (security patches)
9. **Document everything** (network topology, credentials, procedures)
10. **Plan maintenance windows** (for updates, changes)

---

## Troubleshooting

**Service won't start**:
- Check logs: `journalctl -u multiwanbond -n 50`
- Verify config: `multiwanbond --config /etc/multiwanbond/config.json --validate`
- Check permissions: `ls -l /etc/multiwanbond/config.json`

**High memory usage**:
- Check flow table size in config
- Review metrics retention settings
- Consider increasing server resources

**Poor performance**:
- Check CPU usage: `top -p $(pidof multiwanbond)`
- Review load balancing mode
- Verify WAN health
- Consider hardware acceleration

---

## Additional Resources

- [ARCHITECTURE.md](ARCHITECTURE.md) - System architecture
- [SECURITY.md](SECURITY.md) - Security best practices
- [PERFORMANCE.md](PERFORMANCE.md) - Performance tuning
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Troubleshooting guide

---

**Last Updated**: November 2, 2025
**Version**: 1.1
**MultiWANBond Version**: 1.1

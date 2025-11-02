# MultiWANBond Security Guide

**Comprehensive security best practices and hardening guide**

**Version**: 1.1
**Last Updated**: November 2, 2025

---

## Table of Contents

- [Security Overview](#security-overview)
- [Configuration Security](#configuration-security)
- [Network Security](#network-security)
- [Web UI Security](#web-ui-security)
- [Encryption](#encryption)
- [Access Control](#access-control)
- [Audit and Logging](#audit-and-logging)
- [Security Checklist](#security-checklist)

---

## Security Overview

### Defense in Depth

MultiWANBond implements security at multiple layers:

**Layer 1: Network**
- Firewall rules
- Network segmentation
- VPN tunnels

**Layer 2: Transport**
- Encryption (AES-256-GCM, ChaCha20-Poly1305)
- AEAD authentication
- Replay protection

**Layer 3: Application**
- Session management
- Input validation
- CSRF protection
- XSS protection

**Layer 4: Data**
- No plaintext credentials
- Secure key storage
- Key rotation

---

## Configuration Security

### Protecting config.json

**File Permissions** (Linux/macOS):
```bash
# Restrict to owner only
chmod 600 /etc/multiwanbond/config.json

# Verify permissions
ls -l /etc/multiwanbond/config.json
# Should show: -rw------- (600)
```

**File Permissions** (Windows):
```powershell
# Remove inheritance
icacls "C:\Program Files\MultiWANBond\config.json" /inheritance:r

# Grant access only to SYSTEM and Administrators
icacls "C:\Program Files\MultiWANBond\config.json" /grant:r "SYSTEM:F"
icacls "C:\Program Files\MultiWANBond\config.json" /grant:r "Administrators:F"
```

### Secure Password Storage

**Never store plaintext passwords**:
```json
// ❌ BAD
{
  "webui": {
    "password": "MyPassword123"
  }
}
```

**Use environment variables**:
```json
// ✅ GOOD
{
  "webui": {
    "password": "${WEBUI_PASSWORD}"
  }
}
```

**Set environment variable**:
```bash
# Linux/macOS
export WEBUI_PASSWORD="MySecurePassword123!"

# Windows
set WEBUI_PASSWORD=MySecurePassword123!
```

### Strong Passwords

**Requirements**:
- Minimum 16 characters
- Mix of uppercase, lowercase, numbers, symbols
- No dictionary words
- Unique (not reused)

**Example strong password**: `Xy7$mK9!pQ3#wR8@nB5^`

**Generate secure password** (Linux):
```bash
openssl rand -base64 24
# Output: 3K7mP9xQ2wR8nB5vL4cF6hJ1
```

### Encryption Keys

**Pre-Shared Key Security**:

**Generate cryptographically secure key**:
```bash
# Linux/macOS
openssl rand -hex 32
# Output: 64-character hex string

# Windows (PowerShell)
-join ((48..57) + (97..102) | Get-Random -Count 64 | % {[char]$_})
```

**Store securely**:
- Use environment variables
- Use secrets management (Vault, AWS Secrets Manager)
- Never commit to Git
- Rotate regularly (every 90 days recommended)

---

## Network Security

### Firewall Configuration

**Principle**: Only allow necessary traffic

**Minimal Rules**:
```bash
# Allow Web UI from management network only
sudo ufw allow from 192.168.1.0/24 to any port 8080 proto tcp

# Allow bonding traffic (adjust as needed)
sudo ufw allow 9000/udp

# Deny all other incoming
sudo ufw default deny incoming

# Allow all outgoing
sudo ufw default allow outgoing

# Enable firewall
sudo ufw enable
```

**Advanced Rules** (restrict Web UI to specific IPs):
```bash
# Allow only from admin workstation
sudo ufw allow from 192.168.1.50 to any port 8080 proto tcp

# Allow only from monitoring system
sudo ufw allow from 192.168.1.60 to any port 8080 proto tcp
```

### Network Segmentation

**Recommended Network Layout**:
```
┌────────────────────────────────────────────────┐
│         Management Network (192.168.1.0/24)    │
│         - Web UI access                        │
│         - SSH/RDP access                       │
│         - Monitoring systems                   │
└───────────────────┬────────────────────────────┘
                    │
        ┌───────────▼───────────┐
        │  MultiWANBond Server  │
        └───────────┬───────────┘
                    │
        ┌───────────┼───────────┬─────────────┐
        │           │           │             │
    WAN 1       WAN 2       WAN 3        Client VPN
  (Public)    (Public)    (Public)      (10.0.0.0/8)
```

**Isolate WANs**:
- Each WAN on separate physical interface
- No routing between WANs
- Firewall rules prevent WAN-to-WAN traffic

### VPN for Web UI Access

**WireGuard Tunnel** (recommended):
```ini
# Server: /etc/wireguard/wg0.conf
[Interface]
Address = 10.0.0.1/24
ListenPort = 51820
PrivateKey = <server_private_key>

# Admin client
[Peer]
PublicKey = <client_public_key>
AllowedIPs = 10.0.0.2/32
```

**Access Web UI over VPN**:
```
http://10.0.0.1:8080
```

**Benefits**:
- Encrypted tunnel
- No direct internet exposure
- Multi-factor authentication (VPN + Web UI login)

### DDoS Protection

**Rate Limiting** (iptables):
```bash
# Limit connections to Web UI (10/min per IP)
iptables -A INPUT -p tcp --dport 8080 -m state --state NEW -m recent --set
iptables -A INPUT -p tcp --dport 8080 -m state --state NEW -m recent --update --seconds 60 --hitcount 10 -j DROP
```

**Connection Limits**:
```bash
# Max 20 concurrent connections to Web UI per IP
iptables -A INPUT -p tcp --dport 8080 -m connlimit --connlimit-above 20 -j REJECT
```

---

## Web UI Security

### Session Security

**Current Implementation**:
- HttpOnly cookies (prevents XSS)
- SameSite=Strict (prevents CSRF)
- 24-hour expiration
- Server-side validation
- Cryptographic random session IDs (32 bytes)

**Hardening Configuration**:
```json
{
  "webui": {
    "session_timeout_hours": 1,  // Shorter timeout for high-security
    "secure_cookies": true,       // Require HTTPS
    "csrf_protection": true
  }
}
```

### HTTPS Configuration (Recommended for Production)

**Generate self-signed certificate** (testing):
```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

**Use Let's Encrypt** (production):
```bash
sudo certbot certonly --standalone -d multiwanbond.example.com
```

**Configure MultiWANBond** (future feature):
```json
{
  "webui": {
    "tls_enabled": true,
    "tls_cert": "/etc/letsencrypt/live/multiwanbond.example.com/fullchain.pem",
    "tls_key": "/etc/letsencrypt/live/multiwanbond.example.com/privkey.pem"
  }
}
```

**Reverse Proxy with Nginx** (current solution):
```nginx
server {
    listen 443 ssl http2;
    server_name multiwanbond.example.com;

    ssl_certificate /etc/letsencrypt/live/multiwanbond.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/multiwanbond.example.com/privkey.pem;

    # Strong SSL configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers 'ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name multiwanbond.example.com;
    return 301 https://$server_name$request_uri;
}
```

### Content Security Policy

**Future Enhancement**:
```html
<meta http-equiv="Content-Security-Policy" content="
    default-src 'self';
    script-src 'self' https://cdn.jsdelivr.net;
    style-src 'self' 'unsafe-inline';
    img-src 'self' data:;
    connect-src 'self' ws://localhost:8080;
    font-src 'self';
    frame-ancestors 'none';
">
```

### Input Validation

**All user inputs are validated**:
- Username: alphanumeric, 3-32 characters
- Password: 8+ characters (16+ recommended)
- WAN names: alphanumeric, spaces, hyphens, 1-50 characters
- IP addresses: valid IPv4/IPv6 format
- Ports: 1-65535

**Protection against**:
- SQL injection (not applicable, no SQL database)
- XSS (inputs sanitized, outputs escaped)
- Command injection (no shell execution from user input)
- Path traversal (file paths validated)

---

## Encryption

### Encryption Algorithms

**ChaCha20-Poly1305** (Recommended):
- Pros: Fast in software, constant-time (resistant to timing attacks)
- Cons: Not hardware-accelerated on all platforms
- Best for: General use, ARM processors

**AES-256-GCM**:
- Pros: Hardware-accelerated (AES-NI), widely supported
- Cons: Slower in software without AES-NI
- Best for: x86 processors with AES-NI

**Configuration**:
```json
{
  "security": {
    "encryption_enabled": true,
    "encryption_type": "chacha20poly1305",  // or "aes-256-gcm"
    "pre_shared_key": "${ENCRYPTION_KEY}"
  }
}
```

### Key Management

**Key Generation**:
```bash
# Generate 256-bit key (64 hex characters)
openssl rand -hex 32
```

**Key Rotation**:

**Frequency**: Every 90 days (recommended)

**Procedure**:
1. Generate new key
2. Update configuration on all clients/servers
3. Rolling restart (one at a time)
4. Verify connectivity
5. Securely delete old key

**Key Storage**:

**Option 1: Environment Variable** (simplest):
```bash
export ENCRYPTION_KEY="your-64-character-hex-key"
```

**Option 2: Secrets Management** (production):
- HashiCorp Vault
- AWS Secrets Manager
- Azure Key Vault
- Google Secret Manager

### Perfect Forward Secrecy (PFS)

**Future Enhancement**: Diffie-Hellman key exchange for session keys

**Benefits**:
- Compromise of long-term key doesn't compromise past sessions
- Each session has unique encryption key
- Industry best practice

---

## Access Control

### User Management

**Current**: Single admin user

**Future Enhancement**: Multi-user support with roles

**Planned Roles**:
- **Admin**: Full access (config, restart, delete)
- **Operator**: View and manage WANs
- **Read-Only**: View only (monitoring)

### Two-Factor Authentication (2FA)

**Future Enhancement**: TOTP-based 2FA

**Implementation**:
```json
{
  "webui": {
    "totp_enabled": true,
    "totp_required": true
  }
}
```

### API Authentication

**Current**: Cookie-based sessions

**Future Enhancement**: API tokens for programmatic access

**Planned**:
```http
Authorization: Bearer <token>
```

---

## Audit and Logging

### Security Event Logging

**Log All Security Events**:
- Login attempts (success/failure)
- Configuration changes
- WAN state changes
- Session creation/destruction
- API requests

**Example Log**:
```
[2025-11-02 14:30:00] AUDIT login_success user=admin ip=192.168.1.50
[2025-11-02 14:31:15] AUDIT config_change user=admin action=wan_disable wan_id=2
[2025-11-02 14:35:42] AUDIT login_failure user=admin ip=203.0.113.45 reason=invalid_password
```

### Failed Login Tracking

**Implement Rate Limiting** (future):
- Lock account after 5 failed attempts
- Lockout duration: 15 minutes
- CAPTCHA after 3 failed attempts

**Current Workaround** (fail2ban):
```ini
# /etc/fail2ban/filter.d/multiwanbond.conf
[Definition]
failregex = AUDIT login_failure.*ip=<HOST>
ignoreregex =
```

```ini
# /etc/fail2ban/jail.d/multiwanbond.conf
[multiwanbond]
enabled = true
port = 8080
filter = multiwanbond
logpath = /var/log/multiwanbond/multiwanbond.log
maxretry = 5
bantime = 900  # 15 minutes
```

### Log Retention

**Recommendations**:
- **Security logs**: 90 days minimum
- **Audit logs**: 1 year (compliance)
- **Operational logs**: 30 days

**Logrotate**:
```
/var/log/multiwanbond/*.log {
    daily
    rotate 90
    compress
    delaycompress
    missingok
    notifempty
    create 0640 multiwanbond multiwanbond
}
```

### Centralized Logging

**Send to SIEM** (Security Information and Event Management):
- Splunk
- ELK Stack (Elasticsearch, Logstash, Kibana)
- Graylog

**rsyslog Forward**:
```
*.* @@siem.example.com:514
```

---

## Security Checklist

### Initial Setup

- [ ] Change default admin password
- [ ] Use strong password (16+ characters)
- [ ] Enable encryption (ChaCha20-Poly1305 or AES-256-GCM)
- [ ] Generate secure pre-shared key (256-bit)
- [ ] Store keys in environment variables or secrets manager
- [ ] Restrict config.json permissions (600)

### Network Security

- [ ] Configure firewall (allow only necessary ports)
- [ ] Restrict Web UI to management network
- [ ] Consider VPN for remote access
- [ ] Enable rate limiting
- [ ] Segment networks (WANs, management, client)

### Web UI Security

- [ ] Use HTTPS (reverse proxy or native)
- [ ] Set secure session timeout (1-24 hours)
- [ ] Enable HSTS, CSP headers (via reverse proxy)
- [ ] Review failed login attempts regularly

### Operational Security

- [ ] Enable audit logging
- [ ] Set up log retention
- [ ] Configure centralized logging (optional)
- [ ] Implement backup strategy
- [ ] Test disaster recovery
- [ ] Document security procedures

### Ongoing Maintenance

- [ ] Rotate encryption keys (every 90 days)
- [ ] Update software regularly
- [ ] Review logs weekly
- [ ] Audit configuration changes
- [ ] Test backups monthly
- [ ] Conduct security audits annually

---

## Incident Response

### Security Incident Procedure

**1. Detect**:
- Monitor logs for anomalies
- Watch for failed login attempts
- Check for unexpected configuration changes

**2. Contain**:
- Isolate compromised system
- Disable affected accounts
- Block attacker IP addresses

**3. Investigate**:
- Review logs
- Identify attack vector
- Determine extent of compromise

**4. Eradicate**:
- Remove malware/backdoors
- Patch vulnerabilities
- Reset all credentials

**5. Recover**:
- Restore from clean backup
- Verify system integrity
- Monitor for reinfection

**6. Lessons Learned**:
- Document incident
- Update procedures
- Implement additional controls

### Emergency Contacts

Document and maintain:
- Security team contacts
- Vendor support
- Incident response team
- Management escalation

---

## Compliance

### GDPR (if applicable)

- Log only necessary data
- Allow users to export their data
- Implement data retention policies
- Provide data deletion capability

### PCI DSS (if handling payment data)

- Use strong cryptography
- Protect cardholder data (if applicable)
- Maintain audit trails
- Regular security testing

### HIPAA (if handling health data)

- Encrypt data in transit and at rest
- Implement access controls
- Audit all access
- Business associate agreements

**Note**: MultiWANBond is network infrastructure. Compliance primarily affects how you use it, not the software itself.

---

## Security Reporting

**Found a security vulnerability?**

**Do NOT open a public issue.**

**Email**: security@example.com (replace with actual contact)

**Include**:
- Description of vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

**We will**:
- Acknowledge receipt within 48 hours
- Provide timeline for fix
- Credit you in release notes (if desired)

---

## Additional Resources

- [ARCHITECTURE.md](ARCHITECTURE.md) - Security architecture details
- [DEPLOYMENT.md](DEPLOYMENT.md) - Secure deployment practices
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Security troubleshooting

---

**Last Updated**: November 2, 2025
**Version**: 1.1
**MultiWANBond Version**: 1.1

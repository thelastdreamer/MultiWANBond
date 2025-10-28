# Setting Up Web UI Authentication

## Current Status
The Web UI currently runs **without authentication** by default. This guide shows how to enable it.

## Quick Setup (Manual)

### 1. Generate a Random Password

**Option A: PowerShell (Windows)**
```powershell
# Generate 16-character random password
$password = -join ((48..57) + (65..90) + (97..122) | Get-Random -Count 16 | ForEach-Object {[char]$_})
Write-Host "Your Web UI Password: $password"
Write-Host "Save this password securely!"
```

**Option B: Bash (Linux/Mac)**
```bash
# Generate 16-character random password
password=$(< /dev/urandom tr -dc 'A-Za-z0-9' | head -c16)
echo "Your Web UI Password: $password"
echo "Save this password securely!"
```

**Option C: Online Generator**
- Visit: https://passwordsgenerator.net/
- Generate a strong 16+ character password
- Save it securely

### 2. Enable Authentication in Code

Edit `cmd/server/main.go` and modify the Web UI configuration:

```go
// Around line 148-151
webConfig := webui.DefaultConfig()
webConfig.ListenPort = 8080

// ADD THESE LINES:
webConfig.EnableAuth = true
webConfig.Username = "admin"
webConfig.Password = "YOUR_GENERATED_PASSWORD_HERE"  // Replace with your password

webServer := webui.NewServer(webConfig)
```

### 3. Rebuild

```bash
go build -o multiwanbond.exe cmd/server/main.go
```

### 4. Test

Start the server and open http://localhost:8080

You'll see a login prompt:
- **Username**: admin
- **Password**: [your generated password]

---

## Future Enhancement: Auto-Generate During Setup

To fully automate this, the setup wizard would need to:

### Changes Required in `pkg/setup/wizard.go`:

```go
import (
    "crypto/rand"
    "encoding/base64"
    // ... existing imports
)

// Add to Config struct in types.go:
type Config struct {
    // ... existing fields ...
    WebUIUsername string
    WebUIPassword string
}

// Add password generation function:
func generatePassword(length int) (string, error) {
    bytes := make([]byte, length)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// In Run() function, after configuring WANs:
func (w *Wizard) Run() (*Config, error) {
    // ... existing setup code ...

    // Generate Web UI credentials
    fmt.Println("\n" + strings.Repeat("=", 70))
    fmt.Println("SECURITY: Generating Web UI Credentials")
    fmt.Println(strings.Repeat("=", 70))

    password, err := generatePassword(16)
    if err != nil {
        return nil, fmt.Errorf("failed to generate password: %w", err)
    }

    config.WebUIUsername = "admin"
    config.WebUIPassword = password

    fmt.Println()
    fmt.Println("  Web UI URL: http://localhost:8080")
    fmt.Println("  Username:   admin")
    fmt.Printf("  Password:   %s\n", password)
    fmt.Println()
    fmt.Println("  ⚠️  IMPORTANT: Save this password securely!")
    fmt.Println("  ⚠️  You'll need it to access the Web UI.")
    fmt.Println()
    fmt.Print("Press Enter to continue...")
    w.scanner.Scan()

    return config, nil
}
```

### Changes Required in `cmd/server/main.go`:

```go
// In runServer(), after loading config:
webConfig := webui.DefaultConfig()
webConfig.ListenPort = 8080

// Load credentials from config if available
if cfg.WebUI != nil {
    webConfig.EnableAuth = true
    webConfig.Username = cfg.WebUI.Username
    webConfig.Password = cfg.WebUI.Password
}

webServer := webui.NewServer(webConfig)
```

### Changes Required in `pkg/config/config.go`:

```go
// Add to BondConfig struct:
type BondConfig struct {
    // ... existing fields ...
    WebUI *WebUIConfig `json:"webui,omitempty"`
}

// Add new type:
type WebUIConfig struct {
    Username string `json:"username"`
    Password string `json:"password"`
    Enabled  bool   `json:"enabled"`
}
```

---

## Enable HTTPS (Optional but Recommended)

### 1. Generate Self-Signed Certificate

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

### 2. Enable TLS in Code

```go
webConfig.EnableTLS = true
webConfig.CertFile = "cert.pem"
webConfig.KeyFile = "key.pem"
```

### 3. Access via HTTPS

```
https://localhost:8080
```

---

## Security Best Practices

1. **Change Default Password**: If you hardcode a password, change it immediately after first login
2. **Use HTTPS**: Always use HTTPS in production
3. **Restrict Access**: Use firewall rules to limit who can access port 8080
4. **Rotate Passwords**: Change password periodically
5. **Strong Passwords**: Use minimum 16 characters with mixed case, numbers, and symbols

---

## Firewall Configuration

### Windows Firewall

**Allow only localhost**:
```powershell
New-NetFirewallRule -DisplayName "MultiWANBond WebUI" `
    -Direction Inbound `
    -LocalPort 8080 `
    -Protocol TCP `
    -Action Allow `
    -RemoteAddress 127.0.0.1
```

**Allow specific IP**:
```powershell
New-NetFirewallRule -DisplayName "MultiWANBond WebUI" `
    -Direction Inbound `
    -LocalPort 8080 `
    -Protocol TCP `
    -Action Allow `
    -RemoteAddress 192.168.1.100
```

### Linux iptables

**Allow only localhost**:
```bash
iptables -A INPUT -p tcp --dport 8080 -s 127.0.0.1 -j ACCEPT
iptables -A INPUT -p tcp --dport 8080 -j DROP
```

**Allow specific IP**:
```bash
iptables -A INPUT -p tcp --dport 8080 -s 192.168.1.100 -j ACCEPT
iptables -A INPUT -p tcp --dport 8080 -j DROP
```

---

## Troubleshooting

**Problem**: Forgot password

**Solution**:
1. Stop MultiWANBond
2. Edit `cmd/server/main.go`
3. Change `webConfig.Password = "newpassword123"`
4. Rebuild and restart

**Problem**: Can't access from other machines

**Solution**:
1. Change `webConfig.ListenAddr = "0.0.0.0"` (listen on all interfaces)
2. Configure firewall to allow remote access
3. **WARNING**: Only do this with authentication enabled and HTTPS!

---

## Production Checklist

Before deploying to production:

- [ ] Authentication enabled
- [ ] Strong unique password set
- [ ] Password stored securely (not in git)
- [ ] HTTPS/TLS enabled
- [ ] Firewall rules configured
- [ ] Access logs enabled
- [ ] Regular password rotation policy
- [ ] Backup access method documented

---

## Coming Soon

These features are planned for future releases:
- Auto-generate password during setup wizard
- Store credentials in config file
- Web-based password change interface
- Multi-user support with roles
- Session timeout configuration
- Two-factor authentication (2FA)
- API key authentication

---

For now, use the manual setup method above to secure your Web UI.

# MultiWANBond Installation Guide

## Prerequisites

### 1. Install Go

MultiWANBond requires Go 1.21 or later.

**Windows:**
1. Download Go from https://go.dev/dl/
2. Download the Windows installer (`.msi` file)
3. Run the installer
4. Verify installation:
   ```powershell
   go version
   ```

**Linux:**
```bash
# Download and install Go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz

# Add to PATH (add to ~/.bashrc or ~/.profile)
export PATH=$PATH:/usr/local/go/bin

# Verify
go version
```

**macOS:**
```bash
# Using Homebrew
brew install go

# Or download from https://go.dev/dl/
# Verify
go version
```

### 2. Platform-Specific Requirements

**Linux:**
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y build-essential ethtool iproute2

# For full functionality
sudo apt-get install -y iptables ipset

# Optional: for advanced features
sudo apt-get install -y wireguard openvpn
```

**Windows:**
- Run as Administrator for network configuration
- Windows 10/11 or Windows Server 2019/2022

**macOS:**
- macOS 10.15 (Catalina) or later
- Xcode Command Line Tools:
  ```bash
  xcode-select --install
  ```

## Building MultiWANBond

### 1. Clone the Repository

```bash
git clone https://github.com/thelastdreamer/MultiWANBond.git
cd MultiWANBond
```

### 2. Download Dependencies

```bash
go mod download
```

### 3. Build

**Single binary (server + client + web UI):**
```bash
# Current platform
go build -o multiwanbond ./cmd/server

# Or use Makefile
make build
```

**Cross-platform builds:**
```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o multiwanbond-linux-amd64 ./cmd/server

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -o multiwanbond-windows-amd64.exe ./cmd/server

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o multiwanbond-darwin-arm64 ./cmd/server

# All platforms
make build-all
```

### 4. Run Tests

**Network detection test:**
```bash
go run ./cmd/test/network_test.go
```

**Unit tests:**
```bash
go test ./...
```

**Integration tests:**
```bash
go test -tags=integration ./...
```

## Quick Start

### 1. Generate Default Configuration

```bash
./multiwanbond --generate-config > config.json
```

### 2. Edit Configuration

Edit `config.json` to configure your WANs and settings.

See [CONFIGURATION.md](CONFIGURATION.md) for details.

### 3. Start Server Mode

```bash
# Linux (requires elevated privileges for network management)
sudo ./multiwanbond -config config.json -mode server

# Windows (run as Administrator)
multiwanbond.exe -config config.json -mode server
```

### 4. Start Client Mode

```bash
sudo ./multiwanbond -config config.json -mode client
```

### 5. Access Web UI

Open browser and navigate to:
```
https://localhost:8080
```

Default credentials (first-time setup):
- Username: `admin`
- Password: (randomly generated, shown on first start)

## Systemd Service (Linux)

Create `/etc/systemd/system/multiwanbond.service`:

```ini
[Unit]
Description=MultiWANBond Network Bonding Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/multiwanbond -config /etc/multiwanbond/config.json
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable multiwanbond
sudo systemctl start multiwanbond
sudo systemctl status multiwanbond
```

## Windows Service

```powershell
# Install as service
sc create MultiWANBond binPath= "C:\Program Files\MultiWANBond\multiwanbond.exe -config C:\ProgramData\MultiWANBond\config.json" start= auto

# Start service
sc start MultiWANBond
```

## Docker (Future)

```bash
docker build -t multiwanbond .
docker run -d --name multiwanbond --net=host --cap-add=NET_ADMIN multiwanbond
```

## Troubleshooting

### "Permission denied" errors
- Run with `sudo` on Linux/macOS
- Run as Administrator on Windows
- Or set capabilities on Linux:
  ```bash
  sudo setcap cap_net_admin,cap_net_raw+ep ./multiwanbond
  ```

### "Module not found" errors
```bash
go mod tidy
go mod download
```

### Build errors
```bash
# Clean and rebuild
go clean -cache
go build -v ./cmd/server
```

### Network detection fails
- Check if `ethtool` is installed (Linux)
- Verify network interfaces exist:
  ```bash
  ip link show        # Linux
  ipconfig /all       # Windows
  ifconfig           # macOS
  ```

## Development Setup

### Install development tools

```bash
# Linter
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Code formatting
go install golang.org/x/tools/cmd/goimports@latest

# Testing
go install gotest.tools/gotestsum@latest
```

### Run in development mode

```bash
# With race detector and verbose logging
go run -race ./cmd/server -config config.json -log-level debug
```

### Hot reload during development

```bash
# Install air for hot reload
go install github.com/cosmtrek/air@latest

# Run with hot reload
air
```

## Next Steps

- [Quick Start Guide](QUICKSTART.md)
- [Configuration Reference](CONFIGURATION.md)
- [Network Setup Guide](NETWORK_SETUP.md)
- [API Documentation](API.md)
- [Troubleshooting](TROUBLESHOOTING.md)

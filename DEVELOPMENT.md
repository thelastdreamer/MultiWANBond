# MultiWANBond Development Guide

**Complete developer and contributor guide for MultiWANBond**

**Version**: 1.1
**Last Updated**: November 2, 2025

---

## Table of Contents

- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Project Structure](#project-structure)
- [Building from Source](#building-from-source)
- [Running Tests](#running-tests)
- [Code Style Guide](#code-style-guide)
- [Contributing Guidelines](#contributing-guidelines)
- [Adding New Features](#adding-new-features)
- [Debugging](#debugging)
- [Performance Profiling](#performance-profiling)
- [Release Process](#release-process)

---

## Getting Started

### Prerequisites

**Required**:
- **Go 1.21 or later**: [Download](https://go.dev/dl/)
- **Git**: [Download](https://git-scm.com/downloads)
- **Basic networking knowledge**

**Platform-Specific**:
- **Linux**: `libnetlink` development headers
  ```bash
  sudo apt-get install build-essential  # Debian/Ubuntu
  sudo yum install gcc make             # RHEL/CentOS
  ```

- **Windows**: No additional requirements
- **macOS**: Xcode Command Line Tools
  ```bash
  xcode-select --install
  ```

### Cloning the Repository

```bash
# Clone via HTTPS
git clone https://github.com/thelastdreamer/MultiWANBond.git

# Or via SSH (if you have SSH keys set up)
git clone git@github.com:thelastdreamer/MultiWANBond.git

# Navigate to project directory
cd MultiWANBond
```

### Installing Dependencies

```bash
# Download all Go module dependencies
go mod download

# Verify dependencies
go mod verify

# Tidy up (removes unused dependencies)
go mod tidy
```

---

## Development Environment

### Recommended IDE/Editors

**Visual Studio Code** (Recommended):
- Install [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go)
- Automatic formatting, linting, debugging
- Integrated terminal

**GoLand** (JetBrains):
- Full-featured Go IDE
- Built-in debugger, profiler
- 30-day free trial

**Vim/Neovim**:
- Install [vim-go](https://github.com/fatih/vim-go) plugin
- Lightweight, fast
- Learning curve

### VS Code Configuration

Create `.vscode/settings.json`:

```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintOnSave": "package",
  "go.formatTool": "goimports",
  "editor.formatOnSave": true,
  "go.testFlags": ["-v", "-race"],
  "go.coverOnSave": true,
  "go.coverageDecorator": {
    "type": "gutter"
  }
}
```

Create `.vscode/launch.json` for debugging:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch Server",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/server/main.go",
      "args": ["--config", "config.json"]
    },
    {
      "name": "Attach to Process",
      "type": "go",
      "request": "attach",
      "mode": "local",
      "processId": "${command:pickProcess}"
    }
  ]
}
```

### Installing Development Tools

```bash
# golangci-lint (meta-linter)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# goimports (auto-add imports)
go install golang.org/x/tools/cmd/goimports@latest

# air (live reload)
go install github.com/cosmtrek/air@latest

# dlv (debugger)
go install github.com/go-delve/delve/cmd/dlv@latest

# staticcheck (static analysis)
go install honnef.co/go/tools/cmd/staticcheck@latest
```

### Configuring Git

```bash
# Set your identity
git config user.name "Your Name"
git config user.email "your.email@example.com"

# Set up line endings (Windows)
git config core.autocrlf true

# Set up line endings (Linux/macOS)
git config core.autocrlf input
```

---

## Project Structure

```
MultiWANBond/
│
├── cmd/                           # Command-line applications
│   ├── server/
│   │   └── main.go               # Server entry point
│   ├── client/
│   │   └── main.go               # Client entry point (future)
│   └── test/
│       ├── network_detect.go     # Network detection tests
│       ├── health_checker.go     # Health check tests
│       └── final_integration.go  # Integration tests
│
├── pkg/                           # Library packages (reusable)
│   ├── bonder/
│   │   ├── bonder.go             # Core bonding logic
│   │   ├── session.go            # Session management
│   │   └── wan.go                # WAN interface management
│   │
│   ├── nat/
│   │   ├── manager.go            # NAT traversal manager
│   │   ├── stun.go               # STUN client (RFC 5389)
│   │   ├── cgnat.go              # CGNAT detector
│   │   ├── holepunch.go          # UDP hole punching
│   │   └── relay.go              # TURN relay client
│   │
│   ├── dpi/
│   │   ├── classifier.go         # DPI classifier
│   │   ├── protocols.go          # Protocol definitions
│   │   ├── flow.go               # Flow tracking
│   │   └── signatures.go         # Protocol signatures
│   │
│   ├── health/
│   │   ├── monitor.go            # Health monitor
│   │   ├── checker.go            # Check implementations
│   │   └── adaptive.go           # Adaptive intervals
│   │
│   ├── routing/
│   │   ├── router.go             # Routing engine
│   │   ├── policy.go             # Policy routing (Linux)
│   │   └── loadbalancer.go       # Load balancing modes
│   │
│   ├── metrics/
│   │   ├── collector.go          # Metrics collector
│   │   ├── timeseries.go         # Time-series storage
│   │   └── exporter.go           # Export formats
│   │
│   ├── webui/
│   │   ├── server.go             # HTTP/WebSocket server
│   │   ├── api.go                # REST API handlers
│   │   └── websocket.go          # WebSocket handler
│   │
│   ├── processor/
│   │   ├── processor.go          # Packet processor
│   │   ├── encap.go              # Encapsulation
│   │   └── reorder.go            # Reordering buffer
│   │
│   ├── fec/
│   │   └── fec.go                # Reed-Solomon FEC
│   │
│   └── config/
│       └── config.go             # Configuration management
│
├── webui/                         # Web UI files (HTML/CSS/JS)
│   ├── login.html
│   ├── dashboard.html
│   ├── flows.html
│   ├── analytics.html
│   ├── logs.html
│   └── config.html
│
├── docs/                          # Documentation (Markdown)
│   ├── README.md
│   ├── ARCHITECTURE.md
│   ├── API_REFERENCE.md
│   ├── DEVELOPMENT.md            # This file
│   └── ...
│
├── scripts/                       # Build and utility scripts
│   ├── build-releases.sh         # Multi-platform build (Linux/macOS)
│   ├── build-releases.ps1        # Multi-platform build (Windows)
│   └── install.sh                # One-click installer
│
├── go.mod                         # Go module definition
├── go.sum                         # Dependency checksums
├── LICENSE                        # MIT License
└── README.md                      # Main documentation
```

### Package Responsibilities

| Package | Responsibility | Dependencies |
|---------|---------------|--------------|
| `bonder` | Core bonding orchestration | `nat`, `dpi`, `health`, `routing` |
| `nat` | NAT traversal, P2P | None (standalone) |
| `dpi` | Deep packet inspection | None (standalone) |
| `health` | WAN health monitoring | None (standalone) |
| `routing` | Traffic routing, load balancing | None (standalone) |
| `metrics` | Metrics collection, export | None (standalone) |
| `webui` | Web interface, REST API | `bonder` (via interface) |
| `processor` | Packet processing, encryption | `fec` |
| `fec` | Forward error correction | None (standalone) |
| `config` | Configuration management | None (standalone) |

**Design Principle**: Packages should be **loosely coupled** and **highly cohesive**. Each package should be usable independently where possible.

---

## Building from Source

### Quick Build

**Development build** (fast, no optimizations):
```bash
go build -o multiwanbond cmd/server/main.go
```

**Production build** (optimized, smaller binary):
```bash
go build -ldflags="-s -w" -o multiwanbond cmd/server/main.go
```

**Flags Explained**:
- `-o`: Output filename
- `-ldflags="-s -w"`: Strip debug info and symbol table (smaller binary)

### Platform-Specific Builds

**Windows (from Windows)**:
```powershell
go build -o multiwanbond.exe cmd/server/main.go
```

**Linux (from Linux)**:
```bash
go build -o multiwanbond cmd/server/main.go
```

**macOS (from macOS)**:
```bash
go build -o multiwanbond cmd/server/main.go
```

### Cross-Compilation

**Build for Linux from Windows**:
```powershell
$env:GOOS="linux"; $env:GOARCH="amd64"
go build -o multiwanbond-linux-amd64 cmd/server/main.go
```

**Build for Windows from Linux**:
```bash
GOOS=windows GOARCH=amd64 go build -o multiwanbond-windows-amd64.exe cmd/server/main.go
```

**Build for macOS from Linux**:
```bash
GOOS=darwin GOARCH=amd64 go build -o multiwanbond-darwin-amd64 cmd/server/main.go
```

### Multi-Platform Build Script

**Linux/macOS** (`scripts/build-releases.sh`):
```bash
#!/bin/bash
platforms=(
  "windows/amd64"
  "windows/arm64"
  "linux/amd64"
  "linux/arm64"
  "linux/arm"
  "darwin/amd64"
  "darwin/arm64"
)

for platform in "${platforms[@]}"; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  output="multiwanbond-${GOOS}-${GOARCH}"

  if [ "$GOOS" = "windows" ]; then
    output="${output}.exe"
  fi

  echo "Building for $GOOS/$GOARCH..."
  GOOS=$GOOS GOARCH=$GOARCH go build -ldflags="-s -w" -o "dist/$output" cmd/server/main.go
done

echo "Build complete! Binaries in dist/"
```

**Usage**:
```bash
chmod +x scripts/build-releases.sh
./scripts/build-releases.sh
```

---

## Running Tests

### Unit Tests

**Run all tests**:
```bash
go test ./...
```

**Run tests with coverage**:
```bash
go test -cover ./...
```

**Run tests with detailed coverage**:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # View in browser
```

**Run tests with race detector**:
```bash
go test -race ./...
```

**Run specific package tests**:
```bash
go test ./pkg/bonder/
go test ./pkg/nat/
```

### Integration Tests

**Network detection test**:
```bash
go run cmd/test/network_detect.go
```

**Health checker test**:
```bash
go run cmd/test/health_checker.go
```

**Final integration test**:
```bash
go run cmd/test/final_integration.go
```

### Test Scripts

**Windows** (`run-tests.bat`):
```batch
@echo off
echo Running MultiWANBond Tests...

REM Unit tests
echo.
echo [1/3] Running unit tests...
go test ./...

REM Network detection
echo.
echo [2/3] Running network detection test...
go run cmd/test/network_detect.go

REM Health checker
echo.
echo [3/3] Running health checker test...
go run cmd/test/health_checker.go

echo.
echo All tests complete!
pause
```

**Linux/macOS** (`run-tests.sh`):
```bash
#!/bin/bash
echo "Running MultiWANBond Tests..."

# Unit tests
echo ""
echo "[1/3] Running unit tests..."
go test ./...

# Network detection
echo ""
echo "[2/3] Running network detection test..."
go run cmd/test/network_detect.go

# Health checker
echo ""
echo "[3/3] Running health checker test..."
go run cmd/test/health_checker.go

echo ""
echo "All tests complete!"
```

### Writing Tests

**Example test file** (`pkg/bonder/bonder_test.go`):

```go
package bonder

import (
    "testing"
)

func TestBonderCreation(t *testing.T) {
    config := &Config{
        Mode: "client",
    }

    b, err := New(config)
    if err != nil {
        t.Fatalf("Failed to create bonder: %v", err)
    }

    if b == nil {
        t.Fatal("Bonder is nil")
    }
}

func TestWANAddition(t *testing.T) {
    b := &Bonder{
        wans: make(map[uint8]*WANInterface),
    }

    wan := &WANInterface{
        ID:      1,
        Name:    "Test WAN",
        Enabled: true,
    }

    b.AddWAN(wan)

    if len(b.wans) != 1 {
        t.Errorf("Expected 1 WAN, got %d", len(b.wans))
    }
}
```

**Test Naming Convention**:
- Test files: `*_test.go`
- Test functions: `Test<FunctionName>(t *testing.T)`
- Benchmark functions: `Benchmark<FunctionName>(b *testing.B)`
- Example functions: `Example<FunctionName>()`

---

## Code Style Guide

### General Principles

1. **Follow Go conventions**: Use `gofmt` and `goimports`
2. **Keep it simple**: Prefer clarity over cleverness
3. **Document public APIs**: All exported symbols must have comments
4. **Error handling**: Always check and handle errors
5. **No panics in libraries**: Return errors instead

### Formatting

**Automatic formatting**:
```bash
# Format all files
gofmt -w .

# Format and fix imports
goimports -w .
```

**Line length**: Prefer <120 characters, hard limit 140

**Indentation**: Tabs (default in Go)

### Naming Conventions

**Variables**:
```go
// Good
userID := 123
maxConnections := 100

// Bad
user_id := 123  // No underscores
max_connections := 100
```

**Functions**:
```go
// Exported (public)
func GetWANs() []WAN

// Unexported (private)
func parseConfig() error
```

**Constants**:
```go
const (
    // Exported
    DefaultPort = 8080

    // Unexported
    maxRetries = 3
)
```

**Interfaces**:
```go
// Single-method interfaces end with "-er"
type Reader interface {
    Read(p []byte) (n int, err error)
}

// Multi-method interfaces don't have suffix
type File interface {
    Read(p []byte) (n int, err error)
    Write(p []byte) (n int, err error)
    Close() error
}
```

### Comments

**Package comments** (`bonder/bonder.go`):
```go
// Package bonder provides core WAN bonding functionality.
//
// The bonder package orchestrates multiple WAN interfaces, distributes
// traffic across them, and handles failover when WANs fail.
//
// Example usage:
//
//     config := &Config{Mode: "client"}
//     b, err := New(config)
//     if err != nil {
//         log.Fatal(err)
//     }
//     b.Start()
//
package bonder
```

**Function comments**:
```go
// GetWANs returns a map of all configured WAN interfaces.
// The returned map is a copy and safe for concurrent access.
func (b *Bonder) GetWANs() map[uint8]*WANInterface {
    b.mu.RLock()
    defer b.mu.RUnlock()

    wans := make(map[uint8]*WANInterface, len(b.wans))
    for id, wan := range b.wans {
        wans[id] = wan
    }
    return wans
}
```

**Struct comments**:
```go
// Bonder manages multiple WAN interfaces and distributes traffic across them.
type Bonder struct {
    // config is the bonder configuration
    config *Config

    // wans is a map of WAN ID to WAN interface
    wans map[uint8]*WANInterface

    // mu protects concurrent access to wans
    mu sync.RWMutex
}
```

### Error Handling

**Good error handling**:
```go
// Return errors, don't panic
func ParseConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }

    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    return &cfg, nil
}
```

**Error wrapping** (Go 1.13+):
```go
// Use %w to wrap errors (preserves error chain)
return fmt.Errorf("failed to connect to WAN 1: %w", err)

// Use %v if you don't want wrapping
return fmt.Errorf("failed to connect: %v", err)
```

### Concurrency

**Use mutexes for shared data**:
```go
type Counter struct {
    value int
    mu    sync.Mutex
}

func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.value++
}

func (c *Counter) Value() int {
    c.mu.Lock()
    defer c.mu.Unlock()
    return c.value
}
```

**Use channels for communication**:
```go
// Good: Pass messages via channels
events := make(chan Event, 100)
go producer(events)
go consumer(events)

// Bad: Share memory
var events []Event
var mu sync.Mutex
```

**Use sync.WaitGroup for goroutine coordination**:
```go
var wg sync.WaitGroup

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        // Work here
    }(i)
}

wg.Wait()  // Wait for all goroutines
```

### Performance

**Use buffer pools for frequent allocations**:
```go
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 1500)
    },
}

// Get buffer
buf := bufferPool.Get().([]byte)
defer bufferPool.Put(buf)

// Use buffer
// ...
```

**Avoid unnecessary allocations**:
```go
// Good: Reuse slice
wans := make([]WAN, 0, 10)
for _, wan := range allWANs {
    if wan.Enabled {
        wans = append(wans, wan)
    }
}

// Bad: Create new slice each time
var wans []WAN
for _, wan := range allWANs {
    if wan.Enabled {
        wans = append(wans, wan)
    }
}
```

---

## Contributing Guidelines

### Before You Start

1. **Check existing issues**: Search for similar feature requests or bugs
2. **Open an issue**: Discuss your idea before starting work
3. **Fork the repository**: Work in your own fork
4. **Create a branch**: Use descriptive branch names

### Workflow

**1. Fork and clone**:
```bash
# Fork on GitHub (click Fork button)
git clone https://github.com/YOUR_USERNAME/MultiWANBond.git
cd MultiWANBond
git remote add upstream https://github.com/thelastdreamer/MultiWANBond.git
```

**2. Create a feature branch**:
```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-123
```

**Branch naming**:
- `feature/add-quic-support`
- `fix/issue-123-wan-detection`
- `docs/improve-readme`
- `refactor/simplify-routing`

**3. Make your changes**:
- Write code
- Add tests
- Update documentation
- Run tests locally

**4. Commit your changes**:
```bash
git add .
git commit -m "Add QUIC protocol support"
```

**Commit message format**:
```
<type>: <subject>

<body>

<footer>
```

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation change
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance

**Example commit**:
```
feat: Add QUIC protocol support

Implements QUIC as an alternative to UDP for reduced latency
and better NAT traversal. QUIC provides built-in encryption
and connection migration.

- Add quic package with QUIC client/server
- Integrate QUIC with bonder
- Add configuration options for QUIC
- Update documentation

Closes #45
```

**5. Push to your fork**:
```bash
git push origin feature/your-feature-name
```

**6. Create Pull Request**:
- Go to GitHub
- Click "New Pull Request"
- Select your branch
- Fill in PR template
- Submit

### Pull Request Guidelines

**PR Title**: Clear and descriptive
- ✅ "Add QUIC protocol support"
- ✅ "Fix WAN detection on Windows"
- ❌ "Update"
- ❌ "Fix bug"

**PR Description**: Include:
- What problem does this solve?
- How does it solve it?
- Any breaking changes?
- Screenshots (for UI changes)
- Related issues

**Checklist**:
- [ ] Code follows style guide
- [ ] Tests added and passing
- [ ] Documentation updated
- [ ] No breaking changes (or clearly documented)
- [ ] Commit messages are clear

### Code Review Process

1. **Automated checks run**: Tests, linting, build
2. **Maintainer reviews code**: May request changes
3. **Address feedback**: Make requested changes
4. **Approval**: Maintainer approves PR
5. **Merge**: PR is merged into main branch

**Tips for faster review**:
- Keep PRs small and focused
- Write clear commit messages
- Add tests
- Update documentation
- Respond to feedback promptly

---

## Adding New Features

### Example: Adding a New Load Balancing Mode

**1. Define the mode** (`pkg/routing/loadbalancer.go`):
```go
const (
    // ... existing modes ...

    // LoadBalanceModeMinimumHops routes to WAN with fewest network hops
    LoadBalanceModeMinimumHops LoadBalanceMode = "minimum-hops"
)
```

**2. Implement the algorithm** (`pkg/routing/router.go`):
```go
func (r *Router) selectWANMinimumHops(wans []WAN) *WAN {
    var bestWAN *WAN
    minHops := math.MaxInt

    for _, wan := range wans {
        if wan.Hops < minHops {
            minHops = wan.Hops
            bestWAN = &wan
        }
    }

    return bestWAN
}
```

**3. Add to router** (`pkg/routing/router.go`):
```go
func (r *Router) SelectWAN(packet Packet, wans []WAN) *WAN {
    switch r.mode {
    case LoadBalanceModeMinimumHops:
        return r.selectWANMinimumHops(wans)
    // ... other cases ...
    }
}
```

**4. Write tests** (`pkg/routing/router_test.go`):
```go
func TestMinimumHopsSelection(t *testing.T) {
    wans := []WAN{
        {ID: 1, Hops: 10},
        {ID: 2, Hops: 5},
        {ID: 3, Hops: 15},
    }

    router := &Router{mode: LoadBalanceModeMinimumHops}
    wan := router.SelectWAN(nil, wans)

    if wan.ID != 2 {
        t.Errorf("Expected WAN 2 (minimum hops), got WAN %d", wan.ID)
    }
}
```

**5. Update configuration** (`pkg/config/config.go`):
```go
// Validate load balancing mode
func (c *Config) Validate() error {
    validModes := []string{
        "round-robin",
        "weighted",
        "least-used",
        "least-latency",
        "per-flow",
        "adaptive",
        "minimum-hops",  // Add new mode
    }

    // ... validation logic ...
}
```

**6. Update documentation**:
- Add to [README.md](README.md) feature list
- Add to [ARCHITECTURE.md](ARCHITECTURE.md) routing section
- Add to [WEB_UI_USER_GUIDE.md](WEB_UI_USER_GUIDE.md) configuration section

**7. Update Web UI** (`webui/config.html`):
```html
<select id="loadBalanceMode">
    <option value="round-robin">Round-Robin</option>
    <option value="weighted">Weighted</option>
    <!-- ... -->
    <option value="minimum-hops">Minimum Hops</option>
</select>
```

**8. Create Pull Request**: Follow contributing guidelines above

---

## Debugging

### Using Delve Debugger

**Install**:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

**Start debugging**:
```bash
dlv debug cmd/server/main.go -- --config config.json
```

**Common commands**:
```
(dlv) break main.main       # Set breakpoint
(dlv) break bonder.go:123   # Set breakpoint at line
(dlv) continue              # Continue execution
(dlv) next                  # Step over
(dlv) step                  # Step into
(dlv) print var             # Print variable
(dlv) goroutines            # List goroutines
(dlv) goroutine 5           # Switch to goroutine 5
```

### Logging

**Use standard log package**:
```go
import "log"

log.Println("WAN 1 health check successful")
log.Printf("Latency: %.2fms", latency)
log.Fatal("Failed to start server:", err)
```

**Log levels** (future enhancement):
```go
// pkg/logger/logger.go
package logger

type Level int

const (
    LevelDebug Level = iota
    LevelInfo
    LevelWarn
    LevelError
)

func Debug(msg string) {
    if level <= LevelDebug {
        log.Println("[DEBUG]", msg)
    }
}
```

### Common Issues

**"undefined: netlink"**:
- Platform: Linux
- Solution: Missing `vishvananda/netlink` package
- Fix: `go get github.com/vishvananda/netlink`

**Race condition detected**:
- Solution: Use mutexes or channels
- Test: `go test -race ./...`

**Out of memory**:
- Solution: Use buffer pools, limit concurrent operations
- Profile: `go test -memprofile=mem.out`

---

## Performance Profiling

### CPU Profiling

**1. Add profiling to code**:
```go
import _ "net/http/pprof"

go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

**2. Run application**:
```bash
go run cmd/server/main.go
```

**3. Capture profile** (30 seconds):
```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

**4. Analyze**:
```
(pprof) top10        # Top 10 functions by CPU time
(pprof) list <func>  # Show source for function
(pprof) web          # Visualize (requires graphviz)
```

### Memory Profiling

**Capture heap profile**:
```bash
go tool pprof http://localhost:6060/debug/pprof/heap
```

**Analyze**:
```
(pprof) top10               # Top 10 allocations
(pprof) list <func>         # Show source
(pprof) png > profile.png   # Export visualization
```

### Benchmarking

**Write benchmarks** (`pkg/routing/router_bench_test.go`):
```go
func BenchmarkRouteSelection(b *testing.B) {
    router := NewRouter(LoadBalanceModeAdaptive)
    wans := createTestWANs(10)
    packet := createTestPacket()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        router.SelectWAN(packet, wans)
    }
}
```

**Run benchmarks**:
```bash
go test -bench=. ./pkg/routing/
```

**Output**:
```
BenchmarkRouteSelection-8   1000000   1250 ns/op   128 B/op   2 allocs/op
```

**Interpretation**:
- 1,000,000 iterations
- 1,250 nanoseconds per operation
- 128 bytes allocated per operation
- 2 allocations per operation

---

## Release Process

### Version Numbering

**Semantic Versioning** (MAJOR.MINOR.PATCH):
- **MAJOR**: Breaking changes (v1.0.0 → v2.0.0)
- **MINOR**: New features, backward compatible (v1.0.0 → v1.1.0)
- **PATCH**: Bug fixes (v1.0.0 → v1.0.1)

### Release Checklist

**1. Version bump**:
```bash
# Update version in README.md
sed -i 's/version-1.0.0/version-1.1.0/g' README.md

# Update version in code
# (Update constants, documentation, etc.)
```

**2. Update changelog** (`CHANGELOG.md`):
```markdown
## [1.1.0] - 2025-11-02

### Added
- Unified Web UI with cookie-based sessions
- NAT traversal integration with Web UI
- DPI flow analysis page
- Traffic analytics with Chart.js

### Fixed
- WAN detection on Windows ARM64
- Session timeout handling

### Changed
- Improved health check performance by 20%
```

**3. Run full test suite**:
```bash
go test ./...
go test -race ./...
```

**4. Build for all platforms**:
```bash
./scripts/build-releases.sh
```

**5. Create Git tag**:
```bash
git tag -a v1.1.0 -m "Release v1.1.0 - Unified Web UI"
git push origin v1.1.0
```

**6. Create GitHub Release**:
- Go to GitHub Releases
- Click "Create a new release"
- Select tag v1.1.0
- Title: "MultiWANBond v1.1.0"
- Description: Copy from CHANGELOG.md
- Attach binaries from `dist/`
- Click "Publish release"

**7. Update documentation**:
- Ensure all docs reflect new version
- Update installation instructions if needed

---

## Best Practices

### Code Organization

- **One concept per package**: Each package should do one thing well
- **Avoid circular dependencies**: Package A should not depend on Package B if B depends on A
- **Interface-based design**: Define interfaces, depend on abstractions
- **Test coverage**: Aim for >80% coverage on critical packages

### Security

- **Never commit secrets**: Use environment variables or separate config
- **Validate all inputs**: Especially from network and user input
- **Use secure random**: `crypto/rand`, not `math/rand`
- **Keep dependencies updated**: Regularly run `go get -u` and test

### Performance

- **Profile before optimizing**: Don't guess, measure
- **Benchmark changes**: Ensure optimizations actually help
- **Avoid premature optimization**: Readability > micro-optimizations
- **Use appropriate data structures**: Map for lookups, slice for iteration

### Documentation

- **Document public APIs**: Every exported symbol
- **Write examples**: Show how to use your code
- **Keep docs updated**: Change code → change docs
- **Write clear commit messages**: Future you will thank you

---

## Resources

### Go Language

- [Go Documentation](https://go.dev/doc/)
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Blog](https://go.dev/blog/)

### MultiWANBond Docs

- [README.md](README.md) - Project overview
- [ARCHITECTURE.md](ARCHITECTURE.md) - System architecture
- [API_REFERENCE.md](API_REFERENCE.md) - Web UI API
- [WEB_UI_USER_GUIDE.md](WEB_UI_USER_GUIDE.md) - End-user guide

### Community

- [GitHub Issues](https://github.com/thelastdreamer/MultiWANBond/issues)
- [GitHub Discussions](https://github.com/thelastdreamer/MultiWANBond/discussions)

---

**Questions?** Open an issue or start a discussion on GitHub!

**Last Updated**: November 2, 2025
**Version**: 1.1
**MultiWANBond Version**: 1.1

# Project Structure

## Directory Layout

```
MultiWANBond/
├── cmd/                          # Application entry points
│   ├── server/                   # Server executable
│   │   └── main.go              # Server main program
│   └── client/                   # Client executable
│       └── main.go              # Client main program
│
├── pkg/                          # Public packages (can be imported)
│   ├── protocol/                 # Core protocol definitions
│   │   ├── types.go             # Protocol types and structures
│   │   └── interfaces.go        # Protocol interfaces
│   │
│   ├── bonder/                   # Main bonding implementation
│   │   └── bonder.go            # Bonder core logic
│   │
│   ├── health/                   # Health monitoring
│   │   └── checker.go           # Health checker implementation
│   │
│   ├── router/                   # Packet routing
│   │   └── router.go            # Routing logic and algorithms
│   │
│   ├── packet/                   # Packet processing
│   │   └── processor.go         # Encoding, decoding, reordering
│   │
│   ├── fec/                      # Forward Error Correction
│   │   └── reedsolomon.go       # Reed-Solomon FEC implementation
│   │
│   ├── plugin/                   # Plugin system
│   │   └── manager.go           # Plugin manager and base classes
│   │
│   └── config/                   # Configuration management
│       └── config.go            # Config loading and parsing
│
├── configs/                      # Configuration examples
│   ├── example.json             # Full-featured example
│   └── simple.json              # Simple two-WAN example
│
├── docs/                         # Documentation
│   ├── ARCHITECTURE.md          # Architecture documentation
│   ├── QUICKSTART.md            # Quick start guide
│   └── PROJECT_STRUCTURE.md     # This file
│
├── build/                        # Build output (generated)
│   ├── linux/                   # Linux binaries
│   ├── windows/                 # Windows binaries
│   ├── darwin/                  # macOS binaries
│   └── arm/                     # ARM binaries
│
├── .gitignore                    # Git ignore rules
├── .gitattributes               # Git attributes
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums (generated)
├── Makefile                      # Build automation
├── README.md                     # Main documentation
└── LICENSE                       # MIT License
```

## Package Descriptions

### cmd/

Contains application entry points (main packages). These are the executables that users run.

- **server**: Receives connections and can act as a bonding endpoint
- **client**: Initiates connections and demonstrates usage

### pkg/protocol/

Core protocol definitions that all other packages depend on.

- **types.go**: Defines all protocol structures, constants, and enums
- **interfaces.go**: Defines interfaces for components (Bonder, Router, etc.)

This package has no dependencies on other pkg/ packages.

### pkg/bonder/

Main implementation that orchestrates all components.

**Dependencies:** All other pkg/ packages

**Key Responsibilities:**
- Session management
- Component lifecycle
- Data flow coordination
- Event handling

### pkg/health/

Health monitoring for WAN connections.

**Dependencies:** pkg/protocol

**Key Responsibilities:**
- Periodic health checks
- Latency/jitter/loss measurement
- State machine management
- Event generation

### pkg/router/

Intelligent packet routing across WANs.

**Dependencies:** pkg/protocol

**Key Responsibilities:**
- Routing decision algorithms
- Load balancing
- Bandwidth tracking
- Flow mapping

### pkg/packet/

Packet-level operations.

**Dependencies:** pkg/protocol

**Key Responsibilities:**
- Packet encoding/decoding
- Sequence management
- Reordering logic
- Duplicate detection

### pkg/fec/

Forward Error Correction.

**Dependencies:** pkg/protocol

**Key Responsibilities:**
- Reed-Solomon encoding
- Data recovery from FEC
- Redundancy calculation

### pkg/plugin/

Plugin system for extensibility.

**Dependencies:** pkg/protocol

**Key Responsibilities:**
- Plugin lifecycle management
- Filter chain execution
- Metrics collection coordination
- Alert distribution

### pkg/config/

Configuration management.

**Dependencies:** pkg/protocol

**Key Responsibilities:**
- JSON parsing
- Config validation
- Hot reload support
- Type conversion

## Dependency Graph

```
                    cmd/server, cmd/client
                            │
                            ▼
                      pkg/bonder/
                            │
         ┌──────────────────┼──────────────────┐
         │                  │                  │
         ▼                  ▼                  ▼
    pkg/health/        pkg/router/       pkg/packet/
         │                  │                  │
         │                  ▼                  │
         │              pkg/fec/               │
         │                  │                  │
         │                  ▼                  │
         └──────────► pkg/plugin/ ◄───────────┘
                            │
                            ▼
                      pkg/config/
                            │
                            ▼
                      pkg/protocol/
                      (no dependencies)
```

## Build Artifacts

After building, the `build/` directory contains platform-specific binaries:

```
build/
├── linux/
│   ├── multiwanbond-server-amd64
│   ├── multiwanbond-client-amd64
│   ├── multiwanbond-server-arm64
│   └── multiwanbond-client-arm64
│
├── windows/
│   ├── multiwanbond-server-amd64.exe
│   └── multiwanbond-client-amd64.exe
│
├── darwin/
│   ├── multiwanbond-server-amd64
│   ├── multiwanbond-client-amd64
│   ├── multiwanbond-server-arm64    # Apple Silicon
│   └── multiwanbond-client-arm64
│
└── arm/
    ├── multiwanbond-server-armv7    # Raspberry Pi, etc.
    ├── multiwanbond-client-armv7
    ├── multiwanbond-server-arm64
    └── multiwanbond-client-arm64
```

## Adding New Components

### Adding a New Package

1. Create directory under `pkg/`
2. Define interfaces in `pkg/protocol/interfaces.go` if needed
3. Implement the package
4. Import in `pkg/bonder/bonder.go`
5. Update this documentation

Example:
```bash
mkdir -p pkg/mycomponent
touch pkg/mycomponent/mycomponent.go
```

### Adding a Plugin

1. Create plugin in `plugins/` directory (future)
2. Implement `plugin.Plugin` interface
3. Register in configuration
4. Load via plugin manager

### Adding a New Routing Mode

1. Add enum value in `pkg/protocol/types.go`
2. Implement algorithm in `pkg/router/router.go`
3. Add to `Route()` switch statement
4. Document in README.md

## Code Organization Principles

### Separation of Concerns

Each package has a single, well-defined responsibility:
- protocol: definitions only
- bonder: orchestration only
- health: monitoring only
- router: routing only
- etc.

### Interface-Driven Design

All major components are defined by interfaces in `pkg/protocol/interfaces.go`. This allows:
- Easy testing with mocks
- Component substitution
- Plugin development

### Minimal Dependencies

Packages depend only on what they need:
- All packages depend on `pkg/protocol`
- Higher-level packages depend on lower-level
- No circular dependencies

### Cross-Platform by Default

All code is pure Go with no platform-specific imports in core packages. Platform-specific optimizations go in separate files with build tags.

## Testing Structure

```
pkg/
├── protocol/
│   ├── types.go
│   ├── types_test.go
│   ├── interfaces.go
│   └── interfaces_test.go
│
├── bonder/
│   ├── bonder.go
│   ├── bonder_test.go
│   └── bonder_integration_test.go
│
└── ...
```

Test files live alongside implementation files with `_test.go` suffix.

## Configuration Files

Configuration examples are in `configs/`:

- **example.json**: Full-featured configuration showing all options
- **simple.json**: Minimal configuration for quick start

Users typically create their own config files based on these examples.

## Documentation

All documentation is in Markdown format in `docs/`:

- **ARCHITECTURE.md**: Deep dive into architecture
- **QUICKSTART.md**: Get up and running quickly
- **PROJECT_STRUCTURE.md**: This file
- **API.md**: Generated API documentation (future)

## Future Directories

As the project grows, these directories may be added:

```
├── internal/              # Private packages (cannot be imported)
│   └── platform/         # Platform-specific code
│
├── plugins/              # Built-in plugin implementations
│   ├── logger/
│   ├── prometheus/
│   └── compression/
│
├── test/                 # Integration tests
│   ├── fixtures/
│   └── helpers/
│
├── examples/             # Example applications
│   ├── simple/
│   └── advanced/
│
└── scripts/              # Utility scripts
    ├── setup.sh
    └── benchmark.sh
```

## Build System

The Makefile provides targets for all common tasks:

```bash
make build          # Build for current platform
make build-all      # Build for all platforms
make test           # Run tests
make clean          # Clean artifacts
make install        # Install to system
```

See `make help` for all available targets.

## Version Control

### Ignored Files (.gitignore)

- Build artifacts (`build/`)
- Test outputs (`*.test`, `coverage.out`)
- User configs (`config.json`)
- IDE files (`.vscode/`, `.idea/`)
- OS files (`.DS_Store`)

### Included Files

- Source code (`*.go`)
- Example configs (`configs/*.json`)
- Documentation (`*.md`)
- Build configuration (`Makefile`, `go.mod`)

## Getting Started with Development

1. Clone repository
2. Run `go mod download` to get dependencies
3. Run `make build` to build
4. Run `make test` to verify
5. Start coding!

See [CONTRIBUTING.md](../CONTRIBUTING.md) for development guidelines.

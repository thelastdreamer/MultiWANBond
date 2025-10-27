.PHONY: all build clean test install build-all build-linux build-windows build-darwin build-arm help

# Variables
BINARY_NAME=multiwanbond
SERVER_BINARY=multiwanbond-server
CLIENT_BINARY=multiwanbond-client
VERSION?=1.0.0
BUILD_DIR=build
GO=go
GOFLAGS=-ldflags="-s -w -X main.Version=$(VERSION)"

# Default target
all: build

# Help target
help:
	@echo "MultiWANBond Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  make build          - Build for current platform"
	@echo "  make build-all      - Build for all platforms"
	@echo "  make build-linux    - Build for Linux (amd64, arm64)"
	@echo "  make build-windows  - Build for Windows (amd64)"
	@echo "  make build-darwin   - Build for macOS (amd64, arm64)"
	@echo "  make build-arm      - Build for ARM devices"
	@echo "  make test           - Run tests"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make install        - Install to system"
	@echo ""

# Build for current platform
build:
	@echo "Building for current platform..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY) ./cmd/server
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(CLIENT_BINARY) ./cmd/client
	@echo "Build complete: $(BUILD_DIR)/"

# Build for all platforms
build-all: build-linux build-windows build-darwin

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)/linux
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/linux/$(SERVER_BINARY)-amd64 ./cmd/server
	GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/linux/$(CLIENT_BINARY)-amd64 ./cmd/client
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/linux/$(SERVER_BINARY)-arm64 ./cmd/server
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/linux/$(CLIENT_BINARY)-arm64 ./cmd/client
	@echo "Linux builds complete"

# Build for Windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)/windows
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/windows/$(SERVER_BINARY)-amd64.exe ./cmd/server
	GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/windows/$(CLIENT_BINARY)-amd64.exe ./cmd/client
	@echo "Windows builds complete"

# Build for macOS
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)/darwin
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/darwin/$(SERVER_BINARY)-amd64 ./cmd/server
	GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/darwin/$(CLIENT_BINARY)-amd64 ./cmd/client
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/darwin/$(SERVER_BINARY)-arm64 ./cmd/server
	GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/darwin/$(CLIENT_BINARY)-arm64 ./cmd/client
	@echo "macOS builds complete"

# Build for ARM devices (Raspberry Pi, etc.)
build-arm:
	@echo "Building for ARM devices..."
	@mkdir -p $(BUILD_DIR)/arm
	GOOS=linux GOARCH=arm GOARM=7 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/arm/$(SERVER_BINARY)-armv7 ./cmd/server
	GOOS=linux GOARCH=arm GOARM=7 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/arm/$(CLIENT_BINARY)-armv7 ./cmd/client
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/arm/$(SERVER_BINARY)-arm64 ./cmd/server
	GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/arm/$(CLIENT_BINARY)-arm64 ./cmd/client
	@echo "ARM builds complete"

# Run tests
test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "Tests complete"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out
	@echo "Clean complete"

# Install to system
install: build
	@echo "Installing to system..."
	@mkdir -p /usr/local/bin
	cp $(BUILD_DIR)/$(SERVER_BINARY) /usr/local/bin/
	cp $(BUILD_DIR)/$(CLIENT_BINARY) /usr/local/bin/
	@echo "Installation complete"

# Development mode (with race detector)
dev:
	@echo "Building in development mode..."
	$(GO) build -race -o $(BUILD_DIR)/$(SERVER_BINARY)-dev ./cmd/server
	$(GO) build -race -o $(BUILD_DIR)/$(CLIENT_BINARY)-dev ./cmd/client
	@echo "Development build complete"

# Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Formatting complete"

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run ./...
	@echo "Linting complete"

# Generate documentation
docs:
	@echo "Generating documentation..."
	$(GO) doc -all ./pkg/... > docs/API.md
	@echo "Documentation generated"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem ./...
	@echo "Benchmarks complete"

# Update dependencies
deps:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "Dependencies updated"

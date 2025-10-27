#!/bin/bash
# MultiWANBond - Multi-platform Build Script
# Builds releases for Linux, Windows, and macOS

set -e

VERSION=${VERSION:-"1.0.0"}
BUILD_DIR="build"
RELEASE_DIR="releases"

echo "========================================"
echo "MultiWANBond Build Script"
echo "Version: $VERSION"
echo "========================================"
echo ""

# Clean previous builds
echo "Cleaning previous builds..."
rm -rf "$BUILD_DIR" "$RELEASE_DIR"
mkdir -p "$BUILD_DIR" "$RELEASE_DIR"

# Build flags
LDFLAGS="-s -w -X main.version=$VERSION -X main.buildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')"

# Build function
build_binary() {
    local OS=$1
    local ARCH=$2
    local OUTPUT=$3

    echo "Building for $OS/$ARCH..."

    GOOS=$OS GOARCH=$ARCH CGO_ENABLED=0 go build \
        -ldflags "$LDFLAGS" \
        -o "$BUILD_DIR/$OUTPUT" \
        ./cmd/server/main.go

    if [ $? -eq 0 ]; then
        echo "✓ Successfully built $OUTPUT"
    else
        echo "✗ Failed to build $OUTPUT"
        exit 1
    fi
}

# Linux builds
echo ""
echo "Building Linux binaries..."
echo "----------------------------"
build_binary "linux" "amd64" "multiwanbond-linux-amd64"
build_binary "linux" "arm64" "multiwanbond-linux-arm64"
build_binary "linux" "arm" "multiwanbond-linux-arm"

# Windows builds
echo ""
echo "Building Windows binaries..."
echo "----------------------------"
build_binary "windows" "amd64" "multiwanbond-windows-amd64.exe"
build_binary "windows" "arm64" "multiwanbond-windows-arm64.exe"

# macOS builds
echo ""
echo "Building macOS binaries..."
echo "----------------------------"
build_binary "darwin" "amd64" "multiwanbond-darwin-amd64"
build_binary "darwin" "arm64" "multiwanbond-darwin-arm64"

# Create release packages
echo ""
echo "Creating release packages..."
echo "----------------------------"

# Linux amd64
tar -czf "$RELEASE_DIR/multiwanbond-$VERSION-linux-amd64.tar.gz" \
    -C "$BUILD_DIR" multiwanbond-linux-amd64
echo "✓ Created multiwanbond-$VERSION-linux-amd64.tar.gz"

# Linux arm64
tar -czf "$RELEASE_DIR/multiwanbond-$VERSION-linux-arm64.tar.gz" \
    -C "$BUILD_DIR" multiwanbond-linux-arm64
echo "✓ Created multiwanbond-$VERSION-linux-arm64.tar.gz"

# Linux arm
tar -czf "$RELEASE_DIR/multiwanbond-$VERSION-linux-arm.tar.gz" \
    -C "$BUILD_DIR" multiwanbond-linux-arm
echo "✓ Created multiwanbond-$VERSION-linux-arm.tar.gz"

# Windows amd64
cd "$BUILD_DIR"
zip -q "../$RELEASE_DIR/multiwanbond-$VERSION-windows-amd64.zip" multiwanbond-windows-amd64.exe
cd ..
echo "✓ Created multiwanbond-$VERSION-windows-amd64.zip"

# Windows arm64
cd "$BUILD_DIR"
zip -q "../$RELEASE_DIR/multiwanbond-$VERSION-windows-arm64.zip" multiwanbond-windows-arm64.exe
cd ..
echo "✓ Created multiwanbond-$VERSION-windows-arm64.zip"

# macOS amd64
tar -czf "$RELEASE_DIR/multiwanbond-$VERSION-darwin-amd64.tar.gz" \
    -C "$BUILD_DIR" multiwanbond-darwin-amd64
echo "✓ Created multiwanbond-$VERSION-darwin-amd64.tar.gz"

# macOS arm64 (Apple Silicon)
tar -czf "$RELEASE_DIR/multiwanbond-$VERSION-darwin-arm64.tar.gz" \
    -C "$BUILD_DIR" multiwanbond-darwin-arm64
echo "✓ Created multiwanbond-$VERSION-darwin-arm64.tar.gz"

# Generate checksums
echo ""
echo "Generating checksums..."
echo "----------------------------"
cd "$RELEASE_DIR"
sha256sum *.tar.gz *.zip > SHA256SUMS
cd ..
echo "✓ Created SHA256SUMS"

# Summary
echo ""
echo "========================================"
echo "Build Summary"
echo "========================================"
echo "Version: $VERSION"
echo "Output directory: $RELEASE_DIR/"
echo ""
echo "Release packages:"
ls -lh "$RELEASE_DIR/"
echo ""
echo "✓ Build completed successfully!"
echo "========================================"

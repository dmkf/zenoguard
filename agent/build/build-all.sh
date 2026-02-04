#!/bin/bash

# Comprehensive build script for ZenoGuard Agent
# Supports multiple platforms and architectures

set -e

echo "=========================================="
echo "ZenoGuard Agent Multi-Platform Build"
echo "=========================================="

cd "$(dirname "$0")/.."

# Set build variables
VERSION=${VERSION:-"1.0.0"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS="-X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE -s -w"

# Create bin directory
mkdir -p bin

# Function to build for specific platform
build_platform() {
    local os=$1
    local arch=$2
    local ext=$3
    local name="zenoguard-agent-${os}-${arch}${ext}"

    echo "Building for $os/$arch..."

    CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build \
        -ldflags "$LDFLAGS" \
        -o "bin/$name" \
        ./cmd/agent

    if [ $? -eq 0 ]; then
        echo "✓ Successfully built: bin/$name"

        # Get file size
        if [[ "$OSTYPE" == "darwin"* ]]; then
            size=$(stat -f%z "bin/$name" 2>/dev/null || echo "unknown")
        else
            size=$(stat -c%s "bin/$name" 2>/dev/null || echo "unknown")
        fi
        echo "  Size: $(numfmt --to=iec-i --suffix=B $size 2>/dev/null || echo ${size} bytes)"
    else
        echo "✗ Failed to build for $os/$arch"
        return 1
    fi
}

# Build matrix
echo ""
echo "Building for all platforms..."
echo ""

# Linux builds
build_platform "linux" "amd64" ""
build_platform "linux" "arm64" ""
build_platform "linux" "386" ""
build_platform "linux" "arm" "hf" GOARM=6
build_platform "linux" "arm" "hf" GOARM=7

# macOS builds
build_platform "darwin" "amd64" ""
build_platform "darwin" "arm64" ""

# Windows builds
build_platform "windows" "amd64" ".exe"
build_platform "windows" "386" ".exe"

# FreeBSD builds
build_platform "freebsd" "amd64" ""

echo ""
echo "=========================================="
echo "Build Summary"
echo "=========================================="
echo "Version: $VERSION"
echo "Commit: $COMMIT"
echo "Date: $DATE"
echo ""

# List all built binaries
echo "Built binaries:"
ls -lh bin/ | grep zenoguard-agent | awk '{print "  " $9 " (" $5 ")"}'

echo ""
echo "✓ All builds completed successfully!"
echo ""
echo "Usage:"
echo "  Linux AMD64:   bin/zenoguard-agent-linux-amd64"
echo "  Linux ARM64:   bin/zenoguard-agent-linux-arm64"
echo "  macOS AMD64:   bin/zenoguard-agent-darwin-amd64"
echo "  macOS ARM64:   bin/zenoguard-agent-darwin-arm64"
echo "  Windows AMD64: bin/zenoguard-agent-windows-amd64.exe"
echo ""

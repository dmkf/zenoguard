#!/bin/bash

# Quick build script - builds for common platforms only
# Usage: ./build-quick.sh [platform]
#   platform: linux (default), darwin, windows, all

set -e

cd "$(dirname "$0")"

# Set build variables
VERSION=${VERSION:-"1.0.0"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE -s -w"

PLATFORM=${1:-linux}

build() {
    local os=$1
    local arch=$2
    local ext=$3
    local name="zenoguard-agent-${os}-${arch}${ext}"

    echo "Building $name..."

    CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build \
        -ldflags "$LDFLAGS" \
        -o "bin/$name" \
        ./cmd/agent

    echo "✓ bin/$name"
}

mkdir -p bin

case $PLATFORM in
    linux)
        echo "Building for Linux AMD64..."
        build linux amd64 ""
        ;;
    darwin)
        echo "Building for macOS..."
        build darwin amd64 ""
        build darwin arm64 ""
        ;;
    windows)
        echo "Building for Windows..."
        build windows amd64 ".exe"
        ;;
    arm)
        echo "Building for ARM64..."
        build linux arm64 ""
        build darwin arm64 ""
        ;;
    all)
        echo "Building for all platforms..."
        build linux amd64 ""
        build linux arm64 ""
        build darwin amd64 ""
        build darwin arm64 ""
        build windows amd64 ".exe"
        ;;
    *)
        echo "Unknown platform: $PLATFORM"
        echo "Usage: $0 [linux|darwin|windows|arm|all]"
        exit 1
        ;;
esac

echo ""
echo "✓ Build complete!"
ls -lh bin/ | tail -n +2

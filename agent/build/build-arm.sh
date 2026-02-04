#!/bin/bash

# Build script for ARM64 Linux

set -e

echo "Building ZenoGuard Agent for ARM64..."

cd "$(dirname "$0")/.."

# Set build variables
VERSION=${VERSION:-"1.0.0"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS="-X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE"

# Build for Linux ARM64
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
    -ldflags "$LDFLAGS" \
    -o bin/zenoguard-agent-arm64 \
    ./cmd/agent

echo "Build complete: bin/zenoguard-agent-arm64"
echo "Version: $VERSION"
echo "Commit: $COMMIT"
echo "Date: $DATE"

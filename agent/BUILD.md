# ZenoGuard Agent Build Guide

## Prerequisites

- Go 1.21 or higher
- Basic Unix shell (bash, zsh, etc.)

## Build Scripts

### 1. Quick Build (Recommended for development)

Build for a specific platform quickly:

```bash
# Build for Linux AMD64 (default)
./build-quick.sh

# Build for macOS
./build-quick.sh darwin

# Build for Windows
./build-quick.sh windows

# Build for ARM64
./build-quick.sh arm

# Build for all platforms
./build-quick.sh all
```

### 2. Platform-Specific Builds

Build for a specific platform:

```bash
# Linux AMD64
./build/build.sh

# Linux ARM64
./build/build-arm.sh
```

### 3. Complete Build (All platforms)

Build for all supported platforms and architectures:

```bash
./build/build-all.sh
```

This will create binaries for:
- Linux (amd64, arm64, 386, armhf)
- macOS (amd64, arm64)
- Windows (amd64, 386)
- FreeBSD (amd64)

## Output

All binaries are placed in the `bin/` directory with the following naming convention:

```
zenoguard-agent-{os}-{arch}{extension}
```

Examples:
- `zenoguard-agent-linux-amd64` - Linux 64-bit Intel/AMD
- `zenoguard-agent-linux-arm64` - Linux 64-bit ARM
- `zenoguard-agent-darwin-amd64` - macOS Intel
- `zenoguard-agent-darwin-arm64` - macOS Apple Silicon
- `zenoguard-agent-windows-amd64.exe` - Windows 64-bit

## Build Variables

You can customize the build with environment variables:

```bash
# Set version
VERSION=2.0.0 ./build-quick.sh

# Set version and build info
VERSION=2.0.0 BUILD_INFO="custom build" ./build-quick.sh
```

## Cross-Compilation

These scripts use Go's cross-compilation capabilities. No additional toolchains are required for pure Go builds.

### CGO Builds

If you need CGO enabled (for C dependencies), set `CGO_ENABLED=1` and ensure you have the appropriate cross-compilation toolchains installed.

```bash
# Example: Linux AMD64 with CGO (requires gcc-x86-64-linux-gnu)
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o bin/agent ./cmd/agent
```

## Manual Build

Build manually with full control:

```bash
# Basic build
go build -o bin/zenoguard-agent ./cmd/agent

# Build with version info
go build \
  -ldflags "-X main.version=1.0.0 -X main.commit=abc123 -s -w" \
  -o bin/zenoguard-agent \
  ./cmd/agent

# Cross-compile for Linux AMD64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags "-s -w" \
  -o bin/zenoguard-agent-linux-amd64 \
  ./cmd/agent
```

## Build Flags

Common build flags used:

- `-s` - Omit symbol table and debug information
- `-w` - Omit DWARF symbol table
- `-X main.version=<ver>` - Set version variable
- `-X main.commit=<hash>` - Set commit hash
- `-X main.date=<timestamp>` - Set build date

## Testing the Build

After building, you can test the agent:

```bash
# Show version
./bin/zenoguard-agent-linux-amd64 --version

# Show help
./bin/zenoguard-agent-linux-amd64 --help

# Run with test configuration
./bin/zenoguard-agent-linux-amd64 --config test-config.yaml
```

## Deployment

Copy the appropriate binary to your target server:

```bash
# Example: Deploy to Linux server
scp bin/zenoguard-agent-linux-amd64 user@server:/usr/local/bin/zenoguard-agent

# Make executable on server
ssh user@server "chmod +x /usr/local/bin/zenoguard-agent"
```

## Troubleshooting

### Permission Denied

If you get a permission error running build scripts:

```bash
chmod +x build-quick.sh build/build*.sh
```

### Go Version Too Old

If you get errors about Go version:

```bash
# Check Go version
go version

# Update Go (macOS)
brew upgrade go

# Update Go (Linux)
# Visit https://golang.org/dl/
```

### Cross-Compilation Issues

If cross-compilation fails:

1. Ensure you're using Go 1.21+
2. Check that CGO is disabled for pure Go builds: `CGO_ENABLED=0`
3. Verify target platform: `GOOS` and `GOARCH` values

## Architecture Reference

| GOOS    | GOARCH | Description                    |
|---------|--------|--------------------------------|
| linux   | amd64  | Linux 64-bit (Intel/AMD)       |
| linux   | arm64  | Linux 64-bit ARM (ARMv8+)      |
| linux   | 386    | Linux 32-bit                    |
| linux   | arm    | Linux 32-bit ARM (ARMv6/v7)    |
| darwin  | amd64  | macOS Intel                     |
| darwin  | arm64  | macOS Apple Silicon (M1/M2/M3) |
| windows | amd64  | Windows 64-bit                  |
| windows | 386    | Windows 32-bit                  |
| freebsd | amd64  | FreeBSD 64-bit                  |

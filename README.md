# gopose - Docker Compose Port Conflict Auto-Resolution Tool

<div align="center">
  <img src="logo.png" alt="gopose logo" width="200"/>
  
  [![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
  [![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
  [![Go Report Card](https://goreportcard.com/badge/github.com/harakeishi/gopose?style=for-the-badge)](https://goreportcard.com/report/github.com/harakeishi/gopose)
</div>

## Overview

**gopose** (Go Port Override Solution Engine) is a tool that automatically detects and resolves Docker Compose port binding and network conflicts.

It generates a `docker-compose.override.yml` without modifying the original `docker-compose.yml`, and automatically deletes the `override.yml` after resolving port and network conflicts.

### 🎯 Key Features

- ✅ **Non-destructive**: Does not modify the original `docker-compose.yml` file
- ✅ **Auto-detection**: Automatically detects conflicts with system ports in use
- ✅ **Auto-resolution**: Automatically assigns available ports
- ✅ **Network conflict avoidance**: Automatically detects and avoids Docker network subnet conflicts
- ✅ **Auto-cleanup**: Automatically deletes `override.yml` on process termination
- ✅ **SOLID principles**: Designed for maintainability and extensibility
- ✅ **Structured logging**: Detailed log output and debugging capabilities
- ✅ **Cross-platform**: Supports Linux, macOS, and Windows
- ✅ **Parallel processing**: Performs port scanning in parallel

## Installation

### Binary Releases

Download the appropriate binary from [GitHub Releases](https://github.com/harakeishi/gopose/releases):

```bash
# Linux (amd64)
curl -L https://github.com/harakeishi/gopose/releases/latest/download/gopose_linux_amd64.tar.gz | tar xz
sudo mv gopose /usr/local/bin/

# macOS (arm64)
curl -L https://github.com/harakeishi/gopose/releases/latest/download/gopose_darwin_arm64.tar.gz | tar xz
sudo mv gopose /usr/local/bin/

# Windows (amd64)
curl -L https://github.com/harakeishi/gopose/releases/latest/download/gopose_windows_amd64.zip -o gopose.zip
unzip gopose.zip
```

### Go Install

```bash
go install github.com/harakeishi/gopose@latest
```

### Build from Source

```bash
git clone https://github.com/harakeishi/gopose.git
cd gopose
make build
sudo make install
```

## Usage

### Basic Usage

```bash
# Detect and resolve port/network conflicts and prepare Docker Compose
gopose up

```

### Advanced Usage

#### File Specification and Port Range Settings

```bash
# Specify a custom file
gopose up -f custom-compose.yml

# Specify port range
gopose up --port-range 9000-9999

# Specify multiple port ranges
gopose up --port-range 8000-8999,9000-9999
```

#### Exclusion Settings

```bash
# Exclude specific services
gopose up --exclude-services redis,postgres

# Exclude privileged ports
gopose up --exclude-privileged

# Exclude reserved ports
gopose up --exclude-ports 8080,8443,9000
```

#### Output and Logging Settings

```bash
# Dry run (no actual changes)
gopose up --dry-run

# Verbose logging
gopose up --verbose

# Display with detailed information
gopose up --detail # Shows timestamps and fields

# Check status in JSON format
gopose status --output json

# Set log level
gopose up --log-level debug
```

### Configuration File

You can place a configuration file (`.gopose.yaml`) in your home or project directory:

```yaml
port:
  range:
    start: 8000
    end: 9999
  reserved: [8080, 8443, 9000, 9090]
  exclude_privileged: true

file:
  compose_file: "docker-compose.yml"
  override_file: "docker-compose.override.yml"
  backup_enabled: true

watcher:
  interval: "5s"
  cleanup_delay: "30s"

log:
  level: "info"
  format: "text"
  file: "~/.gopose/logs/gopose.log"

resolver:
  strategy: "minimal_change"  # minimal_change, sequential, random
  preserve_dependencies: true
  port_proximity: true
```

### Output Example

```
$ gopose up
Starting port conflict resolution
Starting Docker Compose file detection
Docker Compose file found
Docker Compose file detection completed
Auto-detected Docker Compose file
Starting Docker Compose file parsing
Docker Compose version not specified
Docker Compose file parsing completed
Starting port conflict detection
Starting port scan using netstat
Port scan completed
System port conflict detected
Port conflict detection completed
Port conflict detection completed
Starting port conflict resolution
Starting port scan using netstat
Port scan completed
In-range port filtering completed
Port allocation successful
Starting solution optimization
Solution optimization completed
Port conflict resolution completed
Port resolved
Starting override generation
Port mapping updated
Override generation completed
Starting override validation
Override version not specified, but allowed as it's deprecated in latest Docker Compose versions
Override validation completed
Starting override file write
Override file write completed
Override.yml file generated
Existing Docker networks detected
Docker Compose network configuration detected
Network subnet conflict detected
Network subnet conflict resolved
Stopping existing containers before starting Docker Compose
[+] Running 2/2
 ✔ Container gopose-web-1  Removed                                                                                         0.0s
 ✔ Network gopose_default  Removed                                                                                         0.2s
Starting Docker Compose
Executing Docker Compose
[+] Running 2/2
 ✔ Network gopose_default  Created                                                                                         0.0s
 ✔ Container gopose-web-1  Created                                                                                         0.0s
Attaching to web-1
```

#### With --detail flag

```
$ gopose up --detail
time=2025-06-10T23:31:03.179+09:00 level=INFO msg="Starting port conflict resolution" component=gopose timestamp=2025-06-10T23:31:03.178+09:00 dry_run=false compose_file=docker-compose.yml output_file="" strategy=auto port_range=8000-9999 skip_compose_up=false
time=2025-06-10T23:31:03.179+09:00 level=INFO msg="Docker Compose file detection completed" component=gopose timestamp=2025-06-10T23:31:03.179+09:00 directory=/Users/keishi.hara/src/github.com/harakeishi/gopose found_count=1
time=2025-06-10T23:31:03.179+09:00 level=INFO msg="Auto-detected Docker Compose file" component=gopose timestamp=2025-06-10T23:31:03.179+09:00 file=/Users/keishi.hara/src/github.com/harakeishi/gopose/compose.yml
time=2025-06-10T23:31:03.180+09:00 level=WARN msg="Docker Compose version not specified" component=gopose timestamp=2025-06-10T23:31:03.180+09:00
time=2025-06-10T23:31:03.180+09:00 level=INFO msg="Docker Compose file parsing completed" component=gopose timestamp=2025-06-10T23:31:03.180+09:00 file=/Users/keishi.hara/src/github.com/harakeishi/gopose/compose.yml services_count=1
time=2025-06-10T23:31:03.191+09:00 level=INFO msg="Port scan completed" component=gopose timestamp=2025-06-10T23:31:03.191+09:00 found_ports_count=18
time=2025-06-10T23:31:03.191+09:00 level=WARN msg="System port conflict detected" component=gopose timestamp=2025-06-10T23:31:03.191+09:00 port=3000 service=web
time=2025-06-10T23:31:03.191+09:00 level=INFO msg="Port conflict detection completed" component=gopose timestamp=2025-06-10T23:31:03.191+09:00 conflicts_count=1
time=2025-06-10T23:31:03.191+09:00 level=INFO msg="Port conflict detection completed" component=gopose timestamp=2025-06-10T23:31:03.191+09:00 conflicts_count=1
time=2025-06-10T23:31:03.202+09:00 level=INFO msg="Port scan completed" component=gopose timestamp=2025-06-10T23:31:03.202+09:00 found_ports_count=18
time=2025-06-10T23:31:03.202+09:00 level=INFO msg="Solution optimization completed" component=gopose timestamp=2025-06-10T23:31:03.202+09:00 original_count=1 optimized_count=1
time=2025-06-10T23:31:03.202+09:00 level=INFO msg="Port conflict resolution completed" component=gopose timestamp=2025-06-10T23:31:03.202+09:00 resolved_conflicts=1
time=2025-06-10T23:31:03.202+09:00 level=INFO msg="Port resolved" component=gopose timestamp=2025-06-10T23:31:03.202+09:00 service=web from=3000 to=8001 reason="Automatic port change from 3000 to 8001"
time=2025-06-10T23:31:03.205+09:00 level=INFO msg="Existing Docker networks detected" component=gopose timestamp=2025-06-10T23:31:03.205+09:00 network_count=3
time=2025-06-10T23:31:03.205+09:00 level=INFO msg="Docker Compose network configuration detected" component=gopose timestamp=2025-06-10T23:31:03.205+09:00 network_count=1
time=2025-06-10T23:31:03.205+09:00 level=WARN msg="Network subnet conflict detected" component=gopose timestamp=2025-06-10T23:31:03.205+09:00 network=default conflicting_subnet="172.20.0.0/24"
time=2025-06-10T23:31:03.205+09:00 level=INFO msg="Network subnet conflict resolved" component=gopose timestamp=2025-06-10T23:31:03.205+09:00 network=default original_subnet="172.20.0.0/24" new_subnet="10.20.0.0/24"
time=2025-06-10T23:31:03.202+09:00 level=INFO msg="Override generation completed" component=gopose timestamp=2025-06-10T23:31:03.202+09:00 services_count=1
time=2025-06-10T23:31:03.202+09:00 level=INFO msg="Override validation completed" component=gopose timestamp=2025-06-10T23:31:03.202+09:00
time=2025-06-10T23:31:03.202+09:00 level=INFO msg="Override file write completed" component=gopose timestamp=2025-06-10T23:31:03.202+09:00 output_path=docker-compose.override.yml file_size=607
time=2025-06-10T23:31:03.202+09:00 level=INFO msg="Override.yml file generated" component=gopose timestamp=2025-06-10T23:31:03.202+09:00 output_file=docker-compose.override.yml
time=2025-06-10T23:31:03.202+09:00 level=INFO msg="Stopping existing containers before starting Docker Compose" component=gopose timestamp=2025-06-10T23:31:03.202+09:00
[+] Running 2/2
 ✔ Container gopose-web-1  Removed                                                                                         0.2s
 ✔ Network gopose_default  Removed                                                                                         0.2s
time=2025-06-10T23:31:03.779+09:00 level=INFO msg="Starting Docker Compose" component=gopose timestamp=2025-06-10T23:31:03.779+09:00
time=2025-06-10T23:31:03.780+09:00 level=INFO msg="Executing Docker Compose" component=gopose timestamp=2025-06-10T23:31:03.780+09:00 command="docker compose -f /Users/keishi.hara/src/github.com/harakeishi/gopose/compose.yml -f docker-compose.override.yml up --force-recreate --remove-orphans"
[+] Running 2/2
 ✔ Network gopose_default  Created                                                                                         0.0s
 ✔ Container gopose-web-1  Created                                                                                         0.0s
Attaching to web-1
```

## Network Conflict Avoidance

gopose automatically detects subnet conflicts with existing Docker networks and assigns safe alternative subnets.

### Feature Overview

- **Auto-detection**: Automatically detects existing Docker network subnets
- **Conflict avoidance**: Automatically generates safe alternative subnets when Docker Compose-defined network subnets conflict with existing networks
- **Priority order**: Selects safe subnets in order: `10.x.x.x/24` > `192.168.x.x/24` > `172.x.x.x/24`
- **Conflict avoidance**: Avoids Docker default ranges (`172.17-29.x.x`) and common home router ranges

### Subnet Assignment Strategy

1. **10.x.x.x/24 range**: Safest (starting from `10.20.0.0/24`)
2. **192.168.x.x/24 range**: Avoids common home router ranges (starting from `192.168.100.0/24`)
3. **172.x.x.x/24 range**: Last resort (starting from `172.30.0.0/24`, avoiding Docker default ranges)

### Example

```yaml
# Original docker-compose.yml
networks:
  app-network:
    ipam:
      config:
        - subnet: 172.20.0.0/24  # Conflicts with other Docker networks

# Generated docker-compose.override.yml
networks:
  app-network:
    ipam:
      config:
        - subnet: 10.20.0.0/24  # Automatically changed to safe subnet
```

## Directory Structure

```
gopose/
├── cmd/                 # CLI commands
│   ├── root.go         # Cobra root command + DI container
│   ├── up.go           # up subcommand
│   ├── clean.go        # clean subcommand
│   ├── status.go       # status subcommand
│   └── wire.go         # Dependency injection configuration (Wire)
├── internal/           # Internal implementation
│   ├── app/           # Application layer
│   ├── scanner/       # Port scanning
│   ├── parser/        # Docker Compose parsing
│   ├── resolver/      # Conflict resolution
│   ├── generator/     # Override generation
│   ├── file/          # File operations
│   ├── watcher/       # Process monitoring
│   ├── cleanup/       # Auto-cleanup
│   ├── config/        # Configuration management
│   ├── logger/        # Logging functionality
│   └── errors/        # Error handling
├── pkg/               # Public packages
│   ├── types/         # Type definitions
│   └── testutil/      # Test utilities
├── test/              # Tests
│   ├── unit/          # Unit tests
│   ├── integration/   # Integration tests
│   └── e2e/           # E2E tests
├── docs/              # Documentation
├── scripts/           # Scripts
└── deployments/       # Deployment configurations
```

## Development

### Development Environment Setup

```bash
# Clone repository
git clone https://github.com/harakeishi/gopose.git
cd gopose

# Install dependencies
make deps

# Development build
make dev

# Run tests
make test

# Code quality check
make check
```

### Make Tasks

```bash
# Build
make build              # Regular build
make build-all          # Build for all platforms
make dev                # Development build

# Testing
make test               # Run all tests
make test-unit          # Unit tests
make test-integration   # Integration tests
make test-e2e           # E2E tests
make test-coverage      # Generate coverage

# Code Quality
make fmt                # Code formatting
make lint               # Run linter
make vet                # Run go vet
make check              # Run all checks

# Development
make run                # Execute
make clean              # Clean up
make deps               # Install dependencies

# Release
make release            # Release build
make docker-build       # Docker image build
```

### Testing

```bash
# Run all tests
go test ./...

# Test with coverage
go test -race -coverprofile=coverage.out ./...

# Benchmark tests
go test -bench=. ./...

# Run specific tests only
go test -run TestPortScanner ./internal/scanner/
```

## License

This project is published under the [MIT License](LICENSE).
---

<div align="center">
  <p>Developed by <a href="https://github.com/harakeishi">harakeishi</a></p>
  <p>
    <a href="https://github.com/harakeishi/gopose/issues">🐛 Bug Reports</a> •
    <a href="https://github.com/harakeishi/gopose/discussions">💬 Discussions</a> •
    <a href="https://github.com/harakeishi/gopose/wiki">📖 Wiki</a>
  </p>
</div>
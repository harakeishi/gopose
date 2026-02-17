# gopose - Docker Compose Port Conflict Auto-Resolution Tool

<div align="center">
  <img src="logo.png" alt="gopose logo" width="200"/>
  
  [![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
  [![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
  [![Go Report Card](https://goreportcard.com/badge/github.com/harakeishi/gopose?style=for-the-badge)](https://goreportcard.com/report/github.com/harakeishi/gopose)
</div>

## Overview

**gopose** (Go Port Override Solution Engine) is a tool that automatically detects and resolves Docker Compose port binding and network conflicts.

It generates a `compose.override.yml` without modifying the original `compose.yml`, and automatically deletes the `override.yml` after resolving port and network conflicts.

### 🎯 Key Features

- ✅ **Non-destructive**: Does not modify the original `compose.yml` file
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

# Check version
gopose version
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

#### Reserved Ports Configuration

```bash
# Reserve specific ports (will never be assigned even if unused)
# Use configuration file for persistent settings
```

You can reserve specific ports in `.gopose.yaml` to prevent them from being assigned:

```yaml
port:
  reserved: [8080, 8443, 9000, 9090]  # These ports will never be assigned
```

**Important**: Reserved ports are guaranteed to be skipped during port allocation, regardless of whether they are currently in use or not. This is useful for:
- Preventing conflicts with services you plan to start later
- Reserving ports for manual debugging or testing
- Avoiding specific ports that may be used by other tools or services

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
  # Reserved ports: These ports will NEVER be assigned, even if they are not in use
  # Useful for preventing conflicts with services you plan to start later
  reserved: [8080, 8443, 9000, 9090]
  exclude_privileged: true

file:
  compose_file: "compose.yml"
  override_file: "compose.override.yml"
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

#### Configuration Priority

Configuration settings are applied in the following order (later values override earlier ones):

1. Default configuration (built-in)
2. Configuration file (`.gopose.yaml`)
3. CLI options (e.g., `--port-range`)

**Note**: Reserved ports from the configuration file are always preserved, even when using CLI options to override the port range.

### Output Example

```
$ gopose up
ポート衝突解決を開始
Docker Composeファイル検出開始
Docker Composeファイル発見
Docker Composeファイル検出完了
Docker Composeファイルを自動検出
Docker Composeファイル解析開始
Docker Composeバージョンが指定されていません
Docker Composeファイル解析完了
ポート衝突検出開始
netstatを使用してポートスキャンを開始
ポートスキャン完了
システムポート衝突検出
ポート衝突検出完了
ポート衝突検出完了
ポート衝突解決開始
netstatを使用してポートスキャンを開始
ポートスキャン完了
範囲内ポートフィルタリング完了
ポート割り当て成功
解決案最適化開始
解決案最適化完了
ポート衝突解決完了
ポート解決
Override生成開始
ポートマッピング更新
Override生成完了
Override検証開始
Overrideのバージョンが指定されていませんが、Docker Composeの最新バージョンでは非推奨のため許可します
Override検証完了
Overrideファイル書き込み開始
Overrideファイル書き込み完了
Override.ymlファイルが生成されました
既存Dockerネットワークを検出しました
Docker Composeネットワーク設定を検出
ネットワークサブネット競合を検出
ネットワークサブネット競合を解決
既存のコンテナを停止してからDocker Composeを起動
[+] Running 2/2
 ✔ Container gopose-web-1  Removed                                                                                         0.0s
 ✔ Network gopose_default  Removed                                                                                         0.2s
Docker Composeを起動
Docker Composeを実行
[+] Running 2/2
 ✔ Network gopose_default  Created                                                                                         0.0s
 ✔ Container gopose-web-1  Created                                                                                         0.0s
Attaching to web-1
```

#### With --detail flag

```
$ gopose up --detail
time=2025-06-10T23:31:03.179+09:00 level=INFO msg=ポート衝突解決を開始 component=gopose timestamp=2025-06-10T23:31:03.178+09:00 dry_run=false compose_file=docker-compose.yml output_file="" strategy=auto port_range=8000-9999 skip_compose_up=false
time=2025-06-10T23:31:03.179+09:00 level=INFO msg="Docker Composeファイル検出完了" component=gopose timestamp=2025-06-10T23:31:03.179+09:00 directory=/Users/keishi.hara/src/github.com/harakeishi/gopose found_count=1
time=2025-06-10T23:31:03.179+09:00 level=INFO msg="Docker Composeファイルを自動検出" component=gopose timestamp=2025-06-10T23:31:03.179+09:00 file=/Users/keishi.hara/src/github.com/harakeishi/gopose/compose.yml
time=2025-06-10T23:31:03.180+09:00 level=WARN msg="Docker Composeバージョンが指定されていません" component=gopose timestamp=2025-06-10T23:31:03.180+09:00
time=2025-06-10T23:31:03.180+09:00 level=INFO msg="Docker Composeファイル解析完了" component=gopose timestamp=2025-06-10T23:31:03.180+09:00 file=/Users/keishi.hara/src/github.com/harakeishi/gopose/compose.yml services_count=1
time=2025-06-10T23:31:03.191+09:00 level=INFO msg=ポートスキャン完了 component=gopose timestamp=2025-06-10T23:31:03.191+09:00 found_ports_count=18
time=2025-06-10T23:31:03.191+09:00 level=WARN msg=システムポート衝突検出 component=gopose timestamp=2025-06-10T23:31:03.191+09:00 port=3000 service=web
time=2025-06-10T23:31:03.191+09:00 level=INFO msg=ポート衝突検出完了 component=gopose timestamp=2025-06-10T23:31:03.191+09:00 conflicts_count=1
time=2025-06-10T23:31:03.191+09:00 level=INFO msg=ポート衝突検出完了 component=gopose timestamp=2025-06-10T23:31:03.191+09:00 conflicts_count=1
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=ポートスキャン完了 component=gopose timestamp=2025-06-10T23:31:03.202+09:00 found_ports_count=18
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=解決案最適化完了 component=gopose timestamp=2025-06-10T23:31:03.202+09:00 original_count=1 optimized_count=1
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=ポート衝突解決完了 component=gopose timestamp=2025-06-10T23:31:03.202+09:00 resolved_conflicts=1
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=ポート解決 component=gopose timestamp=2025-06-10T23:31:03.202+09:00 service=web from=3000 to=8001 reason="ポート 3000 から 8001 への自動変更"
time=2025-06-10T23:31:03.205+09:00 level=INFO msg="既存Dockerネットワークを検出しました" component=gopose timestamp=2025-06-10T23:31:03.205+09:00 network_count=3
time=2025-06-10T23:31:03.205+09:00 level=INFO msg="Docker Composeネットワーク設定を検出" component=gopose timestamp=2025-06-10T23:31:03.205+09:00 network_count=1
time=2025-06-10T23:31:03.205+09:00 level=WARN msg="ネットワークサブネット競合を検出" component=gopose timestamp=2025-06-10T23:31:03.205+09:00 network=default conflicting_subnet="172.20.0.0/24"
time=2025-06-10T23:31:03.205+09:00 level=INFO msg="ネットワークサブネット競合を解決" component=gopose timestamp=2025-06-10T23:31:03.205+09:00 network=default original_subnet="172.20.0.0/24" new_subnet="10.20.0.0/24"
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=Override生成完了 component=gopose timestamp=2025-06-10T23:31:03.202+09:00 services_count=1
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=Override検証完了 component=gopose timestamp=2025-06-10T23:31:03.202+09:00
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=Overrideファイル書き込み完了 component=gopose timestamp=2025-06-10T23:31:03.202+09:00 output_path=compose.override.yml file_size=607
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=Override.ymlファイルが生成されました component=gopose timestamp=2025-06-10T23:31:03.202+09:00 output_file=compose.override.yml
time=2025-06-10T23:31:03.202+09:00 level=INFO msg="既存のコンテナを停止してからDocker Composeを起動" component=gopose timestamp=2025-06-10T23:31:03.202+09:00
[+] Running 2/2
 ✔ Container gopose-web-1  Removed                                                                                         0.2s
 ✔ Network gopose_default  Removed                                                                                         0.2s
time=2025-06-10T23:31:03.779+09:00 level=INFO msg="Docker Composeを起動" component=gopose timestamp=2025-06-10T23:31:03.779+09:00
time=2025-06-10T23:31:03.780+09:00 level=INFO msg="Docker Composeを実行" component=gopose timestamp=2025-06-10T23:31:03.780+09:00 command="docker compose -f /Users/keishi.hara/src/github.com/harakeishi/gopose/compose.yml -f compose.override.yml up --force-recreate --remove-orphans"
[+] Running 2/2
 ✔ Network gopose_default  Created                                                                                         0.0s
 ✔ Container gopose-web-1  Created                                                                                         0.0s
Attaching to web-1
```

## Reserved Ports

The `reserved` configuration allows you to specify ports that should never be assigned by gopose, regardless of whether they are currently in use.

### Use Cases

- **Future services**: Reserve ports for services you plan to start later
- **Manual debugging**: Keep specific ports available for manual testing
- **External tools**: Avoid conflicts with ports used by other development tools
- **Consistency**: Ensure certain ports remain free across different environments

### Example Configuration

```yaml
# .gopose.yaml
port:
  range:
    start: 8000
    end: 8100
  reserved:
    - 8080  # Reserved for main application
    - 8443  # Reserved for HTTPS proxy
    - 9000  # Reserved for debugging
    - 9090  # Reserved for monitoring tools
  exclude_privileged: true
```

### Behavior

With the above configuration:
- If gopose needs to assign ports for Docker Compose services with conflicts on ports 80 and 443
- It will skip ports 8080, 8443, 9000, and 9090 (even if they are not in use)
- Available ports like 8000-8079, 8081-8442, 8444-8999, 9001-9089, 9091-8100 will be considered
- For example, ports 8000, 8001, etc. might be assigned instead

### Testing Reserved Ports

You can verify that reserved ports are working correctly:

```bash
# Create a test configuration
cat > .gopose.yaml << EOF
port:
  range:
    start: 8000
    end: 8100
  reserved: [8000, 8001, 8002, 8050]
EOF

# Run gopose - it will skip the reserved ports
gopose up --dry-run

# Check the generated compose.override.yml
# You should see ports starting from 8003 (skipping 8000-8002)
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

# Generated compose.override.yml
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
│   ├── version.go      # version subcommand
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

## Contributors ✨

Thanks goes to these wonderful people ([emoji key](https://allcontributors.org/docs/en/emoji-key)):

<!-- ALL-CONTRIBUTORS-LIST:START - Do not remove or modify this section -->
<!-- prettier-ignore-start -->
<!-- markdownlint-disable -->
<table>
  <tbody>
    <tr>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/t-daisuke"><img src="https://avatars.githubusercontent.com/u/50610194?v=4?s=100" width="100px;" alt="doskoi"/><br /><sub><b>doskoi</b></sub></a><br /><a href="https://github.com/harakeishi/gopose/commits?author=t-daisuke" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="https://github.com/litencatt"><img src="https://avatars.githubusercontent.com/u/17349045?v=4?s=100" width="100px;" alt="Kosuke Nakamura"/><br /><sub><b>Kosuke Nakamura</b></sub></a><br /><a href="https://github.com/harakeishi/gopose/commits?author=litencatt" title="Code">💻</a></td>
      <td align="center" valign="top" width="14.28%"><a href="http://akinoriakatsuka.com"><img src="https://avatars.githubusercontent.com/u/77688294?v=4?s=100" width="100px;" alt="Akinori Takigawa"/><br /><sub><b>Akinori Takigawa</b></sub></a><br /><a href="https://github.com/harakeishi/gopose/commits?author=akinoriakatsuka" title="Code">💻</a></td>
    </tr>
  </tbody>
</table>

<!-- markdownlint-restore -->
<!-- prettier-ignore-end -->

<!-- ALL-CONTRIBUTORS-LIST:END -->
---

<div align="center">
  <p>Developed by <a href="https://github.com/harakeishi">harakeishi</a></p>
  <p>
    <a href="https://github.com/harakeishi/gopose/issues">🐛 Bug Reports</a> •
    <a href="https://github.com/harakeishi/gopose/discussions">💬 Discussions</a> •
    <a href="https://github.com/harakeishi/gopose/wiki">📖 Wiki</a>
  </p>
</div>

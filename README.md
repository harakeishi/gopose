# gopose

<div align="center">
  <img src="logo.png" alt="gopose logo" width="200"/>

  **Automatically detect and resolve Docker Compose port & network conflicts.**

  [![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
  [![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
  [![Go Report Card](https://goreportcard.com/badge/github.com/harakeishi/gopose?style=for-the-badge)](https://goreportcard.com/report/github.com/harakeishi/gopose)
</div>

## The Problem

When running multiple Docker Compose projects, port conflicts are inevitable. You either manually hunt for available ports or wade through cryptic error messages.

**gopose** fixes this automatically. It scans for conflicts and generates a `compose.override.yml` with safe port assignments — without touching your original `compose.yml`.

```
$ gopose up

  SERVICE   ORIGINAL   RESOLVED   REASON
  web       3000       8001       port in use
  api       5432       8002       port in use

  NETWORK   ORIGINAL         RESOLVED
  default   172.20.0.0/24    10.20.0.0/24

Override file generated: compose.override.yml
Run `docker compose up` to start services.
```

## Quick Start

```bash
go install github.com/harakeishi/gopose@latest

cd your-compose-project/
gopose up
```

That's it. gopose detects conflicts, generates the override, and you run `docker compose up` as usual.

## Installation

### Go Install

```bash
go install github.com/harakeishi/gopose@latest
```

### Binary Releases

Download from [GitHub Releases](https://github.com/harakeishi/gopose/releases):

```bash
# macOS (arm64)
curl -L https://github.com/harakeishi/gopose/releases/latest/download/gopose_darwin_arm64.tar.gz | tar xz
sudo mv gopose /usr/local/bin/

# Linux (amd64)
curl -L https://github.com/harakeishi/gopose/releases/latest/download/gopose_linux_amd64.tar.gz | tar xz
sudo mv gopose /usr/local/bin/

# Windows (amd64)
curl -L https://github.com/harakeishi/gopose/releases/latest/download/gopose_windows_amd64.zip -o gopose.zip
unzip gopose.zip
```

### Build from Source

```bash
git clone https://github.com/harakeishi/gopose.git
cd gopose
make build
sudo make install
```

## Usage

### Commands

| Command | Description |
|---------|-------------|
| `gopose up` | Detect conflicts and generate `compose.override.yml` |
| `gopose up --dry-run` | Preview changes without generating files |
| `gopose status` | Show current conflict status |
| `gopose version` | Print version |

### Common Options

| Option | Description | Example |
|--------|-------------|---------|
| `-f` | Specify compose file | `gopose up -f custom-compose.yml` |
| `--port-range` | Set port allocation range | `gopose up --port-range 9000-9999` |
| `--dry-run` | Preview without changes | `gopose up --dry-run` |
| `--verbose` | Enable verbose logging | `gopose up --verbose` |
| `--detail` | Show timestamps and fields | `gopose up --detail` |

## Configuration

Create a `.gopose.yaml` in your project or home directory:

```yaml
port:
  range:
    start: 8000
    end: 9999
  reserved: [8080, 8443]  # Never assigned, even if available

resolver:
  strategy: "minimal_change"
```

See [Configuration Reference](docs/configuration.md) for all options.

## Features

- **Non-destructive** — Never modifies your original `compose.yml`
- **Port conflict resolution** — Detects system port conflicts and assigns available ports
- **Network conflict resolution** — Detects Docker network subnet conflicts and assigns safe alternatives ([details](docs/network-conflict.md))
- **Reserved ports** — Exclude specific ports from allocation ([details](docs/reserved-ports.md))
- **Cross-platform** — Linux, macOS, and Windows

## Documentation

| Document | Description |
|----------|-------------|
| [Configuration Reference](docs/configuration.md) | Full `.gopose.yaml` options |
| [Reserved Ports](docs/reserved-ports.md) | Port reservation behavior and examples |
| [Network Conflict Avoidance](docs/network-conflict.md) | Subnet conflict resolution details |

## Contributing

```bash
git clone https://github.com/harakeishi/gopose.git
cd gopose
make deps    # Install dependencies
make test    # Run tests
make check   # Lint + vet + test
```

## License

[MIT License](LICENSE)

## Contributors

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
    <a href="https://github.com/harakeishi/gopose/issues">Bug Reports</a> ·
    <a href="https://github.com/harakeishi/gopose/discussions">Discussions</a> ·
    <a href="https://github.com/harakeishi/gopose/wiki">Wiki</a>
  </p>
</div>

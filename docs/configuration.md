# Configuration Reference

gopose can be configured via a `.gopose.yaml` file placed in your home directory or project directory.

## Configuration File Location

gopose searches for `.gopose.yaml` in the following order:

1. Current directory (`./.gopose.yaml`)
2. Home directory (`~/.gopose.yaml`)

## Configuration Priority

Settings are applied in the following order (later values override earlier ones):

1. Default configuration (built-in)
2. Configuration file (`.gopose.yaml`)
3. CLI options (e.g., `--port-range`)

> **Note**: Reserved ports from the configuration file are always preserved, even when using CLI options to override the port range.

## Full Configuration Example

```yaml
port:
  range:
    start: 8000
    end: 9999
  # Reserved ports: never assigned, even if not in use
  # See docs/reserved-ports.md for details
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
  level: "info"        # debug, info, warn, error
  format: "text"       # text, json
  file: "~/.gopose/logs/gopose.log"

resolver:
  strategy: "minimal_change"  # minimal_change, sequential, random
  preserve_dependencies: true
  port_proximity: true
```

## Section Details

### port

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `range.start` | int | `8000` | Start of port range for allocation |
| `range.end` | int | `9999` | End of port range for allocation |
| `reserved` | []int | `[]` | Ports that will never be assigned |
| `exclude_privileged` | bool | `true` | Skip privileged ports (< 1024) |

### file

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `compose_file` | string | `"compose.yml"` | Docker Compose file to read |
| `override_file` | string | `"compose.override.yml"` | Override file to generate |
| `backup_enabled` | bool | `true` | Back up existing override file |

### watcher

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `interval` | string | `"5s"` | Process monitoring interval |
| `cleanup_delay` | string | `"30s"` | Delay before cleanup on termination |

### log

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `level` | string | `"info"` | Log level (debug, info, warn, error) |
| `format` | string | `"text"` | Log format (text, json) |
| `file` | string | `""` | Log file path (empty = stderr) |

### resolver

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `strategy` | string | `"minimal_change"` | Resolution strategy |
| `preserve_dependencies` | bool | `true` | Preserve service dependencies |
| `port_proximity` | bool | `true` | Prefer ports close to original |

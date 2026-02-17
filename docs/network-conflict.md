# Network Conflict Avoidance

gopose automatically detects subnet conflicts with existing Docker networks and assigns safe alternative subnets.

## How It Works

1. **Detection**: Scans existing Docker network subnets
2. **Comparison**: Checks if Docker Compose-defined subnets conflict with existing networks
3. **Resolution**: Generates safe alternative subnets in the override file

## Subnet Assignment Strategy

gopose selects safe subnets in the following priority order:

| Priority | Range | Starting From | Notes |
|----------|-------|---------------|-------|
| 1st | `10.x.x.x/24` | `10.20.0.0/24` | Safest option |
| 2nd | `192.168.x.x/24` | `192.168.100.0/24` | Avoids common home router ranges |
| 3rd | `172.x.x.x/24` | `172.30.0.0/24` | Last resort, avoids Docker defaults (172.17-29.x.x) |

## Example

**Original `compose.yml`:**

```yaml
networks:
  app-network:
    ipam:
      config:
        - subnet: 172.20.0.0/24  # Conflicts with other Docker networks
```

**Generated `compose.override.yml`:**

```yaml
networks:
  app-network:
    ipam:
      config:
        - subnet: 10.20.0.0/24  # Automatically changed to safe subnet
```

## Verbose Output

Use `--verbose` to see network conflict detection details:

```bash
gopose up --verbose
```

This will show which networks were detected, which subnets conflict, and what alternatives were assigned.

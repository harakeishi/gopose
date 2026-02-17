# Reserved Ports

The `reserved` configuration allows you to specify ports that should never be assigned by gopose, regardless of whether they are currently in use.

## Use Cases

- **Future services**: Reserve ports for services you plan to start later
- **Manual debugging**: Keep specific ports available for manual testing
- **External tools**: Avoid conflicts with ports used by other development tools
- **Consistency**: Ensure certain ports remain free across different environments

## Configuration

Add reserved ports to your `.gopose.yaml`:

```yaml
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

## Behavior

With the above configuration:

- If gopose needs to assign ports for services with conflicts on ports 80 and 443
- It will **skip** ports 8080, 8443, 9000, and 9090 (even if they are not in use)
- Available ports like 8000-8079, 8081-8099 will be considered
- For example, ports 8000, 8001 might be assigned instead

## Verification

You can verify that reserved ports work correctly:

```bash
# Create a test configuration
cat > .gopose.yaml << EOF
port:
  range:
    start: 8000
    end: 8100
  reserved: [8000, 8001, 8002, 8050]
EOF

# Run gopose in dry-run mode
gopose up --dry-run

# Check the generated compose.override.yml
# Ports should start from 8003 (skipping 8000-8002)
```

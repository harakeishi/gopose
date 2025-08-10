# gopose E2E Tests

## Directory Structure

```
tests/
└── e2e/
    ├── gopose_e2e_test.yml     # Main E2E test suite
    └── fixtures/
        ├── basic.yml           # Basic compose configuration
        └── conflict.yml        # Conflict test configuration
```

## Usage

```bash
# Run E2E tests
make test-e2e
```

## Test Coverage

The E2E test suite covers gopose's essential functionality:

1. **Dry-run mode**: Verifies `--dry-run` doesn't create files
2. **Normal operation**: Confirms basic `gopose up` works  
3. **Conflict resolution**: Tests port conflict detection and resolution

## Requirements

- [runn](https://github.com/k1LoW/runn) CLI testing framework
- Python 3 (for port conflict simulation)
- Built gopose binary at project root
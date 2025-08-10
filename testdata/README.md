# gopose Test Data

## Directory Structure

```
testdata/
└── gopose/
    ├── gopose_e2e_test.yml       # Main E2E test suite  
    └── fixtures/
        ├── no-conflict.yml       # Basic compose (no conflicts)
        └── port-conflict.yml     # Port conflict scenario
```

## Usage

```bash
# Run E2E tests
make test-e2e
```

## Test Coverage

1. **Dry-run mode**: Verifies `--dry-run` doesn't create files
2. **No conflict case**: Confirms `gopose up` with available ports
3. **Port conflict resolution**: Tests conflict detection and resolution

## Requirements

- [runn](https://github.com/k1LoW/runn) CLI testing framework
- Go toolchain (for building port conflict simulator)
- Built gopose binary at project root
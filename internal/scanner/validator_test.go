package scanner

import (
    "context"
    "testing"

    "github.com/harakeishi/gopose/internal/logger"
    "github.com/harakeishi/gopose/pkg/types"
)

func TestValidatePort(t *testing.T) {
    validator := NewPortValidatorImpl(&logger.NopLogger{})
    if err := validator.ValidatePort(context.Background(), 8080); err != nil {
        t.Fatalf("expected valid port: %v", err)
    }
    if err := validator.ValidatePort(context.Background(), 70000); err == nil {
        t.Fatalf("expected error for invalid port")
    }
}

func TestValidatePortRange(t *testing.T) {
    validator := NewPortValidatorImpl(&logger.NopLogger{})
    pr := types.PortRange{Start: 8000, End: 9000}
    if err := validator.ValidatePortRange(context.Background(), pr); err != nil {
        t.Fatalf("expected valid range: %v", err)
    }
    if err := validator.ValidatePortRange(context.Background(), types.PortRange{Start: 9000, End: 8000}); err == nil {
        t.Fatalf("expected error for reversed range")
    }
}

func TestValidatePortMapping(t *testing.T) {
    validator := NewPortValidatorImpl(&logger.NopLogger{})
    mapping := types.PortMapping{Host: 8080, Container: 80, Protocol: "tcp"}
    if err := validator.ValidatePortMapping(context.Background(), mapping); err != nil {
        t.Fatalf("expected valid mapping: %v", err)
    }
    bad := types.PortMapping{Host: 8080, Container: 80, Protocol: "icmp"}
    if err := validator.ValidatePortMapping(context.Background(), bad); err == nil {
        t.Fatalf("expected invalid protocol error")
    }
}

type stubDetector struct {
    used []int
}

func (s *stubDetector) DetectUsedPorts(ctx context.Context) ([]int, error) {
    return append([]int{}, s.used...), nil
}

func (s *stubDetector) DetectUsedPortsInRange(ctx context.Context, portRange types.PortRange) ([]int, error) {
    var res []int
    for _, port := range s.used {
        if port >= portRange.Start && port <= portRange.End {
            res = append(res, port)
        }
    }
    return res, nil
}

func (s *stubDetector) IsPortInUse(ctx context.Context, port int) (bool, error) {
    for _, used := range s.used {
        if used == port {
            return true, nil
        }
    }
    return false, nil
}

type stubAllocator struct{}

func (s *stubAllocator) AllocatePort(ctx context.Context, config types.PortConfig) (int, error) {
    return config.Range.Start, nil
}

func (s *stubAllocator) AllocatePorts(ctx context.Context, count int, config types.PortConfig) ([]int, error) {
    return []int{config.Range.Start}, nil
}

func (s *stubAllocator) AllocatePortsForServices(ctx context.Context, services []types.Service, config types.PortConfig) (map[string]int, error) {
    return map[string]int{"service": config.Range.Start}, nil
}

type stubValidator struct{}

func (s *stubValidator) ValidatePort(ctx context.Context, port int) error { return nil }
func (s *stubValidator) ValidatePortRange(ctx context.Context, portRange types.PortRange) error { return nil }
func (s *stubValidator) ValidatePortMapping(ctx context.Context, mapping types.PortMapping) error { return nil }

func TestPortScannerScanAndValidate(t *testing.T) {
    detector := &stubDetector{used: []int{8001}}
    allocator := &stubAllocator{}
    validator := &stubValidator{}
    scanner := NewPortScannerImpl(detector, allocator, validator, &logger.NopLogger{})

    result, err := scanner.ScanAndValidate(context.Background(), types.PortRange{Start: 8000, End: 8002})
    if err != nil {
        t.Fatalf("ScanAndValidate error: %v", err)
    }
    if len(result.UsedPorts) != 1 || len(result.AvailablePorts) != 2 {
        t.Fatalf("unexpected scan result: %#v", result)
    }
}

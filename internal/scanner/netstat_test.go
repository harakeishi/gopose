package scanner

import (
    "context"
    "testing"

    "github.com/harakeishi/gopose/internal/logger"
    "github.com/harakeishi/gopose/pkg/types"
)

func TestParseNetstatOutput(t *testing.T) {
    detector := &NetstatPortDetector{logger: &logger.NopLogger{}}
    sample := "tcp4       0      0  127.0.0.1.8080         *.*                    LISTEN\n" +
        "tcp6       0      0  *.9090                 *.*                    LISTEN\n" +
        "udp4       0      0  127.0.0.1.5353         *.*                    LISTEN\n"

    ports, err := detector.parseNetstatOutput(sample)
    if err != nil {
        t.Fatalf("parseNetstatOutput error: %v", err)
    }
    if len(ports) != 3 {
        t.Fatalf("expected 3 ports, got %d", len(ports))
    }
}

type mockDetector struct {
    ports []int
}

func (m *mockDetector) DetectUsedPorts(ctx context.Context) ([]int, error) {
    return append([]int{}, m.ports...), nil
}

func (m *mockDetector) DetectUsedPortsInRange(ctx context.Context, portRange types.PortRange) ([]int, error) {
    var res []int
    for _, port := range m.ports {
        if port >= portRange.Start && port <= portRange.End {
            res = append(res, port)
        }
    }
    return res, nil
}

func (m *mockDetector) IsPortInUse(ctx context.Context, port int) (bool, error) {
    for _, p := range m.ports {
        if p == port {
            return true, nil
        }
    }
    return false, nil
}

func TestPortAllocatorAllocatePort(t *testing.T) {
    allocator := NewPortAllocatorImpl(&mockDetector{ports: []int{8000, 8001}}, &logger.NopLogger{})
    port, err := allocator.AllocatePort(context.Background(), types.PortConfig{
        Range:             types.PortRange{Start: 8000, End: 8005},
        Reserved:          []int{8002},
        ExcludePrivileged: true,
    })
    if err != nil {
        t.Fatalf("AllocatePort error: %v", err)
    }
    if port != 8003 {
        t.Fatalf("expected first available port 8003, got %d", port)
    }
}

func TestPortAllocatorAllocatePorts(t *testing.T) {
    allocator := NewPortAllocatorImpl(&mockDetector{ports: []int{8000}}, &logger.NopLogger{})
    ports, err := allocator.AllocatePorts(context.Background(), 2, types.PortConfig{Range: types.PortRange{Start: 8000, End: 8005}})
    if err != nil {
        t.Fatalf("AllocatePorts error: %v", err)
    }
    if len(ports) != 2 {
        t.Fatalf("expected two ports, got %d", len(ports))
    }
}

func TestAllocatePortsForServices(t *testing.T) {
    services := []types.Service{{Name: "api", Ports: []types.PortMapping{{Host: 80, Container: 80}}}}
    allocator := NewPortAllocatorImpl(&mockDetector{ports: []int{}}, &logger.NopLogger{})
    assignments, err := allocator.AllocatePortsForServices(context.Background(), services, types.PortConfig{Range: types.PortRange{Start: 8000, End: 8005}})
    if err != nil {
        t.Fatalf("AllocatePortsForServices error: %v", err)
    }
    if assignments["api"] == 0 {
        t.Fatalf("expected allocation for api service")
    }
}

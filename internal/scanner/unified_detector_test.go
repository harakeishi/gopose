package scanner

import (
    "context"
    "errors"
    "testing"

    "github.com/harakeishi/gopose/internal/logger"
    "github.com/harakeishi/gopose/pkg/types"
)

type fakePortDetector struct {
    ports []int
}

func (f *fakePortDetector) DetectUsedPorts(ctx context.Context) ([]int, error) {
    if f.ports == nil {
        return nil, errors.New("failed")
    }
    return append([]int{}, f.ports...), nil
}

func (f *fakePortDetector) DetectUsedPortsInRange(ctx context.Context, portRange types.PortRange) ([]int, error) {
    return f.ports, nil
}

func (f *fakePortDetector) IsPortInUse(ctx context.Context, port int) (bool, error) {
    return false, nil
}

type fakeNetworkDetector struct {
    networks []NetworkInfo
    err      error
}

func (f *fakeNetworkDetector) DetectNetworks(ctx context.Context) ([]NetworkInfo, error) {
    if f.err != nil {
        return nil, f.err
    }
    return f.networks, nil
}

func TestUnifiedConflictDetector(t *testing.T) {
    portDetector := &fakePortDetector{ports: []int{8080}}
    networkDetector := &fakeNetworkDetector{networks: []NetworkInfo{{Name: "project_net", Subnets: []string{"10.0.0.0/24"}}}}
    detector := NewUnifiedConflictDetectorImpl(portDetector, networkDetector, &logger.NopLogger{})

    config := &types.ComposeConfig{Services: map[string]types.Service{
        "api": {Ports: []types.PortMapping{{Host: 8080, Container: 8080}}},
    }, Networks: map[string]types.Network{
        "net": {IPAM: types.IPAM{Config: []types.IPAMConfig{{Subnet: "10.0.0.0/24"}}}},
    }}

    info, err := detector.DetectConflicts(context.Background(), config, "project")
    if err != nil {
        t.Fatalf("DetectConflicts error: %v", err)
    }
    if !info.HasConflicts() {
        t.Fatalf("expected conflicts to be detected")
    }
    if info.PortConflicts[0].Type != types.ConflictTypeSystem {
        t.Fatalf("expected system conflict type")
    }
    if len(info.NetworkConflicts) == 0 {
        t.Fatalf("expected network conflict due to subnet")
    }
}

func TestDetectPortConflictsCompose(t *testing.T) {
    portDetector := &fakePortDetector{ports: []int{}}
    detector := NewUnifiedConflictDetectorImpl(portDetector, &fakeNetworkDetector{}, &logger.NopLogger{})
    config := &types.ComposeConfig{Services: map[string]types.Service{
        "api": {Ports: []types.PortMapping{{Host: 8080, Container: 8080}}},
        "web": {Ports: []types.PortMapping{{Host: 8080, Container: 80}}},
    }}

    conflicts, err := detector.DetectPortConflicts(context.Background(), config)
    if err != nil {
        t.Fatalf("DetectPortConflicts error: %v", err)
    }
    if len(conflicts) == 0 || conflicts[0].Type != types.ConflictTypeCompose {
        t.Fatalf("expected compose conflict")
    }
}

func TestDetectNetworkConflictsHandlesError(t *testing.T) {
    detector := NewUnifiedConflictDetectorImpl(&fakePortDetector{ports: []int{}}, &fakeNetworkDetector{err: errors.New("boom")}, &logger.NopLogger{})
    config := &types.ComposeConfig{Networks: map[string]types.Network{
        "net": {IPAM: types.IPAM{Config: []types.IPAMConfig{{Subnet: "10.0.0.0/24"}}}},
    }}

    conflicts, err := detector.DetectConflicts(context.Background(), config, "project")
    if err != nil {
        t.Fatalf("DetectConflicts should continue even with network error: %v", err)
    }
    if len(conflicts.NetworkConflicts) != 0 {
        t.Fatalf("expected empty network conflicts when detection fails")
    }
}

func TestGetServiceNetworkIPs(t *testing.T) {
    detector := NewUnifiedConflictDetectorImpl(&fakePortDetector{}, &fakeNetworkDetector{}, &logger.NopLogger{})
    config := &types.ComposeConfig{Services: map[string]types.Service{
        "api": {Networks: map[string]types.ServiceNetwork{"net": {IPv4Address: "10.0.0.5"}}},
        "web": {Networks: map[string]types.ServiceNetwork{}},
    }}

    ips := detector.getServiceNetworkIPs(config, "net")
    if ips["api"] != "10.0.0.5" {
        t.Fatalf("expected service IP mapping")
    }
}

package resolver

import (
    "context"
    "errors"
    "testing"

    "github.com/harakeishi/gopose/internal/logger"
    "github.com/harakeishi/gopose/pkg/types"
)

type stubPortDetector struct {
    ports []int
}

func (s *stubPortDetector) DetectUsedPorts(ctx context.Context) ([]int, error) {
    return append([]int{}, s.ports...), nil
}

func (s *stubPortDetector) DetectUsedPortsInRange(ctx context.Context, portRange types.PortRange) ([]int, error) {
    var filtered []int
    for _, port := range s.ports {
        if port >= portRange.Start && port <= portRange.End {
            filtered = append(filtered, port)
        }
    }
    return filtered, nil
}

func (s *stubPortDetector) IsPortInUse(ctx context.Context, port int) (bool, error) {
    for _, p := range s.ports {
        if p == port {
            return true, nil
        }
    }
    return false, nil
}

type stubPortAllocator struct {
    ports []int
    idx   int
}

func (s *stubPortAllocator) AllocatePort(ctx context.Context, config types.PortConfig) (int, error) {
    if s.idx >= len(s.ports) {
        return 0, errors.New("no ports")
    }
    port := s.ports[s.idx]
    s.idx++
    return port, nil
}

func (s *stubPortAllocator) AllocatePorts(ctx context.Context, count int, config types.PortConfig) ([]int, error) {
    panic("not implemented")
}

func (s *stubPortAllocator) AllocatePortsForServices(ctx context.Context, services []types.Service, config types.PortConfig) (map[string]int, error) {
    panic("not implemented")
}

func TestDetectPortConflicts(t *testing.T) {
    detector := &ConflictDetectorImpl{
        portDetector: &stubPortDetector{ports: []int{8080}},
        logger:       &logger.NopLogger{},
    }

    config := &types.ComposeConfig{Services: map[string]types.Service{
        "api": {Ports: []types.PortMapping{{Host: 8080, Container: 8080}}},
        "web": {Ports: []types.PortMapping{{Host: 8080, Container: 80}}},
    }}

    conflicts, err := detector.DetectPortConflicts(context.Background(), config)
    if err != nil {
        t.Fatalf("DetectPortConflicts error: %v", err)
    }
    if len(conflicts) < 2 {
        t.Fatalf("expected conflicts for system and compose ports: %#v", conflicts)
    }
}

func TestAnalyzeConflictSeverity(t *testing.T) {
    detector := &ConflictDetectorImpl{logger: &logger.NopLogger{}}
    conflicts := []types.Conflict{{ServiceName: "api", Port: 80, Type: types.ConflictTypeSystem}}
    result := detector.AnalyzeConflictSeverity(context.Background(), conflicts)
    if result["api:80"] != types.ConflictSeverityHigh {
        t.Fatalf("expected high severity for well-known port, got %v", result["api:80"])
    }
}

func TestConflictResolverAutoIncrement(t *testing.T) {
    resolver := &ConflictResolverImpl{
        portAllocator: &stubPortAllocator{ports: []int{9000}},
        portConfig:    types.PortConfig{Range: types.PortRange{Start: 8000, End: 9000}},
        logger:        &logger.NopLogger{},
    }
    conflicts := []types.Conflict{{ServiceName: "api", Port: 8080}}

    resolutions, err := resolver.ResolvePortConflicts(context.Background(), conflicts, types.ResolutionStrategyAutoIncrement)
    if err != nil {
        t.Fatalf("ResolvePortConflicts error: %v", err)
    }
    if len(resolutions) != 1 || resolutions[0].ResolvedPort != 9000 {
        t.Fatalf("expected allocated port 9000: %#v", resolutions)
    }
}

func TestConflictResolverRangeAllocation(t *testing.T) {
    resolver := &ConflictResolverImpl{
        portAllocator: &stubPortAllocator{ports: []int{9000, 9001, 9002}},
        portConfig:    types.PortConfig{Range: types.PortRange{Start: 8000, End: 8015}},
        logger:        &logger.NopLogger{},
    }
    conflicts := []types.Conflict{{ServiceName: "api", Port: 8080}, {ServiceName: "web", Port: 8081}}

    resolutions, err := resolver.ResolvePortConflicts(context.Background(), conflicts, types.ResolutionStrategyRangeAllocation)
    if err != nil {
        t.Fatalf("ResolvePortConflicts error: %v", err)
    }
    if len(resolutions) == 0 {
        t.Fatalf("expected range allocation resolutions")
    }
}

func TestGenerateResolutionSuggestions(t *testing.T) {
    resolver := &ConflictResolverImpl{portAllocator: &stubPortAllocator{}, portConfig: types.PortConfig{Range: types.PortRange{Start: 8000, End: 9000}}, logger: &logger.NopLogger{}}
    suggestions, err := resolver.GenerateResolutionSuggestions(context.Background(), types.Conflict{ServiceName: "api", Port: 7000})
    if err != nil {
        t.Fatalf("GenerateResolutionSuggestions error: %v", err)
    }
    if len(suggestions) < 2 {
        t.Fatalf("expected multiple suggestions, got %d", len(suggestions))
    }
}

func TestPortResolutionAnalyzer(t *testing.T) {
    analyzer := &PortResolutionAnalyzerImpl{logger: &logger.NopLogger{}}
    resolutions := []types.ConflictResolution{
        {ServiceName: "api", ResolvedPort: 9000, Strategy: types.ResolutionStrategyAutoIncrement},
        {ServiceName: "web", ResolvedPort: 0, Strategy: types.ResolutionStrategyRangeAllocation},
    }

    analysis, err := analyzer.AnalyzeResolutionEffectiveness(context.Background(), resolutions)
    if err != nil {
        t.Fatalf("AnalyzeResolutionEffectiveness error: %v", err)
    }
    if analysis.TotalConflicts != 2 || analysis.ResolvedConflicts != 1 {
        t.Fatalf("unexpected analysis results: %#v", analysis)
    }

    optimized, err := analyzer.OptimizeResolutions(context.Background(), resolutions)
    if err != nil {
        t.Fatalf("OptimizeResolutions error: %v", err)
    }
    if len(optimized) != len(resolutions) {
        t.Fatalf("expected same number of resolutions")
    }
}

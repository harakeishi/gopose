package resolver

import (
	"context"
	"testing"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/internal/testutil"
	"github.com/harakeishi/gopose/pkg/types"
)

func TestNewConflictDetectorImpl(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	mockDetector := &testutil.MockPortDetector{}

	detector := NewConflictDetectorImpl(mockDetector, testLogger)

	if detector == nil {
		t.Fatal("NewConflictDetectorImpl() returned nil")
	}

	if detector.portDetector == nil {
		t.Error("portDetector is nil")
	}

	if detector.logger == nil {
		t.Error("logger is nil")
	}
}

func TestDetectPortConflicts(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name                string
		usedPorts           []int
		config              *types.ComposeConfig
		expectedConflicts   int
		expectedSystemType  int
		expectedComposeType int
	}{
		{
			name:      "system port conflict",
			usedPorts: []int{8080, 9000},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			expectedConflicts:   1,
			expectedSystemType:  1,
			expectedComposeType: 0,
		},
		{
			name:      "compose internal port duplicate",
			usedPorts: []int{},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
					"api": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 3000, Protocol: "tcp"},
						},
					},
				},
			},
			expectedConflicts:   1,
			expectedSystemType:  0,
			expectedComposeType: 1,
		},
		{
			name:      "both system and compose conflicts",
			usedPorts: []int{8080},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
					"api": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 3000, Protocol: "tcp"},
						},
					},
				},
			},
			expectedConflicts:   3, // web and api conflict with system port (2) + api duplicates with web (1) = 3
			expectedSystemType:  2, // both web and api conflict with system port
			expectedComposeType: 1, // api duplicates with web
		},
		{
			name:      "skip host port 0",
			usedPorts: []int{8080},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{
							{Host: 0, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			expectedConflicts:   0,
			expectedSystemType:  0,
			expectedComposeType: 0,
		},
		{
			name:      "multiple services with different ports (no conflict)",
			usedPorts: []int{},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
					"api": {
						Ports: []types.PortMapping{
							{Host: 9000, Container: 3000, Protocol: "tcp"},
						},
					},
					"db": {
						Ports: []types.PortMapping{
							{Host: 5432, Container: 5432, Protocol: "tcp"},
						},
					},
				},
			},
			expectedConflicts:   0,
			expectedSystemType:  0,
			expectedComposeType: 0,
		},
		{
			name:      "multiple system port conflicts",
			usedPorts: []int{8080, 9000, 5432},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
					"api": {
						Ports: []types.PortMapping{
							{Host: 9000, Container: 3000, Protocol: "tcp"},
						},
					},
					"db": {
						Ports: []types.PortMapping{
							{Host: 5432, Container: 5432, Protocol: "tcp"},
						},
					},
				},
			},
			expectedConflicts:   3,
			expectedSystemType:  3,
			expectedComposeType: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDetector := &testutil.MockPortDetector{UsedPorts: tt.usedPorts}
			detector := NewConflictDetectorImpl(mockDetector, testLogger)

			conflicts, err := detector.DetectPortConflicts(ctx, tt.config)

			if err != nil {
				t.Fatalf("DetectPortConflicts() error = %v, want nil", err)
			}

			if len(conflicts) != tt.expectedConflicts {
				t.Errorf("DetectPortConflicts() conflicts count = %d, want %d",
					len(conflicts), tt.expectedConflicts)
			}

			systemTypeCount := 0
			composeTypeCount := 0
			for _, conflict := range conflicts {
				if conflict.Type == types.ConflictTypeSystem {
					systemTypeCount++
				}
				if conflict.Type == types.ConflictTypeCompose {
					composeTypeCount++
				}

				// verify conflict has description
				if conflict.Description == "" {
					t.Error("conflict.Description is empty")
				}
			}

			if systemTypeCount != tt.expectedSystemType {
				t.Errorf("System type conflicts = %d, want %d", systemTypeCount, tt.expectedSystemType)
			}

			if composeTypeCount != tt.expectedComposeType {
				t.Errorf("Compose type conflicts = %d, want %d", composeTypeCount, tt.expectedComposeType)
			}
		})
	}
}

func TestAnalyzeConflictSeverity(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()
	mockDetector := &testutil.MockPortDetector{}
	detector := NewConflictDetectorImpl(mockDetector, testLogger)

	tests := []struct {
		name               string
		conflicts          []types.Conflict
		expectedSeverities map[string]types.ConflictSeverity
	}{
		{
			name: "well-known port system conflict (high)",
			conflicts: []types.Conflict{
				{
					ServiceName: "web",
					Port:        80,
					Type:        types.ConflictTypeSystem,
				},
				{
					ServiceName: "db",
					Port:        5432,
					Type:        types.ConflictTypeSystem,
				},
			},
			expectedSeverities: map[string]types.ConflictSeverity{
				"web:80":   types.ConflictSeverityHigh,
				"db:5432":  types.ConflictSeverityHigh,
			},
		},
		{
			name: "non-well-known port system conflict (medium)",
			conflicts: []types.Conflict{
				{
					ServiceName: "app",
					Port:        10000,
					Type:        types.ConflictTypeSystem,
				},
			},
			expectedSeverities: map[string]types.ConflictSeverity{
				"app:10000": types.ConflictSeverityMedium,
			},
		},
		{
			name: "compose internal conflict (high)",
			conflicts: []types.Conflict{
				{
					ServiceName: "web",
					Port:        8080,
					Type:        types.ConflictTypeCompose,
				},
			},
			expectedSeverities: map[string]types.ConflictSeverity{
				"web:8080": types.ConflictSeverityHigh,
			},
		},
		{
			name: "no conflict type (low)",
			conflicts: []types.Conflict{
				{
					ServiceName: "web",
					Port:        8080,
					Type:        types.ConflictTypeNone,
				},
			},
			expectedSeverities: map[string]types.ConflictSeverity{
				"web:8080": types.ConflictSeverityLow,
			},
		},
		{
			name: "mixed conflicts",
			conflicts: []types.Conflict{
				{ServiceName: "web", Port: 80, Type: types.ConflictTypeSystem},
				{ServiceName: "api", Port: 8080, Type: types.ConflictTypeCompose},
				{ServiceName: "app", Port: 12345, Type: types.ConflictTypeSystem},
			},
			expectedSeverities: map[string]types.ConflictSeverity{
				"web:80":     types.ConflictSeverityHigh,
				"api:8080":   types.ConflictSeverityHigh,
				"app:12345":  types.ConflictSeverityMedium,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.AnalyzeConflictSeverity(ctx, tt.conflicts)

			if len(result) != len(tt.expectedSeverities) {
				t.Errorf("AnalyzeConflictSeverity() result count = %d, want %d",
					len(result), len(tt.expectedSeverities))
			}

			for key, expectedSeverity := range tt.expectedSeverities {
				if result[key] != expectedSeverity {
					t.Errorf("AnalyzeConflictSeverity()[%s] = %v, want %v",
						key, result[key], expectedSeverity)
				}
			}
		})
	}
}

func TestIsWellKnownPort(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	mockDetector := &testutil.MockPortDetector{}
	detector := NewConflictDetectorImpl(mockDetector, testLogger)

	tests := []struct {
		name        string
		port        int
		isWellKnown bool
	}{
		{name: "HTTP", port: 80, isWellKnown: true},
		{name: "HTTPS", port: 443, isWellKnown: true},
		{name: "SSH", port: 22, isWellKnown: true},
		{name: "PostgreSQL", port: 5432, isWellKnown: true},
		{name: "MySQL", port: 3306, isWellKnown: true},
		{name: "Redis", port: 6379, isWellKnown: true},
		{name: "dev port 8080", port: 8080, isWellKnown: true},
		{name: "dev port 3000", port: 3000, isWellKnown: true},
		{name: "non-well-known port", port: 12345, isWellKnown: false},
		{name: "high port", port: 50000, isWellKnown: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.isWellKnownPort(tt.port)
			if result != tt.isWellKnown {
				t.Errorf("isWellKnownPort(%d) = %v, want %v", tt.port, result, tt.isWellKnown)
			}
		})
	}
}

func TestNewConflictResolverImpl(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	mockAllocator := &testutil.MockPortAllocator{NextPort: 8000}

	resolver := NewConflictResolverImpl(mockAllocator, testLogger)

	if resolver == nil {
		t.Fatal("NewConflictResolverImpl() returned nil")
	}

	if resolver.portAllocator == nil {
		t.Error("portAllocator is nil")
	}

	if resolver.logger == nil {
		t.Error("logger is nil")
	}

	// verify default port config
	if resolver.portConfig.Range.Start != 8000 {
		t.Errorf("default portConfig.Range.Start = %d, want 8000", resolver.portConfig.Range.Start)
	}

	if resolver.portConfig.Range.End != 9000 {
		t.Errorf("default portConfig.Range.End = %d, want 9000", resolver.portConfig.Range.End)
	}

	if !resolver.portConfig.ExcludePrivileged {
		t.Error("default portConfig.ExcludePrivileged should be true")
	}
}

func TestNewConflictResolverWithPortConfig(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	mockAllocator := &testutil.MockPortAllocator{NextPort: 8000}

	customConfig := types.PortConfig{
		Range:             types.PortRange{Start: 10000, End: 11000},
		Reserved:          []int{10100, 10200},
		ExcludePrivileged: false,
	}

	resolver := NewConflictResolverWithPortConfig(mockAllocator, customConfig, testLogger)

	if resolver == nil {
		t.Fatal("NewConflictResolverWithPortConfig() returned nil")
	}

	if resolver.portConfig.Range.Start != 10000 {
		t.Errorf("portConfig.Range.Start = %d, want 10000", resolver.portConfig.Range.Start)
	}

	if resolver.portConfig.Range.End != 11000 {
		t.Errorf("portConfig.Range.End = %d, want 11000", resolver.portConfig.Range.End)
	}

	if len(resolver.portConfig.Reserved) != 2 {
		t.Errorf("len(portConfig.Reserved) = %d, want 2", len(resolver.portConfig.Reserved))
	}

	if resolver.portConfig.ExcludePrivileged {
		t.Error("portConfig.ExcludePrivileged should be false")
	}
}

func TestResolvePortConflicts(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name              string
		conflicts         []types.Conflict
		strategy          types.ResolutionStrategy
		expectedResolved  int
		nextPort          int
	}{
		{
			name: "single conflict resolution with AutoIncrement",
			conflicts: []types.Conflict{
				{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystem},
			},
			strategy:         types.ResolutionStrategyAutoIncrement,
			expectedResolved: 1,
			nextPort:         8081,
		},
		{
			name: "multiple conflict resolution with AutoIncrement",
			conflicts: []types.Conflict{
				{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystem},
				{ServiceName: "api", Port: 9000, Type: types.ConflictTypeSystem},
				{ServiceName: "db", Port: 5432, Type: types.ConflictTypeSystem},
			},
			strategy:         types.ResolutionStrategyAutoIncrement,
			expectedResolved: 3,
			nextPort:         8000,
		},
		{
			name: "RangeAllocation strategy",
			conflicts: []types.Conflict{
				{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystem},
				{ServiceName: "api", Port: 9000, Type: types.ConflictTypeSystem},
			},
			strategy:         types.ResolutionStrategyRangeAllocation,
			expectedResolved: 2,
			nextPort:         8000,
		},
		{
			name:             "no conflicts",
			conflicts:        []types.Conflict{},
			strategy:         types.ResolutionStrategyAutoIncrement,
			expectedResolved: 0,
			nextPort:         8000,
		},
		{
			name: "UserDefined strategy (fallback to AutoIncrement)",
			conflicts: []types.Conflict{
				{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystem},
			},
			strategy:         types.ResolutionStrategyUserDefined,
			expectedResolved: 1,
			nextPort:         8081,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAllocator := &testutil.MockPortAllocator{NextPort: tt.nextPort}
			resolver := NewConflictResolverImpl(mockAllocator, testLogger)

			resolutions, err := resolver.ResolvePortConflicts(ctx, tt.conflicts, tt.strategy)

			if err != nil {
				t.Fatalf("ResolvePortConflicts() error = %v, want nil", err)
			}

			if len(resolutions) != tt.expectedResolved {
				t.Errorf("ResolvePortConflicts() resolutions count = %d, want %d",
					len(resolutions), tt.expectedResolved)
			}

			// verify each resolution
			for i, resolution := range resolutions {
				if resolution.ResolvedPort == 0 {
					t.Errorf("resolutions[%d].ResolvedPort is 0", i)
				}

				if resolution.ServiceName == "" {
					t.Errorf("resolutions[%d].ServiceName is empty", i)
				}

				if resolution.Strategy != tt.strategy && tt.strategy != types.ResolutionStrategyUserDefined {
					t.Errorf("resolutions[%d].Strategy = %v, want %v",
						i, resolution.Strategy, tt.strategy)
				}

				if resolution.Reason == "" {
					t.Errorf("resolutions[%d].Reason is empty", i)
				}
			}
		})
	}
}

func TestGenerateResolutionSuggestions(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()
	mockAllocator := &testutil.MockPortAllocator{NextPort: 8081}
	resolver := NewConflictResolverImpl(mockAllocator, testLogger)

	tests := []struct {
		name                 string
		conflict             types.Conflict
		expectedMinSuggestions int
	}{
		{
			name: "suggestions for normal port",
			conflict: types.Conflict{
				ServiceName: "web",
				Port:        8080,
				Type:        types.ConflictTypeSystem,
			},
			expectedMinSuggestions: 2, // AutoIncrement + RangeAllocation
		},
		{
			name: "suggestions for low port (including move to 8000 range)",
			conflict: types.Conflict{
				ServiceName: "web",
				Port:        80,
				Type:        types.ConflictTypeSystem,
			},
			expectedMinSuggestions: 3, // AutoIncrement + RangeAllocation + move to 8000 range
		},
		{
			name: "suggestions for high port",
			conflict: types.Conflict{
				ServiceName: "app",
				Port:        50000,
				Type:        types.ConflictTypeSystem,
			},
			expectedMinSuggestions: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions, err := resolver.GenerateResolutionSuggestions(ctx, tt.conflict)

			if err != nil {
				t.Fatalf("GenerateResolutionSuggestions() error = %v, want nil", err)
			}

			if len(suggestions) < tt.expectedMinSuggestions {
				t.Errorf("GenerateResolutionSuggestions() suggestions count = %d, want at least %d",
					len(suggestions), tt.expectedMinSuggestions)
			}

			// verify each suggestion
			for i, suggestion := range suggestions {
				if suggestion.ResolvedPort == 0 {
					t.Errorf("suggestions[%d].ResolvedPort is 0", i)
				}

				if suggestion.ConflictPort != tt.conflict.Port {
					t.Errorf("suggestions[%d].ConflictPort = %d, want %d",
						i, suggestion.ConflictPort, tt.conflict.Port)
				}

				if suggestion.Reason == "" {
					t.Errorf("suggestions[%d].Reason is empty", i)
				}
			}
		})
	}
}

func TestPortResolutionAnalyzerImpl(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	analyzer := NewPortResolutionAnalyzerImpl(testLogger)

	if analyzer == nil {
		t.Fatal("NewPortResolutionAnalyzerImpl() returned nil")
	}

	if analyzer.logger == nil {
		t.Error("logger is nil")
	}
}

func TestAnalyzeResolutionEffectiveness(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()
	analyzer := NewPortResolutionAnalyzerImpl(testLogger)

	tests := []struct {
		name                   string
		resolutions            []types.ConflictResolution
		expectedResolvedCount  int
		expectedSuccessRate    float64
	}{
		{
			name: "all resolved",
			resolutions: []types.ConflictResolution{
				{ConflictPort: 8080, ResolvedPort: 8081, Strategy: types.ResolutionStrategyAutoIncrement},
				{ConflictPort: 9000, ResolvedPort: 9001, Strategy: types.ResolutionStrategyAutoIncrement},
			},
			expectedResolvedCount: 2,
			expectedSuccessRate:   100.0,
		},
		{
			name: "partially unresolved",
			resolutions: []types.ConflictResolution{
				{ConflictPort: 8080, ResolvedPort: 8081, Strategy: types.ResolutionStrategyAutoIncrement},
				{ConflictPort: 9000, ResolvedPort: 0, Strategy: types.ResolutionStrategyAutoIncrement},
			},
			expectedResolvedCount: 1,
			expectedSuccessRate:   50.0,
		},
		{
			name:                  "no resolutions",
			resolutions:           []types.ConflictResolution{},
			expectedResolvedCount: 0,
			expectedSuccessRate:   0.0,
		},
		{
			name: "all unresolved",
			resolutions: []types.ConflictResolution{
				{ConflictPort: 8080, ResolvedPort: 0, Strategy: types.ResolutionStrategyAutoIncrement},
				{ConflictPort: 9000, ResolvedPort: 0, Strategy: types.ResolutionStrategyAutoIncrement},
			},
			expectedResolvedCount: 0,
			expectedSuccessRate:   0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis, err := analyzer.AnalyzeResolutionEffectiveness(ctx, tt.resolutions)

			if err != nil {
				t.Fatalf("AnalyzeResolutionEffectiveness() error = %v, want nil", err)
			}

			if analysis.ResolvedConflicts != tt.expectedResolvedCount {
				t.Errorf("ResolvedConflicts = %d, want %d",
					analysis.ResolvedConflicts, tt.expectedResolvedCount)
			}

			if analysis.SuccessRate != tt.expectedSuccessRate {
				t.Errorf("SuccessRate = %.2f, want %.2f",
					analysis.SuccessRate, tt.expectedSuccessRate)
			}

			if analysis.TotalConflicts != len(tt.resolutions) {
				t.Errorf("TotalConflicts = %d, want %d",
					analysis.TotalConflicts, len(tt.resolutions))
			}
		})
	}
}

func TestOptimizeResolutions(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()
	analyzer := NewPortResolutionAnalyzerImpl(testLogger)

	tests := []struct {
		name                string
		resolutions         []types.ConflictResolution
		expectedOptimized   int
	}{
		{
			name: "no duplicates",
			resolutions: []types.ConflictResolution{
				{ConflictPort: 8080, ResolvedPort: 8081, ServiceName: "web"},
				{ConflictPort: 9000, ResolvedPort: 9001, ServiceName: "api"},
			},
			expectedOptimized: 2,
		},
		{
			name: "with duplicates (adjusted)",
			resolutions: []types.ConflictResolution{
				{ConflictPort: 8080, ResolvedPort: 8081, ServiceName: "web"},
				{ConflictPort: 9000, ResolvedPort: 8081, ServiceName: "api"}, // duplicate
			},
			expectedOptimized: 2,
		},
		{
			name: "multiple duplicates",
			resolutions: []types.ConflictResolution{
				{ConflictPort: 8080, ResolvedPort: 8100, ServiceName: "web"},
				{ConflictPort: 9000, ResolvedPort: 8100, ServiceName: "api"},
				{ConflictPort: 5432, ResolvedPort: 8100, ServiceName: "db"},
			},
			expectedOptimized: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			optimized, err := analyzer.OptimizeResolutions(ctx, tt.resolutions)

			if err != nil {
				t.Fatalf("OptimizeResolutions() error = %v, want nil", err)
			}

			if len(optimized) != tt.expectedOptimized {
				t.Errorf("OptimizeResolutions() count = %d, want %d",
					len(optimized), tt.expectedOptimized)
			}

			// verify no duplicate ports
			usedPorts := make(map[int]bool)
			for _, resolution := range optimized {
				if usedPorts[resolution.ResolvedPort] {
					t.Errorf("Duplicate resolved port %d found after optimization",
						resolution.ResolvedPort)
				}
				usedPorts[resolution.ResolvedPort] = true
			}
		})
	}
}

func TestGetPortRangeKey(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	analyzer := NewPortResolutionAnalyzerImpl(testLogger)

	tests := []struct {
		name        string
		port        int
		expectedKey string
	}{
		{name: "system port", port: 80, expectedKey: "system_ports"},
		{name: "system port upper bound", port: 1023, expectedKey: "system_ports"},
		{name: "registered port", port: 1024, expectedKey: "registered_ports"},
		{name: "registered port range", port: 3000, expectedKey: "registered_ports"},
		{name: "custom port", port: 5000, expectedKey: "custom_ports"},
		{name: "custom port range", port: 7999, expectedKey: "custom_ports"},
		{name: "development port", port: 8000, expectedKey: "development_ports"},
		{name: "development port range", port: 8999, expectedKey: "development_ports"},
		{name: "high port", port: 9000, expectedKey: "high_ports"},
		{name: "high port range", port: 50000, expectedKey: "high_ports"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.getPortRangeKey(tt.port)
			if result != tt.expectedKey {
				t.Errorf("getPortRangeKey(%d) = %s, want %s", tt.port, result, tt.expectedKey)
			}
		})
	}
}

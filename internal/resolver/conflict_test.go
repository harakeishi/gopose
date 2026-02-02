package resolver

import (
	"context"
	"testing"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

// mockPortDetectorForResolver はテスト用のモックポート検出器
type mockPortDetectorForResolver struct {
	usedPorts []int
	err       error
}

func (m *mockPortDetectorForResolver) DetectUsedPorts(ctx context.Context) ([]int, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.usedPorts, nil
}

func (m *mockPortDetectorForResolver) DetectUsedPortsInRange(ctx context.Context, portRange types.PortRange) ([]int, error) {
	if m.err != nil {
		return nil, m.err
	}
	var portsInRange []int
	for _, port := range m.usedPorts {
		if port >= portRange.Start && port <= portRange.End {
			portsInRange = append(portsInRange, port)
		}
	}
	return portsInRange, nil
}

func (m *mockPortDetectorForResolver) IsPortInUse(ctx context.Context, port int) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	for _, p := range m.usedPorts {
		if p == port {
			return true, nil
		}
	}
	return false, nil
}

// mockPortAllocatorForResolver はテスト用のモックポート割り当て器
type mockPortAllocatorForResolver struct {
	nextPort int
	err      error
}

func (m *mockPortAllocatorForResolver) AllocatePort(ctx context.Context, config types.PortConfig) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	port := m.nextPort
	m.nextPort++
	return port, nil
}

func (m *mockPortAllocatorForResolver) AllocatePorts(ctx context.Context, count int, config types.PortConfig) ([]int, error) {
	if m.err != nil {
		return nil, m.err
	}
	ports := make([]int, count)
	for i := 0; i < count; i++ {
		ports[i] = m.nextPort
		m.nextPort++
	}
	return ports, nil
}

func (m *mockPortAllocatorForResolver) AllocatePortsForServices(ctx context.Context, services []types.Service, config types.PortConfig) (map[string]int, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make(map[string]int)
	for _, service := range services {
		result[service.Name] = m.nextPort
		m.nextPort++
	}
	return result, nil
}

func TestNewConflictDetectorImpl(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	mockDetector := &mockPortDetectorForResolver{}

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
			name:      "システムポート衝突",
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
			name:      "Compose内ポート重複",
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
			name:      "システムとCompose両方の衝突",
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
			expectedConflicts:   3, // webとapiがシステムポートと衝突(2) + apiがwebと重複(1) = 3
			expectedSystemType:  2, // webとapi両方がシステムポートと衝突
			expectedComposeType: 1, // apiがwebと重複
		},
		{
			name:      "ホストポート0はスキップ",
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
			name:      "複数サービスで異なるポートを使用（衝突なし）",
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
			name:      "複数のシステムポート衝突",
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
			mockDetector := &mockPortDetectorForResolver{usedPorts: tt.usedPorts}
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

				// 衝突に説明が含まれることを確認
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
	mockDetector := &mockPortDetectorForResolver{}
	detector := NewConflictDetectorImpl(mockDetector, testLogger)

	tests := []struct {
		name               string
		conflicts          []types.Conflict
		expectedSeverities map[string]types.ConflictSeverity
	}{
		{
			name: "有名ポートのシステム衝突（高）",
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
			name: "非有名ポートのシステム衝突（中）",
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
			name: "Compose内衝突（高）",
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
			name: "衝突タイプなし（低）",
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
			name: "混在衝突",
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
	mockDetector := &mockPortDetectorForResolver{}
	detector := NewConflictDetectorImpl(mockDetector, testLogger)
	_ = detector // 使用する変数として明示

	tests := []struct {
		name       string
		port       int
		isWellKnown bool
	}{
		{name: "HTTP", port: 80, isWellKnown: true},
		{name: "HTTPS", port: 443, isWellKnown: true},
		{name: "SSH", port: 22, isWellKnown: true},
		{name: "PostgreSQL", port: 5432, isWellKnown: true},
		{name: "MySQL", port: 3306, isWellKnown: true},
		{name: "Redis", port: 6379, isWellKnown: true},
		{name: "開発用8080", port: 8080, isWellKnown: true},
		{name: "開発用3000", port: 3000, isWellKnown: true},
		{name: "非有名ポート", port: 12345, isWellKnown: false},
		{name: "高ポート", port: 50000, isWellKnown: false},
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
	mockAllocator := &mockPortAllocatorForResolver{nextPort: 8000}

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

	// デフォルトポート設定を確認
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
	mockAllocator := &mockPortAllocatorForResolver{nextPort: 8000}

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
			name: "AutoIncrement戦略で単一衝突解決",
			conflicts: []types.Conflict{
				{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystem},
			},
			strategy:         types.ResolutionStrategyAutoIncrement,
			expectedResolved: 1,
			nextPort:         8081,
		},
		{
			name: "AutoIncrement戦略で複数衝突解決",
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
			name: "RangeAllocation戦略",
			conflicts: []types.Conflict{
				{ServiceName: "web", Port: 8080, Type: types.ConflictTypeSystem},
				{ServiceName: "api", Port: 9000, Type: types.ConflictTypeSystem},
			},
			strategy:         types.ResolutionStrategyRangeAllocation,
			expectedResolved: 2,
			nextPort:         8000,
		},
		{
			name:             "衝突なし",
			conflicts:        []types.Conflict{},
			strategy:         types.ResolutionStrategyAutoIncrement,
			expectedResolved: 0,
			nextPort:         8000,
		},
		{
			name: "UserDefined戦略（AutoIncrementにフォールバック）",
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
			mockAllocator := &mockPortAllocatorForResolver{nextPort: tt.nextPort}
			resolver := NewConflictResolverImpl(mockAllocator, testLogger)

			resolutions, err := resolver.ResolvePortConflicts(ctx, tt.conflicts, tt.strategy)

			if err != nil {
				t.Fatalf("ResolvePortConflicts() error = %v, want nil", err)
			}

			if len(resolutions) != tt.expectedResolved {
				t.Errorf("ResolvePortConflicts() resolutions count = %d, want %d",
					len(resolutions), tt.expectedResolved)
			}

			// 各解決案の検証
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
	mockAllocator := &mockPortAllocatorForResolver{nextPort: 8081}
	resolver := NewConflictResolverImpl(mockAllocator, testLogger)

	tests := []struct {
		name                 string
		conflict             types.Conflict
		expectedMinSuggestions int
	}{
		{
			name: "通常ポートの提案",
			conflict: types.Conflict{
				ServiceName: "web",
				Port:        8080,
				Type:        types.ConflictTypeSystem,
			},
			expectedMinSuggestions: 2, // AutoIncrement + RangeAllocation
		},
		{
			name: "低ポート番号の提案（8000番台への移動提案を含む）",
			conflict: types.Conflict{
				ServiceName: "web",
				Port:        80,
				Type:        types.ConflictTypeSystem,
			},
			expectedMinSuggestions: 3, // AutoIncrement + RangeAllocation + 8000番台移動
		},
		{
			name: "高ポート番号の提案",
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

			// 各提案の検証
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
			name: "全て解決済み",
			resolutions: []types.ConflictResolution{
				{ConflictPort: 8080, ResolvedPort: 8081, Strategy: types.ResolutionStrategyAutoIncrement},
				{ConflictPort: 9000, ResolvedPort: 9001, Strategy: types.ResolutionStrategyAutoIncrement},
			},
			expectedResolvedCount: 2,
			expectedSuccessRate:   100.0,
		},
		{
			name: "一部未解決",
			resolutions: []types.ConflictResolution{
				{ConflictPort: 8080, ResolvedPort: 8081, Strategy: types.ResolutionStrategyAutoIncrement},
				{ConflictPort: 9000, ResolvedPort: 0, Strategy: types.ResolutionStrategyAutoIncrement},
			},
			expectedResolvedCount: 1,
			expectedSuccessRate:   50.0,
		},
		{
			name:                  "解決案なし",
			resolutions:           []types.ConflictResolution{},
			expectedResolvedCount: 0,
			expectedSuccessRate:   0.0,
		},
		{
			name: "全て未解決",
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
			name: "重複なし",
			resolutions: []types.ConflictResolution{
				{ConflictPort: 8080, ResolvedPort: 8081, ServiceName: "web"},
				{ConflictPort: 9000, ResolvedPort: 9001, ServiceName: "api"},
			},
			expectedOptimized: 2,
		},
		{
			name: "重複あり（調整される）",
			resolutions: []types.ConflictResolution{
				{ConflictPort: 8080, ResolvedPort: 8081, ServiceName: "web"},
				{ConflictPort: 9000, ResolvedPort: 8081, ServiceName: "api"}, // 重複
			},
			expectedOptimized: 2,
		},
		{
			name: "複数の重複",
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

			// ポートの重複がないことを確認
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
		{name: "システムポート", port: 80, expectedKey: "system_ports"},
		{name: "システムポート上限", port: 1023, expectedKey: "system_ports"},
		{name: "登録ポート", port: 1024, expectedKey: "registered_ports"},
		{name: "登録ポート範囲", port: 3000, expectedKey: "registered_ports"},
		{name: "カスタムポート", port: 5000, expectedKey: "custom_ports"},
		{name: "カスタムポート範囲", port: 7999, expectedKey: "custom_ports"},
		{name: "開発ポート", port: 8000, expectedKey: "development_ports"},
		{name: "開発ポート範囲", port: 8999, expectedKey: "development_ports"},
		{name: "高ポート", port: 9000, expectedKey: "high_ports"},
		{name: "高ポート範囲", port: 50000, expectedKey: "high_ports"},
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

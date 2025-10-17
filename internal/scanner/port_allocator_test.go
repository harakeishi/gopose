package scanner

import (
	"context"
	"testing"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

// mockPortDetector はテスト用のモックポート検出器
type mockPortDetector struct {
	usedPorts []int
}

func (m *mockPortDetector) DetectUsedPorts(ctx context.Context) ([]int, error) {
	return m.usedPorts, nil
}

func (m *mockPortDetector) DetectUsedPortsInRange(ctx context.Context, portRange types.PortRange) ([]int, error) {
	var portsInRange []int
	for _, port := range m.usedPorts {
		if port >= portRange.Start && port <= portRange.End {
			portsInRange = append(portsInRange, port)
		}
	}
	return portsInRange, nil
}

func (m *mockPortDetector) IsPortInUse(ctx context.Context, port int) (bool, error) {
	for _, p := range m.usedPorts {
		if p == port {
			return true, nil
		}
	}
	return false, nil
}

func TestAllocatePort_WithReservedPorts(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name          string
		usedPorts     []int
		reservedPorts []int
		portRange     types.PortRange
		expectedPort  int
	}{
		{
			name:          "予約ポートをスキップして次の利用可能ポートを割り当て",
			usedPorts:     []int{},
			reservedPorts: []int{8000, 8001, 8002},
			portRange:     types.PortRange{Start: 8000, End: 8100},
			expectedPort:  8003,
		},
		{
			name:          "使用中ポートと予約ポートの両方をスキップ",
			usedPorts:     []int{8003, 8004},
			reservedPorts: []int{8000, 8001, 8002},
			portRange:     types.PortRange{Start: 8000, End: 8100},
			expectedPort:  8005,
		},
		{
			name:          "範囲中間の予約ポートをスキップ",
			usedPorts:     []int{},
			reservedPorts: []int{8010, 8020, 8030},
			portRange:     types.PortRange{Start: 8000, End: 8100},
			expectedPort:  8000,
		},
		{
			name:          "最初が予約され、次が利用可能",
			usedPorts:     []int{},
			reservedPorts: []int{9000},
			portRange:     types.PortRange{Start: 9000, End: 9100},
			expectedPort:  9001,
		},
		{
			name:          "複数の予約ポートをスキップ",
			usedPorts:     []int{},
			reservedPorts: []int{7000, 7001, 7002, 7003, 7004, 7005, 7006, 7007, 7008, 7009},
			portRange:     types.PortRange{Start: 7000, End: 7100},
			expectedPort:  7010,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDetector := &mockPortDetector{usedPorts: tt.usedPorts}
			allocator := NewPortAllocatorImpl(mockDetector, testLogger)

			config := types.PortConfig{
				Range:             tt.portRange,
				Reserved:          tt.reservedPorts,
				ExcludePrivileged: false,
			}

			port, err := allocator.AllocatePort(ctx, config)
			if err != nil {
				t.Errorf("AllocatePort() error = %v, want nil", err)
				return
			}

			if port != tt.expectedPort {
				t.Errorf("AllocatePort() = %d, want %d", port, tt.expectedPort)
			}

			// 予約ポートに割り当てられていないことを確認
			for _, reserved := range tt.reservedPorts {
				if port == reserved {
					t.Errorf("AllocatePort() = %d, which is a reserved port", port)
				}
			}
		})
	}
}

func TestAllocatePorts_WithReservedPorts(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name          string
		usedPorts     []int
		reservedPorts []int
		portRange     types.PortRange
		count         int
		expectedPorts []int
	}{
		{
			name:          "複数ポート割り当てで予約ポートをスキップ",
			usedPorts:     []int{},
			reservedPorts: []int{8000, 8001, 8002},
			portRange:     types.PortRange{Start: 8000, End: 8100},
			count:         3,
			expectedPorts: []int{8003, 8004, 8005},
		},
		{
			name:          "予約ポートと使用中ポートをスキップして連続割り当て",
			usedPorts:     []int{8004, 8005},
			reservedPorts: []int{8000, 8001, 8002},
			portRange:     types.PortRange{Start: 8000, End: 8100},
			count:         3,
			expectedPorts: []int{8003, 8006, 8007},
		},
		{
			name:          "予約ポートが範囲全体に散在",
			usedPorts:     []int{},
			reservedPorts: []int{8000, 8002, 8004, 8006, 8008},
			portRange:     types.PortRange{Start: 8000, End: 8100},
			count:         5,
			expectedPorts: []int{8001, 8003, 8005, 8007, 8009},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDetector := &mockPortDetector{usedPorts: tt.usedPorts}
			allocator := NewPortAllocatorImpl(mockDetector, testLogger)

			config := types.PortConfig{
				Range:             tt.portRange,
				Reserved:          tt.reservedPorts,
				ExcludePrivileged: false,
			}

			ports, err := allocator.AllocatePorts(ctx, tt.count, config)
			if err != nil {
				t.Errorf("AllocatePorts() error = %v, want nil", err)
				return
			}

			if len(ports) != len(tt.expectedPorts) {
				t.Errorf("AllocatePorts() returned %d ports, want %d", len(ports), len(tt.expectedPorts))
				return
			}

			for i, port := range ports {
				if port != tt.expectedPorts[i] {
					t.Errorf("AllocatePorts()[%d] = %d, want %d", i, port, tt.expectedPorts[i])
				}

				// 予約ポートに割り当てられていないことを確認
				for _, reserved := range tt.reservedPorts {
					if port == reserved {
						t.Errorf("AllocatePorts() allocated reserved port %d", port)
					}
				}

				// 使用中ポートに割り当てられていないことを確認
				for _, used := range tt.usedPorts {
					if port == used {
						t.Errorf("AllocatePorts() allocated used port %d", port)
					}
				}
			}
		})
	}
}

func TestAllocatePortsForServices_WithReservedPorts(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name          string
		usedPorts     []int
		reservedPorts []int
		portRange     types.PortRange
		services      []types.Service
		expectedPorts map[string]int
	}{
		{
			name:          "各サービスに予約ポートをスキップして割り当て",
			usedPorts:     []int{},
			reservedPorts: []int{8000, 8001},
			portRange:     types.PortRange{Start: 8000, End: 8100},
			services: []types.Service{
				{Name: "web", Ports: []types.PortMapping{{Host: 80, Container: 80}}},
				{Name: "api", Ports: []types.PortMapping{{Host: 443, Container: 80}}},
			},
			expectedPorts: map[string]int{
				"web": 8002,
				"api": 8003,
			},
		},
		{
			name:          "予約ポートが散在する中でサービスに割り当て",
			usedPorts:     []int{},
			reservedPorts: []int{8000, 8002, 8004},
			portRange:     types.PortRange{Start: 8000, End: 8100},
			services: []types.Service{
				{Name: "web", Ports: []types.PortMapping{{Host: 80, Container: 80}}},
				{Name: "api", Ports: []types.PortMapping{{Host: 443, Container: 80}}},
				{Name: "db", Ports: []types.PortMapping{{Host: 5432, Container: 5432}}},
			},
			expectedPorts: map[string]int{
				"web": 8001,
				"api": 8003,
				"db":  8005,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDetector := &mockPortDetector{usedPorts: tt.usedPorts}
			allocator := NewPortAllocatorImpl(mockDetector, testLogger)

			config := types.PortConfig{
				Range:             tt.portRange,
				Reserved:          tt.reservedPorts,
				ExcludePrivileged: false,
			}

			result, err := allocator.AllocatePortsForServices(ctx, tt.services, config)
			if err != nil {
				t.Errorf("AllocatePortsForServices() error = %v, want nil", err)
				return
			}

			if len(result) != len(tt.expectedPorts) {
				t.Errorf("AllocatePortsForServices() returned %d services, want %d", len(result), len(tt.expectedPorts))
				return
			}

			for serviceName, expectedPort := range tt.expectedPorts {
				actualPort, exists := result[serviceName]
				if !exists {
					t.Errorf("AllocatePortsForServices() missing service %s", serviceName)
					continue
				}

				if actualPort != expectedPort {
					t.Errorf("AllocatePortsForServices()[%s] = %d, want %d", serviceName, actualPort, expectedPort)
				}

				// 予約ポートに割り当てられていないことを確認
				for _, reserved := range tt.reservedPorts {
					if actualPort == reserved {
						t.Errorf("AllocatePortsForServices() allocated reserved port %d to service %s", actualPort, serviceName)
					}
				}
			}
		})
	}
}

func TestReservedPorts_UnusedPortsAreStillReserved(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	// 未使用のポート範囲で、いくつかのポートを予約
	mockDetector := &mockPortDetector{usedPorts: []int{}} // 使用中のポートなし
	allocator := NewPortAllocatorImpl(mockDetector, testLogger)

	config := types.PortConfig{
		Range:             types.PortRange{Start: 9000, End: 9100},
		Reserved:          []int{9000, 9001, 9002, 9050, 9051, 9052}, // 未使用でも予約
		ExcludePrivileged: false,
	}

	// 10個のポートを割り当て
	ports, err := allocator.AllocatePorts(ctx, 10, config)
	if err != nil {
		t.Fatalf("AllocatePorts() error = %v, want nil", err)
	}

	// 予約ポートが割り当てられていないことを確認
	reservedMap := make(map[int]bool)
	for _, reserved := range config.Reserved {
		reservedMap[reserved] = true
	}

	for _, port := range ports {
		if reservedMap[port] {
			t.Errorf("AllocatePorts() allocated reserved port %d, even though it was unused", port)
		}
	}

	// 期待される最初の利用可能ポートは9003（9000-9002は予約済み）
	expectedFirstPort := 9003
	if ports[0] != expectedFirstPort {
		t.Errorf("AllocatePorts()[0] = %d, want %d (first non-reserved port)", ports[0], expectedFirstPort)
	}
}

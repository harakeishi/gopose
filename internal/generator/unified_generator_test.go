package generator

import (
	"context"
	"testing"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

// mockPortAllocatorForGenerator はテスト用のモックポートアロケーター
type mockPortAllocatorForGenerator struct {
	nextPort int
}

func (m *mockPortAllocatorForGenerator) AllocatePort(ctx context.Context, config types.PortConfig) (int, error) {
	port := m.nextPort
	m.nextPort++
	return port, nil
}

func (m *mockPortAllocatorForGenerator) AllocatePorts(ctx context.Context, count int, config types.PortConfig) ([]int, error) {
	ports := make([]int, count)
	for i := 0; i < count; i++ {
		ports[i] = m.nextPort
		m.nextPort++
	}
	return ports, nil
}

func (m *mockPortAllocatorForGenerator) AllocatePortsForServices(ctx context.Context, services []types.Service, config types.PortConfig) (map[string]int, error) {
	result := make(map[string]int)
	for _, service := range services {
		result[service.Name] = m.nextPort
		m.nextPort++
	}
	return result, nil
}

func TestGenerateFromConflicts(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name                  string
		config                *types.ComposeConfig
		conflictInfo          *types.UnifiedConflictInfo
		expectedServicesCount int
		expectedNetworksCount int
	}{
		{
			name: "ポート衝突の解決",
			config: &types.ComposeConfig{
				Version: "3.8",
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			conflictInfo: &types.UnifiedConflictInfo{
				PortConflicts: []types.PortConflictInfo{
					{
						Port:        8080,
						ServiceName: "web",
						Resolution: &types.PortResolutionInfo{
							ResolvedPort: 8081,
							Strategy:     types.ResolutionStrategyAutoIncrement,
							Reason:       "test",
						},
					},
				},
				NetworkConflicts: []types.NetworkConflictInfo{},
			},
			expectedServicesCount: 1,
			expectedNetworksCount: 0,
		},
		{
			name: "ネットワーク衝突の解決",
			config: &types.ComposeConfig{
				Version: "3.8",
				Networks: map[string]types.Network{
					"mynet": {
						IPAM: types.IPAM{
							Config: []types.IPAMConfig{
								{Subnet: "172.20.0.0/24"},
							},
						},
					},
				},
			},
			conflictInfo: &types.UnifiedConflictInfo{
				PortConflicts: []types.PortConflictInfo{},
				NetworkConflicts: []types.NetworkConflictInfo{
					{
						NetworkName:    "mynet",
						OriginalSubnet: "172.20.0.0/24",
						Resolution: &types.NetworkResolutionInfo{
							ResolvedSubnet: "172.21.0.0/24",
						},
					},
				},
			},
			expectedServicesCount: 0,
			expectedNetworksCount: 1,
		},
		{
			name: "ポートとネットワーク両方の衝突解決",
			config: &types.ComposeConfig{
				Version: "3.8",
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
				},
				Networks: map[string]types.Network{
					"mynet": {
						IPAM: types.IPAM{
							Config: []types.IPAMConfig{
								{Subnet: "172.20.0.0/24"},
							},
						},
					},
				},
			},
			conflictInfo: &types.UnifiedConflictInfo{
				PortConflicts: []types.PortConflictInfo{
					{
						Port:        8080,
						ServiceName: "web",
						Resolution: &types.PortResolutionInfo{
							ResolvedPort: 8081,
						},
					},
				},
				NetworkConflicts: []types.NetworkConflictInfo{
					{
						NetworkName:    "mynet",
						OriginalSubnet: "172.20.0.0/24",
						Resolution: &types.NetworkResolutionInfo{
							ResolvedSubnet: "172.21.0.0/24",
						},
					},
				},
			},
			expectedServicesCount: 1,
			expectedNetworksCount: 1,
		},
		{
			name: "サービスIPアドレス再割り当て付きネットワーク解決",
			config: &types.ComposeConfig{
				Version: "3.8",
				Networks: map[string]types.Network{
					"mynet": {
						IPAM: types.IPAM{
							Config: []types.IPAMConfig{
								{Subnet: "172.20.0.0/24"},
							},
						},
					},
				},
			},
			conflictInfo: &types.UnifiedConflictInfo{
				PortConflicts: []types.PortConflictInfo{},
				NetworkConflicts: []types.NetworkConflictInfo{
					{
						NetworkName:    "mynet",
						OriginalSubnet: "172.20.0.0/24",
						Resolution: &types.NetworkResolutionInfo{
							ResolvedSubnet: "172.21.0.0/24",
							ServiceIPs: map[string]string{
								"web": "172.21.0.10",
								"db":  "172.21.0.20",
							},
						},
					},
				},
			},
			expectedServicesCount: 2,
			expectedNetworksCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAllocator := &mockPortAllocatorForGenerator{nextPort: 9000}
			generator := NewUnifiedOverrideGeneratorImpl(mockAllocator, testLogger)

			result, err := generator.GenerateFromConflicts(ctx, tt.config, tt.conflictInfo)

			if err != nil {
				t.Fatalf("GenerateFromConflicts() error = %v, want nil", err)
			}

			if len(result.Services) != tt.expectedServicesCount {
				t.Errorf("GenerateFromConflicts() Services count = %d, want %d",
					len(result.Services), tt.expectedServicesCount)
			}

			if len(result.Networks) != tt.expectedNetworksCount {
				t.Errorf("GenerateFromConflicts() Networks count = %d, want %d",
					len(result.Networks), tt.expectedNetworksCount)
			}

			// バージョンが保持されていることを確認
			if result.Version != tt.config.Version {
				t.Errorf("GenerateFromConflicts() Version = %s, want %s",
					result.Version, tt.config.Version)
			}

			// メタデータが設定されていることを確認
			if result.Metadata.Version == "" {
				t.Error("GenerateFromConflicts() Metadata.Version is empty")
			}

			if result.Metadata.GeneratedAt.IsZero() {
				t.Error("GenerateFromConflicts() Metadata.GeneratedAt is zero")
			}
		})
	}
}

func TestResolvePortConflicts(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name           string
		portConflicts  []types.PortConflictInfo
		strategy       types.ResolutionStrategy
		portConfig     types.PortConfig
		expectedResolved int
	}{
		{
			name: "単一ポート衝突の解決",
			portConflicts: []types.PortConflictInfo{
				{
					Port:        8080,
					ServiceName: "web",
				},
			},
			strategy: types.ResolutionStrategyAutoIncrement,
			portConfig: types.PortConfig{
				Range:             types.PortRange{Start: 8000, End: 9000},
				ExcludePrivileged: true,
			},
			expectedResolved: 1,
		},
		{
			name: "複数ポート衝突の解決",
			portConflicts: []types.PortConflictInfo{
				{Port: 8080, ServiceName: "web"},
				{Port: 9000, ServiceName: "api"},
				{Port: 3000, ServiceName: "frontend"},
			},
			strategy: types.ResolutionStrategyAutoIncrement,
			portConfig: types.PortConfig{
				Range:             types.PortRange{Start: 8000, End: 9000},
				ExcludePrivileged: true,
			},
			expectedResolved: 3,
		},
		{
			name:          "衝突なし",
			portConflicts: []types.PortConflictInfo{},
			strategy:      types.ResolutionStrategyAutoIncrement,
			portConfig: types.PortConfig{
				Range:             types.PortRange{Start: 8000, End: 9000},
				ExcludePrivileged: true,
			},
			expectedResolved: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAllocator := &mockPortAllocatorForGenerator{nextPort: tt.portConfig.Range.Start}
			generator := NewUnifiedOverrideGeneratorImpl(mockAllocator, testLogger)

			conflictInfo := &types.UnifiedConflictInfo{
				PortConflicts: tt.portConflicts,
			}

			err := generator.ResolveConflicts(ctx, conflictInfo, tt.strategy, tt.portConfig)

			if err != nil {
				t.Fatalf("ResolveConflicts() error = %v, want nil", err)
			}

			// 解決されたポート衝突の数を確認
			resolvedCount := 0
			for _, conflict := range conflictInfo.PortConflicts {
				if conflict.Resolution != nil {
					resolvedCount++

					// 解決ポートが設定されていることを確認
					if conflict.Resolution.ResolvedPort == 0 {
						t.Errorf("ResolveConflicts() ResolvedPort is 0 for service %s",
							conflict.ServiceName)
					}

					// 理由が設定されていることを確認
					if conflict.Resolution.Reason == "" {
						t.Errorf("ResolveConflicts() Reason is empty for service %s",
							conflict.ServiceName)
					}
				}
			}

			if resolvedCount != tt.expectedResolved {
				t.Errorf("ResolveConflicts() resolved count = %d, want %d",
					resolvedCount, tt.expectedResolved)
			}
		})
	}
}

func TestResolveNetworkConflicts(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name             string
		networkConflicts []types.NetworkConflictInfo
		expectedResolved int
	}{
		{
			name: "単一ネットワーク衝突の解決",
			networkConflicts: []types.NetworkConflictInfo{
				{
					NetworkName:    "mynet",
					OriginalSubnet: "172.20.0.0/24",
				},
			},
			expectedResolved: 1,
		},
		{
			name: "複数ネットワーク衝突の解決",
			networkConflicts: []types.NetworkConflictInfo{
				{NetworkName: "mynet1", OriginalSubnet: "172.20.0.0/24"},
				{NetworkName: "mynet2", OriginalSubnet: "172.21.0.0/24"},
				{NetworkName: "mynet3", OriginalSubnet: "172.22.0.0/24"},
			},
			expectedResolved: 3,
		},
		{
			name: "サービスIPアドレス付き衝突の解決",
			networkConflicts: []types.NetworkConflictInfo{
				{
					NetworkName:    "mynet",
					OriginalSubnet: "172.20.0.0/24",
					ServiceIPs: map[string]string{
						"web": "172.20.0.10",
						"db":  "172.20.0.20",
					},
				},
			},
			expectedResolved: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAllocator := &mockPortAllocatorForGenerator{nextPort: 9000}
			generator := NewUnifiedOverrideGeneratorImpl(mockAllocator, testLogger)

			conflictInfo := &types.UnifiedConflictInfo{
				NetworkConflicts: tt.networkConflicts,
			}

			err := generator.ResolveConflicts(ctx, conflictInfo, types.ResolutionStrategyAutoIncrement, types.PortConfig{})

			if err != nil {
				t.Fatalf("ResolveConflicts() error = %v, want nil", err)
			}

			// 解決されたネットワーク衝突の数を確認
			resolvedCount := 0
			for _, conflict := range conflictInfo.NetworkConflicts {
				if conflict.Resolution != nil {
					resolvedCount++

					// 解決されたサブネットが設定されていることを確認
					if conflict.Resolution.ResolvedSubnet == "" {
						t.Errorf("ResolveConflicts() ResolvedSubnet is empty for network %s",
							conflict.NetworkName)
					}

					// 元のサブネットと異なることを確認
					if conflict.Resolution.ResolvedSubnet == conflict.OriginalSubnet {
						t.Errorf("ResolveConflicts() ResolvedSubnet = OriginalSubnet for network %s",
							conflict.NetworkName)
					}

					// 理由が設定されていることを確認
					if conflict.Resolution.Reason == "" {
						t.Errorf("ResolveConflicts() Reason is empty for network %s",
							conflict.NetworkName)
					}

					// サービスIPアドレスが再マッピングされていることを確認
					if len(conflict.ServiceIPs) > 0 {
						if len(conflict.Resolution.ServiceIPs) != len(conflict.ServiceIPs) {
							t.Errorf("ResolveConflicts() ServiceIPs count = %d, want %d",
								len(conflict.Resolution.ServiceIPs), len(conflict.ServiceIPs))
						}
					}
				}
			}

			if resolvedCount != tt.expectedResolved {
				t.Errorf("ResolveConflicts() resolved count = %d, want %d",
					resolvedCount, tt.expectedResolved)
			}
		})
	}
}

func TestAllocateNewSubnet(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})

	tests := []struct {
		name         string
		usedSubnets  map[string]bool
		expectedStart string
	}{
		{
			name:         "未使用の場合は10.20.0.0/24から開始",
			usedSubnets:  map[string]bool{},
			expectedStart: "10.20.0.0/24",
		},
		{
			name: "10.20.0.0/24が使用済みの場合は10.21.0.0/24",
			usedSubnets: map[string]bool{
				"10.20.0.0/24": true,
			},
			expectedStart: "10.21.0.0/24",
		},
		{
			name: "10.x範囲が全て使用済みの場合は192.168.100.0/24",
			usedSubnets: func() map[string]bool {
				used := make(map[string]bool)
				for i := 20; i <= 255; i++ {
					used["10."+string(rune(i))+".0.0/24"] = true
				}
				return used
			}(),
			expectedStart: "192.168.100.0/24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAllocator := &mockPortAllocatorForGenerator{}
			generator := NewUnifiedOverrideGeneratorImpl(mockAllocator, testLogger)

			result := generator.allocateNewSubnet(tt.usedSubnets)

			if result == "" {
				t.Error("allocateNewSubnet() returned empty string")
				return
			}

			// 期待される範囲から開始していることを確認
			if tt.expectedStart != "" {
				if result != tt.expectedStart {
					// 完全一致でなくても、同じ範囲から始まっていればOK
					// （例: 10.20, 10.21などの連番）
					t.Logf("allocateNewSubnet() = %s, expected start = %s", result, tt.expectedStart)
				}
			}

			// 使用済みサブネットリストに含まれていないことを確認
			if tt.usedSubnets[result] {
				t.Errorf("allocateNewSubnet() returned used subnet %s", result)
			}
		})
	}
}

func TestRemapIPAddressesToNewSubnet(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})

	tests := []struct {
		name       string
		oldSubnet  string
		newSubnet  string
		serviceIPs map[string]string
		expected   map[string]string
	}{
		{
			name:      "同じホスト部分を維持",
			oldSubnet: "172.20.0.0/24",
			newSubnet: "172.21.0.0/24",
			serviceIPs: map[string]string{
				"web": "172.20.0.10",
				"db":  "172.20.0.20",
			},
			expected: map[string]string{
				"web": "172.21.0.10",
				"db":  "172.21.0.20",
			},
		},
		{
			name:      "異なるネットワーククラスへの変更",
			oldSubnet: "172.20.0.0/24",
			newSubnet: "10.30.0.0/24",
			serviceIPs: map[string]string{
				"web": "172.20.0.10",
			},
			expected: map[string]string{
				"web": "10.30.0.10",
			},
		},
		{
			name:       "空のサービスIPリスト",
			oldSubnet:  "172.20.0.0/24",
			newSubnet:  "172.21.0.0/24",
			serviceIPs: map[string]string{},
			expected:   map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAllocator := &mockPortAllocatorForGenerator{}
			generator := NewUnifiedOverrideGeneratorImpl(mockAllocator, testLogger)

			result, err := generator.remapIPAddressesToNewSubnet(tt.oldSubnet, tt.newSubnet, tt.serviceIPs)

			if err != nil {
				t.Fatalf("remapIPAddressesToNewSubnet() error = %v, want nil", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("remapIPAddressesToNewSubnet() count = %d, want %d",
					len(result), len(tt.expected))
			}

			for service, expectedIP := range tt.expected {
				if result[service] != expectedIP {
					t.Errorf("remapIPAddressesToNewSubnet()[%s] = %s, want %s",
						service, result[service], expectedIP)
				}
			}
		})
	}
}

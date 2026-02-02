package scanner

import (
	"context"
	"testing"
	"time"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

// mockNetworkDetector はテスト用のモックネットワーク検出器
type mockNetworkDetector struct {
	networks []NetworkInfo
	err      error
}

func (m *mockNetworkDetector) DetectNetworks(ctx context.Context) ([]NetworkInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.networks, nil
}

func TestDetectConflicts(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name                  string
		usedPorts             []int
		networks              []NetworkInfo
		config                *types.ComposeConfig
		projectName           string
		expectedPortConflicts int
		expectedNetConflicts  int
	}{
		{
			name:      "ポート衝突のみ検出",
			usedPorts: []int{8080, 9000},
			networks:  []NetworkInfo{},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Name: "web",
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
					"api": {
						Name: "api",
						Ports: []types.PortMapping{
							{Host: 9000, Container: 3000, Protocol: "tcp"},
						},
					},
				},
				Networks: map[string]types.Network{},
			},
			projectName:           "",
			expectedPortConflicts: 2,
			expectedNetConflicts:  0,
		},
		{
			name:      "ネットワーク衝突のみ検出",
			usedPorts: []int{},
			networks: []NetworkInfo{
				{
					Name:    "existing_net",
					Subnets: []string{"172.20.0.0/24"},
				},
			},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{},
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
			projectName:           "",
			expectedPortConflicts: 0,
			expectedNetConflicts:  1,
		},
		{
			name:      "ポートとネットワーク両方の衝突検出",
			usedPorts: []int{8080},
			networks: []NetworkInfo{
				{
					Name:    "existing_net",
					Subnets: []string{"172.20.0.0/24"},
				},
			},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Name: "web",
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
			projectName:           "",
			expectedPortConflicts: 1,
			expectedNetConflicts:  1,
		},
		{
			name:      "衝突なし",
			usedPorts: []int{3000, 4000},
			networks: []NetworkInfo{
				{
					Name:    "other_net",
					Subnets: []string{"172.30.0.0/24"},
				},
			},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Name: "web",
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
			projectName:           "",
			expectedPortConflicts: 0,
			expectedNetConflicts:  0,
		},
		{
			name:      "プロジェクト名プレフィックス付きネットワーク名の衝突",
			usedPorts: []int{},
			networks: []NetworkInfo{
				{
					Name:    "myproject_mynet",
					Subnets: []string{"172.21.0.0/24"},
				},
			},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{},
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
			projectName:           "myproject",
			expectedPortConflicts: 0,
			expectedNetConflicts:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPortDetector := &mockPortDetector{usedPorts: tt.usedPorts}
			mockNetDetector := &mockNetworkDetector{networks: tt.networks}
			detector := NewUnifiedConflictDetectorImpl(mockPortDetector, mockNetDetector, testLogger)

			result, err := detector.DetectConflicts(ctx, tt.config, tt.projectName)

			if err != nil {
				t.Fatalf("DetectConflicts() error = %v, want nil", err)
			}

			if len(result.PortConflicts) != tt.expectedPortConflicts {
				t.Errorf("DetectConflicts() PortConflicts count = %d, want %d",
					len(result.PortConflicts), tt.expectedPortConflicts)
			}

			if len(result.NetworkConflicts) != tt.expectedNetConflicts {
				t.Errorf("DetectConflicts() NetworkConflicts count = %d, want %d",
					len(result.NetworkConflicts), tt.expectedNetConflicts)
			}

			// GeneratedAt が設定されていることを確認
			if result.GeneratedAt.IsZero() {
				t.Error("DetectConflicts() GeneratedAt is zero, want non-zero")
			}

			// 現在時刻との差が1秒未満であることを確認（処理時間を考慮）
			if time.Since(result.GeneratedAt) > time.Second {
				t.Errorf("DetectConflicts() GeneratedAt = %v, too old", result.GeneratedAt)
			}
		})
	}
}

func TestDetectPortConflicts(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name              string
		usedPorts         []int
		config            *types.ComposeConfig
		expectedConflicts int
		expectedTypes     []types.ConflictType
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
			expectedConflicts: 1,
			expectedTypes:     []types.ConflictType{types.ConflictTypeSystem},
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
			expectedConflicts: 1,
			expectedTypes:     []types.ConflictType{types.ConflictTypeCompose},
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
			expectedConflicts: 2,
			// システムポート衝突が優先されるため、両方ともシステム衝突として検出される
			expectedTypes:     []types.ConflictType{types.ConflictTypeSystem, types.ConflictTypeSystem},
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
			expectedConflicts: 0,
			expectedTypes:     []types.ConflictType{},
		},
		{
			name:      "複数サービスで異なるポートを使用",
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
			expectedConflicts: 0,
			expectedTypes:     []types.ConflictType{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPortDetector := &mockPortDetector{usedPorts: tt.usedPorts}
			mockNetDetector := &mockNetworkDetector{}
			detector := NewUnifiedConflictDetectorImpl(mockPortDetector, mockNetDetector, testLogger)

			conflicts, err := detector.DetectPortConflicts(ctx, tt.config)

			if err != nil {
				t.Fatalf("DetectPortConflicts() error = %v, want nil", err)
			}

			if len(conflicts) != tt.expectedConflicts {
				t.Errorf("DetectPortConflicts() conflicts count = %d, want %d",
					len(conflicts), tt.expectedConflicts)
			}

			// 衝突タイプの確認
			for i, conflict := range conflicts {
				if i < len(tt.expectedTypes) {
					if conflict.Type != tt.expectedTypes[i] {
						t.Errorf("DetectPortConflicts() conflict[%d].Type = %v, want %v",
							i, conflict.Type, tt.expectedTypes[i])
					}
				}
			}

			// 各衝突に説明が含まれることを確認
			for _, conflict := range conflicts {
				if conflict.Description == "" {
					t.Errorf("DetectPortConflicts() conflict description is empty")
				}
			}
		})
	}
}

func TestDetectNetworkConflicts(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name              string
		networks          []NetworkInfo
		config            *types.ComposeConfig
		projectName       string
		expectedConflicts int
		expectedTypes     []types.NetworkConflictType
	}{
		{
			name: "サブネット衝突",
			networks: []NetworkInfo{
				{
					Name:    "existing_net",
					Subnets: []string{"172.20.0.0/24"},
				},
			},
			config: &types.ComposeConfig{
				Networks: map[string]types.Network{
					"mynet": {
						IPAM: types.IPAM{
							Config: []types.IPAMConfig{
								{Subnet: "172.20.0.0/24"},
							},
						},
					},
				},
				Services: map[string]types.Service{},
			},
			projectName:       "",
			expectedConflicts: 1,
			expectedTypes:     []types.NetworkConflictType{types.NetworkConflictTypeSubnet},
		},
		{
			name: "ネットワーク名衝突",
			networks: []NetworkInfo{
				{
					Name:    "mynet",
					Subnets: []string{"172.21.0.0/24"},
				},
			},
			config: &types.ComposeConfig{
				Networks: map[string]types.Network{
					"mynet": {
						IPAM: types.IPAM{
							Config: []types.IPAMConfig{
								{Subnet: "172.20.0.0/24"},
							},
						},
					},
				},
				Services: map[string]types.Service{},
			},
			projectName:       "",
			expectedConflicts: 1,
			expectedTypes:     []types.NetworkConflictType{types.NetworkConflictTypeName},
		},
		{
			name: "プロジェクト名プレフィックス付きネットワーク名衝突",
			networks: []NetworkInfo{
				{
					Name:    "myproject_mynet",
					Subnets: []string{"172.21.0.0/24"},
				},
			},
			config: &types.ComposeConfig{
				Networks: map[string]types.Network{
					"mynet": {
						IPAM: types.IPAM{
							Config: []types.IPAMConfig{
								{Subnet: "172.20.0.0/24"},
							},
						},
					},
				},
				Services: map[string]types.Service{},
			},
			projectName:       "myproject",
			expectedConflicts: 1,
			expectedTypes:     []types.NetworkConflictType{types.NetworkConflictTypeName},
		},
		{
			name: "IPAMConfig未設定（衝突なし）",
			networks: []NetworkInfo{
				{
					Name:    "existing_net",
					Subnets: []string{"172.20.0.0/24"},
				},
			},
			config: &types.ComposeConfig{
				Networks: map[string]types.Network{
					"mynet": {
						IPAM: types.IPAM{
							Config: []types.IPAMConfig{},
						},
					},
				},
				Services: map[string]types.Service{},
			},
			projectName:       "",
			expectedConflicts: 0,
			expectedTypes:     []types.NetworkConflictType{},
		},
		{
			name: "サブネット空文字列（衝突なし）",
			networks: []NetworkInfo{
				{
					Name:    "existing_net",
					Subnets: []string{"172.20.0.0/24"},
				},
			},
			config: &types.ComposeConfig{
				Networks: map[string]types.Network{
					"mynet": {
						IPAM: types.IPAM{
							Config: []types.IPAMConfig{
								{Subnet: ""},
							},
						},
					},
				},
				Services: map[string]types.Service{},
			},
			projectName:       "",
			expectedConflicts: 0,
			expectedTypes:     []types.NetworkConflictType{},
		},
		{
			name: "サービスIPアドレス付きサブネット衝突",
			networks: []NetworkInfo{
				{
					Name:    "existing_net",
					Subnets: []string{"172.20.0.0/24"},
				},
			},
			config: &types.ComposeConfig{
				Networks: map[string]types.Network{
					"mynet": {
						IPAM: types.IPAM{
							Config: []types.IPAMConfig{
								{Subnet: "172.20.0.0/24"},
							},
						},
					},
				},
				Services: map[string]types.Service{
					"web": {
						Networks: map[string]types.ServiceNetwork{
							"mynet": {
								IPv4Address: "172.20.0.10",
							},
						},
					},
					"db": {
						Networks: map[string]types.ServiceNetwork{
							"mynet": {
								IPv4Address: "172.20.0.20",
							},
						},
					},
				},
			},
			projectName:       "",
			expectedConflicts: 1,
			expectedTypes:     []types.NetworkConflictType{types.NetworkConflictTypeSubnet},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPortDetector := &mockPortDetector{}
			mockNetDetector := &mockNetworkDetector{networks: tt.networks}
			detector := NewUnifiedConflictDetectorImpl(mockPortDetector, mockNetDetector, testLogger)

			conflicts, err := detector.DetectNetworkConflicts(ctx, tt.config, tt.projectName)

			if err != nil {
				t.Fatalf("DetectNetworkConflicts() error = %v, want nil", err)
			}

			if len(conflicts) != tt.expectedConflicts {
				t.Errorf("DetectNetworkConflicts() conflicts count = %d, want %d",
					len(conflicts), tt.expectedConflicts)
			}

			// 衝突タイプの確認
			for i, conflict := range conflicts {
				if i < len(tt.expectedTypes) {
					if conflict.ConflictType != tt.expectedTypes[i] {
						t.Errorf("DetectNetworkConflicts() conflict[%d].ConflictType = %v, want %v",
							i, conflict.ConflictType, tt.expectedTypes[i])
					}
				}
			}

			// サービスIPアドレスの確認（サブネット衝突の場合）
			if tt.name == "サービスIPアドレス付きサブネット衝突" {
				if len(conflicts) > 0 {
					conflict := conflicts[0]
					if len(conflict.ServiceIPs) != 2 {
						t.Errorf("DetectNetworkConflicts() ServiceIPs count = %d, want 2",
							len(conflict.ServiceIPs))
					}
				}
			}
		})
	}
}

func TestGetServiceNetworkIPs(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})

	tests := []struct {
		name         string
		config       *types.ComposeConfig
		networkName  string
		expectedIPs  map[string]string
	}{
		{
			name: "複数サービスのIPアドレスを取得",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Networks: map[string]types.ServiceNetwork{
							"mynet": {
								IPv4Address: "172.20.0.10",
							},
						},
					},
					"db": {
						Networks: map[string]types.ServiceNetwork{
							"mynet": {
								IPv4Address: "172.20.0.20",
							},
						},
					},
				},
			},
			networkName: "mynet",
			expectedIPs: map[string]string{
				"web": "172.20.0.10",
				"db":  "172.20.0.20",
			},
		},
		{
			name: "ネットワーク名が一致しない場合は空",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Networks: map[string]types.ServiceNetwork{
							"other_net": {
								IPv4Address: "172.20.0.10",
							},
						},
					},
				},
			},
			networkName: "mynet",
			expectedIPs: map[string]string{},
		},
		{
			name: "IPv4Address未設定の場合はスキップ",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Networks: map[string]types.ServiceNetwork{
							"mynet": {
								IPv4Address: "",
							},
						},
					},
				},
			},
			networkName: "mynet",
			expectedIPs: map[string]string{},
		},
		{
			name: "Networksがnilの場合は空",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Networks: nil,
					},
				},
			},
			networkName: "mynet",
			expectedIPs: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPortDetector := &mockPortDetector{}
			mockNetDetector := &mockNetworkDetector{}
			detector := NewUnifiedConflictDetectorImpl(mockPortDetector, mockNetDetector, testLogger)

			result := detector.getServiceNetworkIPs(tt.config, tt.networkName)

			if len(result) != len(tt.expectedIPs) {
				t.Errorf("getServiceNetworkIPs() count = %d, want %d",
					len(result), len(tt.expectedIPs))
			}

			for service, expectedIP := range tt.expectedIPs {
				if result[service] != expectedIP {
					t.Errorf("getServiceNetworkIPs()[%s] = %s, want %s",
						service, result[service], expectedIP)
				}
			}
		})
	}
}

package scanner

import (
	"context"
	"testing"
	"time"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

// mockNetworkDetector is a mock network detector for testing
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
			name:      "detect port conflicts only",
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
			name:      "detect network conflicts only",
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
			name:      "detect both port and network conflicts",
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
			name:      "no conflicts",
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
			name:      "network name conflict with project name prefix",
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

			// verify GeneratedAt is set
			if result.GeneratedAt.IsZero() {
				t.Error("DetectConflicts() GeneratedAt is zero, want non-zero")
			}

			// verify GeneratedAt is within 1 second (accounting for processing time)
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
			expectedConflicts: 1,
			expectedTypes:     []types.ConflictType{types.ConflictTypeSystem},
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
			expectedConflicts: 1,
			expectedTypes:     []types.ConflictType{types.ConflictTypeCompose},
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
			expectedConflicts: 2,
			// system port conflict takes priority, both detected as system conflict
			expectedTypes:     []types.ConflictType{types.ConflictTypeSystem, types.ConflictTypeSystem},
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
			expectedConflicts: 0,
			expectedTypes:     []types.ConflictType{},
		},
		{
			name:      "multiple services with different ports",
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

			// verify conflict type
			for i, conflict := range conflicts {
				if i < len(tt.expectedTypes) {
					if conflict.Type != tt.expectedTypes[i] {
						t.Errorf("DetectPortConflicts() conflict[%d].Type = %v, want %v",
							i, conflict.Type, tt.expectedTypes[i])
					}
				}
			}

			// verify each conflict has description
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
			name: "subnet conflict",
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
			name: "network name conflict",
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
			name: "network name conflict with project prefix",
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
			name: "no IPAM config (no conflict)",
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
			name: "empty subnet string (no conflict)",
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
			name: "subnet conflict with service IP addresses",
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

			// verify conflict type
			for i, conflict := range conflicts {
				if i < len(tt.expectedTypes) {
					if conflict.ConflictType != tt.expectedTypes[i] {
						t.Errorf("DetectNetworkConflicts() conflict[%d].ConflictType = %v, want %v",
							i, conflict.ConflictType, tt.expectedTypes[i])
					}
				}
			}

			// verify service IP addresses (for subnet conflict)
			if tt.name == "subnet conflict with service IP addresses" {
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
			name: "get IP addresses for multiple services",
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
			name: "empty when network name does not match",
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
			name: "skip when IPv4Address is not set",
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
			name: "empty when Networks is nil",
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

func TestNewUnifiedConflictDetectorImpl(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	mockPortDetector := &mockPortDetector{}
	mockNetDetector := &mockNetworkDetector{}

	detector := NewUnifiedConflictDetectorImpl(mockPortDetector, mockNetDetector, testLogger)

	if detector == nil {
		t.Fatal("NewUnifiedConflictDetectorImpl() returned nil")
	}

	if detector.portDetector == nil {
		t.Error("portDetector is nil")
	}

	if detector.networkDetector == nil {
		t.Error("networkDetector is nil")
	}

	if detector.logger == nil {
		t.Error("logger is nil")
	}
}

func TestDetectPortConflictsEdgeCases(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name              string
		usedPorts         []int
		config            *types.ComposeConfig
		expectedConflicts int
	}{
		{
			name:      "no services",
			usedPorts: []int{8080},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{},
			},
			expectedConflicts: 0,
		},
		{
			name:      "no port mappings",
			usedPorts: []int{8080},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{},
					},
				},
			},
			expectedConflicts: 0,
		},
		{
			name:      "all host ports are 0",
			usedPorts: []int{8080},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{
							{Host: 0, Container: 80},
						},
					},
				},
			},
			expectedConflicts: 0,
		},
		{
			name:      "multiple port mappings in same service",
			usedPorts: []int{8080, 8443},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80},
							{Host: 8443, Container: 443},
						},
					},
				},
			},
			expectedConflicts: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPortDetector := &mockPortDetector{usedPorts: tt.usedPorts}
			mockNetDetector := &mockNetworkDetector{}
			detector := NewUnifiedConflictDetectorImpl(mockPortDetector, mockNetDetector, testLogger)

			conflicts, err := detector.DetectPortConflicts(ctx, tt.config)

			if err != nil {
				t.Fatalf("DetectPortConflicts() unexpected error: %v", err)
			}

			if len(conflicts) != tt.expectedConflicts {
				t.Errorf("DetectPortConflicts() conflicts count = %d, want %d",
					len(conflicts), tt.expectedConflicts)
			}
		})
	}
}

func TestDetectNetworkConflictsEdgeCases(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name              string
		networks          []NetworkInfo
		config            *types.ComposeConfig
		projectName       string
		expectedConflicts int
	}{
		{
			name:     "no networks",
			networks: []NetworkInfo{},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{},
				Networks: map[string]types.Network{},
			},
			expectedConflicts: 0,
		},
		{
			name: "no network definitions in compose",
			networks: []NetworkInfo{
				{Name: "existing", Subnets: []string{"172.20.0.0/24"}},
			},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{},
				Networks: map[string]types.Network{},
			},
			expectedConflicts: 0,
		},
		{
			name: "no IPAM config",
			networks: []NetworkInfo{
				{Name: "existing", Subnets: []string{"172.20.0.0/24"}},
			},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{},
				Networks: map[string]types.Network{
					"mynet": {
						IPAM: types.IPAM{
							Config: []types.IPAMConfig{},
						},
					},
				},
			},
			expectedConflicts: 0,
		},
		{
			name: "multiple network conflicts",
			networks: []NetworkInfo{
				{Name: "net1", Subnets: []string{"172.20.0.0/24"}},
				{Name: "net2", Subnets: []string{"172.21.0.0/24"}},
			},
			config: &types.ComposeConfig{
				Services: map[string]types.Service{},
				Networks: map[string]types.Network{
					"mynet1": {
						IPAM: types.IPAM{
							Config: []types.IPAMConfig{
								{Subnet: "172.20.0.0/24"},
							},
						},
					},
					"mynet2": {
						IPAM: types.IPAM{
							Config: []types.IPAMConfig{
								{Subnet: "172.21.0.0/24"},
							},
						},
					},
				},
			},
			expectedConflicts: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPortDetector := &mockPortDetector{}
			mockNetDetector := &mockNetworkDetector{networks: tt.networks}
			detector := NewUnifiedConflictDetectorImpl(mockPortDetector, mockNetDetector, testLogger)

			conflicts, err := detector.DetectNetworkConflicts(ctx, tt.config, tt.projectName)

			if err != nil {
				t.Fatalf("DetectNetworkConflicts() unexpected error: %v", err)
			}

			if len(conflicts) != tt.expectedConflicts {
				t.Errorf("DetectNetworkConflicts() conflicts count = %d, want %d",
					len(conflicts), tt.expectedConflicts)
			}
		})
	}
}

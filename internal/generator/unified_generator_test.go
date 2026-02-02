package generator

import (
	"context"
	"fmt"
	"testing"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/internal/testutil"
	"github.com/harakeishi/gopose/pkg/types"
)

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
			name: "port conflict resolution",
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
			name: "network conflict resolution",
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
			name: "both port and network conflict resolution",
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
			name: "network resolution with service IP reassignment",
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
			mockAllocator := &testutil.MockPortAllocator{NextPort: 9000}
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

			// verify version is preserved
			if result.Version != tt.config.Version {
				t.Errorf("GenerateFromConflicts() Version = %s, want %s",
					result.Version, tt.config.Version)
			}

			// verify metadata is set
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
			name: "single port conflict resolution",
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
			name: "multiple port conflict resolution",
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
			name:          "no conflicts",
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
			mockAllocator := &testutil.MockPortAllocator{NextPort: tt.portConfig.Range.Start}
			generator := NewUnifiedOverrideGeneratorImpl(mockAllocator, testLogger)

			conflictInfo := &types.UnifiedConflictInfo{
				PortConflicts: tt.portConflicts,
			}

			err := generator.ResolveConflicts(ctx, conflictInfo, tt.strategy, tt.portConfig)

			if err != nil {
				t.Fatalf("ResolveConflicts() error = %v, want nil", err)
			}

			// verify resolved port conflict count
			resolvedCount := 0
			for _, conflict := range conflictInfo.PortConflicts {
				if conflict.Resolution != nil {
					resolvedCount++

					// verify resolved port is set
					if conflict.Resolution.ResolvedPort == 0 {
						t.Errorf("ResolveConflicts() ResolvedPort is 0 for service %s",
							conflict.ServiceName)
					}

					// verify reason is set
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
			name: "single network conflict resolution",
			networkConflicts: []types.NetworkConflictInfo{
				{
					NetworkName:    "mynet",
					OriginalSubnet: "172.20.0.0/24",
				},
			},
			expectedResolved: 1,
		},
		{
			name: "multiple network conflict resolution",
			networkConflicts: []types.NetworkConflictInfo{
				{NetworkName: "mynet1", OriginalSubnet: "172.20.0.0/24"},
				{NetworkName: "mynet2", OriginalSubnet: "172.21.0.0/24"},
				{NetworkName: "mynet3", OriginalSubnet: "172.22.0.0/24"},
			},
			expectedResolved: 3,
		},
		{
			name: "conflict resolution with service IP addresses",
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
			mockAllocator := &testutil.MockPortAllocator{NextPort: 9000}
			generator := NewUnifiedOverrideGeneratorImpl(mockAllocator, testLogger)

			conflictInfo := &types.UnifiedConflictInfo{
				NetworkConflicts: tt.networkConflicts,
			}

			err := generator.ResolveConflicts(ctx, conflictInfo, types.ResolutionStrategyAutoIncrement, types.PortConfig{})

			if err != nil {
				t.Fatalf("ResolveConflicts() error = %v, want nil", err)
			}

			// verify resolved network conflict count
			resolvedCount := 0
			for _, conflict := range conflictInfo.NetworkConflicts {
				if conflict.Resolution != nil {
					resolvedCount++

					// verify resolved subnet is set
					if conflict.Resolution.ResolvedSubnet == "" {
						t.Errorf("ResolveConflicts() ResolvedSubnet is empty for network %s",
							conflict.NetworkName)
					}

					// verify differs from original subnet
					if conflict.Resolution.ResolvedSubnet == conflict.OriginalSubnet {
						t.Errorf("ResolveConflicts() ResolvedSubnet = OriginalSubnet for network %s",
							conflict.NetworkName)
					}

					// verify reason is set
					if conflict.Resolution.Reason == "" {
						t.Errorf("ResolveConflicts() Reason is empty for network %s",
							conflict.NetworkName)
					}

					// verify service IP addresses are remapped
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
			name:         "starts from 10.20.0.0/24 when unused",
			usedSubnets:  map[string]bool{},
			expectedStart: "10.20.0.0/24",
		},
		{
			name: "10.21.0.0/24 when 10.20.0.0/24 is used",
			usedSubnets: map[string]bool{
				"10.20.0.0/24": true,
			},
			expectedStart: "10.21.0.0/24",
		},
		{
			name: "192.168.100.0/24 when all 10.x range is used",
			usedSubnets: func() map[string]bool {
				used := make(map[string]bool)
				for i := 20; i <= 255; i++ {
					used[fmt.Sprintf("10.%d.0.0/24", i)] = true
				}
				return used
			}(),
			expectedStart: "192.168.100.0/24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAllocator := &testutil.MockPortAllocator{}
			generator := NewUnifiedOverrideGeneratorImpl(mockAllocator, testLogger)

			result := generator.allocateNewSubnet(tt.usedSubnets)

			if result == "" {
				t.Error("allocateNewSubnet() returned empty string")
				return
			}

			// verify exact match with expected subnet
			if tt.expectedStart != "" && result != tt.expectedStart {
				t.Errorf("allocateNewSubnet() = %s, want %s", result, tt.expectedStart)
			}

			// verify result is not in used subnets list
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
			name:      "maintain same host part",
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
			name:      "change to different network class",
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
			name:       "empty service IP list",
			oldSubnet:  "172.20.0.0/24",
			newSubnet:  "172.21.0.0/24",
			serviceIPs: map[string]string{},
			expected:   map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAllocator := &testutil.MockPortAllocator{}
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

func TestGeneratePortOverrides(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name            string
		config          *types.ComposeConfig
		portConflicts   []types.PortConflictInfo
		expectedServices int
		expectedPortUpdated bool
	}{
		{
			name: "generate override for resolved port conflict",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			portConflicts: []types.PortConflictInfo{
				{
					ServiceName: "web",
					Port:        8080,
					Resolution: &types.PortResolutionInfo{
						ResolvedPort: 8081,
					},
				},
			},
			expectedServices: 1,
			expectedPortUpdated: true,
		},
		{
			name: "multi-service port conflicts",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80},
						},
					},
					"api": {
						Ports: []types.PortMapping{
							{Host: 9000, Container: 3000},
						},
					},
				},
			},
			portConflicts: []types.PortConflictInfo{
				{
					ServiceName: "web",
					Port:        8080,
					Resolution: &types.PortResolutionInfo{
						ResolvedPort: 8081,
					},
				},
				{
					ServiceName: "api",
					Port:        9000,
					Resolution: &types.PortResolutionInfo{
						ResolvedPort: 9001,
					},
				},
			},
			expectedServices: 2,
			expectedPortUpdated: true,
		},
		{
			name: "conflict without resolution (skipped)",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80},
						},
					},
				},
			},
			portConflicts: []types.PortConflictInfo{
				{
					ServiceName: "web",
					Port:        8080,
					Resolution:  nil,
				},
			},
			expectedServices: 1,
			expectedPortUpdated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAllocator := &testutil.MockPortAllocator{NextPort: 8000}
			generator := NewUnifiedOverrideGeneratorImpl(mockAllocator, testLogger)

			override := &types.OverrideConfig{
				Services: make(map[string]types.ServiceOverride),
			}

			err := generator.generatePortOverrides(ctx, tt.config, tt.portConflicts, override)

			if err != nil {
				t.Fatalf("generatePortOverrides() error = %v, want nil", err)
			}

			if len(override.Services) != tt.expectedServices {
				t.Errorf("len(override.Services) = %d, want %d",
					len(override.Services), tt.expectedServices)
			}

			if tt.expectedPortUpdated && len(override.Services) > 0 {
				for serviceName, serviceOverride := range override.Services {
					if len(serviceOverride.Ports) == 0 {
						t.Errorf("Service %s has no port overrides", serviceName)
					}
				}
			}
		})
	}
}

func TestGenerateNetworkOverrides(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name               string
		config             *types.ComposeConfig
		networkConflicts   []types.NetworkConflictInfo
		expectedNetworks   int
		expectedServices   int
	}{
		{
			name: "network subnet conflict resolution",
			config: &types.ComposeConfig{
				Networks: map[string]types.Network{
					"mynet": {},
				},
			},
			networkConflicts: []types.NetworkConflictInfo{
				{
					NetworkName:    "mynet",
					OriginalSubnet: "172.20.0.0/24",
					Resolution: &types.NetworkResolutionInfo{
						ResolvedSubnet: "10.20.0.0/24",
					},
				},
			},
			expectedNetworks: 1,
			expectedServices: 0,
		},
		{
			name: "network resolution with service IP reassignment",
			config: &types.ComposeConfig{
				Networks: map[string]types.Network{
					"mynet": {},
				},
			},
			networkConflicts: []types.NetworkConflictInfo{
				{
					NetworkName:    "mynet",
					OriginalSubnet: "172.20.0.0/24",
					Resolution: &types.NetworkResolutionInfo{
						ResolvedSubnet: "10.20.0.0/24",
						ServiceIPs: map[string]string{
							"web": "10.20.0.10",
							"db":  "10.20.0.20",
						},
					},
				},
			},
			expectedNetworks: 1,
			expectedServices: 2,
		},
		{
			name: "no resolution (skipped)",
			config: &types.ComposeConfig{
				Networks: map[string]types.Network{
					"mynet": {},
				},
			},
			networkConflicts: []types.NetworkConflictInfo{
				{
					NetworkName: "mynet",
					Resolution:  nil,
				},
			},
			expectedNetworks: 0,
			expectedServices: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAllocator := &testutil.MockPortAllocator{NextPort: 8000}
			generator := NewUnifiedOverrideGeneratorImpl(mockAllocator, testLogger)

			override := &types.OverrideConfig{
				Services: make(map[string]types.ServiceOverride),
				Networks: make(map[string]types.NetworkOverride),
			}

			err := generator.generateNetworkOverrides(ctx, tt.config, tt.networkConflicts, override)

			if err != nil {
				t.Fatalf("generateNetworkOverrides() error = %v, want nil", err)
			}

			if len(override.Networks) != tt.expectedNetworks {
				t.Errorf("len(override.Networks) = %d, want %d",
					len(override.Networks), tt.expectedNetworks)
			}

			if len(override.Services) != tt.expectedServices {
				t.Errorf("len(override.Services) = %d, want %d",
					len(override.Services), tt.expectedServices)
			}

			// verify network overrides
			for _, conflict := range tt.networkConflicts {
				if conflict.Resolution != nil {
					networkOverride, exists := override.Networks[conflict.NetworkName]
					if !exists {
						t.Errorf("Network override not found for %s", conflict.NetworkName)
						continue
					}

					if len(networkOverride.IPAM.Config) == 0 {
						t.Error("IPAM config is empty")
						continue
					}

					if networkOverride.IPAM.Config[0].Subnet != conflict.Resolution.ResolvedSubnet {
						t.Errorf("Subnet = %s, want %s",
							networkOverride.IPAM.Config[0].Subnet,
							conflict.Resolution.ResolvedSubnet)
					}
				}
			}
		})
	}
}

func TestPopulateMetadata(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	mockAllocator := &testutil.MockPortAllocator{NextPort: 8000}
	generator := NewUnifiedOverrideGeneratorImpl(mockAllocator, testLogger)

	tests := []struct {
		name                string
		conflictInfo        *types.UnifiedConflictInfo
		expectedResolutions int
	}{
		{
			name: "port conflict with resolution info",
			conflictInfo: &types.UnifiedConflictInfo{
				PortConflicts: []types.PortConflictInfo{
					{
						ServiceName: "web",
						Service:     "web",
						Port:        8080,
						Resolution: &types.PortResolutionInfo{
							ResolvedPort: 8081,
							Strategy:     types.ResolutionStrategyAutoIncrement,
							Reason:       "Test reason",
						},
					},
					{
						ServiceName: "api",
						Port:        9000,
						Resolution: &types.PortResolutionInfo{
							ResolvedPort: 9001,
							Strategy:     types.ResolutionStrategyAutoIncrement,
							Reason:       "Another reason",
						},
					},
				},
			},
			expectedResolutions: 2,
		},
		{
			name: "no resolution info (not added)",
			conflictInfo: &types.UnifiedConflictInfo{
				PortConflicts: []types.PortConflictInfo{
					{
						ServiceName: "web",
						Port:        8080,
						Resolution:  nil,
					},
				},
			},
			expectedResolutions: 0,
		},
		{
			name: "mixed (some with resolution info)",
			conflictInfo: &types.UnifiedConflictInfo{
				PortConflicts: []types.PortConflictInfo{
					{
						ServiceName: "web",
						Port:        8080,
						Resolution: &types.PortResolutionInfo{
							ResolvedPort: 8081,
							Strategy:     types.ResolutionStrategyAutoIncrement,
							Reason:       "Test",
						},
					},
					{
						ServiceName: "api",
						Port:        9000,
						Resolution:  nil,
					},
				},
			},
			expectedResolutions: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			override := &types.OverrideConfig{
				Metadata: types.OverrideMetadata{
					Resolutions: []types.ConflictResolution{},
				},
			}

			generator.populateMetadata(tt.conflictInfo, override)

			if len(override.Metadata.Resolutions) != tt.expectedResolutions {
				t.Errorf("len(override.Metadata.Resolutions) = %d, want %d",
					len(override.Metadata.Resolutions), tt.expectedResolutions)
			}

			// verify resolution details
			for i, resolution := range override.Metadata.Resolutions {
				if resolution.ServiceName == "" {
					t.Errorf("resolution[%d].ServiceName is empty", i)
				}
				if resolution.ConflictPort == 0 {
					t.Errorf("resolution[%d].ConflictPort is 0", i)
				}
				if resolution.ResolvedPort == 0 {
					t.Errorf("resolution[%d].ResolvedPort is 0", i)
				}
			}
		})
	}
}

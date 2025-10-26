package generator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

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

func TestGenerateFromConflicts(t *testing.T) {
	allocator := &stubPortAllocator{ports: []int{9000}}
	generator := NewUnifiedOverrideGeneratorImpl(allocator, &logger.NopLogger{})

	config := &types.ComposeConfig{Services: map[string]types.Service{
		"web": {Ports: []types.PortMapping{{Host: 8080, Container: 80}}},
	}}
	conflicts := &types.UnifiedConflictInfo{
		PortConflicts: []types.PortConflictInfo{{
			ServiceName: "web",
			Port:        8080,
			Resolution:  &types.PortResolutionInfo{ResolvedPort: 9000},
		}},
		NetworkConflicts: []types.NetworkConflictInfo{{
			NetworkName: "net",
			Resolution: &types.NetworkResolutionInfo{
				ResolvedSubnet: "10.1.0.0/24",
				ServiceIPs:     map[string]string{"web": "10.1.0.2"},
			},
		}},
	}

	override, err := generator.GenerateFromConflicts(context.Background(), config, conflicts)
	if err != nil {
		t.Fatalf("GenerateFromConflicts error: %v", err)
	}
	if override.Services["web"].Ports[0].Host != 9000 {
		t.Fatalf("expected port override to be applied")
	}
	if override.Networks["net"].IPAM.Config[0].Subnet != "10.1.0.0/24" {
		t.Fatalf("expected network override")
	}
	if len(override.Metadata.Resolutions) != 1 {
		t.Fatalf("expected metadata resolutions")
	}
}

func TestResolveConflictsAppliesAllocator(t *testing.T) {
	allocator := &stubPortAllocator{ports: []int{9001, 9002}}
	generator := NewUnifiedOverrideGeneratorImpl(allocator, &logger.NopLogger{})

	info := &types.UnifiedConflictInfo{PortConflicts: []types.PortConflictInfo{{ServiceName: "web", Port: 8080}}}
	portConfig := types.PortConfig{Range: types.PortRange{Start: 8000, End: 9005}}

	if err := generator.ResolveConflicts(context.Background(), info, types.ResolutionStrategyAutoIncrement, portConfig); err != nil {
		t.Fatalf("ResolveConflicts error: %v", err)
	}
	if info.PortConflicts[0].Resolution == nil || info.PortConflicts[0].Resolution.ResolvedPort != 9001 {
		t.Fatalf("expected port to be resolved via allocator")
	}
}

func TestGeneratePortOverrides(t *testing.T) {
	generator := NewUnifiedOverrideGeneratorImpl(&stubPortAllocator{}, &logger.NopLogger{})
	config := &types.ComposeConfig{Services: map[string]types.Service{
		"api": {Ports: []types.PortMapping{{Host: 8080, Container: 8080}}},
	}}
	conflicts := []types.PortConflictInfo{{ServiceName: "api", Port: 8080, Resolution: &types.PortResolutionInfo{ResolvedPort: 9000}}}
	override := &types.OverrideConfig{Services: map[string]types.ServiceOverride{}, Networks: map[string]types.NetworkOverride{}}

	if err := generator.generatePortOverrides(context.Background(), config, conflicts, override); err != nil {
		t.Fatalf("generatePortOverrides error: %v", err)
	}
	if override.Services["api"].Ports[0].Host != 9000 {
		t.Fatalf("expected resolved port in override")
	}
}

func TestGenerateNetworkOverrides(t *testing.T) {
	generator := NewUnifiedOverrideGeneratorImpl(&stubPortAllocator{}, &logger.NopLogger{})
	config := &types.ComposeConfig{}
	conflicts := []types.NetworkConflictInfo{{
		NetworkName: "net",
		Resolution: &types.NetworkResolutionInfo{
			ResolvedSubnet: "10.2.0.0/24",
			ServiceIPs:     map[string]string{"web": "10.2.0.2"},
		},
	}}
	override := &types.OverrideConfig{Services: map[string]types.ServiceOverride{}, Networks: map[string]types.NetworkOverride{}}

	if err := generator.generateNetworkOverrides(context.Background(), config, conflicts, override); err != nil {
		t.Fatalf("generateNetworkOverrides error: %v", err)
	}
	if override.Networks["net"].IPAM.Config[0].Subnet != "10.2.0.0/24" {
		t.Fatalf("expected subnet override")
	}
	if override.Services["web"].Networks["net"].IPv4Address != "10.2.0.2" {
		t.Fatalf("expected service IP override")
	}
}

func TestPopulateMetadata(t *testing.T) {
	generator := NewUnifiedOverrideGeneratorImpl(&stubPortAllocator{}, &logger.NopLogger{})
	conflict := &types.UnifiedConflictInfo{
		PortConflicts: []types.PortConflictInfo{{
			ServiceName: "api",
			Port:        8080,
			Resolution:  &types.PortResolutionInfo{ResolvedPort: 9000, Strategy: types.ResolutionStrategyAutoIncrement, Reason: "auto"},
		}},
		GeneratedAt: time.Now(),
	}
	override := &types.OverrideConfig{}

	generator.populateMetadata(conflict, override)
	if len(override.Metadata.Resolutions) != 1 {
		t.Fatalf("expected resolution metadata entry")
	}
	if override.Metadata.Resolutions[0].ResolvedPort != 9000 {
		t.Fatalf("metadata not populated correctly")
	}
}

func TestAllocateNewSubnet(t *testing.T) {
	generator := NewUnifiedOverrideGeneratorImpl(&stubPortAllocator{}, &logger.NopLogger{})
	used := map[string]bool{"10.20.0.0/24": true}
	subnet := generator.allocateNewSubnet(used)
	if subnet == "" || used[subnet] {
		t.Fatalf("unexpected subnet allocation: %s", subnet)
	}
}

func TestRemapIPAddressesToNewSubnet(t *testing.T) {
	generator := NewUnifiedOverrideGeneratorImpl(&stubPortAllocator{}, &logger.NopLogger{})
	serviceIPs := map[string]string{"api": "10.0.0.5"}
	remapped, err := generator.remapIPAddressesToNewSubnet("10.0.0.0/24", "10.1.0.0/24", serviceIPs)
	if err != nil {
		t.Fatalf("remap error: %v", err)
	}
	if remapped["api"] != "10.1.0.5" {
		t.Fatalf("expected host portion preserved")
	}
}

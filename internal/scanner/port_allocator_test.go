package scanner

import (
	"context"
	"fmt"
	"testing"

	"github.com/harakeishi/gopose/internal/testutil"
	"github.com/harakeishi/gopose/pkg/types"
)

// mockPortDetector is a mock port detector for testing
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
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	tests := []struct {
		name          string
		usedPorts     []int
		reservedPorts []int
		portRange     types.PortRange
		expectedPort  int
	}{
		{
			name:          "skip reserved ports and allocate next available",
			usedPorts:     []int{},
			reservedPorts: []int{8000, 8001, 8002},
			portRange:     types.PortRange{Start: 8000, End: 8100},
			expectedPort:  8003,
		},
		{
			name:          "skip both used and reserved ports",
			usedPorts:     []int{8003, 8004},
			reservedPorts: []int{8000, 8001, 8002},
			portRange:     types.PortRange{Start: 8000, End: 8100},
			expectedPort:  8005,
		},
		{
			name:          "skip reserved ports in middle of range",
			usedPorts:     []int{},
			reservedPorts: []int{8010, 8020, 8030},
			portRange:     types.PortRange{Start: 8000, End: 8100},
			expectedPort:  8000,
		},
		{
			name:          "first is reserved, next is available",
			usedPorts:     []int{},
			reservedPorts: []int{9000},
			portRange:     types.PortRange{Start: 9000, End: 9100},
			expectedPort:  9001,
		},
		{
			name:          "skip multiple reserved ports",
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

			// verify not allocated to reserved port
			for _, reserved := range tt.reservedPorts {
				if port == reserved {
					t.Errorf("AllocatePort() = %d, which is a reserved port", port)
				}
			}
		})
	}
}

func TestAllocatePorts_WithReservedPorts(t *testing.T) {
	testLogger := testutil.NewTestLogger()
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
			name:          "skip reserved ports in multi-port allocation",
			usedPorts:     []int{},
			reservedPorts: []int{8000, 8001, 8002},
			portRange:     types.PortRange{Start: 8000, End: 8100},
			count:         3,
			expectedPorts: []int{8003, 8004, 8005},
		},
		{
			name:          "skip reserved and used ports in sequential allocation",
			usedPorts:     []int{8004, 8005},
			reservedPorts: []int{8000, 8001, 8002},
			portRange:     types.PortRange{Start: 8000, End: 8100},
			count:         3,
			expectedPorts: []int{8003, 8006, 8007},
		},
		{
			name:          "reserved ports scattered across range",
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

				// verify not allocated to reserved port
				for _, reserved := range tt.reservedPorts {
					if port == reserved {
						t.Errorf("AllocatePorts() allocated reserved port %d", port)
					}
				}

				// verify not allocated to used port
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
	testLogger := testutil.NewTestLogger()
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
			name:          "allocate to services skipping reserved ports",
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
			name:          "allocate to services with scattered reserved ports",
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

				// verify not allocated to reserved port
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
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	// reserve some ports in an unused port range
	mockDetector := &mockPortDetector{usedPorts: []int{}} // no ports in use
	allocator := NewPortAllocatorImpl(mockDetector, testLogger)

	config := types.PortConfig{
		Range:             types.PortRange{Start: 9000, End: 9100},
		Reserved:          []int{9000, 9001, 9002, 9050, 9051, 9052}, // reserved even if unused
		ExcludePrivileged: false,
	}

	// allocate 10 ports
	ports, err := allocator.AllocatePorts(ctx, 10, config)
	if err != nil {
		t.Fatalf("AllocatePorts() error = %v, want nil", err)
	}

	// verify reserved ports are not allocated
	reservedMap := make(map[int]bool)
	for _, reserved := range config.Reserved {
		reservedMap[reserved] = true
	}

	for _, port := range ports {
		if reservedMap[port] {
			t.Errorf("AllocatePorts() allocated reserved port %d, even though it was unused", port)
		}
	}

	// expected first available port is 9003 (9000-9002 are reserved)
	expectedFirstPort := 9003
	if ports[0] != expectedFirstPort {
		t.Errorf("AllocatePorts()[0] = %d, want %d (first non-reserved port)", ports[0], expectedFirstPort)
	}
}

// --- Edge case tests ---

func TestAllocatePort_NoAvailablePorts(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	// All ports in the range are used
	mockDetector := &mockPortDetector{usedPorts: []int{5000, 5001, 5002}}
	allocator := NewPortAllocatorImpl(mockDetector, testLogger)

	config := types.PortConfig{
		Range:             types.PortRange{Start: 5000, End: 5002},
		ExcludePrivileged: false,
	}

	_, err := allocator.AllocatePort(ctx, config)
	if err == nil {
		t.Error("AllocatePort() expected error when all ports are used, got nil")
	}
}

func TestAllocatePort_ExcludePrivileged(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	// Range includes privileged ports (80-1100), ExcludePrivileged=true
	mockDetector := &mockPortDetector{usedPorts: []int{}}
	allocator := NewPortAllocatorImpl(mockDetector, testLogger)

	config := types.PortConfig{
		Range:             types.PortRange{Start: 80, End: 1100},
		ExcludePrivileged: true,
	}

	port, err := allocator.AllocatePort(ctx, config)
	if err != nil {
		t.Fatalf("AllocatePort() error = %v, want nil", err)
	}

	if port <= 1023 {
		t.Errorf("AllocatePort() = %d, want > 1023 when ExcludePrivileged is true", port)
	}
}

func TestAllocatePorts_ZeroCount(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	mockDetector := &mockPortDetector{usedPorts: []int{}}
	allocator := NewPortAllocatorImpl(mockDetector, testLogger)

	config := types.PortConfig{
		Range:             types.PortRange{Start: 8000, End: 8100},
		ExcludePrivileged: false,
	}

	ports, err := allocator.AllocatePorts(ctx, 0, config)
	if err != nil {
		t.Errorf("AllocatePorts() error = %v, want nil", err)
	}

	if len(ports) != 0 {
		t.Errorf("AllocatePorts() returned %d ports, want 0", len(ports))
	}
}

func TestAllocatePorts_InsufficientPorts(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	// Range has 3 ports (6000-6002), but 2 are used, leaving only 1 available
	mockDetector := &mockPortDetector{usedPorts: []int{6000, 6001}}
	allocator := NewPortAllocatorImpl(mockDetector, testLogger)

	config := types.PortConfig{
		Range:             types.PortRange{Start: 6000, End: 6002},
		ExcludePrivileged: false,
	}

	_, err := allocator.AllocatePorts(ctx, 3, config)
	if err == nil {
		t.Error("AllocatePorts() expected error when insufficient ports available, got nil")
	}
}

func TestAllocatePortsForServices_NoPortServices(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	mockDetector := &mockPortDetector{usedPorts: []int{}}
	allocator := NewPortAllocatorImpl(mockDetector, testLogger)

	config := types.PortConfig{
		Range:             types.PortRange{Start: 8000, End: 8100},
		ExcludePrivileged: false,
	}

	// Service with empty Ports slice
	services := []types.Service{
		{Name: "worker", Ports: []types.PortMapping{}},
	}

	result, err := allocator.AllocatePortsForServices(ctx, services, config)
	if err != nil {
		t.Errorf("AllocatePortsForServices() error = %v, want nil", err)
	}

	if len(result) != 0 {
		t.Errorf("AllocatePortsForServices() returned %d entries, want 0", len(result))
	}
}

func TestAllocatePortsForServices_DetectorError(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	// Use testutil.MockPortDetector which supports Err field
	mockDetector := &testutil.MockPortDetector{
		UsedPorts: []int{},
		Err:       fmt.Errorf("detector failure"),
	}
	allocator := NewPortAllocatorImpl(mockDetector, testLogger)

	config := types.PortConfig{
		Range:             types.PortRange{Start: 8000, End: 8100},
		ExcludePrivileged: false,
	}

	services := []types.Service{
		{Name: "web", Ports: []types.PortMapping{{Host: 80, Container: 80}}},
	}

	_, err := allocator.AllocatePortsForServices(ctx, services, config)
	if err == nil {
		t.Error("AllocatePortsForServices() expected error when detector fails, got nil")
	}
}

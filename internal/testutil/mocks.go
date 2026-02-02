package testutil

import (
	"context"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

// NewTestLogger creates a logger for testing with minimal configuration.
func NewTestLogger() logger.Logger {
	factory := logger.NewStructuredLoggerFactory(false)
	l, _ := factory.Create(types.LogConfig{})
	return l
}

// MockPortDetector is a mock port detector for testing.
type MockPortDetector struct {
	UsedPorts []int
	Err       error
}

func (m *MockPortDetector) DetectUsedPorts(ctx context.Context) ([]int, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.UsedPorts, nil
}

func (m *MockPortDetector) DetectUsedPortsInRange(ctx context.Context, portRange types.PortRange) ([]int, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	var portsInRange []int
	for _, port := range m.UsedPorts {
		if port >= portRange.Start && port <= portRange.End {
			portsInRange = append(portsInRange, port)
		}
	}
	return portsInRange, nil
}

func (m *MockPortDetector) IsPortInUse(ctx context.Context, port int) (bool, error) {
	if m.Err != nil {
		return false, m.Err
	}
	for _, p := range m.UsedPorts {
		if p == port {
			return true, nil
		}
	}
	return false, nil
}

// MockPortAllocator is a mock port allocator for testing.
type MockPortAllocator struct {
	NextPort int
	Err      error
}

func (m *MockPortAllocator) AllocatePort(ctx context.Context, config types.PortConfig) (int, error) {
	if m.Err != nil {
		return 0, m.Err
	}
	port := m.NextPort
	m.NextPort++
	return port, nil
}

func (m *MockPortAllocator) AllocatePorts(ctx context.Context, count int, config types.PortConfig) ([]int, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	ports := make([]int, count)
	for i := 0; i < count; i++ {
		ports[i] = m.NextPort
		m.NextPort++
	}
	return ports, nil
}

func (m *MockPortAllocator) AllocatePortsForServices(ctx context.Context, services []types.Service, config types.PortConfig) (map[string]int, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	result := make(map[string]int)
	for _, service := range services {
		result[service.Name] = m.NextPort
		m.NextPort++
	}
	return result, nil
}

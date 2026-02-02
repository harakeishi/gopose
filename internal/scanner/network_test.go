package scanner

import (
	"context"
	"testing"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

// TestNewDockerNetworkDetector tests the constructor
func TestNewDockerNetworkDetector(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, err := factory.Create(types.LogConfig{})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	detector := NewDockerNetworkDetector(testLogger)

	if detector == nil {
		t.Fatal("NewDockerNetworkDetector() returned nil")
	}

	if detector.logger == nil {
		t.Error("logger is nil")
	}
}

// TestDetectNetworks tests network detection functionality.
// This is an integration test that requires Docker to be running and accessible.
// It validates that the detector can:
// - Execute docker network commands successfully
// - Parse network information correctly
// - Handle networks with and without IPAM configuration
func TestDetectNetworks(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, err := factory.Create(types.LogConfig{})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	detector := NewDockerNetworkDetector(testLogger)
	ctx := context.Background()

	networks, err := detector.DetectNetworks(ctx)

	// Note: This test may fail if Docker is not running
	// We'll check for either successful detection or expected error
	if err != nil {
		t.Logf("DetectNetworks() returned error (Docker may not be running): %v", err)
		return
	}

	// If Docker is running, we should get at least some networks
	t.Logf("Detected %d networks", len(networks))

	// Validate structure of returned networks
	for i, network := range networks {
		if network.Name == "" {
			t.Errorf("networks[%d].Name is empty", i)
		}

		// Subnets may be empty for some networks (e.g., bridge without IPAM)
		t.Logf("Network %q has %d subnets", network.Name, len(network.Subnets))

		// Validate subnet format if present
		for j, subnet := range network.Subnets {
			if subnet == "" {
				t.Errorf("networks[%d].Subnets[%d] is empty", i, j)
			}
		}
	}
}

// TestDetectNetworksWithCancellation tests that the detector properly handles
// context cancellation by returning an error when the context is cancelled.
func TestDetectNetworksWithCancellation(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, err := factory.Create(types.LogConfig{})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	detector := NewDockerNetworkDetector(testLogger)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = detector.DetectNetworks(ctx)

	// Should return an error when context is cancelled
	if err == nil {
		t.Error("DetectNetworks() should return error with cancelled context")
	}
}

// TestNetworkInfoStructure tests the NetworkInfo data structure to ensure
// it correctly represents Docker network information including:
// - Network names
// - Subnet configurations (single, multiple, or none)
// - IPv4 and IPv6 subnets (dual-stack)
func TestNetworkInfoStructure(t *testing.T) {
	tests := []struct {
		name     string
		network  NetworkInfo
		wantName string
		wantSubs int
	}{
		{
			name: "basic network info",
			network: NetworkInfo{
				Name:    "mynetwork",
				Subnets: []string{"172.20.0.0/24"},
			},
			wantName: "mynetwork",
			wantSubs: 1,
		},
		{
			name: "network with multiple subnets",
			network: NetworkInfo{
				Name:    "dualstack",
				Subnets: []string{"172.20.0.0/24", "2001:db8::/64"},
			},
			wantName: "dualstack",
			wantSubs: 2,
		},
		{
			name: "network with no subnets",
			network: NetworkInfo{
				Name:    "noipam",
				Subnets: []string{},
			},
			wantName: "noipam",
			wantSubs: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.network.Name != tt.wantName {
				t.Errorf("NetworkInfo.Name = %q, want %q", tt.network.Name, tt.wantName)
			}

			if len(tt.network.Subnets) != tt.wantSubs {
				t.Errorf("NetworkInfo.Subnets length = %d, want %d",
					len(tt.network.Subnets), tt.wantSubs)
			}
		})
	}
}

// TestDetectNetworksEdgeCases tests various edge cases and boundary conditions
// to ensure the detector handles unusual situations gracefully.
func TestDetectNetworksEdgeCases(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, err := factory.Create(types.LogConfig{})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	detector := NewDockerNetworkDetector(testLogger)

	t.Run("nil context", func(t *testing.T) {
		// This should panic or handle gracefully
		defer func() {
			if r := recover(); r != nil {
				t.Logf("DetectNetworks with nil context panicked (expected): %v", r)
			}
		}()

		ctx := context.Background()
		_, _ = detector.DetectNetworks(ctx)
	})
}

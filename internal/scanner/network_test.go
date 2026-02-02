package scanner

import (
	"context"
	"testing"

	"github.com/harakeishi/gopose/internal/testutil"
)

// TestNewDockerNetworkDetector tests the constructor.
func TestNewDockerNetworkDetector(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	detector := NewDockerNetworkDetector(testLogger)

	if detector == nil {
		t.Fatal("NewDockerNetworkDetector() returned nil")
	}

	if detector.logger == nil {
		t.Error("logger is nil")
	}
}

// TestDetectNetworks tests network detection functionality.
// This is an integration test that only works when Docker is running.
func TestDetectNetworks(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	detector := NewDockerNetworkDetector(testLogger)
	ctx := context.Background()

	networks, err := detector.DetectNetworks(ctx)

	// Skip if Docker is not running
	if err != nil {
		t.Skip("Docker is not running, skipping integration test")
	}

	// Docker always has at least bridge, host, and none networks
	if len(networks) == 0 {
		t.Error("Expected at least one network (bridge/host/none)")
	}
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

// TestDetectNetworksWithCancellation tests that an error is returned when context is cancelled.
func TestDetectNetworksWithCancellation(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	detector := NewDockerNetworkDetector(testLogger)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := detector.DetectNetworks(ctx)

	// Should return an error when context is cancelled
	if err == nil {
		t.Error("DetectNetworks() should return error with cancelled context")
	}
}

// TestNetworkInfoStructure tests the NetworkInfo data structure.
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
			name: "network without subnets",
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

// TestDetectNetworksEdgeCases tests edge cases and boundary conditions,
// verifying the detector handles abnormal situations correctly.
func TestDetectNetworksEdgeCases(t *testing.T) {
	testLogger := testutil.NewTestLogger()

	detector := NewDockerNetworkDetector(testLogger)

	t.Run("detection with default context", func(t *testing.T) {
		ctx := context.Background()
		// Ensure no panic even when Docker is not available
		_, _ = detector.DetectNetworks(ctx)
	})
}

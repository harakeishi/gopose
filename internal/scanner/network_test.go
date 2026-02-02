package scanner

import (
	"context"
	"testing"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

// TestNewDockerNetworkDetector はコンストラクタをテストします。
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

// TestDetectNetworks はネットワーク検出機能をテストします。
// Dockerが起動している環境でのみ正常に動作する統合テストです。
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

// TestDetectNetworksWithCancellation はコンテキストキャンセル時にエラーを返すことをテストします。
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

// TestNetworkInfoStructure はNetworkInfoデータ構造のテストです。
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

// TestDetectNetworksEdgeCases はエッジケースや境界条件をテストし、
// 検出器が異常な状況を適切に処理することを確認します。
func TestDetectNetworksEdgeCases(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, err := factory.Create(types.LogConfig{})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	detector := NewDockerNetworkDetector(testLogger)

	t.Run("detection with default context", func(t *testing.T) {
		ctx := context.Background()
		// Ensure no panic even when Docker is not available
		_, _ = detector.DetectNetworks(ctx)
	})
}

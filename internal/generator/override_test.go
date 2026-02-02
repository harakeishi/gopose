package generator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

func TestGenerateOverride(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	generator := NewOverrideGeneratorImpl(testLogger)
	ctx := context.Background()

	tests := []struct {
		name         string
		config       *types.ComposeConfig
		resolutions  []types.ConflictResolution
		expectError  bool
		serviceCount int
	}{
		{
			name: "normal override generation",
			config: &types.ComposeConfig{
				Version: "3.8",
				Services: map[string]types.Service{
					"web": {
						Name: "web",
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			resolutions: []types.ConflictResolution{
				{
					ServiceName:  "web",
					ConflictPort: 8080,
					ResolvedPort: 8081,
					Strategy:     types.StrategyMinimalChange,
					Reason:       "Port conflict resolved",
					Timestamp:    time.Now(),
				},
			},
			expectError:  false,
			serviceCount: 1,
		},
		{
			name: "multi-service override generation",
			config: &types.ComposeConfig{
				Version: "3.8",
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
							{Host: 3000, Container: 3000, Protocol: "tcp"},
						},
					},
				},
			},
			resolutions: []types.ConflictResolution{
				{
					ServiceName:  "web",
					ConflictPort: 8080,
					ResolvedPort: 8081,
					Strategy:     types.StrategyMinimalChange,
					Reason:       "Port conflict resolved",
					Timestamp:    time.Now(),
				},
				{
					ServiceName:  "api",
					ConflictPort: 3000,
					ResolvedPort: 3001,
					Strategy:     types.StrategyMinimalChange,
					Reason:       "Port conflict resolved",
					Timestamp:    time.Now(),
				},
			},
			expectError:  false,
			serviceCount: 2,
		},
		{
			name: "override generation with no resolutions",
			config: &types.ComposeConfig{
				Version: "3.8",
				Services: map[string]types.Service{
					"web": {
						Name: "web",
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			resolutions:  []types.ConflictResolution{},
			expectError:  false,
			serviceCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generator.GenerateOverride(ctx, tt.config, tt.resolutions)

			if tt.expectError {
				if err == nil {
					t.Errorf("GenerateOverride() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GenerateOverride() unexpected error: %v", err)
				return
			}

			if len(result.Services) != tt.serviceCount {
				t.Errorf("GenerateOverride() service count = %d, want %d", len(result.Services), tt.serviceCount)
			}

			if result.Version != tt.config.Version {
				t.Errorf("GenerateOverride() version = %q, want %q", result.Version, tt.config.Version)
			}

			if len(result.Metadata.Resolutions) != len(tt.resolutions) {
				t.Errorf("GenerateOverride() resolutions count = %d, want %d", len(result.Metadata.Resolutions), len(tt.resolutions))
			}
		})
	}
}

func TestWriteOverrideFile(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	generator := NewOverrideGeneratorImpl(testLogger)
	ctx := context.Background()

	tests := []struct {
		name        string
		override    *types.OverrideConfig
		expectError bool
	}{
		{
			name: "normal file write",
			override: &types.OverrideConfig{
				Version: "3.8",
				Services: map[string]types.ServiceOverride{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8081, Container: 80, Protocol: "tcp"},
						},
					},
				},
				Metadata: types.OverrideMetadata{
					GeneratedAt: time.Now(),
					Version:     "1.0.0",
					Resolutions: []types.ConflictResolution{},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create temp directory
			tmpDir := t.TempDir()
			outputPath := filepath.Join(tmpDir, "docker-compose.override.yml")

			err := generator.WriteOverrideFile(ctx, tt.override, outputPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("WriteOverrideFile() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("WriteOverrideFile() unexpected error: %v", err)
				return
			}

			// verify file was created
			if _, err := os.Stat(outputPath); os.IsNotExist(err) {
				t.Errorf("WriteOverrideFile() file not created")
			}

			// verify file contents
			content, err := os.ReadFile(outputPath)
			if err != nil {
				t.Errorf("Failed to read generated file: %v", err)
			}

			if len(content) == 0 {
				t.Errorf("WriteOverrideFile() generated empty file")
			}
		})
	}
}

func TestValidateOverride(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	generator := NewOverrideGeneratorImpl(testLogger)
	ctx := context.Background()

	tests := []struct {
		name        string
		override    *types.OverrideConfig
		expectError bool
	}{
		{
			name: "valid override validation",
			override: &types.OverrideConfig{
				Version: "3.8",
				Services: map[string]types.ServiceOverride{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
				},
				Metadata: types.OverrideMetadata{
					GeneratedAt: time.Now(),
					Version:     "1.0.0",
					Resolutions: []types.ConflictResolution{
						{
							ServiceName:  "web",
							ConflictPort: 8080,
							ResolvedPort: 8081,
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "duplicate port detection",
			override: &types.OverrideConfig{
				Version: "3.8",
				Services: map[string]types.ServiceOverride{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
							{Host: 8080, Container: 443, Protocol: "tcp"},
						},
					},
				},
				Metadata: types.OverrideMetadata{
					GeneratedAt: time.Now(),
					Version:     "1.0.0",
					Resolutions: []types.ConflictResolution{},
				},
			},
			expectError: true,
		},
		{
			name: "invalid port range",
			override: &types.OverrideConfig{
				Version: "3.8",
				Services: map[string]types.ServiceOverride{
					"web": {
						Ports: []types.PortMapping{
							{Host: -1, Container: 80, Protocol: "tcp"},
						},
					},
				},
				Metadata: types.OverrideMetadata{
					GeneratedAt: time.Now(),
					Version:     "1.0.0",
					Resolutions: []types.ConflictResolution{},
				},
			},
			expectError: true,
		},
		{
			name: "empty services",
			override: &types.OverrideConfig{
				Version:  "3.8",
				Services: map[string]types.ServiceOverride{},
				Metadata: types.OverrideMetadata{
					GeneratedAt: time.Now(),
					Version:     "1.0.0",
					Resolutions: []types.ConflictResolution{},
				},
			},
			expectError: false, // produces warning but not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.ValidateOverride(ctx, tt.override)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateOverride() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateOverride() unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateOverrideYAML(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	generator := NewOverrideGeneratorImpl(testLogger)

	tests := []struct {
		name     string
		override *types.OverrideConfig
		contains []string
	}{
		{
			name: "basic YAML generation",
			override: &types.OverrideConfig{
				Version: "3.8",
				Services: map[string]types.ServiceOverride{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8081, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			contains: []string{
				"services:",
				"web:",
				"ports: !override",
				"8081:80",
			},
		},
		{
			name: "YAML generation with network config",
			override: &types.OverrideConfig{
				Version: "3.8",
				Services: map[string]types.ServiceOverride{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
						Networks: map[string]types.ServiceNetwork{
							"mynetwork": {
								IPv4Address: "10.5.0.2",
							},
						},
					},
				},
			},
			contains: []string{
				"services:",
				"web:",
				"networks:",
				"mynetwork:",
				"ipv4_address: 10.5.0.2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.generateOverrideYAML(tt.override)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("generateOverrideYAML() does not contain %q", expected)
				}
			}
		})
	}
}

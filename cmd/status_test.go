package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harakeishi/gopose/internal/testutil"
	"github.com/harakeishi/gopose/pkg/types"
)

func TestCollectPortInfos(t *testing.T) {
	tests := []struct {
		name     string
		config   *types.ComposeConfig
		override *types.OverrideConfig
		expected []PortInfo
	}{
		{
			name: "single service without override",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Name: "web",
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			override: nil,
			expected: []PortInfo{
				{
					Service:       "web",
					HostPort:      8080,
					ContainerPort: 80,
					Protocol:      "tcp",
					Overridden:    false,
				},
			},
		},
		{
			name: "single service with override",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Name: "web",
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			override: &types.OverrideConfig{
				Services: map[string]types.ServiceOverride{
					"web": {
						Ports: []types.PortMapping{
							{Host: 9090, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			expected: []PortInfo{
				{
					Service:       "web",
					HostPort:      9090,
					ContainerPort: 80,
					Protocol:      "tcp",
					Overridden:    true,
					OriginalPort:  8080,
				},
			},
		},
		{
			name: "multiple services sorted alphabetically",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"db": {
						Name: "db",
						Ports: []types.PortMapping{
							{Host: 5432, Container: 5432, Protocol: "tcp"},
						},
					},
					"api": {
						Name: "api",
						Ports: []types.PortMapping{
							{Host: 3000, Container: 3000, Protocol: "tcp"},
						},
					},
					"web": {
						Name: "web",
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			override: nil,
			expected: []PortInfo{
				{Service: "api", HostPort: 3000, ContainerPort: 3000, Protocol: "tcp", Overridden: false},
				{Service: "db", HostPort: 5432, ContainerPort: 5432, Protocol: "tcp", Overridden: false},
				{Service: "web", HostPort: 8080, ContainerPort: 80, Protocol: "tcp", Overridden: false},
			},
		},
		{
			name: "service with multiple ports",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Name: "web",
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
							{Host: 8443, Container: 443, Protocol: "tcp"},
						},
					},
				},
			},
			override: nil,
			expected: []PortInfo{
				{Service: "web", HostPort: 8080, ContainerPort: 80, Protocol: "tcp", Overridden: false},
				{Service: "web", HostPort: 8443, ContainerPort: 443, Protocol: "tcp", Overridden: false},
			},
		},
		{
			name: "partial override - only one port overridden",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Name: "web",
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
							{Host: 8443, Container: 443, Protocol: "tcp"},
						},
					},
				},
			},
			override: &types.OverrideConfig{
				Services: map[string]types.ServiceOverride{
					"web": {
						Ports: []types.PortMapping{
							{Host: 9090, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			expected: []PortInfo{
				{Service: "web", HostPort: 9090, ContainerPort: 80, Protocol: "tcp", Overridden: true, OriginalPort: 8080},
				{Service: "web", HostPort: 8443, ContainerPort: 443, Protocol: "tcp", Overridden: false},
			},
		},
		{
			name: "override with same port - should not mark as overridden",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Name: "web",
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			override: &types.OverrideConfig{
				Services: map[string]types.ServiceOverride{
					"web": {
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			expected: []PortInfo{
				{Service: "web", HostPort: 8080, ContainerPort: 80, Protocol: "tcp", Overridden: false},
			},
		},
		{
			name: "service without ports",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"worker": {
						Name:  "worker",
						Ports: []types.PortMapping{},
					},
				},
			},
			override: nil,
			expected: []PortInfo{},
		},
		{
			name: "override for non-existent service - should be ignored",
			config: &types.ComposeConfig{
				Services: map[string]types.Service{
					"web": {
						Name: "web",
						Ports: []types.PortMapping{
							{Host: 8080, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			override: &types.OverrideConfig{
				Services: map[string]types.ServiceOverride{
					"other": {
						Ports: []types.PortMapping{
							{Host: 9090, Container: 80, Protocol: "tcp"},
						},
					},
				},
			},
			expected: []PortInfo{
				{Service: "web", HostPort: 8080, ContainerPort: 80, Protocol: "tcp", Overridden: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := collectPortInfos(tt.config, tt.override)

			if len(result) != len(tt.expected) {
				t.Errorf("collectPortInfos() returned %d items, want %d", len(result), len(tt.expected))
				return
			}

			for i, expected := range tt.expected {
				actual := result[i]
				if actual.Service != expected.Service {
					t.Errorf("result[%d].Service = %q, want %q", i, actual.Service, expected.Service)
				}
				if actual.HostPort != expected.HostPort {
					t.Errorf("result[%d].HostPort = %d, want %d", i, actual.HostPort, expected.HostPort)
				}
				if actual.ContainerPort != expected.ContainerPort {
					t.Errorf("result[%d].ContainerPort = %d, want %d", i, actual.ContainerPort, expected.ContainerPort)
				}
				if actual.Protocol != expected.Protocol {
					t.Errorf("result[%d].Protocol = %q, want %q", i, actual.Protocol, expected.Protocol)
				}
				if actual.Overridden != expected.Overridden {
					t.Errorf("result[%d].Overridden = %v, want %v", i, actual.Overridden, expected.Overridden)
				}
				if actual.OriginalPort != expected.OriginalPort {
					t.Errorf("result[%d].OriginalPort = %d, want %d", i, actual.OriginalPort, expected.OriginalPort)
				}
			}
		})
	}
}

func TestOutputTable(t *testing.T) {
	tests := []struct {
		name      string
		portInfos []PortInfo
		contains  []string
	}{
		{
			name: "basic output",
			portInfos: []PortInfo{
				{Service: "web", HostPort: 8080, ContainerPort: 80, Protocol: "tcp", Overridden: false},
			},
			contains: []string{
				"SERVICE",
				"HOST PORT",
				"CONTAINER PORT",
				"STATUS",
				"web",
				"8080",
				"80",
				"-",
			},
		},
		{
			name: "overridden port",
			portInfos: []PortInfo{
				{Service: "web", HostPort: 9090, ContainerPort: 80, Protocol: "tcp", Overridden: true, OriginalPort: 8080},
			},
			contains: []string{
				"web",
				"9090",
				"80",
				"overridden",
				"(8080)",
			},
		},
		{
			name:      "empty list",
			portInfos: []PortInfo{},
			contains: []string{
				"SERVICE",
				"HOST PORT",
				"CONTAINER PORT",
				"STATUS",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputTable(&buf, tt.portInfos)
			if err != nil {
				t.Errorf("outputTable() error = %v", err)
				return
			}

			output := buf.String()
			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("outputTable() output does not contain %q\nGot:\n%s", expected, output)
				}
			}
		})
	}
}

func TestOutputJSON(t *testing.T) {
	tests := []struct {
		name      string
		portInfos []PortInfo
	}{
		{
			name: "basic output",
			portInfos: []PortInfo{
				{Service: "web", HostPort: 8080, ContainerPort: 80, Protocol: "tcp", Overridden: false},
			},
		},
		{
			name: "overridden port",
			portInfos: []PortInfo{
				{Service: "web", HostPort: 9090, ContainerPort: 80, Protocol: "tcp", Overridden: true, OriginalPort: 8080},
			},
		},
		{
			name:      "empty list",
			portInfos: []PortInfo{},
		},
		{
			name: "multiple services",
			portInfos: []PortInfo{
				{Service: "api", HostPort: 3000, ContainerPort: 3000, Protocol: "tcp", Overridden: false},
				{Service: "db", HostPort: 5432, ContainerPort: 5432, Protocol: "tcp", Overridden: false},
				{Service: "web", HostPort: 9090, ContainerPort: 80, Protocol: "tcp", Overridden: true, OriginalPort: 8080},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := outputJSON(&buf, tt.portInfos)
			if err != nil {
				t.Errorf("outputJSON() error = %v", err)
				return
			}

			// Verify JSON is valid
			var decoded []PortInfo
			if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
				t.Errorf("outputJSON() produced invalid JSON: %v\nOutput:\n%s", err, buf.String())
				return
			}

			// Verify content matches
			if len(decoded) != len(tt.portInfos) {
				t.Errorf("outputJSON() decoded %d items, want %d", len(decoded), len(tt.portInfos))
				return
			}

			for i, expected := range tt.portInfos {
				actual := decoded[i]
				if actual.Service != expected.Service {
					t.Errorf("decoded[%d].Service = %q, want %q", i, actual.Service, expected.Service)
				}
				if actual.HostPort != expected.HostPort {
					t.Errorf("decoded[%d].HostPort = %d, want %d", i, actual.HostPort, expected.HostPort)
				}
				if actual.ContainerPort != expected.ContainerPort {
					t.Errorf("decoded[%d].ContainerPort = %d, want %d", i, actual.ContainerPort, expected.ContainerPort)
				}
				if actual.Overridden != expected.Overridden {
					t.Errorf("decoded[%d].Overridden = %v, want %v", i, actual.Overridden, expected.Overridden)
				}
				if actual.OriginalPort != expected.OriginalPort {
					t.Errorf("decoded[%d].OriginalPort = %d, want %d", i, actual.OriginalPort, expected.OriginalPort)
				}
			}
		})
	}
}

func TestLoadOverrideConfig(t *testing.T) {
	logger := testutil.NewTestLogger()
	ctx := context.Background()

	t.Run("override file exists", func(t *testing.T) {
		// Create temp directory
		tmpDir, err := os.MkdirTemp("", "gopose-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create compose.yml
		composeContent := `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
`
		composePath := filepath.Join(tmpDir, "compose.yml")
		if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
			t.Fatalf("Failed to write compose.yml: %v", err)
		}

		// Create compose.override.yml
		overrideContent := `services:
  web:
    ports:
      - "9090:80"
`
		overridePath := filepath.Join(tmpDir, "compose.override.yml")
		if err := os.WriteFile(overridePath, []byte(overrideContent), 0644); err != nil {
			t.Fatalf("Failed to write compose.override.yml: %v", err)
		}

		// Test
		override, err := loadOverrideConfig(ctx, composePath, logger)
		if err != nil {
			t.Errorf("loadOverrideConfig() error = %v", err)
			return
		}

		if override == nil {
			t.Error("loadOverrideConfig() returned nil")
			return
		}

		webOverride, exists := override.Services["web"]
		if !exists {
			t.Error("loadOverrideConfig() did not include 'web' service")
			return
		}

		if len(webOverride.Ports) != 1 {
			t.Errorf("loadOverrideConfig() web.Ports length = %d, want 1", len(webOverride.Ports))
			return
		}

		if webOverride.Ports[0].Host != 9090 {
			t.Errorf("loadOverrideConfig() web.Ports[0].Host = %d, want 9090", webOverride.Ports[0].Host)
		}
	})

	t.Run("override file does not exist", func(t *testing.T) {
		// Create temp directory
		tmpDir, err := os.MkdirTemp("", "gopose-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create compose.yml only (no override)
		composeContent := `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
`
		composePath := filepath.Join(tmpDir, "compose.yml")
		if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
			t.Fatalf("Failed to write compose.yml: %v", err)
		}

		// Test
		override, err := loadOverrideConfig(ctx, composePath, logger)
		if err == nil {
			t.Error("loadOverrideConfig() expected error when override file does not exist")
			return
		}

		if override != nil {
			t.Error("loadOverrideConfig() should return nil when override file does not exist")
		}
	})

	t.Run("docker-compose.override.yml format", func(t *testing.T) {
		// Create temp directory
		tmpDir, err := os.MkdirTemp("", "gopose-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create compose.yml
		composeContent := `services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
`
		composePath := filepath.Join(tmpDir, "compose.yml")
		if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
			t.Fatalf("Failed to write compose.yml: %v", err)
		}

		// Create docker-compose.override.yml (alternative format)
		overrideContent := `services:
  web:
    ports:
      - "7070:80"
`
		overridePath := filepath.Join(tmpDir, "docker-compose.override.yml")
		if err := os.WriteFile(overridePath, []byte(overrideContent), 0644); err != nil {
			t.Fatalf("Failed to write docker-compose.override.yml: %v", err)
		}

		// Test
		override, err := loadOverrideConfig(ctx, composePath, logger)
		if err != nil {
			t.Errorf("loadOverrideConfig() error = %v", err)
			return
		}

		if override == nil {
			t.Error("loadOverrideConfig() returned nil")
			return
		}

		webOverride := override.Services["web"]
		if webOverride.Ports[0].Host != 7070 {
			t.Errorf("loadOverrideConfig() web.Ports[0].Host = %d, want 7070", webOverride.Ports[0].Host)
		}
	})
}

func TestPortInfo_JSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		portInfo PortInfo
	}{
		{
			name: "basic port info",
			portInfo: PortInfo{
				Service:       "web",
				HostPort:      8080,
				ContainerPort: 80,
				Protocol:      "tcp",
				Overridden:    false,
			},
		},
		{
			name: "overridden port info",
			portInfo: PortInfo{
				Service:       "web",
				HostPort:      9090,
				ContainerPort: 80,
				Protocol:      "tcp",
				Overridden:    true,
				OriginalPort:  8080,
			},
		},
		{
			name: "with host IP",
			portInfo: PortInfo{
				Service:       "web",
				HostPort:      8080,
				ContainerPort: 80,
				Protocol:      "tcp",
				HostIP:        "127.0.0.1",
				Overridden:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data, err := json.Marshal(tt.portInfo)
			if err != nil {
				t.Errorf("json.Marshal() error = %v", err)
				return
			}

			// Deserialize
			var decoded PortInfo
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Errorf("json.Unmarshal() error = %v", err)
				return
			}

			// Verify
			if decoded.Service != tt.portInfo.Service {
				t.Errorf("Service = %q, want %q", decoded.Service, tt.portInfo.Service)
			}
			if decoded.HostPort != tt.portInfo.HostPort {
				t.Errorf("HostPort = %d, want %d", decoded.HostPort, tt.portInfo.HostPort)
			}
			if decoded.ContainerPort != tt.portInfo.ContainerPort {
				t.Errorf("ContainerPort = %d, want %d", decoded.ContainerPort, tt.portInfo.ContainerPort)
			}
			if decoded.Protocol != tt.portInfo.Protocol {
				t.Errorf("Protocol = %q, want %q", decoded.Protocol, tt.portInfo.Protocol)
			}
			if decoded.HostIP != tt.portInfo.HostIP {
				t.Errorf("HostIP = %q, want %q", decoded.HostIP, tt.portInfo.HostIP)
			}
			if decoded.Overridden != tt.portInfo.Overridden {
				t.Errorf("Overridden = %v, want %v", decoded.Overridden, tt.portInfo.Overridden)
			}
			if decoded.OriginalPort != tt.portInfo.OriginalPort {
				t.Errorf("OriginalPort = %d, want %d", decoded.OriginalPort, tt.portInfo.OriginalPort)
			}
		})
	}
}

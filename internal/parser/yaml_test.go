package parser

import (
	"context"
	"testing"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

func TestExpandVariables(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "env var expansion - with default value",
			input:    "${APP_PORT:-3000}:80",
			envVars:  map[string]string{},
			expected: "3000:80",
		},
		{
			name:     "env var expansion - env var is set",
			input:    "${APP_PORT:-3000}:80",
			envVars:  map[string]string{"APP_PORT": "4000"},
			expected: "4000:80",
		},
		{
			name:     "env var expansion - ${VAR} format",
			input:    "${PORT}:80",
			envVars:  map[string]string{"PORT": "5000"},
			expected: "5000:80",
		},
		{
			name:     "env var expansion - $VAR format",
			input:    "$PORT:80",
			envVars:  map[string]string{"PORT": "6000"},
			expected: "6000:80",
		},
		{
			name:     "env var expansion - multiple variables",
			input:    "${HOST_PORT:-8080}:${CONTAINER_PORT:-80}",
			envVars:  map[string]string{"HOST_PORT": "9090"},
			expected: "9090:80",
		},
		{
			name:     "no env vars",
			input:    "3000:80",
			envVars:  map[string]string{},
			expected: "3000:80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set env vars
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			result := expandVariables(tt.input)
			if result != tt.expected {
				t.Errorf("expandVariables(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParsePortString(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	parser := NewYamlComposeParser(testLogger)
	ctx := context.Background()

	tests := []struct {
		name     string
		input    string
		envVars  map[string]string
		expected *types.PortMapping
		hasError bool
	}{
		{
			name:    "env var expansion - default value",
			input:   "${APP_PORT:-3000}:80",
			envVars: map[string]string{},
			expected: &types.PortMapping{
				Host:      3000,
				Container: 80,
				Protocol:  "tcp",
			},
			hasError: false,
		},
		{
			name:    "env var expansion - env var is set",
			input:   "${APP_PORT:-3000}:80",
			envVars: map[string]string{"APP_PORT": "4000"},
			expected: &types.PortMapping{
				Host:      4000,
				Container: 80,
				Protocol:  "tcp",
			},
			hasError: false,
		},
		{
			name:    "standard port format",
			input:   "8080:80",
			envVars: map[string]string{},
			expected: &types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "tcp",
			},
			hasError: false,
		},
		{
			name:    "env var with IP address",
			input:   "127.0.0.1:${PORT:-3000}:80",
			envVars: map[string]string{},
			expected: &types.PortMapping{
				Host:      3000,
				Container: 80,
				Protocol:  "tcp",
				HostIP:    "127.0.0.1",
			},
			hasError: false,
		},
		{
			name:     "invalid port format - string",
			input:    "abc:80",
			envVars:  map[string]string{},
			expected: nil,
			hasError: true,
		},
		{
			name:     "invalid port format - wrong delimiter",
			input:    "8080-80",
			envVars:  map[string]string{},
			expected: nil,
			hasError: true,
		},
		{
			name:    "container port only",
			input:   "80",
			envVars: map[string]string{},
			expected: &types.PortMapping{
				Host:      0,
				Container: 80,
				Protocol:  "tcp",
			},
			hasError: false,
		},
		{
			name:    "protocol specified - UDP",
			input:   "8080:80/udp",
			envVars: map[string]string{},
			expected: &types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "udp",
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars not included in this test to empty string
			// Note: t.Setenv("VAR", "") sets empty string, not truly unset
			for _, envKey := range []string{"PORT", "APP_PORT", "HOST_PORT", "CONTAINER_PORT"} {
				if _, exists := tt.envVars[envKey]; !exists {
					t.Setenv(envKey, "")
					// This sets empty string (not unset) to prevent interference
					// t.Setenv restores original value after test completes
				}
			}

			// set env vars
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			result, err := parser.parsePortString(ctx, tt.input)

			if tt.hasError {
				if err == nil {
					t.Errorf("parsePortString(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("parsePortString(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result.Host != tt.expected.Host {
				t.Errorf("parsePortString(%q) Host = %d, want %d", tt.input, result.Host, tt.expected.Host)
			}

			if result.Container != tt.expected.Container {
				t.Errorf("parsePortString(%q) Container = %d, want %d", tt.input, result.Container, tt.expected.Container)
			}

			if result.Protocol != tt.expected.Protocol {
				t.Errorf("parsePortString(%q) Protocol = %q, want %q", tt.input, result.Protocol, tt.expected.Protocol)
			}

			if result.HostIP != tt.expected.HostIP {
				t.Errorf("parsePortString(%q) HostIP = %q, want %q", tt.input, result.HostIP, tt.expected.HostIP)
			}
		})
	}
}

// TestParseServicePorts tests the ParseServicePorts method
func TestParseServicePorts(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	parser := NewYamlComposeParser(testLogger)
	ctx := context.Background()

	tests := []struct {
		name     string
		service  map[string]interface{}
		expected []types.PortMapping
		hasError bool
	}{
		{
			name:     "empty service definition",
			service:  map[string]interface{}{},
			expected: []types.PortMapping{},
			hasError: false,
		},
		{
			name: "service without ports defined",
			service: map[string]interface{}{
				"image": "nginx",
			},
			expected: []types.PortMapping{},
			hasError: false,
		},
		{
			name: "multiple ports (string array)",
			service: map[string]interface{}{
				"ports": []interface{}{
					"3000:3000",
					"3001:3001",
					"3002:3002/udp",
				},
			},
			expected: []types.PortMapping{
				{Host: 3000, Container: 3000, Protocol: "tcp"},
				{Host: 3001, Container: 3001, Protocol: "tcp"},
				{Host: 3002, Container: 3002, Protocol: "udp"},
			},
			hasError: false,
		},
		{
			name: "integer format ports",
			service: map[string]interface{}{
				"ports": []interface{}{
					80,
					443,
				},
			},
			expected: []types.PortMapping{
				{Host: 0, Container: 80, Protocol: "tcp"},
				{Host: 0, Container: 443, Protocol: "tcp"},
			},
			hasError: false,
		},
		{
			name: "invalid ports format (not array)",
			service: map[string]interface{}{
				"ports": "8080:80",
			},
			expected: nil,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseServicePorts(ctx, tt.service)

			if tt.hasError {
				if err == nil {
					t.Errorf("ParseServicePorts() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseServicePorts() unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("ParseServicePorts() returned %d ports, want %d", len(result), len(tt.expected))
				return
			}

			for i, expected := range tt.expected {
				if result[i].Host != expected.Host {
					t.Errorf("Port[%d] Host = %d, want %d", i, result[i].Host, expected.Host)
				}
				if result[i].Container != expected.Container {
					t.Errorf("Port[%d] Container = %d, want %d", i, result[i].Container, expected.Container)
				}
				if result[i].Protocol != expected.Protocol {
					t.Errorf("Port[%d] Protocol = %q, want %q", i, result[i].Protocol, expected.Protocol)
				}
			}
		})
	}
}

// TestParsePortObject tests the parsePortObject method
func TestParsePortObject(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	parser := NewYamlComposeParser(testLogger)
	ctx := context.Background()

	tests := []struct {
		name     string
		portObj  map[string]interface{}
		expected *types.PortMapping
		hasError bool
	}{
		{
			name: "complete port object",
			portObj: map[string]interface{}{
				"target":    80,
				"published": 8080,
				"protocol":  "tcp",
				"host_ip":   "127.0.0.1",
			},
			expected: &types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "tcp",
				HostIP:    "127.0.0.1",
			},
			hasError: false,
		},
		{
			name: "minimal port object",
			portObj: map[string]interface{}{
				"target": 80,
			},
			expected: &types.PortMapping{
				Host:      0,
				Container: 80,
				Protocol:  "tcp",
			},
			hasError: false,
		},
		{
			name: "string format port numbers",
			portObj: map[string]interface{}{
				"target":    "80",
				"published": "8080",
			},
			expected: &types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "tcp",
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.parsePortObject(ctx, tt.portObj)

			if tt.hasError {
				if err == nil {
					t.Errorf("parsePortObject() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("parsePortObject() unexpected error: %v", err)
				return
			}

			if result.Host != tt.expected.Host {
				t.Errorf("Host = %d, want %d", result.Host, tt.expected.Host)
			}
			if result.Container != tt.expected.Container {
				t.Errorf("Container = %d, want %d", result.Container, tt.expected.Container)
			}
			if result.Protocol != tt.expected.Protocol {
				t.Errorf("Protocol = %q, want %q", result.Protocol, tt.expected.Protocol)
			}
			if result.HostIP != tt.expected.HostIP {
				t.Errorf("HostIP = %q, want %q", result.HostIP, tt.expected.HostIP)
			}
		})
	}
}

// TestParseEnvironment tests the parseEnvironment method
func TestParseEnvironment(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	parser := NewYamlComposeParser(testLogger)

	tests := []struct {
		name     string
		env      interface{}
		expected map[string]string
	}{
		{
			name: "array format - key=value",
			env: []interface{}{
				"NODE_ENV=production",
				"DEBUG=true",
			},
			expected: map[string]string{
				"NODE_ENV": "production",
				"DEBUG":    "true",
			},
		},
		{
			name: "array format - key only",
			env: []interface{}{
				"API_KEY",
				"SECRET",
			},
			expected: map[string]string{
				"API_KEY": "",
				"SECRET":  "",
			},
		},
		{
			name: "map format",
			env: map[string]interface{}{
				"NODE_ENV": "development",
				"PORT":     4000,
				"DEBUG":    "false",
			},
			expected: map[string]string{
				"NODE_ENV": "development",
				"PORT":     "4000",
				"DEBUG":    "false",
			},
		},
		{
			name:     "nil",
			env:      nil,
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseEnvironment(tt.env)

			if len(result) != len(tt.expected) {
				t.Errorf("parseEnvironment() returned %d entries, want %d", len(result), len(tt.expected))
			}

			for key, expectedValue := range tt.expected {
				if value, exists := result[key]; !exists {
					t.Errorf("parseEnvironment() missing key %q", key)
				} else if value != expectedValue {
					t.Errorf("parseEnvironment()[%q] = %q, want %q", key, value, expectedValue)
				}
			}
		})
	}
}

// TestParseDependsOn tests the parseDependsOn method
func TestParseDependsOn(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	parser := NewYamlComposeParser(testLogger)

	tests := []struct {
		name     string
		depends  interface{}
		expected []string
	}{
		{
			name: "array format",
			depends: []interface{}{
				"db",
				"cache",
				"queue",
			},
			expected: []string{"db", "cache", "queue"},
		},
		{
			name: "map format",
			depends: map[string]interface{}{
				"db": map[string]interface{}{
					"condition": "service_healthy",
				},
				"cache": map[string]interface{}{
					"condition": "service_started",
				},
			},
			expected: []string{"db", "cache"},
		},
		{
			name:     "nil",
			depends:  nil,
			expected: []string(nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseDependsOn(tt.depends)

			if len(result) != len(tt.expected) {
				t.Errorf("parseDependsOn() returned %d items, want %d", len(result), len(tt.expected))
				return
			}

			// map order is not guaranteed, use contains check
			for _, expected := range tt.expected {
				found := false
				for _, item := range result {
					if item == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("parseDependsOn() missing expected item %q", expected)
				}
			}
		})
	}
}

// TestParseNetworks tests the parseNetworks method
func TestParseNetworks(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	parser := NewYamlComposeParser(testLogger)

	tests := []struct {
		name     string
		networks interface{}
		expected map[string]types.ServiceNetwork
	}{
		{
			name: "array format - network names only",
			networks: []interface{}{
				"backend",
				"frontend",
			},
			expected: map[string]types.ServiceNetwork{
				"backend":  {},
				"frontend": {},
			},
		},
		{
			name: "map format - IPv4 address specified",
			networks: map[string]interface{}{
				"backend": map[string]interface{}{
					"ipv4_address": "172.28.0.5",
				},
			},
			expected: map[string]types.ServiceNetwork{
				"backend": {IPv4Address: "172.28.0.5"},
			},
		},
		{
			name: "map format - empty config",
			networks: map[string]interface{}{
				"backend": map[string]interface{}{},
			},
			expected: map[string]types.ServiceNetwork{
				"backend": {},
			},
		},
		{
			name:     "nil",
			networks: nil,
			expected: map[string]types.ServiceNetwork{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.parseNetworks(tt.networks)

			if len(result) != len(tt.expected) {
				t.Errorf("parseNetworks() returned %d networks, want %d", len(result), len(tt.expected))
				return
			}

			for name, expected := range tt.expected {
				network, exists := result[name]
				if !exists {
					t.Errorf("parseNetworks() missing network %q", name)
					continue
				}
				if network.IPv4Address != expected.IPv4Address {
					t.Errorf("Network[%q] IPv4Address = %q, want %q", name, network.IPv4Address, expected.IPv4Address)
				}
			}
		})
	}
}

// TestConvertToNetwork tests the convertToNetwork method
func TestConvertToNetwork(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	parser := NewYamlComposeParser(testLogger)
	ctx := context.Background()

	tests := []struct {
		name        string
		networkName string
		networkMap  map[string]interface{}
		expected    types.Network
		hasError    bool
	}{
		{
			name:        "empty network definition",
			networkName: "backend",
			networkMap:  map[string]interface{}{},
			expected: types.Network{
				Driver: "bridge",
				IPAM: types.IPAM{
					Driver: "default",
					Config: []types.IPAMConfig{},
				},
				Labels: map[string]string{},
			},
			hasError: false,
		},
		{
			name:        "network without ipam",
			networkName: "frontend",
			networkMap: map[string]interface{}{
				"driver": "bridge",
			},
			expected: types.Network{
				Driver: "bridge",
				IPAM: types.IPAM{
					Driver: "default",
					Config: []types.IPAMConfig{},
				},
				Labels: map[string]string{},
			},
			hasError: false,
		},
		{
			name:        "multiple subnets case",
			networkName: "multi-subnet",
			networkMap: map[string]interface{}{
				"driver": "bridge",
				"ipam": map[string]interface{}{
					"driver": "default",
					"config": []interface{}{
						map[string]interface{}{
							"subnet":  "172.28.0.0/16",
							"gateway": "172.28.0.1",
						},
						map[string]interface{}{
							"subnet":  "172.29.0.0/16",
							"gateway": "172.29.0.1",
						},
					},
				},
			},
			expected: types.Network{
				Driver: "bridge",
				IPAM: types.IPAM{
					Driver: "default",
					Config: []types.IPAMConfig{
						{Subnet: "172.28.0.0/16", Gateway: "172.28.0.1"},
						{Subnet: "172.29.0.0/16", Gateway: "172.29.0.1"},
					},
				},
				Labels: map[string]string{},
			},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.convertToNetwork(ctx, tt.networkName, tt.networkMap)

			if tt.hasError {
				if err == nil {
					t.Errorf("convertToNetwork() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("convertToNetwork() unexpected error: %v", err)
				return
			}

			if result.Driver != tt.expected.Driver {
				t.Errorf("Driver = %q, want %q", result.Driver, tt.expected.Driver)
			}

			if result.IPAM.Driver != tt.expected.IPAM.Driver {
				t.Errorf("IPAM.Driver = %q, want %q", result.IPAM.Driver, tt.expected.IPAM.Driver)
			}

			if len(result.IPAM.Config) != len(tt.expected.IPAM.Config) {
				t.Errorf("IPAM.Config has %d entries, want %d", len(result.IPAM.Config), len(tt.expected.IPAM.Config))
			}

			for i, expectedConfig := range tt.expected.IPAM.Config {
				if i >= len(result.IPAM.Config) {
					break
				}
				if result.IPAM.Config[i].Subnet != expectedConfig.Subnet {
					t.Errorf("IPAM.Config[%d].Subnet = %q, want %q", i, result.IPAM.Config[i].Subnet, expectedConfig.Subnet)
				}
				if result.IPAM.Config[i].Gateway != expectedConfig.Gateway {
					t.Errorf("IPAM.Config[%d].Gateway = %q, want %q", i, result.IPAM.Config[i].Gateway, expectedConfig.Gateway)
				}
			}
		})
	}
}

// TestParseComposeFileWithEdgeCases tests parsing edge case YAML files
func TestParseComposeFileWithEdgeCases(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	parser := NewYamlComposeParser(testLogger)
	ctx := context.Background()

	tests := []struct {
		name     string
		filepath string
		hasError bool
		checkFn  func(*testing.T, *types.ComposeConfig)
	}{
		{
			name:     "edge case file",
			filepath: "../../testdata/parser/edge_cases.yml",
			hasError: false,
			checkFn: func(t *testing.T, config *types.ComposeConfig) {
				if config == nil {
					t.Fatal("config is nil")
				}

				// check service count
				expectedServices := 9
				if len(config.Services) != expectedServices {
					t.Errorf("Services count = %d, want %d", len(config.Services), expectedServices)
				}

				// check port count for multi-ports service
				if service, exists := config.Services["multi-ports"]; exists {
					if len(service.Ports) != 3 {
						t.Errorf("multi-ports has %d ports, want 3", len(service.Ports))
					}
				} else {
					t.Error("multi-ports service not found")
				}

				// check network count
				expectedNetworks := 3
				if len(config.Networks) != expectedNetworks {
					t.Errorf("Networks count = %d, want %d", len(config.Networks), expectedNetworks)
				}

				// check IPAM config for multi-subnet network
				if network, exists := config.Networks["multi-subnet"]; exists {
					if len(network.IPAM.Config) != 2 {
						t.Errorf("multi-subnet has %d IPAM configs, want 2", len(network.IPAM.Config))
					}
				} else {
					t.Error("multi-subnet network not found")
				}
			},
		},
		{
			name:     "file with invalid port format",
			filepath: "../../testdata/parser/invalid.yml",
			hasError: true,
			checkFn:  nil,
		},
		{
			name:     "nonexistent file",
			filepath: "testdata/parser/nonexistent.yml",
			hasError: true,
			checkFn:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parser.ParseComposeFile(ctx, tt.filepath)

			if tt.hasError {
				if err == nil {
					t.Errorf("ParseComposeFile(%q) expected error, got nil", tt.filepath)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseComposeFile(%q) unexpected error: %v", tt.filepath, err)
				return
			}

			if tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}

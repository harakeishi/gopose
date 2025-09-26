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
			name:     "環境変数の展開 - デフォルト値あり",
			input:    "${APP_PORT:-3000}:80",
			envVars:  map[string]string{},
			expected: "3000:80",
		},
		{
			name:     "環境変数の展開 - 環境変数が設定されている",
			input:    "${APP_PORT:-3000}:80",
			envVars:  map[string]string{"APP_PORT": "4000"},
			expected: "4000:80",
		},
		{
			name:     "環境変数の展開 - ${VAR}形式",
			input:    "${PORT}:80",
			envVars:  map[string]string{"PORT": "5000"},
			expected: "5000:80",
		},
		{
			name:     "環境変数の展開 - $VAR形式",
			input:    "$PORT:80",
			envVars:  map[string]string{"PORT": "6000"},
			expected: "6000:80",
		},
		{
			name:     "環境変数の展開 - 複数の変数",
			input:    "${HOST_PORT:-8080}:${CONTAINER_PORT:-80}",
			envVars:  map[string]string{"HOST_PORT": "9090"},
			expected: "9090:80",
		},
		{
			name:     "環境変数なし",
			input:    "3000:80",
			envVars:  map[string]string{},
			expected: "3000:80",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数をセット
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
			name:    "環境変数展開 - デフォルト値",
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
			name:    "環境変数展開 - 環境変数が設定済み",
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
			name:    "通常のポート形式",
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
			name:    "IPアドレス付き環境変数",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 環境変数をセット
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
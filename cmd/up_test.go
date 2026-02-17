package cmd

import (
	"testing"

	"github.com/harakeishi/gopose/pkg/types"
)

func TestCreatePortConfig_WithReservedPorts(t *testing.T) {
	tests := []struct {
		name              string
		portRangeStr      string
		baseConfig        types.PortConfig
		expectedRange     types.PortRange
		expectedReserved  []int
		expectError       bool
	}{
		{
			name:         "preserve reserved ports from config",
			portRangeStr: "",
			baseConfig: types.PortConfig{
				Range:             types.PortRange{Start: 8000, End: 9000},
				Reserved:          []int{8080, 8443, 9000},
				ExcludePrivileged: true,
			},
			expectedRange:    types.PortRange{Start: 8000, End: 9000},
			expectedReserved: []int{8080, 8443, 9000},
			expectError:      false,
		},
		{
			name:         "preserve reserved ports when CLI overrides range",
			portRangeStr: "7000-7999",
			baseConfig: types.PortConfig{
				Range:             types.PortRange{Start: 8000, End: 9000},
				Reserved:          []int{7080, 7443},
				ExcludePrivileged: true,
			},
			expectedRange:    types.PortRange{Start: 7000, End: 7999},
			expectedReserved: []int{7080, 7443},
			expectError:      false,
		},
		{
			name:         "empty reserved ports list",
			portRangeStr: "",
			baseConfig: types.PortConfig{
				Range:             types.PortRange{Start: 8000, End: 9000},
				Reserved:          []int{},
				ExcludePrivileged: true,
			},
			expectedRange:    types.PortRange{Start: 8000, End: 9000},
			expectedReserved: []int{},
			expectError:      false,
		},
		{
			name:         "many reserved ports",
			portRangeStr: "",
			baseConfig: types.PortConfig{
				Range:             types.PortRange{Start: 8000, End: 8100},
				Reserved:          []int{8000, 8001, 8002, 8003, 8004, 8005, 8010, 8020, 8050},
				ExcludePrivileged: true,
			},
			expectedRange:    types.PortRange{Start: 8000, End: 8100},
			expectedReserved: []int{8000, 8001, 8002, 8003, 8004, 8005, 8010, 8020, 8050},
			expectError:      false,
		},
		{
			name:         "invalid port range format",
			portRangeStr: "invalid",
			baseConfig: types.PortConfig{
				Range:             types.PortRange{Start: 8000, End: 9000},
				Reserved:          []int{8080},
				ExcludePrivileged: true,
			},
			expectError: true,
		},
		{
			name:         "start port greater than end port",
			portRangeStr: "9000-8000",
			baseConfig: types.PortConfig{
				Range:             types.PortRange{Start: 8000, End: 9000},
				Reserved:          []int{8080},
				ExcludePrivileged: true,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := createPortConfig(tt.portRangeStr, tt.baseConfig)

			if tt.expectError {
				if err == nil {
					t.Errorf("createPortConfig() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("createPortConfig() unexpected error: %v", err)
				return
			}

			// verify port range
			if result.Range.Start != tt.expectedRange.Start {
				t.Errorf("createPortConfig() Range.Start = %d, want %d", result.Range.Start, tt.expectedRange.Start)
			}
			if result.Range.End != tt.expectedRange.End {
				t.Errorf("createPortConfig() Range.End = %d, want %d", result.Range.End, tt.expectedRange.End)
			}

			// verify reserved ports
			if len(result.Reserved) != len(tt.expectedReserved) {
				t.Errorf("createPortConfig() Reserved length = %d, want %d", len(result.Reserved), len(tt.expectedReserved))
				return
			}

			for i, reserved := range result.Reserved {
				if reserved != tt.expectedReserved[i] {
					t.Errorf("createPortConfig() Reserved[%d] = %d, want %d", i, reserved, tt.expectedReserved[i])
				}
			}

			// verify ExcludePrivileged is preserved
			if result.ExcludePrivileged != tt.baseConfig.ExcludePrivileged {
				t.Errorf("createPortConfig() ExcludePrivileged = %v, want %v", result.ExcludePrivileged, tt.baseConfig.ExcludePrivileged)
			}
		})
	}
}

func TestParsePortRange(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    types.PortRange
		expectError bool
	}{
		{
			name:        "valid port range",
			input:       "8000-9000",
			expected:    types.PortRange{Start: 8000, End: 9000},
			expectError: false,
		},
		{
			name:        "empty string uses default range",
			input:       "",
			expected:    types.PortRange{Start: 8000, End: 9999},
			expectError: false,
		},
		{
			name:        "range with spaces",
			input:       "7000 - 7999",
			expected:    types.PortRange{Start: 7000, End: 7999},
			expectError: false,
		},
		{
			name:        "invalid format",
			input:       "8000",
			expectError: true,
		},
		{
			name:        "invalid start port",
			input:       "abc-9000",
			expectError: true,
		},
		{
			name:        "invalid end port",
			input:       "8000-xyz",
			expectError: true,
		},
		{
			name:        "out of range port (start is 0 or less)",
			input:       "0-9000",
			expectError: true,
		},
		{
			name:        "out of range port (end exceeds 65535)",
			input:       "8000-70000",
			expectError: true,
		},
		{
			name:        "start greater than end",
			input:       "9000-8000",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePortRange(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("parsePortRange(%q) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("parsePortRange(%q) unexpected error: %v", tt.input, err)
				return
			}

			if result.Start != tt.expected.Start {
				t.Errorf("parsePortRange(%q) Start = %d, want %d", tt.input, result.Start, tt.expected.Start)
			}

			if result.End != tt.expected.End {
				t.Errorf("parsePortRange(%q) End = %d, want %d", tt.input, result.End, tt.expected.End)
			}
		})
	}
}

func TestCreatePortConfig_PortRangeEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		portRange   string
		expectError bool
	}{
		{"three segments", "8000-8500-9000", true},
		{"negative port string", "-1-9000", true},
		{"port 65535 upper bound", "65000-65535", false},
		{"single port range (start equals end)", "8000-8000", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := types.PortConfig{
				Range:    types.PortRange{Start: 8000, End: 9000},
				Reserved: []int{},
			}
			_, err := createPortConfig(tt.portRange, base)
			if tt.expectError && err == nil {
				t.Error("expected error")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

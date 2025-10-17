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
			name:         "設定ファイルの予約ポートを保持",
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
			name:         "CLIでポート範囲を上書きしても予約ポートは保持",
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
			name:         "空の予約ポートリスト",
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
			name:         "多数の予約ポート",
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
			name:         "無効なポート範囲形式",
			portRangeStr: "invalid",
			baseConfig: types.PortConfig{
				Range:             types.PortRange{Start: 8000, End: 9000},
				Reserved:          []int{8080},
				ExcludePrivileged: true,
			},
			expectError: true,
		},
		{
			name:         "開始ポートが終了ポートより大きい",
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

			// ポート範囲の確認
			if result.Range.Start != tt.expectedRange.Start {
				t.Errorf("createPortConfig() Range.Start = %d, want %d", result.Range.Start, tt.expectedRange.Start)
			}
			if result.Range.End != tt.expectedRange.End {
				t.Errorf("createPortConfig() Range.End = %d, want %d", result.Range.End, tt.expectedRange.End)
			}

			// 予約ポートの確認
			if len(result.Reserved) != len(tt.expectedReserved) {
				t.Errorf("createPortConfig() Reserved length = %d, want %d", len(result.Reserved), len(tt.expectedReserved))
				return
			}

			for i, reserved := range result.Reserved {
				if reserved != tt.expectedReserved[i] {
					t.Errorf("createPortConfig() Reserved[%d] = %d, want %d", i, reserved, tt.expectedReserved[i])
				}
			}

			// ExcludePrivilegedが保持されていることを確認
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
			name:        "有効なポート範囲",
			input:       "8000-9000",
			expected:    types.PortRange{Start: 8000, End: 9000},
			expectError: false,
		},
		{
			name:        "空文字列はデフォルト範囲",
			input:       "",
			expected:    types.PortRange{Start: 8000, End: 9999},
			expectError: false,
		},
		{
			name:        "スペース付きの範囲",
			input:       "7000 - 7999",
			expected:    types.PortRange{Start: 7000, End: 7999},
			expectError: false,
		},
		{
			name:        "無効な形式",
			input:       "8000",
			expectError: true,
		},
		{
			name:        "開始ポートが無効",
			input:       "abc-9000",
			expectError: true,
		},
		{
			name:        "終了ポートが無効",
			input:       "8000-xyz",
			expectError: true,
		},
		{
			name:        "範囲外のポート（開始が0以下）",
			input:       "0-9000",
			expectError: true,
		},
		{
			name:        "範囲外のポート（終了が65535超）",
			input:       "8000-70000",
			expectError: true,
		},
		{
			name:        "開始が終了より大きい",
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

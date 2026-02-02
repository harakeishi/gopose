package scanner

import (
	"context"
	"testing"

	"github.com/harakeishi/gopose/internal/errors"
	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

func TestValidatePort(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	validator := NewPortValidatorImpl(testLogger)
	ctx := context.Background()

	tests := []struct {
		name        string
		port        int
		expectError bool
	}{
		{
			name:        "有効なポート - 1",
			port:        1,
			expectError: false,
		},
		{
			name:        "有効なポート - 8080",
			port:        8080,
			expectError: false,
		},
		{
			name:        "有効なポート - 65535",
			port:        65535,
			expectError: false,
		},
		{
			name:        "無効なポート - 0",
			port:        0,
			expectError: true,
		},
		{
			name:        "無効なポート - 負の数",
			port:        -1,
			expectError: true,
		},
		{
			name:        "無効なポート - 65536",
			port:        65536,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePort(ctx, tt.port)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidatePort(%d) expected error, got nil", tt.port)
				}
				// エラーコードの確認
				var appErr *errors.AppError
				if e, ok := err.(*errors.AppError); ok {
					appErr = e
					if appErr.Code != errors.ErrPortRangeInvalid {
						t.Errorf("Expected error code ErrPortRangeInvalid, got %v", appErr.Code)
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePort(%d) unexpected error: %v", tt.port, err)
				}
			}
		})
	}
}

func TestValidatePortRange(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	validator := NewPortValidatorImpl(testLogger)
	ctx := context.Background()

	tests := []struct {
		name        string
		portRange   types.PortRange
		expectError bool
	}{
		{
			name: "有効な範囲 - 8000-9000",
			portRange: types.PortRange{
				Start: 8000,
				End:   9000,
			},
			expectError: false,
		},
		{
			name: "有効な範囲 - 全範囲",
			portRange: types.PortRange{
				Start: 1,
				End:   65535,
			},
			expectError: false,
		},
		{
			name: "無効な範囲 - 開始が終了より大きい",
			portRange: types.PortRange{
				Start: 9000,
				End:   8000,
			},
			expectError: true,
		},
		{
			name: "無効な範囲 - 開始ポートが0",
			portRange: types.PortRange{
				Start: 0,
				End:   9000,
			},
			expectError: true,
		},
		{
			name: "無効な範囲 - 終了ポートが65536",
			portRange: types.PortRange{
				Start: 8000,
				End:   65536,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePortRange(ctx, tt.portRange)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidatePortRange() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePortRange() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidatePortMapping(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	validator := NewPortValidatorImpl(testLogger)
	ctx := context.Background()

	tests := []struct {
		name        string
		mapping     types.PortMapping
		expectError bool
	}{
		{
			name: "有効なマッピング - TCP",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "tcp",
			},
			expectError: false,
		},
		{
			name: "有効なマッピング - UDP",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "udp",
			},
			expectError: false,
		},
		{
			name: "有効なマッピング - SCTP",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "sctp",
			},
			expectError: false,
		},
		{
			name: "有効なマッピング - プロトコル未指定",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "",
			},
			expectError: false,
		},
		{
			name: "無効なマッピング - 無効なホストポート",
			mapping: types.PortMapping{
				Host:      0,
				Container: 80,
				Protocol:  "tcp",
			},
			expectError: true,
		},
		{
			name: "無効なマッピング - 無効なコンテナポート",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 70000,
				Protocol:  "tcp",
			},
			expectError: true,
		},
		{
			name: "無効なマッピング - 無効なプロトコル",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "http",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePortMapping(ctx, tt.mapping)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidatePortMapping() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("ValidatePortMapping() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestScanAndValidate(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	// モックを作成
	mockDetector := &mockPortDetector{
		usedPorts: []int{8080, 8081, 8082},
	}

	mockAllocator := NewPortAllocatorImpl(mockDetector, testLogger)
	validator := NewPortValidatorImpl(testLogger)

	scanner := NewPortScannerImpl(mockDetector, mockAllocator, validator, testLogger)

	tests := []struct {
		name            string
		portRange       types.PortRange
		expectedUsed    int
		expectedAvail   int
		expectError     bool
	}{
		{
			name: "正常なスキャン",
			portRange: types.PortRange{
				Start: 8000,
				End:   8100,
			},
			expectedUsed:  3, // 8080, 8081, 8082
			expectedAvail: 98, // 101 - 3
			expectError:   false,
		},
		{
			name: "無効な範囲",
			portRange: types.PortRange{
				Start: 9000,
				End:   8000,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := scanner.ScanAndValidate(ctx, tt.portRange)

			if tt.expectError {
				if err == nil {
					t.Errorf("ScanAndValidate() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ScanAndValidate() unexpected error: %v", err)
			}

			if len(result.UsedPorts) != tt.expectedUsed {
				t.Errorf("UsedPorts count = %d, want %d", len(result.UsedPorts), tt.expectedUsed)
			}

			if len(result.AvailablePorts) != tt.expectedAvail {
				t.Errorf("AvailablePorts count = %d, want %d", len(result.AvailablePorts), tt.expectedAvail)
			}

			// ポート情報の検証
			if len(result.PortInfo) != len(result.UsedPorts) {
				t.Errorf("PortInfo count = %d, want %d", len(result.PortInfo), len(result.UsedPorts))
			}
		})
	}
}

func TestNewPortValidatorImpl(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})

	validator := NewPortValidatorImpl(testLogger)

	if validator == nil {
		t.Fatal("NewPortValidatorImpl() returned nil")
	}

	if validator.logger == nil {
		t.Error("logger is nil")
	}
}

func TestValidatePortEdgeCases(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	validator := NewPortValidatorImpl(testLogger)
	ctx := context.Background()

	tests := []struct {
		name        string
		port        int
		expectError bool
	}{
		{name: "最小有効ポート", port: 1, expectError: false},
		{name: "最大有効ポート", port: 65535, expectError: false},
		{name: "ゼロ（無効）", port: 0, expectError: true},
		{name: "負の数（無効）", port: -1, expectError: true},
		{name: "最大値超過（無効）", port: 65536, expectError: true},
		{name: "大きな負の数（無効）", port: -100, expectError: true},
		{name: "大幅超過（無効）", port: 100000, expectError: true},
		{name: "一般的なHTTPポート", port: 80, expectError: false},
		{name: "一般的なHTTPSポート", port: 443, expectError: false},
		{name: "開発用ポート", port: 3000, expectError: false},
		{name: "高ポート番号", port: 50000, expectError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePort(ctx, tt.port)

			if (err != nil) != tt.expectError {
				t.Errorf("ValidatePort(%d) error = %v, expectError = %v", tt.port, err, tt.expectError)
			}

			if err != nil && tt.expectError {
				var appErr *errors.AppError
				if e, ok := err.(*errors.AppError); ok {
					appErr = e
					if appErr.Code != errors.ErrPortRangeInvalid {
						t.Errorf("Expected error code ErrPortRangeInvalid, got %v", appErr.Code)
					}
				}
			}
		})
	}
}

func TestValidatePortRangeEdgeCases(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	validator := NewPortValidatorImpl(testLogger)
	ctx := context.Background()

	tests := []struct {
		name        string
		portRange   types.PortRange
		expectError bool
	}{
		{
			name:        "最小範囲（1ポート）",
			portRange:   types.PortRange{Start: 8080, End: 8080},
			expectError: false,
		},
		{
			name:        "全ポート範囲",
			portRange:   types.PortRange{Start: 1, End: 65535},
			expectError: false,
		},
		{
			name:        "逆順範囲（無効）",
			portRange:   types.PortRange{Start: 9000, End: 8000},
			expectError: true,
		},
		{
			name:        "開始ポートが0（無効）",
			portRange:   types.PortRange{Start: 0, End: 1000},
			expectError: true,
		},
		{
			name:        "終了ポートが最大値超過（無効）",
			portRange:   types.PortRange{Start: 1000, End: 65536},
			expectError: true,
		},
		{
			name:        "両方とも無効",
			portRange:   types.PortRange{Start: -1, End: 70000},
			expectError: true,
		},
		{
			name:        "標準的な範囲",
			portRange:   types.PortRange{Start: 8000, End: 9000},
			expectError: false,
		},
		{
			name:        "大きな範囲（警告が出る）",
			portRange:   types.PortRange{Start: 1, End: 20000},
			expectError: false, // 警告は出るがエラーではない
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePortRange(ctx, tt.portRange)

			if (err != nil) != tt.expectError {
				t.Errorf("ValidatePortRange() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}

func TestValidatePortMappingProtocols(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	validator := NewPortValidatorImpl(testLogger)
	ctx := context.Background()

	tests := []struct {
		name        string
		mapping     types.PortMapping
		expectError bool
	}{
		{
			name: "TCPプロトコル",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "tcp",
			},
			expectError: false,
		},
		{
			name: "UDPプロトコル",
			mapping: types.PortMapping{
				Host:      53,
				Container: 53,
				Protocol:  "udp",
			},
			expectError: false,
		},
		{
			name: "SCTPプロトコル",
			mapping: types.PortMapping{
				Host:      9899,
				Container: 9899,
				Protocol:  "sctp",
			},
			expectError: false,
		},
		{
			name: "プロトコル未指定（デフォルトTCP）",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "",
			},
			expectError: false,
		},
		{
			name: "無効なプロトコル",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "http",
			},
			expectError: true,
		},
		{
			name: "大文字プロトコル（無効）",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "TCP",
			},
			expectError: true,
		},
		{
			name: "無効なホストポートとプロトコル",
			mapping: types.PortMapping{
				Host:      0,
				Container: 80,
				Protocol:  "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidatePortMapping(ctx, tt.mapping)

			if (err != nil) != tt.expectError {
				t.Errorf("ValidatePortMapping() error = %v, expectError = %v", err, tt.expectError)
			}
		})
	}
}

func TestNewPortScannerImpl(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	mockDetector := &mockPortDetector{}
	mockAllocator := NewPortAllocatorImpl(mockDetector, testLogger)
	validator := NewPortValidatorImpl(testLogger)

	scanner := NewPortScannerImpl(mockDetector, mockAllocator, validator, testLogger)

	if scanner == nil {
		t.Fatal("NewPortScannerImpl() returned nil")
	}

	if scanner.PortDetector == nil {
		t.Error("PortDetector is nil")
	}

	if scanner.PortAllocator == nil {
		t.Error("PortAllocator is nil")
	}

	if scanner.PortValidator == nil {
		t.Error("PortValidator is nil")
	}

	if scanner.logger == nil {
		t.Error("logger is nil")
	}
}

func TestScanAndValidateWithDifferentRanges(t *testing.T) {
	factory := logger.NewStructuredLoggerFactory(false)
	testLogger, _ := factory.Create(types.LogConfig{})
	ctx := context.Background()

	tests := []struct {
		name            string
		usedPorts       []int
		portRange       types.PortRange
		expectedUsed    int
		expectedAvail   int
	}{
		{
			name:          "空の使用ポート",
			usedPorts:     []int{},
			portRange:     types.PortRange{Start: 8000, End: 8010},
			expectedUsed:  0,
			expectedAvail: 11,
		},
		{
			name:          "範囲外の使用ポート",
			usedPorts:     []int{7000, 7500},
			portRange:     types.PortRange{Start: 8000, End: 8010},
			expectedUsed:  0,
			expectedAvail: 11,
		},
		{
			name:          "範囲内の使用ポート",
			usedPorts:     []int{8000, 8005, 8010},
			portRange:     types.PortRange{Start: 8000, End: 8010},
			expectedUsed:  3,
			expectedAvail: 8,
		},
		{
			name:          "全て使用中",
			usedPorts:     []int{8000, 8001, 8002},
			portRange:     types.PortRange{Start: 8000, End: 8002},
			expectedUsed:  3,
			expectedAvail: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDetector := &mockPortDetector{usedPorts: tt.usedPorts}
			mockAllocator := NewPortAllocatorImpl(mockDetector, testLogger)
			validator := NewPortValidatorImpl(testLogger)
			scanner := NewPortScannerImpl(mockDetector, mockAllocator, validator, testLogger)

			result, err := scanner.ScanAndValidate(ctx, tt.portRange)

			if err != nil {
				t.Fatalf("ScanAndValidate() unexpected error: %v", err)
			}

			if len(result.UsedPorts) != tt.expectedUsed {
				t.Errorf("UsedPorts count = %d, want %d", len(result.UsedPorts), tt.expectedUsed)
			}

			if len(result.AvailablePorts) != tt.expectedAvail {
				t.Errorf("AvailablePorts count = %d, want %d", len(result.AvailablePorts), tt.expectedAvail)
			}

			// 使用ポートと利用可能ポートに重複がないことを確認
			usedMap := make(map[int]bool)
			for _, port := range result.UsedPorts {
				usedMap[port] = true
			}

			for _, port := range result.AvailablePorts {
				if usedMap[port] {
					t.Errorf("Port %d appears in both UsedPorts and AvailablePorts", port)
				}
			}
		})
	}
}

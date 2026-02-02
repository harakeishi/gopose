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

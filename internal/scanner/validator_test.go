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

	// Basic valid/invalid cases (boundary values tested in TestValidatePortEdgeCases)
	tests := []struct {
		name        string
		port        int
		expectError bool
	}{
		{
			name:        "valid port - 8080",
			port:        8080,
			expectError: false,
		},
		{
			name:        "invalid port - 0",
			port:        0,
			expectError: true,
		},
		{
			name:        "invalid port - negative number",
			port:        -1,
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
				// verify error code
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
			name: "valid range - 8000-9000",
			portRange: types.PortRange{
				Start: 8000,
				End:   9000,
			},
			expectError: false,
		},
		{
			name: "valid range - full range",
			portRange: types.PortRange{
				Start: 1,
				End:   65535,
			},
			expectError: false,
		},
		{
			name: "invalid range - start greater than end",
			portRange: types.PortRange{
				Start: 9000,
				End:   8000,
			},
			expectError: true,
		},
		{
			name: "invalid range - start port is 0",
			portRange: types.PortRange{
				Start: 0,
				End:   9000,
			},
			expectError: true,
		},
		{
			name: "invalid range - end port is 65536",
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
			name: "valid mapping - TCP",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "tcp",
			},
			expectError: false,
		},
		{
			name: "valid mapping - UDP",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "udp",
			},
			expectError: false,
		},
		{
			name: "valid mapping - SCTP",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "sctp",
			},
			expectError: false,
		},
		{
			name: "valid mapping - no protocol specified",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "",
			},
			expectError: false,
		},
		{
			name: "invalid mapping - invalid host port",
			mapping: types.PortMapping{
				Host:      0,
				Container: 80,
				Protocol:  "tcp",
			},
			expectError: true,
		},
		{
			name: "invalid mapping - invalid container port",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 70000,
				Protocol:  "tcp",
			},
			expectError: true,
		},
		{
			name: "invalid mapping - invalid protocol",
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

	// create mock
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
			name: "normal scan",
			portRange: types.PortRange{
				Start: 8000,
				End:   8100,
			},
			expectedUsed:  3, // 8080, 8081, 8082
			expectedAvail: 98, // 101 - 3
			expectError:   false,
		},
		{
			name: "invalid range",
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

			// verify port info
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
		{name: "minimum valid port", port: 1, expectError: false},
		{name: "maximum valid port", port: 65535, expectError: false},
		{name: "zero (invalid)", port: 0, expectError: true},
		{name: "negative number (invalid)", port: -1, expectError: true},
		{name: "exceeds maximum (invalid)", port: 65536, expectError: true},
		{name: "large negative number (invalid)", port: -100, expectError: true},
		{name: "far exceeds maximum (invalid)", port: 100000, expectError: true},
		{name: "common HTTP port", port: 80, expectError: false},
		{name: "common HTTPS port", port: 443, expectError: false},
		{name: "development port", port: 3000, expectError: false},
		{name: "high port number", port: 50000, expectError: false},
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
			name:        "minimum range (1 port)",
			portRange:   types.PortRange{Start: 8080, End: 8080},
			expectError: false,
		},
		{
			name:        "full port range",
			portRange:   types.PortRange{Start: 1, End: 65535},
			expectError: false,
		},
		{
			name:        "reversed range (invalid)",
			portRange:   types.PortRange{Start: 9000, End: 8000},
			expectError: true,
		},
		{
			name:        "start port is 0 (invalid)",
			portRange:   types.PortRange{Start: 0, End: 1000},
			expectError: true,
		},
		{
			name:        "end port exceeds maximum (invalid)",
			portRange:   types.PortRange{Start: 1000, End: 65536},
			expectError: true,
		},
		{
			name:        "both invalid",
			portRange:   types.PortRange{Start: -1, End: 70000},
			expectError: true,
		},
		{
			name:        "standard range",
			portRange:   types.PortRange{Start: 8000, End: 9000},
			expectError: false,
		},
		{
			name:        "large range (produces warning)",
			portRange:   types.PortRange{Start: 1, End: 20000},
			expectError: false, // produces warning but not error
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
			name: "TCP protocol",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "tcp",
			},
			expectError: false,
		},
		{
			name: "UDP protocol",
			mapping: types.PortMapping{
				Host:      53,
				Container: 53,
				Protocol:  "udp",
			},
			expectError: false,
		},
		{
			name: "SCTP protocol",
			mapping: types.PortMapping{
				Host:      9899,
				Container: 9899,
				Protocol:  "sctp",
			},
			expectError: false,
		},
		{
			name: "no protocol (defaults to TCP)",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "",
			},
			expectError: false,
		},
		{
			name: "invalid protocol",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "http",
			},
			expectError: true,
		},
		{
			name: "uppercase protocol (invalid)",
			mapping: types.PortMapping{
				Host:      8080,
				Container: 80,
				Protocol:  "TCP",
			},
			expectError: true,
		},
		{
			name: "invalid host port and protocol",
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
			name:          "empty used ports",
			usedPorts:     []int{},
			portRange:     types.PortRange{Start: 8000, End: 8010},
			expectedUsed:  0,
			expectedAvail: 11,
		},
		{
			name:          "used ports outside range",
			usedPorts:     []int{7000, 7500},
			portRange:     types.PortRange{Start: 8000, End: 8010},
			expectedUsed:  0,
			expectedAvail: 11,
		},
		{
			name:          "used ports within range",
			usedPorts:     []int{8000, 8005, 8010},
			portRange:     types.PortRange{Start: 8000, End: 8010},
			expectedUsed:  3,
			expectedAvail: 8,
		},
		{
			name:          "all ports in use",
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

			// verify no overlap between used and available ports
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

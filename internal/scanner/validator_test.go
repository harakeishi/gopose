package scanner

import (
	"context"
	"testing"

	"github.com/harakeishi/gopose/internal/errors"
	"github.com/harakeishi/gopose/internal/testutil"
	"github.com/harakeishi/gopose/pkg/types"
)

func TestValidatePort(t *testing.T) {
	testLogger := testutil.NewTestLogger()
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
	testLogger := testutil.NewTestLogger()
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
	testLogger := testutil.NewTestLogger()
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

func TestNewPortValidatorImpl(t *testing.T) {
	testLogger := testutil.NewTestLogger()

	validator := NewPortValidatorImpl(testLogger)

	if validator == nil {
		t.Fatal("NewPortValidatorImpl() returned nil")
	}

	if validator.logger == nil {
		t.Error("logger is nil")
	}
}

func TestValidatePortEdgeCases(t *testing.T) {
	testLogger := testutil.NewTestLogger()
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
	testLogger := testutil.NewTestLogger()
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
	testLogger := testutil.NewTestLogger()
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


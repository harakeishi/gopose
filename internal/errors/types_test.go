package errors

import (
	"errors"
	"testing"

	"github.com/harakeishi/gopose/pkg/types"
)

func TestAppErrorError(t *testing.T) {
	tests := []struct {
		name     string
		appError *AppError
		expected string
	}{
		{
			name: "Error without cause",
			appError: &AppError{
				Code:    ErrFileNotFound,
				Message: "File not found",
			},
			expected: "File not found",
		},
		{
			name: "Error with cause",
			appError: &AppError{
				Code:    ErrFileReadFailed,
				Message: "Failed to read file",
				Cause:   errors.New("permission denied"),
			},
			expected: "Failed to read file: permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.appError.Error()
			if result != tt.expected {
				t.Errorf("AppError.Error() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAppErrorUnwrap(t *testing.T) {
	cause := errors.New("original error")
	appError := &AppError{
		Code:    ErrInternalError,
		Message: "Internal error",
		Cause:   cause,
	}

	result := appError.Unwrap()
	if result != cause {
		t.Errorf("AppError.Unwrap() = %v, want %v", result, cause)
	}
}

func TestAppErrorGetSeverity(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected types.Severity
	}{
		{
			name:     "File error severity",
			code:     ErrFileNotFound,
			expected: types.SeverityError,
		},
		{
			name:     "Port error severity",
			code:     ErrPortUnavailable,
			expected: types.SeverityWarning,
		},
		{
			name:     "Docker error severity",
			code:     ErrDockerNotFound,
			expected: types.SeverityError,
		},
		{
			name:     "Config error severity",
			code:     ErrConfigInvalid,
			expected: types.SeverityError,
		},
		{
			name:     "Process error severity",
			code:     ErrProcessNotFound,
			expected: types.SeverityWarning,
		},
		{
			name:     "Unknown error severity",
			code:     ErrUnknown,
			expected: types.SeverityError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appError := &AppError{
				Code:    tt.code,
				Message: "Test error",
			}
			result := appError.GetSeverity()
			if result != tt.expected {
				t.Errorf("AppError.GetSeverity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAppErrorIsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected bool
	}{
		{
			name:     "Port unavailable is retryable",
			code:     ErrPortUnavailable,
			expected: true,
		},
		{
			name:     "Process not found is retryable",
			code:     ErrProcessNotFound,
			expected: true,
		},
		{
			name:     "File permission is retryable",
			code:     ErrFilePermission,
			expected: true,
		},
		{
			name:     "File not found is not retryable",
			code:     ErrFileNotFound,
			expected: false,
		},
		{
			name:     "Unknown error is not retryable",
			code:     ErrUnknown,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appError := &AppError{
				Code:    tt.code,
				Message: "Test error",
			}
			result := appError.IsRetryable()
			if result != tt.expected {
				t.Errorf("AppError.IsRetryable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAppErrorWithField(t *testing.T) {
	appError := &AppError{
		Code:    ErrFileNotFound,
		Message: "File not found",
	}

	result := appError.WithField("path", "/test/path")

	if result.Fields == nil {
		t.Error("AppError.WithField() should initialize Fields map")
	}

	if result.Fields["path"] != "/test/path" {
		t.Errorf("AppError.WithField() field value = %v, want %v", result.Fields["path"], "/test/path")
	}

	// Add another field
	result.WithField("type", "yaml")

	if len(result.Fields) != 2 {
		t.Errorf("AppError.WithField() field count = %v, want 2", len(result.Fields))
	}
}

func TestAppErrorWithCause(t *testing.T) {
	cause := errors.New("original error")
	appError := &AppError{
		Code:    ErrFileReadFailed,
		Message: "Failed to read file",
	}

	result := appError.WithCause(cause)

	if result.Cause != cause {
		t.Errorf("AppError.WithCause() = %v, want %v", result.Cause, cause)
	}
}

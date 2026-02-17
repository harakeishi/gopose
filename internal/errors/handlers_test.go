package errors

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestAppErrorHandler_Handle_NilError(t *testing.T) {
	handler := NewAppErrorHandler()
	ctx := context.Background()

	result := handler.Handle(ctx, nil)
	if result != nil {
		t.Errorf("Handle(nil) = %v, want nil", result)
	}
}

func TestAppErrorHandler_Handle_AppError(t *testing.T) {
	handler := NewAppErrorHandler()
	ctx := context.Background()

	original := &AppError{
		Code:    ErrFileNotFound,
		Message: "test file not found",
	}

	result := handler.Handle(ctx, original)

	appErr, ok := result.(*AppError)
	if !ok {
		t.Fatalf("Handle(AppError) returned type %T, want *AppError", result)
	}

	if appErr != original {
		t.Errorf("Handle(AppError) returned different instance, want same instance")
	}

	if appErr.Code != ErrFileNotFound {
		t.Errorf("Handle(AppError).Code = %v, want %v", appErr.Code, ErrFileNotFound)
	}
}

func TestAppErrorHandler_Handle_ConvertStandardErrors(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode ErrorCode
	}{
		{
			name:         "os.ErrNotExist converts to ErrFileNotFound",
			err:          os.ErrNotExist,
			expectedCode: ErrFileNotFound,
		},
		{
			name:         "os.ErrPermission converts to ErrFilePermission",
			err:          os.ErrPermission,
			expectedCode: ErrFilePermission,
		},
		{
			name:         "wrapped ECONNREFUSED converts to ErrDockerAPIFailed",
			err:          fmt.Errorf("connection error: %w", syscall.ECONNREFUSED),
			expectedCode: ErrDockerAPIFailed,
		},
		{
			name:         "wrapped EADDRINUSE converts to ErrPortUnavailable",
			err:          fmt.Errorf("bind error: %w", syscall.EADDRINUSE),
			expectedCode: ErrPortUnavailable,
		},
		{
			name:         "unknown error converts to ErrUnknown",
			err:          fmt.Errorf("some unknown error"),
			expectedCode: ErrUnknown,
		},
	}

	handler := NewAppErrorHandler()
	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.Handle(ctx, tt.err)

			appErr, ok := result.(*AppError)
			if !ok {
				t.Fatalf("Handle() returned type %T, want *AppError", result)
			}

			if appErr.Code != tt.expectedCode {
				t.Errorf("Handle().Code = %v, want %v", appErr.Code, tt.expectedCode)
			}

			if appErr.Cause == nil {
				t.Error("Handle().Cause should not be nil for converted errors")
			}
		})
	}
}

func TestAppErrorHandler_IsRetryable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name: "AppError with ErrPortUnavailable is retryable",
			err: &AppError{
				Code:    ErrPortUnavailable,
				Message: "port unavailable",
			},
			expected: true,
		},
		{
			name: "AppError with ErrProcessNotFound is retryable",
			err: &AppError{
				Code:    ErrProcessNotFound,
				Message: "process not found",
			},
			expected: true,
		},
		{
			name: "AppError with ErrFilePermission is retryable",
			err: &AppError{
				Code:    ErrFilePermission,
				Message: "permission denied",
			},
			expected: true,
		},
		{
			name: "AppError with ErrFileNotFound is not retryable",
			err: &AppError{
				Code:    ErrFileNotFound,
				Message: "file not found",
			},
			expected: false,
		},
		{
			name: "AppError with ErrUnknown is not retryable",
			err: &AppError{
				Code:    ErrUnknown,
				Message: "unknown error",
			},
			expected: false,
		},
		{
			name: "AppError with ErrConfigInvalid is not retryable",
			err: &AppError{
				Code:    ErrConfigInvalid,
				Message: "config invalid",
			},
			expected: false,
		},
		{
			name:     "syscall ECONNREFUSED is retryable",
			err:      syscall.ECONNREFUSED,
			expected: true,
		},
		{
			name:     "syscall ETIMEDOUT is retryable",
			err:      syscall.ETIMEDOUT,
			expected: true,
		},
		{
			name:     "syscall ENOENT is retryable",
			err:      syscall.ENOENT,
			expected: true,
		},
		{
			name:     "wrapped ECONNREFUSED is retryable",
			err:      fmt.Errorf("wrapped: %w", syscall.ECONNREFUSED),
			expected: true,
		},
		{
			name:     "generic error is not retryable",
			err:      fmt.Errorf("some generic error"),
			expected: false,
		},
	}

	handler := NewAppErrorHandler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestAppErrorHandler_GetRetryConfig(t *testing.T) {
	tests := []struct {
		name              string
		err               error
		expectedMaxRetry  int
		expectedBaseDelay time.Duration
		expectedMaxDelay  time.Duration
		expectedBackoff   float64
	}{
		{
			name: "ErrPortUnavailable returns port retry config",
			err: &AppError{
				Code:    ErrPortUnavailable,
				Message: "port unavailable",
			},
			expectedMaxRetry:  5,
			expectedBaseDelay: 200 * time.Millisecond,
			expectedMaxDelay:  2 * time.Second,
			expectedBackoff:   1.5,
		},
		{
			name: "ErrPortScanFailed returns port retry config",
			err: &AppError{
				Code:    ErrPortScanFailed,
				Message: "port scan failed",
			},
			expectedMaxRetry:  5,
			expectedBaseDelay: 200 * time.Millisecond,
			expectedMaxDelay:  2 * time.Second,
			expectedBackoff:   1.5,
		},
		{
			name: "ErrProcessNotFound returns process retry config",
			err: &AppError{
				Code:    ErrProcessNotFound,
				Message: "process not found",
			},
			expectedMaxRetry:  3,
			expectedBaseDelay: 500 * time.Millisecond,
			expectedMaxDelay:  5 * time.Second,
			expectedBackoff:   2.0,
		},
		{
			name: "ErrDockerAPIFailed returns process retry config",
			err: &AppError{
				Code:    ErrDockerAPIFailed,
				Message: "docker api failed",
			},
			expectedMaxRetry:  3,
			expectedBaseDelay: 500 * time.Millisecond,
			expectedMaxDelay:  5 * time.Second,
			expectedBackoff:   2.0,
		},
		{
			name: "ErrFileNotFound returns default config",
			err: &AppError{
				Code:    ErrFileNotFound,
				Message: "file not found",
			},
			expectedMaxRetry:  3,
			expectedBaseDelay: 100 * time.Millisecond,
			expectedMaxDelay:  5 * time.Second,
			expectedBackoff:   2.0,
		},
		{
			name: "ErrUnknown returns default config",
			err: &AppError{
				Code:    ErrUnknown,
				Message: "unknown",
			},
			expectedMaxRetry:  3,
			expectedBaseDelay: 100 * time.Millisecond,
			expectedMaxDelay:  5 * time.Second,
			expectedBackoff:   2.0,
		},
		{
			name:              "non-AppError returns default config",
			err:               fmt.Errorf("generic error"),
			expectedMaxRetry:  3,
			expectedBaseDelay: 100 * time.Millisecond,
			expectedMaxDelay:  5 * time.Second,
			expectedBackoff:   2.0,
		},
	}

	handler := NewAppErrorHandler()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := handler.GetRetryConfig(tt.err)

			if config.MaxRetries != tt.expectedMaxRetry {
				t.Errorf("GetRetryConfig().MaxRetries = %v, want %v", config.MaxRetries, tt.expectedMaxRetry)
			}

			if config.BaseDelay != tt.expectedBaseDelay {
				t.Errorf("GetRetryConfig().BaseDelay = %v, want %v", config.BaseDelay, tt.expectedBaseDelay)
			}

			if config.MaxDelay != tt.expectedMaxDelay {
				t.Errorf("GetRetryConfig().MaxDelay = %v, want %v", config.MaxDelay, tt.expectedMaxDelay)
			}

			if config.BackoffFactor != tt.expectedBackoff {
				t.Errorf("GetRetryConfig().BackoffFactor = %v, want %v", config.BackoffFactor, tt.expectedBackoff)
			}
		})
	}
}

func TestNewAppErrorHandler_DefaultConfig(t *testing.T) {
	handler := NewAppErrorHandler()

	if handler == nil {
		t.Fatal("NewAppErrorHandler() returned nil")
	}

	config := handler.defaultRetryConfig

	if config.MaxRetries != 3 {
		t.Errorf("defaultRetryConfig.MaxRetries = %v, want 3", config.MaxRetries)
	}

	if config.BaseDelay != 100*time.Millisecond {
		t.Errorf("defaultRetryConfig.BaseDelay = %v, want %v", config.BaseDelay, 100*time.Millisecond)
	}

	if config.MaxDelay != 5*time.Second {
		t.Errorf("defaultRetryConfig.MaxDelay = %v, want %v", config.MaxDelay, 5*time.Second)
	}

	if config.BackoffFactor != 2.0 {
		t.Errorf("defaultRetryConfig.BackoffFactor = %v, want 2.0", config.BackoffFactor)
	}
}

func TestErrorFactoryFunctions(t *testing.T) {
	t.Run("NewFileNotFoundError", func(t *testing.T) {
		path := "/tmp/test/missing.yml"
		err := NewFileNotFoundError(path)

		if err.Code != ErrFileNotFound {
			t.Errorf("Code = %v, want %v", err.Code, ErrFileNotFound)
		}

		if err.Fields == nil {
			t.Fatal("Fields should not be nil")
		}

		if err.Fields["path"] != path {
			t.Errorf("Fields[\"path\"] = %v, want %v", err.Fields["path"], path)
		}

		if err.Message == "" {
			t.Error("Message should not be empty")
		}
	})

	t.Run("NewPortConflictError", func(t *testing.T) {
		port := 8080
		service := "web"
		err := NewPortConflictError(port, service)

		if err.Code != ErrPortConflict {
			t.Errorf("Code = %v, want %v", err.Code, ErrPortConflict)
		}

		if err.Fields == nil {
			t.Fatal("Fields should not be nil")
		}

		if err.Fields["port"] != port {
			t.Errorf("Fields[\"port\"] = %v, want %v", err.Fields["port"], port)
		}

		if err.Fields["service"] != service {
			t.Errorf("Fields[\"service\"] = %v, want %v", err.Fields["service"], service)
		}

		if err.Message == "" {
			t.Error("Message should not be empty")
		}
	})

	t.Run("NewConfigInvalidError", func(t *testing.T) {
		field := "port_range"
		value := "invalid"
		err := NewConfigInvalidError(field, value)

		if err.Code != ErrConfigInvalid {
			t.Errorf("Code = %v, want %v", err.Code, ErrConfigInvalid)
		}

		if err.Fields == nil {
			t.Fatal("Fields should not be nil")
		}

		if err.Fields["field"] != field {
			t.Errorf("Fields[\"field\"] = %v, want %v", err.Fields["field"], field)
		}

		if err.Fields["value"] != value {
			t.Errorf("Fields[\"value\"] = %v, want %v", err.Fields["value"], value)
		}

		if err.Message == "" {
			t.Error("Message should not be empty")
		}
	})

	t.Run("NewDockerComposeInvalidError", func(t *testing.T) {
		path := "/tmp/docker-compose.yml"
		reason := "invalid syntax at line 10"
		err := NewDockerComposeInvalidError(path, reason)

		if err.Code != ErrComposeInvalid {
			t.Errorf("Code = %v, want %v", err.Code, ErrComposeInvalid)
		}

		if err.Fields == nil {
			t.Fatal("Fields should not be nil")
		}

		if err.Fields["path"] != path {
			t.Errorf("Fields[\"path\"] = %v, want %v", err.Fields["path"], path)
		}

		if err.Fields["reason"] != reason {
			t.Errorf("Fields[\"reason\"] = %v, want %v", err.Fields["reason"], reason)
		}

		if err.Message == "" {
			t.Error("Message should not be empty")
		}
	})
}

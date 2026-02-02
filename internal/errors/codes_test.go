package errors

import "testing"

func TestErrorCodeCategory(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected ErrorCategory
	}{
		{
			name:     "File error category",
			code:     ErrFileNotFound,
			expected: ErrorCategoryFile,
		},
		{
			name:     "Port error category",
			code:     ErrPortUnavailable,
			expected: ErrorCategoryPort,
		},
		{
			name:     "Docker error category",
			code:     ErrDockerNotFound,
			expected: ErrorCategoryDocker,
		},
		{
			name:     "Compose error category",
			code:     ErrComposeInvalid,
			expected: ErrorCategoryDocker,
		},
		{
			name:     "Config error category",
			code:     ErrConfigInvalid,
			expected: ErrorCategoryConfig,
		},
		{
			name:     "Process error category",
			code:     ErrProcessNotFound,
			expected: ErrorCategoryProcess,
		},
		{
			name:     "Unknown error category",
			code:     ErrUnknown,
			expected: ErrorCategoryUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.code.Category()
			if result != tt.expected {
				t.Errorf("ErrorCode.Category() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestErrorCodeString(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected string
	}{
		{
			name:     "File not found",
			code:     ErrFileNotFound,
			expected: "FILE_NOT_FOUND",
		},
		{
			name:     "Port unavailable",
			code:     ErrPortUnavailable,
			expected: "PORT_UNAVAILABLE",
		},
		{
			name:     "Unknown error",
			code:     ErrUnknown,
			expected: "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.code.String()
			if result != tt.expected {
				t.Errorf("ErrorCode.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

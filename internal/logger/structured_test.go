package logger

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/harakeishi/gopose/pkg/types"
)

func TestNewStructuredLoggerFactory(t *testing.T) {
	tests := []struct {
		name     string
		detailed bool
	}{
		{
			name:     "detailed logger factory",
			detailed: true,
		},
		{
			name:     "simple logger factory",
			detailed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewStructuredLoggerFactory(tt.detailed)
			if factory == nil {
				t.Fatal("NewStructuredLoggerFactory() returned nil")
			}
			if factory.detailed != tt.detailed {
				t.Errorf("NewStructuredLoggerFactory() detailed = %v, want %v", factory.detailed, tt.detailed)
			}
		})
	}
}

func TestStructuredLoggerFactoryCreate(t *testing.T) {
	factory := NewStructuredLoggerFactory(false)
	config := types.LogConfig{
		Level:  "info",
		Format: "text",
	}

	logger, err := factory.Create(config)
	if err != nil {
		t.Errorf("StructuredLoggerFactory.Create() error = %v", err)
	}
	if logger == nil {
		t.Error("StructuredLoggerFactory.Create() returned nil logger")
	}
}

func TestStructuredLoggerFactoryCreateWithName(t *testing.T) {
	factory := NewStructuredLoggerFactory(true)
	config := types.LogConfig{
		Level:  "debug",
		Format: "json",
	}

	logger, err := factory.CreateWithName("test-component", config)
	if err != nil {
		t.Errorf("StructuredLoggerFactory.CreateWithName() error = %v", err)
	}
	if logger == nil {
		t.Error("StructuredLoggerFactory.CreateWithName() returned nil logger")
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name          string
		levelStr      string
		expectedLevel slog.Level
	}{
		{name: "debug level", levelStr: "debug", expectedLevel: slog.LevelDebug},
		{name: "info level", levelStr: "info", expectedLevel: slog.LevelInfo},
		{name: "warn level", levelStr: "warn", expectedLevel: slog.LevelWarn},
		{name: "error level", levelStr: "error", expectedLevel: slog.LevelError},
		{name: "unknown level defaults to info", levelStr: "unknown", expectedLevel: slog.LevelInfo},
		{name: "empty string defaults to info", levelStr: "", expectedLevel: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := parseLogLevel(tt.levelStr)
			if level != tt.expectedLevel {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.levelStr, level, tt.expectedLevel)
			}
		})
	}
}

func TestStructuredLoggerWithField(t *testing.T) {
	factory := NewStructuredLoggerFactory(false)
	config := DefaultConfig()
	logger, _ := factory.Create(config)

	newLogger := logger.WithField("test_key", "test_value")
	if newLogger == nil {
		t.Error("WithField() returned nil")
	}

	// Ensure original logger is not modified
	structLogger, ok := logger.(*StructuredLogger)
	if !ok {
		t.Fatal("Logger is not StructuredLogger")
	}
	if len(structLogger.fields) != 0 {
		t.Error("Original logger was modified")
	}
}

func TestStructuredLoggerWithFields(t *testing.T) {
	factory := NewStructuredLoggerFactory(false)
	config := DefaultConfig()
	logger, _ := factory.Create(config)

	fields := []types.Field{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: 123},
	}

	newLogger := logger.WithFields(fields...)
	if newLogger == nil {
		t.Error("WithFields() returned nil")
	}

	structLogger, ok := newLogger.(*StructuredLogger)
	if !ok {
		t.Fatal("Logger is not StructuredLogger")
	}
	if len(structLogger.fields) != 2 {
		t.Errorf("WithFields() added %d fields, want 2", len(structLogger.fields))
	}
}

func TestStructuredLoggerWithError(t *testing.T) {
	factory := NewStructuredLoggerFactory(false)
	config := DefaultConfig()
	logger, _ := factory.Create(config)

	err := errors.New("test error")
	newLogger := logger.WithError(err)
	if newLogger == nil {
		t.Error("WithError() returned nil")
	}

	structLogger, ok := newLogger.(*StructuredLogger)
	if !ok {
		t.Fatal("Logger is not StructuredLogger")
	}
	if structLogger.err != err {
		t.Error("WithError() did not set error correctly")
	}
}

func TestStructuredLoggerLogging(t *testing.T) {
	// Test with detailed=false to avoid complex slog output verification
	factory := NewStructuredLoggerFactory(false)
	config := DefaultConfig()
	logger, _ := factory.Create(config)

	ctx := context.Background()

	// These should not panic
	t.Run("Debug logging", func(t *testing.T) {
		logger.Debug(ctx, "debug message", types.Field{Key: "test", Value: "value"})
	})

	t.Run("Info logging", func(t *testing.T) {
		logger.Info(ctx, "info message")
	})

	t.Run("Warn logging", func(t *testing.T) {
		logger.Warn(ctx, "warn message")
	})

	t.Run("Error logging", func(t *testing.T) {
		logger.Error(ctx, "error message", errors.New("test error"))
	})
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Level != "warn" {
		t.Errorf("DefaultConfig().Level = %v, want warn", config.Level)
	}
	if config.Format != "text" {
		t.Errorf("DefaultConfig().Format = %v, want text", config.Format)
	}
	if config.File != "" {
		t.Errorf("DefaultConfig().File = %v, want empty string", config.File)
	}
	if config.MaxSize != 100 {
		t.Errorf("DefaultConfig().MaxSize = %v, want 100", config.MaxSize)
	}
	if config.MaxAge != 30 {
		t.Errorf("DefaultConfig().MaxAge = %v, want 30", config.MaxAge)
	}
	if !config.Compress {
		t.Error("DefaultConfig().Compress = false, want true")
	}
}

func TestFormatJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		contains string
	}{
		{
			name:     "simple map",
			input:    map[string]string{"key": "value"},
			contains: "key",
		},
		{
			name:     "struct",
			input:    types.Field{Key: "test", Value: "value"},
			contains: "key",
		},
		{
			name:     "string",
			input:    "test string",
			contains: "test string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatJSON(tt.input)
			if result == "" {
				t.Error("FormatJSON() returned empty string")
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("FormatJSON() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

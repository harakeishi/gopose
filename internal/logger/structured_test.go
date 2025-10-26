package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/harakeishi/gopose/pkg/types"
)

func TestParseLogLevel(t *testing.T) {
	tests := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
		"other": slog.LevelInfo,
	}
	for input, want := range tests {
		if got := parseLogLevel(input); got != want {
			t.Fatalf("parseLogLevel(%s)=%v want %v", input, got, want)
		}
	}
}

func TestStructuredLoggerWithFields(t *testing.T) {
	factory := NewStructuredLoggerFactory(true)

	// Redirect stdout so the structured logger writes to a controllable buffer
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() {
		w.Close()
		r.Close()
		os.Stdout = origStdout
	})

	cfg := types.LogConfig{Level: "debug", Format: "json"}
	log, err := factory.CreateWithName("test", cfg)
	if err != nil {
		t.Fatalf("CreateWithName error: %v", err)
	}

	base := log.WithField("request_id", "abc").WithFields(types.Field{Key: "user", Value: "demo"})
	if structured, ok := base.(*StructuredLogger); ok {
		structured.detailed = true
	}
	base.Error(context.Background(), "message", os.ErrNotExist)

	w.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading redirected stdout: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "message") || !strings.Contains(content, "request_id") {
		t.Fatalf("expected structured content, got %s", content)
	}
}

func TestStructuredLoggerWithError(t *testing.T) {
	factory := NewStructuredLoggerFactory(true)
	logger, err := factory.Create(types.LogConfig{Level: "info"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	withErr := logger.WithError(os.ErrClosed)
	l, ok := withErr.(*StructuredLogger)
	if !ok || l.err == nil {
		t.Fatalf("expected stored error")
	}
}

func TestFormatJSON(t *testing.T) {
	value := map[string]string{"key": "value"}
	if got := FormatJSON(value); !strings.Contains(got, "\"key\"") {
		t.Fatalf("unexpected json: %s", got)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Level != "info" || cfg.Format != "text" {
		t.Fatalf("unexpected default config: %#v", cfg)
	}
}

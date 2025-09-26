package errors

import (
    "context"
    goerrors "errors"
    "fmt"
    "os"
    "syscall"
    "testing"
    "time"

    "github.com/harakeishi/gopose/internal/logger"
)

func TestNewAppErrorHandlerDefaults(t *testing.T) {
    handler := NewAppErrorHandler()
    if handler.defaultRetryConfig.MaxRetries == 0 {
        t.Fatalf("expected default retry config to be populated")
    }
}

func TestAppErrorHandlerHandle(t *testing.T) {
    handler := NewAppErrorHandler()
    ctx := context.Background()

    if err := handler.Handle(ctx, nil); err != nil {
        t.Fatalf("handling nil should return nil")
    }

    appErr := &AppError{Code: ErrUnknown, Message: "boom"}
    if got := handler.Handle(ctx, appErr); got != appErr {
        t.Fatalf("expected handler to return original AppError")
    }

    converted := handler.Handle(ctx, os.ErrNotExist)
    if convertedErr, ok := converted.(*AppError); !ok || convertedErr.Code != ErrFileNotFound {
        t.Fatalf("expected file not found conversion, got %#v", converted)
    }
}

func TestAppErrorHandlerIsRetryable(t *testing.T) {
    handler := NewAppErrorHandler()

    if !handler.IsRetryable(&AppError{Code: ErrPortUnavailable}) {
        t.Fatalf("expected retryable app error")
    }

    if !handler.IsRetryable(syscall.ECONNREFUSED) {
        t.Fatalf("expected ECONNREFUSED to be retryable")
    }

    if handler.IsRetryable(fmt.Errorf("no")) {
        t.Fatalf("unexpected retryable for generic error")
    }
}

func TestAppErrorHandlerGetRetryConfig(t *testing.T) {
    handler := NewAppErrorHandler()
    cfg := handler.GetRetryConfig(&AppError{Code: ErrPortUnavailable})
    if cfg.MaxRetries != 5 {
        t.Fatalf("unexpected retry config: %#v", cfg)
    }

    cfg = handler.GetRetryConfig(&AppError{Code: ErrProcessNotFound})
    if cfg.BaseDelay != 500*time.Millisecond {
        t.Fatalf("expected custom config for process errors")
    }

    cfg = handler.GetRetryConfig(goerrors.New("no"))
    if cfg != handler.defaultRetryConfig {
        t.Fatalf("expected default config for unknown errors")
    }
}

func TestErrorFactories(t *testing.T) {
    if err := NewFileNotFoundError("foo"); err.Code != ErrFileNotFound {
        t.Fatalf("unexpected code: %s", err.Code)
    }
    if err := NewPortConflictError(8080, "api"); err.Fields["port"].(int) != 8080 {
        t.Fatalf("expected port field")
    }
    if err := NewConfigInvalidError("key", "value"); err.Code != ErrConfigInvalid {
        t.Fatalf("unexpected code for config invalid")
    }
    if err := NewDockerComposeInvalidError("path", "bad"); err.Code != ErrComposeInvalid {
        t.Fatalf("unexpected compose code")
    }
}

func TestAppErrorHandlerConvertUnknown(t *testing.T) {
    handler := NewAppErrorHandler()
    err := handler.convertToAppError(fmt.Errorf("boom"))
    if err.Code != ErrUnknown {
        t.Fatalf("expected unknown error code, got %s", err.Code)
    }
}

func TestAppErrorHandlerHandleWithLogger(t *testing.T) {
    handler := NewAppErrorHandler()
    ctx := logger.WithLogger(context.Background(), &logger.NopLogger{})

    err := handler.Handle(ctx, syscall.EADDRINUSE)
    appErr, ok := err.(*AppError)
    if !ok || appErr.Code != ErrPortUnavailable {
        t.Fatalf("expected port unavailable, got %#v", err)
    }
}

package errors

import (
    "errors"
    "testing"

    "github.com/harakeishi/gopose/pkg/types"
)

func TestAppErrorErrorAndUnwrap(t *testing.T) {
    cause := errors.New("root")
    appErr := &AppError{Code: ErrUnknown, Message: "failed", Cause: cause}

    if got := appErr.Error(); got != "failed: root" {
        t.Fatalf("unexpected error string: %s", got)
    }
    if !errors.Is(appErr, cause) {
        t.Fatalf("expected errors.Is to unwrap cause")
    }
}

func TestAppErrorGetSeverity(t *testing.T) {
    tests := []struct {
        code     ErrorCode
        severity types.Severity
    }{
        {ErrFileNotFound, types.SeverityError},
        {ErrPortUnavailable, types.SeverityWarning},
        {ErrDockerAPIFailed, types.SeverityError},
        {ErrConfigInvalid, types.SeverityError},
        {ErrProcessNotFound, types.SeverityWarning},
        {ErrUnknown, types.SeverityError},
    }
    for _, tc := range tests {
        err := &AppError{Code: tc.code}
        if got := err.GetSeverity(); got != tc.severity {
            t.Fatalf("code %s => %v, want %v", tc.code, got, tc.severity)
        }
    }
}

func TestAppErrorIsRetryable(t *testing.T) {
    tests := map[ErrorCode]bool{
        ErrPortUnavailable: true,
        ErrProcessNotFound: true,
        ErrFilePermission:  true,
        ErrFileNotFound:    false,
    }
    for code, want := range tests {
        err := &AppError{Code: code}
        if got := err.IsRetryable(); got != want {
            t.Fatalf("IsRetryable(%s)=%v want %v", code, got, want)
        }
    }
}

func TestAppErrorWithFieldAndCause(t *testing.T) {
    base := &AppError{Code: ErrUnknown, Message: "failed"}
    base.WithField("service", "api").WithCause(errors.New("boom"))

    if base.Fields["service"] != "api" {
        t.Fatalf("expected field to be stored")
    }
    if base.Cause == nil || base.Cause.Error() != "boom" {
        t.Fatalf("expected cause to be set")
    }
}

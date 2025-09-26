package errors

import "testing"

func TestErrorCodeCategory(t *testing.T) {
    tests := map[ErrorCode]ErrorCategory{
        ErrFileNotFound:       ErrorCategoryFile,
        ErrPortUnavailable:    ErrorCategoryPort,
        ErrDockerAPIFailed:    ErrorCategoryDocker,
        ErrComposeInvalid:     ErrorCategoryDocker,
        ErrConfigInvalid:      ErrorCategoryConfig,
        ErrProcessStartFailed: ErrorCategoryProcess,
        ErrUnknown:            ErrorCategoryUnknown,
    }

    for code, want := range tests {
        if got := code.Category(); got != want {
            t.Fatalf("Category(%s)=%s want %s", code, got, want)
        }
    }
}

func TestErrorCodeString(t *testing.T) {
    if got := ErrInternalError.String(); got != string(ErrInternalError) {
        t.Fatalf("String() returned %q", got)
    }
}

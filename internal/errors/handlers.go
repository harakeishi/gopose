package errors

import (
	"context"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
)

// ErrorHandler はエラーハンドリングのインターフェースです。
type ErrorHandler interface {
	Handle(ctx context.Context, err error) error
	IsRetryable(err error) bool
	GetRetryConfig(err error) RetryConfig
}

// RetryConfig はリトライ設定を表します。
type RetryConfig struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// AppErrorHandler は具体的なエラーハンドラー実装です。
type AppErrorHandler struct {
	defaultRetryConfig RetryConfig
}

// NewAppErrorHandler は新しいエラーハンドラーを作成します。
func NewAppErrorHandler() *AppErrorHandler {
	return &AppErrorHandler{
		defaultRetryConfig: RetryConfig{
			MaxRetries:    3,
			BaseDelay:     100 * time.Millisecond,
			MaxDelay:      5 * time.Second,
			BackoffFactor: 2.0,
		},
	}
}

// Handle はエラーを処理します。
func (h *AppErrorHandler) Handle(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	// AppError の場合はそのまま返す
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}

	// 既知のエラータイプを AppError に変換
	return h.convertToAppError(err)
}

// IsRetryable はエラーがリトライ可能かどうかを判定します。
func (h *AppErrorHandler) IsRetryable(err error) bool {
	if appErr, ok := err.(*AppError); ok {
		return appErr.IsRetryable()
	}

	// システムエラーの場合はエラーの種類に応じて判定
	if errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ETIMEDOUT) ||
		errors.Is(err, syscall.ENOENT) {
		return true
	}

	return false
}

// GetRetryConfig はエラーに応じたリトライ設定を返します。
func (h *AppErrorHandler) GetRetryConfig(err error) RetryConfig {
	if appErr, ok := err.(*AppError); ok {
		switch appErr.Code {
		case ErrPortUnavailable, ErrPortScanFailed:
			return RetryConfig{
				MaxRetries:    5,
				BaseDelay:     200 * time.Millisecond,
				MaxDelay:      2 * time.Second,
				BackoffFactor: 1.5,
			}
		case ErrProcessNotFound, ErrDockerAPIFailed:
			return RetryConfig{
				MaxRetries:    3,
				BaseDelay:     500 * time.Millisecond,
				MaxDelay:      5 * time.Second,
				BackoffFactor: 2.0,
			}
		}
	}

	return h.defaultRetryConfig
}

// convertToAppError は既知のエラーを AppError に変換します。
func (h *AppErrorHandler) convertToAppError(err error) *AppError {
	switch {
	case os.IsNotExist(err):
		return &AppError{
			Code:    ErrFileNotFound,
			Message: "ファイルが存在しません",
			Cause:   err,
		}
	case os.IsPermission(err):
		return &AppError{
			Code:    ErrFilePermission,
			Message: "ファイルアクセス権限がありません",
			Cause:   err,
		}
	case errors.Is(err, syscall.EADDRINUSE):
		return &AppError{
			Code:    ErrPortUnavailable,
			Message: "ポートが既に使用されています",
			Cause:   err,
		}
	case errors.Is(err, syscall.ECONNREFUSED):
		return &AppError{
			Code:    ErrDockerAPIFailed,
			Message: "Docker APIへの接続に失敗しました",
			Cause:   err,
		}
	default:
		return &AppError{
			Code:    ErrUnknown,
			Message: fmt.Sprintf("予期しないエラーが発生しました: %v", err),
			Cause:   err,
		}
	}
}

// 事前定義されたエラーのファクトリ関数

// NewFileNotFoundError はファイル未発見エラーを作成します。
func NewFileNotFoundError(path string) *AppError {
	return &AppError{
		Code:    ErrFileNotFound,
		Message: fmt.Sprintf("ファイルが見つかりません: %s", path),
		Fields:  map[string]interface{}{"path": path},
	}
}

// NewPortConflictError はポート衝突エラーを作成します。
func NewPortConflictError(port int, service string) *AppError {
	return &AppError{
		Code:    ErrPortConflict,
		Message: fmt.Sprintf("ポート %d で衝突が発生しました (サービス: %s)", port, service),
		Fields: map[string]interface{}{
			"port":    port,
			"service": service,
		},
	}
}

// NewConfigInvalidError は設定無効エラーを作成します。
func NewConfigInvalidError(field string, value interface{}) *AppError {
	return &AppError{
		Code:    ErrConfigInvalid,
		Message: fmt.Sprintf("設定が無効です: %s = %v", field, value),
		Fields: map[string]interface{}{
			"field": field,
			"value": value,
		},
	}
}

// NewDockerComposeInvalidError はDocker Compose無効エラーを作成します。
func NewDockerComposeInvalidError(path string, reason string) *AppError {
	return &AppError{
		Code:    ErrComposeInvalid,
		Message: fmt.Sprintf("Docker Composeファイルが無効です: %s (%s)", path, reason),
		Fields: map[string]interface{}{
			"path":   path,
			"reason": reason,
		},
	}
}

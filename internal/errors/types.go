// Package errors は、gopose アプリケーション用の構造化エラーハンドリングを提供します。
package errors

import (
	"fmt"

	"github.com/harakeishi/gopose/pkg/types"
)

// AppError はアプリケーション固有のエラーを表します。
type AppError struct {
	Code    ErrorCode              `json:"code"`
	Message string                 `json:"message"`
	Cause   error                  `json:"cause,omitempty"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

// Error は error インターフェースを実装します。
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap は原因エラーを返します。
func (e *AppError) Unwrap() error {
	return e.Cause
}

// GetSeverity はエラーの重要度を返します。
func (e *AppError) GetSeverity() types.Severity {
	switch e.Code.Category() {
	case ErrorCategoryFile:
		return types.SeverityError
	case ErrorCategoryPort:
		return types.SeverityWarning
	case ErrorCategoryDocker:
		return types.SeverityError
	case ErrorCategoryConfig:
		return types.SeverityError
	case ErrorCategoryProcess:
		return types.SeverityWarning
	default:
		return types.SeverityError
	}
}

// IsRetryable はエラーがリトライ可能かどうかを判定します。
func (e *AppError) IsRetryable() bool {
	switch e.Code {
	case ErrPortUnavailable, ErrProcessNotFound, ErrFilePermission:
		return true
	default:
		return false
	}
}

// WithField はエラーにフィールドを追加します。
func (e *AppError) WithField(key string, value interface{}) *AppError {
	if e.Fields == nil {
		e.Fields = make(map[string]interface{})
	}
	e.Fields[key] = value
	return e
}

// WithCause は原因エラーを設定します。
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

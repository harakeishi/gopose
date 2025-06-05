// Package logger は、gopose アプリケーション用の構造化ログ機能を提供します。
package logger

import (
	"context"

	"github.com/harakeishi/gopose/pkg/types"
)

// Logger は構造化ログ出力のインターフェースです。
type Logger interface {
	Debug(ctx context.Context, message string, fields ...types.Field)
	Info(ctx context.Context, message string, fields ...types.Field)
	Warn(ctx context.Context, message string, fields ...types.Field)
	Error(ctx context.Context, message string, err error, fields ...types.Field)
	Fatal(ctx context.Context, message string, err error, fields ...types.Field)

	WithField(key string, value interface{}) Logger
	WithFields(fields ...types.Field) Logger
	WithError(err error) Logger
}

// LoggerFactory はロガーの作成を行うファクトリインターフェースです。
type LoggerFactory interface {
	Create(config types.LogConfig) (Logger, error)
	CreateWithName(name string, config types.LogConfig) (Logger, error)
}

// ContextKey はコンテキストキーの型です。
type ContextKey string

const (
	// ContextKeyLogger はコンテキストにロガーを格納するためのキーです。
	ContextKeyLogger ContextKey = "logger"
	// ContextKeyRequestID はリクエストIDを格納するためのキーです。
	ContextKeyRequestID ContextKey = "request_id"
	// ContextKeyTraceID はトレースIDを格納するためのキーです。
	ContextKeyTraceID ContextKey = "trace_id"
)

// FromContext はコンテキストからロガーを取得します。
func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(ContextKeyLogger).(Logger); ok {
		return logger
	}
	// フォールバック用のNopLoggerを返す
	return &NopLogger{}
}

// WithLogger はコンテキストにロガーを設定します。
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, ContextKeyLogger, logger)
}

// NopLogger は何も出力しないロガーの実装です。
type NopLogger struct{}

func (n *NopLogger) Debug(ctx context.Context, message string, fields ...types.Field)            {}
func (n *NopLogger) Info(ctx context.Context, message string, fields ...types.Field)             {}
func (n *NopLogger) Warn(ctx context.Context, message string, fields ...types.Field)             {}
func (n *NopLogger) Error(ctx context.Context, message string, err error, fields ...types.Field) {}
func (n *NopLogger) Fatal(ctx context.Context, message string, err error, fields ...types.Field) {}
func (n *NopLogger) WithField(key string, value interface{}) Logger                              { return n }
func (n *NopLogger) WithFields(fields ...types.Field) Logger                                     { return n }
func (n *NopLogger) WithError(err error) Logger                                                  { return n }

package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/harakeishi/gopose/pkg/types"
)

// StructuredLogger は構造化ログの実装です。
type StructuredLogger struct {
	logger   *slog.Logger
	fields   []types.Field
	err      error
	detailed bool
}

// StructuredLoggerFactory は構造化ログのファクトリです。
type StructuredLoggerFactory struct {
	detailed bool
}

// NewStructuredLoggerFactory は新しいファクトリを作成します。
// detailed が false の場合、ログ出力はメッセージのみとなります。
func NewStructuredLoggerFactory(detailed bool) *StructuredLoggerFactory {
	return &StructuredLoggerFactory{detailed: detailed}
}

// Create は設定に基づいてロガーを作成します。
func (f *StructuredLoggerFactory) Create(config types.LogConfig) (Logger, error) {
	return f.CreateWithName("gopose", config)
}

// CreateWithName は名前付きロガーを作成します。
func (f *StructuredLoggerFactory) CreateWithName(name string, config types.LogConfig) (Logger, error) {
	var output io.Writer = os.Stdout

	// ファイル出力が指定されている場合
	if config.File != "" {
		file, err := os.OpenFile(config.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("ログファイルのオープンに失敗しました: %w", err)
		}
		output = file
	}

	// ログレベルの設定
	level := parseLogLevel(config.Level)

	// ハンドラーの作成
	var handler slog.Handler
	if config.Format == "json" {
		handler = slog.NewJSONHandler(output, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		handler = slog.NewTextHandler(output, &slog.HandlerOptions{
			Level: level,
		})
	}

	logger := slog.New(handler).With("component", name)

	return &StructuredLogger{
		logger:   logger,
		fields:   []types.Field{},
		detailed: f.detailed,
	}, nil
}

// parseLogLevel は文字列からログレベルを解析します。
func parseLogLevel(levelStr string) slog.Level {
	switch levelStr {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Debug はデバッグレベルのログを出力します。
func (l *StructuredLogger) Debug(ctx context.Context, message string, fields ...types.Field) {
	l.log(ctx, slog.LevelDebug, message, fields...)
}

// Info は情報レベルのログを出力します。
func (l *StructuredLogger) Info(ctx context.Context, message string, fields ...types.Field) {
	l.log(ctx, slog.LevelInfo, message, fields...)
}

// Warn は警告レベルのログを出力します。
func (l *StructuredLogger) Warn(ctx context.Context, message string, fields ...types.Field) {
	l.log(ctx, slog.LevelWarn, message, fields...)
}

// Error はエラーレベルのログを出力します。
func (l *StructuredLogger) Error(ctx context.Context, message string, err error, fields ...types.Field) {
	allFields := append(fields, types.Field{Key: "error", Value: err.Error()})
	l.log(ctx, slog.LevelError, message, allFields...)
}

// Fatal は致命的エラーレベルのログを出力して終了します。
func (l *StructuredLogger) Fatal(ctx context.Context, message string, err error, fields ...types.Field) {
	allFields := append(fields, types.Field{Key: "error", Value: err.Error()})
	l.log(ctx, slog.LevelError, message, allFields...)
	os.Exit(1)
}

// WithField はフィールドを追加した新しいロガーを返します。
func (l *StructuredLogger) WithField(key string, value interface{}) Logger {
	newFields := make([]types.Field, len(l.fields)+1)
	copy(newFields, l.fields)
	newFields[len(l.fields)] = types.Field{Key: key, Value: value}

	return &StructuredLogger{
		logger: l.logger,
		fields: newFields,
		err:    l.err,
	}
}

// WithFields は複数のフィールドを追加した新しいロガーを返します。
func (l *StructuredLogger) WithFields(fields ...types.Field) Logger {
	newFields := make([]types.Field, len(l.fields)+len(fields))
	copy(newFields, l.fields)
	copy(newFields[len(l.fields):], fields)

	return &StructuredLogger{
		logger: l.logger,
		fields: newFields,
		err:    l.err,
	}
}

// WithError はエラーを追加した新しいロガーを返します。
func (l *StructuredLogger) WithError(err error) Logger {
	return &StructuredLogger{
		logger: l.logger,
		fields: l.fields,
		err:    err,
	}
}

// log は実際のログ出力を行います。
func (l *StructuredLogger) log(ctx context.Context, level slog.Level, message string, fields ...types.Field) {
	if !l.detailed {
		fmt.Println(message)
		return
	}

	// コンテキストから追加情報を取得
	attrs := make([]slog.Attr, 0)

	// タイムスタンプ
	attrs = append(attrs, slog.Time("timestamp", time.Now()))

	// コンテキストからリクエストIDやトレースIDを取得
	if requestID, ok := ctx.Value(ContextKeyRequestID).(string); ok {
		attrs = append(attrs, slog.String("request_id", requestID))
	}
	if traceID, ok := ctx.Value(ContextKeyTraceID).(string); ok {
		attrs = append(attrs, slog.String("trace_id", traceID))
	}

	// 事前に設定されたフィールドを追加
	for _, field := range l.fields {
		attrs = append(attrs, slog.Any(field.Key, field.Value))
	}

	// 引数で渡されたフィールドを追加
	for _, field := range fields {
		attrs = append(attrs, slog.Any(field.Key, field.Value))
	}

	// エラーがある場合は追加
	if l.err != nil {
		attrs = append(attrs, slog.String("error", l.err.Error()))
	}

	l.logger.LogAttrs(ctx, level, message, attrs...)
}

// DefaultConfig はデフォルトのログ設定を返します。
func DefaultConfig() types.LogConfig {
	return types.LogConfig{
		Level:    "info",
		Format:   "text",
		File:     "",
		MaxSize:  100,
		MaxAge:   30,
		Compress: true,
	}
}

// FormatJSON は値をJSON形式でフォーマットします。
func FormatJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%+v", v)
	}
	return string(data)
}

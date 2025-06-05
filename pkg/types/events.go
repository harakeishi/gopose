package types

import "time"

// ProcessEvent はプロセス状態変更イベントを表します。
type ProcessEvent struct {
	Type      ProcessEventType       `json:"type"`
	ProcessID int                    `json:"process_id"`
	Name      string                 `json:"name"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// ProcessEventType はプロセスイベントの種類を表します。
type ProcessEventType string

const (
	ProcessEventStarted   ProcessEventType = "started"
	ProcessEventStopped   ProcessEventType = "stopped"
	ProcessEventRestarted ProcessEventType = "restarted"
	ProcessEventError     ProcessEventType = "error"
	ProcessEventHealthy   ProcessEventType = "healthy"
	ProcessEventUnhealthy ProcessEventType = "unhealthy"
)

// Field はログフィールドのキー・値ペアを表します。
type Field struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// LogEntry はログエントリを表します。
type LogEntry struct {
	Level     LogLevel  `json:"level"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Fields    []Field   `json:"fields"`
	Error     error     `json:"error,omitempty"`
}

// LogLevel はログレベルを表します。
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// FileWatchEvent はファイル監視イベントを表します。
type FileWatchEvent struct {
	Type      FileWatchEventType `json:"type"`
	Path      string             `json:"path"`
	Timestamp time.Time          `json:"timestamp"`
}

// FileWatchEventType はファイル監視イベントの種類を表します。
type FileWatchEventType string

const (
	FileWatchEventCreated  FileWatchEventType = "created"
	FileWatchEventModified FileWatchEventType = "modified"
	FileWatchEventDeleted  FileWatchEventType = "deleted"
	FileWatchEventRenamed  FileWatchEventType = "renamed"
)

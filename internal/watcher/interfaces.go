// Package watcher は、プロセス監視と自動クリーンアップ機能を提供します。
package watcher

import (
	"context"
	"time"

	"github.com/harakeishi/gopose/pkg/types"
)

// ProcessWatcher はプロセス監視を行うインターフェースです。
type ProcessWatcher interface {
	Watch(ctx context.Context, processName string) (<-chan types.ProcessEvent, error)
	WatchPID(ctx context.Context, pid int) (<-chan types.ProcessEvent, error)
	WatchMultiple(ctx context.Context, processNames []string) (<-chan types.ProcessEvent, error)
	Stop(ctx context.Context) error
	IsRunning(ctx context.Context, processName string) (bool, error)
	GetProcessInfo(ctx context.Context, processName string) (*ProcessInfo, error)
}

// CleanupManager は自動クリーンアップを管理するインターフェースです。
type CleanupManager interface {
	RegisterTarget(ctx context.Context, target CleanupTarget) error
	UnregisterTarget(ctx context.Context, targetID string) error
	ExecuteCleanup(ctx context.Context, targetID string) error
	ExecuteAllCleanup(ctx context.Context) error
	ScheduleCleanup(ctx context.Context, targetID string, delay time.Duration) error
}

// DockerWatcher はDocker固有の監視を行うインターフェースです。
type DockerWatcher interface {
	WatchComposeProject(ctx context.Context, projectName string) (<-chan types.ProcessEvent, error)
	WatchComposeService(ctx context.Context, projectName, serviceName string) (<-chan types.ProcessEvent, error)
	IsComposeRunning(ctx context.Context, projectName string) (bool, error)
	GetComposeStatus(ctx context.Context, projectName string) (*ComposeStatus, error)
}

// EventProcessor はイベント処理を行うインターフェースです。
type EventProcessor interface {
	ProcessEvent(ctx context.Context, event types.ProcessEvent) error
	RegisterHandler(eventType types.ProcessEventType, handler EventHandler) error
	UnregisterHandler(eventType types.ProcessEventType) error
}

// EventHandler はイベントハンドラーのインターフェースです。
type EventHandler interface {
	Handle(ctx context.Context, event types.ProcessEvent) error
}

// ProcessInfo はプロセス情報を表します。
type ProcessInfo struct {
	PID         int               `json:"pid"`
	Name        string            `json:"name"`
	Command     string            `json:"command"`
	Status      string            `json:"status"`
	StartTime   time.Time         `json:"start_time"`
	CPUPercent  float64           `json:"cpu_percent"`
	MemoryMB    float64           `json:"memory_mb"`
	Environment map[string]string `json:"environment"`
}

// ComposeStatus はDocker Composeの状態を表します。
type ComposeStatus struct {
	ProjectName string                   `json:"project_name"`
	Services    map[string]ServiceStatus `json:"services"`
	Networks    []string                 `json:"networks"`
	Volumes     []string                 `json:"volumes"`
	Status      string                   `json:"status"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

// ServiceStatus はサービスの状態を表します。
type ServiceStatus struct {
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Health    string    `json:"health"`
	Ports     []string  `json:"ports"`
	Image     string    `json:"image"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CleanupTarget はクリーンアップ対象を表します。
type CleanupTarget struct {
	ID          string            `json:"id"`
	Type        CleanupType       `json:"type"`
	Path        string            `json:"path"`
	ProcessName string            `json:"process_name"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	TTL         time.Duration     `json:"ttl"`
}

// CleanupType はクリーンアップの種類を表します。
type CleanupType string

const (
	CleanupTypeFile    CleanupType = "file"
	CleanupTypeProcess CleanupType = "process"
	CleanupTypeCompose CleanupType = "compose"
	CleanupTypeCustom  CleanupType = "custom"
)

// WatcherConfig は監視設定を表します。
type WatcherConfig struct {
	Interval           time.Duration `json:"interval"`
	Timeout            time.Duration `json:"timeout"`
	RetryCount         int           `json:"retry_count"`
	RetryInterval      time.Duration `json:"retry_interval"`
	BufferSize         int           `json:"buffer_size"`
	HealthCheckEnabled bool          `json:"health_check_enabled"`
}

// EventFilter はイベントフィルターを表します。
type EventFilter struct {
	EventTypes   []types.ProcessEventType `json:"event_types"`
	ProcessNames []string                 `json:"process_names"`
	MinSeverity  types.Severity           `json:"min_severity"`
	TimeRange    *TimeRange               `json:"time_range"`
}

// TimeRange は時間範囲を表します。
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

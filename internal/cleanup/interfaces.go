// Package cleanup は、自動クリーンアップ機能を提供します。
package cleanup

import (
	"context"
	"time"
)

// CleanupManager は自動クリーンアップ機能のメインインターフェースです。
type CleanupManager interface {
	RegisterTarget(ctx context.Context, target CleanupTarget) error
	UnregisterTarget(ctx context.Context, targetID string) error
	ExecuteCleanup(ctx context.Context, targetID string) error
	ExecuteAllCleanup(ctx context.Context) error
	ScheduleCleanup(ctx context.Context, targetID string, delay time.Duration) error
	ListTargets(ctx context.Context) ([]CleanupTarget, error)
	GetTarget(ctx context.Context, targetID string) (*CleanupTarget, error)
}

// CleanupScheduler はクリーンアップのスケジューリングを行うインターフェースです。
type CleanupScheduler interface {
	Schedule(ctx context.Context, targetID string, delay time.Duration) error
	Cancel(ctx context.Context, targetID string) error
	GetScheduled(ctx context.Context) ([]ScheduledCleanup, error)
}

// CleanupExecutor はクリーンアップの実行を行うインターフェースです。
type CleanupExecutor interface {
	Execute(ctx context.Context, target CleanupTarget) error
	CanExecute(ctx context.Context, target CleanupTarget) (bool, error)
	ValidateTarget(ctx context.Context, target CleanupTarget) error
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

// ScheduledCleanup はスケジュールされたクリーンアップを表します。
type ScheduledCleanup struct {
	Target      CleanupTarget `json:"target"`
	ScheduledAt time.Time     `json:"scheduled_at"`
	ExecuteAt   time.Time     `json:"execute_at"`
	Status      string        `json:"status"`
}

// CleanupResult はクリーンアップ結果を表します。
type CleanupResult struct {
	TargetID   string        `json:"target_id"`
	Success    bool          `json:"success"`
	ExecutedAt time.Time     `json:"executed_at"`
	Duration   time.Duration `json:"duration"`
	Error      error         `json:"error,omitempty"`
	Details    string        `json:"details"`
}

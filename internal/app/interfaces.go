// Package app は、アプリケーション層のサービスと依存性注入を提供します。
package app

import (
	"context"

	"github.com/harakeishi/gopose/internal/cleanup"
	"github.com/harakeishi/gopose/internal/file"
	"github.com/harakeishi/gopose/internal/generator"
	"github.com/harakeishi/gopose/internal/parser"
	"github.com/harakeishi/gopose/internal/resolver"
	"github.com/harakeishi/gopose/internal/scanner"
	"github.com/harakeishi/gopose/internal/watcher"
	"github.com/harakeishi/gopose/pkg/types"
)

// Application はメインのアプリケーションサービスインターフェースです。
type Application interface {
	Execute(ctx context.Context, config types.Config) (*ExecutionResult, error)
	ExecuteUp(ctx context.Context, config types.Config) (*UpResult, error)
	ExecuteClean(ctx context.Context, config types.Config) (*CleanResult, error)
	ExecuteStatus(ctx context.Context, config types.Config) (*StatusResult, error)
}

// PortService はポート関連操作の集約インターフェースです。
type PortService interface {
	ScanPorts(ctx context.Context, config types.PortConfig) (*scanner.PortScanResult, error)
	AllocatePorts(ctx context.Context, services []types.Service, config types.PortConfig) (map[string]int, error)
	ValidatePortConfiguration(ctx context.Context, config types.PortConfig) error
}

// ComposeService はDocker Compose関連操作の集約インターフェースです。
type ComposeService interface {
	ParseCompose(ctx context.Context, filePath string) (*types.ComposeConfig, error)
	ExtractServices(ctx context.Context, config *types.ComposeConfig) ([]types.Service, error)
	ValidateCompose(ctx context.Context, config *types.ComposeConfig) error
}

// ResolverService は衝突解決関連操作の集約インターフェースです。
type ResolverService interface {
	DetectConflicts(ctx context.Context, services []types.Service, usedPorts []int) ([]types.Conflict, error)
	ResolveConflicts(ctx context.Context, conflicts []types.Conflict, config types.PortConfig) ([]types.ConflictResolution, error)
	CreateResolutionPlan(ctx context.Context, conflicts []types.Conflict, config types.PortConfig) (*types.ResolutionPlan, error)
}

// FileService はファイル関連操作の集約インターフェースです。
type FileService interface {
	WriteOverride(ctx context.Context, override *types.OverrideConfig, path string) error
	BackupFile(ctx context.Context, filePath string) (string, error)
	CleanupOverride(ctx context.Context, overridePath string) error
}

// WatcherService は監視関連操作の集約インターフェースです。
type WatcherService interface {
	StartWatching(ctx context.Context, processName string, targets []cleanup.CleanupTarget) error
	StopWatching(ctx context.Context) error
	RegisterCleanupTarget(ctx context.Context, target cleanup.CleanupTarget) error
}

// ExecutionResult は実行結果を表します。
type ExecutionResult struct {
	Success        bool                       `json:"success"`
	Command        string                     `json:"command"`
	ConflictsFound []types.Conflict           `json:"conflicts_found"`
	Resolutions    []types.ConflictResolution `json:"resolutions"`
	OverridePath   string                     `json:"override_path"`
	BackupPath     string                     `json:"backup_path"`
	Messages       []string                   `json:"messages"`
	Warnings       []string                   `json:"warnings"`
	Errors         []string                   `json:"errors"`
	ExecutionTime  int64                      `json:"execution_time_ms"`
}

// UpResult はupコマンドの実行結果を表します。
type UpResult struct {
	*ExecutionResult
	ServicesProcessed int            `json:"services_processed"`
	PortsAllocated    map[string]int `json:"ports_allocated"`
	OverrideGenerated bool           `json:"override_generated"`
	WatchingStarted   bool           `json:"watching_started"`
}

// CleanResult はcleanコマンドの実行結果を表します。
type CleanResult struct {
	*ExecutionResult
	FilesDeleted    []string `json:"files_deleted"`
	BackupsRestored []string `json:"backups_restored"`
	WatchingStopped bool     `json:"watching_stopped"`
}

// StatusResult はstatusコマンドの実行結果を表します。
type StatusResult struct {
	*ExecutionResult
	ComposeStatus  *watcher.ComposeStatus  `json:"compose_status"`
	PortStatus     *scanner.PortScanResult `json:"port_status"`
	OverrideExists bool                    `json:"override_exists"`
	WatchingActive bool                    `json:"watching_active"`
	ActiveTargets  []cleanup.CleanupTarget `json:"active_targets"`
}

// ServiceContainer は各サービスのコンテナを表します。
type ServiceContainer struct {
	PortScanner       scanner.PortScanner
	ComposeParser     parser.ComposeParser
	ConflictResolver  resolver.PortConflictResolver
	OverrideGenerator generator.OverrideGenerator
	FileManager       file.FileManager
	ProcessWatcher    watcher.ProcessWatcher
	CleanupManager    cleanup.CleanupManager
}

// ApplicationConfig はアプリケーション設定を表します。
type ApplicationConfig struct {
	Config           types.Config
	ServiceContainer *ServiceContainer
}

// CommandOptions はコマンドオプションを表します。
type CommandOptions struct {
	FilePath     string            `json:"file_path"`
	PortRange    *types.PortRange  `json:"port_range"`
	Verbose      bool              `json:"verbose"`
	DryRun       bool              `json:"dry_run"`
	Force        bool              `json:"force"`
	CustomFields map[string]string `json:"custom_fields"`
}

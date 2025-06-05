// Package config は、gopose アプリケーションの設定管理機能を提供します。
package config

import (
	"context"

	"github.com/harakeishi/gopose/pkg/types"
)

// ConfigLoader は設定ファイルの読み込みを行うインターフェースです。
type ConfigLoader interface {
	Load(ctx context.Context, path string) (types.Config, error)
	LoadFromBytes(ctx context.Context, data []byte) (types.Config, error)
	LoadDefaults(ctx context.Context) (types.Config, error)
}

// ConfigValidator は設定の妥当性を検証するインターフェースです。
type ConfigValidator interface {
	Validate(ctx context.Context, config types.Config) error
	ValidatePort(ctx context.Context, config types.PortConfig) error
	ValidateFile(ctx context.Context, config types.FileConfig) error
	ValidateWatcher(ctx context.Context, config types.WatcherConfig) error
	ValidateLog(ctx context.Context, config types.LogConfig) error
}

// ConfigMerger は設定のマージを行うインターフェースです。
type ConfigMerger interface {
	Merge(ctx context.Context, base types.Config, override types.Config) (types.Config, error)
}

// ConfigManager は設定管理の統合インターフェースです。
type ConfigManager interface {
	Load(ctx context.Context, path string) (types.Config, error)
	LoadWithDefaults(ctx context.Context, path string) (types.Config, error)
	Validate(ctx context.Context, config types.Config) error
	Save(ctx context.Context, config types.Config, path string) error
}

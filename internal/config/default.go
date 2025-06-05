package config

import (
	"time"

	"github.com/harakeishi/gopose/pkg/types"
)

// DefaultConfig はデフォルト設定を返します。
func DefaultConfig() *types.AppConfig {
	return &types.AppConfig{
		Port: types.PortConfig{
			Range: types.PortRange{
				Start: 8000,
				End:   9999,
			},
			Reserved:          []int{8080, 8443, 9000, 9090},
			ExcludePrivileged: true,
		},
		File: types.FileConfig{
			ComposeFile:   "docker-compose.yml",
			OverrideFile:  "docker-compose.override.yml",
			BackupEnabled: true,
			BackupDir:     ".gopose/backups",
		},
		Watcher: types.WatcherConfig{
			Interval:      5 * time.Second,
			CleanupDelay:  30 * time.Second,
			MaxRetries:    3,
			RetryInterval: 1 * time.Second,
		},
		Log: types.LogConfig{
			Level:    "info",
			Format:   "text",
			File:     "",
			MaxSize:  100,
			MaxAge:   30,
			Compress: true,
		},
	}
}

// DefaultPortConfig はデフォルトのポート設定を返します。
func DefaultPortConfig() types.PortConfig {
	return types.PortConfig{
		Range: types.PortRange{
			Start: 8000,
			End:   9999,
		},
		Reserved:          []int{8080, 8443, 9000, 9090},
		ExcludePrivileged: true,
	}
}

// DefaultFileConfig はデフォルトのファイル設定を返します。
func DefaultFileConfig() types.FileConfig {
	return types.FileConfig{
		ComposeFile:   "docker-compose.yml",
		OverrideFile:  "docker-compose.override.yml",
		BackupEnabled: true,
		BackupDir:     ".gopose/backups",
	}
}

// DefaultWatcherConfig はデフォルトの監視設定を返します。
func DefaultWatcherConfig() types.WatcherConfig {
	return types.WatcherConfig{
		Interval:      5 * time.Second,
		CleanupDelay:  30 * time.Second,
		MaxRetries:    3,
		RetryInterval: 1 * time.Second,
	}
}

// DefaultLogConfig はデフォルトのログ設定を返します。
func DefaultLogConfig() types.LogConfig {
	return types.LogConfig{
		Level:    "info",
		Format:   "text",
		File:     "",
		MaxSize:  100,
		MaxAge:   30,
		Compress: true,
	}
}

// RecommendedConfigs は推奨設定のバリエーションを提供します。

// DevelopmentConfig は開発環境向けの設定を返します。
func DevelopmentConfig() *types.AppConfig {
	config := DefaultConfig()
	config.Log.Level = "debug"
	config.Log.Format = "text"
	config.Watcher.Interval = 2 * time.Second
	return config
}

// ProductionConfig は本番環境向けの設定を返します。
func ProductionConfig() *types.AppConfig {
	config := DefaultConfig()
	config.Log.Level = "warn"
	config.Log.Format = "json"
	config.Log.File = "/var/log/gopose/gopose.log"
	config.Watcher.Interval = 10 * time.Second
	config.Watcher.CleanupDelay = 60 * time.Second
	return config
}

// TestConfig はテスト環境向けの設定を返します。
func TestConfig() *types.AppConfig {
	config := DefaultConfig()
	config.Log.Level = "debug"
	config.Log.Format = "text"
	config.File.BackupEnabled = false
	config.Watcher.Interval = 100 * time.Millisecond
	config.Watcher.CleanupDelay = 1 * time.Second
	return config
}

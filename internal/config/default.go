package config

import (
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
			ComposeFile:  "compose.yml",
			OverrideFile: "compose.override.yml",
		},
		Log: types.LogConfig{
			Level:  "warn",
			Format: "text",
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
		ComposeFile:  "compose.yml",
		OverrideFile: "compose.override.yml",
	}
}

// DefaultLogConfig はデフォルトのログ設定を返します。
func DefaultLogConfig() types.LogConfig {
	return types.LogConfig{
		Level:  "warn",
		Format: "text",
	}
}

// DevelopmentConfig は開発環境向けの設定を返します。
func DevelopmentConfig() *types.AppConfig {
	config := DefaultConfig()
	config.Log.Level = "debug"
	config.Log.Format = "text"
	return config
}

// ProductionConfig は本番環境向けの設定を返します。
func ProductionConfig() *types.AppConfig {
	config := DefaultConfig()
	config.Log.Level = "warn"
	config.Log.Format = "json"
	config.Log.File = "/var/log/gopose/gopose.log"
	return config
}

// TestConfig はテスト環境向けの設定を返します。
func TestConfig() *types.AppConfig {
	config := DefaultConfig()
	config.Log.Level = "debug"
	config.Log.Format = "text"
	return config
}

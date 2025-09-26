package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/harakeishi/gopose/pkg/types"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	expected := &types.AppConfig{
		Port: types.PortConfig{
			Range:             types.PortRange{Start: 8000, End: 9999},
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

	if !reflect.DeepEqual(cfg, expected) {
		t.Fatalf("unexpected default config: %#v", cfg)
	}
}

func TestDefaultPortConfig(t *testing.T) {
	expected := types.PortConfig{
		Range:             types.PortRange{Start: 8000, End: 9999},
		Reserved:          []int{8080, 8443, 9000, 9090},
		ExcludePrivileged: true,
	}
	if got := DefaultPortConfig(); !reflect.DeepEqual(got, expected) {
		t.Fatalf("DefaultPortConfig() = %#v, want %#v", got, expected)
	}
}

func TestDefaultFileConfig(t *testing.T) {
	expected := types.FileConfig{
		ComposeFile:   "docker-compose.yml",
		OverrideFile:  "docker-compose.override.yml",
		BackupEnabled: true,
		BackupDir:     ".gopose/backups",
	}
	if got := DefaultFileConfig(); !reflect.DeepEqual(got, expected) {
		t.Fatalf("DefaultFileConfig() = %#v, want %#v", got, expected)
	}
}

func TestDefaultWatcherConfig(t *testing.T) {
	expected := types.WatcherConfig{
		Interval:      5 * time.Second,
		CleanupDelay:  30 * time.Second,
		MaxRetries:    3,
		RetryInterval: 1 * time.Second,
	}
	if got := DefaultWatcherConfig(); got != expected {
		t.Fatalf("DefaultWatcherConfig() = %#v, want %#v", got, expected)
	}
}

func TestDefaultLogConfig(t *testing.T) {
	expected := types.LogConfig{
		Level:    "info",
		Format:   "text",
		File:     "",
		MaxSize:  100,
		MaxAge:   30,
		Compress: true,
	}
	if got := DefaultLogConfig(); got != expected {
		t.Fatalf("DefaultLogConfig() = %#v, want %#v", got, expected)
	}
}

func TestRecommendedConfigs(t *testing.T) {
	t.Run("development", func(t *testing.T) {
		cfg := DevelopmentConfig()
		if cfg.Log.Level != "debug" || cfg.Log.Format != "text" {
			t.Fatalf("unexpected dev log config: %#v", cfg.Log)
		}
		if cfg.Watcher.Interval != 2*time.Second {
			t.Fatalf("unexpected dev interval: %v", cfg.Watcher.Interval)
		}
	})

	t.Run("production", func(t *testing.T) {
		cfg := ProductionConfig()
		if cfg.Log.Level != "warn" || cfg.Log.Format != "json" {
			t.Fatalf("unexpected prod log config: %#v", cfg.Log)
		}
		if cfg.Log.File == "" {
			t.Fatalf("expected production log file to be set")
		}
		if cfg.Watcher.Interval != 10*time.Second || cfg.Watcher.CleanupDelay != 60*time.Second {
			t.Fatalf("unexpected prod watcher config: %#v", cfg.Watcher)
		}
	})

	t.Run("test", func(t *testing.T) {
		cfg := TestConfig()
		if cfg.File.BackupEnabled {
			t.Fatalf("expected backups disabled for test config")
		}
		if cfg.Watcher.Interval != 100*time.Millisecond {
			t.Fatalf("unexpected test watcher interval: %v", cfg.Watcher.Interval)
		}
	})
}

package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// ポート設定の確認
	if config.Port.Range.Start != 8000 {
		t.Errorf("Port.Range.Start = %d, want 8000", config.Port.Range.Start)
	}
	if config.Port.Range.End != 9999 {
		t.Errorf("Port.Range.End = %d, want 9999", config.Port.Range.End)
	}
	if !config.Port.ExcludePrivileged {
		t.Error("Port.ExcludePrivileged = false, want true")
	}

	// ファイル設定の確認
	if config.File.ComposeFile != "compose.yml" {
		t.Errorf("File.ComposeFile = %s, want compose.yml", config.File.ComposeFile)
	}
	if config.File.OverrideFile != "compose.override.yml" {
		t.Errorf("File.OverrideFile = %s, want compose.override.yml", config.File.OverrideFile)
	}
	if !config.File.BackupEnabled {
		t.Error("File.BackupEnabled = false, want true")
	}

	// Watcher設定の確認
	if config.Watcher.Interval != 5*time.Second {
		t.Errorf("Watcher.Interval = %v, want 5s", config.Watcher.Interval)
	}
	if config.Watcher.MaxRetries != 3 {
		t.Errorf("Watcher.MaxRetries = %d, want 3", config.Watcher.MaxRetries)
	}

	// ログ設定の確認
	if config.Log.Level != "info" {
		t.Errorf("Log.Level = %s, want info", config.Log.Level)
	}
	if config.Log.Format != "text" {
		t.Errorf("Log.Format = %s, want text", config.Log.Format)
	}
}

func TestDefaultPortConfig(t *testing.T) {
	config := DefaultPortConfig()

	if config.Range.Start != 8000 {
		t.Errorf("Range.Start = %d, want 8000", config.Range.Start)
	}
	if config.Range.End != 9999 {
		t.Errorf("Range.End = %d, want 9999", config.Range.End)
	}

	expectedReserved := []int{8080, 8443, 9000, 9090}
	if len(config.Reserved) != len(expectedReserved) {
		t.Errorf("Reserved length = %d, want %d", len(config.Reserved), len(expectedReserved))
	}

	for i, port := range expectedReserved {
		if config.Reserved[i] != port {
			t.Errorf("Reserved[%d] = %d, want %d", i, config.Reserved[i], port)
		}
	}

	if !config.ExcludePrivileged {
		t.Error("ExcludePrivileged = false, want true")
	}
}

func TestDefaultFileConfig(t *testing.T) {
	config := DefaultFileConfig()

	if config.ComposeFile != "compose.yml" {
		t.Errorf("ComposeFile = %s, want compose.yml", config.ComposeFile)
	}
	if config.OverrideFile != "compose.override.yml" {
		t.Errorf("OverrideFile = %s, want compose.override.yml", config.OverrideFile)
	}
	if !config.BackupEnabled {
		t.Error("BackupEnabled = false, want true")
	}
	if config.BackupDir != ".gopose/backups" {
		t.Errorf("BackupDir = %s, want .gopose/backups", config.BackupDir)
	}
}

func TestDefaultWatcherConfig(t *testing.T) {
	config := DefaultWatcherConfig()

	if config.Interval != 5*time.Second {
		t.Errorf("Interval = %v, want 5s", config.Interval)
	}
	if config.CleanupDelay != 30*time.Second {
		t.Errorf("CleanupDelay = %v, want 30s", config.CleanupDelay)
	}
	if config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", config.MaxRetries)
	}
	if config.RetryInterval != 1*time.Second {
		t.Errorf("RetryInterval = %v, want 1s", config.RetryInterval)
	}
}

func TestDefaultLogConfig(t *testing.T) {
	config := DefaultLogConfig()

	if config.Level != "info" {
		t.Errorf("Level = %s, want info", config.Level)
	}
	if config.Format != "text" {
		t.Errorf("Format = %s, want text", config.Format)
	}
	if config.File != "" {
		t.Errorf("File = %s, want empty string", config.File)
	}
	if config.MaxSize != 100 {
		t.Errorf("MaxSize = %d, want 100", config.MaxSize)
	}
	if config.MaxAge != 30 {
		t.Errorf("MaxAge = %d, want 30", config.MaxAge)
	}
	if !config.Compress {
		t.Error("Compress = false, want true")
	}
}

func TestDevelopmentConfig(t *testing.T) {
	config := DevelopmentConfig()

	if config.Log.Level != "debug" {
		t.Errorf("Log.Level = %s, want debug", config.Log.Level)
	}
	if config.Log.Format != "text" {
		t.Errorf("Log.Format = %s, want text", config.Log.Format)
	}
	if config.Watcher.Interval != 2*time.Second {
		t.Errorf("Watcher.Interval = %v, want 2s", config.Watcher.Interval)
	}

	// 基本設定も正しく継承されているか確認
	if config.Port.Range.Start != 8000 {
		t.Errorf("Port.Range.Start = %d, want 8000", config.Port.Range.Start)
	}
}

func TestProductionConfig(t *testing.T) {
	config := ProductionConfig()

	if config.Log.Level != "warn" {
		t.Errorf("Log.Level = %s, want warn", config.Log.Level)
	}
	if config.Log.Format != "json" {
		t.Errorf("Log.Format = %s, want json", config.Log.Format)
	}
	if config.Log.File != "/var/log/gopose/gopose.log" {
		t.Errorf("Log.File = %s, want /var/log/gopose/gopose.log", config.Log.File)
	}
	if config.Watcher.Interval != 10*time.Second {
		t.Errorf("Watcher.Interval = %v, want 10s", config.Watcher.Interval)
	}
	if config.Watcher.CleanupDelay != 60*time.Second {
		t.Errorf("Watcher.CleanupDelay = %v, want 60s", config.Watcher.CleanupDelay)
	}

	// 基本設定も正しく継承されているか確認
	if config.Port.Range.Start != 8000 {
		t.Errorf("Port.Range.Start = %d, want 8000", config.Port.Range.Start)
	}
}

func TestTestConfig(t *testing.T) {
	config := TestConfig()

	if config.Log.Level != "debug" {
		t.Errorf("Log.Level = %s, want debug", config.Log.Level)
	}
	if config.Log.Format != "text" {
		t.Errorf("Log.Format = %s, want text", config.Log.Format)
	}
	if config.File.BackupEnabled {
		t.Error("File.BackupEnabled = true, want false (for testing)")
	}
	if config.Watcher.Interval != 100*time.Millisecond {
		t.Errorf("Watcher.Interval = %v, want 100ms", config.Watcher.Interval)
	}
	if config.Watcher.CleanupDelay != 1*time.Second {
		t.Errorf("Watcher.CleanupDelay = %v, want 1s", config.Watcher.CleanupDelay)
	}

	// 基本設定も正しく継承されているか確認
	if config.Port.Range.Start != 8000 {
		t.Errorf("Port.Range.Start = %d, want 8000", config.Port.Range.Start)
	}
}

func TestConfigIsolation(t *testing.T) {
	// 各設定関数が独立したインスタンスを返すことを確認
	config1 := DefaultConfig()
	config2 := DefaultConfig()

	// ポインタが異なることを確認
	if config1 == config2 {
		t.Error("DefaultConfig() returns the same pointer, expected different instances")
	}

	// 一方を変更しても他方に影響しないことを確認
	config1.Port.Range.Start = 9999
	if config2.Port.Range.Start == 9999 {
		t.Error("Modifying config1 affected config2, expected independence")
	}
}

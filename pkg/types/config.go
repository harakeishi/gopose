package types

import "time"

// Config は全体設定を表すインターフェースです。
type Config interface {
	GetPort() PortConfig
	GetFile() FileConfig
	GetWatcher() WatcherConfig
	GetLog() LogConfig
	Validate() error
}

// PortConfig はポート関連設定を表します。
type PortConfig struct {
	Range             PortRange `yaml:"range" json:"range"`
	Reserved          []int     `yaml:"reserved" json:"reserved"`
	ExcludePrivileged bool      `yaml:"exclude_privileged" json:"exclude_privileged"`
}

// FileConfig はファイル関連設定を表します。
type FileConfig struct {
	ComposeFile   string `yaml:"compose_file" json:"compose_file"`
	OverrideFile  string `yaml:"override_file" json:"override_file"`
	BackupEnabled bool   `yaml:"backup_enabled" json:"backup_enabled"`
	BackupDir     string `yaml:"backup_dir" json:"backup_dir"`
}

// WatcherConfig は監視関連設定を表します。
type WatcherConfig struct {
	Interval      time.Duration `yaml:"interval" json:"interval"`
	CleanupDelay  time.Duration `yaml:"cleanup_delay" json:"cleanup_delay"`
	MaxRetries    int           `yaml:"max_retries" json:"max_retries"`
	RetryInterval time.Duration `yaml:"retry_interval" json:"retry_interval"`
}

// LogConfig はログ関連設定を表します。
type LogConfig struct {
	Level    string `yaml:"level" json:"level"`
	Format   string `yaml:"format" json:"format"`
	File     string `yaml:"file" json:"file"`
	MaxSize  int    `yaml:"max_size" json:"max_size"`
	MaxAge   int    `yaml:"max_age" json:"max_age"`
	Compress bool   `yaml:"compress" json:"compress"`
}

// AppConfig は具体的な設定実装です。
type AppConfig struct {
	Port    PortConfig    `yaml:"port" json:"port"`
	File    FileConfig    `yaml:"file" json:"file"`
	Watcher WatcherConfig `yaml:"watcher" json:"watcher"`
	Log     LogConfig     `yaml:"log" json:"log"`
}

// GetPort はポート設定を返します。
func (c *AppConfig) GetPort() PortConfig {
	return c.Port
}

// GetFile はファイル設定を返します。
func (c *AppConfig) GetFile() FileConfig {
	return c.File
}

// GetWatcher は監視設定を返します。
func (c *AppConfig) GetWatcher() WatcherConfig {
	return c.Watcher
}

// GetLog はログ設定を返します。
func (c *AppConfig) GetLog() LogConfig {
	return c.Log
}

// Validate は設定の妥当性を検証します。
func (c *AppConfig) Validate() error {
	// TODO: 設定のバリデーションロジックを実装
	return nil
}

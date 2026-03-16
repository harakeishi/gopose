package types

// Config は全体設定を表すインターフェースです。
type Config interface {
	GetPort() PortConfig
	GetFile() FileConfig
	GetLog() LogConfig
}

// PortConfig はポート関連設定を表します。
type PortConfig struct {
	Range             PortRange `yaml:"range" json:"range"`
	Reserved          []int     `yaml:"reserved" json:"reserved"`
	ExcludePrivileged bool      `yaml:"exclude_privileged" json:"exclude_privileged"`
}

// FileConfig はファイル関連設定を表します。
type FileConfig struct {
	ComposeFile  string `yaml:"compose_file" json:"compose_file"`
	OverrideFile string `yaml:"override_file" json:"override_file"`
}

// LogConfig はログ関連設定を表します。
type LogConfig struct {
	Level  string `yaml:"level" json:"level"`
	Format string `yaml:"format" json:"format"`
	File   string `yaml:"file" json:"file"`
}

// AppConfig は具体的な設定実装です。
type AppConfig struct {
	Port PortConfig `yaml:"port" json:"port"`
	File FileConfig `yaml:"file" json:"file"`
	Log  LogConfig  `yaml:"log" json:"log"`
}

// GetPort はポート設定を返します。
func (c *AppConfig) GetPort() PortConfig {
	return c.Port
}

// GetFile はファイル設定を返します。
func (c *AppConfig) GetFile() FileConfig {
	return c.File
}

// GetLog はログ設定を返します。
func (c *AppConfig) GetLog() LogConfig {
	return c.Log
}

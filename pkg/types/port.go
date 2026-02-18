// Package types は、gopose で使用される基本的な型定義を提供します。
package types

import "time"

// PortRange はポート範囲を表す構造体です。
type PortRange struct {
	Start int `yaml:"start" json:"start"`
	End   int `yaml:"end" json:"end"`
}

// PortMapping はDocker Composeのポートマッピングを表します。
type PortMapping struct {
	Host      int    `yaml:"host" json:"host"`
	Container int    `yaml:"container" json:"container"`
	Protocol  string `yaml:"protocol" json:"protocol"`
	HostIP    string `yaml:"host_ip" json:"host_ip"`
}

// ConflictType はポート衝突の種類を表します。
type ConflictType string

const (
	ConflictTypeSystem  ConflictType = "system"
	ConflictTypeCompose ConflictType = "compose"
)

// Severity は衝突の重要度を表します。
type Severity string

const (
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// ResolutionStrategy は解決戦略の種類を表します。
type ResolutionStrategy string

const (
	ResolutionStrategyAutoIncrement ResolutionStrategy = "auto_increment"
)

// ConflictResolution は衝突解決の結果を表します。
type ConflictResolution struct {
	Service      string             `json:"service"`
	ServiceName  string             `json:"service_name"`
	OriginalPort int                `json:"original_port"`
	ConflictPort int                `json:"conflict_port"`
	ResolvedPort int                `json:"resolved_port"`
	Strategy     ResolutionStrategy `json:"strategy"`
	Reason       string             `json:"reason"`
	Timestamp    time.Time          `json:"timestamp"`
}

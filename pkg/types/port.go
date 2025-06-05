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

// Conflict は検出されたポート衝突を表します。
type Conflict struct {
	Service     string       `json:"service"`
	ServiceName string       `json:"service_name"` // エイリアス
	Port        int          `json:"port"`
	Protocol    string       `json:"protocol"`
	Type        ConflictType `json:"type"`
	Severity    Severity     `json:"severity"`
	Description string       `json:"description"`
}

// ConflictType はポート衝突の種類を表します。
type ConflictType string

const (
	ConflictTypeNone         ConflictType = "none"
	ConflictTypeSystem       ConflictType = "system"
	ConflictTypeCompose      ConflictType = "compose"
	ConflictTypeSystemPort   ConflictType = "system_port"
	ConflictTypeServicePort  ConflictType = "service_port"
	ConflictTypeReservedPort ConflictType = "reserved_port"
	ConflictTypeOutOfRange   ConflictType = "out_of_range"
)

// Severity は衝突の重要度を表します。
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// ConflictSeverity はポート衝突の重要度を表します（エイリアス）。
type ConflictSeverity = Severity

const (
	ConflictSeverityLow    ConflictSeverity = SeverityInfo
	ConflictSeverityMedium ConflictSeverity = SeverityWarning
	ConflictSeverityHigh   ConflictSeverity = SeverityError
)

// ResolutionPlan は衝突解決計画を表します。
type ResolutionPlan struct {
	Strategy   ResolutionStrategy `json:"strategy"`
	Priority   int                `json:"priority"`
	Assignment map[string]int     `json:"assignment"`
}

// ResolutionStrategy は解決戦略の種類を表します。
type ResolutionStrategy string

const (
	StrategyMinimalChange             ResolutionStrategy = "minimal_change"
	StrategyProximity                 ResolutionStrategy = "proximity"
	StrategySequential                ResolutionStrategy = "sequential"
	ResolutionStrategyAutoIncrement   ResolutionStrategy = "auto_increment"
	ResolutionStrategyRangeAllocation ResolutionStrategy = "range_allocation"
	ResolutionStrategyUserDefined     ResolutionStrategy = "user_defined"
)

// ConflictResolution は衝突解決の結果を表します。
type ConflictResolution struct {
	Service      string             `json:"service"`
	ServiceName  string             `json:"service_name"` // エイリアス
	OriginalPort int                `json:"original_port"`
	ConflictPort int                `json:"conflict_port"` // エイリアス
	ResolvedPort int                `json:"resolved_port"`
	Strategy     ResolutionStrategy `json:"strategy"`
	Reason       string             `json:"reason"`
	Timestamp    time.Time          `json:"timestamp"`
}

// SystemPortInfo はシステムポート情報を表します。
type SystemPortInfo struct {
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	ProcessName string `json:"process_name"`
	ProcessID   int    `json:"process_id"`
	State       string `json:"state"`
}

// PortScanResult はポートスキャン結果を表します。
type PortScanResult struct {
	UsedPorts      []int            `json:"used_ports"`
	AvailablePorts []int            `json:"available_ports"`
	PortInfo       []SystemPortInfo `json:"port_info"`
	ScanDuration   int64            `json:"scan_duration_ms"`
}

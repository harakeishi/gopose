package types

import "time"

// UnifiedConflictInfo は統一的な衝突情報を表します。
type UnifiedConflictInfo struct {
	PortConflicts    []PortConflictInfo    `json:"port_conflicts"`
	NetworkConflicts []NetworkConflictInfo `json:"network_conflicts"`
	GeneratedAt      time.Time             `json:"generated_at"`
}

// PortConflictInfo はポート衝突情報を表します。
type PortConflictInfo struct {
	Service     string              `json:"service"`
	ServiceName string              `json:"service_name"` // エイリアス
	Port        int                 `json:"port"`
	Protocol    string              `json:"protocol"`
	Type        ConflictType        `json:"type"`
	Description string              `json:"description"`
	Resolution  *PortResolutionInfo `json:"resolution,omitempty"`
}

// NetworkConflictInfo はネットワーク衝突情報を表します。
type NetworkConflictInfo struct {
	NetworkName        string                 `json:"network_name"`
	ConflictType       NetworkConflictType    `json:"conflict_type"`
	OriginalSubnet     string                 `json:"original_subnet"`
	ConflictingSubnet  string                 `json:"conflicting_subnet,omitempty"`
	ConflictingNetwork string                 `json:"conflicting_network,omitempty"`
	Description        string                 `json:"description"`
	Resolution         *NetworkResolutionInfo `json:"resolution,omitempty"`
	ServiceIPs         map[string]string      `json:"service_ips,omitempty"`
}

// NetworkConflictType はネットワーク衝突の種類を表します。
type NetworkConflictType string

const (
	NetworkConflictTypeSubnet NetworkConflictType = "subnet"
	NetworkConflictTypeName   NetworkConflictType = "name"
)

// PortResolutionInfo はポート衝突の解決情報を表します。
type PortResolutionInfo struct {
	ResolvedPort int                `json:"resolved_port"`
	Strategy     ResolutionStrategy `json:"strategy"`
	Reason       string             `json:"reason"`
}

// NetworkResolutionInfo はネットワーク衝突の解決情報を表します。
type NetworkResolutionInfo struct {
	ResolvedSubnet string            `json:"resolved_subnet"`
	ServiceIPs     map[string]string `json:"service_ips,omitempty"`
	Reason         string            `json:"reason"`
}

// HasConflicts は衝突があるかどうかを確認します。
func (u *UnifiedConflictInfo) HasConflicts() bool {
	return len(u.PortConflicts) > 0 || len(u.NetworkConflicts) > 0
}

// HasPortConflicts はポート衝突があるかどうかを確認します。
func (u *UnifiedConflictInfo) HasPortConflicts() bool {
	return len(u.PortConflicts) > 0
}

// HasNetworkConflicts はネットワーク衝突があるかどうかを確認します。
func (u *UnifiedConflictInfo) HasNetworkConflicts() bool {
	return len(u.NetworkConflicts) > 0
}

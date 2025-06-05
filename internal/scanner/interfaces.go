// Package scanner は、ポートスキャンと利用可能ポートの検出機能を提供します。
package scanner

import (
	"context"

	"github.com/harakeishi/gopose/pkg/types"
)

// PortDetector はシステムの使用中ポート検出を行うインターフェースです。
type PortDetector interface {
	DetectUsedPorts(ctx context.Context) ([]int, error)
	DetectUsedPortsInRange(ctx context.Context, portRange types.PortRange) ([]int, error)
	IsPortInUse(ctx context.Context, port int) (bool, error)
}

// PortAllocator は利用可能ポートの割り当てを行うインターフェースです。
type PortAllocator interface {
	AllocatePort(ctx context.Context, config types.PortConfig) (int, error)
	AllocatePorts(ctx context.Context, count int, config types.PortConfig) ([]int, error)
	AllocatePortsForServices(ctx context.Context, services []types.Service, config types.PortConfig) (map[string]int, error)
}

// PortValidator はポート設定の妥当性検証を行うインターフェースです。
type PortValidator interface {
	ValidatePort(ctx context.Context, port int) error
	ValidatePortRange(ctx context.Context, portRange types.PortRange) error
	ValidatePortMapping(ctx context.Context, mapping types.PortMapping) error
}

// PortScanner はポートスキャン機能の統合インターフェースです。
type PortScanner interface {
	PortDetector
	PortAllocator
	PortValidator
}

// SystemPortInfo はシステムのポート情報を表します。
type SystemPortInfo struct {
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	ProcessName string `json:"process_name"`
	ProcessID   int    `json:"process_id"`
	State       string `json:"state"`
}

// PortScanResult はポートスキャンの結果を表します。
type PortScanResult struct {
	UsedPorts      []int            `json:"used_ports"`
	AvailablePorts []int            `json:"available_ports"`
	PortInfo       []SystemPortInfo `json:"port_info"`
	ScanDuration   int64            `json:"scan_duration_ms"`
}

// AllocationStrategy はポート割り当て戦略を表します。
type AllocationStrategy string

const (
	AllocationStrategySequential AllocationStrategy = "sequential"
	AllocationStrategyRandom     AllocationStrategy = "random"
	AllocationStrategyProximity  AllocationStrategy = "proximity"
)

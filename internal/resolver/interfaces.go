// Package resolver は、ポート衝突の検出と解決機能を提供します。
package resolver

import (
	"context"

	"github.com/harakeishi/gopose/pkg/types"
)

// ConflictDetector はポート衝突の検出を行うインターフェースです。
type ConflictDetector interface {
	DetectConflicts(ctx context.Context, services []types.Service, usedPorts []int) ([]types.Conflict, error)
	DetectSystemConflicts(ctx context.Context, ports []int, usedPorts []int) ([]types.Conflict, error)
	DetectServiceConflicts(ctx context.Context, services []types.Service) ([]types.Conflict, error)
}

// ConflictResolutionStrategy は解決戦略の策定を行うインターフェースです。
type ConflictResolutionStrategy interface {
	CreatePlan(ctx context.Context, conflicts []types.Conflict, config types.PortConfig) (*types.ResolutionPlan, error)
	PrioritizeConflicts(ctx context.Context, conflicts []types.Conflict) ([]types.Conflict, error)
	SelectStrategy(ctx context.Context, conflicts []types.Conflict) (types.ResolutionStrategy, error)
}

// ConflictResolver は衝突解決の実行を行うインターフェースです。
type ConflictResolver interface {
	ResolveConflicts(ctx context.Context, conflicts []types.Conflict, config types.PortConfig) ([]types.ConflictResolution, error)
	ResolveConflict(ctx context.Context, conflict types.Conflict, config types.PortConfig) (*types.ConflictResolution, error)
	ApplyResolutions(ctx context.Context, resolutions []types.ConflictResolution, services []types.Service) ([]types.Service, error)
}

// ResolutionValidator は解決結果の検証を行うインターフェースです。
type ResolutionValidator interface {
	ValidateResolution(ctx context.Context, resolution *types.ConflictResolution) error
	ValidateResolutions(ctx context.Context, resolutions []types.ConflictResolution) error
	ValidateNoNewConflicts(ctx context.Context, resolvedServices []types.Service, usedPorts []int) error
}

// PortConflictResolver はポート衝突解決の統合インターフェースです。
type PortConflictResolver interface {
	ConflictDetector
	ConflictResolutionStrategy
	ConflictResolver
	ResolutionValidator
}

// ConflictAnalysis は衝突分析結果を表します。
type ConflictAnalysis struct {
	TotalConflicts      int                        `json:"total_conflicts"`
	ConflictsByType     map[types.ConflictType]int `json:"conflicts_by_type"`
	ConflictsBySeverity map[types.Severity]int     `json:"conflicts_by_severity"`
	AffectedServices    []string                   `json:"affected_services"`
	RecommendedStrategy types.ResolutionStrategy   `json:"recommended_strategy"`
}

// ResolutionResult は解決結果を表します。
type ResolutionResult struct {
	Resolutions        []types.ConflictResolution `json:"resolutions"`
	ResolvedServices   []types.Service            `json:"resolved_services"`
	RemainingConflicts []types.Conflict           `json:"remaining_conflicts"`
	Success            bool                       `json:"success"`
	ExecutionTime      int64                      `json:"execution_time_ms"`
}

// ConflictContext は衝突解決のコンテキスト情報を表します。
type ConflictContext struct {
	AvailablePorts []int                  `json:"available_ports"`
	ReservedPorts  []int                  `json:"reserved_ports"`
	Preferences    map[string]interface{} `json:"preferences"`
	Constraints    []ResolutionConstraint `json:"constraints"`
}

// ResolutionConstraint は解決時の制約を表します。
type ResolutionConstraint struct {
	Type        ConstraintType `json:"type"`
	Value       interface{}    `json:"value"`
	Description string         `json:"description"`
}

// ConstraintType は制約の種類を表します。
type ConstraintType string

const (
	ConstraintTypePortRange       ConstraintType = "port_range"
	ConstraintTypeExcludeServices ConstraintType = "exclude_services"
	ConstraintTypePreferredPorts  ConstraintType = "preferred_ports"
	ConstraintTypeMaxPortDistance ConstraintType = "max_port_distance"
)

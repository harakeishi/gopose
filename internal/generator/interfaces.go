// Package generator は、docker-compose.override.yml生成機能を提供します。
package generator

import (
	"context"

	"github.com/harakeishi/gopose/pkg/types"
)

// UnifiedOverrideGenerator は統一的な衝突情報からoverride生成を行うインターフェースです。
type UnifiedOverrideGenerator interface {
	GenerateFromConflicts(ctx context.Context, config *types.ComposeConfig, conflictInfo *types.UnifiedConflictInfo) (*types.OverrideConfig, error)
	ResolveConflicts(ctx context.Context, conflictInfo *types.UnifiedConflictInfo, strategy types.ResolutionStrategy, portConfig types.PortConfig) error
}

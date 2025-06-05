package resolver

import (
	"context"
	"fmt"
	"sort"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/internal/scanner"
	"github.com/harakeishi/gopose/pkg/types"
)

// ConflictDetectorImpl はポート衝突検出の実装です。
type ConflictDetectorImpl struct {
	portDetector scanner.PortDetector
	logger       logger.Logger
}

// NewConflictDetectorImpl は新しいConflictDetectorImplを作成します。
func NewConflictDetectorImpl(portDetector scanner.PortDetector, logger logger.Logger) *ConflictDetectorImpl {
	return &ConflictDetectorImpl{
		portDetector: portDetector,
		logger:       logger,
	}
}

// DetectPortConflicts はポート衝突を検出します。
func (d *ConflictDetectorImpl) DetectPortConflicts(ctx context.Context, config *types.ComposeConfig) ([]types.Conflict, error) {
	d.logger.Debug(ctx, "ポート衝突検出開始")

	var conflicts []types.Conflict

	// システムで使用中のポートを取得
	usedPorts, err := d.portDetector.DetectUsedPorts(ctx)
	if err != nil {
		return nil, fmt.Errorf("システムポート検出に失敗: %w", err)
	}

	usedPortsMap := make(map[int]bool)
	for _, port := range usedPorts {
		usedPortsMap[port] = true
	}

	// Compose内でのポート重複も検出
	composePortsMap := make(map[int]string) // port -> service name

	// 各サービスのポート設定を確認
	for serviceName, service := range config.Services {
		for _, portMapping := range service.Ports {
			if portMapping.Host == 0 {
				continue // ホストポートが指定されていない場合はスキップ
			}

			conflict := types.Conflict{
				Port:        portMapping.Host,
				Protocol:    portMapping.Protocol,
				ServiceName: serviceName,
				Type:        types.ConflictTypeNone,
			}

			// システムで使用中のポートとの衝突
			if usedPortsMap[portMapping.Host] {
				conflict.Type = types.ConflictTypeSystem
				conflict.Description = fmt.Sprintf("ポート %d は既にシステムで使用されています", portMapping.Host)
				conflicts = append(conflicts, conflict)
				d.logger.Warn(ctx, "システムポート衝突検出",
					types.Field{Key: "port", Value: portMapping.Host},
					types.Field{Key: "service", Value: serviceName})
			}

			// Compose内でのポート重複
			if existingService, exists := composePortsMap[portMapping.Host]; exists {
				conflict.Type = types.ConflictTypeCompose
				conflict.Description = fmt.Sprintf("ポート %d はサービス %s と %s で重複しています",
					portMapping.Host, existingService, serviceName)
				conflicts = append(conflicts, conflict)
				d.logger.Warn(ctx, "Compose内ポート重複検出",
					types.Field{Key: "port", Value: portMapping.Host},
					types.Field{Key: "service1", Value: existingService},
					types.Field{Key: "service2", Value: serviceName})
			} else {
				composePortsMap[portMapping.Host] = serviceName
			}
		}
	}

	d.logger.Info(ctx, "ポート衝突検出完了",
		types.Field{Key: "conflicts_count", Value: len(conflicts)})

	return conflicts, nil
}

// AnalyzeConflictSeverity は衝突の重要度を分析します。
func (d *ConflictDetectorImpl) AnalyzeConflictSeverity(ctx context.Context, conflicts []types.Conflict) map[string]types.ConflictSeverity {
	result := make(map[string]types.ConflictSeverity)

	for _, conflict := range conflicts {
		key := fmt.Sprintf("%s:%d", conflict.ServiceName, conflict.Port)

		switch conflict.Type {
		case types.ConflictTypeSystem:
			if d.isWellKnownPort(conflict.Port) {
				result[key] = types.ConflictSeverityHigh
			} else {
				result[key] = types.ConflictSeverityMedium
			}
		case types.ConflictTypeCompose:
			result[key] = types.ConflictSeverityHigh
		default:
			result[key] = types.ConflictSeverityLow
		}
	}

	d.logger.Debug(ctx, "衝突重要度分析完了",
		types.Field{Key: "analyzed_conflicts", Value: len(result)})

	return result
}

// isWellKnownPort は有名なポート番号かどうかを判定します。
func (d *ConflictDetectorImpl) isWellKnownPort(port int) bool {
	wellKnownPorts := []int{
		21, 22, 23, 25, 53, 80, 110, 143, 443, 993, 995, // 標準的なサービス
		3000, 3306, 5432, 6379, 8080, 8000, 9000, // 開発でよく使用されるポート
	}

	for _, wellKnown := range wellKnownPorts {
		if port == wellKnown {
			return true
		}
	}

	return false
}

// ConflictResolverImpl はポート衝突解決の実装です。
type ConflictResolverImpl struct {
	portAllocator scanner.PortAllocator
	logger        logger.Logger
}

// NewConflictResolverImpl は新しいConflictResolverImplを作成します。
func NewConflictResolverImpl(portAllocator scanner.PortAllocator, logger logger.Logger) *ConflictResolverImpl {
	return &ConflictResolverImpl{
		portAllocator: portAllocator,
		logger:        logger,
	}
}

// ResolvePortConflicts はポート衝突を解決します。
func (r *ConflictResolverImpl) ResolvePortConflicts(ctx context.Context, conflicts []types.Conflict, strategy types.ResolutionStrategy) ([]types.ConflictResolution, error) {
	r.logger.Debug(ctx, "ポート衝突解決開始",
		types.Field{Key: "conflicts_count", Value: len(conflicts)},
		types.Field{Key: "strategy", Value: string(strategy)})

	switch strategy {
	case types.ResolutionStrategyAutoIncrement:
		return r.resolveByAutoIncrement(ctx, conflicts)
	case types.ResolutionStrategyRangeAllocation:
		return r.resolveByRangeAllocation(ctx, conflicts)
	case types.ResolutionStrategyUserDefined:
		return r.resolveByUserDefined(ctx, conflicts)
	default:
		return r.resolveByAutoIncrement(ctx, conflicts) // デフォルト戦略
	}
}

// GenerateResolutionSuggestions は解決案を生成します。
func (r *ConflictResolverImpl) GenerateResolutionSuggestions(ctx context.Context, conflict types.Conflict) ([]types.ConflictResolution, error) {
	var suggestions []types.ConflictResolution

	// 自動インクリメント案
	autoIncrementSuggestion, err := r.generateAutoIncrementSuggestion(ctx, conflict)
	if err == nil {
		suggestions = append(suggestions, autoIncrementSuggestion)
	}

	// 範囲割り当て案
	rangeAllocationSuggestion, err := r.generateRangeAllocationSuggestion(ctx, conflict)
	if err == nil {
		suggestions = append(suggestions, rangeAllocationSuggestion)
	}

	// 特定範囲への移動案
	if conflict.Port < 8000 {
		// 8000番台への移動を提案
		targetPort := 8000 + (conflict.Port % 1000)
		suggestions = append(suggestions, types.ConflictResolution{
			ConflictPort: conflict.Port,
			ResolvedPort: targetPort,
			ServiceName:  conflict.ServiceName,
			Strategy:     types.ResolutionStrategyRangeAllocation,
			Reason:       fmt.Sprintf("開発用ポート範囲(%d)への移動", targetPort),
		})
	}

	r.logger.Debug(ctx, "解決案生成完了",
		types.Field{Key: "service", Value: conflict.ServiceName},
		types.Field{Key: "conflict_port", Value: conflict.Port},
		types.Field{Key: "suggestions_count", Value: len(suggestions)})

	return suggestions, nil
}

// resolveByAutoIncrement は自動インクリメント戦略で解決します。
func (r *ConflictResolverImpl) resolveByAutoIncrement(ctx context.Context, conflicts []types.Conflict) ([]types.ConflictResolution, error) {
	var resolutions []types.ConflictResolution

	// 既に割り当てたポートを管理するため
	allocatedPorts := make([]int, 0, len(conflicts))

	for _, conflict := range conflicts {
		// 元のポートに近い番号から開始
		startPort := conflict.Port + 1
		if startPort < 8000 {
			startPort = 8000
		}

		portConfig := types.PortConfig{
			Range:             types.PortRange{Start: startPort, End: 9000},
			ExcludePrivileged: true,
			Reserved:          allocatedPorts, // 既に割り当てたポートを除外
		}

		allocatedPort, err := r.portAllocator.AllocatePort(ctx, portConfig)
		if err != nil {
			// 元のポート+1での検索に失敗した場合は、8000番台の最初から検索
			portConfig.Range.Start = 8000
			allocatedPort, err = r.portAllocator.AllocatePort(ctx, portConfig)
			if err != nil {
				r.logger.Warn(ctx, "適切な代替ポートが見つかりません",
					types.Field{Key: "service", Value: conflict.ServiceName},
					types.Field{Key: "conflict_port", Value: conflict.Port})
				continue
			}
		}

		resolution := types.ConflictResolution{
			ConflictPort: conflict.Port,
			ResolvedPort: allocatedPort,
			ServiceName:  conflict.ServiceName,
			Strategy:     types.ResolutionStrategyAutoIncrement,
			Reason:       fmt.Sprintf("ポート %d から %d への自動変更", conflict.Port, allocatedPort),
		}
		resolutions = append(resolutions, resolution)

		// 次の割り当てのために予約済みポートに追加
		allocatedPorts = append(allocatedPorts, allocatedPort)
	}

	return resolutions, nil
}

// resolveByRangeAllocation は範囲割り当て戦略で解決します。
func (r *ConflictResolverImpl) resolveByRangeAllocation(ctx context.Context, conflicts []types.Conflict) ([]types.ConflictResolution, error) {
	var resolutions []types.ConflictResolution

	// サービスごとにポートをグループ化
	serviceGroups := make(map[string][]types.Conflict)
	for _, conflict := range conflicts {
		serviceGroups[conflict.ServiceName] = append(serviceGroups[conflict.ServiceName], conflict)
	}

	basePort := 8000
	portIncrement := 100

	for serviceName, serviceConflicts := range serviceGroups {
		// サービス専用のポート範囲を割り当て
		serviceBasePort := basePort
		basePort += portIncrement

		for i, conflict := range serviceConflicts {
			resolvedPort := serviceBasePort + i
			resolution := types.ConflictResolution{
				ConflictPort: conflict.Port,
				ResolvedPort: resolvedPort,
				ServiceName:  conflict.ServiceName,
				Strategy:     types.ResolutionStrategyRangeAllocation,
				Reason:       fmt.Sprintf("サービス %s 専用範囲 %d~ への割り当て", serviceName, serviceBasePort),
			}
			resolutions = append(resolutions, resolution)
		}
	}

	return resolutions, nil
}

// resolveByUserDefined はユーザー定義戦略で解決します。
func (r *ConflictResolverImpl) resolveByUserDefined(ctx context.Context, conflicts []types.Conflict) ([]types.ConflictResolution, error) {
	// 実際の実装では、ユーザーからの入力を受け取る仕組みが必要
	// ここでは簡略化のため、自動インクリメントと同様の処理をする
	r.logger.Info(ctx, "ユーザー定義戦略は未実装のため、自動インクリメントを使用します")
	return r.resolveByAutoIncrement(ctx, conflicts)
}

// generateAutoIncrementSuggestion は自動インクリメント提案を生成します。
func (r *ConflictResolverImpl) generateAutoIncrementSuggestion(ctx context.Context, conflict types.Conflict) (types.ConflictResolution, error) {
	nextPort := conflict.Port + 1

	// 簡単な利用可能性チェック（実際の実装ではportDetectorを使用）
	for nextPort < 65535 {
		// ここでは簡略化
		if nextPort > conflict.Port+100 {
			break // 100個先まで探して見つからなければ諦める
		}
		nextPort++
	}

	return types.ConflictResolution{
		ConflictPort: conflict.Port,
		ResolvedPort: nextPort,
		ServiceName:  conflict.ServiceName,
		Strategy:     types.ResolutionStrategyAutoIncrement,
		Reason:       fmt.Sprintf("ポート %d の次の利用可能ポート", conflict.Port),
	}, nil
}

// generateRangeAllocationSuggestion は範囲割り当て提案を生成します。
func (r *ConflictResolverImpl) generateRangeAllocationSuggestion(ctx context.Context, conflict types.Conflict) (types.ConflictResolution, error) {
	// 8000番台への割り当てを提案
	basePort := 8000
	targetPort := basePort + (conflict.Port % 1000)

	return types.ConflictResolution{
		ConflictPort: conflict.Port,
		ResolvedPort: targetPort,
		ServiceName:  conflict.ServiceName,
		Strategy:     types.ResolutionStrategyRangeAllocation,
		Reason:       fmt.Sprintf("開発用ポート範囲 8000~ への移動"),
	}, nil
}

// PortResolutionAnalyzerImpl はポート解決分析の実装です。
type PortResolutionAnalyzerImpl struct {
	logger logger.Logger
}

// NewPortResolutionAnalyzerImpl は新しいPortResolutionAnalyzerImplを作成します。
func NewPortResolutionAnalyzerImpl(logger logger.Logger) *PortResolutionAnalyzerImpl {
	return &PortResolutionAnalyzerImpl{
		logger: logger,
	}
}

// AnalyzeResolutionEffectiveness は解決案の効果を分析します。
func (a *PortResolutionAnalyzerImpl) AnalyzeResolutionEffectiveness(ctx context.Context, resolutions []types.ConflictResolution) (*ResolutionAnalysis, error) {
	a.logger.Debug(ctx, "解決案効果分析開始", types.Field{Key: "resolutions_count", Value: len(resolutions)})

	analysis := &ResolutionAnalysis{
		TotalConflicts:    len(resolutions),
		ResolvedConflicts: 0,
		StrategyStats:     make(map[types.ResolutionStrategy]int),
		PortRangeStats:    make(map[string]int),
	}

	// 戦略別統計
	for _, resolution := range resolutions {
		if resolution.ResolvedPort > 0 {
			analysis.ResolvedConflicts++
		}
		analysis.StrategyStats[resolution.Strategy]++

		// ポート範囲別統計
		rangeKey := a.getPortRangeKey(resolution.ResolvedPort)
		analysis.PortRangeStats[rangeKey]++
	}

	// 解決率の計算
	if analysis.TotalConflicts > 0 {
		analysis.SuccessRate = float64(analysis.ResolvedConflicts) / float64(analysis.TotalConflicts) * 100
	}

	a.logger.Info(ctx, "解決案効果分析完了",
		types.Field{Key: "success_rate", Value: analysis.SuccessRate},
		types.Field{Key: "resolved_conflicts", Value: analysis.ResolvedConflicts},
		types.Field{Key: "total_conflicts", Value: analysis.TotalConflicts})

	return analysis, nil
}

// OptimizeResolutions は解決案を最適化します。
func (a *PortResolutionAnalyzerImpl) OptimizeResolutions(ctx context.Context, resolutions []types.ConflictResolution) ([]types.ConflictResolution, error) {
	a.logger.Debug(ctx, "解決案最適化開始")

	// 解決案をポート番号順にソート
	sort.Slice(resolutions, func(i, j int) bool {
		return resolutions[i].ResolvedPort < resolutions[j].ResolvedPort
	})

	// 重複する解決ポートの検出と調整
	optimized := make([]types.ConflictResolution, 0, len(resolutions))
	usedPorts := make(map[int]bool)

	for _, resolution := range resolutions {
		originalResolvedPort := resolution.ResolvedPort

		// ポートが重複している場合は次の利用可能ポートを探す
		for usedPorts[resolution.ResolvedPort] {
			resolution.ResolvedPort++
			if resolution.ResolvedPort > 65535 {
				a.logger.Warn(ctx, "ポート範囲を超過しました",
					types.Field{Key: "service", Value: resolution.ServiceName},
					types.Field{Key: "original_port", Value: originalResolvedPort})
				break
			}
		}

		if resolution.ResolvedPort <= 65535 {
			usedPorts[resolution.ResolvedPort] = true

			// 最適化が発生した場合は理由を更新
			if resolution.ResolvedPort != originalResolvedPort {
				resolution.Reason = fmt.Sprintf("%s (最適化により %d から %d に調整)",
					resolution.Reason, originalResolvedPort, resolution.ResolvedPort)
			}

			optimized = append(optimized, resolution)
		}
	}

	a.logger.Info(ctx, "解決案最適化完了",
		types.Field{Key: "original_count", Value: len(resolutions)},
		types.Field{Key: "optimized_count", Value: len(optimized)})

	return optimized, nil
}

// getPortRangeKey はポート番号に基づいて範囲キーを返します。
func (a *PortResolutionAnalyzerImpl) getPortRangeKey(port int) string {
	switch {
	case port < 1024:
		return "system_ports"
	case port < 5000:
		return "registered_ports"
	case port < 8000:
		return "custom_ports"
	case port < 9000:
		return "development_ports"
	default:
		return "high_ports"
	}
}

// ResolutionAnalysis は解決案分析結果を表します。
type ResolutionAnalysis struct {
	TotalConflicts    int                              `json:"total_conflicts"`
	ResolvedConflicts int                              `json:"resolved_conflicts"`
	SuccessRate       float64                          `json:"success_rate"`
	StrategyStats     map[types.ResolutionStrategy]int `json:"strategy_stats"`
	PortRangeStats    map[string]int                   `json:"port_range_stats"`
}

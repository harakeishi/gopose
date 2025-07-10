package generator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/internal/scanner"
	"github.com/harakeishi/gopose/pkg/types"
)

// UnifiedOverrideGeneratorImpl は統一的な衝突情報からoverride生成を行う実装です。
type UnifiedOverrideGeneratorImpl struct {
	portAllocator scanner.PortAllocator
	logger        logger.Logger
}

// NewUnifiedOverrideGeneratorImpl は新しいUnifiedOverrideGeneratorImplを作成します。
func NewUnifiedOverrideGeneratorImpl(portAllocator scanner.PortAllocator, logger logger.Logger) *UnifiedOverrideGeneratorImpl {
	return &UnifiedOverrideGeneratorImpl{
		portAllocator: portAllocator,
		logger:        logger,
	}
}

// GenerateFromConflicts は統一的な衝突情報からoverride設定を生成します。
func (u *UnifiedOverrideGeneratorImpl) GenerateFromConflicts(ctx context.Context, config *types.ComposeConfig, conflictInfo *types.UnifiedConflictInfo) (*types.OverrideConfig, error) {
	u.logger.Debug(ctx, "統一的なOverride生成開始",
		types.Field{Key: "port_conflicts", Value: len(conflictInfo.PortConflicts)},
		types.Field{Key: "network_conflicts", Value: len(conflictInfo.NetworkConflicts)})

	// OverrideConfigの基本構造を作成
	override := &types.OverrideConfig{
		Version:  config.Version,
		Services: make(map[string]types.ServiceOverride),
		Networks: make(map[string]types.NetworkOverride),
		Metadata: types.OverrideMetadata{
			GeneratedAt: time.Now(),
			Version:     "1.0.0",
			Resolutions: []types.ConflictResolution{}, // 統一的解決情報から変換
		},
	}

	// ポート衝突の解決とサービスオーバーライド生成
	if err := u.generatePortOverrides(ctx, config, conflictInfo.PortConflicts, override); err != nil {
		return nil, fmt.Errorf("ポートオーバーライド生成に失敗: %w", err)
	}

	// ネットワーク衝突の解決とネットワークオーバーライド生成
	if err := u.generateNetworkOverrides(ctx, config, conflictInfo.NetworkConflicts, override); err != nil {
		return nil, fmt.Errorf("ネットワークオーバーライド生成に失敗: %w", err)
	}

	// メタデータに解決情報を追加
	u.populateMetadata(conflictInfo, override)

	u.logger.Info(ctx, "統一的なOverride生成完了",
		types.Field{Key: "services_count", Value: len(override.Services)},
		types.Field{Key: "networks_count", Value: len(override.Networks)})

	return override, nil
}

// ResolveConflicts は衝突情報を解決します。
func (u *UnifiedOverrideGeneratorImpl) ResolveConflicts(ctx context.Context, conflictInfo *types.UnifiedConflictInfo, strategy types.ResolutionStrategy, portConfig types.PortConfig) error {
	// ポート衝突の解決
	if err := u.resolvePortConflicts(ctx, conflictInfo.PortConflicts, strategy, portConfig); err != nil {
		return fmt.Errorf("ポート衝突解決に失敗: %w", err)
	}

	// ネットワーク衝突の解決
	if err := u.resolveNetworkConflicts(ctx, conflictInfo.NetworkConflicts); err != nil {
		return fmt.Errorf("ネットワーク衝突解決に失敗: %w", err)
	}

	return nil
}

// generatePortOverrides はポート衝突のオーバーライドを生成します。
func (u *UnifiedOverrideGeneratorImpl) generatePortOverrides(ctx context.Context, config *types.ComposeConfig, portConflicts []types.PortConflictInfo, override *types.OverrideConfig) error {
	// サービス別にポート衝突をグループ化
	serviceConflicts := make(map[string][]types.PortConflictInfo)
	for _, conflict := range portConflicts {
		serviceName := conflict.ServiceName
		if serviceName == "" {
			serviceName = conflict.Service
		}
		serviceConflicts[serviceName] = append(serviceConflicts[serviceName], conflict)
	}

	// 各サービスのポートオーバーライドを生成
	for serviceName, conflicts := range serviceConflicts {
		originalService, exists := config.Services[serviceName]
		if !exists {
			continue
		}

		serviceOverride := types.ServiceOverride{
			Ports: make([]types.PortMapping, len(originalService.Ports)),
		}
		copy(serviceOverride.Ports, originalService.Ports)

		// 解決済みポートで更新
		for _, conflict := range conflicts {
			if conflict.Resolution != nil {
				for i, mapping := range serviceOverride.Ports {
					if mapping.Host == conflict.Port {
						serviceOverride.Ports[i].Host = conflict.Resolution.ResolvedPort
						break
					}
				}
			}
		}

		override.Services[serviceName] = serviceOverride
	}

	return nil
}

// generateNetworkOverrides はネットワーク衝突のオーバーライドを生成します。
func (u *UnifiedOverrideGeneratorImpl) generateNetworkOverrides(ctx context.Context, config *types.ComposeConfig, networkConflicts []types.NetworkConflictInfo, override *types.OverrideConfig) error {
	for _, conflict := range networkConflicts {
		if conflict.Resolution != nil {
			// ネットワークオーバーライドを生成
			override.Networks[conflict.NetworkName] = types.NetworkOverride{
				IPAM: types.IPAM{
					Config: []types.IPAMConfig{{
						Subnet: conflict.Resolution.ResolvedSubnet,
					}},
				},
			}

			// サービスのIPアドレス再割り当てが必要な場合
			if len(conflict.Resolution.ServiceIPs) > 0 {
				for serviceName, newIP := range conflict.Resolution.ServiceIPs {
					if override.Services == nil {
						override.Services = make(map[string]types.ServiceOverride)
					}

					serviceOverride, exists := override.Services[serviceName]
					if !exists {
						serviceOverride = types.ServiceOverride{}
					}

					if serviceOverride.Networks == nil {
						serviceOverride.Networks = make(map[string]types.ServiceNetwork)
					}

					serviceOverride.Networks[conflict.NetworkName] = types.ServiceNetwork{
						IPv4Address: newIP,
					}

					override.Services[serviceName] = serviceOverride
				}
			}
		}
	}

	return nil
}

// populateMetadata はメタデータに解決情報を追加します。
func (u *UnifiedOverrideGeneratorImpl) populateMetadata(conflictInfo *types.UnifiedConflictInfo, override *types.OverrideConfig) {
	var resolutions []types.ConflictResolution

	// ポート解決情報を変換
	for _, conflict := range conflictInfo.PortConflicts {
		if conflict.Resolution != nil {
			resolution := types.ConflictResolution{
				ServiceName:  conflict.ServiceName,
				Service:      conflict.Service,
				ConflictPort: conflict.Port,
				ResolvedPort: conflict.Resolution.ResolvedPort,
				Strategy:     conflict.Resolution.Strategy,
				Reason:       conflict.Resolution.Reason,
				Timestamp:    conflictInfo.GeneratedAt,
			}
			resolutions = append(resolutions, resolution)
		}
	}

	override.Metadata.Resolutions = resolutions
}

// resolvePortConflicts はポート衝突を解決します。
func (u *UnifiedOverrideGeneratorImpl) resolvePortConflicts(ctx context.Context, portConflicts []types.PortConflictInfo, strategy types.ResolutionStrategy, portConfig types.PortConfig) error {
	// 既に割り当てたポートを管理
	allocatedPorts := make([]int, 0, len(portConflicts))

	for i := range portConflicts {
		conflict := &portConflicts[i]

		// 元のポートに近い番号から開始
		startPort := conflict.Port + 1
		if startPort < portConfig.Range.Start {
			startPort = portConfig.Range.Start
		}

		config := types.PortConfig{
			Range:             types.PortRange{Start: startPort, End: portConfig.Range.End},
			ExcludePrivileged: portConfig.ExcludePrivileged,
			Reserved:          append(allocatedPorts, portConfig.Reserved...),
		}

		allocatedPort, err := u.portAllocator.AllocatePort(ctx, config)
		if err != nil {
			// 元のポート+1での検索に失敗した場合は、設定された範囲の最初から検索
			config.Range.Start = portConfig.Range.Start
			allocatedPort, err = u.portAllocator.AllocatePort(ctx, config)
			if err != nil {
				u.logger.Warn(ctx, "適切な代替ポートが見つかりません",
					types.Field{Key: "service", Value: conflict.ServiceName},
					types.Field{Key: "conflict_port", Value: conflict.Port})
				continue
			}
		}

		// 解決情報を設定
		conflict.Resolution = &types.PortResolutionInfo{
			ResolvedPort: allocatedPort,
			Strategy:     strategy,
			Reason:       fmt.Sprintf("ポート %d から %d への自動変更", conflict.Port, allocatedPort),
		}

		// 次の割り当てのために予約済みポートに追加
		allocatedPorts = append(allocatedPorts, allocatedPort)

		u.logger.Info(ctx, "ポート衝突解決",
			types.Field{Key: "service", Value: conflict.ServiceName},
			types.Field{Key: "from", Value: conflict.Port},
			types.Field{Key: "to", Value: allocatedPort})
	}

	return nil
}

// resolveNetworkConflicts はネットワーク衝突を解決します。
func (u *UnifiedOverrideGeneratorImpl) resolveNetworkConflicts(ctx context.Context, networkConflicts []types.NetworkConflictInfo) error {
	usedSubnets := make(map[string]bool)

	for i := range networkConflicts {
		conflict := &networkConflicts[i]

		newSubnet := u.allocateNewSubnet(usedSubnets)
		if newSubnet == "" {
			u.logger.Warn(ctx, "利用可能なサブネットが見つかりません",
				types.Field{Key: "network", Value: conflict.NetworkName})
			continue
		}
		usedSubnets[newSubnet] = true

		// サービスIPアドレスの再マッピング
		var newServiceIPs map[string]string
		if len(conflict.ServiceIPs) > 0 {
			var err error
			newServiceIPs, err = u.remapIPAddressesToNewSubnet(conflict.OriginalSubnet, newSubnet, conflict.ServiceIPs)
			if err != nil {
				u.logger.Warn(ctx, "サービスIPアドレスの再マッピングに失敗",
					types.Field{Key: "network", Value: conflict.NetworkName},
					types.Field{Key: "error", Value: err.Error()})
			}
		}

		// 解決情報を設定
		conflict.Resolution = &types.NetworkResolutionInfo{
			ResolvedSubnet: newSubnet,
			ServiceIPs:     newServiceIPs,
			Reason:         fmt.Sprintf("サブネット %s から %s への自動変更", conflict.OriginalSubnet, newSubnet),
		}

		u.logger.Info(ctx, "ネットワーク衝突解決",
			types.Field{Key: "network", Value: conflict.NetworkName},
			types.Field{Key: "from", Value: conflict.OriginalSubnet},
			types.Field{Key: "to", Value: newSubnet})
	}

	return nil
}

// allocateNewSubnet は新しいサブネットを割り当てます。
func (u *UnifiedOverrideGeneratorImpl) allocateNewSubnet(used map[string]bool) string {
	// 10.x.x.x/24 範囲（最も安全）
	for i := 20; i <= 255; i++ {
		subnet := fmt.Sprintf("10.%d.0.0/24", i)
		if !used[subnet] {
			return subnet
		}
	}

	// 192.168.x.x/24 範囲（一般的なホームルーター範囲を回避）
	for i := 100; i <= 255; i++ {
		subnet := fmt.Sprintf("192.168.%d.0/24", i)
		if !used[subnet] {
			return subnet
		}
	}

	// 172.x.x.x/24 範囲（Dockerデフォルト範囲を回避）
	for i := 30; i <= 255; i++ {
		if i >= 17 && i <= 29 {
			continue // Dockerデフォルト範囲をスキップ
		}
		subnet := fmt.Sprintf("172.%d.0.0/24", i)
		if !used[subnet] {
			return subnet
		}
	}

	return "" // 利用可能なサブネットが見つからない
}

// remapIPAddressesToNewSubnet はIPアドレスを新しいサブネットに再マッピングします。
func (u *UnifiedOverrideGeneratorImpl) remapIPAddressesToNewSubnet(oldSubnet, newSubnet string, serviceIPs map[string]string) (map[string]string, error) {
	// 簡単な実装：同じホスト部分を維持
	newServiceIPs := make(map[string]string)

	for serviceName, oldIP := range serviceIPs {
		// 簡易的な変換（実際のプロダクションでは、より厳密なCIDR処理が必要）
		// ここでは、最後のオクテットを保持する簡単な実装
		parts := strings.Split(oldIP, ".")
		if len(parts) != 4 {
			continue
		}

		newParts := strings.Split(newSubnet, "/")[0]
		newSubnetParts := strings.Split(newParts, ".")
		if len(newSubnetParts) != 4 {
			continue
		}

		// 新しいサブネットのネットワーク部分 + 元のホスト部分
		newIP := fmt.Sprintf("%s.%s.%s.%s", newSubnetParts[0], newSubnetParts[1], newSubnetParts[2], parts[3])
		newServiceIPs[serviceName] = newIP
	}

	return newServiceIPs, nil
}

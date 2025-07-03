package resolver

import (
	"context"
	"fmt"
	"net"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/internal/scanner"
	"github.com/harakeishi/gopose/pkg/types"
)

// NetworkConflictDetector はネットワーク衝突検出のインターフェースです。
type NetworkConflictDetector interface {
	DetectNetworkConflicts(ctx context.Context, config *types.ComposeConfig, projectName string) ([]types.NetworkConflict, error)
}

// NetworkConflictResolver はネットワーク衝突解決のインターフェースです。
type NetworkConflictResolver interface {
	ResolveNetworkConflicts(ctx context.Context, conflicts []types.NetworkConflict) ([]types.NetworkConflictResolution, error)
}

// NetworkConflictDetectorImpl はネットワーク衝突検出の実装です。
type NetworkConflictDetectorImpl struct {
	networkDetector scanner.NetworkDetector
	logger          logger.Logger
}

// NewNetworkConflictDetectorImpl は新しいNetworkConflictDetectorImplを作成します。
func NewNetworkConflictDetectorImpl(networkDetector scanner.NetworkDetector, logger logger.Logger) *NetworkConflictDetectorImpl {
	return &NetworkConflictDetectorImpl{
		networkDetector: networkDetector,
		logger:          logger,
	}
}

// DetectNetworkConflicts はネットワーク衝突を検出します。
func (d *NetworkConflictDetectorImpl) DetectNetworkConflicts(ctx context.Context, config *types.ComposeConfig, projectName string) ([]types.NetworkConflict, error) {
	d.logger.Debug(ctx, "ネットワーク衝突検出開始")

	// 既存のDockerネットワークを検出
	dockerNetworks, err := d.networkDetector.DetectNetworks(ctx)
	if err != nil {
		return nil, fmt.Errorf("Dockerネットワーク検出に失敗: %w", err)
	}

	// Composeファイルからネットワーク設定を取得
	composeNetworks := d.extractComposeNetworks(config)

	var conflicts []types.NetworkConflict
	usedSubnets := make(map[string]bool)
	usedNetworkNames := make(map[string]bool)

	// 既存のDockerネットワークを記録
	for _, network := range dockerNetworks {
		if network.Subnet != "" {
			usedSubnets[network.Subnet] = true
		}
		usedNetworkNames[network.Name] = true
	}

	// 各Composeネットワークの衝突をチェック
	for networkName, networkConfig := range composeNetworks {
		// プロジェクト名を含む実際のネットワーク名を生成
		actualNetworkName := d.generateActualNetworkName(projectName, networkName)

		conflict := types.NetworkConflict{
			NetworkName:    networkName,
			ActualName:     actualNetworkName,
			OriginalSubnet: networkConfig.Subnet,
			ConflictType:   types.NetworkConflictTypeNone,
		}

		// サブネット衝突をチェック
		if networkConfig.Subnet != "" && usedSubnets[networkConfig.Subnet] {
			conflict.ConflictType = types.NetworkConflictTypeSubnet
			conflict.Description = fmt.Sprintf("サブネット %s は既に使用されています", networkConfig.Subnet)
			conflicts = append(conflicts, conflict)

			d.logger.Warn(ctx, "サブネット衝突検出",
				types.Field{Key: "network", Value: networkName},
				types.Field{Key: "subnet", Value: networkConfig.Subnet})
		}

		// ネットワーク名衝突をチェック
		if usedNetworkNames[actualNetworkName] {
			// サブネット衝突がない場合のみネットワーク名衝突として記録
			if conflict.ConflictType == types.NetworkConflictTypeNone {
				conflict.ConflictType = types.NetworkConflictTypeName
				conflict.Description = fmt.Sprintf("ネットワーク名 %s は既に使用されています", actualNetworkName)
				conflicts = append(conflicts, conflict)

				d.logger.Warn(ctx, "ネットワーク名衝突検出",
					types.Field{Key: "network", Value: networkName},
					types.Field{Key: "actual_name", Value: actualNetworkName})
			}
		}

		// 使用済みとしてマーク
		if networkConfig.Subnet != "" {
			usedSubnets[networkConfig.Subnet] = true
		}
		usedNetworkNames[actualNetworkName] = true
	}

	d.logger.Info(ctx, "ネットワーク衝突検出完了",
		types.Field{Key: "conflicts_count", Value: len(conflicts)})

	return conflicts, nil
}

// extractComposeNetworks はComposeファイルからネットワーク設定を抽出します。
func (d *NetworkConflictDetectorImpl) extractComposeNetworks(config *types.ComposeConfig) map[string]types.NetworkConfig {
	networks := make(map[string]types.NetworkConfig)

	if config.Networks != nil {
		for networkName, networkConfig := range config.Networks {
			networks[networkName] = networkConfig
		}
	}

	return networks
}

// generateActualNetworkName はプロジェクト名を含む実際のネットワーク名を生成します。
func (d *NetworkConflictDetectorImpl) generateActualNetworkName(projectName, networkName string) string {
	// Docker Composeの命名規則: {project}_{network}
	return fmt.Sprintf("%s_%s", projectName, networkName)
}

// NetworkConflictResolverImpl はネットワーク衝突解決の実装です。
type NetworkConflictResolverImpl struct {
	logger logger.Logger
}

// NewNetworkConflictResolverImpl は新しいNetworkConflictResolverImplを作成します。
func NewNetworkConflictResolverImpl(logger logger.Logger) *NetworkConflictResolverImpl {
	return &NetworkConflictResolverImpl{
		logger: logger,
	}
}

// ResolveNetworkConflicts はネットワーク衝突を解決します。
func (r *NetworkConflictResolverImpl) ResolveNetworkConflicts(ctx context.Context, conflicts []types.NetworkConflict) ([]types.NetworkConflictResolution, error) {
	r.logger.Debug(ctx, "ネットワーク衝突解決開始", types.Field{Key: "conflicts_count", Value: len(conflicts)})

	var resolutions []types.NetworkConflictResolution
	usedSubnets := make(map[string]bool)

	for _, conflict := range conflicts {
		switch conflict.ConflictType {
		case types.NetworkConflictTypeSubnet:
			resolution, err := r.resolveSubnetConflict(ctx, conflict, usedSubnets)
			if err != nil {
				r.logger.Warn(ctx, "サブネット衝突解決に失敗",
					types.Field{Key: "network", Value: conflict.NetworkName},
					types.Field{Key: "error", Value: err.Error()})
				continue
			}
			resolutions = append(resolutions, resolution)
			usedSubnets[resolution.ResolvedSubnet] = true

		case types.NetworkConflictTypeName:
			resolution, err := r.resolveNetworkNameConflict(ctx, conflict)
			if err != nil {
				r.logger.Warn(ctx, "ネットワーク名衝突解決に失敗",
					types.Field{Key: "network", Value: conflict.NetworkName},
					types.Field{Key: "error", Value: err.Error()})
				continue
			}
			resolutions = append(resolutions, resolution)
		}
	}

	r.logger.Info(ctx, "ネットワーク衝突解決完了",
		types.Field{Key: "resolutions_count", Value: len(resolutions)})

	return resolutions, nil
}

// resolveSubnetConflict はサブネット衝突を解決します。
func (r *NetworkConflictResolverImpl) resolveSubnetConflict(ctx context.Context, conflict types.NetworkConflict, usedSubnets map[string]bool) (types.NetworkConflictResolution, error) {
	// 新しいサブネットを生成
	newSubnet, err := r.allocateNewSubnet(usedSubnets)
	if err != nil {
		return types.NetworkConflictResolution{}, fmt.Errorf("新しいサブネット割り当てに失敗: %w", err)
	}

	resolution := types.NetworkConflictResolution{
		NetworkName:      conflict.NetworkName,
		ConflictType:     conflict.ConflictType,
		OriginalSubnet:   conflict.OriginalSubnet,
		ResolvedSubnet:   newSubnet,
		IPAddressMapping: make(map[string]string),
		Reason:           fmt.Sprintf("サブネット %s から %s への変更", conflict.OriginalSubnet, newSubnet),
	}

	r.logger.Debug(ctx, "サブネット衝突解決",
		types.Field{Key: "network", Value: conflict.NetworkName},
		types.Field{Key: "original_subnet", Value: conflict.OriginalSubnet},
		types.Field{Key: "resolved_subnet", Value: newSubnet})

	return resolution, nil
}

// resolveNetworkNameConflict はネットワーク名衝突を解決します。
func (r *NetworkConflictResolverImpl) resolveNetworkNameConflict(ctx context.Context, conflict types.NetworkConflict) (types.NetworkConflictResolution, error) {
	// ネットワーク名衝突の場合、通常は新しいプロジェクト名を生成するが、
	// ここでは既存のサブネットを使用することで対処
	resolution := types.NetworkConflictResolution{
		NetworkName:    conflict.NetworkName,
		ConflictType:   conflict.ConflictType,
		OriginalSubnet: conflict.OriginalSubnet,
		ResolvedSubnet: conflict.OriginalSubnet, // サブネットは変更しない
		Reason:         fmt.Sprintf("ネットワーク名 %s の衝突を検出（プロジェクト名の変更を推奨）", conflict.ActualName),
	}

	r.logger.Debug(ctx, "ネットワーク名衝突解決",
		types.Field{Key: "network", Value: conflict.NetworkName},
		types.Field{Key: "actual_name", Value: conflict.ActualName})

	return resolution, nil
}

// allocateNewSubnet は新しいサブネットを割り当てます。
func (r *NetworkConflictResolverImpl) allocateNewSubnet(usedSubnets map[string]bool) (string, error) {
	// プライベートアドレス空間から新しいサブネットを生成
	// 172.16.0.0/16 の範囲を使用
	for i := 16; i < 32; i++ {
		subnet := fmt.Sprintf("172.%d.0.0/16", i)
		if !usedSubnets[subnet] {
			return subnet, nil
		}
	}

	// 10.0.0.0/8 の範囲を使用
	for i := 1; i < 255; i++ {
		subnet := fmt.Sprintf("10.%d.0.0/16", i)
		if !usedSubnets[subnet] {
			return subnet, nil
		}
	}

	return "", fmt.Errorf("利用可能なサブネットが見つかりません")
}

// RemapIPAddressesToNewSubnet は指定されたサブネットのIPアドレスを新しいサブネットに再マッピングします。
func (r *NetworkConflictResolverImpl) RemapIPAddressesToNewSubnet(ctx context.Context, originalSubnet, newSubnet string, serviceIPs map[string]string) (map[string]string, error) {
	r.logger.Debug(ctx, "IPアドレス再マッピング開始",
		types.Field{Key: "original_subnet", Value: originalSubnet},
		types.Field{Key: "new_subnet", Value: newSubnet},
		types.Field{Key: "service_count", Value: len(serviceIPs)})

	// 元のサブネットと新しいサブネットをパース
	originalIP, originalNet, err := net.ParseCIDR(originalSubnet)
	if err != nil {
		return nil, fmt.Errorf("元のサブネットのパースに失敗: %w", err)
	}

	newIP, newNet, err := net.ParseCIDR(newSubnet)
	if err != nil {
		return nil, fmt.Errorf("新しいサブネットのパースに失敗: %w", err)
	}

	newServiceIPs := make(map[string]string)

	for serviceName, ipAddress := range serviceIPs {
		// IPアドレスをパース
		serviceIP := net.ParseIP(ipAddress)
		if serviceIP == nil {
			r.logger.Warn(ctx, "無効なIPアドレス",
				types.Field{Key: "service", Value: serviceName},
				types.Field{Key: "ip", Value: ipAddress})
			continue
		}

		// 元のサブネットの範囲内かチェック
		if !originalNet.Contains(serviceIP) {
			r.logger.Warn(ctx, "IPアドレスが元のサブネット範囲外",
				types.Field{Key: "service", Value: serviceName},
				types.Field{Key: "ip", Value: ipAddress},
				types.Field{Key: "subnet", Value: originalSubnet})
			continue
		}

		// 新しいサブネットでの相対位置を計算
		originalBase := originalIP.To4()
		newBase := newIP.To4()
		serviceIPv4 := serviceIP.To4()

		if originalBase == nil || newBase == nil || serviceIPv4 == nil {
			r.logger.Warn(ctx, "IPv4アドレスの処理に失敗",
				types.Field{Key: "service", Value: serviceName})
			continue
		}

		// 相対オフセットを計算
		offset := make([]int, 4)
		for i := 0; i < 4; i++ {
			offset[i] = int(serviceIPv4[i]) - int(originalBase[i])
		}

		// 新しいIPアドレスを生成
		newIPBytes := make([]byte, 4)
		for i := 0; i < 4; i++ {
			newIPBytes[i] = byte(int(newBase[i]) + offset[i])
		}

		newServiceIP := net.IP(newIPBytes).String()

		// 新しいサブネットの範囲内かチェック
		if !newNet.Contains(net.ParseIP(newServiceIP)) {
			r.logger.Warn(ctx, "新しいIPアドレスがサブネット範囲外",
				types.Field{Key: "service", Value: serviceName},
				types.Field{Key: "new_ip", Value: newServiceIP},
				types.Field{Key: "new_subnet", Value: newSubnet})
			continue
		}

		newServiceIPs[serviceName] = newServiceIP
		r.logger.Debug(ctx, "IPアドレス再マッピング",
			types.Field{Key: "service", Value: serviceName},
			types.Field{Key: "original_ip", Value: ipAddress},
			types.Field{Key: "new_ip", Value: newServiceIP})
	}

	r.logger.Info(ctx, "IPアドレス再マッピング完了",
		types.Field{Key: "remapped_count", Value: len(newServiceIPs)})

	return newServiceIPs, nil
}

// GetServiceNetworkIPs は指定されたネットワークを使用するサービスのIPアドレスを取得します。
func (r *NetworkConflictResolverImpl) GetServiceNetworkIPs(ctx context.Context, config *types.ComposeConfig, networkName string) (map[string]string, error) {
	serviceIPs := make(map[string]string)

	for serviceName, service := range config.Services {
		if service.Networks != nil {
			for _, networkConfig := range service.Networks {
				if networkConfig.Name == networkName && networkConfig.IPv4Address != "" {
					serviceIPs[serviceName] = networkConfig.IPv4Address
					break
				}
			}
		}
	}

	r.logger.Debug(ctx, "サービスIPアドレス取得完了",
		types.Field{Key: "network", Value: networkName},
		types.Field{Key: "service_count", Value: len(serviceIPs)})

	return serviceIPs, nil
}

// GenerateNetworkOverride はネットワーク衝突解決結果からoverride設定を生成します。
func (r *NetworkConflictResolverImpl) GenerateNetworkOverride(ctx context.Context, resolutions []types.NetworkConflictResolution, config *types.ComposeConfig) (*types.ComposeConfig, error) {
	r.logger.Debug(ctx, "ネットワークoverride生成開始",
		types.Field{Key: "resolutions_count", Value: len(resolutions)})

	// 元の設定のコピーを作成
	overrideConfig := &types.ComposeConfig{
		Networks: make(map[string]types.NetworkConfig),
		Services: make(map[string]types.Service),
	}

	// ネットワーク設定の更新
	for _, resolution := range resolutions {
		if resolution.ConflictType == types.NetworkConflictTypeSubnet {
			networkConfig := types.NetworkConfig{
				Subnet: resolution.ResolvedSubnet,
			}

			// 元の設定から他のプロパティをコピー
			if originalConfig, exists := config.Networks[resolution.NetworkName]; exists {
				networkConfig.Driver = originalConfig.Driver
				networkConfig.External = originalConfig.External
				// 他のプロパティも必要に応じてコピー
			}

			overrideConfig.Networks[resolution.NetworkName] = networkConfig
		}
	}

	// サービス設定の更新（IPアドレスの再マッピング）
	for _, resolution := range resolutions {
		if len(resolution.IPAddressMapping) > 0 {
			for serviceName, newIP := range resolution.IPAddressMapping {
				if originalService, exists := config.Services[serviceName]; exists {
					// サービスの設定をコピー
					serviceConfig := originalService

					// ネットワーク設定を更新
					if serviceConfig.Networks != nil {
						for i, networkConfig := range serviceConfig.Networks {
							if networkConfig.Name == resolution.NetworkName {
								serviceConfig.Networks[i].IPv4Address = newIP
								break
							}
						}
					}

					overrideConfig.Services[serviceName] = serviceConfig
				}
			}
		}
	}

	r.logger.Info(ctx, "ネットワークoverride生成完了",
		types.Field{Key: "networks_count", Value: len(overrideConfig.Networks)},
		types.Field{Key: "services_count", Value: len(overrideConfig.Services)})

	return overrideConfig, nil
}
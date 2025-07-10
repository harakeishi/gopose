package scanner

import (
	"context"
	"fmt"
	"time"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

// UnifiedConflictDetectorImpl は統一的な衝突検知の実装です。
type UnifiedConflictDetectorImpl struct {
	portDetector    PortDetector
	networkDetector NetworkDetector
	logger          logger.Logger
}

// NewUnifiedConflictDetectorImpl は新しいUnifiedConflictDetectorImplを作成します。
func NewUnifiedConflictDetectorImpl(portDetector PortDetector, networkDetector NetworkDetector, logger logger.Logger) *UnifiedConflictDetectorImpl {
	return &UnifiedConflictDetectorImpl{
		portDetector:    portDetector,
		networkDetector: networkDetector,
		logger:          logger,
	}
}

// DetectConflicts は統一的な衝突検知を実行します。
func (u *UnifiedConflictDetectorImpl) DetectConflicts(ctx context.Context, config *types.ComposeConfig, projectName string) (*types.UnifiedConflictInfo, error) {
	u.logger.Info(ctx, "統一的な衝突検知を開始")

	conflictInfo := &types.UnifiedConflictInfo{
		GeneratedAt: time.Now(),
	}

	// ポート衝突検知
	portConflicts, err := u.DetectPortConflicts(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("ポート衝突検知に失敗: %w", err)
	}
	conflictInfo.PortConflicts = portConflicts

	// ネットワーク衝突検知
	networkConflicts, err := u.DetectNetworkConflicts(ctx, config, projectName)
	if err != nil {
		u.logger.Warn(ctx, "ネットワーク衝突検知に失敗しました",
			types.Field{Key: "error", Value: err.Error()})
		conflictInfo.NetworkConflicts = []types.NetworkConflictInfo{}
	} else {
		conflictInfo.NetworkConflicts = networkConflicts
	}

	u.logger.Info(ctx, "統一的な衝突検知完了",
		types.Field{Key: "port_conflicts", Value: len(conflictInfo.PortConflicts)},
		types.Field{Key: "network_conflicts", Value: len(conflictInfo.NetworkConflicts)})

	return conflictInfo, nil
}

// DetectPortConflicts はポート衝突検知を実行します。
func (u *UnifiedConflictDetectorImpl) DetectPortConflicts(ctx context.Context, config *types.ComposeConfig) ([]types.PortConflictInfo, error) {
	u.logger.Debug(ctx, "ポート衝突検知開始")

	var conflicts []types.PortConflictInfo

	// システムで使用中のポートを取得
	usedPorts, err := u.portDetector.DetectUsedPorts(ctx)
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

			conflict := types.PortConflictInfo{
				Port:        portMapping.Host,
				Protocol:    portMapping.Protocol,
				ServiceName: serviceName,
				Service:     serviceName,
			}

			// システムで使用中のポートとの衝突
			if usedPortsMap[portMapping.Host] {
				conflict.Type = types.ConflictTypeSystem
				conflict.Description = fmt.Sprintf("ポート %d は既にシステムで使用されています", portMapping.Host)
				conflicts = append(conflicts, conflict)
				u.logger.Warn(ctx, "システムポート衝突検出",
					types.Field{Key: "port", Value: portMapping.Host},
					types.Field{Key: "service", Value: serviceName})
			} else if existingService, exists := composePortsMap[portMapping.Host]; exists {
				// Compose内でのポート重複
				conflict.Type = types.ConflictTypeCompose
				conflict.Description = fmt.Sprintf("ポート %d はサービス %s と %s で重複しています",
					portMapping.Host, existingService, serviceName)
				conflicts = append(conflicts, conflict)
				u.logger.Warn(ctx, "Composeポート衝突検出",
					types.Field{Key: "port", Value: portMapping.Host},
					types.Field{Key: "service1", Value: existingService},
					types.Field{Key: "service2", Value: serviceName})
			} else {
				composePortsMap[portMapping.Host] = serviceName
			}
		}
	}

	u.logger.Debug(ctx, "ポート衝突検知完了",
		types.Field{Key: "conflicts_count", Value: len(conflicts)})

	return conflicts, nil
}

// DetectNetworkConflicts はネットワーク衝突検知を実行します。
func (u *UnifiedConflictDetectorImpl) DetectNetworkConflicts(ctx context.Context, config *types.ComposeConfig, projectName string) ([]types.NetworkConflictInfo, error) {
	u.logger.Debug(ctx, "ネットワーク衝突検知開始")

	var conflicts []types.NetworkConflictInfo

	// 既存Dockerネットワークを取得
	dockerNets, err := u.networkDetector.DetectNetworks(ctx)
	if err != nil {
		return nil, fmt.Errorf("既存Dockerネットワークの検出に失敗: %w", err)
	}

	usedSubnets := make(map[string]bool)
	usedNetworkNames := make(map[string]bool)
	for _, n := range dockerNets {
		usedNetworkNames[n.Name] = true
		for _, s := range n.Subnets {
			usedSubnets[s] = true
		}
	}

	// プロジェクト名が設定されている場合、動的ネットワーク名も考慮
	projectPrefix := ""
	if projectName != "" {
		projectPrefix = projectName + "_"
	}

	// Composeネットワークを確認
	for netName, network := range config.Networks {
		if len(network.IPAM.Config) == 0 {
			continue
		}

		subnet := network.IPAM.Config[0].Subnet
		if subnet == "" {
			continue
		}

		actualNetworkName := projectPrefix + netName

		// ネットワーク名の衝突をチェック
		if usedNetworkNames[actualNetworkName] {
			conflict := types.NetworkConflictInfo{
				NetworkName:        netName,
				ConflictType:       types.NetworkConflictTypeName,
				OriginalSubnet:     subnet,
				ConflictingNetwork: actualNetworkName,
				Description:        fmt.Sprintf("ネットワーク名 %s は既に使用されています", actualNetworkName),
			}
			conflicts = append(conflicts, conflict)
		}

		// サブネット衝突をチェック
		if usedSubnets[subnet] {
			conflict := types.NetworkConflictInfo{
				NetworkName:       netName,
				ConflictType:      types.NetworkConflictTypeSubnet,
				OriginalSubnet:    subnet,
				ConflictingSubnet: subnet,
				Description:       fmt.Sprintf("サブネット %s は既に使用されています", subnet),
			}

			// サービスIPアドレスも取得
			serviceIPs := u.getServiceNetworkIPs(config, netName)
			if len(serviceIPs) > 0 {
				conflict.ServiceIPs = serviceIPs
			}

			conflicts = append(conflicts, conflict)
		}
	}

	u.logger.Debug(ctx, "ネットワーク衝突検知完了",
		types.Field{Key: "conflicts_count", Value: len(conflicts)})

	return conflicts, nil
}

// getServiceNetworkIPs はネットワーク内のサービスIPアドレスを取得します。
func (u *UnifiedConflictDetectorImpl) getServiceNetworkIPs(config *types.ComposeConfig, networkName string) map[string]string {
	serviceIPs := make(map[string]string)

	for serviceName, service := range config.Services {
		if service.Networks != nil {
			if netConfig, exists := service.Networks[networkName]; exists {
				if netConfig.IPv4Address != "" {
					serviceIPs[serviceName] = netConfig.IPv4Address
				}
			}
		}
	}

	return serviceIPs
}

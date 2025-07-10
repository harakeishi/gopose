package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/harakeishi/gopose/internal/generator"
	"github.com/harakeishi/gopose/internal/parser"
	"github.com/harakeishi/gopose/internal/resolver"
	"github.com/harakeishi/gopose/internal/scanner"
	"github.com/harakeishi/gopose/pkg/types"
	"github.com/spf13/cobra"
)

var (
	filePath           string
	portRange          string
	dryRun             bool
	strategy           string
	outputFile         string
	skipComposeUp      bool
	composeProjectName string
)

// parsePortRange はポート範囲文字列を解析します。
func parsePortRange(portRangeStr string) (types.PortRange, error) {
	if portRangeStr == "" {
		// デフォルトのポート範囲を返す
		return types.PortRange{Start: 8000, End: 9999}, nil
	}

	parts := strings.Split(portRangeStr, "-")
	if len(parts) != 2 {
		return types.PortRange{}, fmt.Errorf("無効なポート範囲形式です。正しい形式: start-end (例: 8000-9999)")
	}

	start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return types.PortRange{}, fmt.Errorf("開始ポートが無効です: %s", parts[0])
	}

	end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return types.PortRange{}, fmt.Errorf("終了ポートが無効です: %s", parts[1])
	}

	if start < 1 || start > 65535 {
		return types.PortRange{}, fmt.Errorf("開始ポートは1-65535の範囲で指定してください: %d", start)
	}

	if end < 1 || end > 65535 {
		return types.PortRange{}, fmt.Errorf("終了ポートは1-65535の範囲で指定してください: %d", end)
	}

	if start > end {
		return types.PortRange{}, fmt.Errorf("開始ポートが終了ポートより大きいです: %d > %d", start, end)
	}

	return types.PortRange{Start: start, End: end}, nil
}

// createPortConfig はCLIオプションからポート設定を作成します。
func createPortConfig(portRangeStr string) (types.PortConfig, error) {
	portRange, err := parsePortRange(portRangeStr)
	if err != nil {
		return types.PortConfig{}, err
	}

	return types.PortConfig{
		Range:             portRange,
		Reserved:          []int{}, // 予約済みポートは空で開始
		ExcludePrivileged: true,    // 特権ポートは除外
	}, nil
}

// detectWorktreeProjectName は現在の git ワークツリーのトップレベルディレクトリ名を
// 取得して返します。worktree環境では現在のディレクトリ名も含めて一意性を確保します。
func detectWorktreeProjectName() (string, error) {
	// 現在の作業ディレクトリを取得
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	
	// gitのトップレベルディレクトリを取得
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	topLevel := strings.TrimSpace(string(output))
	if topLevel == "" {
		return "", nil
	}

	topLevelBase := filepath.Base(topLevel)
	currentDirBase := filepath.Base(currentDir)
	
	// worktree環境の検出：現在のディレクトリがgitトップレベルと異なる場合
	if currentDir != topLevel {
		// worktree環境では "currentdir_topdir" の形式でプロジェクト名を生成
		return fmt.Sprintf("%s_%s", currentDirBase, topLevelBase), nil
	}
	
	return topLevelBase, nil
}

// runDockerCompose はdocker composeコマンドを実行します。
func runDockerCompose(ctx *cobra.Command, composeFile, outputFile string, extraArgs []string) error {
	args := []string{"compose"}

	// compose fileオプションを追加（デフォルトファイル名でない場合のみ）
	if composeFile != "" && composeFile != "docker-compose.yml" {
		args = append(args, "-f", composeFile)
	} else {
		// デフォルトファイルは明示的に指定
		args = append(args, "-f", "docker-compose.yml")
	}

	// override fileが存在する場合は追加
	if outputFile != "" {
		if _, err := os.Stat(outputFile); err == nil {
			args = append(args, "-f", outputFile)
		}
	}

	// プロジェクト名が指定されている場合
	if composeProjectName != "" {
		args = append(args, "-p", composeProjectName)
	}

	// upコマンドを追加
	args = append(args, "up")

	// override.ymlが存在する場合は強制再作成を追加（ユーザーが指定していない場合のみ）
	if outputFile != "" {
		if _, err := os.Stat(outputFile); err == nil {
			if forceRecreate, _ := ctx.Flags().GetBool("force-recreate"); !forceRecreate {
				args = append(args, "--force-recreate")
			}
			// ネットワークとボリュームも再作成
			if removeOrphans, _ := ctx.Flags().GetBool("remove-orphans"); !removeOrphans {
				args = append(args, "--remove-orphans")
			}
		}
	}

	// docker composeの共通オプションを処理
	if detach, _ := ctx.Flags().GetBool("detach"); detach {
		args = append(args, "-d")
	}

	if build, _ := ctx.Flags().GetBool("build"); build {
		args = append(args, "--build")
	}

	if forceRecreate, _ := ctx.Flags().GetBool("force-recreate"); forceRecreate {
		args = append(args, "--force-recreate")
	}

	if noDeps, _ := ctx.Flags().GetBool("no-deps"); noDeps {
		args = append(args, "--no-deps")
	}

	if removeOrphans, _ := ctx.Flags().GetBool("remove-orphans"); removeOrphans {
		args = append(args, "--remove-orphans")
	}

	if scale, _ := ctx.Flags().GetString("scale"); scale != "" {
		for _, scaleOption := range strings.Split(scale, ",") {
			args = append(args, "--scale", strings.TrimSpace(scaleOption))
		}
	}

	if envFiles, _ := ctx.Flags().GetStringSlice("env-file"); len(envFiles) > 0 {
		for _, envFile := range envFiles {
			args = append(args, "--env-file", envFile)
		}
	}

	if abortOnExit, _ := ctx.Flags().GetBool("abort-on-container-exit"); abortOnExit {
		args = append(args, "--abort-on-container-exit")
	}

	if exitCodeFrom, _ := ctx.Flags().GetString("exit-code-from"); exitCodeFrom != "" {
		args = append(args, "--exit-code-from", exitCodeFrom)
	}

	if timeout, _ := ctx.Flags().GetDuration("timeout"); timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%.0f", timeout.Seconds()))
	}

	// 追加の引数（サービス名など）を追加
	args = append(args, extraArgs...)

	// コマンドを実行
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	logger, _ := getLogger(getConfig())
	logger.Info(ctx.Context(), "Docker Composeを実行",
		types.Field{Key: "command", Value: fmt.Sprintf("docker %s", strings.Join(args, " "))})

	return cmd.Run()
}

// stopExistingContainers は既存のコンテナを停止・削除します。
func stopExistingContainers(ctx context.Context, composeFile string) error {
	args := []string{"compose"}

	// compose fileオプションを追加
	if composeFile != "" {
		args = append(args, "-f", composeFile)
	}

	// プロジェクト名が指定されている場合は追加
	if composeProjectName != "" {
		args = append(args, "-p", composeProjectName)
	}

	// downコマンドを追加（コンテナを停止・削除）
	args = append(args, "down")

	// コマンドを実行
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// detectNetworkSubnets collects all subnets configured in Compose file
func getComposeSubnets(config *types.ComposeConfig) map[string]string {
	result := make(map[string]string)
	for name, netCfg := range config.Networks {
		for _, ipCfg := range netCfg.IPAM.Config {
			if ipCfg.Subnet != "" {
				result[name] = ipCfg.Subnet
				break
			}
		}
	}
	return result
}

// getServiceNetworkIPs は指定されたネットワークを使用するサービスのIPアドレスを抽出します
func getServiceNetworkIPs(config *types.ComposeConfig, networkName string) map[string]string {
	result := make(map[string]string)
	for serviceName, service := range config.Services {
		if networkConfig, exists := service.Networks[networkName]; exists {
			if networkConfig.IPv4Address != "" {
				result[serviceName] = networkConfig.IPv4Address
			}
		}
	}
	return result
}

// allocateNewSubnet returns first available subnet from safe ranges, avoiding common conflicts
func allocateNewSubnet(used map[string]bool) string {
	// Priority order: 10.x.x.x/24 > 192.168.x.x/24 > 172.x.x.x/24
	
	// 1. Try 10.x.x.x/24 range (safe for most environments)
	for i := 20; i < 255; i++ { // Skip common ranges like 10.0.x.x, 10.1.x.x
		for j := 0; j < 255; j++ {
			candidate := fmt.Sprintf("10.%d.%d.0/24", i, j)
			if !used[candidate] {
				return candidate
			}
		}
	}
	
	// 2. Try 192.168.x.x/24 range (commonly used but safer than 172.x.x.x)
	for i := 100; i < 255; i++ { // Skip common home router ranges
		candidate := fmt.Sprintf("192.168.%d.0/24", i)
		if !used[candidate] {
			return candidate
		}
	}
	
	// 3. Try 172.x.x.x/24 range (last resort, more likely to conflict)
	for i := 30; i < 100; i++ { // Skip Docker's default range 172.17-29.x.x
		for j := 0; j < 255; j++ {
			candidate := fmt.Sprintf("172.%d.%d.0/24", i, j)
			if !used[candidate] {
				return candidate
			}
		}
	}
	
	return "" // No available subnet found
}

// remapIPAddressesToNewSubnet は既存のIPアドレスを新しいサブネットに再マップします
func remapIPAddressesToNewSubnet(oldSubnet, newSubnet string, serviceIPs map[string]string) (map[string]string, error) {
	// サブネットから基底アドレスを取得
	parts := strings.Split(oldSubnet, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("無効なサブネット形式: %s", oldSubnet)
	}
	oldBase := parts[0]
	
	newParts := strings.Split(newSubnet, "/")
	if len(newParts) != 2 {
		return nil, fmt.Errorf("無効なサブネット形式: %s", newSubnet)
	}
	newBase := newParts[0]
	
	// 既存の基底アドレスと新しい基底アドレスを取得
	oldBaseIP := strings.Split(oldBase, ".")
	newBaseIP := strings.Split(newBase, ".")
	
	if len(oldBaseIP) != 4 || len(newBaseIP) != 4 {
		return nil, fmt.Errorf("無効なIPアドレス形式")
	}
	
	newIPs := make(map[string]string)
	for service, oldIP := range serviceIPs {
		oldIPParts := strings.Split(oldIP, ".")
		if len(oldIPParts) != 4 {
			continue // 無効なIPはスキップ
		}
		
		// 新しいIPアドレスを生成（最後のオクテットのみ保持）
		newIP := fmt.Sprintf("%s.%s.%s.%s", newBaseIP[0], newBaseIP[1], newBaseIP[2], oldIPParts[3])
		newIPs[service] = newIP
	}
	
	return newIPs, nil
}

// upCmd はupコマンドを表します。
var upCmd = &cobra.Command{
	Use:   "up [docker-compose-options...]",
	Short: "ポート衝突・ネットワーク衝突を解決してDocker Composeを起動",
	Long: `Docker Composeのポートバインディング衝突とネットワークサブネット衝突を検出・解決し、docker-compose.override.yml を生成します。

ポート衝突・ネットワーク衝突の自動解決機能を提供し、override.ymlを生成しますが、docker compose upは実行しません。
必要に応じて手動でdocker compose upを実行してください。`,
	Example: `  # 基本的な使用方法
  gopose up

  # 特定のファイルを指定
  gopose up -f custom-compose.yml

  # ポート範囲を指定
  gopose up --port-range 9000-9999

  # Docker Composeオプションを渡す
  gopose up -d --build
  gopose up -- --scale web=3

  # ドライラン（override.ymlの生成のみ）
  gopose up --dry-run
  
  # ネットワーク衝突も含めて解決
  gopose up --verbose  # ネットワーク衝突の詳細ログを表示`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		cfg := getConfig()

		logger, err := getLogger(cfg)
		if err != nil {
			return fmt.Errorf("ロガーの初期化に失敗しました: %w", err)
		}

		// ポート範囲の解析
		portConfig, err := createPortConfig(portRange)
		if err != nil {
			return fmt.Errorf("ポート範囲の解析に失敗しました: %w", err)
		}

		// -p オプションが指定されていない場合は、ワークツリー名をプロジェクト名として自動設定
		if composeProjectName == "" && os.Getenv("COMPOSE_PROJECT_NAME") == "" {
			if pn, err := detectWorktreeProjectName(); err == nil && pn != "" {
				composeProjectName = pn
				logger.Info(ctx, "ワークツリー名をプロジェクト名として使用",
					types.Field{Key: "project_name", Value: composeProjectName})
			}
		}

		logger.Info(ctx, "ポート衝突解決を開始",
			types.Field{Key: "dry_run", Value: dryRun},
			types.Field{Key: "compose_file", Value: filePath},
			types.Field{Key: "output_file", Value: outputFile},
			types.Field{Key: "project_name", Value: composeProjectName},
			types.Field{Key: "strategy", Value: strategy},
			types.Field{Key: "port_range", Value: fmt.Sprintf("%d-%d", portConfig.Range.Start, portConfig.Range.End)})

		// Docker Composeファイルの自動検出（指定されていない場合）
		if filePath == "" || filePath == "docker-compose.yml" {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("作業ディレクトリの取得に失敗: %w", err)
			}

			detector := parser.NewComposeFileDetectorImpl(logger)
			detectedFile, err := detector.GetDefaultComposeFile(ctx, wd)
			if err != nil {
				return fmt.Errorf("Docker Composeファイルの自動検出に失敗: %w", err)
			}
			filePath = detectedFile
			logger.Info(ctx, "Docker Composeファイルを自動検出", types.Field{Key: "file", Value: filePath})
		}

		// Docker Composeファイルの解析
		yamlParser := parser.NewYamlComposeParser(logger)
		config, err := yamlParser.ParseComposeFile(ctx, filePath)
		if err != nil {
			return fmt.Errorf("Docker Composeファイルの解析に失敗: %w", err)
		}

		// ポートスキャンの実行
		portDetector := scanner.NewNetstatPortDetector(logger)
		portAllocator := scanner.NewPortAllocatorImpl(portDetector, logger)

		// ポート衝突の検出
		conflictDetector := resolver.NewConflictDetectorImpl(portDetector, logger)
		conflicts, err := conflictDetector.DetectPortConflicts(ctx, config)
		if err != nil {
			return fmt.Errorf("ポート衝突の検出に失敗: %w", err)
		}

		logger.Info(ctx, "ポート衝突検出完了", types.Field{Key: "conflicts_count", Value: len(conflicts)})

		// 衝突がない場合
		if len(conflicts) == 0 {
			logger.Info(ctx, "ポート衝突は検出されませんでした")
			if skipComposeUp {
				logger.Warn(ctx, "--skip-compose-upオプションは不要になりました。デフォルトでdocker compose upは実行されません。")
			}
			return nil
		}

		// 解決戦略の決定
		resolutionStrategy := types.ResolutionStrategyAutoIncrement
		switch strategy {
		case "auto":
			resolutionStrategy = types.ResolutionStrategyAutoIncrement
		case "range":
			resolutionStrategy = types.ResolutionStrategyRangeAllocation
		case "user":
			resolutionStrategy = types.ResolutionStrategyUserDefined
		}

		// ポート衝突の解決（PortConfigを渡す）
		conflictResolver := resolver.NewConflictResolverWithPortConfig(portAllocator, portConfig, logger)
		resolutions, err := conflictResolver.ResolvePortConflicts(ctx, conflicts, resolutionStrategy)
		if err != nil {
			return fmt.Errorf("ポート衝突の解決に失敗: %w", err)
		}

		// 解決案の最適化
		analyzer := resolver.NewPortResolutionAnalyzerImpl(logger)
		optimizedResolutions, err := analyzer.OptimizeResolutions(ctx, resolutions)
		if err != nil {
			return fmt.Errorf("解決案の最適化に失敗: %w", err)
		}

		// 解決結果の表示
		logger.Info(ctx, "ポート衝突解決完了",
			types.Field{Key: "resolved_conflicts", Value: len(optimizedResolutions)})

		for _, resolution := range optimizedResolutions {
			logger.Info(ctx, "ポート解決",
				types.Field{Key: "service", Value: resolution.ServiceName},
				types.Field{Key: "from", Value: resolution.ConflictPort},
				types.Field{Key: "to", Value: resolution.ResolvedPort},
				types.Field{Key: "reason", Value: resolution.Reason})
		}

		// Override.ymlの生成（ドライランモードでも競合検出のために実行）
		overrideGenerator := generator.NewOverrideGeneratorImpl(logger)
		override, err := overrideGenerator.GenerateOverride(ctx, config, optimizedResolutions)
		if err != nil {
			return fmt.Errorf("Overrideファイルの生成に失敗: %w", err)
		}

		// プロジェクト名をoverrideに設定（Docker Composeコマンドの統一のため）
		if composeProjectName != "" {
			override.Name = composeProjectName
			logger.Debug(ctx, "Override.ymlにプロジェクト名を設定", 
				types.Field{Key: "project_name", Value: composeProjectName})
		}

		// ---------------- Network conflict detection -----------------

		networkDetector := scanner.NewDockerNetworkDetector(logger)
		dockerNets, err := networkDetector.DetectNetworks(ctx)
		if err != nil {
			logger.Warn(ctx, "既存Dockerネットワークの検出に失敗しました。ネットワーク競合チェックをスキップします。", 
				types.Field{Key: "error", Value: err.Error()})
			dockerNets = []scanner.NetworkInfo{} // Continue with empty network list
		} else {
			logger.Info(ctx, "既存Dockerネットワークを検出しました", 
				types.Field{Key: "network_count", Value: len(dockerNets)})
		}

		usedSubnets := make(map[string]bool)
		usedNetworkNames := make(map[string]bool)
		for _, n := range dockerNets {
			usedNetworkNames[n.Name] = true
			for _, s := range n.Subnets {
				usedSubnets[s] = true
				logger.Debug(ctx, "既存ネットワークサブネットを記録", 
					types.Field{Key: "network", Value: n.Name},
					types.Field{Key: "subnet", Value: s})
			}
		}

		composeSubnets := getComposeSubnets(config)
		networkOverrides := make(map[string]types.NetworkOverride)
		
		if len(composeSubnets) > 0 {
			logger.Info(ctx, "Docker Composeネットワーク設定を検出", 
				types.Field{Key: "network_count", Value: len(composeSubnets)})
		}

		// プロジェクト名が設定されている場合、動的ネットワーク名も考慮
		projectPrefix := ""
		if composeProjectName != "" {
			projectPrefix = composeProjectName + "_"
		}

		for netName, subnet := range composeSubnets {
			// プロジェクト名を含む実際のネットワーク名を生成
			actualNetworkName := projectPrefix + netName
			
			logger.Debug(ctx, "ネットワーク競合をチェック", 
				types.Field{Key: "network", Value: netName},
				types.Field{Key: "actual_network_name", Value: actualNetworkName},
				types.Field{Key: "subnet", Value: subnet})
			
			// ネットワーク名の衝突をチェック
			if usedNetworkNames[actualNetworkName] {
				logger.Warn(ctx, "ネットワーク名競合を検出", 
					types.Field{Key: "network", Value: netName},
					types.Field{Key: "conflicting_network_name", Value: actualNetworkName})
			}
			
			// サブネット衝突の従来のチェック
			needsNewSubnet := false
			conflictReason := ""
			
			if subnet == "" {
				logger.Debug(ctx, "ネットワークにサブネット設定がありません", 
					types.Field{Key: "network", Value: netName})
				continue
			}
			
			if usedSubnets[subnet] {
				needsNewSubnet = true
				conflictReason = "サブネット競合"
			} else if usedNetworkNames[actualNetworkName] {
				// ネットワーク名が既に存在し、そのネットワークが異なるサブネットを使用している場合
				needsNewSubnet = true
				conflictReason = "ネットワーク名競合"
			}
			
			if needsNewSubnet {
				logger.Warn(ctx, "ネットワーク競合を検出", 
					types.Field{Key: "network", Value: netName},
					types.Field{Key: "reason", Value: conflictReason},
					types.Field{Key: "conflicting_subnet", Value: subnet})
				
				newSubnet := allocateNewSubnet(usedSubnets)
				if newSubnet == "" {
					logger.Warn(ctx, "利用可能なサブネットが見つかりません。すべての安全な範囲が使用済みです。", 
						types.Field{Key: "network", Value: netName})
					continue
				}
				usedSubnets[newSubnet] = true

				networkOverrides[netName] = types.NetworkOverride{
					IPAM: types.IPAM{
						Config: []types.IPAMConfig{{Subnet: newSubnet}},
					},
				}

				// サービスのIPアドレスも新しいサブネットに再割り当て
				serviceIPs := getServiceNetworkIPs(config, netName)
				logger.Debug(ctx, "サービスIP情報を取得", 
					types.Field{Key: "network", Value: netName},
					types.Field{Key: "service_count", Value: len(serviceIPs)},
					types.Field{Key: "service_ips", Value: fmt.Sprintf("%+v", serviceIPs)})
				if len(serviceIPs) > 0 {
					newServiceIPs, err := remapIPAddressesToNewSubnet(subnet, newSubnet, serviceIPs)
					if err != nil {
						logger.Warn(ctx, "サービスIPアドレスの再マッピングに失敗", 
							types.Field{Key: "network", Value: netName},
							types.Field{Key: "error", Value: err.Error()})
					} else {
						// Override.ymlにサービスのネットワーク設定を追加
						for serviceName, newIP := range newServiceIPs {
							// 既存のサービスオーバーライドを取得または作成
							if override.Services == nil {
								override.Services = make(map[string]types.ServiceOverride)
							}
							serviceOverride, exists := override.Services[serviceName]
							if !exists {
								// 新しいサービスオーバーライドを作成（ポート設定は保持されない）
								serviceOverride = types.ServiceOverride{}
							}
							if serviceOverride.Networks == nil {
								serviceOverride.Networks = make(map[string]types.ServiceNetwork)
							}
							serviceOverride.Networks[netName] = types.ServiceNetwork{
								IPv4Address: newIP,
							}
							override.Services[serviceName] = serviceOverride
							
							logger.Debug(ctx, "Override.Servicesに追加されたサービス", 
								types.Field{Key: "service", Value: serviceName},
								types.Field{Key: "override_services_count", Value: len(override.Services)},
								types.Field{Key: "service_override", Value: fmt.Sprintf("%+v", serviceOverride)})
							
							logger.Info(ctx, "サービスIPアドレスを再割り当て",
								types.Field{Key: "service", Value: serviceName},
								types.Field{Key: "network", Value: netName},
								types.Field{Key: "old_ip", Value: serviceIPs[serviceName]},
								types.Field{Key: "new_ip", Value: newIP})
						}
					}
				}

				logger.Info(ctx, "ネットワーク競合を解決",
					types.Field{Key: "network", Value: netName},
					types.Field{Key: "reason", Value: conflictReason},
					types.Field{Key: "original_subnet", Value: subnet},
					types.Field{Key: "new_subnet", Value: newSubnet})
			} else {
				logger.Debug(ctx, "ネットワーク競合なし", 
					types.Field{Key: "network", Value: netName},
					types.Field{Key: "subnet", Value: subnet})
			}
		}

		// Override.ymlの妥当性検証
		if err := overrideGenerator.ValidateOverride(ctx, override); err != nil {
			return fmt.Errorf("Overrideファイルの検証に失敗: %w", err)
		}

		// 出力ファイル名の決定
		if outputFile == "" {
			outputFile = "docker-compose.override.yml"
		}

		// ネットワークオーバーライドを追加
		if len(networkOverrides) > 0 {
			if override.Networks == nil {
				override.Networks = make(map[string]types.NetworkOverride)
			}
			for k, v := range networkOverrides {
				override.Networks[k] = v
			}
		}

		// ドライランモードでない場合のみファイル書き込み
		if !dryRun {
			// Override.ymlファイルの書き込み
			if err := overrideGenerator.WriteOverrideFile(ctx, override, outputFile); err != nil {
				return fmt.Errorf("Overrideファイルの書き込みに失敗: %w", err)
			}

			logger.Info(ctx, "Override.ymlファイルが生成されました",
				types.Field{Key: "output_file", Value: outputFile})
		} else {
			logger.Info(ctx, "ドライランモードのため、ファイルは生成されません")
		}

		// Docker Composeの実行（--skip-compose-upが指定された場合は後方互換のための警告を表示）
		if skipComposeUp {
			logger.Warn(ctx, "--skip-compose-upオプションは不要になりました。デフォルトでdocker compose upは実行されません。")
		}

		// デフォルトではDocker Composeを実行しない
		if !dryRun {
			logger.Info(ctx, "override.ymlの生成が完了しました。docker compose upを実行する場合は、手動で実行してください。")
		}

		return nil
	},
}

func init() {
	// gopose固有のフラグを定義
	upCmd.Flags().StringVar(&portRange, "port-range", "", "利用するポート範囲 (例: 8000-9999)")
	upCmd.Flags().StringVar(&strategy, "strategy", "auto", "解決戦略 (auto, range, user)")
	upCmd.Flags().StringVarP(&outputFile, "output", "o", "", "出力ファイル名 (デフォルト: docker-compose.override.yml)")
	upCmd.Flags().BoolVar(&dryRun, "dry-run", false, "ドライラン（override.yml生成のみ、Docker Composeは実行しない）")
	upCmd.Flags().BoolVar(&skipComposeUp, "skip-compose-up", false, "[非推奨] このオプションは不要になりました。デフォルトでdocker compose upは実行されません。")

	// Docker Composeオプションもサポート（透過的に渡される）
	upCmd.Flags().StringVarP(&filePath, "file", "f", "docker-compose.yml", "Docker Composeファイルのパス")
	upCmd.Flags().StringVarP(&composeProjectName, "project-name", "p", "", "Docker Composeプロジェクト名")
	upCmd.Flags().BoolP("detach", "d", false, "Detached mode: バックグラウンドでサービスを実行")
	upCmd.Flags().Bool("build", false, "サービス起動前にイメージをビルド")
	upCmd.Flags().Bool("force-recreate", false, "設定が変更されていなくてもコンテナを再作成")
	upCmd.Flags().Bool("no-deps", false, "リンクされたサービスを起動しない")
	upCmd.Flags().Bool("remove-orphans", false, "Composeファイルで定義されていないサービスのコンテナを削除")
	upCmd.Flags().String("scale", "", "サービスの起動数を指定 (例: web=3,db=1)")
	upCmd.Flags().StringSlice("env-file", []string{}, "環境変数ファイルを指定")
	upCmd.Flags().Bool("abort-on-container-exit", false, "いずれかのコンテナが停止したときに全てのコンテナを停止")
	upCmd.Flags().String("exit-code-from", "", "指定されたサービスの終了コードを返す")
	upCmd.Flags().Duration("timeout", 0, "コンテナの停止タイムアウト")

	// 未知のフラグを許可（docker composeに渡すため）
	upCmd.Flags().ParseErrorsWhitelist.UnknownFlags = true
}

package cmd

import (
	"fmt"
	"os"
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
	filePath   string
	portRange  string
	dryRun     bool
	strategy   string
	outputFile string
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

// upCmd はupコマンドを表します。
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "ポート衝突を検出・解決してDocker Composeを準備",
	Long: `Docker Composeのポートバインディング衝突を検出し、自動的に解決します。

元の docker-compose.yml ファイルを変更せずに、docker-compose.override.yml を生成して
ポート衝突を解決します。`,
	Example: `  # 基本的な使用方法
  gopose up

  # 特定のファイルを指定
  gopose up -f custom-compose.yml

  # ポート範囲を指定
  gopose up --port-range 9000-9999

  # 解決戦略を指定
  gopose up --strategy range

  # ドライラン（実際の変更は行わない）
  gopose up --dry-run`,
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

		logger.Info(ctx, "ポート衝突解決を開始",
			types.Field{Key: "dry_run", Value: dryRun},
			types.Field{Key: "compose_file", Value: filePath},
			types.Field{Key: "output_file", Value: outputFile},
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

		if len(conflicts) == 0 {
			logger.Info(ctx, "ポート衝突は検出されませんでした")
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

		// ドライランモードの場合はここで終了
		if dryRun {
			logger.Info(ctx, "ドライランモードのため、ファイルは生成されません")
			return nil
		}

		// Override.ymlの生成
		overrideGenerator := generator.NewOverrideGeneratorImpl(logger)
		override, err := overrideGenerator.GenerateOverride(ctx, config, optimizedResolutions)
		if err != nil {
			return fmt.Errorf("Overrideファイルの生成に失敗: %w", err)
		}

		// Override.ymlの妥当性検証
		if err := overrideGenerator.ValidateOverride(ctx, override); err != nil {
			return fmt.Errorf("Overrideファイルの検証に失敗: %w", err)
		}

		// 出力ファイル名の決定
		if outputFile == "" {
			outputFile = "docker-compose.override.yml"
		}

		// Override.ymlファイルの書き込み
		if err := overrideGenerator.WriteOverrideFile(ctx, override, outputFile); err != nil {
			return fmt.Errorf("Overrideファイルの書き込みに失敗: %w", err)
		}

		logger.Info(ctx, "Override.ymlファイルが生成されました",
			types.Field{Key: "output_file", Value: outputFile})

		return nil
	},
}

func init() {
	// upコマンド固有のフラグを定義
	upCmd.Flags().StringVarP(&filePath, "file", "f", "docker-compose.yml", "Docker Composeファイルのパス")
	upCmd.Flags().StringVar(&portRange, "port-range", "", "利用するポート範囲 (例: 8000-9999)")
	upCmd.Flags().StringVar(&strategy, "strategy", "auto", "解決戦略 (auto, range, user)")
	upCmd.Flags().StringVarP(&outputFile, "output", "o", "", "出力ファイル名 (デフォルト: docker-compose.override.yml)")
	upCmd.Flags().BoolVar(&dryRun, "dry-run", false, "ドライラン（実際の変更は行わない）")
}

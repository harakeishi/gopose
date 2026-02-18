package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/harakeishi/gopose/internal/generator"
	"github.com/harakeishi/gopose/internal/parser"
	"github.com/harakeishi/gopose/internal/scanner"
	"github.com/harakeishi/gopose/pkg/types"
	"github.com/spf13/cobra"
)

var (
	filePath           string
	portRange          string
	dryRun             bool
	outputFile         string
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

// createPortConfig はCLIオプションと設定ファイルからポート設定を作成します。
// baseConfigをベースとして、portRangeStrが指定されている場合のみ範囲を上書きします。
func createPortConfig(portRangeStr string, baseConfig types.PortConfig) (types.PortConfig, error) {
	// CLIでポート範囲が指定されている場合のみ上書き
	if portRangeStr != "" {
		portRange, err := parsePortRange(portRangeStr)
		if err != nil {
			return types.PortConfig{}, err
		}
		baseConfig.Range = portRange
	}

	return baseConfig, nil
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

// upCmd はupコマンドを表します。
var upCmd = &cobra.Command{
	Use:   "up",
	Short: "ポート衝突・ネットワーク衝突を解決してoverride.ymlを生成",
	Long: `Docker Composeのポートバインディング衝突とネットワークサブネット衝突を検出・解決し、docker-compose.override.yml を生成します。

ポート衝突・ネットワーク衝突の自動解決機能を提供し、override.ymlを生成しますが、docker compose upは実行しません。
必要に応じて手動でdocker compose upを実行してください。`,
	Example: `  # 基本的な使用方法
  gopose up

  # 特定のファイルを指定
  gopose up -f custom-compose.yml

  # ポート範囲を指定
  gopose up --port-range 9000-9999

  # ドライラン（override.ymlの生成のみ）
  gopose up --dry-run

  # ネットワーク衝突も含めて解決
  gopose up --verbose  # ネットワーク衝突の詳細ログを表示`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		cfg := getConfig()
		pres := getPresenter()

		logger, err := getLogger(cfg)
		if err != nil {
			return fmt.Errorf("ロガーの初期化に失敗しました: %w", err)
		}

		portConfig, err := createPortConfig(portRange, cfg.GetPort())
		if err != nil {
			return fmt.Errorf("ポート範囲の解析に失敗しました: %w", err)
		}

		if composeProjectName == "" && os.Getenv("COMPOSE_PROJECT_NAME") == "" {
			if pn, err := detectWorktreeProjectName(); err == nil && pn != "" {
				composeProjectName = pn
				logger.Debug(ctx, "ワークツリー名をプロジェクト名として使用",
					types.Field{Key: "project_name", Value: composeProjectName})
			}
		}

		logger.Debug(ctx, "ポート衝突解決を開始",
			types.Field{Key: "dry_run", Value: dryRun},
			types.Field{Key: "compose_file", Value: filePath},
			types.Field{Key: "output_file", Value: outputFile},
			types.Field{Key: "project_name", Value: composeProjectName},
			types.Field{Key: "port_range", Value: fmt.Sprintf("%d-%d", portConfig.Range.Start, portConfig.Range.End)},
			types.Field{Key: "reserved_ports", Value: portConfig.Reserved})

		// Docker Composeファイルの自動検出（指定されていない場合）
		if filePath == "" || filePath == "compose.yml" {
			wd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("作業ディレクトリの取得に失敗: %w", err)
			}

			detector := parser.NewComposeFileDetectorImpl(logger)
			detectedFile, err := detector.GetDefaultComposeFile(ctx, wd)
			if err != nil {
				return fmt.Errorf("docker composeファイルの自動検出に失敗: %w", err)
			}
			filePath = detectedFile
			logger.Debug(ctx, "Docker Composeファイルを自動検出", types.Field{Key: "file", Value: filePath})
		}

		// Docker Composeファイルの解析
		yamlParser := parser.NewYamlComposeParser(logger)
		config, err := yamlParser.ParseComposeFile(ctx, filePath)
		if err != nil {
			return fmt.Errorf("docker composeファイルの解析に失敗: %w", err)
		}

		pres.Progress("Scanning...")

		// 統一的な衝突検知の実行
		portDetector := scanner.NewNetstatPortDetector(logger)
		portAllocator := scanner.NewPortAllocatorImpl(portDetector, logger)
		networkDetector := scanner.NewDockerNetworkDetector(logger)
		unifiedDetector := scanner.NewUnifiedConflictDetectorImpl(portDetector, networkDetector, logger)

		conflictInfo, err := unifiedDetector.DetectConflicts(ctx, config, composeProjectName)
		if err != nil {
			return fmt.Errorf("衝突検知に失敗: %w", err)
		}

		// 衝突がない場合
		if !conflictInfo.HasConflicts() {
			pres.PortConflicts(nil)
			pres.NetworkConflicts(nil)
			pres.Result("No conflicts detected.")
			return nil
		}

		pres.Progress("Resolving...")

		logger.Debug(ctx, "衝突検知完了",
			types.Field{Key: "port_conflicts", Value: len(conflictInfo.PortConflicts)},
			types.Field{Key: "network_conflicts", Value: len(conflictInfo.NetworkConflicts)})

		// 統一的な衝突解決
		unifiedGenerator := generator.NewUnifiedOverrideGeneratorImpl(portAllocator, logger)
		if err := unifiedGenerator.ResolveConflicts(ctx, conflictInfo, types.ResolutionStrategyAutoIncrement, portConfig); err != nil {
			return fmt.Errorf("衝突解決に失敗: %w", err)
		}

		// 衝突回避結果のテーブル表示
		pres.PortConflicts(conflictInfo.PortConflicts)
		pres.NetworkConflicts(conflictInfo.NetworkConflicts)

		// 統一的なOverride.ymlの生成
		override, err := unifiedGenerator.GenerateFromConflicts(ctx, config, conflictInfo)
		if err != nil {
			return fmt.Errorf("overrideファイルの生成に失敗: %w", err)
		}

		// プロジェクト名をoverrideに設定
		if composeProjectName != "" {
			override.Name = composeProjectName
			logger.Debug(ctx, "Override.ymlにプロジェクト名を設定",
				types.Field{Key: "project_name", Value: composeProjectName})
		}

		// Override.ymlの妥当性検証
		overrideGenerator := generator.NewOverrideGeneratorImpl(logger)
		if err := overrideGenerator.ValidateOverride(ctx, override); err != nil {
			return fmt.Errorf("overrideファイルの検証に失敗: %w", err)
		}

		// 出力ファイル名の決定
		if outputFile == "" {
			outputFile = "compose.override.yml"
		}

		// ドライランモードの場合
		if dryRun {
			pres.Result("Dry run: no files written.")
			return nil
		}

		// Override.ymlファイルの書き込み
		if err := overrideGenerator.WriteOverrideFile(ctx, override, outputFile); err != nil {
			return fmt.Errorf("overrideファイルの書き込みに失敗: %w", err)
		}

		pres.Result("Generated: " + outputFile)
		return nil
	},
}

func init() {
	// gopose固有のフラグを定義
	upCmd.Flags().StringVar(&portRange, "port-range", "", "利用するポート範囲 (例: 8000-9999)")
	upCmd.Flags().StringVarP(&outputFile, "output", "o", "", "出力ファイル名 (デフォルト: compose.override.yml)")
	upCmd.Flags().BoolVar(&dryRun, "dry-run", false, "ドライラン（override.yml生成のみ、Docker Composeは実行しない）")

	upCmd.Flags().StringVarP(&filePath, "file", "f", "compose.yml", "Docker Composeファイルのパス")
	upCmd.Flags().StringVarP(&composeProjectName, "project-name", "p", "", "Docker Composeプロジェクト名")
}

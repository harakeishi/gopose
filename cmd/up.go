package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	filePath  string
	portRange string
	dryRun    bool
)

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

  # ドライラン（実際の変更は行わない）
  gopose up --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		cfg := getConfig()

		logger, err := getLogger(cfg)
		if err != nil {
			return fmt.Errorf("ロガーの初期化に失敗しました: %w", err)
		}

		logger.Info(ctx, "gopose up コマンドを開始しています")

		// TODO: 実際の実装をここに追加
		fmt.Println("ポート衝突の検出と解決を実行中...")
		fmt.Println("現在は実装中です。")

		return nil
	},
}

func init() {
	// upコマンド固有のフラグを定義
	upCmd.Flags().StringVarP(&filePath, "file", "f", "docker-compose.yml", "Docker Composeファイルのパス")
	upCmd.Flags().StringVar(&portRange, "port-range", "", "利用するポート範囲 (例: 8000-9999)")
	upCmd.Flags().BoolVar(&dryRun, "dry-run", false, "ドライラン（実際の変更は行わない）")
}

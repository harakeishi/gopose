package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	forceClean bool
	allFiles   bool
)

// cleanCmd はcleanコマンドを表します。
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "生成されたoverride.ymlファイルを削除",
	Long: `gopose により生成された docker-compose.override.yml ファイルを削除します。

バックアップファイルが存在する場合、それらも削除対象になります。`,
	Example: `  # 基本的なクリーンアップ
  gopose clean

  # 確認なしで強制削除
  gopose clean --force

  # すべての関連ファイルを削除
  gopose clean --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		cfg := getConfig()

		logger, err := getLogger(cfg)
		if err != nil {
			return fmt.Errorf("ロガーの初期化に失敗しました: %w", err)
		}

		logger.Info(ctx, "gopose clean コマンドを開始しています")

		// TODO: 実際の実装をここに追加
		fmt.Println("生成されたファイルのクリーンアップを実行中...")
		fmt.Println("現在は実装中です。")

		return nil
	},
}

func init() {
	// cleanコマンド固有のフラグを定義
	cleanCmd.Flags().BoolVar(&forceClean, "force", false, "確認なしで強制削除")
	cleanCmd.Flags().BoolVar(&allFiles, "all", false, "すべての関連ファイルを削除")
}

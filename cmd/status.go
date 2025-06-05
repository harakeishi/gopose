package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	outputFormat string
	detailed     bool
)

// statusCmd はstatusコマンドを表します。
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "現在の状態確認",
	Long: `Docker Composeプロジェクトの現在の状態、ポート使用状況、
および gopose による変更の状況を確認します。`,
	Example: `  # 基本的な状態確認
  gopose status

  # 詳細情報を表示
  gopose status --detailed

  # JSON形式で出力
  gopose status --output json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		cfg := getConfig()

		logger, err := getLogger(cfg)
		if err != nil {
			return fmt.Errorf("ロガーの初期化に失敗しました: %w", err)
		}

		logger.Info(ctx, "gopose status コマンドを開始しています")

		// TODO: 実際の実装をここに追加
		fmt.Println("現在の状態を確認中...")
		fmt.Println("現在は実装中です。")

		return nil
	},
}

func init() {
	// statusコマンド固有のフラグを定義
	statusCmd.Flags().StringVarP(&outputFormat, "output", "o", "text", "出力形式 (text, json, yaml)")
	statusCmd.Flags().BoolVar(&detailed, "detailed", false, "詳細情報を表示")
}

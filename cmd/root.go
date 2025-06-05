// Package cmd は、gopose のコマンドライン機能を提供します。
package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/harakeishi/gopose/internal/config"
	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

var (
	cfgFile string
	verbose bool
)

// rootCmd はルートコマンドを表します。
var rootCmd = &cobra.Command{
	Use:   "gopose",
	Short: "Docker Compose ポート衝突自動解決ツール",
	Long: `gopose は Docker Compose のポートバインディング衝突を自動検出・解決するツールです。

元の docker-compose.yml を変更せずに docker-compose.override.yml を生成し、
ポート衝突解決後、自動的に override.yml を削除します。`,
	Example: `  # ポート衝突を検出・解決してDocker Composeを準備
  gopose up

  # 生成されたoverride.ymlファイルを削除
  gopose clean

  # 現在の状態確認
  gopose status

  # 特定のファイルを指定
  gopose up -f custom-compose.yml

  # ポート範囲を指定
  gopose up --port-range 9000-9999`,
}

// Execute はコマンドを実行します。
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
}

func init() {
	cobra.OnInitialize(initConfig)

	// グローバルフラグの定義
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "設定ファイルのパス (デフォルト: $HOME/.gopose.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "詳細ログ出力")

	// 各サブコマンドをルートコマンドに追加
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(statusCmd)
}

// initConfig は設定を初期化します。
func initConfig() {
	if cfgFile != "" {
		// 設定ファイルが指定された場合
		viper.SetConfigFile(cfgFile)
	} else {
		// ホームディレクトリを取得
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// ホームディレクトリに .gopose.yaml を探す
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".gopose")
	}

	// 環境変数の自動バインド
	viper.AutomaticEnv()

	// 設定ファイルを読み込み
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintln(os.Stderr, "設定ファイルを使用中:", viper.ConfigFileUsed())
	}
}

// getConfig は設定を取得します。
func getConfig() types.Config {
	// デフォルト設定から開始
	cfg := config.DefaultConfig()

	// Viperからの設定をマージ
	if err := viper.Unmarshal(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "設定の読み込みに失敗しました: %v\n", err)
		return cfg
	}

	// verboseフラグが設定されている場合
	if verbose {
		cfg.Log.Level = "debug"
	}

	return cfg
}

// getLogger はロガーを取得します。
func getLogger(cfg types.Config) (logger.Logger, error) {
	factory := logger.NewStructuredLoggerFactory()
	return factory.Create(cfg.GetLog())
}

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	psWithComposeFiles []string
)

// psCmd はpsコマンドを表します。
var psCmd = &cobra.Command{
	Use:   "ps",
	Short: "Docker Composeサービスの状態を表示",
	Long:  `実行中のDocker Composeサービスの状態を表示します。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// docker compose psコマンドの構築
		dockerArgs := []string{"compose"}

		// メインのComposeファイル
		if filePath != "" && filePath != "docker-compose.yml" {
			dockerArgs = append(dockerArgs, "-f", filePath)
		}

		// --withオプションで指定されたファイルを追加
		for _, withFile := range psWithComposeFiles {
			absWithFile, err := filepath.Abs(withFile)
			if err != nil {
				return fmt.Errorf("依存ファイル %s のパス解決に失敗: %w", withFile, err)
			}

			// ファイルの存在確認
			if _, err := os.Stat(absWithFile); err != nil {
				return fmt.Errorf("依存ファイル %s が見つかりません: %w", withFile, err)
			}

			dockerArgs = append(dockerArgs, "-f", absWithFile)
		}

		// プロジェクト名が指定されている場合
		if composeProjectName != "" {
			dockerArgs = append(dockerArgs, "-p", composeProjectName)
		}

		// psコマンドを追加
		dockerArgs = append(dockerArgs, "ps")

		// 追加のフラグを処理
		if all, _ := cmd.Flags().GetBool("all"); all {
			dockerArgs = append(dockerArgs, "--all")
		}

		if quiet, _ := cmd.Flags().GetBool("quiet"); quiet {
			dockerArgs = append(dockerArgs, "--quiet")
		}

		if services, _ := cmd.Flags().GetBool("services"); services {
			dockerArgs = append(dockerArgs, "--services")
		}

		if format, _ := cmd.Flags().GetString("format"); format != "" {
			dockerArgs = append(dockerArgs, "--format", format)
		}

		// 残りの引数（サービス名など）を追加
		dockerArgs = append(dockerArgs, args...)

		// docker compose psを実行
		execCmd := exec.Command("docker", dockerArgs...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin

		return execCmd.Run()
	},
}

func init() {
	// gopose固有のフラグ
	psCmd.Flags().StringSliceVar(&psWithComposeFiles, "with", []string{}, "依存するDocker Composeファイル (複数指定可能)")
	psCmd.Flags().StringVarP(&filePath, "file", "f", "docker-compose.yml", "Docker Composeファイルのパス")
	psCmd.Flags().StringVarP(&composeProjectName, "project-name", "p", "", "Docker Composeプロジェクト名")

	// Docker Compose psのオプション
	psCmd.Flags().BoolP("all", "a", false, "停止中のコンテナも表示")
	psCmd.Flags().BoolP("quiet", "q", false, "コンテナIDのみ表示")
	psCmd.Flags().Bool("services", false, "サービス名のみ表示")
	psCmd.Flags().String("format", "", "出力フォーマットを指定")
}

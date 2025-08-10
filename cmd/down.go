package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	downWithComposeFiles []string
)

// downCmd はdownコマンドを表します。
var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Docker Composeサービスを停止・削除",
	Long:  `Docker Composeサービスを停止し、関連するコンテナ、ネットワーク、ボリュームを削除します。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// docker compose downコマンドの構築
		dockerArgs := []string{"compose"}
		
		// メインのComposeファイル
		if filePath != "" && filePath != "docker-compose.yml" {
			dockerArgs = append(dockerArgs, "-f", filePath)
		}
		
		// --withオプションで指定されたファイルを追加
		for _, withFile := range downWithComposeFiles {
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
		
		// downコマンドを追加
		dockerArgs = append(dockerArgs, "down")
		
		// 追加のフラグを処理
		if removeOrphans, _ := cmd.Flags().GetBool("remove-orphans"); removeOrphans {
			dockerArgs = append(dockerArgs, "--remove-orphans")
		}
		
		if volumes, _ := cmd.Flags().GetBool("volumes"); volumes {
			dockerArgs = append(dockerArgs, "--volumes")
		}
		
		if rmi, _ := cmd.Flags().GetString("rmi"); rmi != "" {
			dockerArgs = append(dockerArgs, "--rmi", rmi)
		}
		
		// docker compose downを実行
		execCmd := exec.Command("docker", dockerArgs...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin
		
		return execCmd.Run()
	},
}

func init() {
	// gopose固有のフラグ
	downCmd.Flags().StringSliceVar(&downWithComposeFiles, "with", []string{}, "依存するDocker Composeファイル (複数指定可能)")
	downCmd.Flags().StringVarP(&filePath, "file", "f", "docker-compose.yml", "Docker Composeファイルのパス")
	downCmd.Flags().StringVarP(&composeProjectName, "project-name", "p", "", "Docker Composeプロジェクト名")
	
	// Docker Compose downのオプション
	downCmd.Flags().Bool("remove-orphans", false, "Composeファイルで定義されていないサービスのコンテナを削除")
	downCmd.Flags().Bool("volumes", false, "ボリュームも削除")
	downCmd.Flags().String("rmi", "", "イメージも削除 (all または local)")
}
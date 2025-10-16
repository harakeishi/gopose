// Package cmd は、gopose のコマンドライン機能を提供します。
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "バージョン情報を表示",
	Long:  "gopose のバージョン情報を表示します。",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gopose version %s\n", appVersion)
	},
}

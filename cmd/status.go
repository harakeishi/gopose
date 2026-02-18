package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/internal/parser"
	"github.com/harakeishi/gopose/pkg/types"
	"github.com/spf13/cobra"
)

var (
	statusOutputFormat string
	statusFilePath     string
)

// PortInfo はポート情報を表します。
type PortInfo struct {
	Service       string `json:"service"`
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
	HostIP        string `json:"host_ip,omitempty"`
	Overridden    bool   `json:"overridden"`
	OriginalPort  int    `json:"original_port,omitempty"`
}

// statusCmd はstatusコマンドを表します。
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "現在の状態確認",
	Long: `Docker Composeプロジェクトの現在の状態、ポート使用状況、
および gopose による変更の状況を確認します。`,
	Example: `  # 基本的な状態確認
  gopose status

  # JSON形式で出力
  gopose status --output json`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().StringVarP(&statusFilePath, "file", "f", "", "Docker Composeファイルのパス")
	statusCmd.Flags().StringVarP(&statusOutputFormat, "output", "o", "table", "出力形式 (table, json)")
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	cfg := getConfig()

	logger, err := getLogger(cfg)
	if err != nil {
		return fmt.Errorf("ロガーの初期化に失敗しました: %w", err)
	}

	// Docker Composeファイルの自動検出
	composeFilePath := statusFilePath
	if composeFilePath == "" {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("作業ディレクトリの取得に失敗: %w", err)
		}

		detector := parser.NewComposeFileDetectorImpl(logger)
		detectedFile, err := detector.GetDefaultComposeFile(ctx, wd)
		if err != nil {
			return fmt.Errorf("docker composeファイルの自動検出に失敗: %w", err)
		}
		composeFilePath = detectedFile
	}

	// Docker Composeファイルの解析
	yamlParser := parser.NewYamlComposeParser(logger)
	config, err := yamlParser.ParseComposeFile(ctx, composeFilePath)
	if err != nil {
		return fmt.Errorf("docker composeファイルの解析に失敗: %w", err)
	}

	// override.ymlの読み込み（存在する場合）
	overrideConfig, err := loadOverrideConfig(ctx, composeFilePath, logger)
	if err != nil {
		logger.Debug(ctx, "override.ymlの読み込みをスキップ", types.Field{Key: "reason", Value: err.Error()})
	}

	// ポート情報の収集
	portInfos := collectPortInfos(config, overrideConfig)

	// 出力
	switch statusOutputFormat {
	case "json":
		return outputJSON(os.Stdout, portInfos)
	default:
		return outputTable(os.Stdout, portInfos)
	}
}

// loadOverrideConfig はoverride.ymlを読み込みます。
// Docker Composeのoverride.ymlはComposeConfigと同じ形式なので、
// YamlComposeParserを使用して解析し、OverrideConfigに変換します。
func loadOverrideConfig(ctx context.Context, composeFilePath string, logger logger.Logger) (*types.OverrideConfig, error) {
	dir := filepath.Dir(composeFilePath)

	// 複数のoverride.ymlファイル名をチェック
	overrideFiles := []string{
		"compose.override.yml",
		"compose.override.yaml",
		"docker-compose.override.yml",
		"docker-compose.override.yaml",
	}

	for _, overrideFile := range overrideFiles {
		overridePath := filepath.Join(dir, overrideFile)
		if _, err := os.Stat(overridePath); err == nil {
			// YamlComposeParserを使用して解析
			yamlParser := parser.NewYamlComposeParser(logger)
			composeConfig, err := yamlParser.ParseComposeFile(ctx, overridePath)
			if err != nil {
				return nil, fmt.Errorf("override.ymlの解析に失敗: %w", err)
			}

			// ComposeConfigをOverrideConfigに変換
			override := &types.OverrideConfig{
				Services: make(map[string]types.ServiceOverride),
			}

			for serviceName, service := range composeConfig.Services {
				override.Services[serviceName] = types.ServiceOverride{
					Ports: service.Ports,
				}
			}

			return override, nil
		}
	}

	return nil, fmt.Errorf("override.ymlが見つかりません")
}

// collectPortInfos はすべてのサービスのポート情報を収集します。
func collectPortInfos(config *types.ComposeConfig, override *types.OverrideConfig) []PortInfo {
	var portInfos []PortInfo

	// サービス名でソート
	serviceNames := make([]string, 0, len(config.Services))
	for name := range config.Services {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)

	for _, serviceName := range serviceNames {
		service := config.Services[serviceName]

		for _, port := range service.Ports {
			info := PortInfo{
				Service:       serviceName,
				HostPort:      port.Host,
				ContainerPort: port.Container,
				Protocol:      port.Protocol,
				HostIP:        port.HostIP,
				Overridden:    false,
			}

			// オーバーライドをチェック
			if override != nil {
				if serviceOverride, exists := override.Services[serviceName]; exists {
					for _, overridePort := range serviceOverride.Ports {
						if overridePort.Container == port.Container {
							if overridePort.Host != port.Host {
								info.OriginalPort = port.Host
								info.HostPort = overridePort.Host
								info.Overridden = true
							}
							break
						}
					}
				}
			}

			portInfos = append(portInfos, info)
		}
	}

	return portInfos
}

// outputTable はテーブル形式で出力します。
func outputTable(out io.Writer, portInfos []PortInfo) error {
	w := tabwriter.NewWriter(out, 0, 0, 4, ' ', 0)
	defer w.Flush()

	// ヘッダー
	fmt.Fprintln(w, "SERVICE\tHOST PORT\tCONTAINER PORT\tSTATUS")

	for _, info := range portInfos {
		status := "-"
		if info.Overridden {
			status = fmt.Sprintf("overridden (%d)", info.OriginalPort)
		}

		fmt.Fprintf(w, "%s\t%d\t%d\t%s\n",
			info.Service,
			info.HostPort,
			info.ContainerPort,
			status,
		)
	}

	return nil
}

// outputJSON はJSON形式で出力します。
func outputJSON(out io.Writer, portInfos []PortInfo) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(portInfos)
}

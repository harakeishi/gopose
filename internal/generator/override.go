package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/harakeishi/gopose/internal/errors"
	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
	"gopkg.in/yaml.v3"
)

// OverrideGeneratorImpl はOverride生成の実装です。
type OverrideGeneratorImpl struct {
	logger logger.Logger
}

// NewOverrideGeneratorImpl は新しいOverrideGeneratorImplを作成します。
func NewOverrideGeneratorImpl(logger logger.Logger) *OverrideGeneratorImpl {
	return &OverrideGeneratorImpl{
		logger: logger,
	}
}

// GenerateOverride はoverride.ymlファイルを生成します。
func (g *OverrideGeneratorImpl) GenerateOverride(ctx context.Context, config *types.ComposeConfig, resolutions []types.ConflictResolution) (*types.OverrideConfig, error) {
	g.logger.Debug(ctx, "Override生成開始",
		types.Field{Key: "resolutions_count", Value: len(resolutions)})

	// OverrideConfigの基本構造を作成
	// Docker Composeの最新バージョンではversionフィールドは非推奨のため、
	// 元のファイルにバージョンが指定されていない場合は空のままにする
	override := &types.OverrideConfig{
		Version:  config.Version, // 空文字列でも許可
		Services: make(map[string]types.ServiceOverride),
		Metadata: types.OverrideMetadata{
			GeneratedAt: time.Now(),
			Version:     "1.0.0", // goposeのバージョン
			Resolutions: resolutions,
		},
	}

	// 解決案をサービス別にグループ化
	serviceResolutions := make(map[string][]types.ConflictResolution)
	for _, resolution := range resolutions {
		serviceName := resolution.ServiceName
		if serviceName == "" {
			serviceName = resolution.Service // フォールバック
		}
		serviceResolutions[serviceName] = append(serviceResolutions[serviceName], resolution)
	}

	// すべてのサービスのオーバーライド設定を生成（ポートを持つサービスのみ）
	for serviceName, service := range config.Services {
		if len(service.Ports) > 0 {
			resolutionList := serviceResolutions[serviceName] // 解決案がない場合は空のスライス
			serviceOverride, err := g.generateServiceOverride(ctx, serviceName, resolutionList, config)
			if err != nil {
				return nil, fmt.Errorf("サービス %s のオーバーライド生成に失敗: %w", serviceName, err)
			}
			override.Services[serviceName] = serviceOverride
		}
	}

	g.logger.Info(ctx, "Override生成完了",
		types.Field{Key: "services_count", Value: len(override.Services)})

	return override, nil
}

// WriteOverrideFile はoverride.ymlファイルをディスクに書き込みます。
func (g *OverrideGeneratorImpl) WriteOverrideFile(ctx context.Context, override *types.OverrideConfig, outputPath string) error {
	g.logger.Debug(ctx, "Overrideファイル書き込み開始",
		types.Field{Key: "output_path", Value: outputPath})

	// ディレクトリが存在しない場合は作成
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &errors.AppError{
			Code:    errors.ErrFileWriteFailed,
			Message: fmt.Sprintf("ディレクトリ作成に失敗: %s", dir),
			Cause:   err,
			Fields: map[string]interface{}{
				"directory": dir,
			},
		}
	}

	// ヘッダーコメントを追加
	header := g.generateFileHeader()

	// カスタムYAML生成（!overrideタグ付き）
	yamlContent := g.generateOverrideYAML(override)

	finalContent := []byte(header + yamlContent)

	// ファイルに書き込み
	if err := os.WriteFile(outputPath, finalContent, 0644); err != nil {
		return &errors.AppError{
			Code:    errors.ErrFileWriteFailed,
			Message: fmt.Sprintf("ファイル書き込みに失敗: %s", outputPath),
			Cause:   err,
			Fields: map[string]interface{}{
				"file_path": outputPath,
			},
		}
	}

	g.logger.Info(ctx, "Overrideファイル書き込み完了",
		types.Field{Key: "output_path", Value: outputPath},
		types.Field{Key: "file_size", Value: len(finalContent)})

	return nil
}

// ValidateOverride はoverride設定の妥当性を検証します。
func (g *OverrideGeneratorImpl) ValidateOverride(ctx context.Context, override *types.OverrideConfig) error {
	g.logger.Debug(ctx, "Override検証開始")

	// バージョン検証（Docker Composeの最新バージョンではversionフィールドは非推奨のため、空でも許可）
	if override.Version == "" {
		g.logger.Debug(ctx, "Overrideのバージョンが指定されていませんが、Docker Composeの最新バージョンでは非推奨のため許可します")
	}

	// サービス数の検証
	if len(override.Services) == 0 {
		g.logger.Warn(ctx, "オーバーライドするサービスがありません")
		return nil
	}

	// 各サービスのポート設定を検証
	for serviceName, serviceOverride := range override.Services {
		if err := g.validateServiceOverride(ctx, serviceName, serviceOverride); err != nil {
			return fmt.Errorf("サービス %s の検証に失敗: %w", serviceName, err)
		}
	}

	// 解決案の重複チェック
	if err := g.validateResolutionUniqueness(ctx, override.Metadata.Resolutions); err != nil {
		return err
	}

	g.logger.Info(ctx, "Override検証完了")
	return nil
}

// generateServiceOverride は個別サービスのオーバーライド設定を生成します。
func (g *OverrideGeneratorImpl) generateServiceOverride(ctx context.Context, serviceName string, resolutions []types.ConflictResolution, originalConfig *types.ComposeConfig) (types.ServiceOverride, error) {
	serviceOverride := types.ServiceOverride{
		Ports: make([]types.PortMapping, 0),
	}

	// 元のサービス設定を取得
	originalService, exists := originalConfig.Services[serviceName]
	if !exists {
		return serviceOverride, &errors.AppError{
			Code:    errors.ErrValidationFailed,
			Message: fmt.Sprintf("元の設定にサービス %s が見つかりません", serviceName),
			Fields: map[string]interface{}{
				"service_name": serviceName,
			},
		}
	}

	// 解決案に基づいてポートマッピングを更新
	portMappings := make([]types.PortMapping, len(originalService.Ports))
	copy(portMappings, originalService.Ports)

	for _, resolution := range resolutions {
		// 対応するポートマッピングを検索して更新
		for i, mapping := range portMappings {
			if mapping.Host == resolution.ConflictPort {
				portMappings[i].Host = resolution.ResolvedPort
				g.logger.Debug(ctx, "ポートマッピング更新",
					types.Field{Key: "service", Value: serviceName},
					types.Field{Key: "old_port", Value: resolution.ConflictPort},
					types.Field{Key: "new_port", Value: resolution.ResolvedPort})
				break
			}
		}
	}

	serviceOverride.Ports = portMappings

	return serviceOverride, nil
}

// validateServiceOverride はサービスオーバーライドの妥当性を検証します。
func (g *OverrideGeneratorImpl) validateServiceOverride(ctx context.Context, serviceName string, serviceOverride types.ServiceOverride) error {
	// ポートの重複チェック
	portMap := make(map[int]bool)
	for _, portMapping := range serviceOverride.Ports {
		if portMapping.Host != 0 { // ホストポートが指定されている場合のみ
			if portMap[portMapping.Host] {
				return &errors.AppError{
					Code:    errors.ErrValidationFailed,
					Message: fmt.Sprintf("サービス %s で重複するホストポート: %d", serviceName, portMapping.Host),
					Fields: map[string]interface{}{
						"service":   serviceName,
						"host_port": portMapping.Host,
					},
				}
			}
			portMap[portMapping.Host] = true
		}

		// ポート範囲の検証
		if portMapping.Host < 0 || portMapping.Host > 65535 {
			return &errors.AppError{
				Code:    errors.ErrValidationFailed,
				Message: fmt.Sprintf("無効なホストポート: %d", portMapping.Host),
				Fields: map[string]interface{}{
					"service":   serviceName,
					"host_port": portMapping.Host,
				},
			}
		}

		if portMapping.Container < 1 || portMapping.Container > 65535 {
			return &errors.AppError{
				Code:    errors.ErrValidationFailed,
				Message: fmt.Sprintf("無効なコンテナポート: %d", portMapping.Container),
				Fields: map[string]interface{}{
					"service":        serviceName,
					"container_port": portMapping.Container,
				},
			}
		}
	}

	return nil
}

// validateResolutionUniqueness は解決案の重複をチェックします。
func (g *OverrideGeneratorImpl) validateResolutionUniqueness(ctx context.Context, resolutions []types.ConflictResolution) error {
	resolvedPorts := make(map[int]string) // port -> service name

	for _, resolution := range resolutions {
		serviceName := resolution.ServiceName
		if serviceName == "" {
			serviceName = resolution.Service
		}

		if existingService, exists := resolvedPorts[resolution.ResolvedPort]; exists {
			return &errors.AppError{
				Code: errors.ErrValidationFailed,
				Message: fmt.Sprintf("解決ポート %d がサービス %s と %s で重複しています",
					resolution.ResolvedPort, existingService, serviceName),
				Fields: map[string]interface{}{
					"resolved_port": resolution.ResolvedPort,
					"service1":      existingService,
					"service2":      serviceName,
				},
			}
		}
		resolvedPorts[resolution.ResolvedPort] = serviceName
	}

	return nil
}


// generateOverrideYAML は!overrideタグ付きのYAMLを生成します。
func (g *OverrideGeneratorImpl) generateOverrideYAML(override *types.OverrideConfig) string {
	var builder strings.Builder

	// プロジェクト名がある場合は先頭に出力
	if override.Name != "" {
		builder.WriteString(fmt.Sprintf("name: %s\n\n", override.Name))
	}

	builder.WriteString("services:\n")

	for serviceName, serviceOverride := range override.Services {
		builder.WriteString(fmt.Sprintf("    %s:\n", serviceName))

		if len(serviceOverride.Ports) > 0 {
			builder.WriteString("        ports: !reset\n")
			for _, port := range serviceOverride.Ports {
				if port.Host != 0 {
					builder.WriteString(fmt.Sprintf("            - \"%d:%d\"\n", port.Host, port.Container))
				}
			}
		}

		if len(serviceOverride.Networks) > 0 {
			builder.WriteString("        networks:\n")
			for netName, netConfig := range serviceOverride.Networks {
				builder.WriteString(fmt.Sprintf("            %s:\n", netName))
				if netConfig.IPv4Address != "" {
					builder.WriteString(fmt.Sprintf("                ipv4_address: %s\n", netConfig.IPv4Address))
				}
			}
		}
	}

	if len(override.Networks) > 0 {
		builder.WriteString("networks:\n")
		for netName, netOverride := range override.Networks {
			builder.WriteString(fmt.Sprintf("    %s:\n", netName))
			if len(netOverride.IPAM.Config) > 0 {
				builder.WriteString("        ipam:\n")
				builder.WriteString("            config:\n")
				for _, cfg := range netOverride.IPAM.Config {
					builder.WriteString(fmt.Sprintf("                - subnet: \"%s\"\n", cfg.Subnet))
				}
			}
		}
	}

	return builder.String()
}

// generateFileHeader はファイルヘッダーコメントを生成します。
func (g *OverrideGeneratorImpl) generateFileHeader() string {
	return fmt.Sprintf(`# Docker Compose Override File
# Generated by gopose (Go Port Override Solution Engine)
# Generated at: %s
# 
# This file contains port mappings to resolve conflicts detected in your
# original docker-compose.yml file. The original file remains unchanged.
# 
# To use this override:
# 1. Keep this file in the same directory as your docker-compose.yml
# 2. Run: docker-compose up
# 
# Docker Compose will automatically merge both files.
# 
# WARNING: This file is auto-generated. Manual changes may be overwritten.

`, time.Now().Format(time.RFC3339))
}

// OverrideTemplateGeneratorImpl はテンプレートベースのOverride生成実装です。
type OverrideTemplateGeneratorImpl struct {
	logger logger.Logger
}

// NewOverrideTemplateGeneratorImpl は新しいOverrideTemplateGeneratorImplを作成します。
func NewOverrideTemplateGeneratorImpl(logger logger.Logger) *OverrideTemplateGeneratorImpl {
	return &OverrideTemplateGeneratorImpl{
		logger: logger,
	}
}

// GenerateFromTemplate はテンプレートからOverrideを生成します。
func (t *OverrideTemplateGeneratorImpl) GenerateFromTemplate(ctx context.Context, templatePath string, data interface{}) (*types.OverrideConfig, error) {
	t.logger.Debug(ctx, "テンプレートからOverride生成開始",
		types.Field{Key: "template_path", Value: templatePath})

	// テンプレートファイルの読み込み
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, &errors.AppError{
			Code:    errors.ErrFileReadFailed,
			Message: fmt.Sprintf("テンプレートファイル読み込みに失敗: %s", templatePath),
			Cause:   err,
			Fields: map[string]interface{}{
				"template_path": templatePath,
			},
		}
	}

	// 簡単なテンプレート処理（実際の実装ではtext/templateを使用することを推奨）
	processedContent := string(templateContent)

	// YAMLとして解析
	var override types.OverrideConfig
	if err := yaml.Unmarshal([]byte(processedContent), &override); err != nil {
		return nil, &errors.AppError{
			Code:    errors.ErrParseFailed,
			Message: "テンプレートのYAML解析に失敗",
			Cause:   err,
			Fields: map[string]interface{}{
				"template_path": templatePath,
			},
		}
	}

	t.logger.Info(ctx, "テンプレートからOverride生成完了",
		types.Field{Key: "services_count", Value: len(override.Services)})

	return &override, nil
}

// ValidateTemplate はテンプレートの妥当性を検証します。
func (t *OverrideTemplateGeneratorImpl) ValidateTemplate(ctx context.Context, templatePath string) error {
	t.logger.Debug(ctx, "テンプレート検証開始",
		types.Field{Key: "template_path", Value: templatePath})

	// ファイルの存在確認
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		return &errors.AppError{
			Code:    errors.ErrFileNotFound,
			Message: fmt.Sprintf("テンプレートファイルが見つかりません: %s", templatePath),
			Fields: map[string]interface{}{
				"template_path": templatePath,
			},
		}
	}

	// ファイル拡張子の確認
	ext := filepath.Ext(templatePath)
	if ext != ".yml" && ext != ".yaml" {
		t.logger.Warn(ctx, "テンプレートファイルの拡張子が標準的ではありません",
			types.Field{Key: "extension", Value: ext})
	}

	// 基本的なYAML構文チェック
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return &errors.AppError{
			Code:    errors.ErrFileReadFailed,
			Message: "テンプレートファイル読み込みに失敗",
			Cause:   err,
		}
	}

	var tempData interface{}
	if err := yaml.Unmarshal(content, &tempData); err != nil {
		return &errors.AppError{
			Code:    errors.ErrValidationFailed,
			Message: "テンプレートのYAML構文が無効です",
			Cause:   err,
			Fields: map[string]interface{}{
				"template_path": templatePath,
			},
		}
	}

	t.logger.Info(ctx, "テンプレート検証完了")
	return nil
}

// Package generator は、docker-compose.override.yml生成機能を提供します。
package generator

import (
	"context"

	"github.com/harakeishi/gopose/pkg/types"
)

// OverrideGenerator はoverride.yml構造生成を行うインターフェースです。
type OverrideGenerator interface {
	Generate(ctx context.Context, resolutions []types.ConflictResolution, originalConfig *types.ComposeConfig) (*types.OverrideConfig, error)
	GenerateForServices(ctx context.Context, services []types.Service) (*types.OverrideConfig, error)
	GenerateMinimal(ctx context.Context, resolutions []types.ConflictResolution) (*types.OverrideConfig, error)
}

// OverrideValidator は生成内容の妥当性検証を行うインターフェースです。
type OverrideValidator interface {
	ValidateOverride(ctx context.Context, override *types.OverrideConfig) error
	ValidateCompatibility(ctx context.Context, override *types.OverrideConfig, original *types.ComposeConfig) error
	ValidatePortMappings(ctx context.Context, override *types.OverrideConfig) error
}

// OverrideMerger はoverride設定のマージを行うインターフェースです。
type OverrideMerger interface {
	MergeOverrides(ctx context.Context, base *types.OverrideConfig, additional *types.OverrideConfig) (*types.OverrideConfig, error)
	MergeWithOriginal(ctx context.Context, original *types.ComposeConfig, override *types.OverrideConfig) (*types.ComposeConfig, error)
}

// MetadataManager はメタデータ管理を行うインターフェースです。
type MetadataManager interface {
	CreateMetadata(ctx context.Context, resolutions []types.ConflictResolution) (*types.OverrideMetadata, error)
	ValidateMetadata(ctx context.Context, metadata *types.OverrideMetadata) error
	ExtractMetadata(ctx context.Context, override *types.OverrideConfig) (*types.OverrideMetadata, error)
}

// TemplateEngine はテンプレートエンジンのインターフェースです。
type TemplateEngine interface {
	RenderOverride(ctx context.Context, template string, data GenerationData) (string, error)
	LoadTemplate(ctx context.Context, templateName string) (string, error)
	RegisterFunction(name string, fn interface{}) error
}

// GenerationData は生成データを表します。
type GenerationData struct {
	Resolutions    []types.ConflictResolution `json:"resolutions"`
	OriginalConfig *types.ComposeConfig       `json:"original_config"`
	Services       []types.Service            `json:"services"`
	Metadata       *types.OverrideMetadata    `json:"metadata"`
	Options        GenerationOptions          `json:"options"`
}

// GenerationOptions は生成オプションを表します。
type GenerationOptions struct {
	MinimalOutput    bool              `json:"minimal_output"`
	IncludeMetadata  bool              `json:"include_metadata"`
	PreserveSections []string          `json:"preserve_sections"`
	CustomFields     map[string]string `json:"custom_fields"`
	Format           OutputFormat      `json:"format"`
}

// OutputFormat は出力形式を表します。
type OutputFormat string

const (
	OutputFormatYAML OutputFormat = "yaml"
	OutputFormatJSON OutputFormat = "json"
)

// GenerationResult は生成結果を表します。
type GenerationResult struct {
	Override         *types.OverrideConfig `json:"override"`
	Warnings         []string              `json:"warnings"`
	ModifiedServices []string              `json:"modified_services"`
	Success          bool                  `json:"success"`
	GenerationTime   int64                 `json:"generation_time_ms"`
}

// ServiceOverrideData はサービスオーバーライドデータを表します。
type ServiceOverrideData struct {
	Name            string                     `json:"name"`
	OriginalService *types.Service             `json:"original_service"`
	Resolutions     []types.ConflictResolution `json:"resolutions"`
	NewPorts        []types.PortMapping        `json:"new_ports"`
}

// OverrideStrategy はオーバーライド戦略を表します。
type OverrideStrategy string

const (
	OverrideStrategyMinimal  OverrideStrategy = "minimal"
	OverrideStrategyComplete OverrideStrategy = "complete"
	OverrideStrategyPreserve OverrideStrategy = "preserve"
)

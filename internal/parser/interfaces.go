// Package parser は、Docker Composeファイルの解析機能を提供します。
package parser

import (
	"context"
	"io"

	"github.com/harakeishi/gopose/pkg/types"
)

// ComposeParser はDocker Composeファイル解析を行うインターフェースです。
type ComposeParser interface {
	Parse(ctx context.Context, filePath string) (*types.ComposeConfig, error)
	ParseFromReader(ctx context.Context, reader io.Reader) (*types.ComposeConfig, error)
	ParseFromBytes(ctx context.Context, data []byte) (*types.ComposeConfig, error)
	Validate(ctx context.Context, config *types.ComposeConfig) error
}

// ServiceExtractor はサービス情報抽出を行うインターフェースです。
type ServiceExtractor interface {
	ExtractServices(ctx context.Context, config *types.ComposeConfig) ([]types.Service, error)
	ExtractService(ctx context.Context, name string, config *types.ComposeConfig) (*types.Service, error)
	ExtractServiceDependencies(ctx context.Context, serviceName string, config *types.ComposeConfig) ([]string, error)
}

// PortExtractor はポートマッピング情報抽出を行うインターフェースです。
type PortExtractor interface {
	ExtractPorts(ctx context.Context, config *types.ComposeConfig) (map[string][]types.PortMapping, error)
	ExtractPortsFromService(ctx context.Context, service *types.Service) ([]types.PortMapping, error)
	ExtractExposedPorts(ctx context.Context, config *types.ComposeConfig) (map[string][]int, error)
}

// FormatDetector はDocker Composeファイル形式検出を行うインターフェースです。
type FormatDetector interface {
	DetectFormat(ctx context.Context, filePath string) (ComposeFormat, error)
	DetectFormatFromBytes(ctx context.Context, data []byte) (ComposeFormat, error)
	DetectVersion(ctx context.Context, config *types.ComposeConfig) (string, error)
}

// ComposeValidator はDocker Compose設定の妥当性検証を行うインターフェースです。
type ComposeValidator interface {
	ValidateConfig(ctx context.Context, config *types.ComposeConfig) error
	ValidateService(ctx context.Context, service *types.Service) error
	ValidateNetworks(ctx context.Context, networks map[string]types.Network) error
	ValidateVolumes(ctx context.Context, volumes map[string]types.Volume) error
}

// ComposeFormat はDocker Composeファイルの形式を表します。
type ComposeFormat string

const (
	ComposeFormatYAML ComposeFormat = "yaml"
	ComposeFormatJSON ComposeFormat = "json"
)

// ParseOptions は解析オプションを表します。
type ParseOptions struct {
	StrictMode       bool     `json:"strict_mode"`
	AllowedVersions  []string `json:"allowed_versions"`
	IgnoreExtensions bool     `json:"ignore_extensions"`
	ValidateOnly     bool     `json:"validate_only"`
}

// ParseResult は解析結果を表します。
type ParseResult struct {
	Config   *types.ComposeConfig `json:"config"`
	Format   ComposeFormat        `json:"format"`
	Version  string               `json:"version"`
	Warnings []string             `json:"warnings"`
	Errors   []string             `json:"errors"`
}

// ServiceInfo はサービス情報の詳細を表します。
type ServiceInfo struct {
	Name         string              `json:"name"`
	Image        string              `json:"image"`
	Ports        []types.PortMapping `json:"ports"`
	Dependencies []string            `json:"dependencies"`
	Networks     []string            `json:"networks"`
	Volumes      []string            `json:"volumes"`
	Environment  map[string]string   `json:"environment"`
}

// ComposeDependencyGraph は依存関係グラフを表します。
type ComposeDependencyGraph struct {
	Services map[string][]string `json:"services"`
	Order    []string            `json:"order"`
}

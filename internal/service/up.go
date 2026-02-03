package service

import (
	"context"
	"fmt"

	"github.com/harakeishi/gopose/internal/generator"
	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/internal/parser"
	"github.com/harakeishi/gopose/internal/scanner"
	"github.com/harakeishi/gopose/pkg/types"
)

// ComposeFileParser はComposeファイル解析のインターフェースです。
type ComposeFileParser interface {
	ParseComposeFile(ctx context.Context, filePath string) (*types.ComposeConfig, error)
}

// UpParams はUpサービスの入力パラメータです。
type UpParams struct {
	ComposeFilePath string
	WorkDir         string
	OutputFile      string
	ProjectName     string
	Strategy        string
	PortConfig      types.PortConfig
	DryRun          bool
}

// UpResult はUpサービスの実行結果です。
type UpResult struct {
	HasConflicts     bool
	PortConflicts    int
	NetworkConflicts int
	OutputFile       string
	Override         *types.OverrideConfig
}

// UpServiceDeps はUpServiceの依存関係です。
type UpServiceDeps struct {
	Logger              logger.Logger
	ComposeFileDetector parser.ComposeFileDetector
	ComposeParser       ComposeFileParser
	ConflictDetector    scanner.UnifiedConflictDetector
	OverrideGenerator   generator.UnifiedOverrideGenerator
	OverrideWriter      generator.OverrideWriter
}

// UpService は衝突検知→解決→override生成のワークフローを担当します。
type UpService struct {
	deps UpServiceDeps
}

// NewUpService は新しいUpServiceを作成します。
func NewUpService(deps UpServiceDeps) *UpService {
	return &UpService{deps: deps}
}

// Execute はUpワークフローを実行します。
func (s *UpService) Execute(ctx context.Context, params UpParams) (*UpResult, error) {
	log := s.deps.Logger

	// Docker Composeファイルの自動検出（指定されていない場合）
	composeFile := params.ComposeFilePath
	if composeFile == "" {
		detected, err := s.deps.ComposeFileDetector.GetDefaultComposeFile(ctx, params.WorkDir)
		if err != nil {
			return nil, fmt.Errorf("docker composeファイルの自動検出に失敗: %w", err)
		}
		composeFile = detected
		log.Info(ctx, "Docker Composeファイルを自動検出", types.Field{Key: "file", Value: composeFile})
	}

	// Docker Composeファイルの解析
	config, err := s.deps.ComposeParser.ParseComposeFile(ctx, composeFile)
	if err != nil {
		return nil, fmt.Errorf("docker composeファイルの解析に失敗: %w", err)
	}

	// 統一的な衝突検知の実行
	conflictInfo, err := s.deps.ConflictDetector.DetectConflicts(ctx, config, params.ProjectName)
	if err != nil {
		return nil, fmt.Errorf("衝突検知に失敗: %w", err)
	}

	// 衝突がない場合
	if !conflictInfo.HasConflicts() {
		log.Info(ctx, "衝突は検出されませんでした")
		return &UpResult{HasConflicts: false}, nil
	}

	// 衝突結果の表示
	log.Info(ctx, "衝突検知完了",
		types.Field{Key: "port_conflicts", Value: len(conflictInfo.PortConflicts)},
		types.Field{Key: "network_conflicts", Value: len(conflictInfo.NetworkConflicts)})

	// 解決戦略の決定
	resolutionStrategy := types.ResolutionStrategyAutoIncrement
	switch params.Strategy {
	case "auto":
		resolutionStrategy = types.ResolutionStrategyAutoIncrement
	case "range":
		resolutionStrategy = types.ResolutionStrategyRangeAllocation
	case "user":
		resolutionStrategy = types.ResolutionStrategyUserDefined
	}

	// 統一的な衝突解決
	if err := s.deps.OverrideGenerator.ResolveConflicts(ctx, conflictInfo, resolutionStrategy, params.PortConfig); err != nil {
		return nil, fmt.Errorf("衝突解決に失敗: %w", err)
	}

	// 解決結果のログ出力
	s.logResolutions(ctx, conflictInfo)

	// 統一的なOverride.ymlの生成
	override, err := s.deps.OverrideGenerator.GenerateFromConflicts(ctx, config, conflictInfo)
	if err != nil {
		return nil, fmt.Errorf("overrideファイルの生成に失敗: %w", err)
	}

	// プロジェクト名をoverrideに設定
	if params.ProjectName != "" {
		override.Name = params.ProjectName
		log.Debug(ctx, "Override.ymlにプロジェクト名を設定",
			types.Field{Key: "project_name", Value: params.ProjectName})
	}

	// Override.ymlの妥当性検証
	if err := s.deps.OverrideWriter.ValidateOverride(ctx, override); err != nil {
		return nil, fmt.Errorf("overrideファイルの検証に失敗: %w", err)
	}

	// 出力ファイル名の決定
	outputFile := params.OutputFile
	if outputFile == "" {
		outputFile = "compose.override.yml"
	}

	// ドライランモードでない場合のみファイル書き込み
	if !params.DryRun {
		if err := s.deps.OverrideWriter.WriteOverrideFile(ctx, override, outputFile); err != nil {
			return nil, fmt.Errorf("overrideファイルの書き込みに失敗: %w", err)
		}
		log.Info(ctx, "Override.ymlファイルが生成されました",
			types.Field{Key: "output_file", Value: outputFile})
	} else {
		log.Info(ctx, "ドライランモードのため、ファイルは生成されません")
	}

	return &UpResult{
		HasConflicts:     true,
		PortConflicts:    len(conflictInfo.PortConflicts),
		NetworkConflicts: len(conflictInfo.NetworkConflicts),
		OutputFile:       outputFile,
		Override:         override,
	}, nil
}

// logResolutions は解決結果をログ出力します。
func (s *UpService) logResolutions(ctx context.Context, conflictInfo *types.UnifiedConflictInfo) {
	log := s.deps.Logger

	for _, conflict := range conflictInfo.PortConflicts {
		if conflict.Resolution != nil {
			log.Info(ctx, "ポート解決",
				types.Field{Key: "service", Value: conflict.ServiceName},
				types.Field{Key: "from", Value: conflict.Port},
				types.Field{Key: "to", Value: conflict.Resolution.ResolvedPort},
				types.Field{Key: "reason", Value: conflict.Resolution.Reason})
		}
	}

	for _, conflict := range conflictInfo.NetworkConflicts {
		if conflict.Resolution != nil {
			log.Info(ctx, "ネットワーク解決",
				types.Field{Key: "network", Value: conflict.NetworkName},
				types.Field{Key: "from", Value: conflict.OriginalSubnet},
				types.Field{Key: "to", Value: conflict.Resolution.ResolvedSubnet},
				types.Field{Key: "reason", Value: conflict.Resolution.Reason})
		}
	}
}

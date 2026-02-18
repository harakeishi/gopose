package scanner

import (
	"context"
	"fmt"

	"github.com/harakeishi/gopose/internal/errors"
	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
)

// PortValidatorImpl はポート検証の実装です。
type PortValidatorImpl struct {
	logger logger.Logger
}

// NewPortValidatorImpl は新しいPortValidatorImplを作成します。
func NewPortValidatorImpl(logger logger.Logger) *PortValidatorImpl {
	return &PortValidatorImpl{
		logger: logger,
	}
}

// ValidatePort は単一ポートの妥当性を検証します。
func (v *PortValidatorImpl) ValidatePort(ctx context.Context, port int) error {
	if port < 1 || port > 65535 {
		return &errors.AppError{
			Code:    errors.ErrPortRangeInvalid,
			Message: fmt.Sprintf("無効なポート番号です: %d", port),
			Fields: map[string]interface{}{
				"port":     port,
				"min_port": 1,
				"max_port": 65535,
			},
		}
	}

	v.logger.Debug(ctx, "ポート検証成功", types.Field{Key: "port", Value: port})
	return nil
}

// ValidatePortRange はポート範囲の妥当性を検証します。
func (v *PortValidatorImpl) ValidatePortRange(ctx context.Context, portRange types.PortRange) error {
	// 開始ポートの検証
	if err := v.ValidatePort(ctx, portRange.Start); err != nil {
		return fmt.Errorf("開始ポートが無効です: %w", err)
	}

	// 終了ポートの検証
	if err := v.ValidatePort(ctx, portRange.End); err != nil {
		return fmt.Errorf("終了ポートが無効です: %w", err)
	}

	// 範囲の論理検証
	if portRange.Start > portRange.End {
		return &errors.AppError{
			Code:    errors.ErrPortRangeInvalid,
			Message: "開始ポートが終了ポートより大きいです",
			Fields: map[string]interface{}{
				"start_port": portRange.Start,
				"end_port":   portRange.End,
			},
		}
	}

	// 範囲のサイズ検証（あまりに大きい範囲は警告）
	rangeSize := portRange.End - portRange.Start + 1
	if rangeSize > 10000 {
		v.logger.Warn(ctx, "非常に大きなポート範囲が指定されています",
			types.Field{Key: "range_size", Value: rangeSize},
			types.Field{Key: "start_port", Value: portRange.Start},
			types.Field{Key: "end_port", Value: portRange.End})
	}

	v.logger.Debug(ctx, "ポート範囲検証成功",
		types.Field{Key: "start_port", Value: portRange.Start},
		types.Field{Key: "end_port", Value: portRange.End},
		types.Field{Key: "range_size", Value: rangeSize})

	return nil
}

// ValidatePortMapping はポートマッピングの妥当性を検証します。
func (v *PortValidatorImpl) ValidatePortMapping(ctx context.Context, mapping types.PortMapping) error {
	// ホストポートの検証
	if err := v.ValidatePort(ctx, mapping.Host); err != nil {
		return fmt.Errorf("ホストポートが無効です: %w", err)
	}

	// コンテナポートの検証
	if err := v.ValidatePort(ctx, mapping.Container); err != nil {
		return fmt.Errorf("コンテナポートが無効です: %w", err)
	}

	// プロトコルの検証
	if mapping.Protocol != "" {
		validProtocols := []string{"tcp", "udp", "sctp"}
		isValid := false
		for _, protocol := range validProtocols {
			if mapping.Protocol == protocol {
				isValid = true
				break
			}
		}
		if !isValid {
			return &errors.AppError{
				Code:    errors.ErrValidationFailed,
				Message: fmt.Sprintf("無効なプロトコルです: %s", mapping.Protocol),
				Fields: map[string]interface{}{
					"protocol":        mapping.Protocol,
					"valid_protocols": validProtocols,
				},
			}
		}
	}

	v.logger.Debug(ctx, "ポートマッピング検証成功",
		types.Field{Key: "host_port", Value: mapping.Host},
		types.Field{Key: "container_port", Value: mapping.Container},
		types.Field{Key: "protocol", Value: mapping.Protocol})

	return nil
}

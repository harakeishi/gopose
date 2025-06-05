package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/harakeishi/gopose/internal/errors"
	"github.com/harakeishi/gopose/internal/logger"
	"github.com/harakeishi/gopose/pkg/types"
	"gopkg.in/yaml.v3"
)

// YamlComposeParser はYAMLベースのDocker Compose解析実装です。
type YamlComposeParser struct {
	logger logger.Logger
}

// NewYamlComposeParser は新しいYamlComposeParserを作成します。
func NewYamlComposeParser(logger logger.Logger) *YamlComposeParser {
	return &YamlComposeParser{
		logger: logger,
	}
}

// ParseComposeFile はDocker Composeファイルを解析します。
func (p *YamlComposeParser) ParseComposeFile(ctx context.Context, filepath string) (*types.ComposeConfig, error) {
	p.logger.Debug(ctx, "Docker Composeファイル解析開始", types.Field{Key: "file", Value: filepath})

	// ファイルの存在確認
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return nil, &errors.AppError{
			Code:    errors.ErrFileNotFound,
			Message: fmt.Sprintf("Docker Composeファイルが見つかりません: %s", filepath),
			Fields: map[string]interface{}{
				"file_path": filepath,
			},
		}
	}

	// ファイル読み込み
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, &errors.AppError{
			Code:    errors.ErrFileReadFailed,
			Message: fmt.Sprintf("ファイル読み込みに失敗しました: %s", filepath),
			Cause:   err,
			Fields: map[string]interface{}{
				"file_path": filepath,
			},
		}
	}

	// YAML解析
	var rawCompose map[string]interface{}
	if err := yaml.Unmarshal(data, &rawCompose); err != nil {
		return nil, &errors.AppError{
			Code:    errors.ErrParseFailed,
			Message: "YAMLの解析に失敗しました",
			Cause:   err,
			Fields: map[string]interface{}{
				"file_path": filepath,
			},
		}
	}

	// ComposeConfigに変換
	config, err := p.convertToComposeConfig(ctx, rawCompose, filepath)
	if err != nil {
		return nil, err
	}

	p.logger.Info(ctx, "Docker Composeファイル解析完了",
		types.Field{Key: "file", Value: filepath},
		types.Field{Key: "services_count", Value: len(config.Services)})

	return config, nil
}

// ParseServicePorts はサービスのポート設定を解析します。
func (p *YamlComposeParser) ParseServicePorts(ctx context.Context, service map[string]interface{}) ([]types.PortMapping, error) {
	portsInterface, exists := service["ports"]
	if !exists {
		return []types.PortMapping{}, nil
	}

	var portMappings []types.PortMapping

	switch ports := portsInterface.(type) {
	case []interface{}:
		for _, portInterface := range ports {
			mapping, err := p.parsePortMapping(ctx, portInterface)
			if err != nil {
				return nil, err
			}
			if mapping != nil {
				portMappings = append(portMappings, *mapping)
			}
		}
	default:
		return nil, &errors.AppError{
			Code:    errors.ErrParseFailed,
			Message: "ポート設定の形式が無効です",
			Fields: map[string]interface{}{
				"ports_type": fmt.Sprintf("%T", ports),
			},
		}
	}

	return portMappings, nil
}

// ValidateComposeVersion はDocker Composeのバージョンを検証します。
func (p *YamlComposeParser) ValidateComposeVersion(ctx context.Context, version string) error {
	if version == "" {
		p.logger.Warn(ctx, "Docker Composeバージョンが指定されていません")
		return nil
	}

	// サポートされているバージョンのリスト
	supportedVersions := []string{"3.0", "3.1", "3.2", "3.3", "3.4", "3.5", "3.6", "3.7", "3.8", "3.9"}

	for _, supported := range supportedVersions {
		if version == supported || strings.HasPrefix(version, supported+".") {
			p.logger.Debug(ctx, "サポートされているDocker Composeバージョン",
				types.Field{Key: "version", Value: version})
			return nil
		}
	}

	// 警告として処理（エラーにはしない）
	p.logger.Warn(ctx, "未サポートのDocker Composeバージョンです",
		types.Field{Key: "version", Value: version},
		types.Field{Key: "supported_versions", Value: supportedVersions})

	return nil
}

// convertToComposeConfig は生のYAMLデータをComposeConfigに変換します。
func (p *YamlComposeParser) convertToComposeConfig(ctx context.Context, raw map[string]interface{}, filepath string) (*types.ComposeConfig, error) {
	config := &types.ComposeConfig{
		Version:  p.extractVersion(raw),
		Services: make(map[string]types.Service),
		FilePath: filepath,
	}

	// バージョン検証
	if err := p.ValidateComposeVersion(ctx, config.Version); err != nil {
		return nil, err
	}

	// サービス解析
	servicesInterface, exists := raw["services"]
	if !exists {
		return nil, &errors.AppError{
			Code:    errors.ErrParseFailed,
			Message: "servicesセクションが見つかりません",
		}
	}

	services, ok := servicesInterface.(map[string]interface{})
	if !ok {
		return nil, &errors.AppError{
			Code:    errors.ErrParseFailed,
			Message: "servicesセクションの形式が無効です",
		}
	}

	for serviceName, serviceInterface := range services {
		serviceMap, ok := serviceInterface.(map[string]interface{})
		if !ok {
			p.logger.Warn(ctx, "サービス設定の形式が無効です",
				types.Field{Key: "service", Value: serviceName})
			continue
		}

		service, err := p.convertToService(ctx, serviceName, serviceMap)
		if err != nil {
			return nil, fmt.Errorf("サービス %s の解析に失敗: %w", serviceName, err)
		}

		config.Services[serviceName] = service
	}

	return config, nil
}

// convertToService はサービス設定を変換します。
func (p *YamlComposeParser) convertToService(ctx context.Context, name string, serviceMap map[string]interface{}) (types.Service, error) {
	service := types.Service{
		Name: name,
	}

	// イメージ情報
	if image, exists := serviceMap["image"]; exists {
		if imageStr, ok := image.(string); ok {
			service.Image = imageStr
		}
	}

	// ポートマッピング解析
	portMappings, err := p.ParseServicePorts(ctx, serviceMap)
	if err != nil {
		return service, err
	}
	service.Ports = portMappings

	// 環境変数
	if env, exists := serviceMap["environment"]; exists {
		service.Environment = p.parseEnvironment(env)
	}

	// 依存関係
	if depends, exists := serviceMap["depends_on"]; exists {
		service.DependsOn = p.parseDependsOn(depends)
	}

	return service, nil
}

// parsePortMapping は個別のポートマッピングを解析します。
func (p *YamlComposeParser) parsePortMapping(ctx context.Context, portInterface interface{}) (*types.PortMapping, error) {
	switch port := portInterface.(type) {
	case string:
		return p.parsePortString(ctx, port)
	case int:
		// コンテナポートのみの場合
		return &types.PortMapping{
			Container: port,
			Protocol:  "tcp",
		}, nil
	case map[string]interface{}:
		return p.parsePortObject(ctx, port)
	default:
		p.logger.Warn(ctx, "サポートされていないポート形式",
			types.Field{Key: "port_type", Value: fmt.Sprintf("%T", port)})
		return nil, nil
	}
}

// parsePortString は文字列形式のポートマッピングを解析します。
func (p *YamlComposeParser) parsePortString(ctx context.Context, portStr string) (*types.PortMapping, error) {
	// 例: "8080:80", "8080:80/tcp", "127.0.0.1:8080:80"

	protocol := "tcp"
	portPart := portStr

	// プロトコル部分を分離
	if strings.Contains(portStr, "/") {
		parts := strings.Split(portStr, "/")
		if len(parts) == 2 {
			portPart = parts[0]
			protocol = parts[1]
		}
	}

	// ポート部分を解析
	re := regexp.MustCompile(`^(?:([^:]+):)?(\d+):(\d+)$|^(\d+)$`)
	matches := re.FindStringSubmatch(portPart)

	if len(matches) == 0 {
		return nil, &errors.AppError{
			Code:    errors.ErrParseFailed,
			Message: fmt.Sprintf("無効なポート形式: %s", portStr),
		}
	}

	var hostPort, containerPort int
	var err error

	if matches[4] != "" {
		// コンテナポートのみ（例: "80"）
		containerPort, err = strconv.Atoi(matches[4])
		if err != nil {
			return nil, &errors.AppError{
				Code:    errors.ErrParseFailed,
				Message: fmt.Sprintf("コンテナポートの解析に失敗: %s", matches[4]),
				Cause:   err,
			}
		}
		hostPort = 0 // ホストポートは指定なし
	} else {
		// ホスト:コンテナ形式（例: "8080:80"）
		hostPort, err = strconv.Atoi(matches[2])
		if err != nil {
			return nil, &errors.AppError{
				Code:    errors.ErrParseFailed,
				Message: fmt.Sprintf("ホストポートの解析に失敗: %s", matches[2]),
				Cause:   err,
			}
		}

		containerPort, err = strconv.Atoi(matches[3])
		if err != nil {
			return nil, &errors.AppError{
				Code:    errors.ErrParseFailed,
				Message: fmt.Sprintf("コンテナポートの解析に失敗: %s", matches[3]),
				Cause:   err,
			}
		}
	}

	mapping := &types.PortMapping{
		Host:      hostPort,
		Container: containerPort,
		Protocol:  protocol,
	}

	// IPアドレスが指定されている場合
	if matches[1] != "" {
		mapping.HostIP = matches[1]
	}

	return mapping, nil
}

// parsePortObject はオブジェクト形式のポートマッピングを解析します。
func (p *YamlComposeParser) parsePortObject(ctx context.Context, portObj map[string]interface{}) (*types.PortMapping, error) {
	mapping := &types.PortMapping{
		Protocol: "tcp", // デフォルト
	}

	// published (ホストポート)
	if published, exists := portObj["published"]; exists {
		if port, ok := published.(int); ok {
			mapping.Host = port
		} else if portStr, ok := published.(string); ok {
			port, err := strconv.Atoi(portStr)
			if err != nil {
				return nil, &errors.AppError{
					Code:    errors.ErrParseFailed,
					Message: fmt.Sprintf("publishedポートの解析に失敗: %s", portStr),
					Cause:   err,
				}
			}
			mapping.Host = port
		}
	}

	// target (コンテナポート)
	if target, exists := portObj["target"]; exists {
		if port, ok := target.(int); ok {
			mapping.Container = port
		} else if portStr, ok := target.(string); ok {
			port, err := strconv.Atoi(portStr)
			if err != nil {
				return nil, &errors.AppError{
					Code:    errors.ErrParseFailed,
					Message: fmt.Sprintf("targetポートの解析に失敗: %s", portStr),
					Cause:   err,
				}
			}
			mapping.Container = port
		}
	}

	// protocol
	if protocol, exists := portObj["protocol"]; exists {
		if protocolStr, ok := protocol.(string); ok {
			mapping.Protocol = protocolStr
		}
	}

	// host_ip
	if hostIP, exists := portObj["host_ip"]; exists {
		if hostIPStr, ok := hostIP.(string); ok {
			mapping.HostIP = hostIPStr
		}
	}

	return mapping, nil
}

// extractVersion はDockerComposeバージョンを抽出します。
func (p *YamlComposeParser) extractVersion(raw map[string]interface{}) string {
	if version, exists := raw["version"]; exists {
		if versionStr, ok := version.(string); ok {
			return versionStr
		}
	}
	return ""
}

// parseEnvironment は環境変数を解析します。
func (p *YamlComposeParser) parseEnvironment(env interface{}) map[string]string {
	result := make(map[string]string)

	switch e := env.(type) {
	case []interface{}:
		for _, item := range e {
			if itemStr, ok := item.(string); ok {
				if strings.Contains(itemStr, "=") {
					parts := strings.SplitN(itemStr, "=", 2)
					result[parts[0]] = parts[1]
				} else {
					result[itemStr] = ""
				}
			}
		}
	case map[string]interface{}:
		for key, value := range e {
			if valueStr, ok := value.(string); ok {
				result[key] = valueStr
			} else {
				result[key] = fmt.Sprintf("%v", value)
			}
		}
	}

	return result
}

// parseDependsOn は依存関係を解析します。
func (p *YamlComposeParser) parseDependsOn(depends interface{}) []string {
	var result []string

	switch d := depends.(type) {
	case []interface{}:
		for _, item := range d {
			if itemStr, ok := item.(string); ok {
				result = append(result, itemStr)
			}
		}
	case map[string]interface{}:
		for key := range d {
			result = append(result, key)
		}
	}

	return result
}

// ComposeFileDetectorImpl はCompose ファイル自動検出の実装です。
type ComposeFileDetectorImpl struct {
	logger logger.Logger
}

// NewComposeFileDetectorImpl は新しいComposeFileDetectorImplを作成します。
func NewComposeFileDetectorImpl(logger logger.Logger) *ComposeFileDetectorImpl {
	return &ComposeFileDetectorImpl{
		logger: logger,
	}
}

// DetectComposeFiles は指定されたディレクトリでCompose ファイルを検出します。
func (d *ComposeFileDetectorImpl) DetectComposeFiles(ctx context.Context, directory string) ([]string, error) {
	d.logger.Debug(ctx, "Docker Composeファイル検出開始", types.Field{Key: "directory", Value: directory})

	// 標準的なファイル名のリスト
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	var foundFiles []string

	for _, candidate := range candidates {
		filePath := filepath.Join(directory, candidate)
		if _, err := os.Stat(filePath); err == nil {
			foundFiles = append(foundFiles, filePath)
			d.logger.Debug(ctx, "Docker Composeファイル発見", types.Field{Key: "file", Value: filePath})
		}
	}

	if len(foundFiles) == 0 {
		return nil, &errors.AppError{
			Code:    errors.ErrFileNotFound,
			Message: fmt.Sprintf("Docker Composeファイルが見つかりません: %s", directory),
			Fields: map[string]interface{}{
				"directory":  directory,
				"candidates": candidates,
			},
		}
	}

	d.logger.Info(ctx, "Docker Composeファイル検出完了",
		types.Field{Key: "directory", Value: directory},
		types.Field{Key: "found_count", Value: len(foundFiles)})

	return foundFiles, nil
}

// GetDefaultComposeFile はデフォルトのCompose ファイルを取得します。
func (d *ComposeFileDetectorImpl) GetDefaultComposeFile(ctx context.Context, directory string) (string, error) {
	files, err := d.DetectComposeFiles(ctx, directory)
	if err != nil {
		return "", err
	}

	// 優先順位に従って最初に見つかったファイルを返す
	return files[0], nil
}

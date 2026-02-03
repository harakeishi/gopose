# gopose 簡素化 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** デッドコード削除と parser/yaml.go のメソッド分割により、gopose のコードベースを簡素化する

**Architecture:** resolver パッケージ全体と generator/interfaces.go の未使用型を削除（約1,400行削減）。parser/yaml.go の長いメソッド群を SRP に沿って分割する。公開インターフェースは変更しない。

**Tech Stack:** Go, Docker Compose YAML parsing

---

## Task 1: resolver パッケージの削除

**Files:**
- Delete: `internal/resolver/conflict.go`
- Delete: `internal/resolver/conflict_test.go`
- Delete: `internal/resolver/interfaces.go`

**Step 1: 既存テストがすべて通ることを確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./...`
Expected: ALL PASS

**Step 2: resolver ディレクトリを削除**

```bash
rm -rf internal/resolver/
```

**Step 3: ビルドとテストが通ることを確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go build ./... && go test ./...`
Expected: ALL PASS（resolver は一切参照されていないため影響なし）

**Step 4: コミット**

```bash
git add -A
git commit -m "ai/refactor: remove unused resolver package (~1,336 lines of dead code)"
```

---

## Task 2: generator/interfaces.go から未使用の型・インターフェースを削除

**Files:**
- Modify: `internal/generator/interfaces.go`

**削除対象:**
- `OverrideGenerator` インターフェース (L10-15) — 実装なし、参照なし
- `OverrideValidator` インターフェース (L23-28) — 実装なし、参照なし
- `OverrideMerger` インターフェース (L30-34) — 実装なし、参照なし
- `MetadataManager` インターフェース (L36-41) — 実装なし、参照なし
- `TemplateEngine` インターフェース (L43-48) — 実装なし、参照なし
- `GenerationData` 構造体 (L50-57) — TemplateEngine からのみ参照
- `GenerationOptions` 構造体 (L59-66) — GenerationData からのみ参照
- `OutputFormat` 型と定数 (L68-74) — GenerationOptions からのみ参照
- `GenerationResult` 構造体 (L76-83) — 参照なし
- `ServiceOverrideData` 構造体 (L85-91) — 参照なし
- `OverrideStrategy` 型と定数 (L93-100) — 参照なし

**残す対象:**
- `UnifiedOverrideGenerator` インターフェース (L17-21) — `UnifiedOverrideGeneratorImpl` が実装

**Step 1: interfaces.go を編集して未使用の型を削除**

削除後のファイル内容:

```go
// Package generator は、docker-compose.override.yml生成機能を提供します。
package generator

import (
	"context"

	"github.com/harakeishi/gopose/pkg/types"
)

// UnifiedOverrideGenerator は統一的な衝突情報からoverride生成を行うインターフェースです。
type UnifiedOverrideGenerator interface {
	GenerateFromConflicts(ctx context.Context, config *types.ComposeConfig, conflictInfo *types.UnifiedConflictInfo) (*types.OverrideConfig, error)
	ResolveConflicts(ctx context.Context, conflictInfo *types.UnifiedConflictInfo, strategy types.ResolutionStrategy, portConfig types.PortConfig) error
}
```

**Step 2: ビルドとテストが通ることを確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go build ./... && go test ./...`
Expected: ALL PASS

**Step 3: コミット**

```bash
git add internal/generator/interfaces.go
git commit -m "ai/refactor: remove unused interfaces and types from generator package"
```

---

## Task 3: parsePortString の分割 (SRP)

**Files:**
- Modify: `internal/parser/yaml.go`
- Test: `internal/parser/yaml_test.go` (既存テストで検証)

**現状:** `parsePortString` (L307-387, 81行) が以下の複数責務を持つ:
1. 環境変数展開
2. プロトコル分離
3. 正規表現マッチ
4. マッチ結果からの PortMapping 構築

**分割方針:**

- `parsePortString` → オーケストレーションのみ（~15行）
- `extractProtocol(portStr string) (portPart, protocol string)` — プロトコル分離
- `buildPortMappingFromMatches(matches []string, protocol string) (*types.PortMapping, error)` — マッチ結果からの構築

**Step 1: ヘルパー関数を追加**

```go
// extractProtocol はポート文字列からプロトコル部分を分離します。
func extractProtocol(portStr string) (portPart, protocol string) {
	if idx := strings.Index(portStr, "/"); idx >= 0 {
		return portStr[:idx], portStr[idx+1:]
	}
	return portStr, "tcp"
}

// buildPortMappingFromRegexMatches は正規表現マッチ結果から PortMapping を構築します。
// matches は re.FindStringSubmatch の結果: [full, hostIP, hostPort, containerPort, singlePort]
func buildPortMappingFromRegexMatches(matches []string, protocol string) (*types.PortMapping, error) {
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

	if matches[1] != "" {
		mapping.HostIP = matches[1]
	}

	return mapping, nil
}
```

**Step 2: parsePortString をリファクタリング**

```go
// portStringPattern はポート文字列の正規表現パターンです。
// 形式: [host_ip:]host_port:container_port または container_port のみ
var portStringPattern = regexp.MustCompile(`^(?:([\d\.]+):)?(\d+):(\d+)$|^(\d+)$`)

func (p *YamlComposeParser) parsePortString(ctx context.Context, portStr string) (*types.PortMapping, error) {
	expanded := expandVariables(portStr)
	portPart, protocol := extractProtocol(expanded)

	matches := portStringPattern.FindStringSubmatch(portPart)
	if len(matches) == 0 {
		return nil, &errors.AppError{
			Code:    errors.ErrParseFailed,
			Message: fmt.Sprintf("無効なポート形式: %s (展開後: %s)", portStr, expanded),
		}
	}

	return buildPortMappingFromRegexMatches(matches, protocol)
}
```

**Step 3: テストが通ることを確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./internal/parser/ -v`
Expected: ALL PASS

**Step 4: コミット**

```bash
git add internal/parser/yaml.go
git commit -m "ai/refactor(parser): split parsePortString into smaller single-responsibility functions"
```

---

## Task 4: convertToComposeConfig の分割 (SRP)

**Files:**
- Modify: `internal/parser/yaml.go`
- Test: `internal/parser/yaml_test.go` (既存テストで検証)

**現状:** `convertToComposeConfig` (L118-205, 88行) が3つの責務を持つ:
1. サービス解析
2. ネットワーク解析
3. ボリューム解析

**分割方針:**

- `convertToComposeConfig` → 初期化とオーケストレーション（~20行）
- `parseServicesSection(ctx, raw) (map[string]types.Service, error)` — サービスセクション解析
- `parseNetworksSection(ctx, raw) (map[string]types.Network, error)` — ネットワークセクション解析
- `parseVolumesSection(ctx, raw) (map[string]types.Volume, error)` — ボリュームセクション解析

**Step 1: 3つのセクション解析関数を追加**

```go
// parseServicesSection はservicesセクションを解析します。
func (p *YamlComposeParser) parseServicesSection(ctx context.Context, raw map[string]interface{}) (map[string]types.Service, error) {
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

	result := make(map[string]types.Service)
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
		result[serviceName] = service
	}
	return result, nil
}

// parseNetworksSection はnetworksセクションを解析します。
func (p *YamlComposeParser) parseNetworksSection(ctx context.Context, raw map[string]interface{}) (map[string]types.Network, error) {
	result := make(map[string]types.Network)
	networksInterface, exists := raw["networks"]
	if !exists {
		return result, nil
	}

	networks, ok := networksInterface.(map[string]interface{})
	if !ok {
		return result, nil
	}

	for networkName, networkInterface := range networks {
		networkMap, ok := networkInterface.(map[string]interface{})
		if !ok {
			p.logger.Warn(ctx, "ネットワーク設定の形式が無効です",
				types.Field{Key: "network", Value: networkName})
			continue
		}

		network, err := p.convertToNetwork(ctx, networkName, networkMap)
		if err != nil {
			return nil, fmt.Errorf("ネットワーク %s の解析に失敗: %w", networkName, err)
		}
		result[networkName] = network
	}
	return result, nil
}

// parseVolumesSection はvolumesセクションを解析します。
func (p *YamlComposeParser) parseVolumesSection(ctx context.Context, raw map[string]interface{}) (map[string]types.Volume, error) {
	result := make(map[string]types.Volume)
	volumesInterface, exists := raw["volumes"]
	if !exists {
		return result, nil
	}

	volumes, ok := volumesInterface.(map[string]interface{})
	if !ok {
		return result, nil
	}

	for volumeName, volumeInterface := range volumes {
		volumeMap, ok := volumeInterface.(map[string]interface{})
		if !ok {
			p.logger.Warn(ctx, "ボリューム設定の形式が無効です",
				types.Field{Key: "volume", Value: volumeName})
			continue
		}

		volume, err := p.convertToVolume(ctx, volumeName, volumeMap)
		if err != nil {
			return nil, fmt.Errorf("ボリューム %s の解析に失敗: %w", volumeName, err)
		}
		result[volumeName] = volume
	}
	return result, nil
}
```

**Step 2: convertToComposeConfig をリファクタリング**

```go
func (p *YamlComposeParser) convertToComposeConfig(ctx context.Context, raw map[string]interface{}, filepath string) (*types.ComposeConfig, error) {
	services, err := p.parseServicesSection(ctx, raw)
	if err != nil {
		return nil, err
	}

	networks, err := p.parseNetworksSection(ctx, raw)
	if err != nil {
		return nil, err
	}

	volumes, err := p.parseVolumesSection(ctx, raw)
	if err != nil {
		return nil, err
	}

	return &types.ComposeConfig{
		Version:  p.extractVersion(raw),
		Services: services,
		Networks: networks,
		Volumes:  volumes,
		FilePath: filepath,
	}, nil
}
```

**Step 3: テストが通ることを確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./internal/parser/ -v`
Expected: ALL PASS

**Step 4: コミット**

```bash
git add internal/parser/yaml.go
git commit -m "ai/refactor(parser): split convertToComposeConfig into section-specific parsers"
```

---

## Task 5: parsePortObject のヘルパー抽出 (SRP)

**Files:**
- Modify: `internal/parser/yaml.go`
- Test: `internal/parser/yaml_test.go` (既存テストで検証)

**現状:** `parsePortObject` (L390-444, 55行) が各フィールドの型変換ロジックを繰り返している。

**分割方針:**

共通の「interface{} から int へ変換」パターンを抽出する。

**Step 1: ヘルパー関数を追加**

```go
// extractIntFromInterface は interface{} から int を抽出します。
// int 型または string 型の数値に対応します。
func extractIntFromInterface(v interface{}, fieldName string) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case string:
		n, err := strconv.Atoi(val)
		if err != nil {
			return 0, &errors.AppError{
				Code:    errors.ErrParseFailed,
				Message: fmt.Sprintf("%sの解析に失敗: %s", fieldName, val),
				Cause:   err,
			}
		}
		return n, nil
	default:
		return 0, nil
	}
}

// extractStringFromMap はマップから文字列値を抽出します。
func extractStringFromMap(m map[string]interface{}, key string) string {
	if v, exists := m[key]; exists {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
```

**Step 2: parsePortObject をリファクタリング**

```go
func (p *YamlComposeParser) parsePortObject(ctx context.Context, portObj map[string]interface{}) (*types.PortMapping, error) {
	mapping := &types.PortMapping{
		Protocol: "tcp",
	}

	if v, exists := portObj["published"]; exists {
		port, err := extractIntFromInterface(v, "publishedポート")
		if err != nil {
			return nil, err
		}
		mapping.Host = port
	}

	if v, exists := portObj["target"]; exists {
		port, err := extractIntFromInterface(v, "targetポート")
		if err != nil {
			return nil, err
		}
		mapping.Container = port
	}

	if protocol := extractStringFromMap(portObj, "protocol"); protocol != "" {
		mapping.Protocol = protocol
	}

	mapping.HostIP = extractStringFromMap(portObj, "host_ip")

	return mapping, nil
}
```

**Step 3: テストが通ることを確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./internal/parser/ -v`
Expected: ALL PASS

**Step 4: コミット**

```bash
git add internal/parser/yaml.go
git commit -m "ai/refactor(parser): extract common type conversion helpers for parsePortObject"
```

---

## Task 6: convertToVolume / convertToNetwork のラベル解析共通化 (DRY)

**Files:**
- Modify: `internal/parser/yaml.go`
- Test: `internal/parser/yaml_test.go` (既存テストで検証)

**現状:** `convertToVolume` と `convertToNetwork` が同じ「map[string]interface{} → map[string]string」変換ロジックを重複して持っている。`convertToVolume` の driver_opts 解析も同じパターン。

**Step 1: 共通ヘルパーを追加**

```go
// extractStringMap はマップ内のキーから map[string]string を抽出します。
func extractStringMap(source map[string]interface{}, key string) map[string]string {
	result := make(map[string]string)
	v, exists := source[key]
	if !exists {
		return result
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return result
	}
	for k, val := range m {
		if s, ok := val.(string); ok {
			result[k] = s
		} else {
			result[k] = fmt.Sprintf("%v", val)
		}
	}
	return result
}
```

**Step 2: convertToNetwork をリファクタリング**

```go
func (p *YamlComposeParser) convertToNetwork(ctx context.Context, name string, networkMap map[string]interface{}) (types.Network, error) {
	network := types.Network{
		Driver: "bridge",
		IPAM: types.IPAM{
			Driver: "default",
			Config: []types.IPAMConfig{},
		},
		Labels: extractStringMap(networkMap, "labels"),
	}

	if driverStr := extractStringFromMap(networkMap, "driver"); driverStr != "" {
		network.Driver = driverStr
	}

	if ipamInterface, exists := networkMap["ipam"]; exists {
		if ipamMap, ok := ipamInterface.(map[string]interface{}); ok {
			ipam, err := p.convertToIPAM(ctx, ipamMap)
			if err != nil {
				return network, fmt.Errorf("IPAM設定の解析に失敗: %w", err)
			}
			network.IPAM = ipam
		}
	}

	return network, nil
}
```

**Step 3: convertToVolume をリファクタリング**

```go
func (p *YamlComposeParser) convertToVolume(ctx context.Context, name string, volumeMap map[string]interface{}) (types.Volume, error) {
	volume := types.Volume{
		Driver:     "local",
		DriverOpts: extractStringMap(volumeMap, "driver_opts"),
		Labels:     extractStringMap(volumeMap, "labels"),
	}

	if driverStr := extractStringFromMap(volumeMap, "driver"); driverStr != "" {
		volume.Driver = driverStr
	}

	return volume, nil
}
```

**Step 4: テストが通ることを確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./internal/parser/ -v`
Expected: ALL PASS

**Step 5: コミット**

```bash
git add internal/parser/yaml.go
git commit -m "ai/refactor(parser): extract common string map conversion to eliminate duplication"
```

---

## Task 7: convertToIPAM のリファクタリング (SRP)

**Files:**
- Modify: `internal/parser/yaml.go`
- Test: `internal/parser/yaml_test.go` (既存テストで検証)

**現状:** `convertToIPAM` (46行) がドライバー抽出と config リスト解析を混在させている。

**Step 1: IPAM config アイテムの解析を分離**

```go
// parseIPAMConfigItem は個別の IPAM config アイテムを解析します。
func parseIPAMConfigItem(configMap map[string]interface{}) types.IPAMConfig {
	return types.IPAMConfig{
		Subnet:  extractStringFromMap(configMap, "subnet"),
		Gateway: extractStringFromMap(configMap, "gateway"),
	}
}
```

**Step 2: convertToIPAM をリファクタリング**

```go
func (p *YamlComposeParser) convertToIPAM(ctx context.Context, ipamMap map[string]interface{}) (types.IPAM, error) {
	ipam := types.IPAM{
		Driver: "default",
		Config: []types.IPAMConfig{},
	}

	if driverStr := extractStringFromMap(ipamMap, "driver"); driverStr != "" {
		ipam.Driver = driverStr
	}

	configList, ok := ipamMap["config"].([]interface{})
	if !ok {
		return ipam, nil
	}

	for _, configItem := range configList {
		configMap, ok := configItem.(map[string]interface{})
		if !ok {
			continue
		}
		ipam.Config = append(ipam.Config, parseIPAMConfigItem(configMap))
	}

	return ipam, nil
}
```

**Step 3: テストが通ることを確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./internal/parser/ -v`
Expected: ALL PASS

**Step 4: コミット**

```bash
git add internal/parser/yaml.go
git commit -m "ai/refactor(parser): simplify convertToIPAM with extracted helper"
```

---

## Task 8: 正規表現のコンパイル済み変数化

**Files:**
- Modify: `internal/parser/yaml.go`

**現状:** `expandVariables` 関数内で毎回3つの正規表現をコンパイルしている (L269, L284, L294)。

**Step 1: パッケージレベル変数に移動**

```go
var (
	// portStringPattern はポート文字列の正規表現パターンです。
	portStringPattern = regexp.MustCompile(`^(?:([\d\.]+):)?(\d+):(\d+)$|^(\d+)$`)

	// varWithDefaultPattern は ${VAR:-default} 形式の環境変数パターンです。
	varWithDefaultPattern = regexp.MustCompile(`\$\{([^}]+):-([^}]*)\}`)

	// varPattern は ${VAR} 形式の環境変数パターンです。
	varPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

	// simpleVarPattern は $VAR 形式の環境変数パターンです。
	simpleVarPattern = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
)
```

**Step 2: expandVariables 内のローカル正規表現を置換**

```go
func expandVariables(input string) string {
	expanded := varWithDefaultPattern.ReplaceAllStringFunc(input, func(match string) string {
		submatches := varWithDefaultPattern.FindStringSubmatch(match)
		if len(submatches) == 3 {
			if value := os.Getenv(submatches[1]); value != "" {
				return value
			}
			return submatches[2]
		}
		return match
	})

	expanded = varPattern.ReplaceAllStringFunc(expanded, func(match string) string {
		submatches := varPattern.FindStringSubmatch(match)
		if len(submatches) == 2 {
			return os.Getenv(submatches[1])
		}
		return match
	})

	expanded = simpleVarPattern.ReplaceAllStringFunc(expanded, func(match string) string {
		submatches := simpleVarPattern.FindStringSubmatch(match)
		if len(submatches) == 2 {
			return os.Getenv(submatches[1])
		}
		return match
	})

	return expanded
}
```

**Step 3: テストが通ることを確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./internal/parser/ -v`
Expected: ALL PASS

**Step 4: コミット**

```bash
git add internal/parser/yaml.go
git commit -m "ai/perf(parser): pre-compile regex patterns as package-level variables"
```

---

## Task 9: 全体テスト + ビルド確認

**Step 1: 全テスト実行**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./... -v`
Expected: ALL PASS

**Step 2: ビルド確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go build ./...`
Expected: SUCCESS

**Step 3: go vet 確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go vet ./...`
Expected: No issues

---

## 期待される成果

| 指標 | Before | After |
|------|--------|-------|
| 総行数 | ~10,830 | ~9,200 |
| resolver パッケージ | 1,336行 | 0行（削除） |
| generator/interfaces.go | 101行 | ~15行 |
| parsePortString | 81行 | ~15行 + ヘルパー |
| convertToComposeConfig | 88行 | ~20行 + セクション解析 |
| parsePortObject | 55行 | ~25行 |
| convertToVolume | 42行 | ~12行 |
| convertToNetwork | 44行 | ~20行 |
| convertToIPAM | 46行 | ~20行 |
| 正規表現コンパイル | 毎回4回 | 起動時1回 |

# 統合テスト拡充 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** gopose の統合テストを全レベルで拡充し、未テストコードのカバレッジ向上とパッケージ間連携の検証を実現する。

**Architecture:** ボトムアップアプローチで、ユニットテスト→パッケージ間統合テスト→Service層E2E→CLI E2Eの順に積み上げる。外部依存はモック中心、ファイルシステム操作はtempdirを使用。

**Tech Stack:** Go, testing パッケージ, os.MkdirTemp, Cobra テスト, 既存の testutil/mocks.go

---

### Task 1: netstat.go の parseNetstatOutput ユニットテスト

**Files:**
- Create: `internal/scanner/netstat_test.go`

**Step 1: テストファイル作成**

```go
package scanner

import (
	"testing"
)

func TestParseNetstatOutput(t *testing.T) {
	detector := &NetstatPortDetector{logger: testLogger()}
	tests := []struct {
		name          string
		input         string
		expectedPorts []int
	}{
		{
			name:          "empty output",
			input:         "",
			expectedPorts: []int{},
		},
		{
			name: "single tcp LISTEN line",
			input: "tcp4       0      0  *.8080                 *.*                    LISTEN",
			expectedPorts: []int{8080},
		},
		{
			name: "multiple LISTEN lines with dedup",
			input: `tcp4       0      0  *.8080                 *.*                    LISTEN
tcp4       0      0  127.0.0.1.3000         *.*                    LISTEN
tcp4       0      0  *.8080                 *.*                    LISTEN`,
			expectedPorts: []int{8080, 3000},
		},
		{
			name: "ignore non-LISTEN lines",
			input: `tcp4       0      0  127.0.0.1.52311        127.0.0.1.3000         ESTABLISHED
tcp4       0      0  *.8080                 *.*                    LISTEN`,
			expectedPorts: []int{8080},
		},
		{
			name: "tcp46 format",
			input: "tcp46      0      0  *.443                  *.*                    LISTEN",
			expectedPorts: []int{443},
		},
		{
			name: "udp LISTEN line",
			input: "udp4       0      0  *.5353                 *.*                    LISTEN",
			expectedPorts: []int{5353},
		},
		{
			name: "no LISTEN lines",
			input: `tcp4       0      0  127.0.0.1.52311        127.0.0.1.3000         ESTABLISHED
tcp4       0      0  127.0.0.1.52312        127.0.0.1.3001         TIME_WAIT`,
			expectedPorts: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ports, err := detector.parseNetstatOutput(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// ポートの数が一致することを確認
			if len(ports) != len(tt.expectedPorts) {
				t.Errorf("got %d ports, want %d", len(ports), len(tt.expectedPorts))
				return
			}
			// 全ての期待ポートが含まれることを確認
			portMap := make(map[int]bool)
			for _, p := range ports {
				portMap[p] = true
			}
			for _, expected := range tt.expectedPorts {
				if !portMap[expected] {
					t.Errorf("expected port %d not found in %v", expected, ports)
				}
			}
		})
	}
}

func testLogger() logger.Logger {
	// testutil を循環インポートしないために同一パッケージ内で定義
	// port_allocator_test.go の既存パターンに合わせて testutil を使用
}
```

注意: `netstat_test.go` は同一パッケージ `scanner` 内なので `parseNetstatOutput` に直接アクセス可能。ロガーは `testutil.NewTestLogger()` を使用する（`port_allocator_test.go` で既にインポート済み）。

**Step 2: テスト実行して確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./internal/scanner/ -run TestParseNetstatOutput -v`
Expected: PASS

**Step 3: コミット**

```bash
git add internal/scanner/netstat_test.go
git commit -m "ai/test: add parseNetstatOutput unit tests for netstat parser"
```

---

### Task 2: netstat.go の PortAllocatorImpl 追加ユニットテスト

**Files:**
- Modify: `internal/scanner/port_allocator_test.go`

既存テストに以下を追加:

**Step 1: エッジケースのテスト追加**

```go
func TestAllocatePort_NoAvailablePorts(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	// 全ポートが使用中
	mockDetector := &mockPortDetector{usedPorts: []int{8000, 8001, 8002}}
	allocator := NewPortAllocatorImpl(mockDetector, testLogger)

	config := types.PortConfig{
		Range:             types.PortRange{Start: 8000, End: 8002},
		ExcludePrivileged: false,
	}

	_, err := allocator.AllocatePort(ctx, config)
	if err == nil {
		t.Fatal("expected error when no ports available")
	}
}

func TestAllocatePort_ExcludePrivileged(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	mockDetector := &mockPortDetector{usedPorts: []int{}}
	allocator := NewPortAllocatorImpl(mockDetector, testLogger)

	config := types.PortConfig{
		Range:             types.PortRange{Start: 80, End: 1100},
		ExcludePrivileged: true,
	}

	port, err := allocator.AllocatePort(ctx, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port <= 1023 {
		t.Errorf("got privileged port %d, expected > 1023", port)
	}
}

func TestAllocatePorts_ZeroCount(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	mockDetector := &mockPortDetector{usedPorts: []int{}}
	allocator := NewPortAllocatorImpl(mockDetector, testLogger)

	config := types.PortConfig{
		Range: types.PortRange{Start: 8000, End: 8100},
	}

	ports, err := allocator.AllocatePorts(ctx, 0, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ports) != 0 {
		t.Errorf("expected empty slice, got %v", ports)
	}
}

func TestAllocatePorts_InsufficientPorts(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	mockDetector := &mockPortDetector{usedPorts: []int{8000, 8001}}
	allocator := NewPortAllocatorImpl(mockDetector, testLogger)

	config := types.PortConfig{
		Range: types.PortRange{Start: 8000, End: 8002},
	}

	// 3ポート中2つが使用中、1つだけ空きだが3つ要求
	_, err := allocator.AllocatePorts(ctx, 3, config)
	if err == nil {
		t.Fatal("expected error when insufficient ports")
	}
}

func TestAllocatePortsForServices_NoPortServices(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	mockDetector := &mockPortDetector{usedPorts: []int{}}
	allocator := NewPortAllocatorImpl(mockDetector, testLogger)

	config := types.PortConfig{
		Range: types.PortRange{Start: 8000, End: 8100},
	}

	services := []types.Service{
		{Name: "worker", Ports: []types.PortMapping{}},
	}

	result, err := allocator.AllocatePortsForServices(ctx, services, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestAllocatePortsForServices_DetectorError(t *testing.T) {
	testLogger := testutil.NewTestLogger()
	ctx := context.Background()

	mockDetector := &mockPortDetector{usedPorts: []int{}}
	// mockPortDetector にエラーを設定するには testutil.MockPortDetector を使う
	mockDetectorWithErr := &testutil.MockPortDetector{Err: fmt.Errorf("detector error")}
	allocator := NewPortAllocatorImpl(mockDetectorWithErr, testLogger)

	config := types.PortConfig{
		Range: types.PortRange{Start: 8000, End: 8100},
	}

	services := []types.Service{
		{Name: "web", Ports: []types.PortMapping{{Host: 80, Container: 80}}},
	}

	_, err := allocator.AllocatePortsForServices(ctx, services, config)
	if err == nil {
		t.Fatal("expected error from detector")
	}
}
```

**Step 2: テスト実行して確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./internal/scanner/ -run "TestAllocate" -v`
Expected: PASS

**Step 3: コミット**

```bash
git add internal/scanner/port_allocator_test.go
git commit -m "ai/test: add PortAllocatorImpl edge case tests"
```

---

### Task 3: errors/handlers.go のユニットテスト

**Files:**
- Create: `internal/errors/handlers_test.go`

**Step 1: テストファイル作成**

```go
package errors

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestAppErrorHandler_Handle_NilError(t *testing.T) {
	h := NewAppErrorHandler()
	result := h.Handle(context.Background(), nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestAppErrorHandler_Handle_AppError(t *testing.T) {
	h := NewAppErrorHandler()
	appErr := &AppError{Code: ErrFileNotFound, Message: "test"}
	result := h.Handle(context.Background(), appErr)
	if result != appErr {
		t.Error("expected same AppError returned")
	}
}

func TestAppErrorHandler_Handle_ConvertStandardErrors(t *testing.T) {
	h := NewAppErrorHandler()
	tests := []struct {
		name         string
		err          error
		expectedCode ErrorCode
	}{
		{"file not exist", os.ErrNotExist, ErrFileNotFound},
		{"permission denied", os.ErrPermission, ErrFilePermission},
		{"connection refused", fmt.Errorf("wrapped: %w", syscall.ECONNREFUSED), ErrDockerAPIFailed},
		{"address in use", fmt.Errorf("wrapped: %w", syscall.EADDRINUSE), ErrPortUnavailable},
		{"unknown error", fmt.Errorf("some unknown error"), ErrUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.Handle(context.Background(), tt.err)
			appErr, ok := result.(*AppError)
			if !ok {
				t.Fatalf("expected *AppError, got %T", result)
			}
			if appErr.Code != tt.expectedCode {
				t.Errorf("expected code %s, got %s", tt.expectedCode, appErr.Code)
			}
		})
	}
}

func TestAppErrorHandler_IsRetryable(t *testing.T) {
	h := NewAppErrorHandler()
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"port unavailable AppError", &AppError{Code: ErrPortUnavailable}, true},
		{"process not found AppError", &AppError{Code: ErrProcessNotFound}, true},
		{"file permission AppError", &AppError{Code: ErrFilePermission}, true},
		{"file not found AppError", &AppError{Code: ErrFileNotFound}, false},
		{"syscall ECONNREFUSED", syscall.ECONNREFUSED, true},
		{"syscall ETIMEDOUT", syscall.ETIMEDOUT, true},
		{"syscall ENOENT", syscall.ENOENT, true},
		{"generic error", fmt.Errorf("generic"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.IsRetryable(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAppErrorHandler_GetRetryConfig(t *testing.T) {
	h := NewAppErrorHandler()
	tests := []struct {
		name               string
		err                error
		expectedMaxRetries int
	}{
		{"port unavailable", &AppError{Code: ErrPortUnavailable}, 5},
		{"port scan failed", &AppError{Code: ErrPortScanFailed}, 5},
		{"process not found", &AppError{Code: ErrProcessNotFound}, 3},
		{"docker api failed", &AppError{Code: ErrDockerAPIFailed}, 3},
		{"unknown AppError", &AppError{Code: ErrUnknown}, 3},         // default
		{"generic error", fmt.Errorf("generic"), 3},                   // default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := h.GetRetryConfig(tt.err)
			if config.MaxRetries != tt.expectedMaxRetries {
				t.Errorf("expected MaxRetries %d, got %d", tt.expectedMaxRetries, config.MaxRetries)
			}
			if config.BaseDelay <= 0 {
				t.Error("expected positive BaseDelay")
			}
			if config.MaxDelay <= 0 {
				t.Error("expected positive MaxDelay")
			}
			if config.BackoffFactor <= 0 {
				t.Error("expected positive BackoffFactor")
			}
		})
	}
}

func TestNewAppErrorHandler_DefaultConfig(t *testing.T) {
	h := NewAppErrorHandler()
	if h.defaultRetryConfig.MaxRetries != 3 {
		t.Errorf("expected default MaxRetries 3, got %d", h.defaultRetryConfig.MaxRetries)
	}
	if h.defaultRetryConfig.BaseDelay != 100*time.Millisecond {
		t.Errorf("expected default BaseDelay 100ms, got %v", h.defaultRetryConfig.BaseDelay)
	}
}

func TestErrorFactoryFunctions(t *testing.T) {
	t.Run("NewFileNotFoundError", func(t *testing.T) {
		err := NewFileNotFoundError("/path/to/file")
		if err.Code != ErrFileNotFound {
			t.Errorf("expected code %s, got %s", ErrFileNotFound, err.Code)
		}
		if err.Fields["path"] != "/path/to/file" {
			t.Errorf("expected path field")
		}
	})

	t.Run("NewPortConflictError", func(t *testing.T) {
		err := NewPortConflictError(8080, "web")
		if err.Code != ErrPortConflict {
			t.Errorf("expected code %s, got %s", ErrPortConflict, err.Code)
		}
		if err.Fields["port"] != 8080 {
			t.Error("expected port field")
		}
		if err.Fields["service"] != "web" {
			t.Error("expected service field")
		}
	})

	t.Run("NewConfigInvalidError", func(t *testing.T) {
		err := NewConfigInvalidError("port_range", "invalid")
		if err.Code != ErrConfigInvalid {
			t.Errorf("expected code %s, got %s", ErrConfigInvalid, err.Code)
		}
	})

	t.Run("NewDockerComposeInvalidError", func(t *testing.T) {
		err := NewDockerComposeInvalidError("/path", "missing services")
		if err.Code != ErrComposeInvalid {
			t.Errorf("expected code %s, got %s", ErrComposeInvalid, err.Code)
		}
	})
}
```

**Step 2: テスト実行して確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./internal/errors/ -run "TestAppErrorHandler|TestNewAppErrorHandler|TestErrorFactory" -v`
Expected: PASS

**Step 3: コミット**

```bash
git add internal/errors/handlers_test.go
git commit -m "ai/test: add error handler and factory function unit tests"
```

---

### Task 4: cmd/up.go の createPortConfig 追加テスト

**Files:**
- Modify: `cmd/up_test.go`

`detectWorktreeProjectName` は git コマンド依存のため、既存テストパターンに合わせて `createPortConfig` の追加エッジケースに集中する。

**Step 1: エッジケーステスト追加**

```go
func TestCreatePortConfig_PortRangeEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		portRange   string
		expectError bool
	}{
		{"three segments", "8000-8500-9000", true},
		{"negative port", "-1-9000", true},
		{"port 65535", "65000-65535", false},
		{"single port range", "8000-8000", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := types.PortConfig{
				Range:    types.PortRange{Start: 8000, End: 9000},
				Reserved: []int{},
			}
			_, err := createPortConfig(tt.portRange, base)
			if tt.expectError && err == nil {
				t.Error("expected error")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
```

**Step 2: テスト実行して確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./cmd/ -run TestCreatePortConfig -v`
Expected: PASS

**Step 3: コミット**

```bash
git add cmd/up_test.go
git commit -m "ai/test: add createPortConfig edge case tests"
```

---

### Task 5: テストデータ追加

**Files:**
- Create: `testdata/integration/compose-multi-service.yml`
- Create: `testdata/integration/compose-network-conflict.yml`
- Create: `testdata/integration/compose-no-ports.yml`

**Step 1: テストデータファイル作成**

`compose-multi-service.yml`:
```yaml
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
  api:
    image: node:alpine
    ports:
      - "3000:3000"
  db:
    image: postgres:13
    ports:
      - "5432:5432"
    environment:
      POSTGRES_PASSWORD: password
  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
```

`compose-network-conflict.yml`:
```yaml
services:
  web:
    image: nginx:alpine
    ports:
      - "8080:80"
    networks:
      - frontend
  api:
    image: node:alpine
    ports:
      - "3000:3000"
    networks:
      - backend

networks:
  frontend:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
  backend:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

`compose-no-ports.yml`:
```yaml
services:
  worker:
    image: python:3.11
    command: python worker.py
  scheduler:
    image: python:3.11
    command: python scheduler.py
```

**Step 2: コミット**

```bash
git add testdata/integration/
git commit -m "ai/test: add integration test data files"
```

---

### Task 6: パッケージ間統合テスト（parser→scanner→generator パイプライン）

**Files:**
- Create: `internal/integration/pipeline_test.go`

**Step 1: テストファイル作成**

`pipeline_test.go` では実際の parser, scanner (DetectorはMock), generator を結合してテストする。

```go
package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/harakeishi/gopose/internal/generator"
	"github.com/harakeishi/gopose/internal/parser"
	"github.com/harakeishi/gopose/internal/scanner"
	"github.com/harakeishi/gopose/internal/testutil"
	"github.com/harakeishi/gopose/pkg/types"
)

func TestPipeline_ParseToOverride_MultiService(t *testing.T) {
	ctx := context.Background()
	log := testutil.NewTestLogger()

	// テストデータのcompose.ymlをパース
	composeParser := parser.NewYamlComposeParser(log)
	config, err := composeParser.ParseComposeFile(ctx, "../../testdata/integration/compose-multi-service.yml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(config.Services) != 4 {
		t.Fatalf("expected 4 services, got %d", len(config.Services))
	}

	// ポート8080,3000が使用中のMockDetector
	mockDetector := &testutil.MockPortDetector{UsedPorts: []int{8080, 3000}}
	conflictDetector := scanner.NewUnifiedConflictDetectorImpl(mockDetector, nil, log)

	conflictInfo, err := conflictDetector.DetectPortConflicts(ctx, config)
	if err != nil {
		t.Fatalf("detect error: %v", err)
	}

	// 8080, 3000の2つのポート衝突が検出されるはず
	if len(conflictInfo) < 2 {
		t.Errorf("expected at least 2 port conflicts, got %d", len(conflictInfo))
	}
}

func TestPipeline_ParseToOverride_NoConflict(t *testing.T) {
	ctx := context.Background()
	log := testutil.NewTestLogger()

	composeParser := parser.NewYamlComposeParser(log)
	config, err := composeParser.ParseComposeFile(ctx, "../../testdata/integration/compose-multi-service.yml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// 使用中ポートなし
	mockDetector := &testutil.MockPortDetector{UsedPorts: []int{}}
	conflictDetector := scanner.NewUnifiedConflictDetectorImpl(mockDetector, nil, log)

	conflictInfo, err := conflictDetector.DetectPortConflicts(ctx, config)
	if err != nil {
		t.Fatalf("detect error: %v", err)
	}

	if len(conflictInfo) != 0 {
		t.Errorf("expected 0 port conflicts, got %d", len(conflictInfo))
	}
}

func TestPipeline_ParseToOverride_NoPorts(t *testing.T) {
	ctx := context.Background()
	log := testutil.NewTestLogger()

	composeParser := parser.NewYamlComposeParser(log)
	config, err := composeParser.ParseComposeFile(ctx, "../../testdata/integration/compose-no-ports.yml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if len(config.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(config.Services))
	}

	// ポートなしサービスは衝突ゼロ
	mockDetector := &testutil.MockPortDetector{UsedPorts: []int{8080}}
	conflictDetector := scanner.NewUnifiedConflictDetectorImpl(mockDetector, nil, log)

	conflictInfo, err := conflictDetector.DetectPortConflicts(ctx, config)
	if err != nil {
		t.Fatalf("detect error: %v", err)
	}

	if len(conflictInfo) != 0 {
		t.Errorf("expected 0 conflicts for no-port services, got %d", len(conflictInfo))
	}
}

func TestPipeline_FullResolve_WritesOverride(t *testing.T) {
	ctx := context.Background()
	log := testutil.NewTestLogger()

	// パース
	composeParser := parser.NewYamlComposeParser(log)
	config, err := composeParser.ParseComposeFile(ctx, "../../testdata/integration/compose-multi-service.yml")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// 衝突検知（8080使用中）
	mockDetector := &testutil.MockPortDetector{UsedPorts: []int{8080}}
	portAllocator := scanner.NewPortAllocatorImpl(mockDetector, log)
	overrideGen := generator.NewUnifiedOverrideGeneratorImpl(portAllocator, log)
	conflictDetector := scanner.NewUnifiedConflictDetectorImpl(mockDetector, nil, log)

	unifiedInfo, err := conflictDetector.DetectConflicts(ctx, config, "test-project")
	if err != nil {
		t.Fatalf("detect error: %v", err)
	}

	if !unifiedInfo.HasConflicts() {
		t.Skip("no conflicts detected, skipping resolve test")
	}

	// 解決
	portConfig := types.PortConfig{
		Range: types.PortRange{Start: 9000, End: 9999},
	}
	if err := overrideGen.ResolveConflicts(ctx, unifiedInfo, types.ResolutionStrategyAutoIncrement, portConfig); err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	// Override生成
	override, err := overrideGen.GenerateFromConflicts(ctx, config, unifiedInfo)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}

	// Override書き込み
	writer := generator.NewOverrideGeneratorImpl(log)
	if err := writer.ValidateOverride(ctx, override); err != nil {
		t.Fatalf("validate error: %v", err)
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "compose.override.yml")
	if err := writer.WriteOverrideFile(ctx, override, outputPath); err != nil {
		t.Fatalf("write error: %v", err)
	}

	// ファイルが存在することを確認
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("override file was not created")
	}

	// ファイル内容が空でないことを確認
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if len(data) == 0 {
		t.Error("override file is empty")
	}
}
```

**Step 2: テスト実行して確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./internal/integration/ -v`
Expected: PASS

**Step 3: コミット**

```bash
git add internal/integration/pipeline_test.go
git commit -m "ai/test: add package integration pipeline tests"
```

---

### Task 7: Service層 E2E テスト

**Files:**
- Create: `internal/integration/service_e2e_test.go`

**Step 1: テストファイル作成**

```go
package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/harakeishi/gopose/internal/generator"
	"github.com/harakeishi/gopose/internal/parser"
	"github.com/harakeishi/gopose/internal/scanner"
	"github.com/harakeishi/gopose/internal/service"
	"github.com/harakeishi/gopose/internal/testutil"
	"github.com/harakeishi/gopose/pkg/types"
)

// buildRealDeps は実際のコンポーネントを使用したUpServiceDepsを構築する。
// PortDetector のみモック。
func buildRealDeps(usedPorts []int) service.UpServiceDeps {
	log := testutil.NewTestLogger()
	mockDetector := &testutil.MockPortDetector{UsedPorts: usedPorts}
	portAllocator := scanner.NewPortAllocatorImpl(mockDetector, log)

	return service.UpServiceDeps{
		Logger:              log,
		ComposeFileDetector: parser.NewComposeFileDetectorImpl(log),
		ComposeParser:       parser.NewYamlComposeParser(log),
		ConflictDetector:    scanner.NewUnifiedConflictDetectorImpl(mockDetector, nil, log),
		OverrideGenerator:   generator.NewUnifiedOverrideGeneratorImpl(portAllocator, log),
		OverrideWriter:      generator.NewOverrideGeneratorImpl(log),
	}
}

func TestServiceE2E_NoConflicts(t *testing.T) {
	ctx := context.Background()

	// tempdirにcompose.ymlを配置
	tmpDir := t.TempDir()
	composeContent, err := os.ReadFile("../../testdata/integration/compose-multi-service.yml")
	if err != nil {
		t.Fatalf("read testdata error: %v", err)
	}
	composePath := filepath.Join(tmpDir, "compose.yml")
	if err := os.WriteFile(composePath, composeContent, 0644); err != nil {
		t.Fatalf("write compose error: %v", err)
	}

	deps := buildRealDeps([]int{}) // ポート使用なし
	svc := service.NewUpService(deps)

	result, err := svc.Execute(ctx, service.UpParams{
		ComposeFilePath: composePath,
		WorkDir:         tmpDir,
		OutputFile:      filepath.Join(tmpDir, "compose.override.yml"),
		Strategy:        "auto",
		PortConfig:      types.PortConfig{Range: types.PortRange{Start: 9000, End: 9999}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.HasConflicts {
		t.Error("expected no conflicts")
	}

	// overrideファイルが生成されていないことを確認
	overridePath := filepath.Join(tmpDir, "compose.override.yml")
	if _, err := os.Stat(overridePath); !os.IsNotExist(err) {
		t.Error("override file should not exist when no conflicts")
	}
}

func TestServiceE2E_WithConflicts_WritesFile(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	composeContent, err := os.ReadFile("../../testdata/integration/compose-multi-service.yml")
	if err != nil {
		t.Fatalf("read testdata error: %v", err)
	}
	composePath := filepath.Join(tmpDir, "compose.yml")
	if err := os.WriteFile(composePath, composeContent, 0644); err != nil {
		t.Fatalf("write compose error: %v", err)
	}

	deps := buildRealDeps([]int{8080, 3000}) // ポート8080,3000使用中
	svc := service.NewUpService(deps)

	overridePath := filepath.Join(tmpDir, "compose.override.yml")
	result, err := svc.Execute(ctx, service.UpParams{
		ComposeFilePath: composePath,
		WorkDir:         tmpDir,
		OutputFile:      overridePath,
		Strategy:        "auto",
		PortConfig:      types.PortConfig{Range: types.PortRange{Start: 9000, End: 9999}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasConflicts {
		t.Fatal("expected conflicts")
	}

	// overrideファイルが存在することを確認
	if _, err := os.Stat(overridePath); os.IsNotExist(err) {
		t.Fatal("override file was not created")
	}

	data, err := os.ReadFile(overridePath)
	if err != nil {
		t.Fatalf("read override error: %v", err)
	}
	if len(data) == 0 {
		t.Error("override file is empty")
	}
}

func TestServiceE2E_DryRun_NoFileWritten(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	composeContent, err := os.ReadFile("../../testdata/integration/compose-multi-service.yml")
	if err != nil {
		t.Fatalf("read testdata error: %v", err)
	}
	composePath := filepath.Join(tmpDir, "compose.yml")
	if err := os.WriteFile(composePath, composeContent, 0644); err != nil {
		t.Fatalf("write compose error: %v", err)
	}

	deps := buildRealDeps([]int{8080})
	svc := service.NewUpService(deps)

	overridePath := filepath.Join(tmpDir, "compose.override.yml")
	result, err := svc.Execute(ctx, service.UpParams{
		ComposeFilePath: composePath,
		WorkDir:         tmpDir,
		OutputFile:      overridePath,
		Strategy:        "auto",
		PortConfig:      types.PortConfig{Range: types.PortRange{Start: 9000, End: 9999}},
		DryRun:          true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.HasConflicts {
		t.Fatal("expected conflicts")
	}

	// dry-runなのでファイルは生成されない
	if _, err := os.Stat(overridePath); !os.IsNotExist(err) {
		t.Error("override file should not exist in dry-run mode")
	}
}

func TestServiceE2E_AutoDetect_ComposeFile(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	composeContent, err := os.ReadFile("../../testdata/integration/compose-no-ports.yml")
	if err != nil {
		t.Fatalf("read testdata error: %v", err)
	}
	// compose.yml として配置（自動検出させる）
	composePath := filepath.Join(tmpDir, "compose.yml")
	if err := os.WriteFile(composePath, composeContent, 0644); err != nil {
		t.Fatalf("write compose error: %v", err)
	}

	deps := buildRealDeps([]int{})
	svc := service.NewUpService(deps)

	result, err := svc.Execute(ctx, service.UpParams{
		ComposeFilePath: "", // 自動検出
		WorkDir:         tmpDir,
		Strategy:        "auto",
		PortConfig:      types.PortConfig{Range: types.PortRange{Start: 9000, End: 9999}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.HasConflicts {
		t.Error("expected no conflicts for no-port services")
	}
}

func TestServiceE2E_ProjectName(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	composeContent, err := os.ReadFile("../../testdata/integration/compose-multi-service.yml")
	if err != nil {
		t.Fatalf("read testdata error: %v", err)
	}
	composePath := filepath.Join(tmpDir, "compose.yml")
	if err := os.WriteFile(composePath, composeContent, 0644); err != nil {
		t.Fatalf("write compose error: %v", err)
	}

	deps := buildRealDeps([]int{8080})
	svc := service.NewUpService(deps)

	overridePath := filepath.Join(tmpDir, "compose.override.yml")
	result, err := svc.Execute(ctx, service.UpParams{
		ComposeFilePath: composePath,
		WorkDir:         tmpDir,
		OutputFile:      overridePath,
		ProjectName:     "my-project",
		Strategy:        "auto",
		PortConfig:      types.PortConfig{Range: types.PortRange{Start: 9000, End: 9999}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Override == nil {
		t.Fatal("expected override")
	}
	if result.Override.Name != "my-project" {
		t.Errorf("expected project name 'my-project', got '%s'", result.Override.Name)
	}
}
```

**Step 2: テスト実行して確認**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./internal/integration/ -run "TestServiceE2E" -v`
Expected: PASS

**Step 3: コミット**

```bash
git add internal/integration/service_e2e_test.go
git commit -m "ai/test: add Service layer E2E tests with real components"
```

---

### Task 8: 全テスト実行確認

**Step 1: 全テスト実行**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./... -v -count=1`
Expected: ALL PASS

**Step 2: テストがfailした場合は修正**

**Step 3: 最終コミット（必要に応じて）**

```bash
git add .
git commit -m "ai/test: fix integration test issues"
```

# Output Styling Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace gopose's verbose CLI output with minimal, clean table-formatted conflict resolution summaries.

**Architecture:** New `internal/presenter/` package with `Presenter` interface and `TablePresenter` implementation using `text/tabwriter`. `cmd/up.go` uses Presenter for user-facing output, Logger for structured logs only.

**Tech Stack:** Go stdlib (`text/tabwriter`, `fmt`, `io`)

---

### Task 1: Create Presenter Interface

**Files:**
- Create: `internal/presenter/interfaces.go`

**Step 1: Write interface file**

```go
package presenter

import "github.com/harakeishi/gopose/pkg/types"

// Presenter is the interface for user-facing CLI output.
type Presenter interface {
	Progress(message string)
	PortConflicts(conflicts []types.PortConflictInfo)
	NetworkConflicts(conflicts []types.NetworkConflictInfo)
	Result(message string)
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go build ./internal/presenter/...`
Expected: No errors

**Step 3: Commit**

```bash
git add internal/presenter/interfaces.go
git commit -m "ai/feat: add Presenter interface for CLI output"
```

---

### Task 2: Write Failing Tests for TablePresenter

**Files:**
- Create: `internal/presenter/table_test.go`

**Step 1: Write tests**

```go
package presenter

import (
	"bytes"
	"strings"
	"testing"

	"github.com/harakeishi/gopose/pkg/types"
)

func TestTablePresenter_Progress(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	p.Progress("Scanning...")

	got := buf.String()
	if got != "Scanning...\n" {
		t.Errorf("Progress() = %q, want %q", got, "Scanning...\n")
	}
}

func TestTablePresenter_PortConflicts_WithConflicts(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	conflicts := []types.PortConflictInfo{
		{
			ServiceName: "web",
			Port:        3000,
			Resolution:  &types.PortResolutionInfo{ResolvedPort: 8001},
		},
		{
			ServiceName: "api",
			Port:        5432,
			Resolution:  &types.PortResolutionInfo{ResolvedPort: 5433},
		},
	}

	p.PortConflicts(conflicts)

	got := buf.String()
	if !strings.Contains(got, "Port Conflicts:") {
		t.Error("expected header 'Port Conflicts:'")
	}
	if !strings.Contains(got, "SERVICE") {
		t.Error("expected column header 'SERVICE'")
	}
	if !strings.Contains(got, "web") {
		t.Error("expected service name 'web'")
	}
	if !strings.Contains(got, "3000") {
		t.Error("expected port '3000'")
	}
	if !strings.Contains(got, "8001") {
		t.Error("expected resolved port '8001'")
	}
	if !strings.Contains(got, "api") {
		t.Error("expected service name 'api'")
	}
}

func TestTablePresenter_PortConflicts_Empty(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	p.PortConflicts(nil)

	got := buf.String()
	if !strings.Contains(got, "Port Conflicts:") {
		t.Error("expected header 'Port Conflicts:'")
	}
	if !strings.Contains(got, "(none)") {
		t.Error("expected '(none)' for empty conflicts")
	}
}

func TestTablePresenter_NetworkConflicts_WithConflicts(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	conflicts := []types.NetworkConflictInfo{
		{
			NetworkName:    "default",
			OriginalSubnet: "172.20.0.0/24",
			Resolution:     &types.NetworkResolutionInfo{ResolvedSubnet: "10.20.0.0/24"},
		},
	}

	p.NetworkConflicts(conflicts)

	got := buf.String()
	if !strings.Contains(got, "Network Conflicts:") {
		t.Error("expected header 'Network Conflicts:'")
	}
	if !strings.Contains(got, "default") {
		t.Error("expected network name 'default'")
	}
	if !strings.Contains(got, "172.20.0.0/24") {
		t.Error("expected original subnet")
	}
	if !strings.Contains(got, "10.20.0.0/24") {
		t.Error("expected resolved subnet")
	}
}

func TestTablePresenter_NetworkConflicts_Empty(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	p.NetworkConflicts(nil)

	got := buf.String()
	if !strings.Contains(got, "Network Conflicts:") {
		t.Error("expected header 'Network Conflicts:'")
	}
	if !strings.Contains(got, "(none)") {
		t.Error("expected '(none)' for empty conflicts")
	}
}

func TestTablePresenter_Result(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	p.Result("Generated: compose.override.yml")

	got := buf.String()
	if got != "Generated: compose.override.yml\n" {
		t.Errorf("Result() = %q, want %q", got, "Generated: compose.override.yml\n")
	}
}

func TestTablePresenter_PortConflicts_NoResolution(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	conflicts := []types.PortConflictInfo{
		{
			ServiceName: "web",
			Port:        3000,
			Resolution:  nil,
		},
	}

	p.PortConflicts(conflicts)

	got := buf.String()
	// Conflicts without resolution should be skipped
	if strings.Contains(got, "web") {
		t.Error("conflicts without resolution should not appear in table")
	}
}

func TestTablePresenter_FullWorkflow(t *testing.T) {
	var buf bytes.Buffer
	p := NewTablePresenter(&buf)

	p.Progress("Scanning...")
	p.Progress("Resolving...")
	p.PortConflicts([]types.PortConflictInfo{
		{
			ServiceName: "web",
			Port:        3000,
			Resolution:  &types.PortResolutionInfo{ResolvedPort: 8001},
		},
	})
	p.NetworkConflicts(nil)
	p.Result("Generated: compose.override.yml")

	got := buf.String()
	// Verify ordering: progress -> tables -> result
	scanIdx := strings.Index(got, "Scanning...")
	resolveIdx := strings.Index(got, "Resolving...")
	portIdx := strings.Index(got, "Port Conflicts:")
	netIdx := strings.Index(got, "Network Conflicts:")
	genIdx := strings.Index(got, "Generated:")

	if scanIdx >= resolveIdx {
		t.Error("Scanning should come before Resolving")
	}
	if resolveIdx >= portIdx {
		t.Error("Resolving should come before Port Conflicts")
	}
	if portIdx >= netIdx {
		t.Error("Port Conflicts should come before Network Conflicts")
	}
	if netIdx >= genIdx {
		t.Error("Network Conflicts should come before Generated")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./internal/presenter/...`
Expected: FAIL (NewTablePresenter not defined)

**Step 3: Commit**

```bash
git add internal/presenter/table_test.go
git commit -m "ai/test: add TablePresenter tests"
```

---

### Task 3: Implement TablePresenter

**Files:**
- Create: `internal/presenter/table.go`

**Step 1: Write implementation**

```go
package presenter

import (
	"fmt"
	"io"
	"strconv"
	"text/tabwriter"

	"github.com/harakeishi/gopose/pkg/types"
)

// TablePresenter formats CLI output as aligned plain-text tables.
type TablePresenter struct {
	w io.Writer
}

// NewTablePresenter creates a new TablePresenter writing to w.
func NewTablePresenter(w io.Writer) *TablePresenter {
	return &TablePresenter{w: w}
}

func (p *TablePresenter) Progress(message string) {
	fmt.Fprintln(p.w, message)
}

func (p *TablePresenter) PortConflicts(conflicts []types.PortConflictInfo) {
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Port Conflicts:")

	// Filter to only resolved conflicts
	var resolved []types.PortConflictInfo
	for _, c := range conflicts {
		if c.Resolution != nil {
			resolved = append(resolved, c)
		}
	}

	if len(resolved) == 0 {
		fmt.Fprintln(p.w, "  (none)")
		return
	}

	tw := tabwriter.NewWriter(p.w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "  SERVICE\tFROM\tTO")
	for _, c := range resolved {
		fmt.Fprintf(tw, "  %s\t%s\t%s\n",
			c.ServiceName,
			strconv.Itoa(c.Port),
			strconv.Itoa(c.Resolution.ResolvedPort),
		)
	}
	tw.Flush()
}

func (p *TablePresenter) NetworkConflicts(conflicts []types.NetworkConflictInfo) {
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, "Network Conflicts:")

	// Filter to only resolved conflicts
	var resolved []types.NetworkConflictInfo
	for _, c := range conflicts {
		if c.Resolution != nil {
			resolved = append(resolved, c)
		}
	}

	if len(resolved) == 0 {
		fmt.Fprintln(p.w, "  (none)")
		return
	}

	tw := tabwriter.NewWriter(p.w, 0, 0, 3, ' ', 0)
	fmt.Fprintln(tw, "  NETWORK\tFROM\tTO")
	for _, c := range resolved {
		fmt.Fprintf(tw, "  %s\t%s\t%s\n",
			c.NetworkName,
			c.OriginalSubnet,
			c.Resolution.ResolvedSubnet,
		)
	}
	tw.Flush()
}

func (p *TablePresenter) Result(message string) {
	fmt.Fprintln(p.w)
	fmt.Fprintln(p.w, message)
}
```

**Step 2: Run tests to verify they pass**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./internal/presenter/... -v`
Expected: All PASS

**Step 3: Commit**

```bash
git add internal/presenter/table.go
git commit -m "ai/feat: implement TablePresenter with tabwriter"
```

---

### Task 4: Integrate Presenter into cmd/up.go

**Files:**
- Modify: `cmd/up.go`
- Modify: `cmd/root.go` (add presenter helper)

**Step 1: Add `getPresenter` helper to `cmd/root.go`**

Add after the `getLogger` function (~line 118):

```go
// getPresenter はプレゼンターを取得します。
func getPresenter() presenter.Presenter {
	return presenter.NewTablePresenter(os.Stdout)
}
```

Add import `"github.com/harakeishi/gopose/internal/presenter"` to root.go.

**Step 2: Rewrite `cmd/up.go` RunE to use Presenter**

Replace the `RunE` function body. Key changes:
- Add `pres := getPresenter()` at the start
- Replace `logger.Info(ctx, "ポート衝突解決を開始", ...)` with `pres.Progress("Scanning...")`
- Replace the per-conflict `logger.Info` loop (lines 238-256) with `pres.PortConflicts(...)` and `pres.NetworkConflicts(...)`
- Replace final log messages with `pres.Result("Generated: " + outputFile)` or `pres.Result("Dry run: no files written.")` or `pres.Result("No conflicts detected.")`
- Keep `logger.Info/Debug/Warn` calls that are meaningful for `--detail` mode (e.g. worktree detection, file auto-detection)
- Remove redundant messages: "override.ymlの生成が完了しました。docker compose upを実行する場合は、手動で実行してください。"

The new RunE:

```go
RunE: func(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()
    cfg := getConfig()
    pres := getPresenter()

    logger, err := getLogger(cfg)
    if err != nil {
        return fmt.Errorf("ロガーの初期化に失敗しました: %w", err)
    }

    portConfig, err := createPortConfig(portRange, cfg.GetPort())
    if err != nil {
        return fmt.Errorf("ポート範囲の解析に失敗しました: %w", err)
    }

    if composeProjectName == "" && os.Getenv("COMPOSE_PROJECT_NAME") == "" {
        if pn, err := detectWorktreeProjectName(); err == nil && pn != "" {
            composeProjectName = pn
            logger.Debug(ctx, "ワークツリー名をプロジェクト名として使用",
                types.Field{Key: "project_name", Value: composeProjectName})
        }
    }

    logger.Debug(ctx, "ポート衝突解決を開始",
        types.Field{Key: "dry_run", Value: dryRun},
        types.Field{Key: "compose_file", Value: filePath},
        types.Field{Key: "output_file", Value: outputFile},
        types.Field{Key: "project_name", Value: composeProjectName},
        types.Field{Key: "strategy", Value: strategy},
        types.Field{Key: "port_range", Value: fmt.Sprintf("%d-%d", portConfig.Range.Start, portConfig.Range.End)},
        types.Field{Key: "reserved_ports", Value: portConfig.Reserved})

    // Docker Composeファイルの自動検出
    if filePath == "" || filePath == "compose.yml" {
        wd, err := os.Getwd()
        if err != nil {
            return fmt.Errorf("作業ディレクトリの取得に失敗: %w", err)
        }
        detector := parser.NewComposeFileDetectorImpl(logger)
        detectedFile, err := detector.GetDefaultComposeFile(ctx, wd)
        if err != nil {
            return fmt.Errorf("docker composeファイルの自動検出に失敗: %w", err)
        }
        filePath = detectedFile
        logger.Debug(ctx, "Docker Composeファイルを自動検出", types.Field{Key: "file", Value: filePath})
    }

    // Docker Composeファイルの解析
    yamlParser := parser.NewYamlComposeParser(logger)
    config, err := yamlParser.ParseComposeFile(ctx, filePath)
    if err != nil {
        return fmt.Errorf("docker composeファイルの解析に失敗: %w", err)
    }

    pres.Progress("Scanning...")

    // 統一的な衝突検知
    portDetector := scanner.NewNetstatPortDetector(logger)
    portAllocator := scanner.NewPortAllocatorImpl(portDetector, logger)
    networkDetector := scanner.NewDockerNetworkDetector(logger)
    unifiedDetector := scanner.NewUnifiedConflictDetectorImpl(portDetector, networkDetector, logger)

    conflictInfo, err := unifiedDetector.DetectConflicts(ctx, config, composeProjectName)
    if err != nil {
        return fmt.Errorf("衝突検知に失敗: %w", err)
    }

    // 衝突がない場合
    if !conflictInfo.HasConflicts() {
        pres.PortConflicts(nil)
        pres.NetworkConflicts(nil)
        pres.Result("No conflicts detected.")
        return nil
    }

    pres.Progress("Resolving...")

    logger.Debug(ctx, "衝突検知完了",
        types.Field{Key: "port_conflicts", Value: len(conflictInfo.PortConflicts)},
        types.Field{Key: "network_conflicts", Value: len(conflictInfo.NetworkConflicts)})

    // 解決戦略の決定
    resolutionStrategy := types.ResolutionStrategyAutoIncrement
    switch strategy {
    case "auto":
        resolutionStrategy = types.ResolutionStrategyAutoIncrement
    case "range":
        resolutionStrategy = types.ResolutionStrategyRangeAllocation
    case "user":
        resolutionStrategy = types.ResolutionStrategyUserDefined
    }

    // 衝突解決
    unifiedGenerator := generator.NewUnifiedOverrideGeneratorImpl(portAllocator, logger)
    if err := unifiedGenerator.ResolveConflicts(ctx, conflictInfo, resolutionStrategy, portConfig); err != nil {
        return fmt.Errorf("衝突解決に失敗: %w", err)
    }

    // テーブル表示
    pres.PortConflicts(conflictInfo.PortConflicts)
    pres.NetworkConflicts(conflictInfo.NetworkConflicts)

    // Override.yml生成
    override, err := unifiedGenerator.GenerateFromConflicts(ctx, config, conflictInfo)
    if err != nil {
        return fmt.Errorf("overrideファイルの生成に失敗: %w", err)
    }

    if composeProjectName != "" {
        override.Name = composeProjectName
    }

    overrideGenerator := generator.NewOverrideGeneratorImpl(logger)
    if err := overrideGenerator.ValidateOverride(ctx, override); err != nil {
        return fmt.Errorf("overrideファイルの検証に失敗: %w", err)
    }

    if outputFile == "" {
        outputFile = "compose.override.yml"
    }

    if dryRun {
        pres.Result("Dry run: no files written.")
        return nil
    }

    if err := overrideGenerator.WriteOverrideFile(ctx, override, outputFile); err != nil {
        return fmt.Errorf("overrideファイルの書き込みに失敗: %w", err)
    }

    pres.Result("Generated: " + outputFile)
    return nil
},
```

**Step 3: Verify build**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go build ./...`
Expected: No errors

**Step 4: Run all tests**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./...`
Expected: All PASS

**Step 5: Commit**

```bash
git add cmd/up.go cmd/root.go
git commit -m "ai/feat: integrate Presenter into up command for clean output"
```

---

### Task 5: Update Existing Tests

**Files:**
- Modify: `cmd/up_test.go` (if tests reference old log messages)

**Step 1: Check existing test expectations**

Read `cmd/up_test.go` and verify no tests assert on removed log messages (e.g. "ポート衝突解決を開始").

**Step 2: Fix any broken assertions**

Update test expectations to match new output format if needed.

**Step 3: Run all tests**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./... -v`
Expected: All PASS

**Step 4: Commit (if changes were needed)**

```bash
git add cmd/up_test.go
git commit -m "ai/test: update up command tests for new output format"
```

---

### Task 6: Final Verification

**Step 1: Run full test suite**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go test ./... -count=1`
Expected: All PASS

**Step 2: Build binary and manual smoke test**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go build -o gopose .`
Expected: Binary builds without errors

**Step 3: Verify go vet**

Run: `cd /Users/keishi.hara/src/github.com/harakeishi/gopose && go vet ./...`
Expected: No issues

# CLI Output Styling Design

## Summary

gopose の CLI 出力をミニマル＆クリーンにリデザインする。不要な途中経過を削り、衝突回避結果をテーブル形式で見やすく表示する。

## Requirements

- 途中経過は簡潔な1行ずつ（`Scanning...`, `Resolving...`）
- 衝突回避結果はテーブル形式で表示
- 衝突なし時はテーブルヘッダー + `(none)` で明示
- カラー出力は使わない（プレーンテキストのみ）
- `--detail` 時の構造化ログ出力は従来通り維持

## Output Format

### Normal (conflicts found)

```
Scanning...
Resolving...

Port Conflicts:
  SERVICE   FROM   TO
  web       3000   8001
  api       5432   5433

Network Conflicts:
  NETWORK   FROM              TO
  default   172.20.0.0/24     10.20.0.0/24

Generated: compose.override.yml
```

### No conflicts

```
Scanning...

Port Conflicts:
  (none)

Network Conflicts:
  (none)

No conflicts detected.
```

### Dry run

```
Scanning...
Resolving...

Port Conflicts:
  SERVICE   FROM   TO
  web       3000   8001

Network Conflicts:
  (none)

Dry run: no files written.
```

## Architecture

### New Component: `internal/presenter/`

```
internal/presenter/
  interfaces.go      -- Presenter interface
  table.go           -- TablePresenter implementation
  table_test.go      -- Tests
```

### SOLID Compliance

- **S (Single Responsibility)**: Presenter handles user-facing output formatting only. Logger handles structured logging. cmd/up.go handles workflow control.
- **O (Open/Closed)**: Presenter is an interface. New output formats (JSON, color) can be added as new implementations without modifying existing code.
- **L (Liskov Substitution)**: All Presenter implementations (TablePresenter, NopPresenter) satisfy the same contract.
- **I (Interface Segregation)**: Presenter is a small, focused interface separate from Logger.
- **D (Dependency Inversion)**: cmd/up.go depends on the Presenter interface, not concrete implementations. TablePresenter depends on io.Writer for testability.

### Interface Design

```go
// internal/presenter/interfaces.go
type Presenter interface {
    Progress(message string)
    PortConflicts(conflicts []types.PortConflictInfo)
    NetworkConflicts(conflicts []types.NetworkConflictInfo)
    Result(message string)
}
```

### Implementation

```go
// internal/presenter/table.go
type TablePresenter struct {
    w io.Writer
}

func NewTablePresenter(w io.Writer) *TablePresenter
```

- Uses `text/tabwriter` (stdlib) for column alignment
- No external dependencies
- `io.Writer` injection for testability

### Changes to cmd/up.go

- When `detailed=false`: use Presenter for all user-facing output, suppress logger output for those messages
- When `detailed=true`: use Logger for structured logging as before (Presenter still used for table output)
- Remove redundant log messages (e.g. "ポート衝突解決を開始", "override.ymlの生成が完了しました")

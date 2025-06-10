# gopose - Docker Compose ポート衝突自動解決ツール

<div align="center">
  <img src="logo.png" alt="gopose logo" width="200"/>
  
  [![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
  [![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
  [![Go Report Card](https://goreportcard.com/badge/github.com/harakeishi/gopose?style=for-the-badge)](https://goreportcard.com/report/github.com/harakeishi/gopose)
</div>

## 概要

**gopose** (Go Port Override Solution Engine) は、Docker Compose のポートバインディング衝突を自動検出・解決するツールです。

元の `docker-compose.yml` を変更せずに `docker-compose.override.yml` を生成し、ポート衝突解決後、自動的に `override.yml` を削除します。

### 🎯 主な特徴

- ✅ **非破壊的**: 元の `docker-compose.yml` ファイルを変更しません
- ✅ **自動検出**: システムの使用中ポートとの衝突を自動検出
- ✅ **自動解決**: 利用可能なポートを自動割り当て
- ✅ **自動クリーンアップ**: プロセス終了時に `override.yml` を自動削除
- ✅ **SOLID原則**: 保守性と拡張性を考慮した設計
- ✅ **構造化ログ**: 詳細なログ出力とデバッグ機能
- ✅ **クロスプラットフォーム**: Linux、macOS、Windows対応
- ✅ **並列処理**: ポートスキャンを並列で実施

## インストール

### ソースからビルド

```bash
git clone https://github.com/harakeishi/gopose.git
cd gopose
make build
sudo make install
```

## 使用方法

### 基本的な使用方法

```bash
# ポート衝突を検出・解決してDocker Composeを準備
gopose up

```

### 高度な使用方法

#### ファイル指定とポート範囲設定

```bash
# 特定のファイルを指定
gopose up -f custom-compose.yml

# ポート範囲を指定
gopose up --port-range 9000-9999

# 複数のポート範囲を指定
gopose up --port-range 8000-8999,9000-9999
```

#### 除外設定

```bash
# 特定のサービスを除外
gopose up --exclude-services redis,postgres

# 特権ポートを除外
gopose up --exclude-privileged

# 予約ポートを除外
gopose up --exclude-ports 8080,8443,9000
```

#### 出力とログ設定

```bash
# ドライラン（実際の変更は行わない）
gopose up --dry-run

# 詳細ログ出力
gopose up --verbose

# 詳細情報を含めて表示
gopose up --detail # タイムスタンプやフィールドを含めて表示

# JSON形式で状態確認
gopose status --output json

# ログレベルを設定
gopose up --log-level debug
```

### 設定ファイル

設定ファイル（`.gopose.yaml`）をホームディレクトリまたはプロジェクトディレクトリに配置できます：

```yaml
port:
  range:
    start: 8000
    end: 9999
  reserved: [8080, 8443, 9000, 9090]
  exclude_privileged: true

file:
  compose_file: "docker-compose.yml"
  override_file: "docker-compose.override.yml"
  backup_enabled: true

watcher:
  interval: "5s"
  cleanup_delay: "30s"

log:
  level: "info"
  format: "text"
  file: "~/.gopose/logs/gopose.log"

resolver:
  strategy: "minimal_change"  # minimal_change, sequential, random
  preserve_dependencies: true
  port_proximity: true
```

### 出力例

```
$ gopose up
ポート衝突解決を開始
Docker Composeファイル検出開始
Docker Composeファイル発見
Docker Composeファイル検出完了
Docker Composeファイルを自動検出
Docker Composeファイル解析開始
Docker Composeバージョンが指定されていません
Docker Composeファイル解析完了
ポート衝突検出開始
netstatを使用してポートスキャンを開始
ポートスキャン完了
システムポート衝突検出
ポート衝突検出完了
ポート衝突検出完了
ポート衝突解決開始
netstatを使用してポートスキャンを開始
ポートスキャン完了
範囲内ポートフィルタリング完了
ポート割り当て成功
解決案最適化開始
解決案最適化完了
ポート衝突解決完了
ポート解決
Override生成開始
ポートマッピング更新
Override生成完了
Override検証開始
Overrideのバージョンが指定されていませんが、Docker Composeの最新バージョンでは非推奨のため許可します
Override検証完了
Overrideファイル書き込み開始
Overrideファイル書き込み完了
Override.ymlファイルが生成されました
既存のコンテナを停止してからDocker Composeを起動
[+] Running 2/2
 ✔ Container gopose-web-1  Removed                                                                                         0.0s
 ✔ Network gopose_default  Removed                                                                                         0.2s
Docker Composeを起動
Docker Composeを実行
[+] Running 2/2
 ✔ Network gopose_default  Created                                                                                         0.0s
 ✔ Container gopose-web-1  Created                                                                                         0.0s
Attaching to web-1
```

detail指定時

```
$ gopose up --detail
time=2025-06-10T23:31:03.179+09:00 level=INFO msg=ポート衝突解決を開始 component=gopose timestamp=2025-06-10T23:31:03.178+09:00 dry_run=false compose_file=docker-compose.yml output_file="" strategy=auto port_range=8000-9999 skip_compose_up=false
time=2025-06-10T23:31:03.179+09:00 level=INFO msg="Docker Composeファイル検出完了" component=gopose timestamp=2025-06-10T23:31:03.179+09:00 directory=/Users/keishi.hara/src/github.com/harakeishi/gopose found_count=1
time=2025-06-10T23:31:03.179+09:00 level=INFO msg="Docker Composeファイルを自動検出" component=gopose timestamp=2025-06-10T23:31:03.179+09:00 file=/Users/keishi.hara/src/github.com/harakeishi/gopose/compose.yml
time=2025-06-10T23:31:03.180+09:00 level=WARN msg="Docker Composeバージョンが指定されていません" component=gopose timestamp=2025-06-10T23:31:03.180+09:00
time=2025-06-10T23:31:03.180+09:00 level=INFO msg="Docker Composeファイル解析完了" component=gopose timestamp=2025-06-10T23:31:03.180+09:00 file=/Users/keishi.hara/src/github.com/harakeishi/gopose/compose.yml services_count=1
time=2025-06-10T23:31:03.191+09:00 level=INFO msg=ポートスキャン完了 component=gopose timestamp=2025-06-10T23:31:03.191+09:00 found_ports_count=18
time=2025-06-10T23:31:03.191+09:00 level=WARN msg=システムポート衝突検出 component=gopose timestamp=2025-06-10T23:31:03.191+09:00 port=3000 service=web
time=2025-06-10T23:31:03.191+09:00 level=INFO msg=ポート衝突検出完了 component=gopose timestamp=2025-06-10T23:31:03.191+09:00 conflicts_count=1
time=2025-06-10T23:31:03.191+09:00 level=INFO msg=ポート衝突検出完了 component=gopose timestamp=2025-06-10T23:31:03.191+09:00 conflicts_count=1
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=ポートスキャン完了 component=gopose timestamp=2025-06-10T23:31:03.202+09:00 found_ports_count=18
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=解決案最適化完了 component=gopose timestamp=2025-06-10T23:31:03.202+09:00 original_count=1 optimized_count=1
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=ポート衝突解決完了 component=gopose timestamp=2025-06-10T23:31:03.202+09:00 resolved_conflicts=1
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=ポート解決 component=gopose timestamp=2025-06-10T23:31:03.202+09:00 service=web from=3000 to=8001 reason="ポート 3000 から 8001 への自動変更"
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=Override生成完了 component=gopose timestamp=2025-06-10T23:31:03.202+09:00 services_count=1
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=Override検証完了 component=gopose timestamp=2025-06-10T23:31:03.202+09:00
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=Overrideファイル書き込み完了 component=gopose timestamp=2025-06-10T23:31:03.202+09:00 output_path=docker-compose.override.yml file_size=607
time=2025-06-10T23:31:03.202+09:00 level=INFO msg=Override.ymlファイルが生成されました component=gopose timestamp=2025-06-10T23:31:03.202+09:00 output_file=docker-compose.override.yml
time=2025-06-10T23:31:03.202+09:00 level=INFO msg="既存のコンテナを停止してからDocker Composeを起動" component=gopose timestamp=2025-06-10T23:31:03.202+09:00
[+] Running 2/2
 ✔ Container gopose-web-1  Removed                                                                                         0.2s
 ✔ Network gopose_default  Removed                                                                                         0.2s
time=2025-06-10T23:31:03.779+09:00 level=INFO msg="Docker Composeを起動" component=gopose timestamp=2025-06-10T23:31:03.779+09:00
time=2025-06-10T23:31:03.780+09:00 level=INFO msg="Docker Composeを実行" component=gopose timestamp=2025-06-10T23:31:03.780+09:00 command="docker compose -f /Users/keishi.hara/src/github.com/harakeishi/gopose/compose.yml -f docker-compose.override.yml up --force-recreate --remove-orphans"
[+] Running 2/2
 ✔ Network gopose_default  Created                                                                                         0.0s
 ✔ Container gopose-web-1  Created                                                                                         0.0s
Attaching to web-1
```

## ディレクトリ構造

```
gopose/
├── cmd/                 # CLIコマンド
│   ├── root.go         # Cobra root command + DI container
│   ├── up.go           # up subcommand
│   ├── clean.go        # clean subcommand
│   ├── status.go       # status subcommand
│   └── wire.go         # 依存性注入設定 (Wire)
├── internal/           # 内部実装
│   ├── app/           # アプリケーション層
│   ├── scanner/       # ポートスキャン
│   ├── parser/        # Docker Compose解析
│   ├── resolver/      # 衝突解決
│   ├── generator/     # Override生成
│   ├── file/          # ファイル操作
│   ├── watcher/       # プロセス監視
│   ├── cleanup/       # 自動クリーンアップ
│   ├── config/        # 設定管理
│   ├── logger/        # ログ機能
│   └── errors/        # エラーハンドリング
├── pkg/               # 公開パッケージ
│   ├── types/         # 型定義
│   └── testutil/      # テストユーティリティ
├── test/              # テスト
│   ├── unit/          # 単体テスト
│   ├── integration/   # 統合テスト
│   └── e2e/           # E2Eテスト
├── docs/              # ドキュメント
├── scripts/           # スクリプト
└── deployments/       # デプロイメント設定
```

## 開発

### 開発環境セットアップ

```bash
# リポジトリをクローン
git clone https://github.com/harakeishi/gopose.git
cd gopose

# 依存関係のインストール
make deps

# 開発用ビルド
make dev

# テスト実行
make test

# コード品質チェック
make check
```

### Make タスク

```bash
# ビルド
make build              # 通常ビルド
make build-all          # 全プラットフォーム向けビルド
make dev                # 開発用ビルド

# テスト
make test               # 全テスト実行
make test-unit          # 単体テスト
make test-integration   # 統合テスト
make test-e2e           # E2Eテスト
make test-coverage      # カバレッジ生成

# コード品質
make fmt                # コードフォーマット
make lint               # リンター実行
make vet                # go vet実行
make check              # 全チェック実行

# 開発
make run                # 実行
make clean              # クリーンアップ
make deps               # 依存関係インストール

# リリース
make release            # リリースビルド
make docker-build       # Dockerイメージビルド
```

### テスト

```bash
# 全テスト実行
go test ./...

# カバレッジ付きテスト
go test -race -coverprofile=coverage.out ./...

# ベンチマークテスト
go test -bench=. ./...

# 特定のテストのみ実行
go test -run TestPortScanner ./internal/scanner/
```

## ライセンス

このプロジェクトは [MIT License](LICENSE) の下で公開されています。
---

<div align="center">
  <p>Developed by <a href="https://github.com/harakeishi">harakeishi</a></p>
  <p>
    <a href="https://github.com/harakeishi/gopose/issues">🐛 バグ報告</a> •
    <a href="https://github.com/harakeishi/gopose/discussions">💬 ディスカッション</a> •
    <a href="https://github.com/harakeishi/gopose/wiki">📖 Wiki</a>
  </p>
</div>

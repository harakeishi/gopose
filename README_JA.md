# gopose - Docker Compose ポート衝突自動解決ツール

<div align="center">
  <img src="logo.png" alt="gopose logo" width="200"/>
  
  [![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
  [![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
  [![Go Report Card](https://goreportcard.com/badge/github.com/harakeishi/gopose?style=for-the-badge)](https://goreportcard.com/report/github.com/harakeishi/gopose)
</div>

## 概要

**gopose** (Go Port Override Solution Engine) は、Docker Compose のポートバインディング衝突とネットワーク衝突を自動検出・解決するツールです。

元の `docker-compose.yml` を変更せずに `docker-compose.override.yml` を生成し、ポート衝突・ネットワーク衝突解決後、自動的に `override.yml` を削除します。

### 🎯 主な特徴

- ✅ **非破壊的**: 元の `docker-compose.yml` ファイルを変更しません
- ✅ **自動検出**: システムの使用中ポートとの衝突を自動検出
- ✅ **自動解決**: 利用可能なポートを自動割り当て
- ✅ **ネットワーク衝突回避**: Dockerネットワークのサブネット衝突を自動検出・回避
- ✅ **自動クリーンアップ**: プロセス終了時に `override.yml` を自動削除
- ✅ **SOLID原則**: 保守性と拡張性を考慮した設計
- ✅ **構造化ログ**: 詳細なログ出力とデバッグ機能
- ✅ **クロスプラットフォーム**: Linux、macOS、Windows対応
- ✅ **並列処理**: ポートスキャンを並列で実施

## インストール

### バイナリリリース

[GitHub Releases](https://github.com/harakeishi/gopose/releases) から適切なバイナリをダウンロード:

```bash
# Linux (amd64)
curl -L https://github.com/harakeishi/gopose/releases/latest/download/gopose_linux_amd64.tar.gz | tar xz
sudo mv gopose /usr/local/bin/

# macOS (arm64)
curl -L https://github.com/harakeishi/gopose/releases/latest/download/gopose_darwin_arm64.tar.gz | tar xz
sudo mv gopose /usr/local/bin/

# Windows (amd64)
curl -L https://github.com/harakeishi/gopose/releases/latest/download/gopose_windows_amd64.zip -o gopose.zip
unzip gopose.zip
```

### Go Install

```bash
go install github.com/harakeishi/gopose@latest
```

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
# ポート衝突・ネットワーク衝突を検出・解決してDocker Composeを準備
gopose up

# バージョン確認
gopose version
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
既存Dockerネットワークを検出しました
Docker Composeネットワーク設定を検出
ネットワークサブネット競合を検出
ネットワークサブネット競合を解決
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

#### detail指定時

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
time=2025-06-10T23:31:03.205+09:00 level=INFO msg="既存Dockerネットワークを検出しました" component=gopose timestamp=2025-06-10T23:31:03.205+09:00 network_count=3
time=2025-06-10T23:31:03.205+09:00 level=INFO msg="Docker Composeネットワーク設定を検出" component=gopose timestamp=2025-06-10T23:31:03.205+09:00 network_count=1
time=2025-06-10T23:31:03.205+09:00 level=WARN msg="ネットワークサブネット競合を検出" component=gopose timestamp=2025-06-10T23:31:03.205+09:00 network=default conflicting_subnet="172.20.0.0/24"
time=2025-06-10T23:31:03.205+09:00 level=INFO msg="ネットワークサブネット競合を解決" component=gopose timestamp=2025-06-10T23:31:03.205+09:00 network=default original_subnet="172.20.0.0/24" new_subnet="10.20.0.0/24"
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

## ネットワーク衝突回避機能

goposeは既存のDockerネットワークとのサブネット衝突を自動検出し、安全な代替サブネットを割り当てます。

### 機能概要

- **自動検出**: 既存のDockerネットワークサブネットを自動検出
- **衝突回避**: Docker Composeで定義されたネットワークのサブネットが既存ネットワークと衝突する場合、安全な代替サブネットを自動生成
- **優先順位**: `10.x.x.x/24` > `192.168.x.x/24` > `172.x.x.x/24` の順で安全なサブネットを選択
- **競合回避**: Dockerのデフォルト範囲（`172.17-29.x.x`）や一般的なホームルーター範囲を回避

### サブネット割り当て戦略

1. **10.x.x.x/24 範囲**: 最も安全（`10.20.0.0/24` から開始）
2. **192.168.x.x/24 範囲**: 一般的なホームルーター範囲を回避（`192.168.100.0/24` から開始）
3. **172.x.x.x/24 範囲**: 最後の手段（`172.30.0.0/24` から開始、Dockerデフォルト範囲を回避）

### 動作例

```yaml
# 元のdocker-compose.yml
networks:
  app-network:
    ipam:
      config:
        - subnet: 172.20.0.0/24  # 他のDockerネットワークと衝突

# 生成されるdocker-compose.override.yml
networks:
  app-network:
    ipam:
      config:
        - subnet: 10.20.0.0/24  # 安全なサブネットに自動変更
```

## ディレクトリ構造

```
gopose/
├── cmd/                 # CLIコマンド
│   ├── root.go         # Cobra root command + DI container
│   ├── up.go           # up subcommand
│   ├── clean.go        # clean subcommand
│   ├── status.go       # status subcommand
│   ├── version.go      # version subcommand
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

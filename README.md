# gopose - Docker Compose ポート衝突自動解決ツール

## 概要

`gopose` (Go Port Override Solution Engine) は、Docker Compose のポートバインディング衝突を自動検出・解決するツールです。

元の `docker-compose.yml` を変更せずに `docker-compose.override.yml` を生成し、ポート衝突解決後、自動的に `override.yml` を削除します。

## 特徴

- ✅ **非破壊的**: 元の `docker-compose.yml` ファイルを変更しません
- ✅ **自動検出**: システムの使用中ポートとの衝突を自動検出
- ✅ **自動解決**: 利用可能なポートを自動割り当て
- ✅ **自動クリーンアップ**: プロセス終了時に `override.yml` を自動削除
- ✅ **SOLID原則**: 保守性・拡張性に優れた設計
- ✅ **構造化ログ**: 詳細なログ出力とデバッグ機能

## インストール

### ソースからビルド

```bash
git clone https://github.com/harakeishi/gopose.git
cd gopose
make build
sudo make install
```

### 開発環境セットアップ

```bash
# 依存関係のインストール
make deps

# 開発用ビルド
make dev

# テスト実行
make test

# コード品質チェック
make check
```

## 使用方法

### 基本的な使用方法

```bash
# ポート衝突を検出・解決してDocker Composeを準備
gopose up

# 生成されたoverride.ymlファイルを削除
gopose clean

# 現在の状態確認
gopose status
```

### オプション付きの使用方法

```bash
# 特定のファイルを指定
gopose up -f custom-compose.yml

# ポート範囲を指定
gopose up --port-range 9000-9999

# ドライラン（実際の変更は行わない）
gopose up --dry-run

# 詳細ログ出力
gopose up --verbose

# JSON形式で状態確認
gopose status --output json
```

## アーキテクチャ

本プロジェクトはSOLID原則に基づいて設計されており、以下の特徴があります：

- **単一責任の原則**: 各コンポーネントが明確な単一責任を持つ
- **開放閉鎖の原則**: インターフェースベースで拡張が容易
- **依存性逆転の原則**: Google Wireを使用した依存性注入
- **テスタビリティ**: インターフェースベースでモック作成が容易

### 主要コンポーネント

- **PortScanner**: システムポートの検出と利用可能ポートの割り当て
- **ComposeParser**: Docker Composeファイルの解析
- **ConflictResolver**: ポート衝突の検出と解決
- **OverrideGenerator**: `docker-compose.override.yml` の生成
- **ProcessWatcher**: Docker Composeプロセスの監視
- **CleanupManager**: 自動クリーンアップの管理

## 設定

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
```

## 開発

### プロジェクト構造

```
gopose/
├── cmd/                 # CLIコマンド
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
└── test/              # テスト
```

### Make タスク

```bash
# ビルド
make build              # 通常ビルド
make build-all          # 全プラットフォーム向けビルド

# テスト
make test               # 全テスト実行
make test-unit          # 単体テスト
make test-coverage      # カバレッジ生成

# コード品質
make fmt                # コードフォーマット
make lint               # リンター実行
make check              # 全チェック実行

# 開発
make dev                # 開発用ビルド
make run                # 実行
make clean              # クリーンアップ
```

## ライセンス

このプロジェクトは [LICENSE](LICENSE) ファイルに記載されたライセンスの下で公開されています。

## コントリビューション

プルリクエストやイシューの報告を歓迎します。設計書（`DESIGN.md`）を参照して、アーキテクチャの理解を深めてください。

## ロードマップ

- [ ] Phase 1: 基盤構築 ✅
- [ ] Phase 2: コア機能実装
- [ ] Phase 3: 統合・監視機能
- [ ] Phase 4: CLI・UX改善
- [ ] Phase 5: パフォーマンス・リリース準備

詳細な実装計画は [DESIGN.md](DESIGN.md) を参照してください。

# gopose E2E Tests

Kent Beck風の段階的テスト進行戦略によるE2Eテストスイート。

## テスト戦略

### Phase 1: "Does it exist?" (存在確認)
- バイナリが存在し実行可能か確認

### Phase 2: "Does it behave?" (基本動作)
- ヘルプ表示、基本コマンド動作確認

### Phase 3: "Does it do what it says?" (宣言通りの動作)
- ドライラン、override.yml生成確認

### Phase 4: "Does it solve real problems?" (実問題解決)
- ポート衝突の検出・解決確認

### Phase 5: "Does it clean up?" (後始末)
- ファイル削除、状態復元確認

## 実行方法

```bash
# runn をインストール
go install github.com/k1LoW/runn/cmd/runn@latest

# 全フェーズ実行
make test-e2e

# 個別フェーズ実行
make test-e2e-phase1  # スモークテスト
make test-e2e-phase2  # 基本動作
make test-e2e-phase3  # コア機能
make test-e2e-phase4  # 衝突解決
make test-e2e-phase5  # クリーンアップ
```

## テストファイル構成

- `fixtures/` - テスト用Docker Composeファイル
  - `basic-compose.yml` - 基本的な構成
  - `conflict-compose.yml` - 衝突を含む構成
- `phase*.yml` - 各フェーズのテストシナリオ
- `all_phases.yml` - 全フェーズ統合実行
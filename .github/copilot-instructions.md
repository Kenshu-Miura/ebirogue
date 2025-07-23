# ebirogue - AI Coding Assistant Instructions

このリポジトリは、Go言語とEbitenライブラリを使用したローグライクゲームです。

## アーキテクチャの要点

### ビルドタグシステム
- **本体**: `//go:build !test` タグで実際のEbiten依存コードを分離
- **テスト**: `//go:build test` タグでスタブ実装を提供
- テスト実行時は必ず `go test -tags test ./...` を使用

### ActionQueueパターン
- すべてのゲーム行動は `ActionQueue` に積まれ、`HandleActionQueue()` で順次実行
- `action.go` で行動を作成、`animation.go` の `HandleActionQueue()` で処理
- アニメーションとメッセージ表示のタイミング制御に使用

### ファイル構成原則
- **機能別分離**: `input.go`(入力)、`move.go`(移動)、`draw.go`(描画)など
- **インターフェース設計**: `interfaces.go` で `Character`, `Item`, `Equipable` を定義
- **効果システム**: `itemeffects.go` にアイテム効果関数をまとめて実装

## 重要な開発パターン

### WASMビルド
```bash
# 最適化ビルド（推奨）
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o ebirogue.wasm
```

### マップシステム
- `map.go` の `generateRooms()` → `connectRooms()` → `updateTileBrightness()` の流れ
- 部屋ベース生成 + 廊下接続アルゴリズム
- プレイヤー周辺のみ明るく表示する視界システム

### インターフェース実装パターン
```go
// interfaces.go で定義、各構造体で実装
type Item interface {
    GetPosition() (int, int)
    GetName() string
    // ...
}
```

### stubシステム活用
- `ebitenstub/` ディレクトリにEbitenの軽量スタブ実装
- テスト時の依存関係解決とヘッドレス実行を可能にする

## 開発ワークフロー

1. **機能追加**: 既存ファイル構成に従い適切なファイルに実装
2. **テスト**: `go test -tags test ./...` でスタブ環境テスト
3. **ビルド**: 本体は `go run .`、WASM版は上記コマンド
4. **フォーマット**: `gofmt -w` を変更前に実行

## 注意点

- 日本語コメントが混在（新規コードは周囲に合わせる）
- `Game` 構造体がゲーム状態の中心（`main.go` で初期化）
- アイテム効果追加時は `itemeffects.go` の既存パターンに従う
- 敵・プレイヤー移動は `move.go` の体力・満腹度回復ロジックを考慮

# wave7-C: 音声システムと音量設定（Codex 用）

ブランチ: `agent/wave7-C-audio`
前提: master の最新。`AGENTS.md`、`docs/agents/README.md`、`docs/agents/feature-completion-plan.md` を先に読む。状態異常のファイルには触れないので wave7-A と並行してよい。

## 目的

今のゲームには音が無い。効果音と BGM を鳴らす仕組みと、設定の音量（AGENTS.md「Priority A: usability」の残り）を作る。

## 作るもの

1. `audio.go`（`!test`）: `github.com/hajimehoshi/ebiten/v2/audio` の `audio.Context`（44100Hz）を1つ持ち、`g.playSE(id)` と `g.playBGM(id)`／`g.stopBGM()` を提供する。テストのビルド（`-tags test`）とヘッドレス（BOT）では何もしない（`audio_stub.go` か `g.persistent` で判定）。
2. 音源: `sound/*.wav`（または `.ogg`）を `//go:embed` で埋め込む。**仮の音源は Go のスクリプト（`tools/generate_sounds.go`）で合成して作る**（矩形波・ノイズの短い音。著作権の問題を避けるため外部の音源は使わない）。BGM は8〜16小節のループを1〜3曲（通常の階・ボス階・クリア）。
3. 鳴らす場所（最低限）: 攻撃の命中・空振り、被弾、敵を倒す、レベルアップ、道具を拾う、階段を降りる、罠が作動、カード・杖・薬の使用、死亡、クリア、メニューの決定・取り消し。呼ぶのは既存の Action の `Execute` の中（演出のタイミングと合わせる）。
4. 設定: 「BGM の音量」「効果音の音量」（0〜10、初期値 6）を `GameSettings` に省略可能な項目で足し、設定画面に並べる。0 で無音。
5. WASM: ブラウザはユーザー操作の前に音を出せないので、最初のキー入力で `audio.Context` を再開する。

## 守ること

- 音の追加でゲームの乱数・ターン進行を変えない（BOT の結果がベースラインと完全一致すること）。
- `ebiten.IsKeyPressed` を直接呼ばない（AGENTS.md の入力の規則）。
- 新しいファイル名に `_js.go` などの GOOS 接尾辞を意図せず付けない。

## 完了条件

- テスト: 音量の設定の保存と読み込み（古い設定ファイルでも初期値になる）、ヘッドレスで音声の関数を呼んでも何も起きない。
- gofmt、`go test -tags test ./...`、`go test ./...`、`GOOS=js GOARCH=wasm go build` が成功。`TestSimBatch -sim.runs=300` がベースラインと一致。
- `docs/agents/done/wave7-C.md`（変更概要・IMPLEMENTED_FEATURES.md への追記案・AGENTS.md の音量設定の削除案・**仮の音源の一覧（正式な音源への差し替え候補）**・未決事項）。実機での音の確認は開発者が行う。
- コミットしてブランチをプッシュする。

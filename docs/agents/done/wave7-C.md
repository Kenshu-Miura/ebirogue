# wave7-C: 音声システムと音量設定

## 変更概要

- ブランチ: `agent/wave7-C-audio`。最新 `origin/master` の `b3224c9` から作成。
- `audio.go` にプロセスで1つの 44100Hz Context、効果音の同時再生（最大16音）、再生終了した Player の解放、BGM のループと切り替えを追加。
- 通常階は dungeon、未撃破のボス階は boss、クリア時は victory。死亡・難易度選択・中断・階移動の暗転中は BGM を停止。
- 攻撃命中・空振り、被弾、敵撃破、レベルアップ、拾得、階移動、作動した罠、カード・薬・杖使用、死亡、クリア、メニュー決定・取消に効果音を接続。既存 Action の Execute と効果の共通処理に追加し、キューの件数・時間・乱数・ターン処理は維持。
- 設定に BGM・効果音の音量（0〜10、初期値6）を追加。左右キーで即時反映し、既存の設定保存経路で保存する。旧JSONの項目省略時は6、明示した0は保存・再読込しても0、不正な範囲は0〜10へ補正。セーブデータのバージョンは変更しない。
- 最初の論理キー入力まで音声初期化を遅延。ブラウザの再開は Ebitengine / Oto のユーザー操作イベント処理に任せる（Ebiten v2.8.8 の audio.Context に公開 Resume メソッドはない）。直接の Ebiten キー参照は追加していない。
- `persistent == false` または headless では Context / Player を生成せず、テストタグでは `audio_stub.go` の空実装を使う。
- `tools/generate_sounds.go` が外部素材なしで WAV を合成。ノイズも独自の固定シードによる合成専用系列で、ゲームの乱数は消費しない。再生成した全WAVのSHA-256一致を確認。
- 音声依存の `github.com/ebitengine/oto/v3 v3.3.3` を go.mod / go.sum に追加。

## IMPLEMENTED_FEATURES.md への追記案

「メニュー・設定」に以下を追記:

- BGM と効果音を再生する。通常階・ボス階・クリアでBGMを切り替え、戦闘・道具・罠・階移動・メニュー操作の効果音を再生する。音源は Go スクリプトで合成した仮WAVを埋め込む。
- BGM・効果音の音量をそれぞれ0〜10（初期値6、0で無音）で変更でき、即時反映・保存する。旧設定ファイルでは初期値となる。BOT・テストでは音声を初期化しない。

## AGENTS.md から削除するロードマップ項目

- Priority A: usability の「BGM・効果音の音量設定（音声システム自体が未実装のため、その導入後に追加する）」を削除する。
- 実装済みの保存設定・音声責務の説明を必要に応じて追記する。
- 本ブランチでは AGENTS.md と IMPLEMENTED_FEATURES.md を変更していない。

## 仮の音源一覧（正式な音源への差し替え候補）

すべて `sound/` 内、44100Hz・16bit・ステレオの WAV。外部の楽曲・音源を使用していない。

| ファイル | 用途 |
|---|---|
| hit.wav | 攻撃命中（矩形波＋ノイズ） |
| miss.wav | 空振り（矩形波＋ノイズ） |
| hurt.wav | 被弾（矩形波＋ノイズ） |
| defeat.wav | 敵撃破 |
| level.wav | レベルアップ |
| pickup.wav | 道具・お金の拾得 |
| stairs.wav | 階移動 |
| trap.wav | 罠作動（矩形波＋ノイズ） |
| use.wav | カード・薬・杖使用 |
| death.wav | 死亡 |
| clear.wav | クリア |
| confirm.wav | メニュー決定 |
| cancel.wav | メニュー取消 |
| dungeon.wav | 通常階の8小節ループ（4/4、240 BPM、8秒） |
| boss.wav | ボス階の8小節ループ（同上） |
| victory.wav | クリアの8小節ループ（同上） |

仮画像の追加・変更はない。

## 未決事項

- 音色・BGMの速さ・音量バランスは仮。正式な音源への差し替えと、実機・ブラウザでの聴感確認は開発者が行う。自動検証はデコード・ビルド・ヘッドレス経路まで。
- キー設定の既存項目位置を維持し、その下にBGM・効果音を並べた。ヘッドレスでは音量項目を巡回に追加しない。random BOTの設定操作を変えないため、従来の3項目巡回を維持した。音声のないBOTの操作経路を変えずに実機の設定を拡張する判断とした。
- メニューと階移動の開始は元からキューに積まれないため、音専用の即時 Action.Execute を使う。入力待ち・ターン・キュー件数を変えない。
- 一度に複数の敵を倒すなどで音が重なる場合は最大16音とし、超えたら古い効果音を閉じる。
- 0は保存すべき有効値なので音量JSONフィールドに omitempty は付けない。「省略可能」はデコード時に初期値を補うことで対応する。

## 検証

- `gofmt -w`、`git diff --check`: 成功。
- `go test -tags test ./...`: 成功。
- `go test ./...`（TestSimSmokeを含む）: 成功。
- `go build -o work/ebirogue .` と `GOOS=js GOARCH=wasm go build -o work/ebirogue.wasm .`: 成功。
- 追加テスト: 音量設定の旧JSON互換・0保存・再読込・範囲補正、音量操作の上限下限・ターン非消費、ヘッドレス/非永続ゲームでの無初期化とゲーム状態不変、Action実行・時間の維持、全埋め込みWAVのデコードとBGM長、testタグの空実装。
- Linuxの通常版テストは仮想ディスプレイ（xvfb-run）を使用。ALSA開発ファイルを作業領域のsysrootに追加して検証した。

### TestSimBatch

- コマンド: `go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=300 -sim.difficulty=all -sim.out=reports`。成功。
- 306冒険（難易度3×性格6×各17）。クリア33・死亡272・BOT行き詰まり1、違反0・ソフトロック0・panic0。
- 音声実装前の master で同じ条件を実行して保存した結果と、`Summary` および306件すべての `Outcomes` がJSONオブジェクトとして完全一致した（レポートの生成日時のみ比較から除外）。既存のBOT行き詰まり1件も一致。
- 実装後レポート: `reports/sim-20261021-005846.json` と同名の `.md`。reports/baseline.json は更新していない。

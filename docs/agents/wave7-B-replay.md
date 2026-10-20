# wave7-B: 入力の記録の保存と再生（Codex 用）

ブランチ: `agent/wave7-B-replay`
前提: master の最新。`AGENTS.md`、`docs/agents/README.md`、`docs/agents/feature-completion-plan.md` を先に読む。状態異常のファイルには触れないので wave7-A と並行してよい。

## 目的

人が遊んだ冒険をシードと入力の列で保存し、同じ結果を再生できるようにする。不具合の報告に使う（BOT の冒険は `TestSimReplay` で再現できるが、人のプレイは再現できない）。

## 今の状態

- `input_source.go` に `InputSource`・`VirtualInput`・`InputRecorder`（入力の記録）がある。冒険の乱数は `rng.go`（シード・状態を保存できる）。
- 保存の入出力は `storage.go` / `storage_js.go` の `storageRead`/`storageWrite`/`storageRemove`/`storageIsNotExist` を通す（WASM は localStorage）。

## 作るもの

1. 冒険ごとの記録: シード・難易度・ゲームのバージョン（`saveDataVersion` とビルドの識別）・フレームごとの論理キーの押下（連続する同じ状態はまとめて小さくする）。`ebirogue_replay_last.json`（最後の冒険）と、死亡・クリア時に `ebirogue_replays.json`（最大10件）へ保存する。
2. 中断セーブから再開した冒険は、中断前の記録も続けて保存する（中断セーブに記録を含める。省略可能な項目にし、古いセーブも読める）。
3. 再生: テストから `go test -run TestReplayFile -count=1 . -args -replay.file=<パス>` でヘッドレス再生し、最後の状態（階・HP・死因・ターン数）を表示する。ゲーム内からの再生は今回は作らない。
4. 設定に「入力の記録を保存する」（初期値オン）を足す。`GameSettings` に省略可能な項目で追加する。
5. メニューの「記録」の一覧で、記録のある冒険に印を付ける（任意。難しければ未決事項へ）。

## 守ること

- 記録は `g.persistent` のときだけ書く（テスト・BOT は書かない）。
- フレームの数え方は `stepRepeatFrames` などフレーム基準の入力処理と一致させる。実時間に依存しない。
- 再生で一致しないとき（バージョン違いなど）は、ずれたフレームを表示して止める。

## 完了条件

- テスト: 仮想入力で数百フレーム遊んだ冒険を記録→再生して状態が一致する、中断→再開をまたいでも一致する、壊れた記録・古いバージョンの記録を読んでも panic しない、WASM ビルドが通る。
- gofmt、`go test -tags test ./...`、`go test ./...`、`GOOS=js GOARCH=wasm go build` が成功。
- `docs/agents/done/wave7-B.md`（変更概要・IMPLEMENTED_FEATURES.md への追記案・AGENTS.md の Priority C の削除案・再生の手順・未決事項）。
- コミットしてブランチをプッシュする。

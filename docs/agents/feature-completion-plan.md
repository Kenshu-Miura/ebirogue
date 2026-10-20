# 未実装機能の実装計画（wave7〜wave10）

作成: 2026-10-20。開発者の方針: **未実装の機能をすべて実装してから BOT を強化し、そのあとクリア率を目標に合わせる。**

最初に `AGENTS.md`、`docs/agents/README.md`（並列作業のルール・ID 割り当て）、`docs/agents/STATUS.md` を読むこと。各タスクの詳しい仕様は下の表の指示書にある。

## クリア率の目標（開発者の決定、2026-10-20）

| 難易度 | 目標 | 現在（`reports/baseline.json`、expert 各84冒険） |
|---|---|---|
| 初級（10階ボス） | 85%前後 | 91.7% |
| 中級（20階ボス） | 50%前後 | 48.8% |
| 上級（30階ボス） | 10%前後 | 4.8% |

- 測る BOT は **expert**。各難易度 500 冒険（`-sim.persona=expert -sim.runs=1500 -sim.difficulty=all`）で判定する（84冒険ではばらつきが大きい）。
- 調整は wave11（BOT 強化の後）で行う。wave7〜10 の担当は、新しい要素で BOT の結果が大きく動いた場合に報告するだけでよく、数値を合わせにいかない。

## 未実装の機能の一覧（出典）

| # | 機能 | 出典 | タスク |
|---|---|---|---|
| 1 | 状態異常の共通化（残りターン・重複規則・解除・耐性・敵味方の両方） | AGENTS.md「traps and status ailments」、wave2-backlog 1 | wave7-A |
| 2 | 入力履歴のファイル保存とシードからの再生 | AGENTS.md「Priority C」 | wave7-B |
| 3 | 音声システム（効果音・BGM）と音量設定 | AGENTS.md「usability」 | wave7-C |
| 4 | 仮画像の差し替え（罠 5〜14、腐ったウインナー、ボス 42〜44、第3段階 45〜65） | AGENTS.md「traps and status ailments」、STATUS.md | wave7-D |
| 5 | 新しい状態異常（能力値系）: 倍速（海老さん）・攻撃力/防御力の上昇と低下・レベル低下・回復不能、それを起こす薬と杖 | AGENTS.md、wave2-backlog 2 | wave8-A |
| 6 | 新しい状態異常（特殊系）: おにぎり・キグニ族・身代わり・透明・無敵、それを起こす薬と杖 | AGENTS.md、wave2-backlog 2 | wave8-B |
| 7 | 敵も罠を踏む／矢・投げた道具・爆発で罠が作動する | AGENTS.md、wave2-backlog 4 | wave8-C |
| 8 | 階層別のアイテム・罠の出現テーブル（カードの出現率の調整を含む） | AGENTS.md「Priority B」「cards」、wave2-backlog 5 | wave9-A |
| 9 | 特殊地形（水路・壊せる壁・隠し通路） | AGENTS.md「Priority B」 | wave9-B |
| 10 | 壺の追加と合成（合成の壺など） | AGENTS.md「Priority B」 | wave9-C |
| 11 | モンスターハウス（フロア生成で自然に出る） | AGENTS.md「Priority B」、wave2-backlog 6 | wave10-A |
| 12 | 店・売買・店主・泥棒 | AGENTS.md「Priority B」 | wave10-B |
| 13 | アイテム・敵・罠の図鑑と発見率 | AGENTS.md「Priority B」 | wave10-C |
| — | BOT の強化 → クリア率の調整 | 開発者の方針 | wave11（未作成。wave10 の統合後に Claude Code が指示書を作る） |

杖は2種類（シフトチェンジ・封印）、壺は1種類（保存）、薬は回復と状態薬だけしか無い。新しい状態異常を起こす手段として wave8-A/B で薬と杖を、wave9-C で壺を足す。

## 段と依存関係

| 段 | 並列に進めるタスク | 前提 |
|---|---|---|
| 段7 | **wave7-A（単独で多くのファイルに触れる）**。同時に進めてよいのは、状態異常に触れない wave7-B・7-C・7-D だけ | master の最新 |
| 段8 | wave8-A・8-B・8-C | wave7-A の統合 |
| 段9 | wave9-A・9-B・9-C | 段8の統合（9-A は全アイテムがそろってから重みを決めるため、9-C の ID だけ先に決めておく） |
| 段10 | wave10-A・10-B・10-C | 段9の統合（10-A/B は 9-B の地形の上に部屋を作る。10-C は全コンテンツがそろってから） |
| 段11 | BOT 強化 → クリア率の調整 | 段10の統合 |

各段の統合（マージ・衝突の解消・`IMPLEMENTED_FEATURES.md` と `AGENTS.md` の更新・`TestSimBatch` でベースラインを取り直す・`STATUS.md` の更新）は Claude Code が行う。

## 担当と ID の割り当て

| タスク | 担当の目安 | ブランチ | 指示書 | ID |
|---|---|---|---|---|
| wave7-A 状態異常の共通化 | Claude Code | `agent/wave7-A-status-foundation` | `wave7-A-status-foundation.md` | なし |
| wave7-B 入力の記録と再生 | Codex | `agent/wave7-B-replay` | `wave7-B-replay.md` | なし |
| wave7-C 音声と音量設定 | Codex | `agent/wave7-C-audio` | `wave7-C-audio.md` | なし |
| wave7-D 仮画像の差し替え | Codex（画像生成が使える方） | `agent/wave7-D-images` | `wave7-D-images.md` | なし |
| wave8-A 能力値系の状態異常 | Codex | `agent/wave8-A-status-stats` | `wave8-A-status-stats.md` | アイテム 205〜214、罠 15 |
| wave8-B 特殊系の状態異常 | Codex | `agent/wave8-B-status-special` | `wave8-B-status-special.md` | アイテム 215〜224 |
| wave8-C 罠の相互作用 | Codex | `agent/wave8-C-trap-interaction` | `wave8-C-trap-interaction.md` | なし |
| wave9-A 出現テーブル | Claude Code | `agent/wave9-A-floor-tables` | `wave9-A-floor-tables.md` | なし |
| wave9-B 特殊地形 | Codex | `agent/wave9-B-terrain` | `wave9-B-terrain.md` | なし |
| wave9-C 壺と合成 | Codex | `agent/wave9-C-pots-synthesis` | `wave9-C-pots-synthesis.md` | アイテム 225〜239 |
| wave10-A モンスターハウス | Codex | `agent/wave10-A-monster-house` | `wave10-A-monster-house.md` | なし |
| wave10-B 店 | Claude Code | `agent/wave10-B-shop` | `wave10-B-shop.md` | 敵 126〜128（店主・番犬） |
| wave10-C 図鑑 | Codex | `agent/wave10-C-encyclopedia` | `wave10-C-encyclopedia.md` | なし |

担当は目安で、Claude Code と Codex のどちらが行ってもよい。設計の判断が多いもの（7-A・9-A・10-B）を Claude Code にしている。

## 新しい要素の共通の約束

- `AGENTS.md` の「新要素追加チェックリスト」と「Roadmap acceptance criteria」に従う。
- 新しいカード・杖・薬は BOT の `itemRoles`、指輪は `ringValues`、手放す価値は `itemValues` に**最低限の分類だけ**入れる（BOT が新しい道具で行き詰まらないため。使い方の工夫は wave11 で行う）。
- 状態異常は wave7-A の共通処理（`applyStatus` など。名前は 7-A で決まる）を通して付ける。直接 `StatusAilments` の項目を書き換えない。
- 中断セーブに新しい状態を足すときは省略可能な項目にし、古いセーブも読めるようにする（`saveDataVersion` を上げない）。上げる必要があるときは完了報告に理由を書く。
- 完了報告は `docs/agents/done/<タスクID>.md`（変更概要・IMPLEMENTED_FEATURES.md への追記案・AGENTS.md から削除するロードマップ項目・仮画像の一覧・未決事項・`TestSimBatch -sim.runs=300` の要点）。

## 貼り付け用のプロンプト

`<指示書>` と `<ブランチ>` を上の表の値に置き換えて使う。

### Codex 用

```text
ebirogue リポジトリで、docs/agents/<指示書> のタスクを実装してください。

1. 最初に AGENTS.md、docs/agents/README.md、docs/agents/STATUS.md、docs/agents/feature-completion-plan.md、docs/agents/<指示書> を読む。
2. master の最新からブランチ <ブランチ> を作って作業する。master へ直接コミット・プッシュしない。
3. 並列作業のルールに従い、AGENTS.md と IMPLEMENTED_FEATURES.md は編集しない。代わりに docs/agents/done/<タスクID>.md に完了報告を書く。
4. 大きいファイル（itemeffects.go、items.go、content_cards_test.go など）は丸ごと読まず grep で該当箇所を探す。
5. 仕様に書かれていない判断が必要になったら、無難な案で進めて完了報告の「未決事項」に書く（作業は止めない）。
6. 完了条件: gofmt -w、go test -tags test ./...、go test ./...（TestSimSmoke を含む）がすべて成功し、
   go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=300 -sim.difficulty=all -sim.out=reports
   で違反・ソフトロック・panic が 0。
7. コミットしてブランチをプッシュする。
```

### Claude Code 用

```text
ebirogue の docs/agents/<指示書> のタスクを実装してください。

- 最初に AGENTS.md、docs/agents/README.md、docs/agents/STATUS.md、docs/agents/feature-completion-plan.md、docs/agents/<指示書> を読んでください。
- ブランチは <ブランチ> を使ってください（セッションに別のブランチが割り当てられている場合はそちらで作業し、完了報告にブランチ名を書いてください）。
- 並列作業のルールに従い、AGENTS.md と IMPLEMENTED_FEATURES.md は編集せず、docs/agents/done/<タスクID>.md に完了報告を書いてください。
- 仕様に書かれていない判断は無難な案で進め、完了報告の「未決事項」に書いてください。
- gofmt、go test -tags test ./...、go test ./...、TestSimBatch（-sim.runs=300 -sim.difficulty=all）で違反0を確認してから、コミットしてプッシュしてください。
```

### 統合用（各段のすべての担当が終わったら Claude Code に）

```text
ebirogue の段<N>（<タスクIDの一覧>）のブランチを master に統合してください。
docs/agents/feature-completion-plan.md の「段と依存関係」と、各ブランチの docs/agents/done/<タスクID>.md を読み、
衝突を解消し、担当をまたぐテストを content_integration_test.go に足し、IMPLEMENTED_FEATURES.md と AGENTS.md を更新し、
TestSimBatch（-sim.runs=1500 -sim.difficulty=all）でベースラインを取り直して STATUS.md を更新してください。
```

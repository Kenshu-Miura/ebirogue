# wave7-A: 状態異常の共通化（Claude Code 用、単独で直列）

ブランチ: `agent/wave7-A-status-foundation`
前提: master の最新。`AGENTS.md`、`docs/agents/README.md`、`docs/agents/STATUS.md`、`docs/agents/feature-completion-plan.md` を先に読む。
このタスクは段8（新しい状態異常 wave8-A/B、罠の相互作用 wave8-C）の土台になる。**新しい状態異常は作らない。** 今ある状態異常の挙動を変えずに、付与・経過・解除・表示・保存の経路を1か所にまとめる。

## 今の状態（調査済み）

- `StatusAilments`（`main.go` と `game_stub.go`）は状態ごとの項目（`BearTrap`・`Confusion`・`Sleep`・`Blind`・`Slow`・`Haste`・`MouthSeal`・`Backward` の残りターン、`Paralysis`・`Seal` の bool、`HasteOnWake`）。
- 付与は約25ファイルで `StatusAilments.<項目> = n` を直接書いている（`grep -rn 'StatusAilments\.' --include=*.go`）。重ねがけの規則（上書き・長い方・加算）が箇所ごとに違う。
- 残りターンの減算は `move.go` の `decrementStatusAilments`（海老さんと敵）。治ったときのメッセージもここに手書き。
- 無効化は `statusImmunities` / `g.playerImmuneTo` / `g.playerStatusGuard`（`equipment_abilities.go`・`equipment_hooks.go`）。状態の種類 `StatusKind` はここで定義されている。

## 作るもの

1. **状態の定義表**（タグなしの純粋なファイル、例 `status_effects.go`）: `StatusKind` ごとに、名前・HUD の表示・持続の型（ターン／ずっと／解除されるまで）・重ねがけの規則（上書き・長い方を残す・加算して上限まで）・治ったときのメッセージ・海老さんに付くか・敵に付くか・「悪い状態」か（困った時のカード・薬で治す対象）。
2. **共通の付与・解除**:
   - 海老さん: `g.applyPlayerStatus(kind, turns, source)` → 耐性（`playerStatusGuard`）・重ねがけの規則・メッセージを通して付ける。戻り値で付いたかどうか。`g.clearPlayerStatus(kind)`・`g.clearPlayerBadStatuses()`。
   - 敵: `g.applyEnemyStatus(index, kind, turns)`（ボスの耐性 `bosses.go` を通す）、`g.clearEnemyStatus`。
3. **経過の一本化**: `decrementStatusAilments` を定義表を回す形にし、メッセージも表から出す。睡眠から覚めて倍速になる `HasteOnWake` など今の特別な挙動は残す。
4. **今の付与箇所をすべて置き換える**: 罠・カード・薬・杖・敵の特殊攻撃・ボス・装備能力・困った時のカード・BOT の参照。置き換えの前後で挙動を変えない。
5. **保存**: 構造体の項目名は今のまま残してよい（中断セーブの互換のため）。新しい状態を足すときに表へ1行と `StatusAilments` の項目1つを足すだけで、付与・経過・HUD・保存・困った時のカードに反映される形にする。追加手順を完了報告に書く（wave8 の担当が読む）。
6. HUD の状態表示（`draw_hud.go`）も表から出す。

## 守ること

- **挙動を変えない**: 置き換えの前に今の重ねがけの挙動をテストで固定する（状態ごとに「2回かけたとき」「耐性」「治ったときのメッセージ」）。置き換えの後、`TestSimBatch -sim.runs=1500 -sim.difficulty=all` が `reports/baseline.json` と**完全一致**すること（乱数の消費を変えない）。一致しない場合は原因を突き止め、意図した変更なら完了報告に書く。
- 新しい関数は `game_stub.go` の最小 `Game` でもコンパイルが通るようにする（タグなしのコードから `Game` の新しい項目を使うなら stub にも足す）。
- 封印・金縛りなど bool の状態も同じ表で扱う（持続の型「解除されるまで」）。

## 完了条件

- gofmt、`go test -tags test ./...`、`go test ./...` が成功。
- `TestSimBatch`（1500冒険）がベースラインと一致。
- `grep -rn 'StatusAilments\.[A-Za-z]* *[-+]\?=' --include=*.go . | grep -v _test` で残る直接の書き換えが、共通処理の内部と理由のある箇所（完了報告に列挙）だけになっている。
- `docs/agents/done/wave7-A.md`: 変更概要、**新しい状態異常の追加手順**、IMPLEMENTED_FEATURES.md への追記案、AGENTS.md の「状態異常は残りターン…共通処理へまとめる」の削除案、未決事項。
- コミットしてブランチをプッシュする。

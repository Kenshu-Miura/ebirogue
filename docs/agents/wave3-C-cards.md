# wave3-C: カード A（アイテム ID 167〜173、Codex 用）

ブランチ: `agent/wave3-C-cards`
前提: master に共通基盤 F1〜F5（`agent/content-foundation`）が統合されていること。カードは既存の `img/card.png` を共用するので、画像 img-1 の統合は待たなくてよい。
最初に `AGENTS.md`（特に「新要素追加チェックリスト」、`itemRoles` の規則、「既存カード効果はランダム部分を `rollXxx(intn)` の純粋関数に分離」）、`docs/agents/README.md` の並列作業のルール、`docs/agents/STATUS.md`、`docs/agents/content-expansion-plan.md` を読む。`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない。大きいファイル（`itemeffects.go`・`content_cards_test.go`・`items.go`）は丸ごと読まず `grep` で該当箇所を探す。
並行して wave3-A（武器、`items_weapons.go`）と wave3-B（盾、`items_shields.go`）が動く。**このタスクで触る表は `items_cards.go` だけ。** `items.go` の `baseItemTemplates` には追記しない。

## 作るもの

7種。テンプレートは既存のカードと同じ形（`ItemType: "Card"`、`Type: "Card"`、`Char: 'C'`、`UseActions: map[string]UseAction{"UseCard": 効果関数}`）。効果関数は**新しいファイル `card_effects_wave3.go`（`//go:build !test`）**に書く（`itemeffects.go` には追記しない）。

| ID | 名前 | 効果 | BOT の分類（`itemRoles`） |
|---|---|---|---|
| 167 | ウインナー化のカード | 道具を1つ選び、ウインナー（`newTransformedSausage()`、ID 1 と同じ）に変える。装備中の道具・このカード自身は選べない（選ぶと「装備中の道具は変えられない」）。壺を選ぶと中身ごと変わる | `roleNone`（BOT は使わない） |
| 168 | 呪いのカード | 道具を1つ選び、呪う（既存の `isCurseableItem` / `curseItem`。武器・盾・矢・指輪だけ）。呪えない道具なら「何も起こらなかった」。装備能力で呪いを無効化できる（`g.playerImmunityMessage(StatusCurse)`）ときは防いだと表示して呪わない | `roleHarmful` |
| 169 | 罠消しのカード | フロアの罠（`g.state.MapTraps`）をすべて消す。「罠がn個消えた」（0個なら「何も起こらなかった」） | `roleBuff`（使う条件: 発見済みの罠が1つ以上ある） |
| 170 | 敵集合のカード | フロアの敵をプレイヤーの近くへ集める。もともとプレイヤーに近い敵から順に、プレイヤーに近い空き床（部屋の床か通路、敵・プレイヤー・階段の無いマス）へ置く。置き場所が足りなければ残りは動かない。ボスは動かさない | `roleHarmful` |
| 171 | 道具寄せのカード | フロアの床の道具を、プレイヤーの周囲の空き床（`g.itemDropFree`。床か通路、道具・罠の無いマス）へ近い順に集める。床に貼りついた聖域のカードと、敵の足元に擬態している化け貝（敵なので対象外）は動かさない。足元の道具も動かさない | `roleBuff`（使う条件: プレイヤーから3マスより遠い床の道具が1つ以上ある） |
| 172 | バクチのカード | 1/2 で所持金（`Player.Cash`）が「2倍＋100」、1/2 で0になる。判定は純粋関数 `rollGamble(cash int, intn func(int) int) (newCash int, won bool)` | `roleNone` |
| 173 | 困った時のカード | HP を最大 HP まで回復し（最大 HP は増えない）、困った状態を1つだけ解消する。順番: ①悪い状態異常（混乱・睡眠・目潰し・鈍足・口封じ・トラバサミ・金縛り・封印。どれか1つでもあればそれらを**すべて**解く）②空腹（満腹度0なら満腹度を50にする）③呪い（`uncursePlayerBelongings`。所持品と装備の呪いをすべて解く）④パワー低下（パワーが最大パワー未満なら最大まで戻す）。どれにも当たらなければ回復だけ | `roleHeal` |

174（ゾワゾワのカード）は段3の担当なので作らない。

## 仕様

- **道具を選ぶカード（167・168）の選択画面**: 真実の眼のカード（`identifyItem` → `g.useIdentifyItem = true` → 所持品の一覧で選ぶ → `executeItemIdentify`）と同じ流れにする。真実の眼の処理を壊さないよう、次のように一般化する。
  - `Game` に `itemTargetEffect func(g *Game, targetIndex int)`（選んだ道具に効果を与える処理。nil なら選択中でない）を追加する。セーブ不要（選択画面は中断できない状態ではないか、`input.go` の X キーで取り消せること）。
  - カードの効果関数は `g.tmpSelectedItemIndex`（使ったカードの添字）を設定して `g.itemTargetEffect` を入れ、所持品画面を開く。`input.go` の所持品の Z / X の分岐（`useIdentifyItem` / `usePotInsert` と並ぶ箇所）に `itemTargetEffect != nil` の分岐を足す。X で取り消すとカードは消費しない。
  - 選んだ後、カードを所持品から消す（真実の眼と同じく `tmpSelectedItemIndex` の位置。選んだ道具の添字がカードより後ろなら1つずれる点に注意）、効果のメッセージを Action で積む、`isActioned = true`。
  - 足元から使った場合（`GroundItemActioned`）も真実の眼と同じく扱う。
  - BOT の `bot_test.go` の `windowOpen` の条件（`g.useIdentifyItem || g.usePotInsert`）に `g.itemTargetEffect != nil` を足す（BOT が選択画面で止まらないように）。167・168 は BOT が使わない分類なので、BOT の選択処理は不要。
- 169〜173 は選択の要らない効果で、既存の `fullHealCard`・`increaseFloorTraps` と同じく `determineItemSource` → Action を積む → `removeUsedItem` の形にする。
- 敵集合・道具寄せの置き場所の計算は、マップと座標だけを受け取る純粋関数（タグなしのファイル、例: `card_gather.go`）にして、近い順（チェビシェフ距離、同じ距離なら上→下・左→右の順）に並べる。乱数は使わない。動かした後は `g.miniMapDirty = true`。敵を動かしたら `Animating = false`、`dx, dy = 0`。
- 困った時のカードの判定順は純粋関数（例: `troubleToFix(player Player) troubleKind`）に分けてテストする。
- 呪いのカードで指輪を呪うときは `curseItem` を通す（装備中の鼓舞の指輪の補正が付け直される）。
- 未識別の表示名・`N` の任意名・真実の眼の扱いは既存のカードと同じ（何もしなくてよいはずだが、テストで確かめる）。
- 床出現: カードカテゴリの合計の重み（`itemCategorySpawnWeights["Card"] = 31`）は変えない。7種を追加するとパワーアップのカードを含む既存のカードが少し出にくくなる（31のうち、カード内の重みの合計が 31→38 になる）。計画どおりなので重みは付けない。
- BOT（`bot_expert_test.go`）: `itemRoles` に表の分類を追加する。`roleBuff` の2枚は `expertBuffUsable` に使う条件を足す。
- 説明文（`Description`）は既存のカードと同じ調子で書く。
- ヘルプ（`help.go`）の道具（カード）のページに7枚を1行ずつ追記する。

## テスト（`content_cards_wave3_test.go`、`//go:build !test`）

`content_foundation_test.go` の補助（`newFoundationTestGame`・`runQueuedActions`）と `content_item_traps_test.go` の `newItemTrapTestGame` を使ってよい。

- 7種のテンプレートが存在し、名前・種類・`UseActions` が正しい。`itemRoles` に7種が登録されている。
- ウインナー化: 選んだ道具がウインナーになる／装備中は選べない／X で取り消すとカードが残る／カードより前の道具・後ろの道具を選んでも正しい道具が変わる／壺を選ぶと中身ごと消える／足元から使ったとき。
- 呪い: 武器・盾・矢・指輪が呪われる（装備中の鼓舞の指輪はパワーが −3 側へ付け直される）／食料などは何も起こらない／`statusImmunities` に仮の能力で `StatusCurse` を登録した指輪を装備すると防ぐ。
- 罠消し: 罠がすべて消える・個数のメッセージ／罠が無いとき。
- 敵集合: 近い順に置かれる・置き場所が足りないとき・ボスは動かない・壁や階段の上に置かない。
- 道具寄せ: 近い順に集まる・聖域のカード（貼りつき）と足元の道具は動かない・置き場所が足りないとき。
- バクチ: 勝ち（2倍＋100）・負け（0）・所持金0のときの勝ち（100）。
- 困った時: ①〜④の順に1つだけ解消される（複数の困った状態を同時に持たせて確かめる）・HP は最大まで回復し最大 HP は増えない・何も無いとき。
- セーブ→復元でカードが同じテンプレートに戻る。
- `go test -run TestSimSmoke -count=1 .` で違反0。

## 完了条件

- `gofmt -w`、`go test -tags test ./...`、`go test ./...`、`go test -run TestSimSmoke -count=1 .` がすべて成功する。
- `go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1500 -sim.difficulty=all -sim.out=reports` を1回実行し、`reports/baseline.json` との差分（餓死の数・床の道具の割合・expert のクリア率・「自分に不利なカードによる被害」）を完了報告に書く（ベースラインは更新しない）。
- `docs/agents/done/wave3-C.md` に、変更概要、IMPLEMENTED_FEATURES.md への追記案（カードの一覧）、仕様の判断、未決事項を書く。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

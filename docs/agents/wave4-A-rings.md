# wave4-A: 指輪 A（アイテム ID 175〜188・194〜199、Codex 用）

ブランチ: `agent/wave4-A-rings`
前提: master に共通基盤 F1〜F5（`equipment_hooks.go`・`statusImmunities`・`items_rings.go`）と段2の共通基盤（`agent/content-plan-2`。`expansionHelpPages`）が統合されていること。`help.go` に `expansionHelpPages` が無ければ作業を始めず、その旨を報告する。
最初に `AGENTS.md`（特に「ファイル分担」の `equipment_abilities.go` / `equipment_hooks.go`、「新要素追加チェックリスト」）、`docs/agents/README.md` の並列作業のルール、`docs/agents/STATUS.md`、`docs/agents/content-expansion-plan.md` を読む。`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない。大きいファイル（`itemeffects.go`・`items.go`・`item.go`）は丸ごと読まず `grep` で該当箇所を探す。
並行して wave4-B・wave4-C（敵、`enemies_expansion_a1.go`・`a2.go`）が動く。wave3-A/B/C（武器・盾・カード）がまだ統合されていない場合もある。

## 触るファイル（ほかの担当と衝突しないための約束）

| ファイル | やること |
|---|---|
| `items_rings.go` | 表 `ringItemTemplates` に20種を追加 |
| `ring_abilities.go`（新規、`//go:build !test`） | 能力 ID の定数、`registerAbilityHooks`・`statusImmunities` への登録、ヘルプのページ（`init()`） |
| `ring_logic.go`（新規、タグなし）と `ring_logic_test.go` | 乱数や数値の計算の純粋関数（満腹度の増減・湧き間隔・確率の判定など）とその単体テスト |
| `content_rings_wave4_test.go`（新規、`//go:build !test`） | 結合テスト |
| 下の「既存の処理へのつなぎ込み」の各ファイル | 表に書いた関数の中だけ、数行の追記 |
| `docs/agents/done/wave4-A.md`（新規） | 完了報告 |

`equipment_abilities.go` の定数ブロックと `statusImmunities` の初期値、`help.go` には追記しない（wave3 と衝突するため）。

## 作るもの

20種。すべて `ItemType: "Accessory"`、`Type: "Accessory"`、`Char: '!'`、`UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}`（既存の鼓舞の指輪 ID 10 と同じ形）。画像は既存の `img/ring.png` を共用する（`Type` が同じなので追加の登録は不要）。説明文（`Description`）は既存の鼓舞の指輪と同じ調子で、効果の数値を書く。

「重ね」は2つ装備したときの扱い。フックの引数 `stacks`（`g.playerAbilityCount`）で数える。

| ID | 名前 | 能力 ID（定数名 = 文字列） | 仕組み | 効果 | 重ね |
|---|---|---|---|---|---|
| 175 | 回復の指輪 | `AbilityRegeneration` = `"regeneration"` | `OnTurn` | 満腹度が1以上なら、毎ターン HP を追加で1回復（最大 HP まで、`recoveredValue`）。20ターンごと（`g.moveCount%20 == 0`）に満腹度が追加で1減る（0未満にしない） | 回復量・減る量が×2 |
| 176 | 胃拡張の指輪 | `AbilityStomachExpand` = `"stomach_expand"` | 装備時 | 装備すると最大満腹度+25、外すと−25 | 指輪ごと |
| 177 | 胃縮小の指輪 | `AbilityStomachShrink` = `"stomach_shrink"`、`AbilityAlwaysCursed` = `"always_cursed"` | 装備時 | 装備すると最大満腹度−25、外すと+25。装備したとき満腹度が最大を超えていれば最大まで減らす。**必ず呪われている** | 指輪ごと |
| 178 | しあわせの指輪 | `AbilityHappiness` = `"happiness"` | `OnTurn` | 毎ターン経験値+1（`checkLevelUp` も呼ぶ） | ×2 |
| 179 | 毒消しの指輪 | `AbilityPoisonWard` = `"poison_ward"` | `statusImmunities` → `StatusPoison` | 毒でパワーが下がらない（毒ヘビ系・毒矢の罠・腐ったウインナー） | — |
| 180 | 混乱よけの指輪 | `AbilityConfusionWard` = `"confusion_ward"` | → `StatusConfusion` | 混乱しない | — |
| 181 | 睡眠よけの指輪 | `AbilitySleepWard` = `"sleep_ward"` | → `StatusSleep` | 眠らない | — |
| 182 | 錆よけの指輪 | `AbilityRustWard` = `"rust_ward"` | → `StatusRust` | 装備中の武器・盾の強化値がさびで下がらない | — |
| 183 | 呪いよけの指輪 | `AbilityCurseWard` = `"curse_ward"` | → `StatusCurse` | 持ち物が呪われない（ノロイガニ系など） | — |
| 184 | 気配察知の指輪 | `AbilitySenseEnemies` = `"sense_enemies"` | `updateEnemyVisibility` | フロアの敵をすべてミニマップに表示する（擬態中の敵は表示しない。目潰し中は表示しない） | — |
| 185 | 道具感知の指輪 | `AbilitySenseItems` = `"sense_items"` | `updateItemVisibility` | フロアの床の道具をすべてミニマップに表示する（目潰し中は表示しない）。外すと、まだ見つけていない道具は見えなくなる（`PlayerDiscovered` は変えず `ShowOnMiniMap` だけを立てる） | — |
| 186 | 透視の指輪 | `AbilityClairvoyance` = `"clairvoyance"` | 上の2つ | 敵と道具の両方を表示する（気配察知と道具感知を合わせた効果） | — |
| 187 | 鑑定師の指輪 | `AbilityAppraiser` = `"appraiser"` | `PickUpItem` | 拾った道具が未識別なら識別する（`Identifiable` の `SetIdentified(true)`、「〜だった。」と表示） | — |
| 188 | 浮遊の指輪 | `AbilityLevitation` = `"levitation"` | `checkForTrapAtPosition` | 罠を踏まない。未発見の罠の上に乗ったら発見済みにして「〜の上を浮いて通り過ぎた。」と表示する（発見済みなら何も表示しない） | — |
| 194 | 高飛びの指輪 | `AbilityHighJump` = `"high_jump"` | `OnTurn` | 毎ターン 2%（`g.abilityIntn()(100) < 2*stacks`）で「海老さんは高く飛び上がった！」と表示し、ワープの罠と同じ処理（`warpTrapEffect(g)`、`move_traps.go`）でフロアのどこかへ移る | 確率×2 |
| 195 | 爆発の指輪 | `AbilityExplosionRing` = `"explosion_ring"` | `OnTurn` | 毎ターン 1/200（`g.abilityIntn()(200) < stacks`）で大爆発。周囲1マスの敵はボス以外すべて倒れ（`g.applyDamageToEnemy(index, 敵の HP)`、経験値は入る）、ボスは50ダメージ。自分の HP は1になる（`g.damagePlayer(HP-1, DamageSource{Kind: DamageSourceSelfDestruct, Detail: "爆発の指輪"})`。HP が1なら減らない）。状態の変更は Action の `Execute` 内で行う | 確率×2 |
| 196 | 垂れ流しの指輪 | `AbilityLeakItems` = `"leak_items"`、`AbilityAlwaysCursed` | `OnTurn` | 毎ターン 3%（`< 3*stacks`）で、装備していない持ち物を1つ、向いている方向の1〜2マス先（置けなければ周囲8マス）へ落とす（`g.dropPlayerItemsForward(1, 向きの dx, dy)`、`displacement.go`。抽選は `g.displacementIntn()` なのでテストでは `randHooks.displacement` も固定する）。**必ず呪われている** | 確率×2 |
| 197 | 小銭垂れ流しの指輪 | `AbilityLeakMoney` = `"leak_money"`、`AbilityAlwaysCursed` | `OnTurn` | 所持金があれば毎ターン 10%（`< 10*stacks`）で、`max(1, 所持金/50)` を小銭として足元に落とす（`g.itemDropFree(x, y)` が false なら落とさない。小銭は `buildItemFromTemplate(0, x, y)` の `Amount` を書き換えて `g.state.Items` へ）。**必ず呪われている** | 確率×2 |
| 198 | 魔物呼びの指輪 | `AbilityMonsterCall` = `"monster_call"` | `SetNextSpawnInterval` | 敵が湧く間隔（20〜30ターン）を `1/(1+重ね数)` にする（最短5ターン）。純粋関数 `ringSpawnInterval(base, stacks int) int` を `ring_logic.go` に置く | ×2 |
| 199 | 罠増しの指輪 | `AbilityTrapIncrease` = `"trap_increase"` | `OnTurn` | 50ターンごと（`g.moveCount%50 == 0`）に、フロアの空いた部屋の床（`Type == "floor"`、罠・道具・階段・プレイヤーが無い）へ隠れた罠を1つ置く。罠の種類は敵の罠設置と同じ抽選（`pickTemplateID(availableTrapIDs(randomMapTrapIDs, g.Floor), g.abilityIntn(), 0)` と `createMapTrapByID`） | 個数×2 |

189〜193（遠投・ヘタ投げ・百発百中・反射・連射）は段3へ回した（下の「段3へ回したもの」）。200〜204 も段3。

## 仕様

- `AbilityAlwaysCursed` は「生成時に必ず呪われている」印の能力 ID（効果のフックは無い）。`items.go` の `rollItemFromTemplate` の `"Accessory"` の分岐で `Cursed: hasEquipmentAbility(template.Abilities, AbilityAlwaysCursed)` にする（今は常に false）。ほかの指輪は今までどおり生成時には呪われていない。呪われた指輪は既存の処理（`isCursedEquipment`）で外せない。おはらいのカードで呪いが解ければ外せる。
- 胃拡張・胃縮小は鼓舞の指輪と同じく `interfaces.go` の `Accessory.UpdatePlayerStats` で最大満腹度を増減する。増減量は純粋関数 `accessorySatietyBonus(abilities []EquipmentAbilityID) int`（`ring_logic.go`）で、胃拡張+25・胃縮小−25（両方の能力を持つ指輪は無い）。最大満腹度は1未満にしない。満腹度は最大満腹度を超えたら切り詰める。HUD の満腹度の表示（`draw_hud.go`）が最大満腹度に追従することを確かめる。
- 状態異常の無効化（179〜183）は `statusImmunities[能力] = []StatusKind{…}` を `ring_abilities.go` の `init()` で登録するだけ。付与箇所はすべて `g.playerImmuneTo` / `g.playerStatusGuard` を経由しているので、メッセージ（「〜の指輪が混乱を防いだ」）も出る（共通基盤で配線済み）。
- 気配察知・道具感知・透視は、`enemies.go` の `updateEnemyVisibility` と `item.go` の `updateItemVisibility` の分岐に「指輪があれば表示」を1つ足す。判定は `g.playerHasAbility(AbilitySenseEnemies) || g.playerHasAbility(AbilityClairvoyance)` など。
- 鑑定師は `item.go` の `PickUpItem` で、所持品に加えた直後に識別する（壺の中身は識別しない）。お金は対象外。
- 浮遊は `maptraps.go` の `checkForTrapAtPosition` の先頭で判定する（`stepOnTrap` は変えない）。
- 魔物呼びは `monster_spawn.go` の `SetNextSpawnInterval` で `ringSpawnInterval` を通す。乱数の使い方（`g.randIntn` の回数）は変えない。
- `OnTurn` のフックは `IncrementMoveCount` の最後（満腹度の減少・自然回復の後）に毎ターン呼ばれる。フックの中の乱数は `g.abilityIntn()`（テストで `g.randHooks.ability` で固定できる）。メッセージは `g.EnqueueMessage`、状態を大きく変える処理（ワープ・爆発・持ち物を落とす）は Action を積んで `Execute` 内で行う。
- 床出現: `items.go` の `itemCategorySpawnWeights["Accessory"]` を **1 → 3** にする（今は指輪が鼓舞の指輪1種なので重み1。21種に増えても指輪全体が約1%しか出ないと、ほとんどの指輪を見かけないため）。カテゴリ内の重み（`itemSpawnWeights`）は付けない。変更の前後で `TestSimBatch` の「床の道具」と餓死の数を比べ、完了報告に書く（数値は開発者が変えてよい）。
- 識別・任意名・真実の眼の扱いは既存の指輪と同じ（未識別の表示名は既存の `Accessory` の処理に任せる）。
- BOT（`bot_expert_test.go`）は指輪を装備しない（今のまま）。BOT の変更は不要。

## 既存の処理へのつなぎ込み（各1〜数行）

| ファイル | 関数 | 追記 |
|---|---|---|
| `items.go` | `rollItemFromTemplate` の `"Accessory"`、`itemCategorySpawnWeights` | 必ず呪われている印、指輪の重み 1→3 |
| `interfaces.go` | `Accessory.UpdatePlayerStats` | 最大満腹度の増減 |
| `enemies.go` | `updateEnemyVisibility` | 気配察知・透視 |
| `item.go` | `updateItemVisibility`、`PickUpItem` | 道具感知・透視、鑑定師 |
| `maptraps.go` | `checkForTrapAtPosition` | 浮遊 |
| `monster_spawn.go` | `SetNextSpawnInterval` | 魔物呼び |

## ヘルプ

`ring_abilities.go` の `init()` で、新しいページを1枚加える（`help.go` は編集しない）。

```go
expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "指輪", Lines: []string{ … }})
```

1種1行（例:「回復の指輪：自然回復が増えるが、腹が減りやすい」）。20種で15行を超える場合は2ページ（「指輪（1）」「指輪（2）」）に分ける（1ページの表示は `helpPageSize` = 15行、スクロールできる）。

## テスト

`ring_logic_test.go`（タグなし）: `accessorySatietyBonus`、`ringSpawnInterval`（重ね0・1・2、最短5）、そのほか切り出した純粋関数。

`content_rings_wave4_test.go`（`//go:build !test`）: `content_foundation_test.go` の補助（`newFoundationTestGame`・`equip`・`testRing`）と `content_poison_test.go` の `runQueuedActions` を使ってよい。乱数は `g.randHooks.ability` で固定する。

- 20種のテンプレートが存在し、名前・能力 ID が表どおり。177・196・197 は生成すると必ず呪われていて、ほかは呪われていない（`rollItemFromTemplate` を複数の乱数で）。
- 能力ごとに**正常系と、効かない条件**（指輪を外す・別の指輪）:
  - 回復: 満腹度があると毎ターン HP+1 多い／満腹度0では増えない／20ターン目に満腹度が追加で1減る／2つで+2。
  - 胃拡張・胃縮小: 装備で最大満腹度±25、外すと戻る／胃縮小で満腹度が最大を超えたら切り詰める／胃縮小は呪われていて外せない。
  - しあわせ: 毎ターン経験値+1、レベルが上がる境界。
  - 状態異常の5種: 対応する状態異常（混乱の罠・睡眠・毒ヘビの毒・さびの罠・ノロイガニの呪い）を防ぎ、別の状態異常は防がない。
  - 気配察知・道具感知・透視: 別の部屋の未発見の敵・道具が `ShowOnMiniMap` になる／擬態中の敵は表示しない／目潰し中は表示しない／外すと未発見の道具は消える。
  - 鑑定師: 未識別の杖を拾うと識別される／指輪が無ければ識別されない。
  - 浮遊: 罠のマスに乗っても効果が起きず、未発見なら発見済みになる。
  - 高飛び: 確率に当たるとワープする、外れると動かない。
  - 爆発: 当たると周囲1マスの敵が倒れ、2マス先の敵は残り、ボスは50ダメージ、HP が1になる／HP が1なら死なない。
  - 垂れ流し・小銭垂れ流し: 当たると持ち物が前方に・小銭が足元に落ちる、装備中の物は落ちない、所持金0なら何もしない。
  - 魔物呼び: 装備中は湧き間隔が短くなる。
  - 罠増し: 50ターン目に罠が1つ増え、49ターン目には増えない。
- セーブ: 20種それぞれ装備したまま `buildSaveData` → `applySaveData` で能力 ID・呪い・最大満腹度が保たれる。
- `go test -run TestSimSmoke -count=1 .` で違反0。

## 段3へ回したもの

| ID | 指輪 | 理由 |
|---|---|---|
| 189 遠投・190 ヘタ投げ・191 百発百中・192 反射・193 連射 | 投擲・射撃 | このゲームの投げた道具と矢は必ず命中し（命中率が無い）、`ThrowItem`（`item.go`）は飛ぶ途中の処理を差し込む口が無い。壁と敵の貫通・跳ね返り・複数本の発射・命中率を入れるには、投擲の軌道と命中の処理を共通化する作業（段3の共通基盤）が要る |

## 完了条件

- `gofmt -w`、`go test -tags test ./...`、`go test ./...`、`go test -run TestSimSmoke -count=1 .` がすべて成功する。
- `go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1500 -sim.difficulty=all -sim.out=reports` を1回実行し、`reports/baseline.json` との差分（床の道具の割合・餓死の数・expert のクリア率）を完了報告に書く（ベースラインは更新しない）。
- `docs/agents/done/wave4-A.md` に、変更概要、IMPLEMENTED_FEATURES.md への追記案（指輪の一覧）、仕様の判断、未決事項（確率・数値の妥当性）を書く。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

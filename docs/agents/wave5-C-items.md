# wave5-C: アイテム B（アイテム ID 147・163〜166・174・189〜193・200〜202・204、Codex 用）

ブランチ: `agent/wave5-C-items`
前提: master に段3の共通基盤（`agent/content-plan-3`。`projectile.go`・`equipment_harm.go`・敵の状態ゾワゾワ）、段2の指輪 A（wave4-A。`items_rings.go` の20種と `AbilityAlwaysCursed`）、画像 img-2 が統合されていること。`projectile.go` や `img/giant_harisen.png` が master に無い、または `AbilityAlwaysCursed` が定義されていなければ作業を始めず、その旨を報告する。
最初に `AGENTS.md`（特に「ファイル分担」の `equipment_abilities.go` / `equipment_hooks.go` / `displacement.go`、「新要素追加チェックリスト」）、`docs/agents/README.md` の並列作業のルール、`docs/agents/STATUS.md`、`docs/agents/content-expansion-plan.md`、`projectile.go` の冒頭のコメントを読む。`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない。大きいファイル（`itemeffects.go`・`items.go`・`item.go`）は丸ごと読まず `grep` で該当箇所を探す。
並行して wave5-A・wave5-B（敵、`enemies_expansion_b1.go`・`b2.go`）が動く。

## 触るファイル（ほかの担当と衝突しないための約束）

| ファイル | やること |
|---|---|
| `items_weapons.go`・`items_shields.go`・`items_cards.go`・`items_rings.go` | それぞれの表に追加（147／163〜166／174／189〜193・200〜202・204） |
| `item_abilities_stage3.go`（新規、`//go:build !test`） | 能力 ID の定数、`registerAbilityHooks`・`statusImmunities` への登録、ゾワゾワのカードの効果、ヘルプのページ（`init()`） |
| `main.go` | `specialWeaponImagePaths` に 147、`specialArmorImagePaths` に 163〜166 の画像を1行ずつ追記 |
| `bot_expert_test.go` | `itemRoles` に 174 を1つ追記 |
| `content_items_wave5c_test.go`（新規、`//go:build !test`） | テスト |
| `docs/agents/done/wave5-C.md`（新規） | 完了報告 |

`equipment_abilities.go` の定数ブロックと `statusImmunities` の初期値、`help.go`、`projectile.go`、`equipment_harm.go`、`enemy_behaviors.go` には追記しない。共通基盤に足りないものがあれば、自分で共通のファイルを直さず、完了報告に書く。

## 作るもの

16種。画像は img-2 の `img/<ファイル>.png`（`Type` を `main.go` の画像の表に追記）。指輪は既存の `img/ring.png`、カードは `img/card.png` を共用する（`Type: "Accessory"`・`"Card"` なので追加の登録は不要）。指輪とカードのテンプレートの形は同じ表の既存の行（wave4-A・wave3-C）に合わせる。説明文（`Description`）は既存の行と同じ調子で、効果の数値を書く。

「重ね」は指輪を2つ装備したときの扱い。フックの引数 `stacks`（`g.playerAbilityCount`）で数える。

### 武器・盾・カード

| ID | 名前 | Type / 画像 | 値 | 能力 ID（定数名 = 文字列） | 仕組み | 効果 |
|---|---|---|---|---|---|---|
| 147 | 巨大ハリセン | GiantHarisen / giant_harisen | 攻撃 4 | `AbilityGiantFan` = `"giant_fan"` | `AfterPlayerHit` | 敵を倒さなかったとき 25%（`g.abilityIntn()(100) < 25`）で、その敵を海老さんから離れる向きへ最大10マス吹き飛ばす（`g.knockbackEnemy(添字, dx, dy, 10, 5)`。壁や敵にぶつかると5ダメージ。ボスは動かない）。添字は `g.findEnemyForDefense(hit.EnemyID, hit.Enemy.X, hit.Enemy.Y)` で引き直す |
| 163 | 痛恨よけの盾 | CriticalGuardShield / critical_guard_shield | 防御 5 | `AbilityCriticalGuard` = `"critical_guard"` | `ModifyIncomingDamage` | 敵の痛恨の一撃（`incoming.Critical`）のダメージを1/3にする（最低1） |
| 164 | いたずらよけの盾 | MischiefGuardShield / mischief_guard_shield | 防御 4 | `AbilityTossWard` = `"toss_ward"` | `statusImmunities` → `StatusItemToss` | 持ち物を投げ捨てられない（イタズラカワハギ系、wave5-B） |
| 165 | 不動の盾 | ImmovableShield / immovable_shield | 防御 4 | `AbilityImmovable`（既存） | なし（F3 で配線済み） | 吹き飛ばし・引き寄せ・場所替えを受けない |
| 166 | ざぶとん | Zabuton / zabuton | 防御 5 | `AbilityBreakFall`（既存） | なし（F3 で配線済み） | 吹き飛ばされてぶつかってもダメージを受けない |
| 174 | ゾワゾワのカード | Card | — | — | カードの効果 | 周囲8マスの敵（ボスを含む）を20ターンのゾワゾワにする（`StatusAilments.Backward = 20`。ボスは行動時に `normalizeBossStatus` が短くする）。ゾワゾワの敵は海老さんを攻撃するとき反対の向きへ攻撃し、そこにいる別の敵に当たる（特殊攻撃・痛恨も出ない。共通基盤の `enemyAttackBackward`）。1体以上なら「〜はゾワゾワした！」（敵ごと、目潰し中は `g.enemyDisplayName`）、いなければ「しかし何も起こらなかった」。効果関数の形は金縛りのカード（`itemeffects.go` の「金縛り効果関数」）と同じく `determineItemSource` → 効果の Action → `removeUsedItem` |

### 指輪

すべて `ItemType: "Accessory"`、`Type: "Accessory"`、`Char: '!'`、`UseActions: map[string]UseAction{"AccessoryEffect": func(g *Game) {}}`（wave4-A の行と同じ形）。

| ID | 名前 | 能力 ID（定数名 = 文字列） | 仕組み | 効果 | 重ね |
|---|---|---|---|---|---|
| 189 | 遠投の指輪 | `AbilityLongThrow` = `"long_throw"` | `ModifyProjectile` | 投げた道具・撃った矢（`p.Kind` が `ProjectileThrow` か `ProjectileShot`）が壁と敵を通り抜ける（`p.PierceWalls = true`、`p.PierceTargets = true`）。射程（10マス）の端まで飛び、道中の敵すべてに当たる。1体にでも当たれば道具は消え、当たらなければ最後に通った床に落ちる（共通基盤）。杖の魔法弾（`ProjectileBolt`）は変えない | — |
| 190 | ヘタ投げの指輪 | `AbilityPoorThrow` = `"poor_throw"`、`AbilityAlwaysCursed` | `ModifyProjectile` | 投げた道具・撃った矢が必ず外れる（`p.HitPercent = 0`。外れた道具は敵の後ろへ飛び続ける）。**必ず呪われている** | — |
| 191 | 百発百中の指輪 | `AbilitySureThrow` = `"sure_throw"` | `ModifyProjectile` | 投げた道具・撃った矢が必ず当たる（`p.SureHit = true`）。ヘタ投げより優先し、`BeforeHit` で外れにする敵（段4の打ち返す敵など）にも当たる | — |
| 192 | 反射の指輪 | `AbilityBounceThrow` = `"bounce_throw"` | `ModifyProjectile` | 投げた道具・撃った矢が壁で跳ね返る（`p.Bounces += 3 * stacks`）。跳ね返りは射程を使わない。戻ってきても海老さんには当たらず素通りする（共通基盤）。遠投と両方あるときは跳ね返りが先 | 回数×2 |
| 193 | 連射の指輪 | `AbilityRapidShot` = `"rapid_shot"` | `ModifyProjectile` | 矢を撃つ（`ProjectileShot`）と、矢が残っている限り一度に `1 + stacks` 本撃つ（`p.Volley = 1 + stacks`。`shootArrow` が1本ずつ消費して続けて撃つ）。投げたときは変えない | 3本 |
| 200 | 弾きよけの指輪 | `AbilityKnockOffWard` = `"knock_off_ward"` | `statusImmunities` → `StatusKnockOff` | 装備を弾き飛ばされない（ハジキガニ系、wave5-A） | — |
| 201 | 痛恨の指輪 | `AbilityCriticalCurse` = `"critical_curse"`、`AbilityAlwaysCursed` | `BeforeEnemyAttack` | 敵の通常攻撃の痛恨の確率を +10%（`attack.CriticalPercent += 10 * stacks`。痛恨を持たない敵も出すようになる）。**必ず呪われている** | +20% |
| 202 | 諸刃の指輪 | `AbilityDoubleEdge` = `"double_edge"` | `BeforePlayerAttack`・`BeforeEnemyAttack` | 自分の通常攻撃の会心が出やすくなる（`a.CriticalDenominator = min(a.CriticalDenominator, 8 / stacks)`。1つで 1/8、2つで 1/4）が、敵の痛恨の確率も +10%（`10 * stacks`） | 上のとおり |
| 204 | 不動の指輪 | `AbilityImmovable`（既存） | なし（F3 で配線済み） | 吹き飛ばし・引き寄せ・場所替えを受けない（不動の盾と同じ） | — |

## 仕様

- `ModifyProjectile`（`equipment_hooks.go`）は海老さんが道具を投げる・矢を撃つ・杖を振るとき、飛ぶ前に1回呼ばれる。`p.Kind` で種類を見分ける。飛び方の項目（`PierceWalls`・`PierceTargets`・`Bounces`・`HitPercent`・`SureHit`・`Volley`）の意味は `projectile.go` の `Projectile` のコメントのとおり。命中率が100%以上なら乱数を使わない（今の BOT の結果を変えない）。
- 投げた道具と矢は今も「必ず当たる」（命中率100%）。百発百中の指輪はヘタ投げの指輪と、`BeforeHit` で外れにする敵に対してだけ意味がある。投擲の基本の命中率を下げるかどうかは**開発者が決める**未決事項（完了報告に「百発百中の価値が小さい」ことを書く）。
- 外れたときは共通処理が「〜は〜に当たらなかった」と表示する。
- 弾きよけ・いたずらよけは `statusImmunities` に登録するだけで、共通処理（`knockOffPlayerEquipment`・`tossPlayerItem`）が「〜が〜を防いだ」と表示して防ぐ。
- 190・201 は生成時に必ず呪われている（wave4-A の `AbilityAlwaysCursed` の仕組み）。
- 巨大ハリセンの吹き飛ばしは、命中した通常攻撃のダメージの後（`AfterPlayerHit` は Action の `Execute` 内）に起きる。三方向攻撃・連撃の武器とは合成できないので考えなくてよい。
- ゾワゾワの残りターンは敵の状態異常として毎ターン減り（`IncrementMoveCount`）、中断セーブに残る（`StatusAilments`。共通基盤）。
- 床出現: 新しいアイテムはそれぞれのカテゴリの中で抽選される（F5）。カテゴリの重み（`itemCategorySpawnWeights`）とカテゴリ内の重み（`itemSpawnWeights`）は変えない。
- 識別・任意名・真実の眼の扱いは既存の同じ種類の道具と同じ（未識別の表示名は既存の処理に任せる）。
- BOT: `itemRoles` に `174: roleEscape`（隣の敵が自分を攻撃しなくなる）を追記する。BOT は指輪を装備しない（今のまま）。

## ヘルプ

`item_abilities_stage3.go` の `init()` で、新しいページを2枚加える（`help.go` は編集しない）。

```go
expansionHelpPages = append(expansionHelpPages,
	HelpPage{Title: "指輪：飛び道具・痛恨", Lines: []string{ … }},
	HelpPage{Title: "道具：追加の盾・武器・カード", Lines: []string{ … }},
)
```

1種1行（例:「遠投の指輪：投げた物・矢が壁と敵を突き抜ける」「ゾワゾワのカード：周りの敵が反対を向いて攻撃する」）。1ページは15行まで（`helpPageSize`）。

## テスト（`content_items_wave5c_test.go`、`//go:build !test`）

`content_foundation_test.go` の `newFoundationTestGame`・`equip`・`testRing`・`testArmor`・`withTestImmunity`、`content_poison_test.go` の `runQueuedActions`、`content_projectile_test.go` の `newProjectileTestGame`・`finishThrow`・`floorItemsAt`、`content_enemy_foundation_test.go` の `addTestEnemy` を使ってよい。指輪は `buildItemFromTemplate(ID, 0, 0)` で作って `equip` する。乱数は `g.randHooks.ability`・`projectile`・`enemyCritical`・`damage` で固定する。

- 16種のテンプレートが存在し、名前・攻撃力/防御力・能力 ID が表どおり。190・201 は生成すると必ず呪われていて、ほかの指輪は呪われていない。147・163〜166 の画像が登録されている（`specialWeaponImagePaths`・`specialArmorImagePaths`）。
- 能力ごとに**正常系と、効かない条件**（外す・別の指輪・種類が違う）:
  - 遠投: 壁の向こうの床に落ちる／一直線の敵2体に当たって道具が消える／杖の魔法弾は壁で止まる。
  - ヘタ投げ: 隣の敵にも当たらず後ろへ飛ぶ／矢も外れる。百発百中と両方なら当たる。
  - 反射: 壁で跳ね返って後ろの敵に当たる／海老さんには当たらない／2つで6回まで跳ね返る。
  - 連射: 矢が2本減って2回ダメージ／2つで3本／残り1本なら1本だけ／投げたときは1つ。
  - 弾きよけ・いたずらよけ: `g.knockOffPlayerEquipment`・`g.tossPlayerItem` が何もせず「防いだ」と表示する。
  - 痛恨・諸刃: `randHooks.enemyCritical` を 9 にすると痛恨持ちでない敵も痛恨、10 で通常（2つなら 19/20）／諸刃は `randHooks.damage` で会心の分母が8（2つで4）になる。
  - 痛恨よけの盾: 痛恨のダメージが1/3（最低1）、痛恨でない攻撃は変わらない。
  - 不動の盾・不動の指輪・ざぶとん: `g.knockbackPlayer` で動かない／ぶつかってもダメージを受けない。
  - 巨大ハリセン: `randHooks.ability` が 24 で敵が吹き飛ぶ、25 で吹き飛ばない／倒したときは吹き飛ばさない／ボスは動かない。
  - ゾワゾワのカード: 周囲8マスの敵が `Backward = 20`、2マス先の敵は変わらない／ゾワゾワの敵が海老さんの隣にいても海老さんは攻撃されず、反対側の敵が攻撃される／周りに敵がいなければ「何も起こらなかった」／カードが消費される。
- セーブ: 16種それぞれ持ったまま（指輪・盾・武器は装備したまま）`buildSaveData` → `applySaveData` で能力 ID・呪い・強化値が保たれる。ゾワゾワの敵の残りターンも保たれる。
- `go test -run TestSimSmoke -count=1 .` で違反0。

## 完了条件

- `gofmt -w`、`go test -tags test ./...`、`go test ./...`、`go test -run TestSimSmoke -count=1 .` がすべて成功する。
- `go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1500 -sim.difficulty=all -sim.out=reports` を1回実行し、`reports/baseline.json` との差分（床の道具の割合・餓死の数・expert のクリア率）を完了報告に書く（ベースラインは更新しない）。
- `docs/agents/done/wave5-C.md` に、変更概要、IMPLEMENTED_FEATURES.md への追記案（アイテムの一覧と効果）、仕様の判断、未決事項（投擲の基本の命中率・確率・数値の妥当性・ゲーム内で確認してほしい見た目〔跳ね返った道具の飛ぶ様子〕）を書く。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

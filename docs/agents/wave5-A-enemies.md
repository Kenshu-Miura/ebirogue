# wave5-A: 敵 B 前半（敵 ID 90〜98、Codex 用）

ブランチ: `agent/wave5-A-enemies`
前提: master に段3の共通基盤（`agent/content-plan-3`。`projectile.go`・`equipment_harm.go`・`enemies_expansion_b1.go`）、段2（wave4-A/B/C）、画像 img-2 が統合されていること。`enemies_expansion_b1.go` や `img/umi_dango.png` が master に無ければ作業を始めず、その旨を報告する。
最初に `AGENTS.md`（特に「新要素追加チェックリスト」の敵の規則と「ファイル分担」の `enemy_behaviors.go`・`displacement.go`・`equipment_harm.go`）、`docs/agents/README.md` の並列作業のルール、`docs/agents/STATUS.md`、`docs/agents/content-expansion-plan.md` を読む。`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない。
並行して wave5-B（敵 B 後半、`enemies_expansion_b2.go`）と wave5-C（アイテム B）が動く。

## 触るファイル（ほかの担当と衝突しないための約束）

| ファイル | やること |
|---|---|
| `enemies_expansion_b1.go` | 表 `expansionB1MonsterFamilies` への3系統の追加と、特殊攻撃・行動の関数。**このタスクのコードはすべてここに書く** |
| `content_enemies_wave5a_test.go`（新規） | テスト |
| `docs/agents/done/wave5-A.md`（新規） | 完了報告 |

ほかのファイル（`enemies.go`・`enemy_expansion.go`・`monster_spawn.go`・`main.go`・`draw.go`・`help.go`・`enemy_ai.go`・`enemy_behaviors.go`・`displacement.go`・`equipment_harm.go`）は編集しない。定義・上位種化・画像・出現階は表に書けば `enemy_expansion.go` が登録する（登録の1行は共通基盤で追加済み）。ヘルプは同じファイルの `init()` で `expansionHelpPages` にページを加える。
共通基盤に足りないものがあれば、自分で共通のファイルを直さず、完了報告に書く（このファイル内で閉じた補助関数を作るのはよい）。

## 作るもの

3系統9体。上位種は基本種の固有能力（特殊攻撃・確率・行動・属性・痛恨・回避）をそのまま引き継ぐ（`monsterStage` には能力値と表示だけを書く）。画像はすべて img-2 で作成済み。`Char` は既存の敵と重ならない小文字にした（画像があるので画面には出ない）。

| ID | 名前 | Type | 画像（`img/<>.png`） | Char | HP | 攻撃 | 防御 | 経験値 | 出現階（重み） |
|---|---|---|---|---|---|---|---|---|---|
| 90 | ウミダンゴ | UmiDango | umi_dango | d | 34 | 6 | 6 | 20 | 4〜10（6） |
| 91 | オオウミダンゴ | OoUmiDango | oo_umi_dango | d | 60 | 11 | 9 | 50 | 11〜18（6） |
| 92 | ダイオウウミダンゴ | DaiouUmiDango | daiou_umi_dango | d | 102 | 22 | 11 | 125 | 19〜30（6） |
| 93 | カエルアンコウ | KaeruAnkou | kaeru_ankou | f | 26 | 5 | 4 | 14 | 3〜9（6） |
| 94 | オオカエルアンコウ | OoKaeruAnkou | oo_kaeru_ankou | f | 48 | 10 | 7 | 38 | 10〜17（6） |
| 95 | ヌシカエルアンコウ | NushiKaeruAnkou | nushi_kaeru_ankou | f | 82 | 20 | 9 | 95 | 18〜30（6） |
| 96 | ハジキガニ | HajikiGani | hajiki_gani | h | 40 | 9 | 8 | 28 | 7〜13（6） |
| 97 | ケンゴウガニ | KengouGani | kengou_gani | h | 70 | 15 | 11 | 70 | 14〜21（6） |
| 98 | ケンセイガニ | KenseiGani | kensei_gani | h | 119 | 30 | 13 | 175 | 22〜30（6） |

- 能力値の基準: 3〜8階のイシガニ 32/7/5/15・ノロイガニ 38/8/6/22、第2段階のガンセキガニ 60/12/9/44、第3段階は第2段階の HP×1.7・攻撃×2・防御+2・経験値×2.5（`balance-level99.md`）。段2の同じ出現帯の敵（wave4-B/C）とも比べ、明らかに外れていれば直して完了報告に書く。
- 出現階の重みの目安: 同じ階の既存の敵は5〜10。第3段階は9階以降だけ（`TestDeepFloorSpawnTables`）。
- 属性（`Traits`）: ハジキガニ系に `EnemyTraitShell`（甲殻）。ほかは付けない。

### 固有の能力

すべて状態の変更（ダメージ・吹き飛ばし・引き寄せ・装備を弾く）は Action の `Execute` 内で行う（`AGENTS.md` の「落とし穴」）。`SpecialAttack` は `func(e *Enemy, g *Game)` で添字を受け取らないので、`enqueueEnemyFixedAttack` に渡す添字は `g.findEnemyForDefense(e.ID, e.X, e.Y)` で引く。向き `(dx, dy)` は敵から海老さんへの向き（`sign(海老さん − 敵)`）。

| 系統 | 設定する項目 | 能力 |
|---|---|---|
| ウミダンゴ系 | `SpecialAttack`、`SpecialAttackProbability: 0.25` | 隣接して攻撃するとき25%で、丸まって体当たりする。防御力で減らない固定ダメージ（90: 6、91: 11、92: 20。ID ごとの表 `umiDangoRamDamage map[int]int`）を `g.enqueueEnemyFixedAttack(i, damage, EnemyDamageNormal, EnemyAttackMelee, "〜が丸まって体当たりしてきた。", "体当たり", onHit)` で与え、当たったとき（`onHit`）に海老さんを向き `(dx, dy)` へ2マス吹き飛ばし（`g.knockbackPlayer(dx, dy, 2, 5, 死因)`。壁や敵にぶつかると5ダメージ、受け身で防ぐ、不動なら動かない）、続けて持ち物を1〜2個（`1 + g.displacementIntn()(2)`）同じ向きへ落とす（`g.dropPlayerItemsForward(個数, dx, dy)`。装備中は落ちない）。死因は「ウミダンゴの体当たり」（`enemyRangedSource(e.ID, e.Name, "体当たり")`） |
| カエルアンコウ系 | `Behavior` | 海老さんを発見済みで、`g.straightLineToPlayer(i, 3)` が成り立ち距離が2以上なら、50%（`g.randIntn(2) == 0`）で疑似餌を伸ばして引き寄せる（「〜が疑似餌を伸ばした！」の Action の `Execute` で `g.pullPlayer(-dx, -dy, 距離-1)`）。引き寄せは自分の隣で止まり、途中の罠を踏む（共通基盤）。不動なら「海老さんは踏みとどまった」。引き寄せたら `true`。隣にいる・直線上にいない・確率に外れたら `false`（通常の行動） |
| ハジキガニ系 | `SpecialAttack`、`SpecialAttackProbability: 0.25` | 隣接して攻撃するとき25%で、海老さんが盾を装備していれば、鋏で盾を弾き飛ばす（「〜が盾を鋏ではじいた！」の Action の `Execute` で `g.knockOffPlayerEquipment(盾, dx, dy, 4)`。海老さんの後ろへ4マス飛んで落ちる。呪われていても外れる。弾きよけの指輪〔wave5-C の 200〕は `StatusKnockOff` の無効化で防ぐ）。盾が無ければ通常攻撃（`g.enqueueEnemyNormalAttack(i)`） |

- 封印中は `Behavior` も `SpecialAttack` も呼ばれない（共通基盤）。睡眠・金縛り・混乱・目潰し・擬態の処理は `Behavior` より先に行われる。ゾワゾワ（wave5-C のカード）の敵は `AttackFromEnemy` が反対向きの攻撃に差し替える（特殊攻撃も出ない、共通基盤）。
- `Behavior`・`SpecialAttack` の関数が `MonsterDefinitions` や `CreateEnemyByID` を（`enemyRangedSource` などほかの関数経由でも）参照すると、表の初期化が循環してコンパイルエラー（initialization cycle）になる。そのときは表には書かず、`init()` で `updateMonsterDefinitions(func(d *MonsterDefinition) { d.SpecialAttack = … }, 90, 91, 92)` のように3段階すべてに設定する（`bosses.go` の `init` と同じ）。
- 乱数は敵の行動の判定に `g.randIntn`、持ち物を落とす個数に `g.displacementIntn()`（テストで `randHooks.displacement` を固定できる）。メッセージの敵の名前は `g.enemyDisplayName(e.Name)`。

## ヘルプ

`enemies_expansion_b1.go` の `init()` で、新しいページを1枚加える（`help.go` は編集しない）。

```go
expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（3）", Lines: []string{ … }})
```

1系統1〜2行（例:「ウミダンゴ：体当たりで吹き飛ばし、持ち物を落とさせる」「カエルアンコウ：3マス先から引き寄せる。途中の罠も踏む」「ハジキガニ：盾を弾き飛ばす。弾きよけの指輪で防げる」）。最後に「上位種も同じ能力を持つ。」。

## テスト（`content_enemies_wave5a_test.go`、`//go:build !test`）

`content_enemy_foundation_test.go` の `addTestEnemy`、`content_foundation_test.go` の `newFoundationTestGame`・`equip`・`testArmor`・`withTestImmunity`、`content_poison_test.go` の `runQueuedActions`、`content_stage3_foundation_test.go` の `withStatusImmunityRing` を使ってよい。特殊攻撃を確実に出すには `g.random()` の系列ではなく、`MonsterDefinitions[id].SpecialAttack(&g.state.Enemies[i], g)` を直接呼ぶ。

- 表: 9体が存在し、名前・Type・能力値・属性が表どおり。`MonsterLevelUpTable` が 90→91→92 のようにつながる。出現階が表どおり（代表の階で `FloorSpawnTables` に入っている／範囲外の階には無い）。画像・上位種の色違い・出現テーブルは既存のテスト（`TestEveryBaseMonsterHasUpperSpecies`・`TestEnemySprites`・`TestUpperSpeciesSpritesOnlyRecolorOriginal`・`TestDeepFloorSpawnTables`・`TestUpperSpeciesAppearInFloorSpawnTables`）が表から自動で確かめるので、通ることを確認する。
- ウミダンゴ: 体当たりが防御力に関係なく 6 ダメージ（91・92 は 11・20）／2マス吹き飛ぶ／壁にぶつかると追加で5ダメージ、受け身（`AbilityBreakFall` の指輪）なら受けない／不動（`AbilityImmovable`）なら動かない／持ち物が1〜2個落ち、装備中の物は落ちない／盾で回避・反射されたら吹き飛ばない／封印中は体当たりしない。
- カエルアンコウ: 縦・斜めの3マス先から隣まで引き寄せる／2マス先なら1マス引き寄せる／途中の罠で止まって踏む／4マス先・直線でない・間に壁か敵がいると引き寄せない／隣にいれば通常攻撃（`false`）／不動なら動かない／確率に外れると引き寄せない／封印中は引き寄せない。
- ハジキガニ: 盾が外れて後ろ（4マス先、壁があれば手前）に落ち、防御力が下がる／呪われた盾も外れる／盾が無ければ通常攻撃／`StatusKnockOff` を防ぐ指輪で防ぐ／封印中は弾かない／甲殻斬り（`AbilityShellSlayer`）の特効が効く。
- セーブ: 9体それぞれ `enemyToSaved` → `savedToEnemy` で ID・HP・能力（`MonsterDefinitions[id].Behavior` など）が保たれる。
- `go test -run TestSimSmoke -count=1 .` で違反0。

## 完了条件

- `gofmt -w`、`go test -tags test ./...`、`go test ./...`、`go test -run TestSimSmoke -count=1 .` がすべて成功する。
- `go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1500 -sim.difficulty=all -sim.out=reports` を1回実行し、`reports/baseline.json` との差分（死因の上位に新しい敵がどれだけ入ったか、expert のクリア率と到達階、餓死の数）を完了報告に書く（ベースラインは更新しない）。
- `docs/agents/done/wave5-A.md` に、変更概要、IMPLEMENTED_FEATURES.md への追記案（敵の一覧と能力）、仕様の判断、未決事項（数値の妥当性・ゲーム内で確認してほしい見た目）を書く。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

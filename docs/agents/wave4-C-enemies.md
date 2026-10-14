# wave4-C: 敵 A 後半（敵 ID 78〜89、Codex 用）

ブランチ: `agent/wave4-C-enemies`
前提: master に段2の共通基盤（`agent/content-plan-2`。`enemy_expansion.go`・`enemy_behaviors.go`・`enemies_expansion_a2.go`）と画像 img-1 が統合されていること。`enemies_expansion_a2.go` や `img/iyashi_namako.png` が master に無ければ作業を始めず、その旨を報告する。
最初に `AGENTS.md`（特に「新要素追加チェックリスト」の敵の規則と「ファイル分担」）、`docs/agents/README.md` の並列作業のルール、`docs/agents/STATUS.md`、`docs/agents/content-expansion-plan.md` を読む。`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない。
並行して wave4-A（指輪、`items_rings.go`）と wave4-B（敵 A 前半、`enemies_expansion_a1.go`）が動く。wave3-A/B/C（武器・盾・カード）がまだ統合されていない場合もある。

## 触るファイル（ほかの担当と衝突しないための約束）

| ファイル | やること |
|---|---|
| `enemies_expansion_a2.go` | 表 `expansionA2MonsterFamilies` への4系統の追加と、特殊攻撃・行動・撃破時の関数。**このタスクのコードはすべてここに書く** |
| `content_enemies_wave4c_test.go`（新規） | テスト |
| `docs/agents/done/wave4-C.md`（新規） | 完了報告 |

ほかのファイル（`enemies.go`・`monster_spawn.go`・`main.go`・`draw.go`・`help.go`・`enemy_ai.go`・`enemy_behaviors.go`・`enemy_inventory_attacks.go`）は編集しない。定義・上位種化（`MonsterLevelUpTable`）・画像（`upperEnemyImagePaths`）・出現階（`FloorSpawnTables`）は、表に書けば `enemy_expansion.go` が登録する。ヘルプは同じファイルの `init()` で `expansionHelpPages` にページを加える（下の「ヘルプ」）。
共通基盤に足りないものがあれば、自分で共通のファイルを直さず、完了報告に書く（このファイル内で閉じた補助関数を作るのはよい）。

## 作るもの

4系統12体。上位種は基本種の固有能力（特殊攻撃・確率・行動・撃破時の処理・属性・痛恨・回避）をそのまま引き継ぐ（`monsterStage` には能力値と表示だけを書く）。画像はすべて img-1 で作成済み。

| ID | 名前 | Type | 画像（`img/<>.png`） | Char | HP | 攻撃 | 防御 | 経験値 | 出現階（重み） |
|---|---|---|---|---|---|---|---|---|---|
| 78 | イヤシナマコ | IyashiNamako | iyashi_namako | L | 30 | 3 | 4 | 20 | 4〜10（5） |
| 79 | ジヒナマコ | JihiNamako | jihi_namako | L | 54 | 6 | 7 | 50 | 11〜18（5） |
| 80 | セイボナマコ | SeiboNamako | seibo_namako | L | 92 | 12 | 9 | 125 | 19〜30（5） |
| 81 | ゼニガタアザラシ | ZenigataAzarashi | zenigata_azarashi | X | 28 | 6 | 4 | 18 | 3〜8（6） |
| 82 | キンカアザラシ | KinkaAzarashi | kinka_azarashi | X | 50 | 11 | 7 | 45 | 9〜16（6） |
| 83 | センリョウアザラシ | SenryouAzarashi | senryou_azarashi | X | 85 | 22 | 9 | 112 | 17〜30（6） |
| 84 | スナモグリ | Sunamoguri | sunamoguri | K | 32 | 7 | 5 | 22 | 5〜11（7） |
| 85 | スナカケモグリ | SunakakeMoguri | sunakake_moguri | K | 58 | 12 | 8 | 55 | 12〜19（7） |
| 86 | スナアラシモグリ | SunaarashiMoguri | sunaarashi_moguri | K | 98 | 24 | 10 | 138 | 20〜30（7） |
| 87 | ヨロイエビ | YoroiEbi | yoroi_ebi | V | 40 | 9 | 8 | 30 | 6〜12（7） |
| 88 | カブトエビ | KabutoEbi | kabuto_ebi | V | 70 | 15 | 11 | 70 | 13〜20（7） |
| 89 | ムシャエビ | MushaEbi | musha_ebi | V | 119 | 30 | 13 | 175 | 21〜30（7） |

- 能力値の基準: 3〜8階のイシガニ 32/7/5/15・ノロイガニ 38/8/6/22・ワナシヤドカリ 40/8/7/32、第2段階のガンセキガニ 60/12/9/44、第3段階は第2段階の HP×1.7・攻撃×2・防御+2・経験値×2.5（`balance-level99.md`）。ヨロイエビ系は倒した後にユウレイエビ系が出るぶん、防御を高めにして経験値も多めにした。
- 出現階の重みの目安: 同じ階の既存の敵は5〜10。第3段階は9階以降だけ。
- 属性（`Traits`）: スナモグリ系・ヨロイエビ系に `EnemyTraitShell`（甲殻。甲殻斬りの特効が効く）。ほかは付けない。

### 固有の能力

| 系統 | 設定する項目 | 能力 |
|---|---|---|
| イヤシナマコ系 | `Behavior` | 自分では攻撃しない（いつも `true`）。`g.injuredAllyNear(i, 5)` で周囲5マス以内の HP の減った敵（ボス・擬態中を除く）を探し、隣（チェビシェフ距離1）にいれば回復する（78: 15、79: 30、80: 60。`g.healEnemy`）。回復は Action の `Execute` 内で行い、実行時に相手を `g.findEnemyForDefense(ID, X, Y)` で引き直す（「〜が〜の傷を癒やした。」）。隣にいなければ `g.stepEnemyToward(i, 相手の X, Y)` で近づく。回復する相手がいなければ `moveRandomly(g, i)` |
| ゼニガタアザラシ系 | `SpecialAttack`、`SpecialAttackProbability: 0.35` | 隣接して攻撃するとき35%で、通常攻撃の代わりに所持金（`Player.Cash`）を盗む。盗む額は `min(Cash, 上限)`（81: 200、82: 500、83: 1000）。所持金が0、またはすでに盗品を持っている（`HeldItem != nil`）なら通常攻撃（`g.enqueueEnemyNormalAttack(i)`）。守銭の盾（`g.playerBlocksTheft()`）なら `enqueueStealAttack` と同じ文で防ぐ。盗んだら、所持金を減らし、盗んだ額の小銭を `HeldItem` に持たせ、`Fleeing = true` にして、海老さんのいる部屋の外の空いた床へワープする（「〜は 200 を盗んでワープした！」）。倒すと既存の `dropEnemyHeldItem` がその場に小銭を落とす。以後は盗品を持った敵の逃走（`moveFleeingEnemy`）になる |
| スナモグリ系 | `SpecialAttack`、`SpecialAttackProbability: 0.3`、`EvasionPercent: 15` | 隣接して攻撃するとき30%で砂をかける。防御力で減らない固定ダメージ（84: 6、85: 10、86: 18）を `g.enqueueEnemyFixedAttack(i, damage, EnemyDamageNormal, EnemyAttackMelee, "〜が砂をかけた。", "砂かけ", onHit)` で与え、当たったとき（`onHit`）に30%（`g.randIntn(100) < 30`）で目潰し5ターン（`g.playerStatusGuard(StatusBlind)` で防げる。防いだら「〜が目潰しを防いだ」）。海老さんの通常攻撃を15%でかわす（F4 の既存の仕組み。封印・睡眠・金縛り中はかわさない） |
| ヨロイエビ系 | `OnDefeated` | 倒れると「〜の鎧から何かが抜け出した…」と表示し、`g.scheduleEnemySpawn(霊の ID, defeated.X, defeated.Y, 3)` で3ターン後に倒れた場所の近く（2マス以内の空いた床）へ同じ段階のユウレイエビ系を出す（87→11 ユウレイエビ、88→29 シニガミエビ、89→56 メイフエビ）。どの倒れ方（海老さんの攻撃・道具・敵どうし・爆発・真空斬り）でも出る。封印中に倒れたら出ない（共通基盤）。ジェノサイドでユウレイエビ系を封じていれば出ない。フロアを移動すると予約は消える。予約は中断セーブに残る |

- 盗んだ小銭は `buildItemFromTemplate(0, …)` を呼ぶと表の初期化が循環するので、`newTransformedSausage`（`enemy_inventory_attacks.go`）と同じく `&Money{BaseItem: BaseItem{Entity: Entity{Char: '!'}, ID: 0, Type: "Kane", Name: "小銭", …}, Amount: 額, Identified: true}` をこのファイルの関数で直接作る（テンプレート 0〔`items.go`〕の `Type`・`Name`・`Description`・`Char`・`UseActions`〔`"UseMoney": money`〕と同じ値にする）。拾うと `PickUpItem` が所持金に足す。
- アザラシのワープ先は、このファイルに補助関数（例: `g.warpEnemyAwayFromPlayer(i) bool`）を作る。候補は `g.canPlaceEnemyAt(x, y)` が true で、`g.IsInPlayerVisibleRoom(x, y)` が false の床。`g.randIntn(len(候補))` で選ぶ。候補が無ければワープしない（盗みは成立する）。ワープ後は `Animating = false`、`dx, dy = 0`、`g.miniMapDirty = true`。
- 状態の変更（所持金・`HeldItem`・回復・ワープ）は Action の `Execute` 内で行う（`AGENTS.md` の「落とし穴」）。`enqueueStealAttack` の書き方を手本にする。
- 封印中は `Behavior`・`SpecialAttack`・`OnDefeated` は呼ばれない（共通基盤）。睡眠・金縛り・混乱・目潰し・擬態の処理は `Behavior` より先に行われる。
- `Behavior`・`SpecialAttack`・`OnDefeated` の関数が `MonsterDefinitions` や `CreateEnemyByID` を（ほかの関数経由でも）参照すると、表の初期化が循環してコンパイルエラー（initialization cycle）になる。そのときは表には書かず、`init()` で `updateMonsterDefinitions(func(d *MonsterDefinition) { d.Behavior = … }, 78, 79, 80)` のように3段階すべてに設定する（`bosses.go` の `init` と同じ）。
- 乱数はすべて `g.randIntn`（敵の行動の系列）。メッセージの敵の名前は `g.enemyDisplayName(e.Name)`。

## ヘルプ

`enemies_expansion_a2.go` の `init()` で、新しいページを1枚加える（`help.go` は編集しない）。

```go
expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（2）", Lines: []string{ … }})
```

1系統1〜2行（例:「イヤシナマコ：攻撃せず、傷ついた敵を回復する」「ゼニガタアザラシ：お金を盗んでワープする。守銭の盾で防げる」「スナモグリ：砂で目潰しをしてくる。攻撃をかわすことがある」「ヨロイエビ：倒すと少し後にユウレイエビ系が現れる」）。最後に「上位種も同じ能力を持つ。」。

## テスト（`content_enemies_wave4c_test.go`、`//go:build !test`）

`content_enemy_foundation_test.go` の `addTestEnemy`、`content_foundation_test.go` の `newFoundationTestGame`・`equip`・`testArmor`、`content_poison_test.go` の `runQueuedActions` を使ってよい。部屋の判定が要るテストは `g.rooms` に部屋を入れる（アザラシのワープ先には部屋の外の床も要るので、マップを広げるか、2部屋を作る）。

- 表: 12体が存在し、名前・Type・能力値・属性が表どおり。`MonsterLevelUpTable` が 78→79→80 のようにつながる。出現階が表どおり。画像・上位種の色違い・出現テーブルは既存のテストが表から自動で確かめるので、通ることを確認する。
- イヤシナマコ: 隣の傷ついた敵を回復し最大 HP を超えない／離れていれば近づく／海老さんの隣でも攻撃しない／ボスは回復しない／封印中は回復せず通常の敵として攻撃する。
- ゼニガタアザラシ: 所持金を上限まで盗み、部屋の外へワープして逃げる／所持金0なら通常攻撃／守銭の盾で防ぐ（所持金は減らない）／倒すと盗んだ額の小銭を落とし、拾うと所持金が戻る／盗品を持った状態でセーブ→復元すると小銭が残る。
- スナモグリ: 砂が防御力に関係なく 6 ダメージ／30% で目潰し5ターン、外れると目潰しなし／状態異常耐性で防ぐ（`statusImmunities` に仮の能力で `StatusBlind` を登録した指輪、`content_foundation_test.go` の `withTestImmunity`）／`randHooks.enemyEvasion` を 14 にすると海老さんの攻撃をかわし、15 で当たる。
- ヨロイエビ: 倒して3ターン（`g.processPendingEnemySpawns()` を3回、または `AdvanceTurn`）後にユウレイエビが近くに出る、88・89 はシニガミエビ・メイフエビ／封印中に倒すと出ない／ユウレイエビ系をジェノサイドで封じていると出ない／敵に倒されても出る。
- セーブ: 12体それぞれ `enemyToSaved` → `savedToEnemy` で ID・HP が保たれる。
- `go test -run TestSimSmoke -count=1 .` で違反0。

## 完了条件

- `gofmt -w`、`go test -tags test ./...`、`go test ./...`、`go test -run TestSimSmoke -count=1 .` がすべて成功する。
- `go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1500 -sim.difficulty=all -sim.out=reports` を1回実行し、`reports/baseline.json` との差分（死因の上位に新しい敵がどれだけ入ったか、expert のクリア率と到達階、餓死の数）を完了報告に書く（ベースラインは更新しない）。
- `docs/agents/done/wave4-C.md` に、変更概要、IMPLEMENTED_FEATURES.md への追記案（敵の一覧と能力）、仕様の判断、未決事項（数値の妥当性・ゲーム内で確認してほしい見た目）を書く。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

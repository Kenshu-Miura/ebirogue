# wave4-B: 敵 A 前半（敵 ID 66〜77、Codex 用）

ブランチ: `agent/wave4-B-enemies`
前提: master に段2の共通基盤（`agent/content-plan-2`。`enemy_expansion.go`・`enemy_behaviors.go`・`enemies_expansion_a1.go`）と画像 img-1 が統合されていること。`enemies_expansion_a1.go` や `img/hibana_umiushi.png` が master に無ければ作業を始めず、その旨を報告する。
最初に `AGENTS.md`（特に「新要素追加チェックリスト」の敵の規則と「ファイル分担」）、`docs/agents/README.md` の並列作業のルール、`docs/agents/STATUS.md`、`docs/agents/content-expansion-plan.md` を読む。`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない。
並行して wave4-A（指輪、`items_rings.go`）と wave4-C（敵 A 後半、`enemies_expansion_a2.go`）が動く。wave3-A/B/C（武器・盾・カード）がまだ統合されていない場合もある。

## 触るファイル（ほかの担当と衝突しないための約束）

| ファイル | やること |
|---|---|
| `enemies_expansion_a1.go` | 表 `expansionA1MonsterFamilies` への4系統の追加と、特殊攻撃・行動の関数。**このタスクのコードはすべてここに書く** |
| `content_enemies_wave4b_test.go`（新規） | テスト |
| `docs/agents/done/wave4-B.md`（新規） | 完了報告 |

ほかのファイル（`enemies.go`・`monster_spawn.go`・`main.go`・`draw.go`・`help.go`・`enemy_ai.go`・`enemy_behaviors.go`）は編集しない。定義・上位種化（`MonsterLevelUpTable`）・画像（`upperEnemyImagePaths`）・出現階（`FloorSpawnTables`）は、表に書けば `enemy_expansion.go` が登録する。ヘルプは同じファイルの `init()` で `expansionHelpPages` にページを加える（下の「ヘルプ」）。
共通基盤に足りないものがあれば、自分で共通のファイルを直さず、完了報告に書く（このファイル内で閉じた補助関数を作るのはよい）。

## 作るもの

4系統12体。上位種は基本種の固有能力（特殊攻撃・確率・行動・属性・痛恨・回避）をそのまま引き継ぐ（`monsterStage` には能力値と表示だけを書く）。画像はすべて img-1 で作成済み。

| ID | 名前 | Type | 画像（`img/<>.png`） | Char | HP | 攻撃 | 防御 | 経験値 | 出現階（重み） |
|---|---|---|---|---|---|---|---|---|---|
| 66 | ヒバナウミウシ | HibanaUmiushi | hibana_umiushi | A | 26 | 6 | 3 | 16 | 3〜4（5）、5〜8（8） |
| 67 | カエンウミウシ | KaenUmiushi | kaen_umiushi | A | 48 | 11 | 6 | 40 | 7〜8（4）、9〜16（7） |
| 68 | ゴウカウミウシ | GoukaUmiushi | gouka_umiushi | A | 82 | 22 | 8 | 100 | 15〜16（4）、17〜30（7） |
| 69 | テッポウエビ | TeppouEbi | teppou_ebi | P | 30 | 7 | 4 | 18 | 4〜5（5）、6〜10（8） |
| 70 | タイホウエビ | TaihouEbi | taihou_ebi | P | 54 | 12 | 7 | 45 | 10〜17（7） |
| 71 | ゴウホウエビ | GouhouEbi | gouhou_ebi | P | 92 | 24 | 9 | 112 | 18〜30（7） |
| 72 | ニゲハゼ | NigeHaze | nige_haze | Z | 14 | 2 | 2 | 24 | 2〜9（4） |
| 73 | トンズラハゼ | TonzuraHaze | tonzura_haze | Z | 26 | 4 | 5 | 60 | 8〜16（4） |
| 74 | キンピカハゼ | KinpikaHaze | kinpika_haze | Z | 44 | 8 | 7 | 150 | 15〜30（3） |
| 75 | ヨビコミボラ | YobikomiBora | yobikomi_bora | Y | 28 | 5 | 6 | 22 | 5〜11（5） |
| 76 | ヨビアツメボラ | YobiatsumeBora | yobiatsume_bora | Y | 50 | 9 | 9 | 55 | 12〜19（5） |
| 77 | ダイショウカンボラ | DaishoukanBora | daishoukan_bora | Y | 85 | 18 | 11 | 138 | 20〜30（5） |

- 能力値の基準: 3〜8階のイシガニ 32/7/5/15・ハリセンボウ 18/6/2/8・ノロイガニ 38/8/6/22、第2段階のガンセキガニ 60/12/9/44、第3段階は第2段階の HP×1.7・攻撃×2・防御+2・経験値×2.5（`balance-level99.md`）。
- 出現階の重みの目安: 同じ階の既存の敵は5〜10。第3段階は9階以降だけ（`TestDeepFloorSpawnTables` が8階以前の第3段階を禁止している）。
- 属性（`Traits`）は付けない。

### 固有の能力

| 系統 | 設定する項目 | 能力 |
|---|---|---|
| ヒバナウミウシ系 | `SpecialAttack`、`SpecialAttackProbability: 0.3` | 隣接して攻撃するとき30%で通常攻撃の代わりに火を吹く。25%（`g.randIntn(100) < 25`）で不発（「〜は火を吹こうとしたが、しけっていた。」、ダメージなし）。当たれば防御力で減らない固定ダメージ（66: 8、67: 14、68: 24。ID ごとの表 `hibanaFireDamage map[int]int` を引く）を `g.enqueueEnemyFixedAttack(i, damage, EnemyDamageFire, EnemyAttackRanged, "〜は火を吹いた。", "炎", nil)` で与える。属性が炎なので火喰いの盾で半減、反射の盾で跳ね返る。死因は「ヒバナウミウシの炎」 |
| テッポウエビ系 | `Behavior` | 海老さんが**ちょうど2マス先**の直線上（縦・横・斜め）にいて、間のマスが壁（`Blocked`）でも敵でもなければ、その場から通常攻撃する（`g.AttackFromEnemy(i)` を呼んで `true`）。斜めでも角の壁で止まらない（「角越しも可」）。それ以外は `false`（隣接時の通常攻撃・接近はいつもどおり）。海老さんを発見していない（`PlayerDiscovered` が false）ときは撃たない |
| ニゲハゼ系 | `Behavior`、`SpecialMovement: SpecialMovementDoubleSpeed` | 攻撃しない。海老さんを発見していれば `g.moveEnemyAwayFromPlayer(i, g.randIntn)`（盗品を持った敵の逃走と同じ処理）で離れ、発見していなければ `moveRandomly(g, i)`。追い詰められて動けなくても攻撃しない。いつも `true` を返す。倍速は既存の固有能力（封印で止まる） |
| ヨビコミボラ系 | `Behavior` | 海老さんと同じ部屋（`isSameRoom(…, g.rooms)`）にいて発見済みで、自分の周囲2マスにいるほかの敵が3体未満のとき、25%（`g.randIntn(4) == 0`）でほら貝を鳴らし（「〜がほら貝を鳴らした！」）、`g.summonMonstersAround(e.X, e.Y, 2, 呼ぶ数, monsterFamilyIDs(75))` で今の階の敵を呼ぶ（呼ぶ数は 75: 1、76: 1、77: 2）。1体も来なければ「しかし何も来なかった。」。鳴らしたら `true`、条件を満たさない・確率に外れたら `false`（通常の行動） |

- 封印中は `Behavior` も `SpecialAttack` も呼ばれない（共通基盤）。睡眠・金縛り・混乱・目潰し・擬態の処理は `Behavior` より先に行われる。
- `Behavior` の関数が `MonsterDefinitions` や `CreateEnemyByID` を（`summonMonstersAround`・`monsterFamilyIDs` 経由でも）参照すると、表の初期化が循環してコンパイルエラー（initialization cycle）になる。そのときは表には書かず、`init()` で `updateMonsterDefinitions(func(d *MonsterDefinition) { d.Behavior = … }, 75, 76, 77)` のように3段階すべてに設定する（`bosses.go` の `init` と同じ）。ヒバナの特殊攻撃も同様。
- 乱数はすべて `g.randIntn`（敵の行動の系列）。メッセージの敵の名前は `g.enemyDisplayName(e.Name)`（目潰し中は「何者」になる）。
- 呼び出された敵は普通の敵と同じ（経験値あり、上位種化あり）。フロアの敵の上限 19 体は `summonMonstersAround` が守る。

## ヘルプ

`enemies_expansion_a1.go` の `init()` で、新しいページを1枚加える（`help.go` は編集しない）。

```go
expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（1）", Lines: []string{ … }})
```

1系統1〜2行（例:「ヒバナウミウシ：隣から火を吹く。火喰いの盾で半減」「テッポウエビ：2マス先から攻撃してくる」「ニゲハゼ：攻撃せず逃げ回る。倒すと経験値が多い」「ヨビコミボラ：同じ部屋で仲間を呼ぶ」）。最後に「上位種も同じ能力を持つ。」。

## テスト（`content_enemies_wave4b_test.go`、`//go:build !test`）

`content_enemy_foundation_test.go` の `addTestEnemy`、`content_foundation_test.go` の `newFoundationTestGame`・`equip`・`testArmor`、`content_poison_test.go` の `runQueuedActions` を使ってよい。部屋の判定が要るテストは `g.rooms = []Room{{X: 0, Y: 0, Width: 9, Height: 9}}` を入れる。

- 表: 12体が存在し、名前・Type・能力値が表どおり。`MonsterLevelUpTable` が 66→67→68 のようにつながる。出現階が表どおり（各系統の代表の階で `FloorSpawnTables` に入っている／範囲外の階には無い）。画像・上位種の色違い・出現テーブルは既存のテスト（`TestEveryBaseMonsterHasUpperSpecies`・`TestEnemySprites`・`TestUpperSpeciesSpritesOnlyRecolorOriginal`・`TestDeepFloorSpawnTables`・`TestUpperSpeciesAppearInFloorSpawnTables`）が表から自動で確かめるので、通ることを確認する。
- ヒバナ: 火が当たると防御力に関係なく 8 ダメージ（67・68 は 14・24）／不発ならダメージなし／火喰いの盾で半減／封印中は火を吹かず通常攻撃。
- テッポウ: 2マス先（縦・斜め）から攻撃する／間に壁か敵がいると攻撃しない／3マス先では攻撃しない／斜めの角の壁越しに攻撃できる／封印中は2マス先から攻撃しない。
- ニゲハゼ: 隣にいても攻撃しない（行動の後に Action が積まれない）／発見済みなら海老さんから離れる／封印中は通常の敵として攻撃する／倍速で2回動く。
- ヨビコミボラ: 同じ部屋で確率に当たると周囲2マスに敵が増える（呼ばれた敵は発見済み、ボラ系ではない）／確率に外れる・別の部屋・周りに敵が3体いるときは呼ばない／封印中は呼ばない／ジェノサイドで封じた系統は呼ばない／フロアの上限で止まる。
- セーブ: 12体それぞれ `enemyToSaved` → `savedToEnemy` で ID・HP・能力（`MonsterDefinitions[id].Behavior` など）が保たれる。
- `go test -run TestSimSmoke -count=1 .` で違反0。

## 完了条件

- `gofmt -w`、`go test -tags test ./...`、`go test ./...`、`go test -run TestSimSmoke -count=1 .` がすべて成功する。
- `go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1500 -sim.difficulty=all -sim.out=reports` を1回実行し、`reports/baseline.json` との差分（死因の上位に新しい敵がどれだけ入ったか、expert のクリア率と到達階、餓死の数）を完了報告に書く（ベースラインは更新しない）。
- `docs/agents/done/wave4-B.md` に、変更概要、IMPLEMENTED_FEATURES.md への追記案（敵の一覧と能力）、仕様の判断、未決事項（数値の妥当性・ゲーム内で確認してほしい見た目）を書く。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

# wave6-A: 敵 C（フクレフグ系 108〜110・シビレエイ系 114〜116、Codex 用）

ブランチ: `agent/wave6-A-enemies`
前提: master に段4の共通基盤（`agent/content-plan-4`。`enemy_explosion.go`・`enemy_dig.go`・`enemy_mimic.go`・`projectile.go` の打ち返し・`enemies_expansion_c1.go`）と画像 img-2 が統合されていること。`enemies_expansion_c1.go` や `img/fukure_fugu.png` が master に無ければ作業を始めず、その旨を報告する。
最初に `AGENTS.md`（特に「新要素追加チェックリスト」の敵の規則と「ファイル分担」の `enemy_behaviors.go`・`enemy_explosion.go`）、`docs/agents/README.md` の並列作業のルール、`docs/agents/STATUS.md`、`docs/agents/content-expansion-plan.md` を読む。`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない。
並行して wave6-B（アナジャコ系・コバンザメ系、`enemies_expansion_c2.go`）と wave6-C（ミミックダコ系・ウチカエシエビ系、`enemies_expansion_c3.go`）が動く。

## 触るファイル（ほかの担当と衝突しないための約束）

| ファイル | やること |
|---|---|
| `enemies_expansion_c1.go` | 表 `expansionC1MonsterFamilies` への2系統の追加と、特殊攻撃・行動・被弾時の関数。**このタスクのコードはすべてここに書く** |
| `content_enemies_wave6a_test.go`（新規） | テスト |
| `docs/agents/done/wave6-A.md`（新規） | 完了報告 |

ほかのファイル（`enemies.go`・`enemy_expansion.go`・`monster_spawn.go`・`main.go`・`draw.go`・`help.go`・`enemy_ai.go`・`enemy_behaviors.go`・`enemy_explosion.go`・`projectile.go`・`savegame.go`）は編集しない。定義・上位種化・画像・出現階は表に書けば `enemy_expansion.go` が登録する（登録の1行は共通基盤で追加済み）。ヘルプは同じファイルの `init()` で `expansionHelpPages` にページを加える。
共通基盤に足りないものがあれば、自分で共通のファイルを直さず、完了報告に書く（このファイル内で閉じた補助関数を作るのはよい）。

## 作るもの

2系統6体。上位種は基本種の固有能力（特殊攻撃・確率・行動・被弾時の処理・属性）をそのまま引き継ぐ（`monsterStage` には能力値と表示だけを書く）。画像はすべて img-2 で作成済み。`Char` は既存の敵と重ならない小文字にした（画像があるので画面には出ない）。

| ID | 名前 | Type | 画像（`img/<>.png`） | Char | HP | 攻撃 | 防御 | 経験値 | 出現階（重み） |
|---|---|---|---|---|---|---|---|---|---|
| 108 | フクレフグ | FukureFugu | fukure_fugu | p | 32 | 7 | 5 | 24 | 6〜12（6） |
| 109 | バクハツフグ | BakuhatsuFugu | bakuhatsu_fugu | p | 56 | 12 | 8 | 60 | 13〜20（6） |
| 110 | ダイバクハツフグ | DaibakuhatsuFugu | daibakuhatsu_fugu | p | 95 | 24 | 10 | 150 | 21〜30（6） |
| 114 | シビレエイ | ShibireEi | shibire_ei | e | 38 | 8 | 7 | 30 | 8〜14（5） |
| 115 | デンキエイ | DenkiEi | denki_ei | e | 66 | 14 | 10 | 75 | 15〜22（5） |
| 116 | ライテイエイ | RaiteiEi | raitei_ei | e | 112 | 28 | 12 | 188 | 23〜30（5） |

- 能力値の基準: 同じ出現帯のオオハサミガニ 36/10/6/26（6〜12階）・ハジキガニ 40/9/8/28（7〜13階）・ヨロイエビ 40/9/8/30（6〜12階）。第2段階は基本種の HP×1.75・攻撃×1.7・防御+3・経験値×2.5、第3段階は第2段階の HP×1.7・攻撃×2・防御+2・経験値×2.5（`balance-level99.md`）。明らかに外れていれば直して完了報告に書く。
- 属性（`Traits`）: フクレフグ系に `EnemyTraitExplosive`（爆発。ウニ割りの特効が効く）。シビレエイ系には付けない。

### 敵ごとの汎用の状態の使い方

段4の共通基盤で、敵に系統が自由に使う欄 `Counter`・`Timer`・`Flag`（`enemies.go` の `Enemy`。中断セーブに含まれ、上位種化では消えない）を用意した。このタスクでは次の意味で使い、ファイルの先頭のコメントにも書く。

| 系統 | `Counter` | `Timer` | `Flag` |
|---|---|---|---|
| フクレフグ系 | 自爆を予約したら1（二重に予約しない） | 使わない | 膨らんでいる |
| シビレエイ系 | 帯電していない間に受けた攻撃の回数 | 帯電の残りターン | 帯電している |

### 固有の能力

状態の変更（ダメージ・自爆・帯電）は Action の `Execute` 内か、`OnDamaged`（`g.applyDamageToEnemy` の中＝ Action の `Execute` の中から呼ばれる）・`Behavior`（敵のターン）の中で行う。`OnDamaged` は範囲攻撃のループの中からも呼ばれるので、**その中で敵を取り除かない**（自爆は `g.enqueueEnemyExplosion` が Action として積むので、そのまま呼んでよい）。メッセージの敵の名前は `g.enemyDisplayName(e.Name)`。

| 系統 | 設定する項目 | 能力 |
|---|---|---|
| フクレフグ系 | `OnDamaged`、`Behavior` | **膨らむ**: 攻撃を受けて倒れず、HP が最大 HP の半分以下になったら `Flag = true` にして「〜は膨らんだ！」。膨らんでいる間は `Behavior` が `true` を返し、その場から動かず攻撃もしない。**自爆**: HP が最大 HP の1/4以下になったら（`Counter == 0` のときだけ。予約したら `Counter = 1`）、`g.enqueueEnemyExplosion(EnemyExplosion{BomberIndex: i, CenterX: e.X, CenterY: e.Y, Radius: 1, Damage: 自爆のダメージ, SelfDestruct: true, Message: "〜が破裂した！"})`。1回の攻撃で半分以下と1/4以下を同時にまたいだら、膨らまずにすぐ自爆する。一撃で倒れたら自爆しない（`OnDamaged` は倒れなかったときだけ呼ばれる） |
| シビレエイ系 | `OnDamaged`、`Behavior` | **帯電**: 帯電していない（`Flag == false`）間に攻撃を受けて倒れないたびに `Counter++`。3回になったら `Flag = true`、`Timer = 10`、`Counter = 0` にして「〜は帯電した！」。**雷**: 帯電中、`Behavior` は毎ターン最初に `Timer--` し、0 になったら `Flag = false` にして「〜の帯電が消えた」（その回は `false` を返し通常の行動）。帯電中で、海老さんを発見済みかつ同じ部屋にいる（`isSameRoom(e.X, e.Y, 海老さん.X, 海老さん.Y, g.rooms)`）なら、1/3（`g.randIntn(3) == 0`）で雷を落とす: `g.enqueueEnemyFixedAttack(i, 雷のダメージ, EnemyDamageMagic, EnemyAttackRanged, "〜が雷を落とした！", "雷", nil)`（魔封じの盾で半減、反射の盾で反射、見切りの盾でかわせる）して `true`。**放電の反撃**: 帯電中に攻撃を受けて倒れず、海老さんが隣（チェビシェフ距離1）にいれば、`g.enqueueEnemyFixedAttack(i, 反撃のダメージ, EnemyDamageMagic, EnemyAttackEnvironment, "〜が放電した！", "放電", nil)`（かわせない・反射できない。魔封じの盾で半減）。反撃しても帯電は続く |

ダメージは ID ごとの表（`map[int]int`）にする。

| ID | 自爆のダメージ | ID | 雷のダメージ | 放電の反撃 |
|---|---|---|---|---|
| 108 | 14 | 114 | 8 | 5 |
| 109 | 26 | 115 | 15 | 9 |
| 110 | 48 | 116 | 28 | 16 |

- 自爆の規則（共通基盤 `enemy_explosion.go`）: 海老さんには防御力で減らない固定ダメージ（爆風よけの盾で半減。聖域の上でも受ける）、周囲1マスのほかの敵にも同じダメージ（ボスは50まで）、床の道具は消える。巻き込まれて倒れた敵の経験値は海老さんに入らず、自爆したフグの経験値も入らない（撃破時の処理 `OnDefeated` は呼ばれる）。巻き込まれた別のフクレフグは `OnDamaged` が呼ばれるので連鎖することがある。死因は「フクレフグの爆発」。
- 封印中は `Behavior`・`OnDamaged` とも呼ばれない（共通基盤）。封印されたフクレフグは膨らんでいても動き出し（`Behavior` が呼ばれないため）、自爆もしない。シビレエイの帯電の `Timer` も封印中は減らない（帯電したまま何もしない）ことを完了報告に書く。
- 乱数は `g.randIntn`。`Behavior`・`OnDamaged` の関数が `MonsterDefinitions` や `CreateEnemyByID` を（`enemyRangedSource` などほかの関数経由でも）参照すると、表の初期化が循環してコンパイルエラーになる。そのときは表には書かず、`init()` で `updateMonsterDefinitions(func(d *MonsterDefinition) { d.OnDamaged = … }, 108, 109, 110)` のように3段階すべてに設定する（`enemies_expansion_b1.go` の `init` と同じ）。

## ヘルプ

`enemies_expansion_c1.go` の `init()` で、新しいページを1枚加える（`help.go` は編集しない）。

```go
expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（5）", Lines: []string{ … }})
```

1系統1〜2行（例:「フクレフグ：弱ると膨らんで止まり、さらに弱ると周りを巻き込んで破裂する」「シビレエイ：3回攻撃されると帯電し、同じ部屋に雷を落とす。帯電中に殴ると放電で反撃」）。最後に「上位種も同じ能力を持つ。」。

## テスト（`content_enemies_wave6a_test.go`、`//go:build !test`）

`content_enemy_foundation_test.go` の `addTestEnemy`、`content_foundation_test.go` の `newFoundationTestGame`・`equip`・`testArmor`、`content_poison_test.go` の `runQueuedActions`、`content_stage4_foundation_test.go` の `newExplosionTestGame` を使ってよい。被弾時の処理は `g.applyDamageToEnemy(i, ダメージ)` で、行動は `MonsterDefinitions[id].Behavior(g, i)` を直接呼んで確かめる。

- 表: 6体が存在し、名前・Type・能力値・属性が表どおり。`MonsterLevelUpTable` が 108→109→110・114→115→116 とつながる。出現階が表どおり（代表の階で `FloorSpawnTables` に入っている／範囲外の階には無い）。画像・上位種の色違い・出現テーブルは既存のテスト（`TestEveryBaseMonsterHasUpperSpecies`・`TestEnemySprites`・`TestUpperSpeciesSpritesOnlyRecolorOriginal`・`TestDeepFloorSpawnTables`・`TestUpperSpeciesAppearInFloorSpawnTables`）が表から自動で確かめるので、通ることを確認する。
- フクレフグ: 半分以下で膨らみ、膨らんでいる間は動かない／1/4以下で自爆し、隣の海老さんが固定 14 ダメージ（109・110 は 26・48）、爆風よけの盾で半減／周囲のほかの敵も同じダメージを受け、倒れても海老さんの経験値は増えない／自爆したフグは消え、経験値は入らない／一撃で倒すと自爆しない／二重に自爆を予約しない（同じターンに2回攻撃を受けても爆発は1回）／隣のフクレフグへ連鎖する／床の道具が消える／封印中は膨らまず自爆しない／ウニ割り（`AbilityExplosiveSlayer`）の特効が効く。
- シビレエイ: 2回の被弾では帯電せず、3回目で帯電する／帯電中に同じ部屋なら雷（1/3 の判定が当たる乱数のシードを探して `g.rng = newTurnRNG(シード)` で固定するか、何度か呼んで確かめる）、魔封じの盾で半減、反射の盾で反射／別の部屋・未発見なら雷を落とさない／10ターンで帯電が消える／帯電中に隣で攻撃すると放電の反撃（見切りの盾でもかわせない）、離れた所から投げて当てたときは反撃しない／封印中は帯電しない。
- セーブ: 6体それぞれ `enemyToSaved` → `savedToEnemy` で ID・HP・能力（`MonsterDefinitions[id].OnDamaged` など）と `Counter`・`Timer`・`Flag` が保たれる（膨らんだフグ・帯電したエイを保存して戻す）。
- `go test -run TestSimSmoke -count=1 .` で違反0。

## 完了条件

- `gofmt -w`、`go test -tags test ./...`、`go test ./...`、`go test -run TestSimSmoke -count=1 .` がすべて成功する。
- `go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1500 -sim.difficulty=all -sim.out=reports` を1回実行し、`reports/baseline.json` との差分（死因の上位に新しい敵がどれだけ入ったか、特に「フクレフグの爆発」、expert のクリア率と到達階、餓死の数）を完了報告に書く（ベースラインは更新しない）。
- `docs/agents/done/wave6-A.md` に、変更概要、IMPLEMENTED_FEATURES.md への追記案（敵の一覧と能力）、仕様の判断、未決事項（数値の妥当性・ゲーム内で確認してほしい見た目〔膨らんだフグ・帯電したエイは見た目が変わらない〕）を書く。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

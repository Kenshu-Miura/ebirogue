# wave6-B: 敵 C（アナジャコ系 111〜113・コバンザメ系 120〜122、Codex 用）

ブランチ: `agent/wave6-B-enemies`
前提: master に段4の共通基盤（`agent/content-plan-4`。`enemy_dig.go`・`enemy_explosion.go`・`enemy_mimic.go`・`projectile.go` の打ち返し・`enemies_expansion_c2.go`）と画像 img-2 が統合されていること。`enemies_expansion_c2.go` や `img/ana_jako.png` が master に無ければ作業を始めず、その旨を報告する。
最初に `AGENTS.md`（特に「新要素追加チェックリスト」の敵の規則と「ファイル分担」の `enemy_behaviors.go`・`enemy_dig.go`）、`docs/agents/README.md` の並列作業のルール、`docs/agents/STATUS.md`、`docs/agents/content-expansion-plan.md` を読む。`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない。
並行して wave6-A（フクレフグ系・シビレエイ系、`enemies_expansion_c1.go`）と wave6-C（ミミックダコ系・ウチカエシエビ系、`enemies_expansion_c3.go`）が動く。

## 触るファイル（ほかの担当と衝突しないための約束）

| ファイル | やること |
|---|---|
| `enemies_expansion_c2.go` | 表 `expansionC2MonsterFamilies` への2系統の追加と、特殊攻撃・行動の関数。**このタスクのコードはすべてここに書く** |
| `content_enemies_wave6b_test.go`（新規） | テスト |
| `docs/agents/done/wave6-B.md`（新規） | 完了報告 |

ほかのファイル（`enemies.go`・`enemy_expansion.go`・`monster_spawn.go`・`main.go`・`draw.go`・`help.go`・`enemy_ai.go`・`enemy_behaviors.go`・`enemy_dig.go`・`savegame.go`）は編集しない。定義・上位種化・画像・出現階は表に書けば `enemy_expansion.go` が登録する（登録の1行は共通基盤で追加済み）。ヘルプは同じファイルの `init()` で `expansionHelpPages` にページを加える。
共通基盤に足りないものがあれば、自分で共通のファイルを直さず、完了報告に書く（このファイル内で閉じた補助関数を作るのはよい）。

## 作るもの

2系統6体。上位種は基本種の固有能力（特殊攻撃・確率・行動・属性）をそのまま引き継ぐ（`monsterStage` には能力値と表示だけを書く）。画像はすべて img-2 で作成済み。`Char` は既存の敵と重ならない小文字にした（画像があるので画面には出ない）。

| ID | 名前 | Type | 画像（`img/<>.png`） | Char | HP | 攻撃 | 防御 | 経験値 | 出現階（重み） |
|---|---|---|---|---|---|---|---|---|---|
| 111 | アナジャコ | AnaJako | ana_jako | a | 36 | 8 | 7 | 26 | 7〜13（5） |
| 112 | オオアナジャコ | OoAnaJako | oo_ana_jako | a | 63 | 14 | 10 | 65 | 14〜21（5） |
| 113 | ジゴクアナジャコ | JigokuAnaJako | jigoku_ana_jako | a | 107 | 28 | 12 | 162 | 22〜30（5） |
| 120 | コバンザメ | Kobanzame | kobanzame | z | 28 | 6 | 4 | 20 | 5〜11（6） |
| 121 | オオバンザメ | Oobanzame | oobanzame | z | 50 | 11 | 7 | 50 | 12〜19（6） |
| 122 | センリョウザメ | Senryouzame | senryouzame | z | 85 | 22 | 9 | 125 | 20〜30（6） |

- 能力値の基準: アナジャコ系は同じ出現帯のハジキガニ 40/9/8/28（7〜13階）・スナモグリ 32/7/5/22（5〜11階）。コバンザメ系はゼニガタアザラシ 28/6/4/18（3〜8階、所持金を盗む）・サビヒトデ 30/6/5/18（5〜11階）。第2段階は基本種の HP×1.75・攻撃×1.7・防御+3・経験値×2.5、第3段階は第2段階の HP×1.7・攻撃×2・防御+2・経験値×2.5（`balance-level99.md`）。明らかに外れていれば直して完了報告に書く。
- 属性（`Traits`）: どちらにも付けない。

### 敵ごとの汎用の状態の使い方

アナジャコ系は共通基盤の `Enemy.Alerted`（`g.enemyHoldsUntilNear` が管理する）だけを使う。`Counter`・`Timer`・`Flag` は使わない。コバンザメ系は何も使わない。

### 固有の能力

状態の変更（ダメージ・所持金を減らす）は Action の `Execute` 内で行う（`AGENTS.md` の「落とし穴」）。壁を掘る・動くのは敵のターン（`Behavior`）の中で直接行ってよい（敵の移動と同じ）。メッセージの敵の名前は `g.enemyDisplayName(e.Name)`。

| 系統 | 設定する項目 | 能力 |
|---|---|---|
| アナジャコ系 | `Behavior` | **近づくまで追ってこない**: `g.enemyHoldsUntilNear(i, 2)` が `true` の間（海老さんがチェビシェフ距離2以内に来るか、ダメージを受けるまで）は海老さんを追わずにうろつく: 1/4（`g.randIntn(4) == 0`）で縦横の4方向から1つを選び（`g.randIntn(4)`）`g.stepEnemyDigging(i, dx, dy)`（壁なら掘って進む）、それ以外か進めなかったときは `moveRandomly(g, i)`。どちらでも `true`。**掘りながら追う**: 追い始めた後、海老さんを発見済み（`PlayerDiscovered`）で隣にいなければ `g.stepEnemyTowardDigging(i, 海老さん.X, 海老さん.Y)`（間に壁があれば縦か横に掘って進む）で、動けたら `true`。隣にいる・発見していない・動けないときは `false`（通常の攻撃・行動） |
| コバンザメ系 | `Behavior` | **小銭を投げる**: 海老さんを発見済みで、`g.straightLineToPlayer(i, 5)` が成り立ち距離が2以上、所持金（`g.state.Player.Cash`）が投げる額以上なら、1/2（`g.randIntn(2) == 0`）で海老さんの財布から小銭を吸い出して投げつける。まず「〜が海老さんの小銭を吸い出した！（N円）」の Action（`Execute` で所持金を N 減らす。0 未満にしない）を積み、続けて `g.enqueueEnemyFixedAttack(i, ダメージ, EnemyDamageNormal, EnemyAttackRanged, "〜が小銭を投げつけた！", "小銭", nil)`（防御力で減らない。見切りの盾・カメの甲羅でかわせ、反射の盾で反射）して `true`。投げた小銭は消える（かわしても戻らない）。守銭の盾（`g.playerBlocksTheft()`）を装備していれば吸い出せず、投げない（`false`、メッセージなし）。所持金が足りない・確率に外れた・直線上にいない・隣にいるときは `false`（近づいて通常攻撃） |

投げる額とダメージは ID ごとの表（`map[int]int`）にする。ダメージは額に比例（額×3/5）。

| ID | 投げる額 | ダメージ |
|---|---|---|
| 120 | 10 | 6 |
| 121 | 24 | 14 |
| 122 | 50 | 30 |

- 壁を掘る規則（共通基盤 `enemy_dig.go`）: つるはしと同じく、通れない壁（`wall`・`other`）だけを掘り、マップの外周は掘らない。斜めの壁は掘らない（海老さんが斜めに通れない穴を作らないため）。掘った通路は、海老さんから見えているマスなら踏破済みになりミニマップに出る。部屋の壁に穴が開くのは仕様。
- 封印中は `Behavior` が呼ばれない（共通基盤）。封印されたアナジャコは普通の敵と同じく追ってくる（掘らない）。封印されたコバンザメは小銭を投げない。睡眠・金縛り・混乱・目潰しの処理は `Behavior` より先に行われる。
- 乱数は `g.randIntn`。`Behavior` の関数が `MonsterDefinitions` や `CreateEnemyByID` を（`enemyRangedSource` などほかの関数経由でも）参照すると、表の初期化が循環してコンパイルエラーになる。そのときは表には書かず、`init()` で `updateMonsterDefinitions(func(d *MonsterDefinition) { d.Behavior = … }, 111, 112, 113)` のように3段階すべてに設定する（`enemies_expansion_b1.go` の `init` と同じ）。

## ヘルプ

`enemies_expansion_c2.go` の `init()` で、新しいページを1枚加える（`help.go` は編集しない）。

```go
expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（6）", Lines: []string{ … }})
```

1系統1〜2行（例:「アナジャコ：近づくまで追ってこない。壁を掘りながら歩く」「コバンザメ：離れた所から財布の小銭を投げつける。守銭の盾で防げる」）。最後に「上位種も同じ能力を持つ。」。

## テスト（`content_enemies_wave6b_test.go`、`//go:build !test`）

`content_enemy_foundation_test.go` の `addTestEnemy`、`content_foundation_test.go` の `newFoundationTestGame`・`equip`・`testArmor`、`content_poison_test.go` の `runQueuedActions`、`content_stage4_foundation_test.go` の `newDigTestGame`（x=3 に縦の壁がある 12x9 のマップ）を使ってよい。行動は `MonsterDefinitions[id].Behavior(g, i)` を直接呼んで確かめる（確率は `g.rng` の系列を何度か回すか、条件を満たす乱数のシードを探して固定する）。

- 表: 6体が存在し、名前・Type・能力値・属性が表どおり。`MonsterLevelUpTable` が 111→112→113・120→121→122 とつながる。出現階が表どおり（代表の階で `FloorSpawnTables` に入っている／範囲外の階には無い）。画像・上位種の色違い・出現テーブルは既存のテスト（`TestEveryBaseMonsterHasUpperSpecies`・`TestEnemySprites`・`TestUpperSpeciesSpritesOnlyRecolorOriginal`・`TestDeepFloorSpawnTables`・`TestUpperSpeciesAppearInFloorSpawnTables`）が表から自動で確かめるので、通ることを確認する。
- アナジャコ: 遠くにいる間は海老さんに近づかない（何ターン動かしても距離2以内に入ってこない、または `Alerted` が false のまま）／距離2以内に来ると追い始め、以後は離れても追う／ダメージを受けると追い始める／追うときに間の壁を掘って進み、掘った通路が `corridor` になる／外周の壁と斜めの壁は掘らない／隣にいれば通常攻撃（`false`）／封印中は掘らない／`Alerted` がセーブで保たれる。
- コバンザメ: 直線3マス先から小銭を投げ、所持金が10減り、防御力に関係なく 6 ダメージ（121・122 は 24円・14、50円・30）／所持金が額より少ないと投げない／守銭の盾で投げない／見切りの盾でかわしても所持金は減る／反射の盾でコバンザメにダメージが返る／隣・直線でない・6マス以上離れている・間に壁か敵がいると投げない／封印中は投げない／死因が「コバンザメの小銭」。
- セーブ: 6体それぞれ `enemyToSaved` → `savedToEnemy` で ID・HP・能力（`MonsterDefinitions[id].Behavior` など）が保たれる。
- `go test -run TestSimSmoke -count=1 .` で違反0（掘った穴で敵が壁の中に入らないことは不変条件 `invariants_test.go` が毎行動確かめる）。

## 完了条件

- `gofmt -w`、`go test -tags test ./...`、`go test ./...`、`go test -run TestSimSmoke -count=1 .` がすべて成功する。
- `go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1500 -sim.difficulty=all -sim.out=reports` を1回実行し、`reports/baseline.json` との差分（死因の上位に新しい敵がどれだけ入ったか、expert のクリア率と到達階、餓死の数、所持金の変化があれば）を完了報告に書く（ベースラインは更新しない）。
- `docs/agents/done/wave6-B.md` に、変更概要、IMPLEMENTED_FEATURES.md への追記案（敵の一覧と能力）、仕様の判断、未決事項（数値の妥当性・ゲーム内で確認してほしい見た目〔掘った穴の見え方、小銭が飛ぶ演出が無いこと〕）を書く。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

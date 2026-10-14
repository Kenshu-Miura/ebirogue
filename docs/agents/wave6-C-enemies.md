# wave6-C: 敵 C（ミミックダコ系 117〜119・ウチカエシエビ系 123〜125、Codex 用）

ブランチ: `agent/wave6-C-enemies`
前提: master に段4の共通基盤（`agent/content-plan-4`。`enemy_mimic.go`・`projectile.go` の打ち返し・`enemy_explosion.go`・`enemy_dig.go`・`enemies_expansion_c3.go`）と画像 img-2 が統合されていること。`enemies_expansion_c3.go` や `img/mimic_dako.png` が master に無ければ作業を始めず、その旨を報告する。
最初に `AGENTS.md`（特に「新要素追加チェックリスト」の敵の規則と「ファイル分担」の `enemy_behaviors.go`・`enemy_mimic.go`・`projectile.go`）、`docs/agents/README.md` の並列作業のルール、`docs/agents/STATUS.md`、`docs/agents/content-expansion-plan.md` を読む。`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない。
並行して wave6-A（フクレフグ系・シビレエイ系、`enemies_expansion_c1.go`）と wave6-B（アナジャコ系・コバンザメ系、`enemies_expansion_c2.go`）が動く。

## 触るファイル（ほかの担当と衝突しないための約束）

| ファイル | やること |
|---|---|
| `enemies_expansion_c3.go` | 表 `expansionC3MonsterFamilies` への2系統の追加と、行動の関数。**このタスクのコードはすべてここに書く** |
| `content_enemies_wave6c_test.go`（新規） | テスト |
| `docs/agents/done/wave6-C.md`（新規） | 完了報告 |

ほかのファイル（`enemies.go`・`enemy_expansion.go`・`enemy_mimic.go`・`projectile.go`・`monster_spawn.go`・`main.go`・`draw.go`・`help.go`・`enemy_ai.go`・`enemy_behaviors.go`・`savegame.go`）は編集しない。定義・上位種化・画像・出現階は表に書けば `enemy_expansion.go` が登録する（登録の1行は共通基盤で追加済み）。ヘルプは同じファイルの `init()` で `expansionHelpPages` にページを加える。
共通基盤に足りないものがあれば、自分で共通のファイルを直さず、完了報告に書く（このファイル内で閉じた補助関数を作るのはよい）。

## 作るもの

2系統6体。上位種は基本種の固有能力（化ける・打ち返す確率・行動・属性）をそのまま引き継ぐ（`monsterStage` には能力値と表示だけを書く）。画像はすべて img-2 で作成済み（ミミックダコ系の画像は正体を現したときの姿）。`Char` は既存の敵と重ならない小文字にした（画像があるので画面には出ない）。

| ID | 名前 | Type | 画像（`img/<>.png`） | Char | HP | 攻撃 | 防御 | 経験値 | 出現階（重み） |
|---|---|---|---|---|---|---|---|---|---|
| 117 | ミミックダコ | MimicDako | mimic_dako | m | 34 | 8 | 6 | 30 | 6〜12（5） |
| 118 | ヘンゲダコ | HengeDako | henge_dako | m | 60 | 14 | 9 | 75 | 13〜20（5） |
| 119 | センメンダコ | SenmenDako | senmen_dako | m | 102 | 28 | 11 | 188 | 21〜30（5） |
| 123 | ウチカエシエビ | UchikaeshiEbi | uchikaeshi_ebi | u | 30 | 7 | 6 | 24 | 6〜12（5） |
| 124 | ハジキカエシエビ | HajikikaeshiEbi | hajikikaeshi_ebi | u | 54 | 12 | 9 | 60 | 13〜20（5） |
| 125 | ゼンブカエシエビ | ZenbukaeshiEbi | zenbukaeshi_ebi | u | 92 | 24 | 11 | 150 | 21〜30（5） |

- 能力値の基準: 同じ出現帯のオオハサミガニ 36/10/6/26（6〜12階）・ヨロイエビ 40/9/8/30・ウミダンゴ 34/6/6/20（4〜10階）。ミミックダコ系は化けている分だけ経験値を少し多めにした。第2段階は基本種の HP×1.75・攻撃×1.7・防御+3・経験値×2.5、第3段階は第2段階の HP×1.7・攻撃×2・防御+2・経験値×2.5（`balance-level99.md`）。明らかに外れていれば直して完了報告に書く。
- 属性（`Traits`）: どちらにも付けない。

### ミミックダコ系（別の敵の姿に化ける）

基本種の定義に `Disguise: EnemyDisguiseEnemy` を書くだけで、共通基盤（`enemy_mimic.go`）が次を行う。**このタスクで同じ処理を書かない。**

- 出現時（フロアの生成・湧き・仲間呼び・遅れて現れる敵・モンスターハウスのカード）に、その階の出現テーブルから重みに従って別の敵の姿を選ぶ（`prepareSpawnedEnemy` → `pickDisguiseID`。ボス・化け貝系・ミミックダコ系は選ばない）。その階に候補が無ければ正体のまま。
- 化けている間は、画像・名前・メッセージ・ミニマップがすべて化けた姿になる（`enemyAppearance` が見た目を決め、`syncEnemyAppearance` が `Name`・`Type`・`Char` に写す）。正体の名前は `MonsterDefinitions[e.ID].Name`。能力値・行動は正体（ミミックダコ）のまま、普通の敵として動いて攻撃する。
- 攻撃を受けると正体を現す（通常攻撃・投擲・射撃・杖・`g.applyDamageToEnemy` を通るダメージ・敵の爆発）。メッセージは「<化けた姿>の正体はミミックダコだった！」。封印されている間も正体の姿。すれちがいの指輪で入れ替わっても正体は現さない。
- 上位種になっても同じ姿のまま。分裂した分身も同じ姿。中断セーブで化けた姿（`DisguiseID`）が保たれる。死因は正体の名前で記録される。

このタスクで作るのは、第2・第3段階（118・119）だけの**化け直し**:

| 系統 | 設定する項目 | 能力 |
|---|---|---|
| ヘンゲダコ・センメンダコ（118・119 だけ） | `Behavior`（`init()` で `updateMonsterDefinitions(…, 118, 119)`。117 には付けない） | 正体を現している（`!enemyMasquerading(e)`）間、`Behavior` が呼ばれるたびに `Timer++`。`Timer` が 10 以上で、海老さんが隣にいなければ、`pickDisguiseID(g.Floor, e.ID, g.randIntn)` で選んだ姿に `disguiseEnemyAs(&g.state.Enemies[i], id)` で化け直し、`Timer = 0`。海老さんから見えていれば（同じ部屋）「〜はまた姿を変えた！」を表示する（`g.enemyDisplayName` は化ける前の名前で作る）。化け直した回は `true`、それ以外は `false`（通常の行動）。候補が無ければ化けない。化けている間は `Timer` を増やさない |

### ウチカエシエビ系（飛んできた物を打ち返す）

定義の `ProjectileReturnPercent`（打ち返す確率、%）を書くだけで、共通基盤（`projectile.go` の `tryReturnProjectile`・`returnProjectile`・`hitPlayerWithReturnedProjectile`）が次を行う。**このタスクで同じ処理を書かない。**

| ID | 打ち返す確率 |
|---|---|
| 123 | 40% |
| 124 | 60% |
| 125 | 80% |

基本種の定義に `ProjectileReturnPercent: 40` を書き、上位種は `init()` で `updateMonsterDefinitions(func(d *MonsterDefinition) { d.ProjectileReturnPercent = 60 }, 124)` のように ID ごとに上書きする（上位種は基本種の値を引き継ぐため）。

- 海老さんが投げた道具・撃った矢・振った杖の魔法弾が当たる前に確率で打ち返す（「〜は〜を打ち返した！」）。打ち返しは命中判定より先なので、百発百中の指輪でも防げず、ヘタ投げの指輪でも打ち返される。封印・睡眠・金縛り中は打ち返さない。一度打ち返された物は二度と打ち返されない。
- 打ち返された物は来た道を戻り、海老さんに当たる（戻る途中にほかの敵がいれば、基本の命中率でその敵に当たる）。矢・投げた道具は「ウチカエシエビの打ち返し」の遠距離攻撃としてダメージを与え（敵に当てたときと同じ式で海老さんの防御力で減る。会心は出ない）、見切りの盾・カメの甲羅でかわせ、反射の盾ならウチカエシエビにダメージが返る。薬は海老さんに効く（睡眠薬なら眠る）。杖の魔法弾は海老さんには効果が無い。当たった物は消える。

## ヘルプ

`enemies_expansion_c3.go` の `init()` で、新しいページを1枚加える（`help.go` は編集しない）。

```go
expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（7）", Lines: []string{ … }})
```

1系統1〜2行（例:「ミミックダコ：その階に出る別の敵の姿で現れる。攻撃すると正体を現す」「ヘンゲダコ以上：正体を現してもしばらくすると化け直す」「ウチカエシエビ：投げた物・矢・魔法弾を打ち返す。上位種ほどよく打ち返す」）。最後に「上位種も同じ能力を持つ。」。

## テスト（`content_enemies_wave6c_test.go`、`//go:build !test`）

`content_enemy_foundation_test.go` の `addTestEnemy`、`content_foundation_test.go` の `newFoundationTestGame`・`equip`・`testArmor`・`testRing`、`content_poison_test.go` の `runQueuedActions`、`content_projectile_test.go` の `newProjectileTestGame`・`finishThrow`、`content_stage4_foundation_test.go` の補助を使ってよい。共通基盤そのものの挙動（化けた姿の表示・正体を現す・打ち返しの規則）は `content_stage4_foundation_test.go` が仮の敵で確かめているので、ここでは本物の ID で表と組み合わせを確かめる。

- 表: 6体が存在し、名前・Type・能力値・属性が表どおり。117〜119 の `Disguise` が `EnemyDisguiseEnemy`、123〜125 の `ProjectileReturnPercent` が 40・60・80。`MonsterLevelUpTable` が 117→118→119・123→124→125 とつながる。出現階が表どおり（代表の階で `FloorSpawnTables` に入っている／範囲外の階には無い）。画像・上位種の色違い・出現テーブルは既存のテスト（`TestEveryBaseMonsterHasUpperSpecies`・`TestEnemySprites`・`TestUpperSpeciesSpritesOnlyRecolorOriginal`・`TestDeepFloorSpawnTables`・`TestUpperSpeciesAppearInFloorSpawnTables`）が表から自動で確かめるので、通ることを確認する。
- ミミックダコ: 6〜12階のどこかで `prepareSpawnedEnemy` を通すと、その階の出現テーブルにある別の敵の姿になる（ミミックダコ系・化け貝系・ボスにはならない）／通常攻撃で正体を現す／ヘンゲダコは正体を現して10回目の行動で、隣に海老さんがいなければ化け直す（9回目では化けない、隣にいれば化けない）／ミミックダコ（117）は化け直さない／封印中は化け直さない／化けた姿と `Timer` がセーブで保たれる。
- ウチカエシエビ: 投げた道具を打ち返して海老さんがダメージを受け、ウチカエシエビは無傷（打ち返す側に乱数を固定: `g.randHooks.projectile`）／確率に外れると普通に当たる／矢・杖の魔法弾も打ち返す／眠っている・封印されたウチカエシエビは打ち返さない／死因が「ウチカエシエビの打ち返し」。
- 組み合わせ: 13〜20階でウチカエシエビ系の姿を選んだヘンゲダコに物を投げると、打ち返さずに当たって正体を現す（打ち返すかどうかは正体の定義で決まる）。
- セーブ: 6体それぞれ `enemyToSaved` → `savedToEnemy` で ID・HP・能力が保たれる。
- `go test -run TestSimSmoke -count=1 .` で違反0。

## 完了条件

- `gofmt -w`、`go test -tags test ./...`、`go test ./...`、`go test -run TestSimSmoke -count=1 .` がすべて成功する。
- `go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1500 -sim.difficulty=all -sim.out=reports` を1回実行し、`reports/baseline.json` との差分（死因の上位に新しい敵がどれだけ入ったか、特に「ウチカエシエビの打ち返し」、expert のクリア率と到達階、餓死の数、射撃・投擲の回数の変化）を完了報告に書く（ベースラインは更新しない）。expert BOT はウチカエシエビにも矢を撃つので、打ち返しで倒れる冒険が目立つなら BOT 側の対応（`bot_expert_test.go`）を完了報告の未決事項に書く（このタスクでは BOT を直さない）。
- `docs/agents/done/wave6-C.md` に、変更概要、IMPLEMENTED_FEATURES.md への追記案（敵の一覧と能力）、仕様の判断、未決事項（数値の妥当性・ゲーム内で確認してほしい見た目〔化けた姿・正体を現すとき・打ち返された道具の飛び方〕）を書く。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

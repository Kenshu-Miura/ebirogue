# wave5-B: 敵 B 後半（敵 ID 99〜107、Codex 用）

ブランチ: `agent/wave5-B-enemies`
前提: master に段3の共通基盤（`agent/content-plan-3`。`equipment_harm.go`・`enemy_behaviors.go` の被弾時の処理と分裂・`enemies_expansion_b2.go`）、段2（wave4-A/B/C）、画像 img-2 が統合されていること。`enemies_expansion_b2.go` や `img/sabi_hitode.png` が master に無ければ作業を始めず、その旨を報告する。
最初に `AGENTS.md`（特に「新要素追加チェックリスト」の敵の規則と「ファイル分担」の `enemy_behaviors.go`・`equipment_harm.go`）、`docs/agents/README.md` の並列作業のルール、`docs/agents/STATUS.md`、`docs/agents/content-expansion-plan.md` を読む。`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない。
並行して wave5-A（敵 B 前半、`enemies_expansion_b1.go`）と wave5-C（アイテム B）が動く。

## 触るファイル（ほかの担当と衝突しないための約束）

| ファイル | やること |
|---|---|
| `enemies_expansion_b2.go` | 表 `expansionB2MonsterFamilies` への3系統の追加と、特殊攻撃・被弾時の関数。**このタスクのコードはすべてここに書く** |
| `content_enemies_wave5b_test.go`（新規） | テスト |
| `docs/agents/done/wave5-B.md`（新規） | 完了報告 |

ほかのファイル（`enemies.go`・`enemy_expansion.go`・`monster_spawn.go`・`main.go`・`draw.go`・`help.go`・`enemy_ai.go`・`enemy_behaviors.go`・`equipment_harm.go`・`maptraps.go`・`enemy_inventory_attacks.go`）は編集しない。定義・上位種化・画像・出現階は表に書けば `enemy_expansion.go` が登録する（登録の1行は共通基盤で追加済み）。ヘルプは同じファイルの `init()` で `expansionHelpPages` にページを加える。
共通基盤に足りないものがあれば、自分で共通のファイルを直さず、完了報告に書く（このファイル内で閉じた補助関数を作るのはよい）。

## 作るもの

3系統9体。上位種は基本種の固有能力（特殊攻撃・確率・被弾時の処理・属性・痛恨・回避）をそのまま引き継ぐ（`monsterStage` には能力値と表示だけを書く）。画像はすべて img-2 で作成済み。`Char` は既存の敵と重ならない小文字にした。

| ID | 名前 | Type | 画像（`img/<>.png`） | Char | HP | 攻撃 | 防御 | 経験値 | 出現階（重み） |
|---|---|---|---|---|---|---|---|---|---|
| 99 | サビヒトデ | SabiHitode | sabi_hitode | s | 30 | 6 | 5 | 18 | 5〜11（6） |
| 100 | クサレヒトデ | KusareHitode | kusare_hitode | s | 55 | 11 | 8 | 45 | 12〜19（6） |
| 101 | ドロヒトデ | DoroHitode | doro_hitode | s | 94 | 22 | 10 | 112 | 20〜30（6） |
| 102 | イタズラカワハギ | ItazuraKawahagi | itazura_kawahagi | k | 22 | 4 | 3 | 12 | 3〜9（6） |
| 103 | ワルサカワハギ | WarusaKawahagi | warusa_kawahagi | k | 40 | 8 | 6 | 32 | 10〜17（6） |
| 104 | アクタレカワハギ | AkutareKawahagi | akutare_kawahagi | k | 68 | 16 | 8 | 80 | 18〜30（6） |
| 105 | オオハサミガニ | OohasamiGani | oohasami_gani | g | 36 | 10 | 6 | 26 | 6〜12（6） |
| 106 | ギロチンガニ | GuillotineGani | guillotine_gani | g | 64 | 17 | 9 | 65 | 13〜20（6） |
| 107 | ダンザイガニ | DanzaiGani | danzai_gani | g | 109 | 34 | 11 | 162 | 21〜30（6） |

- 能力値の基準: 3〜8階のイシガニ 32/7/5/15・ノロイガニ 38/8/6/22、第2段階のガンセキガニ 60/12/9/44、第3段階は第2段階の HP×1.7・攻撃×2・防御+2・経験値×2.5（`balance-level99.md`）。オオハサミガニ系は痛恨があるぶん HP を低め・攻撃を高めにした。段2の同じ出現帯の敵（wave4-B/C）とも比べ、明らかに外れていれば直して完了報告に書く。
- 出現階の重みの目安: 同じ階の既存の敵は5〜10。第3段階は9階以降だけ（`TestDeepFloorSpawnTables`）。
- 属性（`Traits`）: オオハサミガニ系に `EnemyTraitShell`（甲殻）。ほかは付けない。

### 固有の能力

すべて状態の変更（強化値・持ち物）は Action の `Execute` 内で行う（`AGENTS.md` の「落とし穴」）。被弾時の処理（`OnDamaged`）はダメージを与える Action の `Execute` の中から呼ばれるので、その中ではそのまま状態を変えてよい。

| 系統 | 設定する項目 | 能力 |
|---|---|---|
| サビヒトデ系 | `SpecialAttack`、`SpecialAttackProbability: 0.3`、`OnDamaged` | 隣接して攻撃するとき30%で、さびた汁をかける（「〜がさびた汁をかけてきた！」の Action の `Execute` で、さびの罠と同じ対象の抽選 `rollRustTarget(武器あり, 盾あり, g.randIntn)` の結果を `g.rustPlayerEquipment(対象, 1)` に渡す。武器か盾の強化値が1下がり、攻撃力・防御力にすぐ反映される。さび止め済み・金の剣/盾（`AbilityRustImmune`）・錆よけの指輪（`StatusRust`）で防ぐ。メッセージは共通処理が出す）。武器も盾も無ければ通常攻撃（`g.enqueueEnemyNormalAttack(i)`）。**被弾時**: 倒れなかったとき20%（`g.randIntn(100) < 20`）で分裂する（`g.splitEnemy(i)`。隣の空いたマスに今の HP の分身が出る。フロアの上限・空きが無ければ出ない）。分身の経験値は0にする（`g.state.Enemies[分身].ExperiencePoints = 0`。分裂させて経験値を稼げないように） |
| イタズラカワハギ系 | `SpecialAttack`、`SpecialAttackProbability: 0.3` | 隣接して攻撃するとき30%で、持ち物を1つ奪って遠くへ投げ捨てる（「〜が持ち物を奪った！」の Action の `Execute` で `g.tossPlayerItem(e.X, e.Y, 杖なら true, g.randIntn)`。装備していない持ち物から、杖があれば杖を優先して選び、ランダムな方向へ10マス飛ばして落とす。メッセージは共通処理が出す）。いたずらよけの盾（wave5-C の 164、`StatusItemToss`）で防ぐ。投げ捨てる物が無ければ「しかし何も取られなかった。」。投げ捨てた後も逃げずにその場に残る |
| オオハサミガニ系 | `CriticalPercent: 15` | 通常攻撃の15%が痛恨の一撃（×1.5、F4 の既存の仕組み。「痛恨の一撃！」と表示）。痛恨よけの盾（wave5-C の 163）で軽減する。特殊攻撃は無い |

- 封印中は `SpecialAttack`・`OnDamaged`・痛恨は働かない（共通基盤）。ゾワゾワ（wave5-C のカード）の敵は `AttackFromEnemy` が反対向きの攻撃に差し替える（特殊攻撃・痛恨も出ない）。
- `OnDamaged` は `g.applyDamageToEnemy` を通るダメージ（海老さんの攻撃・射撃・投擲・杖・カード・罠・吹き飛ばしの衝突・盾の反撃）で呼ばれ、敵どうしの攻撃では呼ばれない。範囲攻撃のループの中からも呼ばれるので、敵を増やすのは `splitEnemy`（末尾への追加）だけにする。
- `SpecialAttack`・`OnDamaged` の関数が `MonsterDefinitions` や `CreateEnemyByID` を（`splitEnemy` などほかの関数経由でも）参照すると、表の初期化が循環してコンパイルエラー（initialization cycle）になる。そのときは表には書かず、`init()` で `updateMonsterDefinitions(func(d *MonsterDefinition) { d.OnDamaged = … }, 99, 100, 101)` のように3段階すべてに設定する（`bosses.go` の `init` と同じ）。
- 乱数はすべて `g.randIntn`（敵の行動の系列）。メッセージの敵の名前は `g.enemyDisplayName(e.Name)`。

## ヘルプ

`enemies_expansion_b2.go` の `init()` で、新しいページを1枚加える（`help.go` は編集しない）。

```go
expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "新しい敵（4）", Lines: []string{ … }})
```

1系統1〜2行（例:「サビヒトデ：武器か盾をさびさせる。攻撃すると分裂することがある（分身は経験値が無い）」「イタズラカワハギ：持ち物を遠くへ投げ捨てる。杖を狙う」「オオハサミガニ：たまに痛恨の一撃を出す」）。最後に「上位種も同じ能力を持つ。」。

## テスト（`content_enemies_wave5b_test.go`、`//go:build !test`）

`content_enemy_foundation_test.go` の `addTestEnemy`、`content_foundation_test.go` の `newFoundationTestGame`・`equip`・`testWeapon`・`testArmor`・`withTestImmunity`、`content_poison_test.go` の `runQueuedActions`、`content_stage3_foundation_test.go` の `withStatusImmunityRing` を使ってよい。特殊攻撃を確実に出すには `MonsterDefinitions[id].SpecialAttack(&g.state.Enemies[i], g)` を直接呼ぶ。敵の行動の乱数は `g.rng` の系列なので、確率の境界を見るテストは `g.rng = newTurnRNG(シード)` で当たる・外れるシードを探して固定するか、判定を小さな関数（例: `sabiHitodeSplits(roll int) bool`）に切り出して境界値を確かめる。

- 表: 9体が存在し、名前・Type・能力値・属性・`CriticalPercent` が表どおり。`MonsterLevelUpTable` が 99→100→101 のようにつながる。出現階が表どおり。画像・上位種の色違い・出現テーブルは既存のテストが表から自動で確かめるので、通ることを確認する。
- サビヒトデ: 武器か盾の強化値が1下がり攻撃力・防御力が下がる／さび止め済み・`AbilityRustImmune`・`StatusRust` を防ぐ指輪では下がらない／武器も盾も無ければ通常攻撃／倒れなかったとき確率で隣に同じ種類・同じ HP の分身が出て、分身の経験値は0／倒したときは分裂しない／確率に外れると分裂しない／封印中はさびさせず分裂もしない／フロアの上限（`MaxEnemiesOnFloor`）で止まる。
- イタズラカワハギ: 杖があれば杖を、無ければほかの持ち物を投げ捨て、所持品から消えて床に落ちる／装備中の物と床に貼りついた聖域のカードは投げ捨てない／持ち物が無ければ何も起きない／`StatusItemToss` を防ぐ指輪（盾）で防ぐ／封印中は投げ捨てない。
- オオハサミガニ: `randHooks.enemyCritical` を 14 にすると痛恨（×1.5、「痛恨の一撃！」）、15 で通常／封印中は痛恨が出ない／甲殻斬りの特効が効く。
- セーブ: 9体それぞれ `enemyToSaved` → `savedToEnemy` で ID・HP・能力が保たれる。分身の経験値0も保たれる（`SavedEnemy.ExperiencePoints`）。
- `go test -run TestSimSmoke -count=1 .` で違反0。

## 完了条件

- `gofmt -w`、`go test -tags test ./...`、`go test ./...`、`go test -run TestSimSmoke -count=1 .` がすべて成功する。
- `go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1500 -sim.difficulty=all -sim.out=reports` を1回実行し、`reports/baseline.json` との差分（死因の上位に新しい敵がどれだけ入ったか、expert のクリア率と到達階、餓死の数）を完了報告に書く（ベースラインは更新しない）。
- `docs/agents/done/wave5-B.md` に、変更概要、IMPLEMENTED_FEATURES.md への追記案（敵の一覧と能力）、仕様の判断、未決事項（分裂の確率・経験値・数値の妥当性・ゲーム内で確認してほしい見た目）を書く。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

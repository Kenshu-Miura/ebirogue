# wave3-B: 盾 A（アイテム ID 148〜162、Codex 用）

ブランチ: `agent/wave3-B-shields`
前提: master に共通基盤 F1〜F5（`agent/content-foundation`）と画像 img-1（`agent/img-content-1`）が統合されていること。`img/bronze_shield.png` などが master に無ければ作業を始めず、その旨を報告する。
最初に `AGENTS.md`（特に「ファイル分担」の `equipment_abilities.go` / `equipment_hooks.go`、「新要素追加チェックリスト」）、`docs/agents/README.md` の並列作業のルール、`docs/agents/STATUS.md`、`docs/agents/content-expansion-plan.md` を読む。`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない。
並行して wave3-A（武器、`items_weapons.go`）と wave3-C（カード、`items_cards.go`）が動く。**このタスクで触る表は `items_shields.go` だけ。** `items.go` の `baseItemTemplates` には追記しない。

## 作るもの

15種。画像は img-1 の `img/<ファイル>.png`。`Type` は画像の選択に使う文字列で、`main.go` の `specialArmorImagePaths` に `"<Type>": "img/<ファイル>.png"` を追記する（`draw.go` の分岐は不要）。

| ID | 名前 | Type | 画像 | 防御力 | 能力 ID（定数名 = 文字列） | 使う仕組み | 効果 |
|---|---|---|---|---|---|---|---|
| 148 | 青銅甲の盾 | BronzeShield | bronze_shield | 4 | なし | — | 木甲と鉄甲の間の基本盾 |
| 149 | 大海の盾 | TaikaiShield | taikai_shield | 10 | なし | — | 深層の基本盾 |
| 150 | 金の盾 | GoldShield | gold_shield | 3 | `AbilityRustImmune`（既存） | なし（既存の `itemResistsRust`） | さびない |
| 151 | 竜鱗の盾 | DragonScaleShield | dragon_scale_shield | 5 | `AbilityDragonGuard` = `"dragon_guard"` | `ModifyIncomingDamage` | 竜の属性（`EnemyTraitDragon`）の敵からの攻撃と、炎（`EnemyDamageFire`）のダメージを半分（最低1） |
| 152 | にぎりよけの盾 | NigiriGuardShield | nigiri_guard_shield | 3 | `AbilityNigiriGuard` = `"nigiri_guard"` | `statusImmunities` → `StatusFoodTransform` | 持ち物をウインナーに変えられない（にぎりエビ系） |
| 153 | 操りよけの盾 | PuppetGuardShield | puppet_guard_shield | 3 | `AbilityPuppetGuard` = `"puppet_guard"` | `statusImmunities` → `StatusManipulation` | あやつりクラゲ系に操られない |
| 154 | 小判の盾 | KobanShield | koban_shield | 6 | `AbilityMoneyGuard` = `"money_guard"` | `ModifyIncomingDamage` ＋ `Effects` | 近接攻撃（`EnemyAttackMelee`）で所持金が20以上なら被ダメージ2/3（最低1）。軽減したときだけ所持金を20減らす（`incoming.Effects` に積む） |
| 155 | 鉄塊の大盾 | IronMassShield | iron_mass_shield | 7 | `AbilitySatietyGuard` = `"satiety_guard"` | `ModifyIncomingDamage` ＋ `Effects` | 近接攻撃で満腹度が10以上なら被ダメージ2/3（最低1）。軽減したときだけ満腹度を2減らす |
| 156 | 腹ペコの盾 | HungryShield | hungry_shield | 5 | `AbilityHungerGuard` = `"hunger_guard"` | `ModifyIncomingDamage` | 満腹度0のとき被ダメージ半分（最低1。敵の攻撃と罠のダメージ。餓死の1ダメージは `damagePlayer` を直接通るので対象外） |
| 157 | 根性の盾 | GutsShield | guts_shield | 6 | `AbilityGuts` = `"guts"` | `PreventDeath` | 倒れるダメージを受けたとき、1/3（`g.abilityIntn()(3) == 0`）で HP1 で踏みとどまる（「根性で踏みとどまった！」）。死因が餓死（`DamageSourceStarvation`）のときは働かない |
| 158 | 絶好調の盾 | PeakShield | peak_shield | 4 | `AbilityPeakGuard` = `"peak_guard"` | `ModifyIncomingDamage` | 攻撃を受けた時点で HP が満タンなら被ダメージ半分（最低1） |
| 159 | 悟りの盾 | SatoriShield | satori_shield | 2 | `AbilitySatori` = `"satori"` | `satietyLossInterval` の変更・満腹度の上限 | 装備中は満腹度が時間で減らない。代わりに装備中は満腹度が1を超えない（装備した時点で1にし、食べても1まで）。外せば元の最大満腹度まで食べられる（最大満腹度そのものは書き換えない） |
| 160 | 七福の御守 | LuckyCharm | lucky_charm | 5 | `AbilityLuckyGuard` = `"lucky_guard"` | `ModifyIncomingDamage` | 攻撃を受けた時点で HP の1の位が7なら被ダメージ半分（最低1） |
| 161 | 呪いの鍋ぶた | CursedPotLid | cursed_pot_lid | 7 | `AbilityCurseGuard` = `"curse_guard"` | `ModifyIncomingDamage` | この盾が呪われている間、被ダメージ−20%（4/5、最低1）。※計画の「パワー+1」は盾の効果として分かりにくいので防御側の効果にした（開発者が変えてよい） |
| 162 | カメの甲羅 | TurtleShell | turtle_shell | 3 | `AbilityShellDodge` = `"shell_dodge"` | `ModifyIncomingDamage`（`Evaded`） | 敵の遠距離攻撃（`EnemyAttackRanged`、爆発を除く）を50%でかわす（`incoming.Evaded = true`、`incoming.EvadedBy = 盾の名前`） |

163〜166 は段3の担当なので作らない。

## 仕様

- 能力 ID の定数、`registerAbilityHooks` と `statusImmunities` への登録は**新しいファイル `shield_abilities.go`（`//go:build !test`）**の `init()` にまとめる。`equipment_abilities.go` の定数ブロックと `statusImmunities` の初期値には追記しない（wave3-A と衝突するため）。
  - 例外: 悟りの盾の「満腹度が減らない」は純粋関数 `satietyLossInterval`（`equipment_abilities.go`）に分岐を足す（`shouldReduceSatiety` が false になる形）。このファイルを触るのはこの1か所だけにする。
- 被ダメージのフックの仕様は `equipment_hooks.go` の `IncomingDamage` のコメントを読むこと。フックは盾の回避・反射・属性軽減（既存の盾能力）の後、残ったダメージにだけ呼ばれる。軽減の後は `max(1, …)` で最低1にする。乱数は `g.abilityIntn()`。
- 所持金は `Player.Cash`、満腹度は `Player.Satiety`。`Effects` は実際にダメージを受けるとき（Action の Execute 内）に実行されるので、その中で減らす。かわされたときは呼ばれない。
- 悟りの盾の満腹度の上限:
  - 「満腹度を増やす処理」を `grep -n "Satiety" *.go` で探し、増やした後に `g.capSatietyForSatori()`（新規。悟りの盾を装備していれば満腹度を1に切り詰める）を呼ぶ。少なくとも食料（`restoreSatiety` など `itemeffects.go`）・腐ったウインナー・かつおぶし（wave3-A が追加する。統合時に呼び出しを足すので、完了報告に書いておく）が対象。
  - 装備した時点の切り詰めは装備の処理（`Player.EquipItem` から呼ばれる `Armor.UpdatePlayerStats`）では `Game` に触れないので、装備のメニュー処理（`executeAction` の装備の分岐）で呼ぶ。
  - 純粋な判定（`satoriSatietyCap(satiety int, equipped bool) int` など）はタグなしのファイルに置いてテストする。
- にぎりよけ・操りよけは `statusImmunities[能力] = []StatusKind{…}` を登録するだけで、にぎりエビ系・あやつりクラゲ系の特技が「〜が食料への変化を防いだ」「〜があやつりを防いだ」と表示して止まる（共通基盤で配線済み）。
- 床出現: 盾カテゴリの合計の重み（`itemCategorySpawnWeights["Armor"] = 14`）は変えない。カテゴリ内の重みも、理由が無ければ付けない。
- 識別・任意名・真実の眼・強化値・呪い・使い捨ての扱いは既存の盾と同じ（`rollItemFromTemplate` の Armor の分岐）。
- BOT（`bot_expert_test.go`）: `bestEquipment` は「防御力+強化値」で選ぶ。鉄塊の大盾（155）・腹ペコの盾（156）・悟りの盾（159）は満腹度に関わるので、重甲の盾（69）と同じく持ち替えの対象から外す。
- 説明文（`Description`）は既存の盾と同じ調子で、効果の数値を書く。
- ヘルプ（`help.go`）の道具のページに、能力のある盾（150〜162）を1行ずつ追記する。

## テスト（`content_shields_wave3_test.go`、`//go:build !test`）

`content_foundation_test.go` の補助（`newFoundationTestGame`・`equip`・`runQueuedActions`）を使ってよい。被ダメージは `enqueueEnemyNormalAttack`（近接）と `tryEnemyRangedAttack`（遠距離）、罠の効果関数で確かめる。盾の既存の回避が混ざらないよう `randHooks.shieldDefense` を 99 に固定し、`randHooks.ability` で能力の乱数を固定する。

- 15種のテンプレートが存在し、名前・Type・防御力・能力 ID が表どおり。`specialArmorImagePaths` に画像があり、ファイルが存在して 30x30。
- 能力ごとに**正常系と、効かない条件**:
  - 金の盾: さびの罠で強化値が下がらない。
  - 竜鱗の盾: 海竜系の近接攻撃と火吹きイカ系の炎が半分／ほかの敵の近接攻撃は等倍。
  - にぎりよけ・操りよけ: 特技が防がれる（メッセージに盾の名前と「防いだ」）／装備していなければ効く。
  - 小判・鉄塊: 条件を満たすと2/3で、所持金・満腹度が減る／足りなければ等倍で減らない／遠距離攻撃では働かない／盾の回避でかわしたときは減らない。
  - 腹ペコ・絶好調・七福: 条件のときだけ半分（満腹度0と1、HP 満タンと満タン−1、HP 17 と 16）。
  - 根性: 1/3 が当たると HP1 で残る／外れると倒れる／餓死では働かない／倒れないダメージでは乱数を使わない。
  - 悟り: 満腹度が時間で減らない・装備すると1になる・食べても1まで／外すと食べて回復できる・最大満腹度は変わらない。
  - 呪いの鍋ぶた: 呪われているときだけ4/5、おはらい後は等倍。
  - カメの甲羅: 遠距離攻撃を50%でかわす（「カメの甲羅で〜をかわした」）／近接・爆発はかわさない。
- セーブ→復元で能力が戻る（テンプレートから生成されるので、`savedToItem(itemToSaved(盾))` の `Abilities` を確かめる）。
- `go test -run TestSimSmoke -count=1 .` で違反0。

## 完了条件

- `gofmt -w`、`go test -tags test ./...`、`go test ./...`、`go test -run TestSimSmoke -count=1 .` がすべて成功する。
- `go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1500 -sim.difficulty=all -sim.out=reports` を1回実行し、`reports/baseline.json` との差分（餓死の数・床の道具の割合・expert のクリア率）を完了報告に書く（ベースラインは更新しない）。
- `docs/agents/done/wave3-B.md` に、変更概要、IMPLEMENTED_FEATURES.md への追記案（盾の一覧）、仕様の判断（表の※の項目、悟りの盾の上限を入れた箇所の一覧を含む）、未決事項を書く。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

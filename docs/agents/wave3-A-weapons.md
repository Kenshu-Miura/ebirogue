# wave3-A: 武器 A（アイテム ID 126〜146、Codex 用）

ブランチ: `agent/wave3-A-weapons`
前提: master に共通基盤 F1〜F5（`agent/content-foundation`）と画像 img-1（`agent/img-content-1`）が統合されていること。`img/katana.png` などが master に無ければ作業を始めず、その旨を報告する。
最初に `AGENTS.md`（特に「ファイル分担」の `equipment_abilities.go` / `equipment_hooks.go`、「新要素追加チェックリスト」）、`docs/agents/README.md` の並列作業のルール、`docs/agents/STATUS.md`、`docs/agents/content-expansion-plan.md` を読む。`AGENTS.md` と `IMPLEMENTED_FEATURES.md` は編集しない。
並行して wave3-B（盾、`items_shields.go`）と wave3-C（カード、`items_cards.go`）が動く。**このタスクで触る表は `items_weapons.go` だけ。** `items.go` の `baseItemTemplates` には追記しない。

## 作るもの

21種。画像は img-1 の `img/<ファイル>.png`。`Type` は画像の選択に使う文字列で、`main.go` の `specialWeaponImagePaths` に `"<Type>": "img/<ファイル>.png"` を追記する（`draw.go` の分岐は不要）。

| ID | 名前 | Type | 画像 | 攻撃力 | 能力 ID（定数名 = 文字列） | 使うフック | 効果 |
|---|---|---|---|---|---|---|---|
| 126 | カタナ | Katana | katana | 5 | なし | — | 中盤の基本武器 |
| 127 | 剛剣 | Gouken | gouken | 10 | なし | — | 深層の基本武器 |
| 128 | 金の剣 | GoldSword | gold_sword | 3 | `AbilityRustImmune`（既存）、`AbilityKeenEdge` = `"keen_edge"` | `BeforePlayerAttack` | さびない（さびの罠で強化値が下がらない。判定は既存の `itemResistsRust` が行うので能力を付けるだけ）。命中率 90%→95%（`attack.Accuracy += 5`） |
| 129 | 甲殻斬り | ShellCutter | shell_cutter | 5 | `AbilityShellSlayer`（既存） | なし（`slayerTraits` が処理） | 甲殻の敵（カニ・ヤドカリ・貝の各系統、鉄壁ガニ王）に×1.5 |
| 130 | ウニ割り | UniBreaker | uni_breaker | 5 | `AbilityExplosiveSlayer`（既存） | なし | 爆発の敵（バクダンウニ系）に×1.5 |
| 131 | 空の刃 | SkyBlade | sky_blade | 4 | `AbilityFloatingSlayer`（既存） | なし | 浮遊の敵（クラゲ3系統）に×1.5 |
| 132 | 連撃刀 | DoubleStrikeSword | double_strike_sword | 4 | `AbilityDoubleStrike` = `"double_strike"` | `AfterPlayerHit` | 命中して敵が倒れなかったとき、25%（`g.abilityIntn()(100) < 25`）で同じ敵へもう1回攻撃する（「連撃！」）。追撃からさらに追撃はしない（2回まで） |
| 133 | 大斧 | GreatAxe | great_axe | 6 | `AbilityCriticalUp` = `"critical_up"` | `BeforePlayerAttack` | 会心率 1/20→1/8（`attack.CriticalDenominator = min(attack.CriticalDenominator, 8)`） |
| 134 | 回復の剣 | HealingSword | healing_sword | 2 | `AbilityLifeSteal` = `"life_steal"` | `AfterPlayerHit` | 与えたダメージの1/4（最低1）だけ HP 回復（最大 HP まで。`recoveredValue`） |
| 135 | 小判こん棒 | KobanClub | koban_club | 6 | `AbilityMoneyStrike` = `"money_strike"` | `BeforePlayerAttack` ＋ `HitEffects` | 所持金（`Player.Cash`）が20以上なら×1.5。命中したときだけ所持金を20減らす（`attack.HitEffects` に積む） |
| 136 | 鉄塊の大剣 | IronMassSword | iron_mass_sword | 7 | `AbilitySatietyStrike` = `"satiety_strike"` | `BeforePlayerAttack` ＋ `HitEffects` | 満腹度が10以上なら×1.5。命中したときだけ満腹度を2減らす |
| 137 | 腹ペコ包丁 | HungryKnife | hungry_knife | 5 | `AbilityHungerStrike` = `"hunger_strike"` | `BeforePlayerAttack` | 満腹度0のとき×2 |
| 138 | 溜め打ちの剣 | ChargeSword | charge_sword | 3 | `AbilityChargeStrike` = `"charge_strike"` | `AfterPlayerMiss`・`BeforePlayerAttack` | 外れる（敵の回避を含む）たびに武器のカウンタ+1。2以上になった次の攻撃は必中かつ会心（`SureHit`・`ForceCritical`）で、カウンタを0に戻す |
| 139 | 四ツ叉の銛 | FourProngHarpoon | four_prong_harpoon | 5 | `AbilityComboStrike` = `"combo_strike"` | `AfterPlayerHit`・`AfterPlayerMiss`・`BeforePlayerAttack` | 命中でカウンタ+1、外れで0。4以上になった次の攻撃は必中かつ会心で、カウンタを0に戻す |
| 140 | 七福の熊手 | LuckyRake | lucky_rake | 5 | `AbilityLuckySeven` = `"lucky_seven"` | `BeforePlayerAttack` | HP の1の位が7のとき必ず会心（`ForceCritical`） |
| 141 | 木づち | Mallet | mallet | 5 | `AbilityMallet` = `"mallet"` | `AfterPlayerSwing` | 正面に敵がいない攻撃で、正面の罠（未発見でも）を壊して消す。罠が無ければ正面の床の道具を壊して消す。壊すたびに25%で強化値−1（装備中は攻撃力も−1。さび止めでも下がる） |
| 142 | 罠探りの棒 | TrapProbe | trap_probe | 3 | `AbilityTrapProbe` = `"trap_probe"` | `AfterPlayerSwing` | 攻撃するたびに、向いている方向の1〜2マス先と、その左右（3マス幅×2マス）の隠れた罠を発見済みにする（「罠を見つけた」）。空振りでも命中でも働く |
| 143 | かつおぶし | Katsuobushi | katsuobushi | 4 | `AbilityEdible` = `"edible"` | なし（メニュー） | 道具メニューに「かじる」を出す（装備中でも可）。満腹度+15（最大満腹度まで）、強化値−1（装備中は攻撃力も−1）、1ターン。強化値が−3のときにかじると食べきって消える（装備中なら外れる）。口封じ中はかじれない |
| 144 | 呪いの包丁 | CursedKnife | cursed_knife | 7 | `AbilityCursePower` = `"curse_power"` | `BeforePlayerAttack` | この武器が呪われている間、総攻撃力+3（`attack.AttackBonus += 3`）。※計画の「パワー+1」は総攻撃力+1にしかならず呪いの不便さに見合わないので+3にした（開発者が変えてよい） |
| 145 | 黄金のつるはし | GoldenPickaxe | golden_pickaxe | 2 | `AbilityDigWall`（既存）、`AbilityRustImmune`（既存） | なし | 壁を掘れて、さびない。※既存の海老つるはしは掘っても劣化しないので、計画の「劣化しない」は「さびない」に置き換えた |
| 146 | 火迅の太刀 | KajinTachi | kajin_tachi | 12 | `AbilityFlameBlade` = `"flame_blade"` | `AfterPlayerSwing` | HP が満タンのとき攻撃すると、向いている方向へ炎が飛び（最大10マス、`computeTrajectory`）、最初に当たった敵に固定20ダメージ（`g.applyDamageToEnemy`。正面の敵にも当たる）。ボスにも効く。壁で止まる |

147（巨大ハリセン）は段3の担当なので作らない。

## 仕様

- 能力 ID の定数とフックの登録は**新しいファイル `weapon_abilities.go`（`//go:build !test`）**にまとめる。`init()` で `registerAbilityHooks(ID, AbilityHooks{...})` する。`equipment_abilities.go` の定数ブロックには追記しない（wave3-B と衝突するため）。
- フックの中の乱数は `g.abilityIntn()`。メッセージは `g.EnqueueMessage`。フックの仕様は `equipment_hooks.go` の型のコメントを読むこと。`BeforePlayerAttack` はメッセージを積む前（命中判定の前）、`AfterPlayerHit` / `AfterPlayerMiss` はメッセージを表示する Action の Execute 内で呼ばれる。
- 溜め打ち・四ツ叉のカウンタは `Weapon` に可変フィールド `AbilityCounter int` を追加して持つ。**`SavedItem` に `AbilityCounter int \`json:",omitempty"\`` を追加し、`itemToSaved` / `savedToItem` の Weapon の分岐で保存・復元する**（`AGENTS.md` のチェックリスト4）。フックでは `attack.Weapon` / `hit.Weapon` から読む（nil のときは何もしない）。
- 連撃刀の追撃は、`AfterPlayerHit` の中から `g.enqueuePlayerAttack(敵の添字, 武器の能力)` を呼んでよい（添字は `g.enemyIndexAt(hit.Enemy.X, hit.Enemy.Y)` で引き直す）。追撃で再び追撃しないよう、`Game` に一時的なフラグ（例: `inDoubleStrike bool`、セーブしない）を持つ。追撃の命中判定・会心は通常どおり。
- 木づち・罠探りの棒・火迅の太刀の処理は `AfterPlayerSwing` で Action を積んで書く（その場で状態を変えない）。罠の削除は `g.state.MapTraps` から、道具の削除は `g.state.Items` から行い、`g.miniMapDirty = true`。罠の検索は `g.mapTrapIndexAt(x, y)`（`displacement.go`）を使う。
- 火迅の太刀の炎は死因・盾の処理の対象外（プレイヤーの攻撃なので）。炎で倒した敵の経験値は通常どおり入る。HP 満タンの判定は攻撃した時点（`AfterPlayerSwing` を呼ばれた時点）の HP で行う。
- かつおぶしの「かじる」:
  - `action.go` の `currentItemActions` に、`AbilityEdible` を持つ武器のときだけ「かじる」を加える（並びは「装備/はずす, かじる, 投げる, 置く, 説明」）。`executeAction` 側の添字での分岐がずれないよう、文字列で分岐する形にそろえるか、追加した位置を考えて直すこと。足元メニュー（`draw_ui.go` の足元の一覧）には加えない。
  - 処理は純粋関数（例: `gnawResult(satiety, maxSatiety, sharpness int) (newSatiety, newSharpness int, consumed bool)`）に分けてテストする。
- 呪いの包丁の「呪われている間」は、攻撃のたびに `attack.Weapon.Cursed` を見る（装備時に能力値を書き換えない。おはらい・ノロイガニの呪いで状態が変わっても正しく効く）。
- 床出現: 武器カテゴリの合計の重み（`itemCategorySpawnWeights["Weapon"] = 12`）は変えない。21種を追加すると既存の武器が出にくくなるが、これは計画どおり。カテゴリ内の重み（`itemSpawnWeights`）も、理由が無ければ付けない。
- 識別・任意名・真実の眼・強化値・呪いの扱いは既存の武器と同じ（`rollItemFromTemplate` の Weapon の分岐をそのまま使う）。
- BOT（`bot_expert_test.go`）: `bestEquipment` は「攻撃力+強化値」で選ぶ。鉄塊の大剣（136）と腹ペコ包丁（137）は満腹度に関わるので、重甲の盾（69）と同じく持ち替えの対象から外す。かつおぶしの「かじる」は BOT に使わせない。
- 説明文（`Description`）は既存の武器と同じ調子で、効果の数値を書く。
- ヘルプ（`help.go`）の道具のページに、能力のある武器（128〜146）を1行ずつ追記する。

## テスト（`content_weapons_wave3_test.go`、`//go:build !test`）

`content_foundation_test.go` の補助（`newFoundationTestGame`・`equip`・`runQueuedActions`）を使ってよい。乱数は `g.randHooks`（`damage`・`attackHit`・`ability`）で固定する。

- 21種のテンプレートが存在し、名前・Type・攻撃力・能力 ID が表どおり。`specialWeaponImagePaths` に画像があり、ファイルが存在する（`TestEnemySprites` の書き方を参考に、30x30 であることも確かめる）。
- 能力ごとに**正常系と、効かない条件**:
  - 金の剣: さびの罠で強化値が下がらない／命中率が95%（`attackHit` に 94 を返すと命中、95 で外れ）。
  - 特効3種: 対象の属性の敵で「特効！」とダメージ増／属性の無い敵では増えない。
  - 連撃刀: 25% が当たると2回攻撃、外れると1回／敵が1回目で倒れたら追撃しない／追撃から追撃しない。
  - 大斧: 会心判定の乱数 0〜7 で会心、8 で会心にならない。
  - 回復の剣: ダメージの1/4回復、最大 HP を超えない。
  - 小判こん棒・鉄塊の大剣: 条件を満たすと×1.5で、命中時だけ所持金・満腹度が減る／足りないときは倍率なしで減らない／外れたら減らない。
  - 腹ペコ包丁: 満腹度0で×2／1以上では等倍。
  - 溜め打ち・四ツ叉: カウンタがたまる・リセットされる・発動後に0へ戻る・セーブ→復元でカウンタが残る。
  - 七福の熊手: HP 17 で会心／HP 16 で会心にならない。
  - 木づち: 正面の罠を壊す／罠が無ければ床の道具を壊す／正面に敵がいるときは壊さない／強化値の低下（25% の当たり・外れ）。
  - 罠探りの棒: 3マス幅×2マスの隠れた罠を見つけ、範囲外は見つけない。
  - かつおぶし: かじると満腹度+15・強化値−1・1ターン消費／満腹度は最大を超えない／強化値−3でかじると消える（装備中なら外れる）／口封じ中はかじれない。
  - 呪いの包丁: 呪われているときだけ総攻撃力+3、おはらい後は+0。
  - 黄金のつるはし: 壁を掘れる・さびない。
  - 火迅の太刀: HP 満タンで炎が最初の敵に20ダメージ・壁で止まる／HP が1でも減っていたら出ない。
- `go test -run TestSimSmoke -count=1 .` で違反0。

## 完了条件

- `gofmt -w`、`go test -tags test ./...`、`go test ./...`、`go test -run TestSimSmoke -count=1 .` がすべて成功する。
- `go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1500 -sim.difficulty=all -sim.out=reports` を1回実行し、`reports/baseline.json` との差分（餓死の数・床の道具の割合・expert のクリア率）を完了報告に書く（ベースラインは更新しない）。
- `docs/agents/done/wave3-A.md` に、変更概要、IMPLEMENTED_FEATURES.md への追記案（武器の一覧）、仕様の判断（表の※の項目を含む）、未決事項を書く。
- ブランチにコミットしてプッシュする（master へは直接プッシュしない）。

# Repository Guidelines

This codebase implements a roguelike game in Go using the Ebiten library.  Each feature lives in its own file (see `README.md` for an overview).

## プロジェクト概要

- 主人公「海老さん」のターン制ローグライク。SFC シレンを参考にしたゲームデザイン。
- Go 1.24 / Ebiten v2 (`github.com/hajimehoshi/ebiten/v2`)。モジュール名は `github.com/Kenshu-Miura/ebirogue`。
- 全ソースはリポジトリ直下の単一 `main` パッケージにフラットに配置（サブパッケージは `ebitenstub/` のみ）。
- コメント・メッセージ・アイテム名などは日本語が基本。
- 画面は論理解像度 640x480（`Layout`）、タイルは 30x30 ピクセル（`tileSize`）。マップは 70x70 タイル。
- プレイヤー向け機能とシステムの実装状況は `IMPLEMENTED_FEATURES.md` を正本とする。**機能・コンテンツを追加、変更、削除する変更では、同じ変更内で同文書の該当項目も追加・修正・削除し、現在のコードと一致させること。** 未実装の予定は同文書へ混ぜず、このファイルの Implementation roadmap で管理する。

## ビルド・実行・テスト

```bash
go run .                                   # ローカル実行（ウィンドウが開く）
go build                                   # 実行ファイル生成
gofmt -w <変更したファイル>                  # コミット前に必須
go test ./...                              # 通常ビルドのテスト（content_*_test.go・TestSimSmoke など）
go test -tags test ./...                   # スタブビルドのテスト（純粋ロジック系）
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o ebirogue.wasm   # WASMビルド
go run -tags debug .                       # デバッグビルド（F1 で混乱薬を追加できる）
```

**テストは2系統ある**点に注意:

- `go test ./...`（タグなし）: 実際のゲームコード（`//go:build !test` 付きファイル）と一緒にコンパイルされる。分野別のコンテンツ結合テスト（`content_*_test.go`）、BOT のテストプレイ（`sim_test.go`）、ファジング（`fuzz_test.go`）はこちらでのみ実行される。
- `go test -tags test ./...`: Ebiten 依存ファイルを除外し、`*_stub.go` と `ebitenstub/` で代替した軽量ビルド。`autostop_test.go` や `inventory_view_test.go` など純粋ロジックのテストが対象。
- **変更時は両方実行すること。**

### BOT によるテストプレイ（シミュレーション）とファジング

```bash
go test -run TestSimSmoke -count=1 .        # 20シード×5性格×500ターン（約1秒）。go test ./... にも含まれる
go test -run TestSimBatch -count=1 -timeout 0 . -args -sim.runs=1000 -sim.difficulty=all -sim.out=reports
go test -run TestSimReplay -count=1 . -args -sim.seed=<シード> -sim.persona=<性格> -sim.difficulty=<beginner|intermediate|advanced>
go test -fuzz=FuzzInputSequence -fuzztime=30m .   # 入力列のファジング
go test -run TestEarlyDeathAnalysis -count=1 -timeout 0 . -args -sim.early=2000 -sim.out=reports   # 序盤の死亡分析（expert・上級）
go test -run TestEarlyDeathDump -count=1 . -v -args -sim.earlydump=<シード>                        # 1件の死亡直前のターンを表示
```

- BOT（`bot_test.go`）はゲーム状態を直接読み、仮想入力（`VirtualInput`）で本物のメニュー経路を操作する。性格は `aggressive` / `cautious` / `itemUser` / `diver` / `random` / `expert`。`expert`（`bot_expert_test.go`）はカード・杖・薬をテンプレート ID ごとの分類表 `itemRoles`（攻撃・逃走・強化・回復・不利）で使い分け、強化値込みで装備を持ち替え、矢を撃ち、遠距離の敵の射線を避けて探索してから降りる。バランス評価には主に `expert` の結果を使い、ほかの性格は比較用に残す。**新しいカード・杖・薬を追加したら `itemRoles` にも分類を追加する。**
- 毎行動で不変条件（`invariants_test.go`: 壁の中・重なり・HP/満腹度/所持品の範囲・座標・倒れた敵の残留）、ソフトロック（`RunUntilInputReady` が上限フレームで戻らない）、panic を検査し、ランダムなターンで中断セーブ→再開の一致も確かめる。
- `TestSimBatch` は `reports/sim-<日時>.json`（全結果）と `.md`（難易度×性格のクリア率・到達階・ターン、判断ミス死〔回復薬を持ったまま死亡・食料を持ったまま餓死・自分に不利なカードによる被害〕と死亡時のレベル・満腹度、性格別の到着時平均レベル〔主要な階〕と死亡時にレベル上限だった割合、性格別の死因上位、道具の操作回数、階層別死亡率、違反一覧、`reports/baseline.json` との差分）を出力する。`-sim.writebaseline` でベースラインを更新する。主なフラグ: `-sim.runs` `-sim.difficulty` `-sim.persona` `-sim.turns` `-sim.seedbase` `-sim.parallel` `-sim.out`、`-sim.immortal`（計測用の不死モード。HP が1未満にならず、BOT が死なずに深い階まで進むので1〜30階のレベル曲線を測れる。`Game.immortal`、`damagePlayer` で適用）。
- `TestEarlyDeathAnalysis`（`sim_early_test.go`）は expert の死亡直前40ターン（HP・行動 `bot.act`・隣の敵・遠距離攻撃が届くか・所持品など）を記録し、1〜`-sim.earlyfloor`（既定5）階の死亡を「BOT の判断ミス」と「避けにくい死」に分類した `reports/early-<日時>.md` を出力する。分類基準は `docs/agents/early-game-analysis.md`。expert は敵がいなければ HP 60% 未満で休憩し（`expertShouldRest`）、投石は直線でなくても届くと判断する。
- `reports/` は `baseline.json` 以外を `.gitignore` 済み。
- ファジングで見つかった入力は `testdata/fuzz/FuzzInputSequence/` に保存され、以後の `go test ./...` で毎回再生される（修正後もコミットして残す）。
- BOT やファジングが見つけた不具合は修正し、`regression_test.go` に回帰テストを追加する。
- ヘッドレス実行（`headless.go`）: `newHeadlessGame(seed, difficulty)` は画像を読み込まず、`fastForward` でメッセージ待ち・アニメーション・フェード・キーリピート待ちを0フレームにする。死亡・クリアで自動リスタートせず `RunEnded()` になる。`StepFrame()` で1フレーム、`RunUntilInputReady(maxFrames)` で入力待ちまで進める。

### ブラウザ配布とデプロイ（WASM / Netlify）

```bash
bash ./build_web.sh                        # public/ に index.html + ebirogue.wasm + wasm_exec.js を生成
python -m http.server 8080 --directory public   # ローカル配信（http://localhost:8080）
```

- `build_web.sh`: WASM をビルドし、`wasm_exec.js`（`$(go env GOROOT)/lib/wasm/`）と `web/index.html` を `public/` へまとめる。
- `web/index.html`: WASM ローダー。`wasm_exec.js` を読み込み `ebirogue.wasm` を起動する。読み込み・エラー表示、`instantiateStreaming` 失敗時の `arrayBuffer` フォールバック付き。
- `netlify.toml`: Netlify のビルド設定。`bash ./build_web.sh` でビルドし `public/` を公開、`GO_VERSION` を `go.mod` と揃え、`.wasm` に `Content-Type: application/wasm` を付ける。リポジトリを Netlify に接続すればプッシュのたびに自動ビルド・公開される。
- 画像は `main.go` の `//go:embed img/*.png`（`imgFS`）でバイナリへ埋め込むため、ファイルシステムの無い WASM でも `loadImage` が動く。**新しい画像を追加したら `img/*.png` に置くだけで自動的に埋め込まれる。**
- `public/` と `ebirogue.wasm` はビルド成果物なのでコミットしない（`.gitignore` 済み）。

### ブラウザでの動作テスト（Claude Code / Codex）

- `.claude/launch.json` に `public/` を配信する `web` 設定がある。`build_web.sh` を実行してからブラウザ MCP の `preview_start`（name: `web`）で起動し、`read_console_messages` / `read_network_requests` で起動を確認できる。
- **注意**: ブラウザのタブが非表示（ペイン非表示）の間は `requestAnimationFrame` が止まり、Ebiten のゲームループが進まないためキーボード入力を伴う操作テストはできない。ゲーム進行を伴うテストはペインを表示した状態で行うこと。起動・画像読み込み・`localStorage` の確認は非表示でも可能。

### ビルドタグの仕組み（重要）

| 種別 | タグ | 例 |
|---|---|---|
| Ebiten 依存のゲーム本体 | `//go:build !test` | `main.go`, `draw.go`, `input.go`, `move.go`, `enemy_ai.go`, `items.go`, `enemies.go`, `itemeffects.go`, `action.go`, `map.go`, `savegame.go`, `direction.go`, `gamehelpers.go` など |
| テスト時のスタブ | `//go:build test` | `game_stub.go`（最小限の `Game`/`Player`/`Enemy` 定義）, `draw_stub.go`, `fonts_stub.go` |
| タグなし（両ビルドで共有） | なし | `helpers.go`, `leveling.go`, `status.go`, `recovery.go`, `autostop.go`, `trajectory.go`, `messagelog.go`, `inventory_view.go`, `help.go`, `equipment_abilities.go`, `damage.go`, `savedata.go`, `ids.go`, `rng.go`, `input_source.go`, `deathcause.go`, `difficulty.go`, `item_trap_selection.go`, `displacement_path.go` |
| デバッグビルド専用 | `//go:build debug && !test` / `!debug && !test` | `debug_on.go`（F1 の処理）/ `debug_off.go`（空の実装） |

- タグなしファイルは Ebiten を import できず、`game_stub.go` の最小 `Game` 構造体でもコンパイルが通る必要がある。**タグなしのコードから `Game` の新フィールドを参照する場合は `game_stub.go` にも同じフィールドを追加する。**
- 新機能のロジックはなるべく「純粋関数をタグなしファイルへ切り出し + 単体テスト」の形にする（`autostop.go` + `autostop_test.go` が手本）。

### ファイル分担（責務ごとの分割）

- `move.go`: プレイヤーの移動・レベルアップ/死亡・`resetGame`。敵の行動AI（`MoveEnemies`/`actEnemy`/`MoveTowardsPlayer`/混乱・目潰し移動）は `enemy_ai.go`。
- `draw.go`: マップ・キャラ・アイテム・投擲・射線などワールド描画とアニメーション。HUD・ミニマップは `draw_hud.go`、インベントリ・メニュー・説明・設定などのウィンドウ描画は `draw_ui.go`。
- `direction.go`: `Direction` ⇔ 移動量 (dx, dy) の変換ヘルパー（`determineDirection`/`directionToDelta`/`getDirections` 等）。方向変換の switch を手書きしない。
- `gamehelpers.go`: 頻出パターンの共通ヘルパー。メッセージだけの Action は `g.EnqueueMessage(msg, duration)`、敵へのダメージ＋撃破処理は `g.applyDamageToEnemy(index, damage)`、投擲は `g.throwWithCallbacks(item, range)`、呪い判定は `isCursedEquipment` 等。**新規コードは同じ処理を手書きせずこれらを使う。**
- `rng.go`（タグなし）: 冒険の乱数源 `gameRNG`（math/rand/v2 の PCG。状態を保存・復元できる）。ターン進行は `g.rng`（`g.random()` / `g.randIntn(n)`）、フロア生成は `newFloorRNG(seed, 階)` の専用系列。**`math/rand` のグローバル関数やパッケージ変数の乱数源は使わない。** `Game` を受け取らない関数には `intn func(int) int` を渡す。テストで特定の判定だけ固定するには `g.randHooks`（`random_hooks.go`。ダメージ・命中・盾・状態異常耐性・装備能力・敵の痛恨・敵の回避・持ち物を落とす処理）を使う。
- `input_source.go`（タグなし）: 論理キー `Key`、`InputSource`、仮想入力 `VirtualInput`、記録 `InputRecorder`。`input_ebiten.go` が Ebiten の実キーボードを包む。**キー入力は `ebiten.IsKeyPressed` / `inpututil` を直接呼ばず `g.keys().Pressed(KeyZ)` / `JustPressed` / `PressDuration` を使う。** 入力のリピート間隔は実時間ではなくフレーム数（`stepRepeatFrames` など）。
- `keybinding.go`（タグなし・純粋）: キー設定の役割名 `keyRoles`、割り当ての補正 `normalizeKeyBindings`・入れ替え `bindPhysicalKey`、画面の案内文の置換 `keyGuide`。画面は `keybinding_ui.go`。**画面にキー名を書く文言は `g.keyGuide(...)` を通す**（ヘルプは `helpPages(bindings)`）。
- `difficulty.go`（タグなし）: 難易度 `Difficulty`、`goalFloor`、`isBossFloor`、`bossIDForFloor`、`stairsUsable`。ボス撃破の判定とクリア処理は `map.go` の `checkBossDefeated` / `clearRun`（フロアからボスが消えたら撃破）。
- `bosses.go`: ボス3体（ID 42〜44）の固有行動（殻・召喚・墨・二連撃・場所替え・炎と魔法・HP半減時の2回行動）と、ボスごとの状態異常耐性・即時消去の無効化の定数。テストは `content_bosses_test.go`（`TestSimBossBalance` は `-sim.immortal` でボス単体の撃破ターン数と被ダメージを測る）。
- `deathcause.go`（タグなし）: 死因 `DamageSource`、死亡文 `deathMessage`、冒険結果 `RunResult` と記録一覧。`runresult_game.go` の `g.damagePlayer(damage, source)` が HP 減少と死因の記録を行い、`g.finishRun(cleared)` が結果を保存する。冒険中の統計 `g.stats`（`RunStats`: 種類別の拾う・使う・投げる・装備・射撃の回数、テンプレート ID ごとの使用回数、直近の使用記録）もセーブと冒険結果に含める。**プレイヤーへのダメージは必ず `damagePlayer`（または `applyPlayerShieldDefense`）を通す。**
- `equipment_abilities.go`（タグなし・純粋）と `equipment_hooks.go`（`!test`）: 装備能力。能力 ID は `EquipmentAbilityID` 定数で、武器・盾・指輪（`Accessory.Abilities`）のテンプレートの `Abilities` に付ける。名前で判定しない。
  - 処理を持つ能力は `init()` で `registerAbilityHooks(ID, AbilityHooks{...})` する。フックは装備中の全装備品（武器・盾・指輪2つ）から、能力1つにつき1回、重ね数 `stacks` 付きで呼ばれる: `BeforePlayerAttack`（命中率・必中・倍率・会心・敵の回避率を `*PlayerAttack` で変える）、`AfterPlayerMiss`・`AfterPlayerHit`・`AfterEnemyDefeated`（攻撃の Action の Execute 内。命中したときだけの代価は `PlayerAttack.HitEffects` に積む、総攻撃力への加算は `AttackBonus`）、`AfterPlayerSwing`（通常攻撃のたび。空振り・壁掘りを含む）、`BeforeEnemyAttack`（痛恨の確率・倍率）、`ModifyIncomingDamage`（盾の回避・反射・軽減の後の被ダメージ。小銭の消費などは `Effects` に積むとダメージを受けるときに実行。`Evaded` で丸ごとかわせる）、`PreventDeath`（`damagePlayer` で HP が0以下になる直前。true で HP1）、`OnTurn`（`IncrementMoveCount` の最後）。フック内の乱数は `g.abilityIntn()`。
  - 重ね数は `g.playerAbilityCount(ID)`（2つの指輪で2）、有無は `g.playerHasAbility(ID)`。
  - 状態異常の無効化は `statusImmunities`（能力 ID → `StatusKind` の一覧）へ書くだけで効く。問い合わせは `g.playerImmuneTo(kind)`、メッセージ付きは `g.playerImmunityMessage(kind)`、盾の状態異常耐性（50%）も含めた判定は `g.playerStatusGuard(kind)`。睡眠・混乱・目潰し・鈍足・トラバサミ・毒（`drainPlayerPower`）・呪い（ノロイガニ系）・さび（さびの罠）・食料化（にぎりエビ系）・あやつり（あやつりクラゲ系）の付与箇所はすべてこれを通っている。その装備品自身だけがさびない能力は `AbilityRustImmune`（`itemResistsRust`）。**新しく状態異常を付与する処理も必ずこれらを通す。**
  - 特効は `slayerTraits`（能力 ID → 敵の属性 `EnemyTrait`）の表から引く。属性は竜・霊・一つ目・吸い取り・甲殻（カニ・ヤドカリ・貝）・爆発（ウニ）・浮遊（クラゲ）で、基本種に付ければ上位種へ引き継がれる。鼓舞の指輪のパワー補正は `accessoryPowerBonus`（`AbilityPowerBoost`）。
- `displacement_path.go`（タグなし・純粋）と `displacement.go`（`!test`）: 位置を動かす共通処理。`g.knockbackPlayer(dx, dy, マス数, 衝突ダメージ, 死因)`・`g.knockbackEnemy(添字, …)`（壁や敵に当たると固定ダメージ、ボスは動かない）、`g.pullPlayer(dx, dy, マス数)`（途中の罠で止まって踏む）、`g.dropPlayerItemsForward(個数, dx, dy)`（装備中を除き前方1〜2マス→周囲へ落とす）。不動 `AbilityImmovable`（`g.playerImmovable()`。吹き飛ばし・引き寄せ・イレカエダコ系とボスの場所替えを防ぐ）と受け身 `AbilityBreakFall`（`g.playerBreaksFall()`。衝突ダメージを防ぐ）で無効化する。どれも Action の Execute の中から呼ぶ。
- `damage.go`（タグなし・純粋）: ダメージ計算式の一元管理。SFC シレン式の「ダメージ = 攻撃力 × (15/16)^防御力 × 乱数(7/8〜9/8) × 倍率」。プレイヤー攻撃は `rollPlayerAttackDamage`（会心1/20＝防御無視×1.5、最低1ダメージ）、敵攻撃は `rollEnemyAttackDamage`、総攻撃力は `playerAttackTotal(攻撃力, パワー, レベル)`。特効などの補正は `DamageParams.Multiplier` へ乗算で合成する（`slayerMultiplier` 参照）。バランス調整は同ファイル冒頭の定数（軽減率・乱数幅・会心率など）を変更する。乱数源は呼び出し側が渡す `intn`（ゲームでは `g.damageIntn()`）で、テストでは `g.randHooks.damage` で差し替えて決定的にする。**新しいダメージ処理は式を手書きせず必ず `rollDamage` 系を経由する。**

## コアゲームループとターン進行

- `main.go` の `Game.Update()` が毎フレーム呼ばれる（60FPS 前提のフレームカウント処理が多い）。`Game.Draw()` が描画。
- 入力受付は `CanAcceptInput()`（ActionQueue にブロッキング Action が無いこと）と各種ウィンドウ表示フラグ（`showInventory`, `showMenu`, `showHelp` など）が全て false のときのみ。
- **ターン進行の流れ**:
  1. `HandleInput()`（input.go）が方向を返し、`MovePlayer(dx, dy)`（move.go）で移動、または `Z` キーで `CheckForEnemies`（攻撃, action.go）。
  2. 行動すると `isActioned = true` になる。行動を消費する操作は**これだけを行い、`AdvanceTurn()` を直接呼ばない**。
  3. 攻撃・アイテム効果・罠などはすべて `Action`（`Duration`/`Message`/`Execute`）として `g.Enqueue()` で `ActionQueue` に積まれ、`HandleActionQueue()`（animation.go）が 1 件ずつ実行してメッセージ表示とアニメーションのタイミングを制御する。`Message` は自動的に `messageLog` にも入る。
  4. キューが空になると `CheckCombatState()`（animation.go）が `AdvanceTurn()`（monster_spawn.go。ターン数・フロア滞在ターン加算、風の警告、仮眠中の敵の起床、湧きチェック）を1回だけ呼び、続けて `IncrementMoveCount()`（満腹度減少・HP自然回復・状態異常ターン減算）と `MoveEnemies()`（敵ターン, enemy_ai.go）を実行する。プレイヤーが鈍足なら敵は 2 回動く（ターン数は1だけ増える）。移動・攻撃・射撃・足踏み（`X`+`Z`）・道具の使用/投擲/装備・扉・睡眠中の自動行動はどれも1ターン、`A`+方向の方向転換やメニュー操作は0ターン（`turn_test.go`）。
- 死亡処理は `checkDeath` → `playerDead` フラグ → 死亡メッセージ → フェードアウト → `resetGame()`（move.go）。

## 主要データ構造（定義場所）

- `Game`（main.go）: 全状態のルート。`state GameState`, `rooms []Room`, 各種画像、UI フラグ多数、`ActionQueue`, ターン/湧き/フロア滞在カウンタ, `settings`, `messageLog`, `customNames` など。
- `GameState`（main.go）: `Map [][]Tile`, `Player`, `Enemies []Enemy`, `Items []Item`, `MapTraps []MapTrap`。
- `Player`（main.go）: HP/満腹度/パワー/レベル/経験値/所持金、`Inventory []Item`（最大20）、装備スロット（武器・防具・矢 各1、アクセサリ2）、`StatusAilments`。
- `Enemy`（enemies.go）: ステータス + `SpecialAttack SpecialAttackFunc` + `SpecialAttackProbability` + `Traits []EnemyTrait`（特効の属性）+ `StatusAilments`。定義 `MonsterDefinition` の `CriticalPercent`（通常攻撃が痛恨の一撃〔×1.5〕になる確率）と `EvasionPercent`（プレイヤーの通常攻撃をかわす確率。必中は無視）は敵の ID から引く（`enemyCriticalPercent` / `enemyEvasionPercent`、封印中は0、回避は睡眠・金縛り中も0）。
- `StatusAilments`（main.go / game_stub.go 両方に定義）: トラバサミ・混乱・睡眠・目潰し・鈍足・倍速・口封じ（残りターン int）、金縛り・封印（bool）、`HasteOnWake`。
- `Tile`（main.go / game_stub.go）: `Type`（"floor", "wall", "corridor", "door" 等の文字列）, `Blocked`, `BlockSight`, `Visited`, `Brightness`。
- `Room`（map.go）: `ID, X, Y, Width, Height, Center`。
- `MapTrap`（maptraps.go）: 座標・名前・発見済みフラグ・効果関数・不発率。
- アイテムは `BaseItem`（items.go）を埋め込んだ具象型 `Weapon`/`Armor`/`Arrow`/`Food`/`Potion`/`Card`/`Money`/`Accessory`/`Cane`/`Trap`/`Pot`。`Weapon`・`Armor`・`Accessory` の `Abilities []EquipmentAbilityID` はテンプレートから生成され、セーブしない。インタフェース `Item`/`Equipable`/`Identifiable`/`Character` は `interfaces.go`。
- 武具の `Sharpness` は強化値（生成時 -1〜3 のランダム、-1 なら呪い付き）。

## コンテンツのデータテーブル（新要素の追加場所）

| コンテンツ | テーブル | 生成関数 | 効果関数の置き場所 |
|---|---|---|---|
| アイテム | `itemTemplates map[int]ItemTemplate`（items.go。ID 0〜71 は `baseItemTemplates`、追加計画の武器・盾・カード・指輪は `items_weapons.go`・`items_shields.go`・`items_cards.go`・`items_rings.go` の表を `mergeItemTemplates` でまとめる。ID の重複は起動時に panic） | `buildItemFromTemplate`（テスト・復元用）/ `g.createItemByID`・`rollItemFromTemplate`（冒険の乱数） | `itemeffects.go` |
| 敵 | `MonsterDefinitions map[int]MonsterDefinition`（enemies.go, ID 0〜65。42〜44 はボス〔`IsBoss`〕、45〜65 は第3段階） | `CreateEnemyByID` | 定義内の `SpecialAttack` クロージャ |
| 罠 | `mapTrapTemplates map[int]mapTrapTemplate`（maptraps.go, 明示 ID 0〜14）。床生成・ワナのカードの重み付き抽選一覧は `floorTrapIDs` | `createMapTrapByID` | 同ファイル、移動罠は `move_traps.go`、ダメージ罠は `maptraps_damage.go`（数値は `trap_damage.go`）、装備・所持品の罠は `maptraps_items.go`（対象の抽選は `item_trap_selection.go`） |
| 階層別湧きテーブル | `FloorSpawnTables map[int][]MonsterSpawnEntry`（monster_spawn.go。1〜8階は個別、9〜30階は `deepFloorSpawnBands` の帯） | — | — |
| 装備能力 | `EquipmentAbilityID` 定数（equipment_abilities.go）。処理は `registerAbilityHooks`（equipment_hooks.go）、状態異常の無効化は `statusImmunities`、特効は `slayerTraits` | 武器・盾・指輪のテンプレートの `Abilities` に付与 | 純粋な判定は equipment_abilities.go、`*Game` を使うフックは能力ごとのファイルか equipment_hooks.go |

- アイテム効果関数（`UseAction = func(g *Game)`）は `determineItemSource(g)` で「インベントリから使ったか足元か」を判定し、最後に `removeUsedItem(g, isInventoryItem)` で消費する、というパターンが基本（itemeffects.go 冒頭参照）。
- 既存カード効果（睡眠・混乱・部屋全体・フロア全体系）はランダム部分を `rollXxx(intn func(int) int)` の純粋関数に分離して `content_cards_test.go` などでテストしている。新規効果も同じ形式にする。

## 新要素追加チェックリスト

新しいアイテム・敵・罠を 1 つ追加するときに触る場所:

1. **テーブル**: 上記の該当テーブルへ追加（ID は `docs/agents/README.md` の割り当て表に従う）。
   - **敵を追加する場合は基本種だけで終わらせず、元画像の色相と能力値を変化させた同系統の上位種（第2段階・第3段階）も同時に追加する。** 上位種は基本種の特殊能力を継承し、`MonsterLevelUpTable` に基本種→第2段階→第3段階の対応を登録する。階層で敵の能力値を補正せず、深い階の強さは上位種の出現で表す（開発者の方針）。第3段階の画像は `go run tools/third_stage_enemy_images.go` と同じ色相回転で作れる。
2. **効果**: `itemeffects.go`（アイテム）/ 定義内クロージャ（敵・罠）。
3. **画像**: `img/` に 30x30 透過 PNG。敵の上位種画像は基本種の輪郭・アルファ・明暗を保ち、色相だけを変更した色違いにする。`NewGame()`（main.go）で読み込み、`getItemImage` / `getEnemyImage` / `DrawMapTraps`（draw.go）の分岐に追加。画像は `BaseItem.Type` や `Enemy.Type` の文字列で選ばれる（例: `"Mintia"` → mintiaImg, `"Card"` → cardImg）。
4. **セーブ対応**: アイテムはテンプレート ID から再生成される（`SavedItem`, savegame.go の `itemToSaved`/`savedToItem`）ので、**新しい可変フィールドを追加した場合は `SavedItem` と両変換関数に追記**する。敵は `SavedEnemy`（ID + 可変ステータス）。罠は `MapTrap.TemplateID` の明示 ID で保存されるので、**一度使った罠 ID は変更・再利用しない**。
5. **湧き/出現**: 敵なら `FloorSpawnTables`。アイテムの床出現は `createItem` がカテゴリ（`ItemTemplate.ItemType`）を `itemCategorySpawnWeights` の重みで選び、そのカテゴリ内で `itemSpawnWeights`（書いていない ID は1）の重みで選ぶ（items.go の `itemSpawnCategories`）。**アイテムを追加してもカテゴリの割合は変わらない**（同じカテゴリのほかの道具が出にくくなる）。新しいカテゴリを作るときだけ `itemCategorySpawnWeights` に追加する（無いと床に出ない。`TestItemCategorySpawnWeights` が検出する）。
6. **テスト**: 分野別のコンテンツテスト（`content_weapons_test.go` / `content_foundation_test.go`（装備能力のフック・属性・位置を動かす処理・痛恨と回避・床出現の重み） / `content_enemies_test.go` / `content_cards_test.go` / `content_pots_test.go` / `content_traps_test.go` / `content_tables_test.go`）にテンプレート存在チェックと効果のテストを追加。`go test -run TestSimSmoke -count=1 .` で不変条件違反が無いことも確認する。
7. **説明**: 必要なら `help.go` のヘルプページも更新。

## セーブ・設定システム

- `savedata.go`（タグなし・純粋）: `SaveData`/`SavedPlayer` 等の構造体、`decodeSaveData`、破損検出の `validateSaveData`、`GameSettings`。`saveDataVersion = 3`（シード・乱数状態・難易度・ボス撃破を含む。構造を壊す変更をしたらインクリメント。旧セーブは破棄され新規冒険になる）。
- `savegame.go`（`!test`）: セーブ/設定の I/O、`buildSaveData`/`applySaveData`、メニューの「中断」→ `saveSuspendData`、フロア移動時の `autoSave`、起動時の `tryResumeFromSave`、死亡時の `deleteSaveFile`。
- 保存先は `storage.go`（`!js`, `os` ファイル）と `storage_js.go`（`js`, ブラウザ `localStorage`）が `storageRead`/`storageWrite`/`storageRemove`/`storageIsNotExist` として抽象化する。`savegame.go` は `os` を直接呼ばずこれらを使う。**新しい保存 I/O もこの4関数経由にすること。**
- キーは `ebirogue_save.json` / `ebirogue_settings.json` / `ebirogue_results.json`（冒険の記録、最大100件）。ネイティブ版はカレントディレクトリのファイル、WASM 版は同名キーの `localStorage` 項目。**WASM でも中断・オートセーブが機能する。**
- ストレージへの書き込みは `g.persistent`（`NewGame` の通常のゲームだけ true）のときだけ行う。テストや BOT の `Game` はセーブ・設定・記録を書き換えない。

## ゲーム仕様早見表

- 初期ステータス: HP 40（`initialPlayerHealth`）/ 満腹度 100 / パワー 8 / 攻撃 3 / 防御 3 / インベントリ上限 20。
- レベル（`leveling.go`、タグなし・純粋）: 上限99。必要経験値は式で生成する `playerLevelExpTable`（`expForLevel` / `levelForExp`）、最大HPの増加は `levelHealthGains`、攻撃力の補正は `levelAttackBonus`。レベルの変更は `Player.setLevel`（move.go。最大HPを差分だけ増減、将来のレベル低下もこれを使う）、経験値獲得後は `checkLevelUp` で複数レベルまとめて上がる。数値の根拠は `docs/agents/balance-level99.md` の決定事項。
- ダメージ計算（damage.go）: 攻撃力 × (15/16)^防御力 × 乱数(7/8〜9/8) × 倍率、最低1ダメージ。プレイヤーの総攻撃力 = 装備込み攻撃力 + パワー + `levelAttackBonus(レベル)`。プレイヤーの通常攻撃・射撃のみ 1/20 で会心の一撃（防御無視×1.5）。特効武器は×1.5（重複なし）。敵の痛恨の一撃は×1.5（`enemyCriticalMultiplier`、防御は通常どおり）で、`CriticalPercent` を持つ敵だけが出す（現在はいない）。
- 満腹度は 10 ターンごとに 1 減少（`satietyLossInterval`、皮甲の盾で 20 ターンに緩和、重甲の盾で 5 ターンに悪化）。満腹度 0 で毎ターン HP-1。満腹度 > 0 なら毎ターン自然回復し、量は Lv1〜9 で1・Lv10〜19 で2・Lv20 以上で3（`naturalRegenPerTurn`, leveling.go）。毒は状態異常ではなくパワー1低下の即時効果（`g.drainPlayerPower`, player_power.go。量は `poisonPowerDrain`, trap_damage.go）。毒ヘビ系・毒矢の罠・腐った食料が使い、完全回復のカードでは戻らない。
- 特殊盾は爆発・炎・魔法の半減、盗難無効、状態異常50%無効、敵攻撃25%回避、爆発以外の遠距離攻撃反射、通常攻撃への半分カウンターを提供する。使い捨ての盾は被弾ごとに強化値と防御力が1低下する。
- 地雷ダメージは現在 HP の半分（`mineTrapDamage`, status.go）、大型地雷は 3/4 切り上げ（`largeMineTrapDamage`, trap_damage.go）。木の矢・鉄の矢・落石の罠は固定の威力 6・12・16（trap_damage.go、初期HP40に合わせた値。根拠は `docs/agents/balance-level99.md` の「残った課題」）。
- 罠の効果関数は `stepOnTrap` が設定する `g.steppedTrapSource` を死因に使う（罠 ID・名前を手書きしない）。落とし穴は `availableTrapIDs` でボス階・30階の抽選から除き、落下は `requestFloorDescent` で予約し、キューが空になりターンの処理が終わってから階段と同じ `beginFloorDescent`（map.go）を通す（すぐ暗転すると残った行動が新しい階で実行される）。
- モンスター湧き: 上限 19 体、20〜30 ターン間隔、プレイヤーから 8 マス以内には湧かない（monster_spawn.go の定数）。
- フロア滞在 1200/1300 ターンで風の警告（`checkFloorTimeWarnings`）。
- 視界: 部屋単位。`updateTileBrightness`（map.go）が現在の部屋+隣接タイルのみ明るくする。敵・アイテムのミニマップ表示は同部屋・隣接・発見済みで決まる（`updateEnemyVisibility` / `updateItemVisibility`）。目潰し中は敵非表示。
- マップ生成: `GenerateRandomMap`（map.go）→ `generateRooms` → `connectRooms`（mapgen.go, 最小全域木＋ループ辺1本で接続）→ 連結性・壁健全性チェック（`floorConnected`/`roomWallsIntact`, 不合格なら再生成）→ 階段・敵・アイテム・罠配置。通路計画などの純粋ロジックはタグなしの mapgen.go にあり、mapgen_test.go / mapgen_stress_test.go でテストする。
- 主なキー操作（初期設定。設定の「キー設定」で変更でき、割り当ては `GameSettings.KeyBindings`、検証・補正・入れ替えは `keybinding.go`、物理キーの読み取りは `input_ebiten.go`）: 方向キー移動 / `Z` 攻撃（正面の罠調査を兼ねる）/ `X`+方向 ダッシュ / `A`+方向 方向転換 / `D` 矢を撃つ / Space 扉を開く / `C` メニュー / `L` メッセージ履歴 / インベントリ内 `F` 絞り込み・`S` ソート・`N` 任意名。`F1` はデバッグビルド（`-tags debug`）専用（`processF1KeyPress`, debug_on.go）。
- 難易度: 初級・中級・上級（標準）で10・20・30階のボスがクリア条件。ボス階（10・20・30階）はボスを倒すまで階段を使えず、30階より下へは進めない。新しい冒険（初回起動・死亡後・クリア後）は難易度選択画面（`openDifficultySelect`、input.go）から始まり、カーソルの初期位置は `GameSettings.LastDifficulty`。中断再開では表示しない。目標階のボス撃破後はクリア画面（`showClearScreen`）を経て選択画面へ戻る。ヘッドレス（BOT）は選択画面を出さず `newHeadlessGame` の引数の難易度で始まる。

## 落とし穴・注意点

- 演出待ち（Action の `Duration`、フェード、アニメーション）は `fastForward` で0になる。新しい演出を追加するときは、`fastForward` 中にも入力待ちへ戻ること（`isInputReady`）を確認する。
- `createItem` / `createEnemy` / 敵の罠設置は、テーブルから作った「ランダム出現の対象 ID 一覧」（`randomItemTemplateIDs` / `randomMonsterIDs` / `randomMapTrapIDs`、`ids.go` の `sortedTemplateIDs`）から抽選する。ID に欠番があっても動くが、`docs/agents/README.md` の ID 割り当て表に従うこと。`IsBoss` の敵はランダム出現の対象外。
- `Action` の `Execute` はメッセージ表示タイミングで遅延実行される。効果の副作用は `Execute` 内に書くこと（Enqueue した時点で即実行してはいけない）。`NonBlocking: true` の Action は入力を止めない。
- 敵・アイテムはスライス（`[]Enemy`, `[]Item`）で保持され、インデックスで参照する処理が多い。ループ中の要素削除は既存コードのパターン（`append(s[:i], s[i+1:]...)` 後の break 等）に合わせる。
- `Player.EquippedItems [5]Item` は後方互換のために残っている旧配列。新規コードは `EquippedWeapon`/`EquippedArmor`/`EquippedArrow`/`EquippedAccessories` を使う（equipment.go）。
- 未識別アイテムの表示名は `inventoryItemLabel`（inventory.go）と `customNames`（テンプレート ID キー）で決まる。識別状態は `Identifiable` インタフェース。
- 画像ロードは `loadImage` が失敗時 `log.Fatal` する。画像ファイルを追加し忘れると起動しない。
- **ファイル名末尾の `_windows.go` / `_js.go` / `_linux.go` 等は Go が暗黙の GOOS ビルド制約として扱う**。例えば `draw_windows.go` という名前は Windows 専用となり WASM ビルドから除外されてしまう（UI 描画ファイルは `draw_ui.go` と命名している）。
- リポジトリ直下のビルド成果物（`ebirogue.exe`, `ebirogue.wasm` 等）はコミット対象にしない。

## Coding style
- Use `gofmt -w` on all modified Go files before committing.
- Follow camelCase naming for variables and functions.
- Add comments in Japanese or English matching the surrounding code.
- Keep functions short and related logic grouped in the existing files (e.g. `input.go`, `move.go`).

## Tests
- 純粋ロジックのテストは Ebiten のスタブを利用するため `go test -tags test ./...` を実行してください。
- コンテンツ（アイテム・カード・敵）の結合テスト `content_*_test.go` と BOT スモークテスト `TestSimSmoke` は通常ビルドでのみ動くため `go test ./...` も実行してください。
- Unit tests are in `*_test.go` files.  They rely on stub files such as `draw_stub.go` and `fonts_stub.go` when built with the `test` tag.

## Pull requests
- Summaries should briefly describe the change and mention if tests were added or updated.
- Always run `go test -tags test ./...` and `go test ./...` (includes `TestSimSmoke`) before submitting a PR.

## Implementation roadmap

以下は [SFC シレンwiki](https://seesaawiki.jp/shiren1/) を参考にした実装予定です。原作の数値や挙動をそのまま複製せず、このゲームの既存バランス、海老さんの世界観、30x30ピクセルの表示に合わせて調整してください。

現在実装済みの追加要素は `IMPLEMENTED_FEATURES.md`（正本）で確認すること。新規機能を実装する前に同文書を確認し、既存要素を重複して追加せず、特殊能力や相互作用を拡張する形で進める。

実装した内容はAGENTS.mdから削除してください。画像が必要な場合は生成が望ましいですが、無理の場合はAGENTS.mdに残タスクとして記載してください。修正はコミット、プッシュしてください。

**並列開発時の例外**: `docs/agents/` の指示書に沿ってブランチで作業する場合は、`docs/agents/README.md` の「並列作業のルール」が、この段落とプロジェクト概要にあるドキュメント同時更新のルールより優先する（ドキュメントは統合担当がマージ時にまとめて更新する）。BOT テスト基盤の実装手順は `docs/agents/phase0-foundation.md` を参照。

### Priority S: clear condition, difficulty and bosses

- クリア条件は「目標階のボスを倒す」（初級10階・中級20階・上級30階）。ボスはランダム出現と上位種化の対象外で、「敵を追加するときは上位種も同時に追加する」規則の例外とする。
- 骨格（難易度・目標階・ボス階の階段封鎖・クリア判定）はフェーズ0、難易度選択・クリア画面・冒険の記録は wave1-D、ボス3体（`bosses.go`、ID 42〜44）は wave1-C で実装済み。ボスの仕様と計測方法は `docs/agents/done/wave1-C.md`。

### Priority A: cards based on scrolls

- 新規カードは既存の `img/card.png` を共用し、あかり・真空斬りを含めて階層別出現テーブル導入時に出現率を調整する。
  （白紙・ジェノサイド・聖域・全滅の特殊使用カードは実装済み。）

### Priority A: enemies and monster behavior

- 敵の特殊能力は封印状態で無効になり、目潰し・睡眠・混乱・鈍足などの共通状態処理と矛盾しないようにする。
- 新しい敵には `img/` 配下へ30x30の透過PNGを用意し、暗い床でも輪郭が判別できることを確認する。

### Priority A: traps and status ailments

- プレイヤーだけでなく敵も罠を踏むようにする。
- 矢、投擲アイテム、爆発で罠を起動できるようにし、罠を攻略へ利用可能にする。
- 追加予定の状態異常・一時効果:
  - おにぎり状態、キグニ族状態、倍速、身代わり、透明、無敵
  - 攻撃力・防御力の上昇と低下、レベル低下、回復不能（口封じは実装済み）
- 状態異常は残りターン、重複時の規則、解除手段、耐性を共通処理へまとめる。可能な状態はプレイヤーと敵の双方へ適用できるようにする。
- 新しい罠には `img/` 配下へ30x30の透過PNGを用意し、発見済み表示とミニマップ表示を確認する。
- 仮画像を正式な画像へ差し替える（wave1・wave2 でスクリプトにより仮作成した 30x30 透過 PNG）。
  - 移動罠: `img/warp_trap.png`（青い渦）、`img/pitfall_trap.png`（穴）、`img/bear_trap.png`（トラバサミ）
  - ダメージ罠: `img/wood_arrow_trap.png`、`img/iron_arrow_trap.png`、`img/falling_rock_trap.png`、`img/large_mine_trap.png`（`go run tools/damage_trap_images.go` で再生成できる単純図形）
  - 装備・所持品の罠（wave2-A）: `img/unequip_trap.png`（装備が外れる図形）、`img/spoil_trap.png`（緑の腐敗霧）、`img/scatter_trap.png`（散らばる3個の物）（`go run tools/item_trap_images.go` で再生成できる単純図形）
  - 腐ったウインナー（アイテム ID 71）: `img/rotten_sausage.png`（`sausage.png` を緑がかった色にしたもの。同じスクリプトで再生成）
  - いずれもゲーム画面上での視認性は未確認。
  - ボス（wave1-C）: `img/boss_crab.png`・`img/boss_octopus.png`・`img/boss_dragon.png`（既存の敵画像の色相を回したもの。`go run tools/boss_images.go` で再生成）。ボスらしい大きさ・形の正式画像に差し替え、暗い床と戦闘中の視認性を確認する。
  - 第3段階の敵（ID 45〜65）: 第2段階の画像の色相を150度回したもの（`tools/third_stage_enemy_images.go`）。基本種や他系統と色が似ていないか、暗い床での視認性を確認して必要なら描き直す。

### Priority A: usability

- 設定メニューを拡張する（フルスクリーン・ミニマップ表示は実装済み）。
  - BGM・効果音の音量設定（音声システム自体が未実装のため、その導入後に追加する）

### Priority B: dungeon systems

- 階層別のアイテム・罠出現テーブルを作る（敵は1〜30階で実装済み）。
- モンスターハウスを追加する。
- 店、商品の購入・売却、店主、泥棒状態を追加する。
- 武器・盾・杖の合成を追加する（保存の壺・壺拡大/吸い出しのカードは実装済み。合成の壺など壺の種類追加は未実装）。
- 水路、壊せる壁、隠し通路などの特殊地形を追加する。
- アイテム・敵・罠の図鑑と発見率を追加する。

### Priority C: replayability and diagnostics

- 通常プレイの入力履歴（`InputRecorder` が記録済み）をファイルへ保存し、シードと合わせて再生できるようにする（BOT の冒険は `TestSimReplay` でシードから再現できる）。

### Roadmap acceptance criteria

- 1つの変更では、上記の小さな実装単位を選び、無関係な大型システムを同時に追加しない。
- 新要素には生成、使用・発動、表示、説明、階層別出現、セーブ対象の各経路を確認する。
- ランダム処理は、可能な限り純粋関数や注入可能な乱数へ分離して単体テストを追加する。
- 状態異常、装備能力、敵の特殊攻撃、罠効果には、正常系と解除・無効化・境界値のテストを追加する。
- 画像を追加した場合は30x30、アルファチャンネル、透明な四隅、ゲーム内での視認性を確認する。
- 実装完了後は `gofmt -w` と `go test -tags test ./...`、`go test ./...`（`TestSimSmoke` を含む）を実行し、可能であれば通常ビルドとWASMビルドも確認する。バランスに関わる変更は `TestSimBatch` のレポートをベースラインと比べる。

# Repository Guidelines

This codebase implements a roguelike game in Go using the Ebiten library.  Each feature lives in its own file (see `README.md` for an overview).

## プロジェクト概要

- 主人公「海老さん」のターン制ローグライク。SFC シレンを参考にしたゲームデザイン。
- Go 1.24 / Ebiten v2 (`github.com/hajimehoshi/ebiten/v2`)。モジュール名は `github.com/Kenshu-Miura/ebirogue`。
- 全ソースはリポジトリ直下の単一 `main` パッケージにフラットに配置（サブパッケージは `ebitenstub/` のみ）。
- コメント・メッセージ・アイテム名などは日本語が基本。
- 画面は論理解像度 640x480（`Layout`）、タイルは 30x30 ピクセル（`tileSize`）。マップは 70x70 タイル。

## ビルド・実行・テスト

```bash
go run .                                   # ローカル実行（ウィンドウが開く）
go build                                   # 実行ファイル生成
gofmt -w <変更したファイル>                  # コミット前に必須
go test ./...                              # 通常ビルドのテスト（content_test.go など）
go test -tags test ./...                   # スタブビルドのテスト（純粋ロジック系）
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o ebirogue.wasm   # WASMビルド
```

**テストは2系統ある**点に注意:

- `go test ./...`（タグなし）: 実際のゲームコード（`//go:build !test` 付きファイル）と一緒にコンパイルされる。`content_test.go`（アイテム・カード・敵などコンテンツの結合テスト）はこちらでのみ実行される。
- `go test -tags test ./...`: Ebiten 依存ファイルを除外し、`*_stub.go` と `ebitenstub/` で代替した軽量ビルド。`autostop_test.go` や `inventory_view_test.go` など純粋ロジックのテストが対象。
- **変更時は両方実行すること。**

### ビルドタグの仕組み（重要）

| 種別 | タグ | 例 |
|---|---|---|
| Ebiten 依存のゲーム本体 | `//go:build !test` | `main.go`, `draw.go`, `input.go`, `move.go`, `enemy_ai.go`, `items.go`, `enemies.go`, `itemeffects.go`, `action.go`, `map.go`, `savegame.go`, `direction.go`, `gamehelpers.go` など |
| テスト時のスタブ | `//go:build test` | `game_stub.go`（最小限の `Game`/`Player`/`Enemy` 定義）, `draw_stub.go`, `fonts_stub.go` |
| タグなし（両ビルドで共有） | なし | `helpers.go`, `status.go`, `recovery.go`, `autostop.go`, `trajectory.go`, `messagelog.go`, `inventory_view.go`, `help.go`, `equipment_abilities.go`, `damage.go`, `savedata.go` |

- タグなしファイルは Ebiten を import できず、`game_stub.go` の最小 `Game` 構造体でもコンパイルが通る必要がある。**タグなしのコードから `Game` の新フィールドを参照する場合は `game_stub.go` にも同じフィールドを追加する。**
- 新機能のロジックはなるべく「純粋関数をタグなしファイルへ切り出し + 単体テスト」の形にする（`autostop.go` + `autostop_test.go` が手本）。

### ファイル分担（責務ごとの分割）

- `move.go`: プレイヤーの移動・レベルアップ/死亡・`resetGame`。敵の行動AI（`MoveEnemies`/`actEnemy`/`MoveTowardsPlayer`/混乱・目潰し移動）は `enemy_ai.go`。
- `draw.go`: マップ・キャラ・アイテム・投擲・射線などワールド描画とアニメーション。HUD・ミニマップは `draw_hud.go`、インベントリ・メニュー・説明・設定などのウィンドウ描画は `draw_ui.go`。
- `direction.go`: `Direction` ⇔ 移動量 (dx, dy) の変換ヘルパー（`determineDirection`/`directionToDelta`/`getDirections` 等）。方向変換の switch を手書きしない。
- `gamehelpers.go`: 頻出パターンの共通ヘルパー。メッセージだけの Action は `g.EnqueueMessage(msg, duration)`、敵へのダメージ＋撃破処理は `g.applyDamageToEnemy(index, damage)`、投擲は `g.throwWithCallbacks(item, range)`、呪い判定は `isCursedEquipment` 等。**新規コードは同じ処理を手書きせずこれらを使う。**
- `damage.go`（タグなし・純粋）: ダメージ計算式の一元管理。SFC シレン式の「ダメージ = 攻撃力 × (15/16)^防御力 × 乱数(7/8〜9/8) × 倍率」。プレイヤー攻撃は `rollPlayerAttackDamage`（会心1/20＝防御無視×1.5、最低1ダメージ）、敵攻撃は `rollEnemyAttackDamage`、総攻撃力は `playerAttackTotal(攻撃力, パワー, レベル)`。特効などの補正は `DamageParams.Multiplier` へ乗算で合成する（`slayerMultiplier` 参照）。バランス調整は同ファイル冒頭の定数（軽減率・乱数幅・会心率など）を変更する。乱数源はパッケージ変数 `damageRandInt` で、テストでは差し替えて決定的にする。**新しいダメージ処理は式を手書きせず必ず `rollDamage` 系を経由する。**

## コアゲームループとターン進行

- `main.go` の `Game.Update()` が毎フレーム呼ばれる（60FPS 前提のフレームカウント処理が多い）。`Game.Draw()` が描画。
- 入力受付は `CanAcceptInput()`（ActionQueue にブロッキング Action が無いこと）と各種ウィンドウ表示フラグ（`showInventory`, `showMenu`, `showHelp` など）が全て false のときのみ。
- **ターン進行の流れ**:
  1. `HandleInput()`（input.go）が方向を返し、`MovePlayer(dx, dy)`（move.go）で移動、または `Z` キーで `CheckForEnemies`（攻撃, action.go）。
  2. 行動すると `isActioned = true` になり `AdvanceTurn()`（monster_spawn.go）でターン数加算・湧きチェック。
  3. 攻撃・アイテム効果・罠などはすべて `Action`（`Duration`/`Message`/`Execute`）として `g.Enqueue()` で `ActionQueue` に積まれ、`HandleActionQueue()`（animation.go）が 1 件ずつ実行してメッセージ表示とアニメーションのタイミングを制御する。`Message` は自動的に `messageLog` にも入る。
  4. キューが空になると `CheckCombatState()`（animation.go）が `IncrementMoveCount()`（毒ダメージ・満腹度減少・HP自然回復・状態異常ターン減算）と `MoveEnemies()`（敵ターン, enemy_ai.go）を実行する。プレイヤーが鈍足なら敵は 2 回動く。
- 死亡処理は `checkDeath` → `playerDead` フラグ → 死亡メッセージ → フェードアウト → `resetGame()`（move.go）。

## 主要データ構造（定義場所）

- `Game`（main.go）: 全状態のルート。`state GameState`, `rooms []Room`, 各種画像、UI フラグ多数、`ActionQueue`, ターン/湧き/フロア滞在カウンタ, `settings`, `messageLog`, `customNames` など。
- `GameState`（main.go）: `Map [][]Tile`, `Player`, `Enemies []Enemy`, `Items []Item`, `MapTraps []MapTrap`。
- `Player`（main.go）: HP/満腹度/パワー/レベル/経験値/所持金、`Inventory []Item`（最大20）、装備スロット（武器・防具・矢 各1、アクセサリ2）、`StatusAilments`。
- `Enemy`（enemies.go）: ステータス + `SpecialAttack SpecialAttackFunc` + `SpecialAttackProbability` + `StatusAilments`。
- `StatusAilments`（main.go / game_stub.go 両方に定義）: 混乱・睡眠・目潰し・毒・鈍足・倍速・口封じ（残りターン int）、金縛り・封印（bool）、`HasteOnWake`。
- `Tile`（main.go / game_stub.go）: `Type`（"floor", "wall", "corridor", "door" 等の文字列）, `Blocked`, `BlockSight`, `Visited`, `Brightness`。
- `Room`（map.go）: `ID, X, Y, Width, Height, Center`。
- `MapTrap`（maptraps.go）: 座標・名前・発見済みフラグ・効果関数・不発率。
- アイテムは `BaseItem`（items.go）を埋め込んだ具象型 `Weapon`/`Armor`/`Arrow`/`Food`/`Potion`/`Card`/`Money`/`Accessory`/`Cane`/`Trap`。インタフェース `Item`/`Equipable`/`Identifiable`/`Character` は `interfaces.go`。
- 武具の `Sharpness` は強化値（生成時 -1〜3 のランダム、-1 なら呪い付き）。

## コンテンツのデータテーブル（新要素の追加場所）

| コンテンツ | テーブル | 生成関数 | 効果関数の置き場所 |
|---|---|---|---|
| アイテム | `itemTemplates map[int]ItemTemplate`（items.go, ID 0〜60 連番） | `buildItemFromTemplate` / `createItemByID` | `itemeffects.go` |
| 敵 | `MonsterDefinitions map[int]MonsterDefinition`（enemies.go, ID 0〜37 連番） | `CreateEnemyByID` | 定義内の `SpecialAttack` クロージャ |
| 罠 | `mapTrapTemplates []mapTrapTemplate`（maptraps.go） | `createMapTrapByID` | 同ファイルの効果クロージャ |
| 階層別湧きテーブル | `FloorSpawnTables map[int][]MonsterSpawnEntry`（monster_spawn.go） | — | — |
| 装備能力 | `EquipmentAbilityID` 定数（equipment_abilities.go） | テンプレートの `Abilities` に付与 | 判定ヘルパーを同ファイルへ |

- アイテム効果関数（`UseAction = func(g *Game)`）は `determineItemSource(g)` で「インベントリから使ったか足元か」を判定し、最後に `removeUsedItem(g, isInventoryItem)` で消費する、というパターンが基本（itemeffects.go 冒頭参照）。
- 既存カード効果（睡眠・混乱・部屋全体・フロア全体系）はランダム部分を `rollXxx(intn func(int) int)` の純粋関数に分離して `content_test.go` でテストしている。新規効果も同じ形式にする。

## 新要素追加チェックリスト

新しいアイテム・敵・罠を 1 つ追加するときに触る場所:

1. **テーブル**: 上記の該当テーブルへ追加（ID は連番を維持）。
   - **敵を追加する場合は基本種だけで終わらせず、元画像の色相と能力値を変化させた同系統の上位種も同時に追加する。** 上位種は基本種の特殊能力を継承し、`MonsterLevelUpTable` に基本種→上位種の対応を登録する。
2. **効果**: `itemeffects.go`（アイテム）/ 定義内クロージャ（敵・罠）。
3. **画像**: `img/` に 30x30 透過 PNG。敵の上位種画像は基本種の輪郭・アルファ・明暗を保ち、色相だけを変更した色違いにする。`NewGame()`（main.go）で読み込み、`getItemImage` / `getEnemyImage` / `DrawMapTraps`（draw.go）の分岐に追加。画像は `BaseItem.Type` や `Enemy.Type` の文字列で選ばれる（例: `"Mintia"` → mintiaImg, `"Card"` → cardImg）。
4. **セーブ対応**: アイテムはテンプレート ID から再生成される（`SavedItem`, savegame.go の `itemToSaved`/`savedToItem`）ので、**新しい可変フィールドを追加した場合は `SavedItem` と両変換関数に追記**する。敵は `SavedEnemy`（ID + 可変ステータス）。罠は**名前→テンプレート ID の対応表 `mapTrapTemplateID`（savegame.go）に必ず追加**する。
5. **湧き/出現**: 敵なら `FloorSpawnTables`。アイテムの床出現は現状 `createItem` が全テンプレートから一様ランダム。
6. **テスト**: `content_test.go` にテンプレート存在チェックと効果のテストを追加。
7. **説明**: 必要なら `help.go` のヘルプページも更新。

## セーブ・設定システム

- `savedata.go`（タグなし・純粋）: `SaveData`/`SavedPlayer` 等の構造体、`decodeSaveData`、破損検出の `validateSaveData`、`GameSettings`。`saveDataVersion = 1`（構造を壊す変更をしたらインクリメント。旧セーブは破棄され新規冒険になる）。
- `savegame.go`（`!test`）: `os.ReadFile`/`WriteFile` による I/O、`buildSaveData`/`applySaveData`、メニューの「中断」→ `saveSuspendData`、フロア移動時の `autoSave`、起動時の `tryResumeFromSave`、死亡時の `deleteSaveFile`。
- ファイルはカレントディレクトリの `ebirogue_save.json` / `ebirogue_settings.json`。**WASM ではファイル保存が失敗する**（ロードマップに localStorage 対応の残タスクあり）。

## ゲーム仕様早見表

- 初期ステータス: HP 100 / 満腹度 100 / パワー 8 / 攻撃 3 / 防御 3 / インベントリ上限 20。経験値テーブルは `levelExpRequirements`（main.go, レベル10まで）。
- ダメージ計算（damage.go）: 攻撃力 × (15/16)^防御力 × 乱数(7/8〜9/8) × 倍率、最低1ダメージ。プレイヤーの総攻撃力 = 装備込み攻撃力 + パワー + レベル。プレイヤーの通常攻撃・射撃のみ 1/20 で会心の一撃（防御無視×1.5）。特効武器は×1.5（重複なし）。
- 満腹度は 10 ターンごとに 1 減少（`satietyLossInterval`、皮甲の盾で 20 ターンに緩和）。満腹度 0 で毎ターン HP-1。満腹度 > 0 なら 5 ターンごとに HP+1 自然回復。毒は毎ターン HP-2。
- 地雷ダメージは現在 HP の半分（`mineTrapDamage`, status.go）。
- モンスター湧き: 上限 19 体、20〜30 ターン間隔、プレイヤーから 8 マス以内には湧かない（monster_spawn.go の定数）。
- フロア滞在 1200/1300 ターンで風の警告（`checkFloorTimeWarnings`）。
- 視界: 部屋単位。`updateTileBrightness`（map.go）が現在の部屋+隣接タイルのみ明るくする。敵・アイテムのミニマップ表示は同部屋・隣接・発見済みで決まる（`updateEnemyVisibility` / `updateItemVisibility`）。目潰し中は敵非表示。
- マップ生成: `GenerateRandomMap`（map.go）→ `generateRooms` → `connectRooms`（mapgen.go, 最小全域木＋ループ辺1本で接続）→ 連結性・壁健全性チェック（`floorConnected`/`roomWallsIntact`, 不合格なら再生成）→ 階段・敵・アイテム・罠配置。通路計画などの純粋ロジックはタグなしの mapgen.go にあり、mapgen_test.go / mapgen_stress_test.go でテストする。
- 主なキー操作: 方向キー移動 / `Z` 攻撃（正面の罠調査を兼ねる）/ `X`+方向 ダッシュ / `A`+方向 方向転換 / `D` 矢を撃つ / Space 扉を開く / `C` メニュー / `L` メッセージ履歴 / インベントリ内 `F` 絞り込み・`S` ソート・`N` 任意名。`F1` はデバッグ用（`processF1KeyPress`, input.go）。

## 落とし穴・注意点

- **`createItem` は `rand.Intn(len(itemTemplates))`、`createEnemy` は `rand.Intn(len(MonsterDefinitions))` で ID を引く**ため、テーブルの ID は 0 からの連番を維持しないと存在しない ID を引いてフォールバック（混乱薬 / エビ）になる。
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
- コンテンツ（アイテム・カード・敵）の結合テスト `content_test.go` は通常ビルドでのみ動くため `go test ./...` も実行してください。
- Unit tests are in `*_test.go` files.  They rely on stub files such as `draw_stub.go` and `fonts_stub.go` when built with the `test` tag.

## Pull requests
- Summaries should briefly describe the change and mention if tests were added or updated.
- Always run `go test -tags test ./...` and `go test ./...` before submitting a PR.

## Implementation roadmap

以下は [SFC シレンwiki](https://seesaawiki.jp/shiren1/) を参考にした実装予定です。原作の数値や挙動をそのまま複製せず、このゲームの既存バランス、海老さんの世界観、30x30ピクセルの表示に合わせて調整してください。

現在実装済みの追加要素は、こん棒・長巻・どうたぬき・海老薙刀・必中の剣・海老つるはし・使い捨ての大剣、木甲・鉄甲・皮甲の盾、マムル・くねくねハニー、睡眠ガス・毒矢・鈍足・地雷・サビ、毒・鈍足です。以下ではこれらを重複して追加せず、特殊能力や相互作用を拡張します。

実装した内容はAGENTS.mdから削除してください。画像が必要な場合は生成が望ましいですが、無理の場合はAGENTS.mdに残タスクとして記載してください。修正はコミット、プッシュしてください。

### Priority A: cards based on scrolls

- 新規カードは既存の `img/card.png` を共用し、あかり・真空斬りを含めて階層別出現テーブル導入時に出現率を調整する。
  （白紙・ジェノサイド・聖域・全滅の特殊使用カードは実装済み。）

### Priority A: equipment abilities

- 特殊防御を持つ盾を追加する。
  - 爆発、炎、魔法、盗難、状態異常への耐性
  - 回避率上昇、反射、カウンター
  - 高防御と引き換えに満腹度消費が増える盾
  - 高防御だが被弾ごとに弱くなる使い捨ての盾

### Priority A: enemies and monster behavior

- 敵の特殊能力は封印状態で無効になり、目潰し・睡眠・混乱・鈍足などの共通状態処理と矛盾しないようにする。
- 新しい敵には `img/` 配下へ30x30の透過PNGを用意し、暗い床でも輪郭が判別できることを確認する。

### Priority A: traps and status ailments

- 移動へ干渉する罠を追加する。
  - フロア内ワープ、次階層への落下、一定ターンの移動不能
- 装備・所持品へ干渉する罠を追加する。
  - 装備解除、食料腐敗、所持品散乱（武器や盾の強化値低下＝サビの罠は実装済み）
- ダメージ罠を追加する。
  - 木の矢、鉄の矢、落石、大型地雷
- プレイヤーだけでなく敵も罠を踏むようにする。
- 矢、投擲アイテム、爆発で罠を起動できるようにし、罠を攻略へ利用可能にする。
- 追加予定の状態異常・一時効果:
  - おにぎり状態、キグニ族状態、倍速、身代わり、透明、無敵
  - 攻撃力・防御力の上昇と低下、レベル低下、回復不能（口封じは実装済み）
- 状態異常は残りターン、重複時の規則、解除手段、耐性を共通処理へまとめる。可能な状態はプレイヤーと敵の双方へ適用できるようにする。
- 新しい罠には `img/` 配下へ30x30の透過PNGを用意し、発見済み表示とミニマップ表示を確認する。

### Priority A: usability

- 設定メニューを拡張する（フルスクリーン・ミニマップ表示は実装済み）。
  - BGM・効果音の音量設定（音声システム自体が未実装のため、その導入後に追加する）
  - キー設定（キーの割り当て変更）
- WASMビルド向けの中断セーブ保存先（ローカルストレージ等）を用意する。現在はOSのファイルシステムに保存するため、WASMでは中断セーブが失敗しメッセージを表示する。

### Priority B: dungeon systems

- 4階以降を含む階層別の敵・アイテム・罠出現テーブルを作る。
- モンスターハウスを追加する。
- 店、商品の購入・売却、店主、泥棒状態を追加する。
- 武器・盾・杖の合成を追加する（保存の壺・壺拡大/吸い出しのカードは実装済み。合成の壺など壺の種類追加は未実装）。
- 水路、壊せる壁、隠し通路などの特殊地形を追加する。
- アイテム・敵・罠の図鑑と発見率を追加する。

### Priority C: replayability and diagnostics

- ダンジョンシードを表示し、同じシードで再挑戦できるようにする。
- 死亡原因、到達階、ターン数、所持品を冒険結果として保存する。
- リプレイまたは入力履歴を保存し、不具合を再現しやすくする。
- デバッグ用F1処理は通常プレイから分離し、デバッグビルドまたは明示的な開発者設定でのみ有効にする。

### Roadmap acceptance criteria

- 1つの変更では、上記の小さな実装単位を選び、無関係な大型システムを同時に追加しない。
- 新要素には生成、使用・発動、表示、説明、階層別出現、セーブ対象の各経路を確認する。
- ランダム処理は、可能な限り純粋関数や注入可能な乱数へ分離して単体テストを追加する。
- 状態異常、装備能力、敵の特殊攻撃、罠効果には、正常系と解除・無効化・境界値のテストを追加する。
- 画像を追加した場合は30x30、アルファチャンネル、透明な四隅、ゲーム内での視認性を確認する。
- 実装完了後は `gofmt -w` と `go test -tags test ./...`、`go test ./...` を実行し、可能であれば通常ビルドとWASMビルドも確認する。

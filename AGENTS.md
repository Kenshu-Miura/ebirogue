# Repository Guidelines

This codebase implements a roguelike game in Go using the Ebiten library.  Each feature lives in its own file (see `README.md` for an overview).

## Coding style
- Use `gofmt -w` on all modified Go files before committing.
- Follow camelCase naming for variables and functions.
- Add comments in Japanese or English matching the surrounding code.
- Keep functions short and related logic grouped in the existing files (e.g. `input.go`, `move.go`).

## Tests
- テストでは Ebiten のスタブを利用するため `go test -tags test ./...` を実行してください。
- Unit tests are in `*_test.go` files.  They rely on stub files such as `draw_stub.go` and `fonts_stub.go` when built with the `test` tag.
- Run `go test -tags test ./...` to execute tests in a headless environment.

## Pull requests
- Summaries should briefly describe the change and mention if tests were added or updated.
- Always run `go test -tags test ./...` before submitting a PR.

## Implementation roadmap

以下は [SFC シレンwiki](https://seesaawiki.jp/shiren1/) を参考にした実装予定です。原作の数値や挙動をそのまま複製せず、このゲームの既存バランス、海老さんの世界観、30x30ピクセルの表示に合わせて調整してください。

現在実装済みの追加要素は、こん棒・長巻・どうたぬき、木甲・鉄甲・皮甲の盾、マムル・くねくねハニー、睡眠ガス・毒矢・鈍足・地雷、毒・鈍足です。以下ではこれらを重複して追加せず、特殊能力や相互作用を拡張します。

実装した内容はAGENTS.mdから削除してください。画像が必要な場合は生成が望ましいですが、無理の場合はAGENTS.mdに残タスクとして記載してください。修正はコミット、プッシュしてください。

### Priority A: cards based on scrolls

- フロアや部屋へ作用するカードを追加する。
  - モンスターハウス生成、敵倍速、地図忘却、拾得禁止、大部屋化、罠増加
- プレイヤー自身へ作用するカードを追加する。
  - 自爆、口封じ、攻撃力上昇、HP全回復と状況別の追加効果
- 装備へ作用するカードを追加する。
  - さび止め（武器・盾の強化値を下げる罠や敵を追加した後に実装する）
- 壺の実装後に、容量増加と中身吸い出しのカードを追加する。
- 特殊な使用方法を持つカードを追加する。
  - 任意のカード効果を書き込む白紙、投げ当てた敵系統を封じるジェノサイド
  - 床に置いて攻撃を防ぐ聖域、部屋内の敵を消滅させる全滅
- 新規カードは既存の `img/card.png` を共用し、あかり・真空斬りを含めて階層別出現テーブル導入時に出現率を調整する。

### Priority A: equipment abilities

- 特効武器を追加する。
  - ドラゴン系に強い武器
  - ゴースト系に強い武器
  - 一ツ目系に強い武器
  - 能力低下系の敵に強い武器
- 攻撃方法が変わる武器を追加する。
  - 正面3方向を攻撃する武器
  - 命中率を上げる武器
  - 壁を掘れるつるはし系武器
  - 高攻撃力だが使用ごとに弱くなる使い捨て武器
- 特殊防御を持つ盾を追加する。
  - 爆発、炎、魔法、盗難、状態異常への耐性
  - 回避率上昇、反射、カウンター
  - 高防御と引き換えに満腹度消費が増える盾
  - 高防御だが被弾ごとに弱くなる使い捨ての盾

### Priority A: enemies and monster behavior

- 遠距離攻撃役を追加する。
  - 直線上へ矢を撃つ敵
  - 障害物越しに石を投げる敵
  - 周囲を巻き込む爆発攻撃を行う敵
- 所持品へ干渉する敵を追加する。
  - アイテムを盗んで逃げる敵
  - 所持品を食料へ変える敵
  - 装備や道具を呪う敵
  - プレイヤーへ道具の使用や移動を強制する敵
- 特殊移動を行う敵を追加する。
  - 壁抜け、倍速、ワープ、場所替え
  - 移動先へ罠を作る敵
  - アイテムや階段へ擬態する敵
- モンスターがほかのモンスターを倒した場合にレベルアップする仕組みと、同系統の上位種を追加する。
- 敵の特殊能力は封印状態で無効になり、目潰し・睡眠・混乱・鈍足などの共通状態処理と矛盾しないようにする。
- 新しい敵には `img/` 配下へ30x30の透過PNGを用意し、暗い床でも輪郭が判別できることを確認する。

### Priority A: traps and status ailments

- 移動へ干渉する罠を追加する。
  - フロア内ワープ、次階層への落下、一定ターンの移動不能
- 装備・所持品へ干渉する罠を追加する。
  - 武器や盾の強化値低下、装備解除、食料腐敗、所持品散乱
- ダメージ罠を追加する。
  - 木の矢、鉄の矢、落石、大型地雷
- プレイヤーだけでなく敵も罠を踏むようにする。
- 矢、投擲アイテム、爆発で罠を起動できるようにし、罠を攻略へ利用可能にする。
- 追加予定の状態異常・一時効果:
  - おにぎり状態、キグニ族状態、倍速、身代わり、透明、無敵
  - 攻撃力・防御力の上昇と低下、レベル低下、回復不能、口封じ
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
- 壺と、武器・盾・杖の合成を追加する。
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
- 実装完了後は `gofmt -w` と `go test -tags test ./...` を実行し、可能であれば通常ビルドとWASMビルドも確認する。

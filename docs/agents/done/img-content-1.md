# img-1: 追加コンテンツ用の画像

作業日: 2026-10-14。ブランチ: `agent/img-content-1`。
起点: master `b30e6d95eaa69dbb7362dd63d81e18558ea50904`。

## 成果物

- `img/` に新規60枚: 武器126〜146の21枚、盾148〜162の15枚、敵66〜89の24枚。
- すべて30×30 RGBA PNG。四隅と外周1pxは透明。外側の半透明なし。
- 既存画像を上書きせず、ゲームコード・AGENTS.md・IMPLEMENTED_FEATURES.mdは変更していない。
- [床上の4倍一覧](img-content-1-sheet.png)。各画像の左はRGB(38,25,21)、右はRGB(95,97,102)。
- 全ファイル名: [`tools/img1-files.txt`](../../../tools/img1-files.txt)。

## 作り方と再生成

既存の ebi、hayate_shako、fire_squid、sea_dragon、weapon、dragon_killer、
sure_hit_sword、armor、counter_shield、shrimp_pickaxe の10枚を開いて参照した。
原画44枚は built-in image_gen で個別生成（透明背景）。上位種16枚は原画を別生成せず色違いにした。
採用原画は `tools/img1-originals/`、被写体一覧とプロンプトは
[`tools/img1-manifest.txt`](../../../tools/img1-manifest.txt) と
[`tools/img1-prompts.md`](../../../tools/img1-prompts.md) に保存。

`process_expansion_sprites.py` はアルファ160未満を除去して切り抜き、
最大26pxへLANCZOS縮小、アルファ128を境に二値化、31色へ減色、
近傍8方向にRGB(3,2,1)の1px輪郭を追加する。
輪郭を含む最大サイズは28px。原画生成は非決定的だが、保存原画からの加工は再現できる。

**統合時の変更（2026-10-14）**: 原画44枚（約30MB）は master に入れていない（履歴から消せず、リポジトリが重くなるため）。原画はブランチ `agent/img-content-1`（コミット `d9dbc42`）に残っている。再加工するときは先に原画を取り出す:

```powershell
git fetch origin agent/img-content-1
git checkout d9dbc42 -- tools/img1-originals
git restore --staged tools/img1-originals   # .gitignore 済み。コミットしない
```

リポジトリ直下から実行（Python + Pillow、Go）:

```powershell
python tools/process_expansion_sprites.py
go run tools/expansion_enemy_images.go
$img1Files = Get-Content tools/img1-files.txt
go run tools/sprite_check.go -sheet docs/agents/done/img-content-1-sheet.png $img1Files
python tools/verify_img1.py
```

`sprite_check.go` はimg-2の名前も登録済み。任意の列挙した画像を検査でき、
違反時は理由を列挙して終了コード1、合格時は0で終わる。
RGBA、サイズ、透明余白、不透明率、外側の半透明、2色の床との平均輪郭明度差を検査する。
陰性対照として余白なし・不透明率100%・床と同明度の画像を与え、3違反と終了コード1を確認した。

## 上位種と色相

`expansion_enemy_images.go` の表を正本とし、第2段階は基本種から、
第3段階は第2段階から生成。HSLの色相を回し、アルファと明度は保存する。
灰色に近い画素は既存スクリプトと同じ条件で彩度を少し加える。

| 系統 | 基本→第2 | 第2→第3 |
|---|---:|---:|
| ヒバナウミウシ | +60° | +150° |
| テッポウエビ | +60° | +150° |
| ニゲハゼ | +60° | +60° |
| ヨビコミボラ | +60° | +150° |
| イヤシナマコ | +60° | +150° |
| ゼニガタアザラシ | +60° | +150° |
| スナモグリ | +60° | +150° |
| ヨロイエビ | +60° | +150° |

ニゲハゼは基本種が紫（主な色相260〜290°）なので、初期値では第3段階が青緑になる。
金ピカハゼの指定に合わせ、第3段階を+60°へ変更して金茶色へ寄せた。
途中の+90°では黄緑が強かったため再調整した。形とアルファ、明度は同じ。

## 目視確認と描き直し

途中と最終の床上一覧を自分で開き、系統・武具の形、輪郭の読み取りを確認した。
最近傍縮小は細部が点状に残ったため採用せず、LANCZOS後の減色を採用。

- `uni_breaker`: 当初は細い曲刀でカタナに似ていた。幅広で深く曲がった三日月の刀身へ描き直した。
- `double_strike_sword`: 当初の残像が刀身に近すぎた。離した紫の残像を含む二重の刃へ描き直した。
- `dragon_scale_shield`: 当初の細かい鱗と中央の竜紋が縮小で潰れた。中央の紋をなくし、大きな鱗へ描き直した。
- 金ピカハゼは上記の色相調整で修正。床に溶ける外周は共通の不透明輪郭で防いだ。

## 検証

- 全60枚の `sprite_check.go`: 違反ゼロ。
- 保存原画から再加工・上位種再生成して、60枚のSHA-256が一致することを確認。
- 上位種16枚のアルファが元画像と完全一致し、HSL明度が丸め誤差内で一致することを確認。
- `gofmt -w tools/expansion_enemy_images.go tools/sprite_check.go` 済み。
- `go test -tags test ./...`: 成功。
- `go test ./...`（TestSimSmoke含む）: 成功。
  通常版テストの初回はサンドボックスの一時ディレクトリ権限で失敗したため、
  サンドボックス外で再実行して成功を確認した。ゲームコードの修正は行っていない。

## 統合時に確認してほしい点

今回は画像のみで、ゲームへの読み込み・描画分岐・テンプレート登録は後の実装担当の範囲。
組み込み後は実寸で、敵の既存系統との区別、武者エビの兜と大砲エビの鋏、
ナマコのハート・ほら貝の音符、盾の×印や拳・太陽・円相、
呪いの紫の気と炎の刀がキャラクターや床アイテムに重なった状態で読めるか確認してほしい。
一覧での床上視認性は確認済み。実際のゲーム画面での確認は未実施。

IMPLEMENTED_FEATURES.mdへの追記案: 画像だけではプレイヤー向け機能は増えないので現時点の追記なし。
各機能の統合時に対応する素材名を記載する。
AGENTS.mdから削除する項目: なし（既存の仮画像差し替えタスクはこの60枚とは別）。
正式画像として作成し、仮画像として差し替えが必要なものはなし。

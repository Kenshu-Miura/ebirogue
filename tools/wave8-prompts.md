# 段8 仮画像の差し替え用プロンプト

段8で仮に作った3枚を正式な画像へ差し替えるための、原画の生成プロンプトと加工手順。
既存の画像（img-1/img-2・wave7-D）と同じ流れで作る。原画はコミットしない（`tools/*-originals/` は `.gitignore` 済み）。

| ファイル | 使われ方 | 今の仮画像 |
|---|---|---|
| `img/no_healing_trap.png` | 回復封じの罠（ID 15）。発見済みの罠としてマップとミニマップの縮小表示に出る | 回復の十字を斜線で封じた図形 |
| `img/player_nigiri.png` | にぎり状態の海老さん（`ebisan.png` の代わり）。**向きに合わせて回転して描く（回転なし＝上向き）** | 白いシャリにエビを乗せた図形 |
| `img/enemy_decoy.png` | 身代わり状態の敵（敵の画像の代わり）。どの敵が身代わりになっても同じ画像 | 茶色の人形の図形 |

## 生成の設定

- 画像生成で1枚ずつ生成し、背景は透過（transparent_background=true）。
- 生成した原画は `tools/wave8-originals/<ファイル名>.png`（`no_healing_trap.png`・`player_nigiri.png`・`enemy_decoy.png`）として保存する。

## 共通プロンプト

各 Subject の前に連結する。

```text
Use case: stylized-concept. Asset type: single game sprite original for a 30x30 roguelike tile. Create exactly ONE isolated object, centered in a square transparent image. Saturated SNES pixel art matching a Japanese sea creature roguelike: chunky readable silhouette, dark brown almost black outline, 3-4 flat shading bands, light from upper left, large simple details readable when reduced to 26 pixels. No text, letters, numbers, watermark, ground, cast shadow, background, framing, or extra objects. Leave generous transparent padding. Subject:
```

## Subject

### no_healing_trap（回復封じの罠）

ほかの罠（`slow_trap.png`・`poison_arrow_trap.png` など）と同じ、上から見た四角い床の板にそろえる。

```text
A square stone pressure plate seen straight from above (top-down), dark slate grey with a beveled lighter rim and small corner rivets, matching a set of roguelike floor trap plates. In the center, a bright red healing cross symbol sealed by a thick black diagonal slash and two small iron chains crossing over it, conveying "healing is blocked". Flat top-down view, no perspective, the plate fills most of the frame
```

### player_nigiri（にぎり状態の海老さん）

主人公の海老さん（`ebisan.png`）がにぎり寿司にされた姿。上向きを基準に回転されるので、上下がはっきりした左右対称の形にする。

```text
A single piece of shrimp nigiri sushi seen from directly above (top-down), oriented vertically with the shrimp tail fan at the TOP edge: an oval mound of white rice with visible grain texture, topped by a butterflied orange-and-white striped cooked shrimp, and a thin dark green nori band wrapped around the middle. A small cute surprised human face with dark hair, round eyes and an open mouth peeks out from the rice at the BOTTOM end, so it reads as the hero turned into sushi. Left-right symmetric composition, chunky outline, readable when rotated in 45 degree steps
```

### enemy_decoy（身代わり状態の敵）

敵の代わりに描かれる身代わり人形。ほかの敵（三・四分の一正面の全身）と並んでも、生き物ではなく「人形」と分かる形にする。

```text
A small straw-and-wood scarecrow decoy doll standing upright, three-quarter frontal full body: a round burlap sack head with a simple stitched X-eye face and a drawn red target circle on its chest, arms made of a horizontal wooden stick with straw tufts at the ends, a straw skirt body and a single wooden stake as its leg. Tan and light brown palette with a red target accent so it stands out on dark floors, clearly an inanimate dummy, not a creature
```

## 加工

```bash
python tools/process_expansion_sprites.py --originals tools/wave8-originals --count 3
```

アルファ160でにじみを除き、26pxへ縮小、31色に減色、外周の半透明を除去、暗色1pxの輪郭を付けて30x30 RGBA で `img/<同じ名前>.png` を上書きする。

## 確認

- `go test ./...`（`TestSpecialNigiriPlaceholder` が `player_nigiri.png` の30x30と透明な四隅を確かめる）と `go test -tags test ./...`。
- 30x30・RGBA・四隅が透明であること。
- 実機（`go run .`）で確認すること:
  - 回復封じの罠: 発見済みの罠として床に表示されたとき、ほかの罠と見分けられるか。ミニマップの表示。
  - にぎりの海老さん: 8方向に回転しても崩れないか。透明の半透明表示（透明の薬）と重なったときも見えるか。
  - 身代わり: 暗い床で輪郭が分かるか。ほかの敵の中で身代わりだと分かるか。
- 差し替えたら `AGENTS.md` の Implementation roadmap の「段8の仮画像」の行を削除する。

## 差し替え結果

指定の共通プロンプト＋各 Subject を組み込み画像生成（透過）へ渡し、3枚を生成・加工済み。原画は `tools/wave8-originals/`（Git対象外）。

`python tools/wave8_sheet.py` で規格検査と `tools/wave8-originals/review/sheet.png` の再生成ができる。実際の床画像に重ねた原寸・3倍表示、にぎりの8方向とアルファ0.4の比較を目視確認済み。このシートはPillowによる比較で、実機の描画確認とは別。

ミニマップの発見済み罠は `draw_hud.go` で共通の赤い×を描き、罠画像は縮小しない。実機操作による発見済み罠・8方向・透明・身代わりの確認は未実施。

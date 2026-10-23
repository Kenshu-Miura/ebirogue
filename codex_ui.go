//go:build !test

package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"image/color"
	"strings"
)

// 図鑑の画面の配置。左に項目の一覧、右に選んだ項目の詳細を出す。
const (
	codexListX, codexListY, codexListW = 28, 106, 200 // 一覧の枠
	codexRowHeight                     = 18
	codexListRows                      = 17       // 一覧に見える行数
	codexNameRunes                     = 13       // 一覧の名前の最大文字数
	codexDetailX, codexDetailW         = 242, 372 // 詳細の枠
	codexDescWidth, codexDescLines     = 28, 12   // 説明文の1行の文字数と行数
)

var codexWhite *ebiten.Image

// codexFillRect は単色の四角を描く（毎フレーム画像を作らないよう1x1の白を拡大する）。
func codexFillRect(screen *ebiten.Image, x, y, w, h int, c color.Color) {
	if codexWhite == nil {
		codexWhite = ebiten.NewImage(1, 1)
		codexWhite.Fill(color.White)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(w), float64(h))
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	screen.DrawImage(codexWhite, op)
}

// codexFrame は枠線付きの暗い領域を描く。
func codexFrame(screen *ebiten.Image, x, y, w, h int) {
	codexFillRect(screen, x-2, y-2, w+4, h+4, color.NRGBA{200, 200, 220, 255})
	codexFillRect(screen, x, y, w, h, color.NRGBA{10, 12, 22, 255})
}
func codexRateLabel(ids []int, records map[int]CodexRecord) string {
	n, m, p := codexDiscovery(ids, records)
	return fmt.Sprintf("発見 %d / 全 %d（%.1f%%）", n, m, p)
}

// codexEnemyDescription は敵の図鑑の説明文を返す。定義の Description を使い、
// ヘルプの文章を名前で探すことはしない（「エビ」で「ユウレイエビ」の行が見つかる取り違えを防ぐ）。
func codexEnemyDescription(id int) string {
	return MonsterDefinitions[id].Description
}

// codexTrapDescription は罠の図鑑の説明文を返す。
func codexTrapDescription(id int) string {
	return mapTrapTemplates[id].Description
}
func codexWrap(s string, width, limit int) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		r := []rune(line)
		for len(r) > width {
			out = append(out, string(r[:width]))
			r = r[width:]
		}
		out = append(out, string(r))
	}
	if len(out) > limit {
		out = out[:limit]
		r := []rune(out[limit-1])
		if len(r) > 0 {
			out[limit-1] = string(r[:len(r)-1]) + "…"
		}
	}
	return out
}

// codexEntry は図鑑の1項目の表示内容。
type codexEntry struct {
	name, desc string
	info       []string
	img        *ebiten.Image
}

var codexItemCategories = map[string]string{"Money": "小銭", "Food": "食料", "Potion": "薬", "Weapon": "武器", "Armor": "盾", "Arrow": "矢", "Accessory": "指輪", "Cane": "杖", "Card": "カード", "Pot": "壺", "Trap": "罠カード"}

// codexEntryFor は図鑑の項目の名前・説明・情報・画像を返す。未発見は「？？？」。
func (g *Game) codexEntryFor(tab, id int, r CodexRecord) codexEntry {
	e := codexEntry{name: "？？？", desc: "まだ見つけていない。"}
	if !r.Seen {
		return e
	}
	switch tab {
	case 0:
		t := itemTemplates[id]
		e.name, e.desc = t.Name, t.Description
		if !r.Identified {
			e.name, e.desc = "未識別の道具", "識別すると名前と説明が表示されます"
		}
		e.info = []string{codexItemCategories[t.ItemType]}
		e.img = g.getItemImage(rollItemFromTemplate(id, 0, 0, func(int) int { return 0 }))
	case 1:
		d := MonsterDefinitions[id]
		e.name, e.desc = d.Name, codexEnemyDescription(id)
		e.info = []string{
			fmt.Sprintf("HP %d  攻 %d  防 %d  経験値 %d", d.MaxHealth, d.AttackPower, d.DefensePower, d.ExperiencePoints),
			fmt.Sprintf("撃破 %d  /  敗北 %d", r.Defeated, r.Deaths),
		}
		e.img = g.getEnemyImage(Enemy{ID: id, Type: d.Type, Name: d.Name})
	case 2:
		e.name, e.desc = mapTrapTemplates[id].Name, codexTrapDescription(id)
		e.info = []string{"未踏"}
		if r.Stepped {
			e.info = []string{"踏んだことがある"}
		}
		e.img = g.codexTrapImage(e.name)
	}
	return e
}

// codexListLabel は一覧の1行（番号と名前。長い名前は省略）を返す。
func codexListLabel(id int, name string) string {
	if r := []rune(name); len(r) > codexNameRunes {
		name = string(r[:codexNameRunes-1]) + "…"
	}
	return fmt.Sprintf("%03d %s", id, name)
}

func (g *Game) drawCodex(screen *ebiten.Image) {
	drawRunPanel(screen, "図鑑")
	tables := []map[int]CodexRecord{g.codex.Items, g.codex.Enemies, g.codex.Traps}
	total, found := 0, 0
	for k := 0; k < 3; k++ {
		n, m, _ := codexDiscovery(codexIDs(k), tables[k])
		found += n
		total += m
	}
	percent := 0.0
	if total > 0 {
		percent = 100 * float64(found) / float64(total)
	}
	text.Draw(screen, fmt.Sprintf("全体：発見 %d / 全 %d（%.1f%%）", found, total, percent), mplusSmallFont, 300, 50, color.White)

	// タブ（選択中は黄色）と、そのタブの発見率
	selected := color.NRGBA{255, 230, 80, 255}
	dim := color.NRGBA{150, 150, 170, 255}
	tabX := 30
	for k, label := range []string{"道具", "敵", "罠"} {
		c := color.Color(dim)
		if k == g.codexTab {
			c = selected
			label = "［" + label + "］"
		}
		text.Draw(screen, label, mplusNormalFont, tabX, 88, c)
		tabX += text.BoundString(mplusNormalFont, label).Dx() + 14
	}
	text.Draw(screen, codexRateLabel(codexIDs(g.codexTab), tables[g.codexTab]), mplusSmallFont, max(tabX, 230), 87, color.White)

	ids := g.codexEntries()
	if len(ids) > 0 {
		g.codexCursor = min(max(g.codexCursor, 0), len(ids)-1)
	}
	g.codexScroll = codexScrollFor(g.codexCursor, g.codexScroll, codexListRows, len(ids))

	// 左：項目の一覧
	listH := codexListRows * codexRowHeight
	codexFrame(screen, codexListX, codexListY, codexListW, listH)
	for row := 0; row < codexListRows && g.codexScroll+row < len(ids); row++ {
		i := g.codexScroll + row
		id := ids[i]
		r := tables[g.codexTab][id]
		y := codexListY + row*codexRowHeight
		c := color.Color(color.White)
		if !r.Seen {
			c = dim
		}
		if i == g.codexCursor {
			codexFillRect(screen, codexListX, y, codexListW, codexRowHeight, color.NRGBA{70, 80, 160, 255})
			c = selected
		}
		name := "？？？"
		if r.Seen {
			name = g.codexEntryFor(g.codexTab, id, r).name
		}
		text.Draw(screen, codexListLabel(id, name), mplusSmallFont, codexListX+6, y+14, c)
	}
	// スクロールバー
	if len(ids) > codexListRows {
		barH := max(12, listH*codexListRows/len(ids))
		barY := codexListY + (listH-barH)*g.codexScroll/(len(ids)-codexListRows)
		codexFillRect(screen, codexListX+codexListW-4, barY, 3, barH, dim)
	}
	text.Draw(screen, fmt.Sprintf("%d / %d", min(g.codexCursor+1, len(ids)), len(ids)), mplusSmallFont, codexListX, codexListY+listH+18, color.White)

	// 右：選んだ項目の詳細
	codexFrame(screen, codexDetailX, codexListY, codexDetailW, listH)
	if len(ids) > 0 {
		id := ids[g.codexCursor]
		e := g.codexEntryFor(g.codexTab, id, tables[g.codexTab][id])
		x, y := codexDetailX+12, codexListY+12
		codexFillRect(screen, x, y, 64, 64, color.NRGBA{40, 45, 70, 255})
		if e.img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(2, 2)
			op.GeoM.Translate(float64(x+2), float64(y+2))
			screen.DrawImage(e.img, op)
		}
		text.Draw(screen, fmt.Sprintf("No.%03d", id), mplusSmallFont, x+76, y+14, dim)
		text.Draw(screen, e.name, mplusNormalFont, x+76, y+34, color.White)
		for j, line := range e.info {
			text.Draw(screen, line, mplusSmallFont, x+76, y+52+j*16, color.White)
		}
		codexFillRect(screen, x, y+78, codexDetailW-24, 1, dim)
		for j, line := range codexWrap(e.desc, codexDescWidth, codexDescLines) {
			text.Draw(screen, line, mplusSmallFont, x, y+100+j*17, color.White)
		}
	}
	if g.codexError != "" {
		text.Draw(screen, g.codexError, mplusSmallFont, codexDetailX, codexListY+listH+18, color.NRGBA{255, 120, 120, 255})
	}
	text.Draw(screen, g.keyGuide("←→：タブ  ↑↓：選択  X：メニュー  C：閉じる"), mplusNormalFont, 30, 462, color.White)
}

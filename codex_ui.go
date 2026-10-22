//go:build !test

package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"image/color"
	"strings"
)

const codexPageSize = 1

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
	text.Draw(screen, fmt.Sprintf("全体：発見 %d / 全 %d（%.1f%%）", found, total, percent), mplusSmallFont, 30, 65, color.White)
	labels := []string{"道具（種類別）", "敵（系統別）", "罠"}
	text.Draw(screen, labels[g.codexTab]+"  "+codexRateLabel(codexIDs(g.codexTab), tables[g.codexTab]), mplusNormalFont, 30, 93, color.White)
	ids := g.codexEntries()
	pages := max(1, (len(ids)+codexPageSize-1)/codexPageSize)
	g.codexPage = min(g.codexPage, pages-1)
	for i := g.codexPage * codexPageSize; i < min(len(ids), (g.codexPage+1)*codexPageSize); i++ {
		id := ids[i]
		r := tables[g.codexTab][id]
		y := 124 + (i%codexPageSize)*94
		name := "？？？"
		desc := ""
		info := ""
		var img *ebiten.Image
		if r.Seen {
			switch g.codexTab {
			case 0:
				t := itemTemplates[id]
				if r.Identified {
					name = t.Name
					desc = t.Description
				} else {
					name = "未識別の道具"
					desc = "識別すると名前と説明が表示されます"
				}
				info = map[string]string{"Money": "小銭", "Food": "食料", "Potion": "薬", "Weapon": "武器", "Armor": "盾", "Arrow": "矢", "Accessory": "指輪", "Cane": "杖", "Card": "カード", "Pot": "壺", "Trap": "罠カード"}[t.ItemType]
				img = g.getItemImage(rollItemFromTemplate(id, 0, 0, func(int) int { return 0 }))
			case 1:
				d := MonsterDefinitions[id]
				name = d.Name
				desc = codexEnemyDescription(id)
				info = fmt.Sprintf("HP %d 攻 %d 防 %d 経験値 %d  撃破 %d / 敗北 %d", d.MaxHealth, d.AttackPower, d.DefensePower, d.ExperiencePoints, r.Defeated, r.Deaths)
				img = g.getEnemyImage(Enemy{ID: id, Type: d.Type, Name: d.Name})
			case 2:
				name = mapTrapTemplates[id].Name
				desc = codexTrapDescription(id)
				info = "未踏"
				if r.Stepped {
					info = "踏んだことがある"
				}
				img = g.codexTrapImage(name)
			}
		}
		if img != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(30, float64(y))
			screen.DrawImage(img, op)
		}
		text.Draw(screen, fmt.Sprintf("%03d  %s", id, name), mplusNormalFont, 72, y+14, color.White)
		text.Draw(screen, info, mplusSmallFont, 72, y+34, color.White)
		for j, line := range codexWrap(desc, 41, 12) {
			text.Draw(screen, line, mplusSmallFont, 35, y+55+j*17, color.White)
		}
	}
	text.Draw(screen, fmt.Sprintf("%d / %d ページ", g.codexPage+1, pages), mplusSmallFont, 30, 418, color.White)
	if g.codexError != "" {
		text.Draw(screen, g.codexError, mplusSmallFont, 30, 398, color.White)
	}
	text.Draw(screen, g.keyGuide("↑↓：タブ  ←→：ページ  X：メニュー  C：閉じる"), mplusNormalFont, 30, 448, color.White)
}

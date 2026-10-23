//go:build !test

package main

import (
	"encoding/json"
	"log"
	"sort"
)

func (g *Game) ensureCodex() {
	if g.codexLoaded {
		return
	}
	g.codexLoaded = true
	g.codex = newCodexData()
	if !g.persistent {
		return
	}
	data, err := storageRead(codexFileName)
	if storageIsNotExist(err) {
		return
	}
	if err == nil {
		g.codex, err = decodeCodex(data)
	}
	if err != nil {
		g.codexError = "図鑑を読み込めません。元の記録を保護しています"
		log.Printf("図鑑: %v", err)
	}
}
func (g *Game) recordCodex(kind, id int, change func(*CodexRecord)) {
	if !g.persistent {
		return
	}
	g.ensureCodex()
	table := []map[int]CodexRecord{g.codex.Items, g.codex.Enemies, g.codex.Traps}[kind]
	r := table[id]
	old := r
	change(&r)
	if r != old {
		table[id] = r
		g.codexDirty = true
	}
}
func (g *Game) observeCodexItem(item Item) {
	if item == nil {
		return
	}
	g.recordCodex(0, item.GetID(), func(r *CodexRecord) {
		r.Seen = true
		identified := true
		if v, ok := item.(Identifiable); ok {
			identified = v.IsIdentified()
		}
		r.Identified = r.Identified || identified
	})
}
func (g *Game) observeCodexInventory() {
	if !g.persistent {
		return
	}
	g.ensureCodex()
	// 未表示の壺の中身などを発見済みにしない。既に見た道具の識別の変化だけを追う。
	var visit func(Item)
	visit = func(item Item) {
		if item == nil {
			return
		}
		if g.codex.Items[item.GetID()].Seen {
			g.observeCodexItem(item)
		}
		if p, ok := item.(*Pot); ok {
			for _, child := range p.Contents {
				visit(child)
			}
		}
	}
	for _, trap := range g.state.MapTraps {
		if trap.Discovered {
			g.recordCodex(2, trap.TemplateID, func(r *CodexRecord) { r.Seen = true })
		}
	}
	for _, item := range g.state.Player.Inventory {
		visit(item)
	}
}
func (g *Game) saveCodex() {
	if !g.persistent {
		return
	}
	g.observeCodexInventory()
	g.ensureCodex()
	if !g.codexDirty || g.codexError != "" {
		return
	}
	data, err := json.Marshal(g.codex)
	if err == nil {
		err = storageWrite(codexFileName, data)
	}
	if err != nil {
		log.Printf("図鑑の保存: %v", err)
		return
	}
	g.codexDirty = false
}
func codexIDs(kind int) []int {
	switch kind {
	case 0:
		return sortedTemplateIDs(itemTemplates, nil)
	case 1:
		return sortedTemplateIDs(MonsterDefinitions, nil)
	default:
		return sortedTemplateIDs(mapTrapTemplates, nil)
	}
}

// 系統の最小 ID 順にまとめ、系統内は上位種化の順に並べる。
func codexEnemyOrder() []int {
	ids := codexIDs(1)
	roots := map[int]int{}
	stages := map[int]int{}
	for _, id := range ids {
		cur := id
		seen := map[int]bool{}
		for !seen[cur] {
			seen[cur] = true
			parent := -1
			for a, b := range MonsterLevelUpTable {
				if b == cur {
					parent = a
					break
				}
			}
			if parent < 0 {
				break
			}
			cur = parent
			stages[id]++
		}
		roots[id] = cur
	}
	sort.SliceStable(ids, func(i, j int) bool {
		a, b := ids[i], ids[j]
		if roots[a] != roots[b] {
			return roots[a] < roots[b]
		}
		return stages[a] < stages[b]
	})
	return ids
}
func (g *Game) openCodex() {
	g.ensureCodex()
	g.showMenu = false
	g.showCodex = true
	g.codexTab = 0
	g.codexCursor = 0
	g.codexScroll = 0
}

// handleCodexInput は図鑑の操作。←→でタブ、↑↓（長押しで連続）で一覧のカーソルを動かす。
func (g *Game) handleCodexInput() {
	if g.keys().JustPressed(KeyC) {
		g.closeAllMenus()
		return
	}
	if g.keys().JustPressed(KeyX) {
		g.showCodex = false
		g.showMenu = true
		return
	}
	if g.keys().JustPressed(KeyLeft) {
		g.setCodexTab((g.codexTab + 2) % 3)
	}
	if g.keys().JustPressed(KeyRight) {
		g.setCodexTab((g.codexTab + 1) % 3)
	}
	n := len(g.codexEntries())
	if g.logScrollKeyPressed(KeyUp) {
		g.codexCursor = codexMoveCursor(g.codexCursor, -1, n, g.keys().JustPressed(KeyUp))
	}
	if g.logScrollKeyPressed(KeyDown) {
		g.codexCursor = codexMoveCursor(g.codexCursor, 1, n, g.keys().JustPressed(KeyDown))
	}
	g.codexScroll = codexScrollFor(g.codexCursor, g.codexScroll, codexListRows, n)
}
func (g *Game) setCodexTab(tab int) {
	g.codexTab = tab
	g.codexCursor = 0
	g.codexScroll = 0
}
func (g *Game) codexEntries() []int {
	ids := codexIDs(g.codexTab)
	if g.codexTab == 1 {
		return codexEnemyOrder()
	}
	if g.codexTab == 0 {
		sort.SliceStable(ids, func(i, j int) bool {
			a, b := itemTemplates[ids[i]], itemTemplates[ids[j]]
			if a.ItemType != b.ItemType {
				return a.ItemType < b.ItemType
			}
			return ids[i] < ids[j]
		})
	}
	return ids
}

// observeCodexEnemy は表示された姿だけを記録し、擬態の正体を漏らさない。
func (g *Game) observeCodexEnemy(enemy Enemy) {
	if isEnemyDisguised(enemy) || g.state.Player.StatusAilments.Blind > 0 {
		return
	}
	g.recordCodex(1, enemyAppearance(enemy).ID, func(r *CodexRecord) { r.Seen = true })
}
func codexOnScreen(x, y, offsetX, offsetY, width, height int) bool {
	left, top := x*tileSize+offsetX, y*tileSize+offsetY
	return left < width && top < height && left+tileSize > 0 && top+tileSize > 0
}

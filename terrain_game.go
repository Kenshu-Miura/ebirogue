//go:build !test

package main

func init() {
	expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "特殊地形", Lines: []string{
		"水路には入れない。飛ぶ道具は通り、", "水に落ちた道具は中身ごと消える。", "割れた壁は攻撃・つるはし・爆発で壊せる。",
		"隠し通路は正面を調べると半分の確率で発見。", "あかりのカード・透視の指輪でも見つかる。",
	}})
}

func (g *Game) revealHiddenAt(x, y int) bool {
	if !g.inMapBounds(x, y) || g.state.Map[y][x].Type != "hidden_corridor" {
		return false
	}
	// 連続した隠し部分をまとめて発見する。
	queue := []Coordinate{{X: x, Y: y}}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		if !g.inMapBounds(p.X, p.Y) || g.state.Map[p.Y][p.X].Type != "hidden_corridor" {
			continue
		}
		t := &g.state.Map[p.Y][p.X]
		t.Type = "corridor"
		t.Blocked = false
		t.BlockSight = false
		t.Visited = true
		for _, d := range []Coordinate{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			queue = append(queue, Coordinate{X: p.X + d.X, Y: p.Y + d.Y})
		}
	}
	g.miniMapDirty = true
	return true
}

func (g *Game) revealHiddenTerrain() {
	for y := range g.state.Map {
		for x := range g.state.Map[y] {
			g.revealHiddenAt(x, y)
		}
	}
}

func (g *Game) searchHiddenTerrain(dx, dy int) {
	x, y := g.state.Player.X+dx, g.state.Player.Y+dy
	if !g.inMapBounds(x, y) || g.state.Map[y][x].Type != "hidden_corridor" {
		return
	}
	g.Enqueue(Action{Execute: func(g *Game) {
		if g.randIntn(2) == 0 && g.revealHiddenAt(x, y) {
			g.EnqueueMessage("隠し通路を見つけた！", 0.5)
		}
	}})
}

func (g *Game) breakCrackedAround(x, y, radius int) {
	for ty := y - radius; ty <= y+radius; ty++ {
		for tx := x - radius; tx <= x+radius; tx++ {
			if g.inMapBounds(tx, ty) && g.state.Map[ty][tx].Type == "cracked_wall" {
				g.digWallAt(tx, ty)
			}
		}
	}
}

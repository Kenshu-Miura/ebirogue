//go:build !test

package main

import "fmt"

func init() {
	// ワナのカード→罠→食料生成→アイテムテーブルの初期化循環を避ける。
	// ID 自体はテーブルに定義済みなので、抽選一覧には他の罠と同様に含まれる。
	template := mapTrapTemplates[13]
	template.Effect = spoilTrapEffect
	mapTrapTemplates[13] = template
}

// itemTrapDropPositions は周囲8マスの空き床（部屋の床か通路）を返す。
// 壁・階段・扉・既存のアイテム・罠・敵のいるマスは除く。
func (g *Game) itemTrapDropPositions() []Coordinate {
	x, y := g.state.Player.GetPosition()
	positions := []Coordinate{}
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if ny < 0 || ny >= len(g.state.Map) || nx < 0 || nx >= len(g.state.Map[ny]) {
				continue
			}
			tile := g.state.Map[ny][nx]
			if (tile.Type != "floor" && tile.Type != "corridor") || tile.Blocked || isOccupied(g, nx, ny) {
				continue
			}
			occupied := false
			for _, item := range g.state.Items {
				ix, iy := item.GetPosition()
				occupied = occupied || (ix == nx && iy == ny)
			}
			for _, trap := range g.state.MapTraps {
				occupied = occupied || (trap.X == nx && trap.Y == ny)
			}
			if !occupied {
				positions = append(positions, Coordinate{X: nx, Y: ny})
			}
		}
	}
	return positions
}

func (g *Game) dropItemFromTrap(item Item, position Coordinate) {
	// 選択位置ではなく同じオブジェクトを所持品から取り除く。
	g.removeThrownItemFromSource(item)
	item.SetPosition(position.X, position.Y)
	if card, ok := item.(*Card); ok && isSanctuaryItem(card) {
		card.Stuck = true
		g.EnqueueMessage(fmt.Sprintf("%sは床に貼りついた", card.Name), 0.4)
	}
	g.state.Items = append(g.state.Items, item)
	g.selectedItemIndex = 0
	g.miniMapDirty = true
}

var unequipTrapEffect = func(g *Game) {
	g.Enqueue(Action{Duration: 0.5, Message: "装備をはずす風が吹いた", Execute: func(g *Game) {
		player := &g.state.Player
		targets := []Equipable{}
		if player.EquippedWeapon != nil {
			targets = append(targets, player.EquippedWeapon)
		}
		if player.EquippedArmor != nil {
			targets = append(targets, player.EquippedArmor)
		}
		indices := make([]int, len(targets))
		for i := range indices {
			indices[i] = i
		}
		selected := rollItemTrapIndices(indices, 1, g.randIntn)
		if len(selected) == 0 {
			g.EnqueueMessage("しかし外せる装備がなかった", 0.4)
			return
		}
		item := targets[selected[0]]
		message, err := player.UnequipItem(item)
		if err != nil {
			return
		}
		g.EnqueueMessage(message, 0.4)
	}})
}

var spoilTrapEffect = func(g *Game) {
	g.Enqueue(Action{Duration: 0.5, Message: "腐った霧が噴き出した", Execute: func(g *Game) {
		candidates := []int{}
		for i, item := range g.state.Player.Inventory {
			if food, ok := item.(*Food); ok && food.ID != rottenSausageItemID {
				candidates = append(candidates, i)
			}
		}
		selected := rollItemTrapIndices(candidates, 1, g.randIntn)
		if len(selected) == 0 {
			g.EnqueueMessage("しかし腐る食料がなかった", 0.4)
			return
		}
		i := selected[0]
		name := g.state.Player.Inventory[i].GetName()
		g.state.Player.Inventory[i] = g.createItemByID(rottenSausageItemID, 0, 0)
		g.EnqueueMessage(fmt.Sprintf("%sが腐ったウインナーになった", name), 0.4)
	}})
}

var scatterTrapEffect = func(g *Game) {
	g.Enqueue(Action{Duration: 0.5, Message: "所持品を散らす風が吹いた", Execute: func(g *Game) {
		positions := g.itemTrapDropPositions()
		candidates := []int{}
		for i, item := range g.state.Player.Inventory {
			if equipment, ok := item.(Equipable); ok && g.state.Player.IsEquipped(equipment) {
				continue
			}
			if isStuckSanctuaryItem(item) {
				continue
			}
			candidates = append(candidates, i)
		}
		selected := rollItemTrapIndices(candidates, min(3, len(positions)), g.randIntn)
		// 削除でインデックスがずれる前に対象を保存する。壺の中身はそのまま保持する。
		items := make([]Item, len(selected))
		for i, index := range selected {
			items[i] = g.state.Player.Inventory[index]
		}
		for _, item := range items {
			i := g.randIntn(len(positions))
			g.dropItemFromTrap(item, positions[i])
			positions = append(positions[:i], positions[i+1:]...)
		}
		if len(items) == 0 {
			g.EnqueueMessage("しかし何も散らばらなかった", 0.4)
		} else {
			g.EnqueueMessage(fmt.Sprintf("所持品が%d個散らばった", len(items)), 0.4)
		}
	}})
}

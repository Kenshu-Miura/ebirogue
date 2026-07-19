//go:build !test

package main

import "fmt"

// 壺（Pot）関連の処理。
// 保存の壺はアイテムを入れて持ち運べ、自由に出し入れできる。
// 投げると割れて中身がその場に散らばる。

// maxPotCapacity は壺拡大のカードなどで増やせる容量の上限
const maxPotCapacity = 8

// expandedPotCapacity は容量を1増やした値を返す。上限を超えては増えない。
func expandedPotCapacity(current int) int {
	if current >= maxPotCapacity {
		return current
	}
	return current + 1
}

// Pot用のItem/Identifiableインタフェース実装
func (p *Pot) Use(g *Game) {
	if action, exists := p.UseActions["PotEffect"]; exists {
		action(g)
	}
}

func (p *Pot) IsIdentified() bool {
	return p.Identified
}

func (p *Pot) SetIdentified(value bool) {
	p.Identified = value
}

func (p *Pot) GetIdentified() bool {
	return p.Identified
}

// GetDescription は基本説明に現在の中身の一覧を付けて返す。
func (p *Pot) GetDescription() string {
	if len(p.Contents) == 0 {
		return p.Description + "（中身は空）"
	}
	names := ""
	for i, item := range p.Contents {
		if i > 0 {
			names += "、"
		}
		name, _ := displayItemName(item)
		names += name
	}
	return p.Description + "（中身: " + names + "）"
}

// inventoryPots はインベントリ内の壺の一覧を返す。
func (g *Game) inventoryPots() []*Pot {
	pots := []*Pot{}
	for _, item := range g.state.Player.Inventory {
		if pot, ok := item.(*Pot); ok {
			pots = append(pots, pot)
		}
	}
	return pots
}

// startPotInsert は「入れる」を選んだときに、壺へ入れるアイテムの選択モードへ移る。
func (g *Game) startPotInsert(pot *Pot) {
	if len(pot.Contents) >= pot.Capacity {
		g.EnqueueMessage(fmt.Sprintf("%sはもういっぱいだ", pot.GetName()), 0.4)
		g.showItemActions = false
		g.showInventory = false
		g.selectedItemIndex = 0
		g.selectedActionIndex = 0
		return
	}
	if len(g.state.Player.Inventory) <= 1 {
		g.EnqueueMessage("壺に入れるものを持っていない", 0.4)
		g.showItemActions = false
		g.showInventory = false
		g.selectedItemIndex = 0
		g.selectedActionIndex = 0
		return
	}
	g.usePotInsert = true
	g.potInsertIndex = g.selectedItemIndex
	g.tmpSelectedItemIndex = g.selectedItemIndex
	g.showItemActions = false
	g.selectedActionIndex = 0
	g.showInventory = true
}

// executePotInsertSelection は選択したアイテムを壺へ入れる。
func (g *Game) executePotInsertSelection() {
	potIndex := g.potInsertIndex
	itemIndex := g.selectedItemIndex

	closeAll := func() {
		g.usePotInsert = false
		g.potInsertIndex = 0
		g.tmpSelectedItemIndex = -1
		g.selectedItemIndex = 0
		g.selectedActionIndex = 0
		g.showItemActions = false
		g.showInventory = false
	}

	if potIndex >= len(g.state.Player.Inventory) || itemIndex >= len(g.state.Player.Inventory) {
		closeAll()
		return
	}
	pot, ok := g.state.Player.Inventory[potIndex].(*Pot)
	if !ok {
		closeAll()
		return
	}
	item := g.state.Player.Inventory[itemIndex]
	itemName, identified := displayItemName(item)

	if _, isPot := item.(*Pot); isPot {
		g.EnqueueMessage("壺に壺は入らない", 0.4)
		closeAll()
		return
	}
	if equipableItem, ok := item.(Equipable); ok && g.state.Player.IsEquipped(equipableItem) {
		g.EnqueueMessage("装備中のものは壺に入れられない", 0.4)
		closeAll()
		return
	}
	if len(pot.Contents) >= pot.Capacity {
		g.EnqueueMessage(fmt.Sprintf("%sはもういっぱいだ", pot.GetName()), 0.4)
		closeAll()
		return
	}

	pot.Contents = append(pot.Contents, item)
	g.state.Player.Inventory = append(g.state.Player.Inventory[:itemIndex], g.state.Player.Inventory[itemIndex+1:]...)
	g.Enqueue(Action{
		Duration:     0.4,
		Message:      fmt.Sprintf("%sを%sに入れた", itemName, pot.GetName()),
		ItemName:     itemName,
		Execute:      func(g *Game) {},
		IsIdentified: identified,
	})
	g.isActioned = true
	closeAll()
}

// executePotTakeOut は壺の中身を持ち物へ取り出す。
// 持ち物がいっぱいになったら残りは壺の中へ残る。
func (g *Game) executePotTakeOut(pot *Pot) {
	if len(pot.Contents) == 0 {
		g.EnqueueMessage(fmt.Sprintf("%sは空っぽだ", pot.GetName()), 0.4)
	} else {
		took := false
		for len(pot.Contents) > 0 && len(g.state.Player.Inventory) < g.state.Player.MaxInventory {
			item := pot.Contents[0]
			pot.Contents = pot.Contents[1:]
			g.state.Player.Inventory = append(g.state.Player.Inventory, item)
			itemName, identified := displayItemName(item)
			g.Enqueue(Action{
				Duration:     0.4,
				Message:      fmt.Sprintf("%sを壺から取り出した", itemName),
				ItemName:     itemName,
				Execute:      func(g *Game) {},
				IsIdentified: identified,
			})
			took = true
		}
		if len(pot.Contents) > 0 {
			g.EnqueueMessage("持ち物がいっぱいで全部は取り出せなかった", 0.4)
		}
		if took {
			g.isActioned = true
		}
	}
	g.showItemActions = false
	g.showInventory = false
	g.selectedItemIndex = 0
	g.selectedActionIndex = 0
}

// scatterPotContents は割れた壺の中身を着地点の周囲の床へ散らばらせる。
// 置ける場所が無かった中身は消える。
func (g *Game) scatterPotContents(pot *Pot, x, y int) {
	positions := []Coordinate{{0, 0}, {-1, 0}, {1, 0}, {0, -1}, {0, 1}, {-1, -1}, {-1, 1}, {1, -1}, {1, 1}}
	for _, item := range pot.Contents {
		for _, dir := range positions {
			newX, newY := x+dir.X, y+dir.Y
			if newX < 0 || newY < 0 || newY >= len(g.state.Map) || newX >= len(g.state.Map[0]) {
				continue
			}
			if g.state.Map[newY][newX].Type == "wall" {
				continue
			}
			occupied := false
			for _, floorItem := range g.state.Items {
				ix, iy := floorItem.GetPosition()
				if ix == newX && iy == newY {
					occupied = true
					break
				}
			}
			if occupied {
				continue
			}
			item.SetPosition(newX, newY)
			g.state.Items = append(g.state.Items, item)
			break
		}
	}
	pot.Contents = nil
	g.miniMapDirty = true
}

// suckOutAllPots は持ち物の壺すべてから中身を取り出して持ち物へ移す。
// 取り出した個数と、持ち物がいっぱいで取り出せない中身が残ったかどうかを返す。
func (g *Game) suckOutAllPots() (moved int, leftover bool) {
	for _, pot := range g.inventoryPots() {
		for len(pot.Contents) > 0 && len(g.state.Player.Inventory) < g.state.Player.MaxInventory {
			item := pot.Contents[0]
			pot.Contents = pot.Contents[1:]
			g.state.Player.Inventory = append(g.state.Player.Inventory, item)
			moved++
		}
		if len(pot.Contents) > 0 {
			leftover = true
		}
	}
	return moved, leftover
}

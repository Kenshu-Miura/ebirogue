//go:build !test

package main

import (
	"fmt"
	"sort"
)

func beginCardTarget(g *Game, effect func(*Game, int)) {
	_, inventory := determineItemSource(g)
	g.tmpSelectedItemIndex = g.selectedItemIndex
	if !inventory {
		g.tmpSelectedItemIndex = g.selectedGroundItemIndex
	}
	g.itemTargetEffect = effect
	g.inventoryFilter = CategoryAll
	g.selectedGroundActionIndex = 0
	g.selectedItemIndex = 0
	g.showInventory = true
	g.ShowGroundItem = false
}

func sausageCard(g *Game) { beginCardTarget(g, transformCardTarget) }
func curseCard(g *Game)   { beginCardTarget(g, curseCardTarget) }

// consumeTargetCard は選択確定時だけカードを消費し、削除後の対象の添字を返す。
func consumeTargetCard(g *Game, target int) int {
	cardIndex := g.tmpSelectedItemIndex
	if g.GroundItemActioned {
		g.state.Items = append(g.state.Items[:cardIndex], g.state.Items[cardIndex+1:]...)
	} else {
		g.state.Player.Inventory = append(g.state.Player.Inventory[:cardIndex], g.state.Player.Inventory[cardIndex+1:]...)
		if target > cardIndex {
			target--
		}
	}
	g.itemTargetEffect = nil
	g.tmpSelectedItemIndex = -1
	g.selectedItemIndex = 0
	g.showInventory = false
	g.GroundItemActioned = false
	g.isActioned = true
	return target
}

func transformCardTarget(g *Game, target int) {
	item := g.state.Player.Inventory[target]
	if !g.GroundItemActioned && target == g.tmpSelectedItemIndex {
		g.EnqueueMessage("装備中の道具は変えられない", 0.4)
		return
	}
	if equipment, ok := item.(Equipable); ok && g.state.Player.IsEquipped(equipment) {
		g.EnqueueMessage("装備中の道具は変えられない", 0.4)
		return
	}
	target = consumeTargetCard(g, target)
	g.Enqueue(Action{Duration: 0.4, Message: "道具がウインナーに変わった。", Execute: func(g *Game) {
		g.state.Player.Inventory[target] = newTransformedSausage()
	}})
}

func curseCardTarget(g *Game, target int) {
	if !g.GroundItemActioned && target == g.tmpSelectedItemIndex {
		return
	}
	item := g.state.Player.Inventory[target]
	consumeTargetCard(g, target)
	g.Enqueue(Action{Duration: 0.4, Message: "呪いのカードを使用した。", Execute: func(g *Game) {
		if message, immune := g.playerImmunityMessage(StatusCurse); immune {
			g.EnqueueMessage(message, 0.4)
		} else if isCurseableItem(item) && curseItem(&g.state.Player, item) {
			g.EnqueueMessage("道具が呪われた。", 0.4)
		} else {
			g.EnqueueMessage("何も起こらなかった", 0.4)
		}
	}})
}

func useWave3Card(g *Game, effect func(*Game)) {
	item, inventory := determineItemSource(g)
	g.Enqueue(Action{Duration: 0.4, Message: fmt.Sprintf("%sを使用した。", item.GetName()), Execute: func(g *Game) { effect(g); g.isActioned = true }})
	removeUsedItem(g, inventory)
}

func eraseTrapsCard(g *Game) {
	useWave3Card(g, func(g *Game) {
		n := len(g.state.MapTraps)
		g.state.MapTraps = nil
		g.miniMapDirty = true
		if n == 0 {
			g.EnqueueMessage("何も起こらなかった", 0.4)
		} else {
			g.EnqueueMessage(fmt.Sprintf("罠が%d個消えた", n), 0.4)
		}
	})
}

// sortedGatherIndices は動かす対象を元の位置が近い順にする（同位置なら元の順を保つ）。
func sortedGatherIndices(positions []Coordinate, center Coordinate) []int {
	indices := make([]int, len(positions))
	for i := range indices {
		indices[i] = i
	}
	sort.SliceStable(indices, func(i, j int) bool {
		a, b := positions[indices[i]], positions[indices[j]]
		if da, db := gatherDistance(a, center), gatherDistance(b, center); da != db {
			return da < db
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.X < b.X
	})
	return indices
}

func gatherEnemiesCard(g *Game) {
	useWave3Card(g, func(g *Game) {
		center := Coordinate{X: g.state.Player.X, Y: g.state.Player.Y}
		positions := make([]Coordinate, len(g.state.Enemies))
		for i, e := range g.state.Enemies {
			positions[i] = Coordinate{X: e.X, Y: e.Y}
		}
		destinations := gatherFloorPositions(g.state.Map, center, positions)
		n := 0
		for _, i := range sortedGatherIndices(positions, center) {
			e := &g.state.Enemies[i]
			if MonsterDefinitions[e.ID].IsBoss {
				continue
			}
			if n >= len(destinations) {
				break
			}
			e.X, e.Y = destinations[n].X, destinations[n].Y
			e.Animating, e.dx, e.dy = false, 0, 0
			n++
		}
		g.miniMapDirty = true
		g.EnqueueMessage(fmt.Sprintf("敵が%d体集まった。", n), 0.4)
	})
}

func gatherItemsCard(g *Game) {
	useWave3Card(g, func(g *Game) {
		center := Coordinate{X: g.state.Player.X, Y: g.state.Player.Y}
		positions := make([]Coordinate, len(g.state.Items))
		for i, item := range g.state.Items {
			x, y := item.GetPosition()
			positions[i] = Coordinate{X: x, Y: y}
		}
		// 座標の並びは純粋関数、落とせる床の判定は既存の共通処理を使う。
		destinations := []Coordinate{}
		for _, pos := range gatherFloorPositions(g.state.Map, center, nil) {
			if g.itemDropFree(pos.X, pos.Y) {
				destinations = append(destinations, pos)
			}
		}
		n := 0
		for _, i := range sortedGatherIndices(positions, center) {
			item := g.state.Items[i]
			if positions[i] == center || isStuckSanctuaryItem(item) {
				continue
			}
			if n >= len(destinations) {
				break
			}
			item.SetPosition(destinations[n].X, destinations[n].Y)
			n++
		}
		g.miniMapDirty = true
		g.EnqueueMessage(fmt.Sprintf("道具が%d個集まった。", n), 0.4)
		g.stealGatheredShopGoods()
	})
}

// stealGatheredShopGoods は道具寄せで店の外へ出た商品の値段を消し、商売中の店なら泥棒にする（開発者の決定）。
func (g *Game) stealGatheredShopGoods() {
	stolen := false
	for _, item := range g.state.Items {
		x, y := item.GetPosition()
		if g.inShopInterior(x, y) {
			continue
		}
		forEachItemIn([]Item{item}, func(inner Item) {
			if itemShopPrice(inner) > 0 {
				setItemShopPrice(inner, 0)
				stolen = true
			}
		})
	}
	if stolen {
		g.becomeShopThief(false)
	}
}

func gambleCard(g *Game) {
	useWave3Card(g, func(g *Game) {
		cash, won := rollGamble(g.state.Player.Cash, g.randIntn)
		g.state.Player.Cash = cash
		if won {
			g.EnqueueMessage(fmt.Sprintf("バクチに勝った！所持金が%dになった。", cash), 0.4)
		} else {
			g.EnqueueMessage("バクチに負けて所持金が0になった。", 0.4)
		}
	})
}

func playerHasCursedBelongings(p *Player) bool {
	items := append([]Item{}, p.Inventory...)
	items = append(items, p.EquippedWeapon, p.EquippedArmor, p.EquippedArrow, p.EquippedAccessories[0], p.EquippedAccessories[1])
	for _, item := range items {
		switch v := item.(type) {
		case *Weapon:
			if v != nil && v.Cursed {
				return true
			}
		case *Armor:
			if v != nil && v.Cursed {
				return true
			}
		case *Arrow:
			if v != nil && v.Cursed {
				return true
			}
		case *Accessory:
			if v != nil && v.Cursed {
				return true
			}
		}
	}
	return false
}

func troubleCard(g *Game) {
	useWave3Card(g, func(g *Game) {
		p := &g.state.Player
		p.Health = p.MaxHealth
		message := "HPが全回復した。"
		switch troubleToFix(p.StatusAilments, p.Satiety, playerHasCursedBelongings(p), p.Power, p.MaxPower) {
		case troubleStatus:
			g.clearPlayerBadStatuses()
			message += "悪い状態異常がすべて解けた。"
		case troubleHunger:
			p.Satiety = min(50, p.MaxSatiety)
			g.capSatietyForSatori() // 悟りの盾を装備中は満腹度1まで
			message += "満腹度が回復した。"
		case troubleCurse:
			uncursePlayerBelongings(p)
			message += "持ち物の呪いがすべて解けた。"
		case troublePower:
			p.Power = p.MaxPower
			message += "パワーが回復した。"
		}
		g.EnqueueMessage(message, 0.4)
	})
}

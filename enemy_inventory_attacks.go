//go:build !test

package main

import (
	"fmt"
	"math/rand"
	"sort"
)

// chooseInventoryIndex は条件を満たす所持品から1つを選ぶ。
// 乱数関数を注入できるため、特殊攻撃の境界値をテストできる。
func chooseInventoryIndex(player *Player, predicate func(Item) bool, intn func(int) int) (int, bool) {
	candidates := make([]int, 0, len(player.Inventory))
	for i, item := range player.Inventory {
		if predicate(item) {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		return 0, false
	}
	return candidates[intn(len(candidates))], true
}

func isEquippedByPlayer(player *Player, item Item) bool {
	equipable, ok := item.(Equipable)
	return ok && player.IsEquipped(equipable)
}

func inventoryItemIndex(player *Player, item Item) (int, bool) {
	for i, candidate := range player.Inventory {
		if candidate == item {
			return i, true
		}
	}
	return 0, false
}

func enqueueStealAttack(enemy *Enemy, g *Game, intn func(int) int) {
	index, ok := chooseInventoryIndex(&g.state.Player, func(item Item) bool {
		return !isEquippedByPlayer(&g.state.Player, item)
	}, intn)
	name := g.enemyDisplayName(enemy.Name)
	if !ok {
		g.EnqueueMessage(fmt.Sprintf("%sは盗もうとしたが、盗める物がなかった。", name), 0.5)
		return
	}

	item := g.state.Player.Inventory[index]
	itemName, identified := displayItemName(item)
	g.Enqueue(Action{
		Duration:     0.5,
		Message:      fmt.Sprintf("%sは%sを盗んだ。", name, itemName),
		ItemName:     itemName,
		IsIdentified: identified,
		Execute: func(g *Game) {
			currentIndex, exists := inventoryItemIndex(&g.state.Player, item)
			if !exists {
				return
			}
			g.state.Player.Inventory = append(g.state.Player.Inventory[:currentIndex], g.state.Player.Inventory[currentIndex+1:]...)
			enemy.HeldItem = item
			enemy.Fleeing = true
		},
	})
}

func enqueueFoodTransformationAttack(enemy *Enemy, g *Game, intn func(int) int) {
	index, ok := chooseInventoryIndex(&g.state.Player, func(item Item) bool {
		if isEquippedByPlayer(&g.state.Player, item) {
			return false
		}
		_, alreadyFood := item.(*Food)
		return !alreadyFood
	}, intn)
	name := g.enemyDisplayName(enemy.Name)
	if !ok {
		g.EnqueueMessage(fmt.Sprintf("%sは握ろうとしたが、何も変わらなかった。", name), 0.5)
		return
	}

	item := g.state.Player.Inventory[index]
	itemName, identified := displayItemName(item)
	g.Enqueue(Action{
		Duration:     0.5,
		Message:      fmt.Sprintf("%sは%sをウインナーに変えた。", name, itemName),
		ItemName:     itemName,
		IsIdentified: identified,
		Execute: func(g *Game) {
			currentIndex, exists := inventoryItemIndex(&g.state.Player, item)
			if !exists {
				return
			}
			g.state.Player.Inventory[currentIndex] = newTransformedSausage()
		},
	})
}

// newTransformedSausage は敵の能力用に基本のウインナーを生成する。
// itemTemplatesを経由しないことで、敵定義とアイテム定義の初期化循環を避ける。
func newTransformedSausage() Item {
	return &Food{
		BaseItem: BaseItem{
			Entity:      Entity{Char: '!'},
			ID:          1,
			Type:        "Sausage",
			Name:        "ウインナー",
			Description: "海老さんが配信中に食べる食事。満腹度を50回復する。",
			UseActions:  map[string]UseAction{"RestoreSatiety": restoreSatiety},
		},
		Satiety:      50,
		MaxStatBonus: 1,
	}
}

func isCurseableItem(item Item) bool {
	switch v := item.(type) {
	case *Weapon:
		return !v.Cursed
	case *Armor:
		return !v.Cursed
	case *Arrow:
		return !v.Cursed
	case *Accessory:
		return !v.Cursed
	default:
		return false
	}
}

func curseItem(player *Player, item Item) bool {
	switch v := item.(type) {
	case *Weapon:
		v.Cursed = true
	case *Armor:
		v.Cursed = true
	case *Arrow:
		v.Cursed = true
	case *Accessory:
		equipped := player.IsEquipped(v)
		if equipped {
			v.UpdatePlayerStats(player, false)
		}
		v.Cursed = true
		if equipped {
			v.UpdatePlayerStats(player, true)
		}
	default:
		return false
	}
	return true
}

func enqueueCurseAttack(enemy *Enemy, g *Game, intn func(int) int) {
	index, ok := chooseInventoryIndex(&g.state.Player, isCurseableItem, intn)
	name := g.enemyDisplayName(enemy.Name)
	if !ok {
		g.EnqueueMessage(fmt.Sprintf("%sは呪いをかけたが、効果はなかった。", name), 0.5)
		return
	}

	item := g.state.Player.Inventory[index]
	itemName, identified := displayItemName(item)
	g.Enqueue(Action{
		Duration:     0.5,
		Message:      fmt.Sprintf("%sの呪いで%sは呪われた。", name, itemName),
		ItemName:     itemName,
		IsIdentified: identified,
		Execute: func(g *Game) {
			curseItem(&g.state.Player, item)
		},
	})
}

func forcedUseItemIndices(player *Player) []int {
	indices := make([]int, 0)
	for i, item := range player.Inventory {
		switch item.(type) {
		case *Food, *Potion:
			indices = append(indices, i)
		}
	}
	return indices
}

func forcedMoveDestinations(g *Game) []Coordinate {
	player := &g.state.Player
	candidates := make([]Coordinate, 0, 8)
	for _, direction := range getDirections() {
		x, y := player.X+direction.dx, player.Y+direction.dy
		if x < 0 || y < 0 || y >= len(g.state.Map) || x >= len(g.state.Map[y]) || g.state.Map[y][x].Blocked {
			continue
		}
		if direction.dx != 0 && direction.dy != 0 {
			if g.state.Map[player.Y][player.X+direction.dx].Blocked || g.state.Map[player.Y+direction.dy][player.X].Blocked {
				continue
			}
		}
		occupied := false
		for _, enemy := range g.state.Enemies {
			if enemy.X == x && enemy.Y == y {
				occupied = true
				break
			}
		}
		if !occupied {
			candidates = append(candidates, Coordinate{X: x, Y: y})
		}
	}
	return candidates
}

func enqueueForcedItemUse(enemy *Enemy, g *Game, indices []int, intn func(int) int) {
	item := g.state.Player.Inventory[indices[intn(len(indices))]]
	itemName, identified := displayItemName(item)
	g.Enqueue(Action{
		Duration:     0.5,
		Message:      fmt.Sprintf("%sに操られ、海老さんは%sを使った。", g.enemyDisplayName(enemy.Name), itemName),
		ItemName:     itemName,
		IsIdentified: identified,
		Execute: func(g *Game) {
			index, exists := inventoryItemIndex(&g.state.Player, item)
			if !exists {
				return
			}
			g.GroundItemActioned = false
			g.selectedItemIndex = index
			item.Use(g)
			g.selectedItemIndex = 0
		},
	})
}

func enqueueForcedMove(enemy *Enemy, g *Game, destinations []Coordinate, intn func(int) int) {
	destination := destinations[intn(len(destinations))]
	dx, dy := destination.X-g.state.Player.X, destination.Y-g.state.Player.Y
	g.Enqueue(Action{
		Duration: 0.5,
		Message:  fmt.Sprintf("%sに操られ、海老さんは勝手に移動した。", g.enemyDisplayName(enemy.Name)),
		Execute: func(g *Game) {
			g.state.Player.X = destination.X
			g.state.Player.Y = destination.Y
			g.state.Player.Direction = determineDirection(dx, dy)
			g.dx, g.dy = dx, dy
			g.Animating = true
		},
	})
}

func enqueueManipulationAttack(enemy *Enemy, g *Game, intn func(int) int) {
	useIndices := forcedUseItemIndices(&g.state.Player)
	moveDestinations := forcedMoveDestinations(g)
	if len(useIndices) == 0 && len(moveDestinations) == 0 {
		g.EnqueueMessage(fmt.Sprintf("%sは操ろうとしたが、何も起こらなかった。", g.enemyDisplayName(enemy.Name)), 0.5)
		return
	}

	forceUse := len(useIndices) > 0 && (len(moveDestinations) == 0 || intn(2) == 0)
	if forceUse {
		enqueueForcedItemUse(enemy, g, useIndices, intn)
		return
	}
	enqueueForcedMove(enemy, g, moveDestinations, intn)
}

// moveEnemyAwayFromPlayer は盗品を持つ敵を、移動可能な範囲でプレイヤーから遠ざける。
func (g *Game) moveEnemyAwayFromPlayer(enemyIndex int, intn func(int) int) bool {
	enemy := &g.state.Enemies[enemyIndex]
	type scoredDirection struct {
		dx, dy int
		score  int
		tie    int
	}
	directions := getDirections()
	candidates := make([]scoredDirection, 0, len(directions))
	for _, direction := range directions {
		x, y := enemy.X+direction.dx, enemy.Y+direction.dy
		distance := abs(x-g.state.Player.X) + abs(y-g.state.Player.Y)
		candidates = append(candidates, scoredDirection{dx: direction.dx, dy: direction.dy, score: distance, tie: intn(1 << 16)})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].tie < candidates[j].tie
		}
		return candidates[i].score > candidates[j].score
	})
	for _, candidate := range candidates {
		if moveEnemy(g, enemyIndex, candidate.dx, candidate.dy) {
			enemy.dx, enemy.dy = candidate.dx, candidate.dy
			enemy.Direction = determineDirection(candidate.dx, candidate.dy)
			enemy.Animating = true
			return true
		}
	}
	return false
}

func (g *Game) moveFleeingEnemy(enemyIndex int) bool {
	return g.moveEnemyAwayFromPlayer(enemyIndex, rand.Intn)
}

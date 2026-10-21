//go:build !test

package main

import "fmt"

const pitfallTrapID = 6

// availableTrapIDs はボス階と最終階で落とし穴を抽選から除く。
func availableTrapIDs(ids []int, floor int) []int {
	if !isBossFloor(floor) && floor < finalFloor {
		return ids
	}
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if id != pitfallTrapID {
			result = append(result, id)
		}
	}
	return result
}

var warpTrapEffect = func(g *Game) {
	g.Enqueue(Action{Duration: 0.5, Message: "海老さんはワープした", Execute: func(g *Game) {
		candidates := []Coordinate{}
		for y, row := range g.state.Map {
			for x, tile := range row {
				if tile.Type == "floor" && !tile.Blocked && !isOccupied(g, x, y) {
					candidates = append(candidates, Coordinate{X: x, Y: y})
				}
			}
		}
		if destination, ok := rollWarpDestination(candidates, g.randIntn); ok {
			g.state.Player.X, g.state.Player.Y = destination.X, destination.Y
			g.dx, g.dy = 0, 0
			g.Animating = false
			g.updateTileBrightness()
			g.MarkVisitedTiles(destination.X, destination.Y)
			g.MarkRoomVisited(destination.X, destination.Y)
			g.miniMapDirty = true
		} else {
			g.EnqueueMessage("しかし移動できる床がなかった", 0.4)
		}
	}})
}

var pitfallTrapEffect = func(g *Game) {
	source := g.steppedTrapSource
	g.Enqueue(Action{Duration: 0.5, Message: "落とし穴が開いた", Execute: func(g *Game) {
		if isBossFloor(g.Floor) || g.Floor >= finalFloor || g.cleared || g.playerDead {
			g.EnqueueMessage("しかし落とし穴は作動しなかった", 0.4)
			return
		}
		damage := rollPitfallDamage(g.randIntn)
		g.damagePlayer(damage, source)
		g.EnqueueMessage(fmt.Sprintf("落下して%dダメージを受けた", damage), 0.4)
		g.state.Player.checkDeath(g)
		if !g.playerDead {
			g.requestFloorDescent()
		}
	}})
}

var bearTrapEffect = func(g *Game) {
	// トラバサミは装備能力による無効化だけを見る（状態異常耐性の盾は効かない）。
	guard, immune := g.playerImmunitySource(StatusBearTrap)
	g.enqueuePlayerStatus(StatusBearTrap, 6, StackDefault, guard, immune) // 表の規則（長い方を残す）
}

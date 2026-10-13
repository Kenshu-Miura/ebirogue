//go:build !test

package main

import "fmt"

func availableTrapIDs(ids []int, floor int) []int {
	if !isBossFloor(floor) && floor < finalFloor {
		return ids
	}
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if id != 6 {
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
	g.Enqueue(Action{Duration: 0.5, Message: "落とし穴が開いた", Execute: func(g *Game) {
		if isBossFloor(g.Floor) || g.Floor >= finalFloor || g.cleared || g.playerDead {
			g.EnqueueMessage("しかし落とし穴は作動しなかった", 0.4)
			return
		}
		damage := rollPitfallDamage(g.randIntn)
		g.damagePlayer(damage, trapSource(6, "落とし穴"))
		g.EnqueueMessage(fmt.Sprintf("落下して%dダメージを受けた", damage), 0.4)
		g.state.Player.checkDeath(g)
		if !g.playerDead {
			g.beginFloorDescent()
		}
	}})
}

var bearTrapEffect = func(g *Game) {
	g.Enqueue(Action{Duration: 0.5, Message: "トラバサミに足を挟まれた", Execute: func(g *Game) {
		g.state.Player.StatusAilments.BearTrap = max(g.state.Player.StatusAilments.BearTrap, 6)
	}})
}

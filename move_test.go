package main

import "testing"

// MovePlayerStub is a minimal stub used for testing.
func (g *Game) MovePlayerStub(dx, dy int) bool {
	newX := g.state.Player.X + dx
	newY := g.state.Player.Y + dy

	// bounds check
	if newY < 0 || newY >= len(g.state.Map) || newX < 0 || newX >= len(g.state.Map[0]) {
		return false
	}
	if g.state.Map[newY][newX].Blocked {
		return false
	}

	for _, e := range g.state.Enemies {
		if e.X == newX && e.Y == newY {
			return false
		}
	}

	g.state.Player.X = newX
	g.state.Player.Y = newY
	g.isActioned = true
	return true
}

func TestMoveIntoEnemyFails(t *testing.T) {
	// 2x1 map
	m := [][]Tile{{{Type: "floor"}, {Type: "floor"}}}
	g := &Game{state: GameState{Map: m, Player: Player{Entity: Entity{X: 0, Y: 0}}, Enemies: []Enemy{{Entity: Entity{X: 1, Y: 0}}}}}

	moved := g.MovePlayerStub(1, 0)
	if moved {
		t.Fatalf("expected move to fail when enemy occupies tile")
	}
	if g.state.Player.X != 0 || g.state.Player.Y != 0 {
		t.Errorf("player position changed to (%d,%d)", g.state.Player.X, g.state.Player.Y)
	}
}

func TestMoveToEmptyTileSucceeds(t *testing.T) {
	// 2x1 map without enemies
	m := [][]Tile{{{Type: "floor"}, {Type: "floor"}}}
	g := &Game{state: GameState{Map: m, Player: Player{Entity: Entity{X: 0, Y: 0}}}}

	moved := g.MovePlayerStub(1, 0)
	if !moved {
		t.Fatalf("expected move to succeed on empty tile")
	}
	if g.state.Player.X != 1 || g.state.Player.Y != 0 {
		t.Errorf("player did not move correctly, got (%d,%d)", g.state.Player.X, g.state.Player.Y)
	}
}

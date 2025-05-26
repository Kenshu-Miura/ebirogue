//go:build test
// +build test

package main

func (g *Game) OpenDoor() {
	playerX, playerY := g.state.Player.X, g.state.Player.Y
	directions := []struct{ dx, dy int }{
		{0, -1}, // Up
		{0, 1},  // Down
		{-1, 0}, // Left
		{1, 0},  // Right
	}
	for _, dir := range directions {
		nx, ny := playerX+dir.dx, playerY+dir.dy
		if ny < 0 || ny >= len(g.state.Map) || nx < 0 || nx >= len(g.state.Map[0]) {
			continue
		}
		tile := g.state.Map[ny][nx]
		if tile.Type == "door" {
			g.state.Map[ny][nx] = Tile{Type: "corridor"}
			g.isActioned = true
		}
	}
}

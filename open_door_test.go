package main

import "testing"

// TestOpenDoorEdge ensures OpenDoor does not panic when the player is at the map edge.
func TestOpenDoorEdge(t *testing.T) {
	// create small map 3x3
	width, height := 3, 3
	m := make([][]Tile, height)
	for y := range m {
		m[y] = make([]Tile, width)
	}
	// place a door to the right of the player
	m[0][1] = Tile{Type: "door"}

	g := &Game{state: GameState{Map: m, Player: Player{Entity: Entity{X: 0, Y: 0}}}}

	// Should not panic
	g.OpenDoor()

	if m[0][1].Type != "corridor" {
		t.Errorf("expected tile to be opened, got %s", m[0][1].Type)
	}
}

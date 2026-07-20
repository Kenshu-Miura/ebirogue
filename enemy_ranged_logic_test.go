package main

import "testing"

func TestStraightLineDirection(t *testing.T) {
	tests := []struct {
		name                       string
		fromX, fromY, toX, toY     int
		wantX, wantY, wantDistance int
		wantOK                     bool
	}{
		{name: "horizontal", fromX: 2, fromY: 3, toX: 7, toY: 3, wantX: 1, wantDistance: 5, wantOK: true},
		{name: "vertical", fromX: 4, fromY: 8, toX: 4, toY: 3, wantY: -1, wantDistance: 5, wantOK: true},
		{name: "diagonal", fromX: 2, fromY: 2, toX: 5, toY: 5, wantX: 1, wantY: 1, wantDistance: 3, wantOK: true},
		{name: "not aligned", fromX: 2, fromY: 2, toX: 5, toY: 4, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotX, gotY, gotDistance, gotOK := straightLineDirection(tt.fromX, tt.fromY, tt.toX, tt.toY)
			if gotX != tt.wantX || gotY != tt.wantY || gotDistance != tt.wantDistance || gotOK != tt.wantOK {
				t.Fatalf("got (%d, %d, %d, %v), want (%d, %d, %d, %v)", gotX, gotY, gotDistance, gotOK, tt.wantX, tt.wantY, tt.wantDistance, tt.wantOK)
			}
		})
	}
}

func TestHasClearStraightLine(t *testing.T) {
	mapState := makeTileGrid(10, 10, "floor")
	enemies := []Enemy{
		{Entity: Entity{X: 1, Y: 5}},
		{Entity: Entity{X: 8, Y: 8}},
	}
	if !hasClearStraightLine(mapState, enemies, 0, 1, 5, 7, 5, 8) {
		t.Fatal("unobstructed line should be clear")
	}

	mapState[5][4] = Tile{Type: "wall", Blocked: true, BlockSight: true}
	if hasClearStraightLine(mapState, enemies, 0, 1, 5, 7, 5, 8) {
		t.Fatal("wall should block a straight-line attack")
	}

	mapState[5][4] = Tile{Type: "floor"}
	enemies = append(enemies, Enemy{Entity: Entity{X: 4, Y: 5}})
	if hasClearStraightLine(mapState, enemies, 0, 1, 5, 7, 5, 8) {
		t.Fatal("another enemy should block a straight-line attack")
	}
}

func TestRangedAreaBoundaries(t *testing.T) {
	if !withinRangedDistance(1, 1, 6, 4, 2, 5) {
		t.Fatal("stone throw should use Chebyshev distance and reach over an offset target")
	}
	if withinRangedDistance(1, 1, 7, 4, 2, 5) {
		t.Fatal("target beyond maximum range should be rejected")
	}
	if !withinBlastRadius(5, 5, 6, 4, 1) || withinBlastRadius(5, 5, 7, 5, 1) {
		t.Fatal("blast radius boundary is incorrect")
	}
}

package main

import (
	"reflect"
	"testing"
)

func TestGatherFloorPositionsOrderAndObstacles(t *testing.T) {
	// 通路・部屋の床だけを選び、プレイヤー・占有・壁・階段を避ける。
	tiles := [][]Tile{
		{{Type: "floor"}, {Type: "corridor"}, {Type: "floor"}},
		{{Type: "stairs"}, {Type: "floor"}, {Type: "floor", Blocked: true}},
		{{Type: "door"}, {Type: "floor"}}, // 長さの異なる行も扱える。
	}
	got := gatherFloorPositions(tiles, Coordinate{X: 1, Y: 1}, []Coordinate{{X: 0, Y: 0}})
	want := []Coordinate{{X: 1, Y: 0}, {X: 2, Y: 0}, {X: 1, Y: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	positions := []Coordinate{{X: 0, Y: 0}, {X: 2, Y: 2}, {X: 1, Y: 1}, {X: 1, Y: 0}}
	sortGatherPositions(positions, Coordinate{X: 1, Y: 1})
	want = []Coordinate{{X: 1, Y: 1}, {X: 0, Y: 0}, {X: 1, Y: 0}, {X: 2, Y: 2}}
	if !reflect.DeepEqual(positions, want) {
		t.Fatalf("distance/tie order %v", positions)
	}
}

func TestTroubleToFixEveryBadStatus(t *testing.T) {
	for _, status := range []StatusAilments{{Confusion: 1}, {Sleep: 1}, {Blind: 1}, {Slow: 1}, {MouthSeal: 1}, {BearTrap: 1}, {Paralysis: true}, {Seal: true}} {
		if got := troubleToFix(status, 0, true, 0, 8); got != troubleStatus {
			t.Fatalf("%+v: %v", status, got)
		}
	}
	if got := troubleToFix(StatusAilments{Haste: 3}, 50, false, 8, 8); got != troubleNone {
		t.Fatal("haste must remain beneficial")
	}
}

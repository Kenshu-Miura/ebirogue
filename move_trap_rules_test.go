package main

import "testing"

func TestMoveTrapRolls(t *testing.T) {
	if _, ok := rollWarpDestination(nil, func(int) int { t.Fatal("empty candidates consumed RNG"); return 0 }); ok {
		t.Fatal("empty warp succeeded")
	}
	candidates := []Coordinate{{X: 2, Y: 3}, {X: 4, Y: 5}}
	for i, want := range candidates {
		got, ok := rollWarpDestination(candidates, func(n int) int {
			if n != 2 {
				t.Fatal(n)
			}
			return i
		})
		if !ok || got != want {
			t.Fatalf("warp = %v/%v", got, ok)
		}
	}
	for _, tc := range []struct{ roll, want int }{{0, 4}, {32, 5}} {
		if got := rollPitfallDamage(func(int) int { return tc.roll }); got != tc.want {
			t.Fatalf("fall damage = %d", got)
		}
	}
}

package main

import (
	"reflect"
	"testing"
)

func TestRollItemTrapIndices(t *testing.T) {
	candidates := []int{2, 5, 8, 10}
	got := rollItemTrapIndices(candidates, 3, func(n int) int { return n - 1 })
	if !reflect.DeepEqual(got, []int{10, 8, 5}) || !reflect.DeepEqual(candidates, []int{2, 5, 8, 10}) {
		t.Fatalf("selection/input = %v/%v", got, candidates)
	}
	for _, limit := range []int{0, -1} {
		if len(rollItemTrapIndices(candidates, limit, func(int) int { t.Fatal("unexpected random call"); return 0 })) != 0 {
			t.Fatal("invalid limit")
		}
	}
	if len(rollItemTrapIndices(nil, 3, func(int) int { t.Fatal("unexpected random call"); return 0 })) != 0 {
		t.Fatal("empty candidates")
	}
	if len(rollItemTrapIndices(candidates, 20, func(int) int { return 0 })) != 4 {
		t.Fatal("selection exceeds candidates")
	}
}

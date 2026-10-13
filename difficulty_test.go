package main

import "testing"

func TestGoalFloorAndBossFloors(t *testing.T) {
	want := map[Difficulty]int{DifficultyBeginner: 10, DifficultyIntermediate: 20, DifficultyAdvanced: 30}
	for d, floor := range want {
		if got := goalFloor(d); got != floor {
			t.Errorf("goalFloor(%v) = %d, want %d", d, got, floor)
		}
		if !isClearingBossFloor(floor, d) {
			t.Errorf("%v should clear on floor %d", d, floor)
		}
	}
	for floor := 0; floor <= 40; floor++ {
		boss := floor == 10 || floor == 20 || floor == 30
		if isBossFloor(floor) != boss {
			t.Errorf("isBossFloor(%d) = %v", floor, !boss)
		}
		if _, ok := bossIDForFloor(floor); ok != boss {
			t.Errorf("bossIDForFloor(%d) ok = %v", floor, ok)
		}
	}
	// 目標階より手前のボスでは冒険は続く
	if isClearingBossFloor(10, DifficultyIntermediate) || isClearingBossFloor(20, DifficultyAdvanced) {
		t.Error("an earlier boss must not clear the run")
	}
}

func TestStairsUsableRules(t *testing.T) {
	if !stairsUsable(9, false) || stairsUsable(10, false) || !stairsUsable(10, true) {
		t.Error("boss floor stairs open only after the boss is defeated")
	}
	if stairsUsable(30, true) || stairsUsable(31, true) {
		t.Error("cannot go below the final floor")
	}
}

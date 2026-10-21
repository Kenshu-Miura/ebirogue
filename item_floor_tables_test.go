package main

import "testing"

func TestSpawnBandIndex(t *testing.T) {
	cases := map[int]int{0: 0, 1: 0, 3: 0, 4: 1, 8: 1, 9: 2, 12: 2, 13: 3, 16: 3, 17: 4, 20: 4, 21: 5, 25: 5, 26: 6, 30: 6, 31: 6}
	for floor, want := range cases {
		if got := spawnBandIndex(floor); got != want {
			t.Errorf("spawnBandIndex(%d) = %d, want %d", floor, got, want)
		}
	}
	if got := spawnBandLabel(10); got != "9〜12階" {
		t.Errorf("label = %q", got)
	}
	// 帯は1階から30階まで切れ目なく並ぶ。
	next := 1
	for _, band := range spawnFloorBands {
		if band.From != next || band.To < band.From {
			t.Fatalf("band %+v does not start at %d", band, next)
		}
		next = band.To + 1
	}
	if next != finalFloor+1 {
		t.Fatalf("bands end at %d, want %d", next-1, finalFloor)
	}
}

func TestItemFloorRuleAppears(t *testing.T) {
	rule := itemFloorRule{From: 4, Until: 9}
	for floor, want := range map[int]bool{3: false, 4: true, 8: true, 9: false} {
		if got := rule.appearsOnFloor(floor); got != want {
			t.Errorf("floor %d: %v, want %v", floor, got, want)
		}
	}
	if !(itemFloorRule{}).appearsOnFloor(1) || !(itemFloorRule{}).appearsOnFloor(30) {
		t.Fatal("default rule should appear on every floor")
	}
	// 書いていない ID は全階・重み1。
	if itemFloorWeight(-1, 1) != 1 || itemFloorWeight(-1, 30) != 1 {
		t.Fatal("unlisted item should have weight 1")
	}
}

func TestScaledSpawnWeight(t *testing.T) {
	if got := scaledSpawnWeight(4, 1); got != 4*spawnWeightScale {
		t.Errorf("neutral = %d", got)
	}
	if got := scaledSpawnWeight(1, 1.5); got != 150 {
		t.Errorf("1.5x = %d", got)
	}
	if got := scaledSpawnWeight(1, 0.0001); got != 1 {
		t.Errorf("tiny multiplier should keep weight 1, got %d", got)
	}
	if scaledSpawnWeight(0, 2) != 0 || scaledSpawnWeight(3, 0) != 0 {
		t.Error("zero weight or multiplier should give 0")
	}
	if scaledCount(10, 1.25) != 13 || scaledCount(7, 1) != 7 || scaledCount(5, -1) != 0 {
		t.Error("scaledCount")
	}
	tuning := difficultySpawnTuning{FoodHealWeight: 1, StrongItemWeight: 1, ItemCount: 1.5, TrapCount: 0.5}
	if tuning.floorItemCount() != 15 || tuning.floorTrapCount(7) != 4 {
		t.Errorf("counts %d %d", tuning.floorItemCount(), tuning.floorTrapCount(7))
	}
	if spawnTuningFor(Difficulty(99)) != neutralSpawnTuning {
		t.Error("unknown difficulty should be neutral")
	}
}

// 偽の表で、補正の倍率がカテゴリと道具の重みに掛かることを確かめる。
func TestPickFloorSpawnIDTuning(t *testing.T) {
	categories := []floorSpawnCategory{
		{Name: "Food", Weight: 1, Entries: []floorSpawnEntry{{ID: 1, Weight: 1}}},
		{Name: "Potion", Weight: 1, Entries: []floorSpawnEntry{{ID: 2, Weight: 1, Heal: true}, {ID: 3, Weight: 1}}},
		{Name: "Weapon", Weight: 2, Entries: []floorSpawnEntry{{ID: 4, Weight: 1}, {ID: 5, Weight: 1, Strong: true}}},
	}
	count := func(tuning difficultySpawnTuning) map[int]int {
		counts := map[int]int{}
		// 1回目（カテゴリ）と2回目（カテゴリ内）の乱数を全通り試す。
		total := 0
		for _, c := range categories {
			total += c.categoryWeightWith(tuning)
		}
		for r1 := 0; r1 < total; r1++ {
			// カテゴリを決める乱数 r1 に対し、カテゴリ内を全通り回す。
			calls := 0
			var innerTotal int
			pickFloorSpawnID(categories, tuning, func(n int) int {
				calls++
				if calls == 1 {
					return r1
				}
				innerTotal = n
				return 0
			}, -1)
			for r2 := 0; r2 < innerTotal; r2++ {
				calls = 0
				id := pickFloorSpawnID(categories, tuning, func(n int) int {
					calls++
					if calls == 1 {
						return r1
					}
					return r2
				}, -1)
				counts[id]++
			}
		}
		return counts
	}
	neutral := count(neutralSpawnTuning)
	// 補正なし: 食料 1/4、薬 1/4（回復薬と通常で半分ずつ）、武器 2/4（強い道具と通常で半分ずつ）。
	// 各カテゴリの乱数の幅 × カテゴリ内の幅で数えるので、比で比べる。
	if neutral[2] != neutral[3] || neutral[4] != neutral[5] {
		t.Fatalf("neutral counts %v", neutral)
	}
	boosted := count(difficultySpawnTuning{FoodHealWeight: 2, StrongItemWeight: 3, ItemCount: 1, TrapCount: 1})
	if boosted[2] != 2*boosted[3] {
		t.Errorf("heal potion should be twice the normal potion: %v", boosted)
	}
	if boosted[5] != 3*boosted[4] {
		t.Errorf("strong weapon should be three times the normal weapon: %v", boosted)
	}
	if got := pickFloorSpawnID(nil, neutralSpawnTuning, func(int) int { return 0 }, 42); got != 42 {
		t.Errorf("empty table should return fallback, got %d", got)
	}
}

func TestPickWeightedTrapID(t *testing.T) {
	ids := []int{8, 9, 11}
	counts := map[int]int{}
	total := trapFloorWeight(8, 1) + trapFloorWeight(9, 1) + trapFloorWeight(11, 1)
	for r := 0; r < total; r++ {
		counts[pickWeightedTrapID(ids, 1, func(int) int { return r }, -1)]++
	}
	if counts[9] != 0 || counts[11] != 0 || counts[8] != trapFloorWeight(8, 1) {
		t.Fatalf("floor 1 counts %v", counts)
	}
	if got := pickWeightedTrapID([]int{999}, 1, func(int) int { return 0 }, -1); got != -1 {
		t.Errorf("unknown trap should return fallback, got %d", got)
	}
}

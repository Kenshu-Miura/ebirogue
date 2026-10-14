package main

import "testing"

func TestAccessorySatietyBonus(t *testing.T) {
	for _, tc := range []struct {
		abilities []EquipmentAbilityID
		want      int
	}{
		{nil, 0}, {[]EquipmentAbilityID{"power_boost"}, 0},
		{[]EquipmentAbilityID{"stomach_expand"}, 25}, {[]EquipmentAbilityID{"stomach_shrink"}, -25},
		{[]EquipmentAbilityID{"stomach_expand", "stomach_expand"}, 25},
		{[]EquipmentAbilityID{"stomach_expand", "stomach_shrink"}, 0},
	} {
		if got := accessorySatietyBonus(tc.abilities); got != tc.want {
			t.Fatalf("%v: %d != %d", tc.abilities, got, tc.want)
		}
	}
}

func TestRingSpawnInterval(t *testing.T) {
	for _, tc := range []struct{ base, stacks, want int }{
		{20, 0, 20}, {30, 0, 30}, {20, 1, 10}, {30, 1, 15}, {20, 2, 6}, {30, 2, 10}, {10, 2, 5}, {20, 10, 5},
	} {
		if got := ringSpawnInterval(tc.base, tc.stacks); got != tc.want {
			t.Fatalf("%+v: %d", tc, got)
		}
	}
}

func TestRingChanceAndMoney(t *testing.T) {
	for _, threshold := range []int{1, 2, 3, 10} {
		for stacks := 0; stacks <= 2; stacks++ {
			for roll := 0; roll < 200; roll++ {
				if got := ringChance(roll, threshold, stacks); got != (roll < threshold*stacks) {
					t.Fatalf("chance %d/%d/%d", roll, threshold, stacks)
				}
			}
		}
	}
	for _, tc := range []struct{ cash, want int }{{0, 0}, {1, 1}, {49, 1}, {50, 1}, {100, 2}, {999, 19}} {
		if got := ringMoneyDrop(tc.cash); got != tc.want {
			t.Fatalf("cash %d: %d", tc.cash, got)
		}
	}
}

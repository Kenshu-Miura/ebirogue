package main

import "testing"

func TestSatoriSatietyCap(t *testing.T) {
	for _, satiety := range []int{0, 1, 50, 100} {
		if got := satoriSatietyCap(satiety, true); got != min(satiety, 1) {
			t.Errorf("equipped cap(%d) = %d", satiety, got)
		}
		if got := satoriSatietyCap(satiety, false); got != satiety {
			t.Errorf("unequipped cap(%d) = %d", satiety, got)
		}
	}
	for _, count := range []int{1, 5, 10, 20, 1000} {
		if shouldReduceSatiety(count, []EquipmentAbilityID{"satori"}) {
			t.Errorf("satori loses satiety at %d", count)
		}
	}
}

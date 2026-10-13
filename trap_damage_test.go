package main

import "testing"

func TestLargeMineDamage(t *testing.T) {
	for hp := 1; hp <= 1000; hp++ {
		got := largeMineTrapDamage(hp)
		if got <= mineTrapDamage(hp) || got > hp || got != (hp*largeMineHealthNumerator+largeMineHealthDenominator-1)/largeMineHealthDenominator {
			t.Fatalf("hp %d: damage %d", hp, got)
		}
	}
	if largeMineTrapDamage(0) != 0 {
		t.Fatal("dead player")
	}
}

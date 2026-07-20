package main

import "testing"

func TestSatietyLossInterval(t *testing.T) {
	tests := []struct {
		name      string
		abilities []EquipmentAbilityID
		want      int
	}{
		{name: "no armor abilities", want: 10},
		{name: "unrelated ability", abilities: []EquipmentAbilityID{"other"}, want: 10},
		{name: "satiety conservation", abilities: []EquipmentAbilityID{AbilitySatietyConservation}, want: 20},
		{name: "duplicate ability does not stack", abilities: []EquipmentAbilityID{AbilitySatietyConservation, AbilitySatietyConservation}, want: 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := satietyLossInterval(tt.abilities); got != tt.want {
				t.Fatalf("satietyLossInterval(%v) = %d, want %d", tt.abilities, got, tt.want)
			}
		})
	}
}

func TestShouldReduceSatiety(t *testing.T) {
	tests := []struct {
		name      string
		moveCount int
		abilities []EquipmentAbilityID
		want      bool
	}{
		{name: "initial turn", moveCount: 0, want: false},
		{name: "normal armor at ten turns", moveCount: 10, want: true},
		{name: "conservation prevents loss at ten turns", moveCount: 10, abilities: []EquipmentAbilityID{AbilitySatietyConservation}, want: false},
		{name: "conservation loses satiety at twenty turns", moveCount: 20, abilities: []EquipmentAbilityID{AbilitySatietyConservation}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldReduceSatiety(tt.moveCount, tt.abilities); got != tt.want {
				t.Fatalf("shouldReduceSatiety(%d, %v) = %t, want %t", tt.moveCount, tt.abilities, got, tt.want)
			}
		})
	}
}

func TestSlayerMultiplier(t *testing.T) {
	tests := []struct {
		name           string
		abilities      []EquipmentAbilityID
		traits         []EnemyTrait
		wantMultiplier float64
		wantMatched    bool
	}{
		{name: "dragon match", abilities: []EquipmentAbilityID{AbilityDragonSlayer}, traits: []EnemyTrait{EnemyTraitDragon}, wantMultiplier: slayerDamageMultiplier, wantMatched: true},
		{name: "ghost match", abilities: []EquipmentAbilityID{AbilityGhostSlayer}, traits: []EnemyTrait{EnemyTraitGhost}, wantMultiplier: slayerDamageMultiplier, wantMatched: true},
		{name: "different trait", abilities: []EquipmentAbilityID{AbilityDragonSlayer}, traits: []EnemyTrait{EnemyTraitGhost}, wantMultiplier: 1.0},
		{name: "no weapon ability", traits: []EnemyTrait{EnemyTraitDragon}, wantMultiplier: 1.0},
		{name: "multiple matches do not stack", abilities: []EquipmentAbilityID{AbilityOneEyeSlayer, AbilityDrainerSlayer}, traits: []EnemyTrait{EnemyTraitOneEye, EnemyTraitDrainer}, wantMultiplier: slayerDamageMultiplier, wantMatched: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMultiplier, gotMatched := slayerMultiplier(tt.abilities, tt.traits)
			if gotMultiplier != tt.wantMultiplier || gotMatched != tt.wantMatched {
				t.Fatalf("slayerMultiplier() = (%f, %t), want (%f, %t)", gotMultiplier, gotMatched, tt.wantMultiplier, tt.wantMatched)
			}
		})
	}
}

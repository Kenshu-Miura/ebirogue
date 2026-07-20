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

func TestApplySlayerBonus(t *testing.T) {
	tests := []struct {
		name        string
		damage      int
		abilities   []EquipmentAbilityID
		traits      []EnemyTrait
		wantDamage  int
		wantMatched bool
	}{
		{name: "dragon match", damage: 10, abilities: []EquipmentAbilityID{AbilityDragonSlayer}, traits: []EnemyTrait{EnemyTraitDragon}, wantDamage: 15, wantMatched: true},
		{name: "ghost match rounds up", damage: 9, abilities: []EquipmentAbilityID{AbilityGhostSlayer}, traits: []EnemyTrait{EnemyTraitGhost}, wantDamage: 14, wantMatched: true},
		{name: "different trait", damage: 10, abilities: []EquipmentAbilityID{AbilityDragonSlayer}, traits: []EnemyTrait{EnemyTraitGhost}, wantDamage: 10},
		{name: "no weapon ability", damage: 10, traits: []EnemyTrait{EnemyTraitDragon}, wantDamage: 10},
		{name: "multiple matches do not stack", damage: 10, abilities: []EquipmentAbilityID{AbilityOneEyeSlayer, AbilityDrainerSlayer}, traits: []EnemyTrait{EnemyTraitOneEye, EnemyTraitDrainer}, wantDamage: 15, wantMatched: true},
		{name: "zero damage stays zero", damage: 0, abilities: []EquipmentAbilityID{AbilityDragonSlayer}, traits: []EnemyTrait{EnemyTraitDragon}, wantDamage: 0},
		{name: "negative boundary stays unchanged", damage: -1, abilities: []EquipmentAbilityID{AbilityDragonSlayer}, traits: []EnemyTrait{EnemyTraitDragon}, wantDamage: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDamage, gotMatched := applySlayerBonus(tt.damage, tt.abilities, tt.traits)
			if gotDamage != tt.wantDamage || gotMatched != tt.wantMatched {
				t.Fatalf("applySlayerBonus() = (%d, %t), want (%d, %t)", gotDamage, gotMatched, tt.wantDamage, tt.wantMatched)
			}
		})
	}
}

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

func TestPlayerAttackDeltas(t *testing.T) {
	tests := []struct {
		name      string
		dx, dy    int
		abilities []EquipmentAbilityID
		want      []Coordinate
	}{
		{name: "normal attack", dx: 0, dy: -1, want: []Coordinate{{X: 0, Y: -1}}},
		{name: "three way up", dx: 0, dy: -1, abilities: []EquipmentAbilityID{AbilityThreeWayAttack}, want: []Coordinate{{X: 0, Y: -1}, {X: -1, Y: -1}, {X: 1, Y: -1}}},
		{name: "three way diagonal", dx: 1, dy: -1, abilities: []EquipmentAbilityID{AbilityThreeWayAttack}, want: []Coordinate{{X: 1, Y: -1}, {X: 0, Y: -1}, {X: 1, Y: 0}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := playerAttackDeltas(tt.dx, tt.dy, tt.abilities)
			if len(got) != len(tt.want) {
				t.Fatalf("playerAttackDeltas() length = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("playerAttackDeltas()[%d] = %#v, want %#v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestRollPlayerAttackHit(t *testing.T) {
	if !rollPlayerAttackHit(nil, func(int) int { return 89 }) {
		t.Fatal("roll 89 should hit with normal accuracy")
	}
	if rollPlayerAttackHit(nil, func(int) int { return 90 }) {
		t.Fatal("roll 90 should miss with normal accuracy")
	}
	if !rollPlayerAttackHit([]EquipmentAbilityID{AbilitySureHit}, func(int) int { return 99 }) {
		t.Fatal("sure-hit weapon should always hit")
	}
}

func TestNextDisposableWeaponAttackPower(t *testing.T) {
	if got, changed := nextDisposableWeaponAttackPower(12, nil); got != 12 || changed {
		t.Fatalf("normal weapon = (%d, %t), want (12, false)", got, changed)
	}
	if got, changed := nextDisposableWeaponAttackPower(12, []EquipmentAbilityID{AbilityDisposable}); got != 11 || !changed {
		t.Fatalf("disposable weapon = (%d, %t), want (11, true)", got, changed)
	}
	if got, changed := nextDisposableWeaponAttackPower(0, []EquipmentAbilityID{AbilityDisposable}); got != 0 || changed {
		t.Fatalf("spent disposable weapon = (%d, %t), want (0, false)", got, changed)
	}
}

func TestIsDiggableTile(t *testing.T) {
	wall := Tile{Type: "wall", Blocked: true}
	other := Tile{Type: "other", Blocked: true}
	floor := Tile{Type: "floor", Blocked: false}
	if !isDiggableTile(wall, 2, 2, 5, 5) || !isDiggableTile(other, 2, 2, 5, 5) {
		t.Fatal("interior wall and other tiles should be diggable")
	}
	if isDiggableTile(floor, 2, 2, 5, 5) {
		t.Fatal("floor should not be diggable")
	}
	if isDiggableTile(wall, 0, 2, 5, 5) || isDiggableTile(wall, 4, 2, 5, 5) {
		t.Fatal("outer map edge should not be diggable")
	}
}

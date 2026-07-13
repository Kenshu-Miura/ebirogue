//go:build !test

package main

import "testing"

func TestAddedEquipmentTemplates(t *testing.T) {
	weapons := map[int]int{20: 2, 21: 4, 22: 6}
	for id, wantPower := range weapons {
		weapon, ok := buildItemFromTemplate(id, 0, 0).(*Weapon)
		if !ok || weapon.AttackPower != wantPower {
			t.Fatalf("weapon %d = %#v, want attack power %d", id, weapon, wantPower)
		}
	}

	armors := map[int]int{23: 2, 24: 5, 25: 3}
	for id, wantPower := range armors {
		armor, ok := buildItemFromTemplate(id, 0, 0).(*Armor)
		if !ok || armor.DefensePower != wantPower {
			t.Fatalf("armor %d = %#v, want defense power %d", id, armor, wantPower)
		}
		if id == 25 && !hasEquipmentAbility(armor.Abilities, AbilitySatietyConservation) {
			t.Fatal("皮甲の盾に満腹度消費軽減能力がありません")
		}
	}
}

func TestAddedContentDefinitions(t *testing.T) {
	if MonsterDefinitions[2].Name != "マムル" || MonsterDefinitions[3].Name != "くねくねハニー" {
		t.Fatal("added monster definitions are missing")
	}

	wantTraps := []string{"睡眠ガスの罠", "毒矢の罠", "鈍足の罠", "地雷"}
	for id, wantName := range wantTraps {
		if got := createMapTrapByID(id, 1, 2); got.Name != wantName {
			t.Fatalf("trap %d = %q, want %q", id, got.Name, wantName)
		}
	}
}

func TestRecoveryItemTemplates(t *testing.T) {
	foods := map[int]struct {
		name         string
		satiety      int
		fullRecovery bool
	}{
		26: {name: "ジャンボウインナー", satiety: 100},
		27: {name: "海老天むす", fullRecovery: true},
	}
	for id, want := range foods {
		food, ok := buildItemFromTemplate(id, 0, 0).(*Food)
		if !ok || food.Name != want.name || food.Satiety != want.satiety || food.FullRecovery != want.fullRecovery {
			t.Fatalf("food %d = %#v, want %#v", id, food, want)
		}
	}

	potions := map[int]struct {
		name         string
		health       int
		fullRecovery bool
	}{
		28: {name: "大粒ミンティア", health: 60},
		29: {name: "海老印の栄養ドリンク", fullRecovery: true},
	}
	for id, want := range potions {
		potion, ok := buildItemFromTemplate(id, 0, 0).(*Potion)
		if !ok || potion.Name != want.name || potion.Health != want.health || potion.FullRecovery != want.fullRecovery {
			t.Fatalf("potion %d = %#v, want %#v", id, potion, want)
		}
	}
}

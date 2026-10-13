//go:build !test

package main

// 罠のコンテンツテスト。

import "testing"

func TestRollRustTarget(t *testing.T) {
	intnZero := func(int) int { return 0 }
	intnOne := func(int) int { return 1 }
	if got := rollRustTarget(true, true, intnZero); got != rustWeapon {
		t.Fatalf("both equipped with roll 0 = %d, want rustWeapon", got)
	}
	if got := rollRustTarget(true, true, intnOne); got != rustArmor {
		t.Fatalf("both equipped with roll 1 = %d, want rustArmor", got)
	}
	if got := rollRustTarget(true, false, intnZero); got != rustWeapon {
		t.Fatalf("weapon only = %d, want rustWeapon", got)
	}
	if got := rollRustTarget(false, true, intnZero); got != rustArmor {
		t.Fatalf("armor only = %d, want rustArmor", got)
	}
	if got := rollRustTarget(false, false, intnZero); got != rustNone {
		t.Fatalf("nothing equipped = %d, want rustNone", got)
	}
}

func TestRustTrapRustsWeapon(t *testing.T) {
	weapon := &Weapon{BaseItem: BaseItem{Name: "こん棒", Type: "Weapon"}, AttackPower: 2, Sharpness: 1}
	g := &Game{state: GameState{Player: Player{Inventory: []Item{weapon}}}}
	g.state.Player.EquipItem(weapon)
	attackAfterEquip := g.state.Player.AttackPower

	rustTrapEffect(g)
	g.ActionQueue.Queue[0].Execute(g)

	if weapon.Sharpness != 0 {
		t.Fatalf("weapon sharpness = %d, want 0", weapon.Sharpness)
	}
	if g.state.Player.AttackPower != attackAfterEquip-1 {
		t.Fatalf("attack power = %d, want %d", g.state.Player.AttackPower, attackAfterEquip-1)
	}
}

func TestRustTrapRespectsRustProof(t *testing.T) {
	armor := &Armor{BaseItem: BaseItem{Name: "木甲の盾", Type: "Armor"}, DefensePower: 2, Sharpness: 1, RustProof: true}
	g := &Game{state: GameState{Player: Player{Inventory: []Item{armor}}}}
	g.state.Player.EquipItem(armor)
	defenseAfterEquip := g.state.Player.DefensePower

	rustTrapEffect(g)
	g.ActionQueue.Queue[0].Execute(g)

	if armor.Sharpness != 1 || g.state.Player.DefensePower != defenseAfterEquip {
		t.Fatalf("rust-proof armor changed: sharpness %d, defense %d", armor.Sharpness, g.state.Player.DefensePower)
	}
	if len(g.ActionQueue.Queue) < 2 || g.ActionQueue.Queue[1].Message != "しかし木甲の盾は錆びなかった" {
		t.Fatal("さび止め済みの装備は錆びないメッセージを表示するはず")
	}
}

func TestRustTrapWithoutEquipment(t *testing.T) {
	g := &Game{state: GameState{Player: Player{}}}

	rustTrapEffect(g)
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.ActionQueue.Queue) < 2 || g.ActionQueue.Queue[1].Message != "しかし何も起こらなかった" {
		t.Fatal("装備なしでは不発メッセージを表示するはず")
	}
}

func TestTrapCardIncreasesTraps(t *testing.T) {
	card := buildItemFromTemplate(40, 0, 0)
	g := &Game{
		state: GameState{
			Map:      makeTestFloorMap(8, 8),
			Player:   Player{Entity: Entity{X: 2, Y: 2}, Inventory: []Item{card}},
			MapTraps: []MapTrap{createMapTrapByID(0, 3, 3)},
		},
		rooms: []Room{{ID: 0, X: 0, Y: 0, Width: 8, Height: 8}},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.state.MapTraps) < 6 || len(g.state.MapTraps) > 9 {
		t.Fatalf("traps = %d, want 6..9", len(g.state.MapTraps))
	}
	seen := map[Coordinate]bool{}
	for _, trap := range g.state.MapTraps {
		pos := Coordinate{X: trap.X, Y: trap.Y}
		if seen[pos] {
			t.Fatalf("duplicate trap at (%d, %d)", trap.X, trap.Y)
		}
		seen[pos] = true
		if trap.X == 2 && trap.Y == 2 {
			t.Fatal("trap should not be placed on the player")
		}
	}
	for _, trap := range g.state.MapTraps[1:] {
		if trap.Discovered {
			t.Fatal("added traps should be undiscovered")
		}
	}
}

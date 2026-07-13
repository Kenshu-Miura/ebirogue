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

func TestAddedCardTemplates(t *testing.T) {
	cards := map[int]string{
		30: "あかりのカード",
		31: "真空斬りのカード",
	}
	for id, wantName := range cards {
		card, ok := buildItemFromTemplate(id, 0, 0).(*Card)
		if !ok || card.Name != wantName {
			t.Fatalf("card %d = %#v, want %q", id, card, wantName)
		}
	}
}

func TestRollVacuumSlashDamage(t *testing.T) {
	if got := rollVacuumSlashDamage(func(int) int { return 0 }); got != 12 {
		t.Fatalf("minimum damage = %d, want 12", got)
	}
	if got := rollVacuumSlashDamage(func(n int) int { return n - 1 }); got != 24 {
		t.Fatalf("maximum damage = %d, want 24", got)
	}
}

func TestRoomWideCardEffectArea(t *testing.T) {
	rooms := []Room{{ID: 1, X: 0, Y: 0, Width: 5, Height: 5}}
	if !isInRoomWideEffect(2, 2, 3, 3, rooms) {
		t.Fatal("enemy in the same room should be affected")
	}
	if isInRoomWideEffect(2, 2, 6, 6, rooms) {
		t.Fatal("enemy outside the room should not be affected")
	}
	if !isInRoomWideEffect(6, 6, 7, 7, rooms) {
		t.Fatal("adjacent enemy in a corridor should be affected")
	}
	if isInRoomWideEffect(6, 6, 8, 6, rooms) {
		t.Fatal("distant enemy in a corridor should not be affected")
	}
}

func TestRevealFloorCard(t *testing.T) {
	card := buildItemFromTemplate(30, 0, 0)
	floorItem := buildItemFromTemplate(1, 4, 4)
	g := &Game{
		state: GameState{
			Map:      [][]Tile{{{Type: "floor"}, {Type: "wall"}}},
			Player:   Player{Inventory: []Item{card}},
			Enemies:  []Enemy{{Entity: Entity{X: 3, Y: 3}}},
			Items:    []Item{floorItem},
			MapTraps: []MapTrap{{X: 1, Y: 1, Discovered: false}},
		},
	}

	card.Use(g)
	if len(g.state.Player.Inventory) != 0 {
		t.Fatal("used card should be removed from inventory")
	}
	g.ActionQueue.Queue[0].Execute(g)

	if !g.state.Map[0][0].Visited || !g.state.Map[0][1].Visited {
		t.Fatal("all map tiles should be revealed")
	}
	if !g.state.Enemies[0].PlayerDiscovered || !g.state.Enemies[0].ShowOnMiniMap {
		t.Fatal("enemy should be revealed on the minimap")
	}
	if !floorItem.GetPlayerDiscovered() || !floorItem.GetShowOnMiniMap() {
		t.Fatal("floor item should be revealed on the minimap")
	}
	if g.state.MapTraps[0].Discovered {
		t.Fatal("あかりのカード should not reveal traps")
	}
}

func TestVacuumSlashCardDamagesOnlyEffectArea(t *testing.T) {
	card := buildItemFromTemplate(31, 0, 0)
	g := &Game{
		state: GameState{
			Player: Player{Entity: Entity{X: 2, Y: 2}, Inventory: []Item{card}, Level: 1},
			Enemies: []Enemy{
				{Entity: Entity{X: 3, Y: 3}, Health: 100, StatusAilments: StatusAilments{Sleep: 5, Paralysis: true}},
				{Entity: Entity{X: 6, Y: 6}, Health: 100},
			},
		},
		rooms: []Room{{ID: 1, X: 0, Y: 0, Width: 5, Height: 5}},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if got := g.state.Enemies[0].Health; got < 76 || got > 88 {
		t.Fatalf("same-room enemy health = %d, want 76..88", got)
	}
	if g.state.Enemies[0].StatusAilments.Sleep != 0 || g.state.Enemies[0].StatusAilments.Paralysis {
		t.Fatal("damage should wake and release the affected enemy")
	}
	if got := g.state.Enemies[1].Health; got != 100 {
		t.Fatalf("outside enemy health = %d, want 100", got)
	}
}

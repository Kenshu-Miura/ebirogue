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

func TestSleepCardMarksHasteOnWake(t *testing.T) {
	card := buildItemFromTemplate(12, 0, 0)
	g := &Game{
		state: GameState{
			Player: Player{Entity: Entity{X: 2, Y: 2}, Inventory: []Item{card}},
			Enemies: []Enemy{
				{Entity: Entity{X: 3, Y: 3}, Health: 10},
				{Entity: Entity{X: 6, Y: 6}, Health: 10},
			},
		},
		rooms: []Room{{ID: 1, X: 0, Y: 0, Width: 5, Height: 5}},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if got := g.state.Enemies[0].StatusAilments; got.Sleep != 10 || !got.HasteOnWake {
		t.Fatalf("same-room enemy status = %#v, want Sleep 10 with HasteOnWake", got)
	}
	if got := g.state.Enemies[1].StatusAilments; got.Sleep != 0 || got.HasteOnWake {
		t.Fatalf("outside enemy status = %#v, want unaffected", got)
	}
}

func TestEnemyWakesUpHasted(t *testing.T) {
	g := &Game{
		state: GameState{
			Enemies: []Enemy{{Name: "マムル", StatusAilments: StatusAilments{Sleep: 1, HasteOnWake: true}}},
		},
	}

	g.decrementStatusAilments()

	if got := g.state.Enemies[0].StatusAilments; got.Sleep != 0 || got.HasteOnWake || got.Haste != hasteOnWakeTurns {
		t.Fatalf("enemy status after waking = %#v, want Haste %d", got, hasteOnWakeTurns)
	}
	if len(g.ActionQueue.Queue) == 0 {
		t.Fatal("waking up hasted should enqueue a message")
	}

	g.decrementStatusAilments()
	if got := g.state.Enemies[0].StatusAilments.Haste; got != hasteOnWakeTurns-1 {
		t.Fatalf("haste turns = %d, want %d", got, hasteOnWakeTurns-1)
	}
}

func TestRollIdentifyAll(t *testing.T) {
	if !rollIdentifyAll(func(int) int { return 0 }) {
		t.Fatal("最小値では全識別が発動するはず")
	}
	if rollIdentifyAll(func(n int) int { return n - 1 }) {
		t.Fatal("最大値では全識別は発動しないはず")
	}
}

func TestIdentifyAllInventory(t *testing.T) {
	weapon := buildItemFromTemplate(20, 0, 0)
	cane := buildItemFromTemplate(9, 0, 0)
	food := buildItemFromTemplate(1, 0, 0)
	g := &Game{
		state: GameState{
			Player: Player{Inventory: []Item{weapon, cane, food}},
		},
	}

	if got := identifyAllInventory(g); got != 2 {
		t.Fatalf("identified %d items, want 2", got)
	}
	if !weapon.(Identifiable).GetIdentified() || !cane.(Identifiable).GetIdentified() {
		t.Fatal("識別可能なアイテムはすべて識別されるはず")
	}
}

func TestEquipmentCardTemplates(t *testing.T) {
	cards := map[int]string{
		32: "おはらいのカード",
		33: "武器強化のカード",
		34: "盾強化のカード",
	}
	for id, wantName := range cards {
		card, ok := buildItemFromTemplate(id, 0, 0).(*Card)
		if !ok || card.Name != wantName {
			t.Fatalf("card %d = %#v, want %q", id, card, wantName)
		}
	}
}

func TestUncursePlayerBelongings(t *testing.T) {
	weapon := &Weapon{BaseItem: BaseItem{Name: "こん棒", Type: "Weapon"}, AttackPower: 2, Sharpness: -1, Cursed: true}
	accessory := &Accessory{BaseItem: BaseItem{Name: "鼓舞の指輪", Type: "Accessory"}, Cursed: true}
	plainArmor := &Armor{BaseItem: BaseItem{Name: "木甲の盾", Type: "Armor"}, DefensePower: 2}
	player := &Player{Inventory: []Item{weapon, accessory, plainArmor}}
	player.EquipItem(weapon)
	player.EquipItem(accessory)
	attackAfterEquip := player.AttackPower

	uncursed := uncursePlayerBelongings(player)

	if len(uncursed) != 2 {
		t.Fatalf("uncursed = %v, want 2 items", uncursed)
	}
	if weapon.Cursed || accessory.Cursed {
		t.Fatal("装備中の呪いはすべて解けるはず")
	}
	if player.AttackPower != attackAfterEquip {
		t.Fatalf("attack power = %d, want unchanged %d", player.AttackPower, attackAfterEquip)
	}
	// 呪われたアクセサリは補正が反転しているため、解呪後は正の補正へ付け直される
	if player.Power != 3 || player.MaxPower != 3 {
		t.Fatalf("accessory stats = Power %d MaxPower %d, want 3/3", player.Power, player.MaxPower)
	}
}

func TestReinforceWeaponCard(t *testing.T) {
	card := buildItemFromTemplate(33, 0, 0)
	weapon := &Weapon{BaseItem: BaseItem{Name: "こん棒", Type: "Weapon"}, AttackPower: 2, Cursed: true}
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card, weapon}}}}
	g.state.Player.EquipItem(weapon)

	card.Use(g)
	if len(g.state.Player.Inventory) != 1 {
		t.Fatal("used card should be removed from inventory")
	}
	g.ActionQueue.Queue[0].Execute(g)

	if weapon.Sharpness != 1 || weapon.Cursed {
		t.Fatalf("weapon = %#v, want Sharpness 1 and uncursed", weapon)
	}
	if g.state.Player.AttackPower != 3 {
		t.Fatalf("player attack power = %d, want 3", g.state.Player.AttackPower)
	}
}

func TestReinforceArmorCard(t *testing.T) {
	card := buildItemFromTemplate(34, 0, 0)
	armor := &Armor{BaseItem: BaseItem{Name: "木甲の盾", Type: "Armor"}, DefensePower: 2}
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card, armor}}}}
	g.state.Player.EquipItem(armor)

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if armor.Sharpness != 1 {
		t.Fatalf("armor sharpness = %d, want 1", armor.Sharpness)
	}
	if g.state.Player.DefensePower != 3 {
		t.Fatalf("player defense power = %d, want 3", g.state.Player.DefensePower)
	}
}

func TestReinforceArmorCardWithoutShield(t *testing.T) {
	card := buildItemFromTemplate(34, 0, 0)
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card}}}}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if g.state.Player.DefensePower != 0 {
		t.Fatalf("defense power = %d, want unchanged 0", g.state.Player.DefensePower)
	}
	if len(g.ActionQueue.Queue) < 2 || g.ActionQueue.Queue[1].Message != "しかし何も起こらなかった" {
		t.Fatal("盾なしで使用した場合は不発メッセージを表示するはず")
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

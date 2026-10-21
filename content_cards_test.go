//go:build !test

package main

// カード・回復アイテムなど使用アイテムのコンテンツテスト。

import (
	"fmt"
	"strings"
	"testing"
)

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
	accessory := &Accessory{BaseItem: BaseItem{Name: "鼓舞の指輪", Type: "Accessory"}, Cursed: true, Abilities: []EquipmentAbilityID{AbilityPowerBoost}}
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

func TestFloorCardTemplates(t *testing.T) {
	cards := map[int]string{
		35: "モンスターハウスのカード",
		36: "敵倍速のカード",
		37: "地図忘却のカード",
		38: "拾得禁止のカード",
		39: "大部屋のカード",
		40: "罠のカード",
	}
	for id, wantName := range cards {
		card, ok := buildItemFromTemplate(id, 0, 0).(*Card)
		if !ok || card.Name != wantName {
			t.Fatalf("card %d = %#v, want %q", id, card, wantName)
		}
	}
}

func TestFloorCardRolls(t *testing.T) {
	minIntn := func(int) int { return 0 }
	maxIntn := func(n int) int { return n - 1 }
	if got := rollMonsterHouseEnemyCount(minIntn); got != 5 {
		t.Fatalf("minimum enemy count = %d, want 5", got)
	}
	if got := rollMonsterHouseEnemyCount(maxIntn); got != 8 {
		t.Fatalf("maximum enemy count = %d, want 8", got)
	}
	if got := rollMonsterHouseItemCount(minIntn); got != 2 {
		t.Fatalf("minimum item count = %d, want 2", got)
	}
	if got := rollMonsterHouseItemCount(maxIntn); got != 3 {
		t.Fatalf("maximum item count = %d, want 3", got)
	}
	if got := rollExtraTrapCount(minIntn); got != 5 {
		t.Fatalf("minimum trap count = %d, want 5", got)
	}
	if got := rollExtraTrapCount(maxIntn); got != 8 {
		t.Fatalf("maximum trap count = %d, want 8", got)
	}
}

func TestPickFreeCellsInRoom(t *testing.T) {
	room := Room{X: 0, Y: 0, Width: 5, Height: 5} // 内側は3x3の9マス
	isFree := func(x, y int) bool { return !(x == 1 && y == 1) }
	intn := func(n int) int { return 0 }

	cells := pickFreeCellsInRoom(room, isFree, 20, intn)
	if len(cells) != 8 {
		t.Fatalf("free cells = %d, want 8 (clamped to available)", len(cells))
	}
	for _, cell := range cells {
		if cell.X < 1 || cell.X > 3 || cell.Y < 1 || cell.Y > 3 {
			t.Fatalf("cell %v is outside the room interior", cell)
		}
		if cell.X == 1 && cell.Y == 1 {
			t.Fatal("occupied cell should not be picked")
		}
	}

	if got := pickFreeCellsInRoom(room, isFree, 3, intn); len(got) != 3 {
		t.Fatalf("picked cells = %d, want 3", len(got))
	}
}

// テスト用に外周が壁、内側が床のマップを作る

func TestMonsterHouseCard(t *testing.T) {
	card := buildItemFromTemplate(35, 0, 0)
	g := &Game{
		state: GameState{
			Map:    makeTestFloorMap(8, 8),
			Player: Player{Entity: Entity{X: 2, Y: 2}, Inventory: []Item{card}},
		},
		rooms: []Room{{ID: 0, X: 0, Y: 0, Width: 8, Height: 8}},
	}

	card.Use(g)
	if len(g.state.Player.Inventory) != 0 {
		t.Fatal("used card should be removed from inventory")
	}
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.state.Enemies) < 5 || len(g.state.Enemies) > 8 {
		t.Fatalf("spawned enemies = %d, want 5..8", len(g.state.Enemies))
	}
	for _, enemy := range g.state.Enemies {
		if enemy.X < 1 || enemy.X > 6 || enemy.Y < 1 || enemy.Y > 6 {
			t.Fatalf("enemy at (%d, %d) is outside the room interior", enemy.X, enemy.Y)
		}
		if enemy.X == 2 && enemy.Y == 2 {
			t.Fatal("enemy should not spawn on the player")
		}
		if !enemy.PlayerDiscovered || !enemy.ShowOnMiniMap || enemy.StatusAilments.Sleep != 0 {
			t.Fatalf("enemy %#v should be awake and discovered", enemy.StatusAilments)
		}
	}
	if len(g.state.Items) < 2 || len(g.state.Items) > 3 {
		t.Fatalf("spawned items = %d, want 2..3", len(g.state.Items))
	}
}

func TestMonsterHouseCardInCorridor(t *testing.T) {
	card := buildItemFromTemplate(35, 0, 0)
	g := &Game{
		state: GameState{
			Map:    makeTestFloorMap(8, 8),
			Player: Player{Entity: Entity{X: 2, Y: 2}, Inventory: []Item{card}},
		},
		rooms: []Room{}, // プレイヤーはどの部屋にもいない
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.state.Enemies) != 0 {
		t.Fatalf("enemies = %d, want 0 in corridor", len(g.state.Enemies))
	}
	if len(g.ActionQueue.Queue) < 2 || g.ActionQueue.Queue[1].Message != "しかし何も起こらなかった" {
		t.Fatal("通路で使用した場合は不発メッセージを表示するはず")
	}
}

func TestHasteAllEnemiesCard(t *testing.T) {
	card := buildItemFromTemplate(36, 0, 0)
	g := &Game{
		state: GameState{
			Player:  Player{Inventory: []Item{card}},
			Enemies: []Enemy{{Entity: Entity{X: 1, Y: 1}}, {Entity: Entity{X: 5, Y: 5}}},
		},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	for i, enemy := range g.state.Enemies {
		if enemy.StatusAilments.Haste != enemyHasteCardTurns {
			t.Fatalf("enemy %d haste = %d, want %d", i, enemy.StatusAilments.Haste, enemyHasteCardTurns)
		}
	}
}

func TestForgetFloorMapCard(t *testing.T) {
	card := buildItemFromTemplate(37, 0, 0)
	floorItem := buildItemFromTemplate(1, 1, 1)
	floorItem.SetPlayerDiscovered(true)
	floorItem.SetShowOnMiniMap(true)
	g := &Game{
		state: GameState{
			Map:     [][]Tile{{{Type: "floor", Visited: true}, {Type: "wall", Visited: true}}},
			Player:  Player{Inventory: []Item{card}},
			Enemies: []Enemy{{PlayerDiscovered: true, ShowOnMiniMap: true}},
			Items:   []Item{floorItem},
		},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if g.state.Map[0][0].Visited || g.state.Map[0][1].Visited {
		t.Fatal("all map tiles should be forgotten")
	}
	if g.state.Enemies[0].PlayerDiscovered || g.state.Enemies[0].ShowOnMiniMap {
		t.Fatal("enemy should be hidden from the minimap")
	}
	if floorItem.GetPlayerDiscovered() || floorItem.GetShowOnMiniMap() {
		t.Fatal("floor item should be hidden from the minimap")
	}
	if !g.miniMapDirty {
		t.Fatal("minimap should be marked dirty")
	}
}

func TestPickupBanCard(t *testing.T) {
	card := buildItemFromTemplate(38, 0, 0)
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card}}}}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if !g.pickupBanned {
		t.Fatal("pickupBanned should be set")
	}
}

func TestPickupBannedBlocksPickup(t *testing.T) {
	floorItem := buildItemFromTemplate(1, 2, 2)
	g := &Game{
		state: GameState{
			Player: Player{Entity: Entity{X: 2, Y: 2}},
			Items:  []Item{floorItem},
		},
		pickupBanned: true,
	}

	g.PickupItem()
	if len(g.ActionQueue.Queue) == 0 {
		t.Fatal("拾得禁止中はメッセージを表示するはず")
	}
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.state.Player.Inventory) != 0 {
		t.Fatal("拾得禁止中はアイテムを拾えないはず")
	}
	if len(g.state.Items) != 1 {
		t.Fatal("床のアイテムはそのまま残るはず")
	}
}

func TestBigRoomCard(t *testing.T) {
	card := buildItemFromTemplate(39, 0, 0)
	mapGrid := make([][]Tile, 6)
	for y := range mapGrid {
		mapGrid[y] = make([]Tile, 6)
		for x := range mapGrid[y] {
			mapGrid[y][x] = Tile{Type: "other", Blocked: true, BlockSight: true}
		}
	}
	mapGrid[3][3] = Tile{Type: "stairs"}
	g := &Game{
		state: GameState{
			Map:    mapGrid,
			Player: Player{Entity: Entity{X: 2, Y: 2}, Inventory: []Item{card}},
		},
		rooms: []Room{{ID: 1, X: 1, Y: 1, Width: 3, Height: 3}},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.rooms) != 1 || g.rooms[0].Width != 6 || g.rooms[0].Height != 6 {
		t.Fatalf("rooms = %#v, want one room covering the map", g.rooms)
	}
	if g.state.Map[0][0].Type != "wall" || !g.state.Map[0][0].Blocked {
		t.Fatal("map boundary should be wall")
	}
	if g.state.Map[2][4].Type != "floor" || g.state.Map[2][4].Blocked {
		t.Fatal("map interior should be floor")
	}
	if g.state.Map[3][3].Type != "stairs" {
		t.Fatal("stairs should be preserved")
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

func TestSelfTargetCardTemplates(t *testing.T) {
	cards := map[int]string{
		41: "自爆のカード",
		42: "口封じのカード",
		43: "パワーアップのカード",
		44: "完全回復のカード",
	}
	for id, wantName := range cards {
		card, ok := buildItemFromTemplate(id, 0, 0).(*Card)
		if !ok || card.Name != wantName {
			t.Fatalf("card %d = %#v, want %q", id, card, wantName)
		}
	}
}

func TestSplitEnemiesByExplosion(t *testing.T) {
	enemies := []Enemy{
		{Entity: Entity{X: 3, Y: 3}, Name: "隣接"},
		{Entity: Entity{X: 1, Y: 1}, Name: "斜め隣接"},
		{Entity: Entity{X: 5, Y: 2}, Name: "遠く"},
	}
	survivors, destroyed := splitEnemiesByExplosion(2, 2, enemies)
	if len(survivors) != 1 || survivors[0].Name != "遠く" {
		t.Fatalf("survivors = %#v, want only 遠く", survivors)
	}
	if len(destroyed) != 2 {
		t.Fatalf("destroyed = %#v, want 2 enemies", destroyed)
	}
}

func TestSelfDestructCard(t *testing.T) {
	card := buildItemFromTemplate(41, 0, 0)
	g := &Game{
		state: GameState{
			Player: Player{Entity: Entity{X: 2, Y: 2}, Health: 100, MaxHealth: 100, Inventory: []Item{card}},
			Enemies: []Enemy{
				{Entity: Entity{X: 3, Y: 2}, Health: 100, ExperiencePoints: 10},
				{Entity: Entity{X: 6, Y: 6}, Health: 100},
			},
		},
	}

	card.Use(g)
	for i := 0; i < len(g.ActionQueue.Queue); i++ {
		g.ActionQueue.Queue[i].Execute(g)
	}

	if g.state.Player.Health != 50 {
		t.Fatalf("player health = %d, want 50", g.state.Player.Health)
	}
	if len(g.state.Enemies) != 1 || g.state.Enemies[0].X != 6 {
		t.Fatalf("enemies = %#v, want only the distant enemy", g.state.Enemies)
	}
	if g.state.Player.ExperiencePoints != 0 {
		t.Fatal("消し飛んだ敵の経験値は入らないはず")
	}
	if len(g.state.Player.Inventory) != 0 {
		t.Fatal("card should be consumed")
	}
}

func TestMouthSealCard(t *testing.T) {
	card := buildItemFromTemplate(42, 0, 0)
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card}}}}

	card.Use(g)
	if msg := runQueuedActions(g); !strings.HasSuffix(msg, "海老さんは口を封じられた") {
		t.Fatalf("messages = %q", msg)
	}

	if got := g.state.Player.StatusAilments.MouthSeal; got != mouthSealCardTurns {
		t.Fatalf("mouth seal turns = %d, want %d", got, mouthSealCardTurns)
	}

	g.decrementStatusAilments()
	if got := g.state.Player.StatusAilments.MouthSeal; got != mouthSealCardTurns-1 {
		t.Fatalf("mouth seal turns after a turn = %d, want %d", got, mouthSealCardTurns-1)
	}
}

func TestIsMouthItem(t *testing.T) {
	if !isMouthItem(buildItemFromTemplate(1, 0, 0)) || !isMouthItem(buildItemFromTemplate(2, 0, 0)) || !isMouthItem(buildItemFromTemplate(30, 0, 0)) {
		t.Fatal("食料・薬・カードは口封じの対象のはず")
	}
	if isMouthItem(buildItemFromTemplate(9, 0, 0)) || isMouthItem(buildItemFromTemplate(20, 0, 0)) {
		t.Fatal("杖や武器は口封じの対象外のはず")
	}
}

func TestMouthSealBlocksItemUse(t *testing.T) {
	potion := buildItemFromTemplate(2, 0, 0)
	g := &Game{
		state: GameState{
			Player: Player{
				Inventory:      []Item{potion},
				StatusAilments: StatusAilments{MouthSeal: 5},
			},
		},
	}

	g.executeAction()

	if len(g.state.Player.Inventory) != 1 {
		t.Fatal("口封じ中はアイテムが消費されないはず")
	}
	if len(g.ActionQueue.Queue) != 1 || g.ActionQueue.Queue[0].Message != "口が封じられていて使えない" {
		t.Fatalf("queue = %#v, want blocked message", g.ActionQueue.Queue)
	}
	if g.isActioned {
		t.Fatal("口封じで使えなかった場合はターンを消費しないはず")
	}
}

func TestBoostPower(t *testing.T) {
	tests := []struct {
		power, maxPower    int
		wantPower, wantMax int
	}{
		{power: 5, maxPower: 8, wantPower: 8, wantMax: 8},
		{power: 0, maxPower: 8, wantPower: 3, wantMax: 8},
		{power: 8, maxPower: 8, wantPower: 9, wantMax: 9},
	}
	for _, tt := range tests {
		gotPower, gotMax := boostPower(tt.power, tt.maxPower)
		if gotPower != tt.wantPower || gotMax != tt.wantMax {
			t.Errorf("boostPower(%d, %d) = (%d, %d), want (%d, %d)", tt.power, tt.maxPower, gotPower, gotMax, tt.wantPower, tt.wantMax)
		}
	}
}

func TestPowerUpCard(t *testing.T) {
	card := buildItemFromTemplate(43, 0, 0)
	g := &Game{state: GameState{Player: Player{Power: 4, MaxPower: 8, Inventory: []Item{card}}}}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if g.state.Player.Power != 7 || g.state.Player.MaxPower != 8 {
		t.Fatalf("power = %d/%d, want 7/8", g.state.Player.Power, g.state.Player.MaxPower)
	}
}

func TestFullHealResult(t *testing.T) {
	if health, maxHealth := fullHealResult(40, 100); health != 100 || maxHealth != 100 {
		t.Fatalf("fullHealResult(40, 100) = (%d, %d), want (100, 100)", health, maxHealth)
	}
	if health, maxHealth := fullHealResult(100, 100); health != 105 || maxHealth != 105 {
		t.Fatalf("fullHealResult(100, 100) = (%d, %d), want (105, 105)", health, maxHealth)
	}
}

func TestFullHealCardDoesNotRestorePower(t *testing.T) {
	card := buildItemFromTemplate(44, 0, 0)
	g := &Game{
		state: GameState{
			Player: Player{
				Health: 40, MaxHealth: 100,
				Power: 3, MaxPower: 8,
				Inventory: []Item{card},
			},
		},
	}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if g.state.Player.Health != 100 || g.state.Player.MaxHealth != 100 {
		t.Fatalf("health = %d/%d, want 100/100", g.state.Player.Health, g.state.Player.MaxHealth)
	}
	if g.state.Player.Power != 3 || g.state.Player.MaxPower != 8 {
		t.Fatalf("power = %d/%d, want 3/8 (完全回復はパワーを戻さない)", g.state.Player.Power, g.state.Player.MaxPower)
	}
}

func TestRustProofCardTemplate(t *testing.T) {
	card, ok := buildItemFromTemplate(45, 0, 0).(*Card)
	if !ok || card.Name != "さび止めのカード" {
		t.Fatalf("card 45 = %#v, want さび止めのカード", card)
	}
}

func TestRustProofCard(t *testing.T) {
	card := buildItemFromTemplate(45, 0, 0)
	weapon := &Weapon{BaseItem: BaseItem{Name: "こん棒", Type: "Weapon"}, AttackPower: 2}
	armor := &Armor{BaseItem: BaseItem{Name: "木甲の盾", Type: "Armor"}, DefensePower: 2}
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card, weapon, armor}}}}
	g.state.Player.EquipItem(weapon)
	g.state.Player.EquipItem(armor)

	card.Use(g)
	if len(g.state.Player.Inventory) != 2 {
		t.Fatal("used card should be removed from inventory")
	}
	g.ActionQueue.Queue[0].Execute(g)

	if !weapon.RustProof || !armor.RustProof {
		t.Fatal("装備中の武器と盾が錆びなくなるはず")
	}
}

func TestRustProofCardWithoutEquipment(t *testing.T) {
	card := buildItemFromTemplate(45, 0, 0)
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card}}}}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.ActionQueue.Queue) < 2 || g.ActionQueue.Queue[1].Message != "しかし何も起こらなかった" {
		t.Fatal("装備なしで使用した場合は不発メッセージを表示するはず")
	}
}

func TestRustProofSurvivesSaveRoundTrip(t *testing.T) {
	weapon, ok := buildItemFromTemplate(20, 0, 0).(*Weapon)
	if !ok {
		t.Fatal("template 20 should be a weapon")
	}
	weapon.RustProof = true
	restored, err := savedToItem(itemToSaved(weapon))
	if err != nil {
		t.Fatalf("savedToItem error: %v", err)
	}
	if restoredWeapon, ok := restored.(*Weapon); !ok || !restoredWeapon.RustProof {
		t.Fatal("RustProof should survive a save round trip")
	}

	armor, ok := buildItemFromTemplate(23, 0, 0).(*Armor)
	if !ok {
		t.Fatal("template 23 should be an armor")
	}
	armor.RustProof = true
	restored, err = savedToItem(itemToSaved(armor))
	if err != nil {
		t.Fatalf("savedToItem error: %v", err)
	}
	if restoredArmor, ok := restored.(*Armor); !ok || !restoredArmor.RustProof {
		t.Fatal("RustProof should survive a save round trip")
	}
}

func TestFullHealCardAtFullHealth(t *testing.T) {
	card := buildItemFromTemplate(44, 0, 0)
	g := &Game{state: GameState{Player: Player{Health: 100, MaxHealth: 100, Inventory: []Item{card}}}}

	card.Use(g)
	g.ActionQueue.Queue[0].Execute(g)

	if g.state.Player.Health != 105 || g.state.Player.MaxHealth != 105 {
		t.Fatalf("health = %d/%d, want 105/105", g.state.Player.Health, g.state.Player.MaxHealth)
	}
}

// --- 特殊な使用方法を持つカード ---

func TestSpecialUseCardTemplates(t *testing.T) {
	cards := map[int]string{
		53: "白紙のカード",
		54: "ジェノサイドのカード",
		55: "聖域のカード",
		56: "全滅のカード",
	}
	for id, wantName := range cards {
		card, ok := buildItemFromTemplate(id, 0, 0).(*Card)
		if !ok || card.Name != wantName {
			t.Fatalf("card %d = %#v, want %q", id, card, wantName)
		}
	}
}

func TestBlankCardOptionIDs(t *testing.T) {
	ids := blankCardOptionIDs()
	if len(ids) == 0 {
		t.Fatal("白紙のカードへ書き込める候補がない")
	}
	prev := -1
	for _, id := range ids {
		if id <= prev {
			t.Fatalf("option IDs are not sorted: %v", ids)
		}
		prev = id
		template := itemTemplates[id]
		if template.ItemType != "Card" {
			t.Fatalf("option %d is not a card template", id)
		}
		if template.Name == blankCardName {
			t.Fatal("白紙のカード自身は書き込み候補に含めない")
		}
	}
}

func TestWriteCardEffect(t *testing.T) {
	blank := buildItemFromTemplate(53, 0, 0).(*Card)
	if !writeCardEffect(blank, 30) {
		t.Fatal("あかりのカードを書き込めるはず")
	}
	if blank.ID != 30 || blank.Name != "あかりのカード" {
		t.Fatalf("written card = ID %d, name %q", blank.ID, blank.Name)
	}
	if _, ok := blank.UseActions["UseCard"]; !ok {
		t.Fatal("書き込んだカードは使用できるはず")
	}

	if writeCardEffect(blank, 4) {
		t.Fatal("武器テンプレートは書き込めないはず")
	}
	if writeCardEffect(blank, 53) {
		t.Fatal("白紙のカード自身は書き込めないはず")
	}
}

func TestBlankCardOpensSelectionWithoutConsuming(t *testing.T) {
	card := buildItemFromTemplate(53, 0, 0)
	g := &Game{state: GameState{Player: Player{Inventory: []Item{card}}}}

	card.Use(g)

	if !g.showBlankCardMenu || g.blankCardTarget == nil {
		t.Fatal("白紙のカードを使うと選択ウィンドウが開くはず")
	}
	if len(g.state.Player.Inventory) != 1 {
		t.Fatal("書き込むまで白紙のカードは消費されないはず")
	}
}

func TestBlankCardScrollTop(t *testing.T) {
	if got := blankCardScrollTop(0, 5, 10); got != 0 {
		t.Fatalf("scroll top = %d, want 0 when everything fits", got)
	}
	if got := blankCardScrollTop(0, 20, 10); got != 0 {
		t.Fatalf("scroll top = %d, want 0 at list head", got)
	}
	if got := blankCardScrollTop(15, 20, 10); got != 6 {
		t.Fatalf("scroll top = %d, want 6 to keep cursor visible", got)
	}
	if got := blankCardScrollTop(19, 20, 10); got != 10 {
		t.Fatalf("scroll top = %d, want 10 at list tail", got)
	}
}

func TestThrowOnlyCardHintsDoNotConsume(t *testing.T) {
	for _, id := range []int{54, 55} {
		card := buildItemFromTemplate(id, 0, 0)
		g := &Game{state: GameState{Player: Player{Inventory: []Item{card}}}}
		card.Use(g)
		if len(g.state.Player.Inventory) != 1 {
			t.Fatalf("card %d should not be consumed by reading", id)
		}
		if len(g.ActionQueue.Queue) != 1 {
			t.Fatalf("card %d should enqueue a hint message", id)
		}
	}
}

func TestMonsterFamilyIDs(t *testing.T) {
	for _, id := range []int{0, 18, 45} {
		if got := monsterFamilyIDs(id); fmt.Sprint(got) != "[0 18 45]" {
			t.Fatalf("family of %d = %v, want [0 18 45]", id, got)
		}
	}
	// 化け貝は道具と階段の擬態で別系統
	if got := monsterFamilyIDs(16); fmt.Sprint(got) != "[16 34 61]" {
		t.Fatalf("family of 16 = %v, want [16 34 61]", got)
	}
}

func TestGenocideCardSealsFamily(t *testing.T) {
	card := buildItemFromTemplate(54, 0, 0)
	base := CreateEnemyByID(0, 3, 2)
	upper := CreateEnemyByID(18, 5, 5)
	other := CreateEnemyByID(2, 6, 6)
	g := &Game{state: GameState{
		Player:  Player{Entity: Entity{X: 2, Y: 2}},
		Enemies: []Enemy{base, upper, other},
	}}

	g.onTargetHit(&g.state.Enemies[0], card, 0)
	for i := 0; i < len(g.ActionQueue.Queue); i++ {
		g.ActionQueue.Queue[i].Execute(g)
	}

	if len(g.state.Enemies) != 1 || g.state.Enemies[0].ID != 2 {
		t.Fatalf("remaining enemies = %#v, want only マムル", g.state.Enemies)
	}
	if !g.genocidedMonsterIDs[0] || !g.genocidedMonsterIDs[18] {
		t.Fatalf("genocided IDs = %v, want 0 and 18", g.genocidedMonsterIDs)
	}
	if g.state.Player.ExperiencePoints != 0 {
		t.Fatal("封じた敵の経験値は入らないはず")
	}
}

func TestSelectMonsterForFloorSkipsGenocided(t *testing.T) {
	// 1階のテーブルは {0, 2, 1}。エビ系統を封じると先頭候補はマムルになる。
	banned := map[int]bool{0: true, 18: true}
	if got := selectMonsterForFloor(1, banned, func(int) int { return 0 }); got != 2 {
		t.Fatalf("selected monster = %d, want 2", got)
	}

	allBanned := map[int]bool{0: true, 1: true, 2: true}
	if got := selectMonsterForFloor(1, allBanned, func(int) int { return 0 }); got != -1 {
		t.Fatalf("selected monster = %d, want -1 when all are banned", got)
	}
}

func TestRemoveGenocidedEnemies(t *testing.T) {
	g := &Game{
		state: GameState{
			Enemies: []Enemy{CreateEnemyByID(0, 1, 1), CreateEnemyByID(2, 2, 2)},
		},
		genocidedMonsterIDs: map[int]bool{0: true},
	}
	g.removeGenocidedEnemies()
	if len(g.state.Enemies) != 1 || g.state.Enemies[0].ID != 2 {
		t.Fatalf("enemies after removal = %#v, want only マムル", g.state.Enemies)
	}
}

func TestSplitEnemiesByRoomWideEffect(t *testing.T) {
	rooms := []Room{{ID: 1, X: 0, Y: 0, Width: 5, Height: 5}}
	enemies := []Enemy{
		{Entity: Entity{X: 3, Y: 3}, Name: "同室"},
		{Entity: Entity{X: 6, Y: 6}, Name: "室外"},
	}
	survivors, destroyed := splitEnemiesByRoomWideEffect(2, 2, enemies, rooms)
	if len(survivors) != 1 || survivors[0].Name != "室外" {
		t.Fatalf("survivors = %#v, want only 室外", survivors)
	}
	if len(destroyed) != 1 || destroyed[0] != "同室" {
		t.Fatalf("destroyed = %#v, want only 同室", destroyed)
	}
}

func TestAnnihilationCard(t *testing.T) {
	card := buildItemFromTemplate(56, 0, 0)
	g := &Game{
		state: GameState{
			Player: Player{Entity: Entity{X: 2, Y: 2}, Inventory: []Item{card}},
			Enemies: []Enemy{
				{Entity: Entity{X: 3, Y: 3}, Health: 100, ExperiencePoints: 10},
				{Entity: Entity{X: 6, Y: 6}, Health: 100},
			},
		},
		rooms: []Room{{ID: 1, X: 0, Y: 0, Width: 5, Height: 5}},
	}

	card.Use(g)
	if len(g.state.Player.Inventory) != 0 {
		t.Fatal("used card should be removed from inventory")
	}
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.state.Enemies) != 1 || g.state.Enemies[0].X != 6 {
		t.Fatalf("enemies = %#v, want only the outside enemy", g.state.Enemies)
	}
	if g.state.Player.ExperiencePoints != 0 {
		t.Fatal("消し去った敵の経験値は入らないはず")
	}
}

func TestSanctuaryBlocksEnemyAttack(t *testing.T) {
	sanctuary := buildItemFromTemplate(55, 2, 2)
	melee := CreateEnemyByID(0, 3, 2)
	g := &Game{state: GameState{
		Map:     makeTestFloorMap(10, 10),
		Player:  Player{Entity: Entity{X: 2, Y: 2}, Health: 100},
		Enemies: []Enemy{melee},
		Items:   []Item{sanctuary},
	}}

	g.AttackFromEnemy(0)
	g.AttackFromEnemyBlind(0)
	if len(g.ActionQueue.Queue) != 0 {
		t.Fatal("聖域の上にいるプレイヤーは攻撃されないはず")
	}

	ranged := CreateEnemyByID(4, 6, 2)
	ranged.PlayerDiscovered = true
	g.state.Enemies = append(g.state.Enemies, ranged)
	if g.tryEnemyRangedAttack(1) {
		t.Fatal("聖域の上にいるプレイヤーは遠距離攻撃も受けないはず")
	}
}

func TestSanctuaryBlocksEnemyMovement(t *testing.T) {
	sanctuary := buildItemFromTemplate(55, 3, 2)
	enemy := CreateEnemyByID(0, 2, 2)
	enemy.PlayerDiscovered = true
	g := &Game{state: GameState{
		Map:     makeTestFloorMap(10, 10),
		Player:  Player{Entity: Entity{X: 5, Y: 2}},
		Enemies: []Enemy{enemy},
		Items:   []Item{sanctuary},
	}}

	if isPositionFree(g, 3, 2, 0) {
		t.Fatal("聖域のカードの上は敵の移動先にならないはず")
	}
	g.moveEnemyTowardsPlayer(0)
	if g.state.Enemies[0].X == 3 && g.state.Enemies[0].Y == 2 {
		t.Fatal("敵は聖域のカードの上へ移動できないはず")
	}
}

func TestStuckSanctuaryCannotBePickedUp(t *testing.T) {
	sanctuary := buildItemFromTemplate(55, 2, 2).(*Card)
	sanctuary.Stuck = true
	g := &Game{state: GameState{
		Player: Player{Entity: Entity{X: 2, Y: 2}},
		Items:  []Item{sanctuary},
	}}

	g.PickupItem()
	if len(g.ActionQueue.Queue) != 1 {
		t.Fatalf("queued actions = %d, want 1", len(g.ActionQueue.Queue))
	}
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.state.Player.Inventory) != 0 || len(g.state.Items) != 1 {
		t.Fatal("貼りついた聖域のカードは拾えないはず")
	}
}

func TestUnstuckSanctuaryCanBePickedUp(t *testing.T) {
	sanctuary := buildItemFromTemplate(55, 2, 2)
	g := &Game{state: GameState{
		Player: Player{Entity: Entity{X: 2, Y: 2}},
		Items:  []Item{sanctuary},
	}}

	g.PickupItem()
	g.ActionQueue.Queue[0].Execute(g)

	if len(g.state.Player.Inventory) != 1 || len(g.state.Items) != 0 {
		t.Fatal("貼りついていない聖域のカードは拾えるはず")
	}
}

func TestSpecialCardSaveRoundTrip(t *testing.T) {
	// 貼りついた聖域のカード
	sanctuary := buildItemFromTemplate(55, 1, 1).(*Card)
	sanctuary.Stuck = true
	restored, err := savedToItem(itemToSaved(sanctuary))
	if err != nil {
		t.Fatalf("restore sanctuary card: %v", err)
	}
	if restoredCard, ok := restored.(*Card); !ok || !restoredCard.Stuck {
		t.Fatal("Stuck should survive a save round trip")
	}

	// 書き込み済みの白紙のカード
	blank := buildItemFromTemplate(53, 0, 0).(*Card)
	writeCardEffect(blank, 30)
	restored, err = savedToItem(itemToSaved(blank))
	if err != nil {
		t.Fatalf("restore written blank card: %v", err)
	}
	if restoredCard, ok := restored.(*Card); !ok || restoredCard.Name != "あかりのカード" {
		t.Fatalf("restored written card = %#v, want あかりのカード", restored)
	}
}

func TestGenocideSurvivesSaveRoundTrip(t *testing.T) {
	g := &Game{
		state: GameState{
			Map:    makeTestFloorMap(4, 4),
			Player: Player{Entity: Entity{X: 1, Y: 1}},
		},
		genocidedMonsterIDs: map[int]bool{0: true, 18: true},
	}

	save := g.buildSaveData()
	if len(save.GenocidedIDs) != 2 || save.GenocidedIDs[0] != 0 || save.GenocidedIDs[1] != 18 {
		t.Fatalf("saved genocided IDs = %v, want [0 18]", save.GenocidedIDs)
	}

	restored := &Game{}
	if err := restored.applySaveData(save); err != nil {
		t.Fatalf("applySaveData failed: %v", err)
	}
	if !restored.genocidedMonsterIDs[0] || !restored.genocidedMonsterIDs[18] {
		t.Fatalf("restored genocided IDs = %v, want 0 and 18", restored.genocidedMonsterIDs)
	}
}

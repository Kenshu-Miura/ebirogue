//go:build !test

package main

// 武器・盾・矢など装備品のコンテンツテスト。

import (
	"strings"
	"testing"
)

func TestInventoryShootUsesShootingState(t *testing.T) {
	arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
	arrow.ShotCount = 3
	g := &Game{
		state: GameState{
			Map: makeTestFloorMap(20, 20),
			Player: Player{
				Entity:    Entity{X: 10, Y: 10},
				Direction: Right,
				Inventory: []Item{arrow},
			},
		},
		selectedActionIndex: 1,
	}

	g.executeAction()

	if !g.dPressed {
		t.Fatal("inventory shoot should enable shooting state for rotation and damage")
	}
	if arrow.ShotCount != 2 {
		t.Fatalf("remaining arrows = %d, want 2", arrow.ShotCount)
	}
	if len(g.ActionQueue.Queue) != 1 || !strings.HasSuffix(g.ActionQueue.Queue[0].Message, "を撃った") {
		t.Fatalf("queued action = %#v, want shooting message", g.ActionQueue.Queue)
	}

	g.ActionQueue.Queue[0].Execute(g)
	shotArrow, ok := g.ThrownItem.Item.(*Arrow)
	if !ok {
		t.Fatalf("thrown item = %T, want *Arrow", g.ThrownItem.Item)
	}
	if shotArrow.ShotCount != 1 {
		t.Fatalf("shot arrow count = %d, want 1", shotArrow.ShotCount)
	}
	if g.ThrownItem.DX != 1 || g.ThrownItem.DY != 0 {
		t.Fatalf("shot direction = (%d, %d), want (1, 0)", g.ThrownItem.DX, g.ThrownItem.DY)
	}
}

func TestInventoryShootLastEquippedArrowUnequips(t *testing.T) {
	arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
	arrow.ShotCount = 1
	g := &Game{
		state: GameState{
			Map: makeTestFloorMap(20, 20),
			Player: Player{
				Entity:        Entity{X: 10, Y: 10},
				Direction:     Down,
				Inventory:     []Item{arrow},
				EquippedArrow: arrow,
			},
		},
		selectedActionIndex: 1,
	}

	g.executeAction()

	if len(g.state.Player.Inventory) != 0 {
		t.Fatalf("inventory length = %d, want 0 after last arrow", len(g.state.Player.Inventory))
	}
	if g.state.Player.EquippedArrow != nil {
		t.Fatal("last equipped arrow should be unequipped")
	}
	if !g.dPressed {
		t.Fatal("last arrow should still be treated as a shot")
	}
}

func TestAddedEquipmentTemplates(t *testing.T) {
	weapons := map[int]int{20: 2, 21: 4, 22: 6}
	for id, wantPower := range weapons {
		weapon, ok := buildItemFromTemplate(id, 0, 0).(*Weapon)
		if !ok || weapon.AttackPower != wantPower {
			t.Fatalf("weapon %d = %#v, want attack power %d", id, weapon, wantPower)
		}
	}

	slayerWeapons := map[int]EquipmentAbilityID{
		49: AbilityDragonSlayer,
		50: AbilityGhostSlayer,
		51: AbilityOneEyeSlayer,
		52: AbilityDrainerSlayer,
	}
	for id, wantAbility := range slayerWeapons {
		weapon, ok := buildItemFromTemplate(id, 0, 0).(*Weapon)
		if !ok || !hasEquipmentAbility(weapon.Abilities, wantAbility) {
			t.Fatalf("slayer weapon %d = %#v, want ability %q", id, weapon, wantAbility)
		}
		restored, err := savedToItem(itemToSaved(weapon))
		if err != nil {
			t.Fatalf("restore slayer weapon %d: %v", id, err)
		}
		if !hasEquipmentAbility(restored.(*Weapon).Abilities, wantAbility) {
			t.Fatalf("restored slayer weapon %d lost ability %q", id, wantAbility)
		}
	}

	specialAttackWeapons := map[int]struct {
		power   int
		ability EquipmentAbilityID
	}{
		57: {power: 4, ability: AbilityThreeWayAttack},
		58: {power: 3, ability: AbilitySureHit},
		59: {power: 2, ability: AbilityDigWall},
		60: {power: 12, ability: AbilityDisposable},
	}
	for id, want := range specialAttackWeapons {
		weapon, ok := buildItemFromTemplate(id, 0, 0).(*Weapon)
		if !ok || weapon.AttackPower != want.power || !hasEquipmentAbility(weapon.Abilities, want.ability) {
			t.Fatalf("special attack weapon %d = %#v, want power %d and ability %q", id, weapon, want.power, want.ability)
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

	specialArmors := map[int]struct {
		power   int
		ability EquipmentAbilityID
	}{
		61: {power: 4, ability: AbilityExplosionResistance},
		62: {power: 4, ability: AbilityFireResistance},
		63: {power: 4, ability: AbilityMagicResistance},
		64: {power: 3, ability: AbilityTheftResistance},
		65: {power: 3, ability: AbilityStatusResistance},
		66: {power: 2, ability: AbilityEvasion},
		67: {power: 3, ability: AbilityReflection},
		68: {power: 5, ability: AbilityCounter},
		69: {power: 12, ability: AbilitySatietyHunger},
		70: {power: 15, ability: AbilityDisposableArmor},
	}
	for id, want := range specialArmors {
		armor, ok := buildItemFromTemplate(id, 0, 0).(*Armor)
		if !ok || armor.DefensePower != want.power || !hasEquipmentAbility(armor.Abilities, want.ability) {
			t.Fatalf("special armor %d = %#v, want power %d and ability %q", id, armor, want.power, want.ability)
		}
		restored, err := savedToItem(itemToSaved(armor))
		if err != nil {
			t.Fatalf("restore special armor %d: %v", id, err)
		}
		if !hasEquipmentAbility(restored.(*Armor).Abilities, want.ability) {
			t.Fatalf("restored special armor %d lost ability %q", id, want.ability)
		}
	}
}

func TestAddedContentDefinitions(t *testing.T) {
	if MonsterDefinitions[2].Name != "マムル" || MonsterDefinitions[3].Name != "くねくねハニー" {
		t.Fatal("added monster definitions are missing")
	}
	rangedEnemies := map[int]struct {
		name string
		kind RangedAttackKind
	}{
		4:  {name: "ハリセンボウ", kind: RangedAttackArrow},
		5:  {name: "イシガニ", kind: RangedAttackRock},
		6:  {name: "バクダンウニ", kind: RangedAttackExplosion},
		38: {name: "火吹きイカ", kind: RangedAttackFire},
		39: {name: "マホウクラゲ", kind: RangedAttackMagic},
	}
	for id, want := range rangedEnemies {
		definition := MonsterDefinitions[id]
		if definition.Name != want.name || definition.RangedAttack.Kind != want.kind {
			t.Fatalf("ranged monster %d = %#v, want %q with kind %d", id, definition, want.name, want.kind)
		}
	}
	inventoryEnemies := map[int]string{
		7:  "コソドロヤドカリ",
		8:  "にぎりエビ",
		9:  "ノロイガニ",
		10: "あやつりクラゲ",
	}
	for id, wantName := range inventoryEnemies {
		definition := MonsterDefinitions[id]
		if definition.Name != wantName || definition.SpecialAttack == nil || definition.SpecialAttackProbability <= 0 {
			t.Fatalf("inventory monster %d = %#v, want %q with a special attack", id, definition, wantName)
		}
	}

	wantTraps := []string{"睡眠ガスの罠", "毒矢の罠", "鈍足の罠", "地雷", "サビの罠"}
	for id, wantName := range wantTraps {
		if got := createMapTrapByID(id, 1, 2); got.Name != wantName {
			t.Fatalf("trap %d = %q, want %q", id, got.Name, wantName)
		}
	}
}

func TestEnemyTraitsForSlayerWeapons(t *testing.T) {
	wantTraits := map[int][]EnemyTrait{
		1:  {EnemyTraitDrainer},
		3:  {EnemyTraitOneEye, EnemyTraitDrainer},
		11: {EnemyTraitGhost},
		36: {EnemyTraitDragon},
		37: {EnemyTraitDragon},
	}
	for id, traits := range wantTraits {
		definition, ok := MonsterDefinitions[id]
		if !ok {
			t.Fatalf("monster %d is missing", id)
		}
		for _, trait := range traits {
			if !hasEnemyTrait(definition.Traits, trait) {
				t.Fatalf("monster %d (%s) is missing trait %q", id, definition.Name, trait)
			}
		}
	}

	enemy := CreateEnemyByID(36, 2, 3)
	restored := savedToEnemy(enemyToSaved(&enemy))
	if !hasEnemyTrait(restored.Traits, EnemyTraitDragon) {
		t.Fatal("restored sea dragon lost its dragon trait")
	}

	foundDragon := map[int]bool{36: false, 37: false}
	for floor := 1; floor <= deepestSpawnFloor; floor++ {
		for _, entry := range FloorSpawnTables[floor] {
			if _, ok := foundDragon[entry.MonsterID]; ok {
				foundDragon[entry.MonsterID] = true
			}
		}
	}
	for id, found := range foundDragon {
		if !found {
			t.Fatalf("dragon monster %d is missing from floor spawn tables", id)
		}
	}
}

func TestSlayerWeaponAffectsPlayerNormalAttack(t *testing.T) {
	// 会心判定を必ず外し、乱数係数を最大(9/8)へ固定して決定的にする
	hooks := randHooks{damage: func(n int) int { return n - 1 }, attackHit: func(int) int { return 0 }}

	weapon := buildItemFromTemplate(49, 0, 0).(*Weapon)
	enemy := CreateEnemyByID(36, 2, 1)
	g := &Game{randHooks: hooks,
		state: GameState{
			Map: makeTestFloorMap(5, 5),
			Player: Player{
				Entity:         Entity{X: 1, Y: 1},
				AttackPower:    20,
				Power:          8,
				Level:          1,
				EquippedWeapon: weapon,
			},
			Enemies: []Enemy{enemy},
		},
	}

	if !g.CheckForEnemies(1, 0) || len(g.ActionQueue.Queue) != 1 {
		t.Fatal("normal attack was not queued")
	}
	if !strings.HasPrefix(g.ActionQueue.Queue[0].Message, "特効！") {
		t.Fatalf("attack message = %q, want slayer indication", g.ActionQueue.Queue[0].Message)
	}
	before := g.state.Enemies[0].Health
	g.ActionQueue.Queue[0].Execute(g)
	damage := before - g.state.Enemies[0].Health
	// 総攻撃力29 × (15/16)^9 × 9/8 × 特効1.5 ≒ 27.38 → 27
	if damage != 27 {
		t.Fatalf("slayer attack damage = %d, want 27", damage)
	}
}

func TestThreeWayWeaponAttacksFrontArc(t *testing.T) {
	hooks := randHooks{attackHit: func(int) int { return 0 }}

	weapon := buildItemFromTemplate(57, 0, 0).(*Weapon)
	enemies := []Enemy{
		CreateEnemyByID(0, 2, 1),
		CreateEnemyByID(0, 1, 1),
		CreateEnemyByID(0, 3, 1),
		CreateEnemyByID(0, 3, 2),
	}
	for i := range enemies {
		enemies[i].Health = 1000
		enemies[i].MaxHealth = 1000
	}
	g := &Game{randHooks: hooks, state: GameState{
		Map: makeTestFloorMap(5, 5),
		Player: Player{
			Entity:         Entity{X: 2, Y: 2},
			Direction:      Up,
			AttackPower:    10,
			Power:          8,
			Level:          1,
			EquippedWeapon: weapon,
		},
		Enemies: enemies,
	}}

	g.CheckForEnemies(0, -1)
	if len(g.ActionQueue.Queue) != 3 {
		t.Fatalf("three-way actions = %d, want 3", len(g.ActionQueue.Queue))
	}
	for len(g.ActionQueue.Queue) > 0 {
		action := g.ActionQueue.Queue[0]
		g.ActionQueue.Queue = g.ActionQueue.Queue[1:]
		action.Execute(g)
	}
	for i := 0; i < 3; i++ {
		if g.state.Enemies[i].Health >= 1000 {
			t.Fatalf("front-arc enemy %d was not damaged", i)
		}
	}
	if g.state.Enemies[3].Health != 1000 {
		t.Fatal("enemy outside front arc should not be damaged")
	}
}

func TestSureHitWeaponAndNormalAttackMiss(t *testing.T) {
	hooks := randHooks{attackHit: func(int) int { return 99 }}

	newGame := func(weapon *Weapon) *Game {
		enemy := CreateEnemyByID(0, 2, 1)
		return &Game{randHooks: hooks, state: GameState{
			Map:     makeTestFloorMap(5, 5),
			Player:  Player{Entity: Entity{X: 1, Y: 1}, AttackPower: 10, Power: 8, Level: 1, EquippedWeapon: weapon},
			Enemies: []Enemy{enemy},
		}}
	}

	normal := newGame(buildItemFromTemplate(20, 0, 0).(*Weapon))
	normal.CheckForEnemies(1, 0)
	if !strings.Contains(normal.ActionQueue.Queue[0].Message, "外れた") {
		t.Fatalf("normal attack message = %q, want miss", normal.ActionQueue.Queue[0].Message)
	}

	sureHit := newGame(buildItemFromTemplate(58, 0, 0).(*Weapon))
	before := sureHit.state.Enemies[0].Health
	sureHit.CheckForEnemies(1, 0)
	if strings.Contains(sureHit.ActionQueue.Queue[0].Message, "外れた") {
		t.Fatalf("sure-hit attack message = %q", sureHit.ActionQueue.Queue[0].Message)
	}
	sureHit.ActionQueue.Queue[0].Execute(sureHit)
	if sureHit.state.Enemies[0].Health >= before {
		t.Fatal("sure-hit weapon should damage the enemy even at roll 99")
	}
}

func TestPickaxeWeaponDigsWall(t *testing.T) {
	weapon := buildItemFromTemplate(59, 0, 0).(*Weapon)
	gameMap := makeTestFloorMap(5, 5)
	gameMap[1][2] = Tile{Type: "wall", Blocked: true, BlockSight: true}
	g := &Game{state: GameState{
		Map:    gameMap,
		Player: Player{Entity: Entity{X: 2, Y: 2}, Direction: Up, EquippedWeapon: weapon},
	}}

	g.CheckForEnemies(0, -1)
	if len(g.ActionQueue.Queue) != 1 || g.ActionQueue.Queue[0].Message != "壁を掘った。" {
		t.Fatalf("dig action = %#v", g.ActionQueue.Queue)
	}
	g.ActionQueue.Queue[0].Execute(g)
	tile := g.state.Map[1][2]
	if tile.Type != "corridor" || tile.Blocked || tile.BlockSight || !tile.Visited {
		t.Fatalf("dug tile = %#v", tile)
	}
}

func TestDisposableWeaponWeakensAndPersists(t *testing.T) {
	weapon := buildItemFromTemplate(60, 0, 0).(*Weapon)
	weapon.Sharpness = 0
	g := &Game{state: GameState{
		Map:    makeTestFloorMap(5, 5),
		Player: Player{Entity: Entity{X: 2, Y: 2}, Direction: Up, AttackPower: 15, EquippedWeapon: weapon},
	}}

	g.CheckForEnemies(0, -1)
	if len(g.ActionQueue.Queue) != 2 {
		t.Fatalf("disposable attack actions = %d, want swing and wear", len(g.ActionQueue.Queue))
	}
	g.ActionQueue.Queue[1].Execute(g)
	if weapon.AttackPower != 11 || g.state.Player.AttackPower != 14 {
		t.Fatalf("after wear: weapon power %d, player attack %d", weapon.AttackPower, g.state.Player.AttackPower)
	}

	restored, err := savedToItem(itemToSaved(weapon))
	if err != nil {
		t.Fatalf("restore disposable weapon: %v", err)
	}
	if restored.(*Weapon).AttackPower != 11 {
		t.Fatalf("restored disposable power = %d, want 11", restored.(*Weapon).AttackPower)
	}
}

func TestDefensiveShieldCombatEffects(t *testing.T) {
	maxDamage := func(n int) int { return n - 1 }
	evade := randHooks{damage: maxDamage, shieldDefense: func(int) int { return 0 }}
	noEvade := randHooks{damage: maxDamage, shieldDefense: func(int) int { return 99 }}

	t.Run("evasion", func(t *testing.T) {
		armor := buildItemFromTemplate(66, 0, 0).(*Armor)
		g := &Game{randHooks: evade, state: GameState{
			Player:  Player{Entity: Entity{X: 2, Y: 2}, Health: 100, DefensePower: 3, EquippedArmor: armor},
			Enemies: []Enemy{CreateEnemyByID(0, 3, 2)},
		}}
		g.enqueueEnemyNormalAttack(0)
		g.ActionQueue.Queue[0].Execute(g)
		if g.state.Player.Health != 100 || !strings.Contains(g.ActionQueue.Queue[0].Message, "かわした") {
			t.Fatalf("evasion result: HP %d, message %q", g.state.Player.Health, g.ActionQueue.Queue[0].Message)
		}
	})

	t.Run("counter", func(t *testing.T) {
		armor := buildItemFromTemplate(68, 0, 0).(*Armor)
		enemy := CreateEnemyByID(0, 3, 2)
		enemy.Health = 100
		g := &Game{randHooks: noEvade, state: GameState{
			Player:  Player{Entity: Entity{X: 2, Y: 2}, Health: 100, DefensePower: 3, EquippedArmor: armor},
			Enemies: []Enemy{enemy},
		}}
		g.enqueueEnemyNormalAttack(0)
		g.ActionQueue.Queue[0].Execute(g)
		if g.state.Player.Health >= 100 || g.state.Enemies[0].Health >= 100 {
			t.Fatalf("counter result: player HP %d, enemy HP %d", g.state.Player.Health, g.state.Enemies[0].Health)
		}
	})

	t.Run("ranged reflection", func(t *testing.T) {
		armor := buildItemFromTemplate(67, 0, 0).(*Armor)
		enemy := CreateEnemyByID(4, 1, 4)
		enemy.PlayerDiscovered = true
		enemy.Health = 100
		g := &Game{randHooks: noEvade, state: GameState{
			Map:     makeTestFloorMap(10, 10),
			Player:  Player{Entity: Entity{X: 7, Y: 4}, Health: 100, DefensePower: 3, EquippedArmor: armor},
			Enemies: []Enemy{enemy},
		}}
		if !g.tryEnemyRangedAttack(0) {
			t.Fatal("arrow attack should be available")
		}
		g.ActionQueue.Queue[0].Execute(g)
		if g.state.Player.Health != 100 || g.state.Enemies[0].Health >= 100 {
			t.Fatalf("reflection result: player HP %d, enemy HP %d", g.state.Player.Health, g.state.Enemies[0].Health)
		}
	})

	t.Run("elemental resistance", func(t *testing.T) {
		for _, tt := range []struct {
			enemyID int
			armorID int
		}{
			{enemyID: 6, armorID: 61},
			{enemyID: 38, armorID: 62},
			{enemyID: 39, armorID: 63},
		} {
			armor := buildItemFromTemplate(tt.armorID, 0, 0).(*Armor)
			enemy := CreateEnemyByID(tt.enemyID, 1, 4)
			enemy.PlayerDiscovered = true
			g := &Game{randHooks: noEvade, state: GameState{
				Map:     makeTestFloorMap(10, 10),
				Player:  Player{Entity: Entity{X: 5, Y: 4}, Health: 100, EquippedArmor: armor},
				Enemies: []Enemy{enemy},
			}}
			if !g.tryEnemyRangedAttack(0) {
				t.Fatalf("enemy %d attack should be available", tt.enemyID)
			}
			if !strings.Contains(g.ActionQueue.Queue[0].Message, "威力を抑え") {
				t.Fatalf("enemy %d resistance message = %q", tt.enemyID, g.ActionQueue.Queue[0].Message)
			}
		}
	})
}

func TestInventoryAndStatusResistanceShields(t *testing.T) {
	t.Run("theft", func(t *testing.T) {
		item := buildItemFromTemplate(1, 0, 0)
		armor := buildItemFromTemplate(64, 0, 0).(*Armor)
		enemy := CreateEnemyByID(7, 2, 2)
		g := &Game{state: GameState{
			Player:  Player{Inventory: []Item{item, armor}, EquippedArmor: armor},
			Enemies: []Enemy{enemy},
		}}
		enqueueStealAttack(&g.state.Enemies[0], g, func(int) int { return 0 })
		g.ActionQueue.Queue[0].Execute(g)
		if len(g.state.Player.Inventory) != 2 || g.state.Enemies[0].HeldItem != nil {
			t.Fatal("theft resistance shield should preserve inventory")
		}
	})

	t.Run("status", func(t *testing.T) {
		armor := buildItemFromTemplate(65, 0, 0).(*Armor)
		enemy := CreateEnemyByID(3, 2, 2)
		g := &Game{randHooks: randHooks{statusResistance: func(int) int { return 0 }}, state: GameState{Player: Player{EquippedArmor: armor}, Enemies: []Enemy{enemy}}}
		g.state.Enemies[0].SpecialAttack(&g.state.Enemies[0], g)
		g.ActionQueue.Queue[0].Execute(g)
		if g.state.Player.StatusAilments.Slow != 0 {
			t.Fatal("status resistance shield should block a successful resistance roll")
		}
	})
}

func TestDisposableShieldWeakensAndPersists(t *testing.T) {
	hooks := randHooks{damage: func(n int) int { return n - 1 }, shieldDefense: func(int) int { return 99 }}

	armor := buildItemFromTemplate(70, 0, 0).(*Armor)
	armor.Sharpness = 0
	g := &Game{randHooks: hooks, state: GameState{
		Player:  Player{Entity: Entity{X: 2, Y: 2}, Health: 100, DefensePower: 18, EquippedArmor: armor},
		Enemies: []Enemy{CreateEnemyByID(0, 3, 2)},
	}}
	g.enqueueEnemyNormalAttack(0)
	g.ActionQueue.Queue[0].Execute(g)
	if armor.Sharpness != -1 || g.state.Player.DefensePower != 17 {
		t.Fatalf("after wear: sharpness %d, player defense %d", armor.Sharpness, g.state.Player.DefensePower)
	}
	restored, err := savedToItem(itemToSaved(armor))
	if err != nil {
		t.Fatalf("restore disposable shield: %v", err)
	}
	if restored.(*Armor).Sharpness != -1 {
		t.Fatalf("restored disposable sharpness = %d, want -1", restored.(*Armor).Sharpness)
	}
}

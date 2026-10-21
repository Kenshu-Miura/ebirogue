//go:build !test

package main

import (
	"strings"
	"testing"
)

func TestWave5ATablesAndSave(t *testing.T) {
	names := []string{"ウミダンゴ", "オオウミダンゴ", "ダイオウウミダンゴ", "カエルアンコウ", "オオカエルアンコウ", "ヌシカエルアンコウ", "ハジキガニ", "ケンゴウガニ", "ケンセイガニ"}
	types := []string{"UmiDango", "OoUmiDango", "DaiouUmiDango", "KaeruAnkou", "OoKaeruAnkou", "NushiKaeruAnkou", "HajikiGani", "KengouGani", "KenseiGani"}
	stats := [][4]int{{34, 6, 6, 20}, {60, 11, 9, 50}, {102, 22, 11, 125}, {26, 5, 4, 14}, {48, 10, 7, 38}, {82, 20, 9, 95}, {40, 9, 8, 28}, {70, 15, 11, 70}, {119, 30, 13, 175}}
	ranges := [][2]int{{4, 10}, {11, 18}, {19, 30}, {3, 9}, {10, 17}, {18, 30}, {7, 13}, {14, 21}, {22, 30}}
	for k, want := range stats {
		id := 90 + k
		d, ok := MonsterDefinitions[id]
		if !ok || d.Name != names[k] || d.Type != types[k] || [4]int{d.Health, d.AttackPower, d.DefensePower, d.ExperiencePoints} != want || d.Char != []rune{'d', 'f', 'h'}[k/3] {
			t.Fatalf("definition %d: %+v", id, d)
		}
		if id >= 96 {
			if len(d.Traits) != 1 || d.Traits[0] != EnemyTraitShell {
				t.Fatal("shell trait missing")
			}
		} else if len(d.Traits) != 1 || !hasEnemyTrait(d.Traits, EnemyTraitAquatic) {
			t.Fatal("unexpected trait")
		}
		if k%3 < 2 && MonsterLevelUpTable[id] != id+1 {
			t.Fatalf("level up %d", id)
		}
		for floor := 1; floor <= 30; floor++ {
			weight := 0
			for _, entry := range FloorSpawnTables[floor] {
				if entry.MonsterID == id {
					weight += entry.Weight
				}
			}
			expected := 0
			if floor >= ranges[k][0] && floor <= ranges[k][1] {
				expected = 6
			}
			if weight != expected {
				t.Fatalf("id %d floor %d weight %d", id, floor, weight)
			}
		}
		e := CreateEnemyByID(id, 2, 2)
		e.Health--
		restored := savedToEnemy(enemyToSaved(&e))
		if restored.ID != id || restored.Health != e.Health || restored.Type != e.Type {
			t.Fatalf("save %d", id)
		}
		if id >= 93 && id <= 95 {
			if d.Behavior == nil {
				t.Fatal("behavior missing")
			}
		} else if restored.SpecialAttack == nil || d.SpecialAttackProbability != 0.25 {
			t.Fatal("attack missing")
		}
	}
	for _, p := range helpPages(defaultKeyBindings()) {
		if p.Title == "新しい敵（3）" {
			if !strings.Contains(strings.Join(p.Lines, ""), "上位種") {
				t.Fatal("help incomplete")
			}
			return
		}
	}
	t.Fatal("help missing")
}

func TestWave5ARam(t *testing.T) {
	for _, id := range []int{90, 91, 92} {
		for _, defense := range []int{0, 100} {
			g := newFoundationTestGame()
			g.state.Player.DefensePower = defense
			addTestEnemy(g, id, 3, 4)
			MonsterDefinitions[id].SpecialAttack(&g.state.Enemies[0], g)
			if g.state.Player.Health != 40 || g.state.Player.X != 4 {
				t.Fatal("immediate ram")
			}
			runQueuedActions(g)
			if g.state.Player.Health != 40-umiDangoRamDamage[id] || g.state.Player.X != 6 {
				t.Fatalf("ram %d: %+v", id, g.state.Player)
			}
		}
	}
	for _, tc := range []struct {
		name      string
		ability   EquipmentAbilityID
		wall      bool
		damage, x int
	}{
		{"wall", "", true, 11, 5}, {"break fall", AbilityBreakFall, true, 6, 5}, {"immovable", AbilityImmovable, false, 6, 4}, {"evade", AbilityEvasion, false, 0, 4}, {"melee not reflected", AbilityReflection, false, 6, 6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newFoundationTestGame()
			addTestEnemy(g, 90, 3, 4)
			if tc.ability != "" {
				equip(g, testArmor(tc.ability))
			}
			g.randHooks.shieldDefense = func(int) int { return 0 }
			if tc.wall {
				g.state.Map[4][6] = Tile{Type: "wall", Blocked: true}
			}
			MonsterDefinitions[90].SpecialAttack(&g.state.Enemies[0], g)
			runQueuedActions(g)
			if g.state.Player.Health != 40-tc.damage || g.state.Player.X != tc.x {
				t.Fatalf("HP %d x %d", g.state.Player.Health, g.state.Player.X)
			}
		})
	}
	for _, count := range []int{1, 2} {
		g := newFoundationTestGame()
		addTestEnemy(g, 90, 3, 4)
		weapon := testWeapon()
		equip(g, weapon)
		g.state.Player.Inventory = append(g.state.Player.Inventory, buildItemFromTemplate(1, 0, 0), buildItemFromTemplate(2, 0, 0))
		g.randHooks.displacement = func(n int) int {
			if n == 2 {
				return count - 1
			}
			return 0
		}
		MonsterDefinitions[90].SpecialAttack(&g.state.Enemies[0], g)
		runQueuedActions(g)
		if len(g.state.Items) != count || len(g.state.Player.Inventory) != 3-count || g.state.Player.EquippedWeapon != weapon {
			t.Fatal("drop count/equipment")
		}
	}
}

// 行動確率に成功/失敗するシードを探し、同じ系列を再現する。
func wave5APullGame(success bool) *Game {
	for seed := int64(1); ; seed++ {
		rng := newTurnRNG(seed)
		if (rng.Intn(2) == 0) == success {
			g := newFoundationTestGame()
			g.rng = newTurnRNG(seed)
			return g
		}
	}
}

func TestWave5APull(t *testing.T) {
	for _, tc := range []struct {
		name                                              string
		x, y                                              int
		block, enemy, undiscovered, miss, immovable, trap bool
		want                                              bool
		px, py                                            int
	}{
		{name: "vertical", x: 4, y: 1, want: true, px: 4, py: 2},
		{name: "diagonal", x: 1, y: 1, want: true, px: 2, py: 2},
		{name: "two", x: 2, y: 4, want: true, px: 3, py: 4},
		{name: "four", x: 4, y: 0, px: 4, py: 4},
		{name: "not line", x: 1, y: 2, px: 4, py: 4},
		{name: "wall", x: 1, y: 4, block: true, px: 4, py: 4},
		{name: "enemy", x: 1, y: 4, enemy: true, px: 4, py: 4},
		{name: "adjacent", x: 3, y: 4, px: 4, py: 4},
		{name: "undiscovered", x: 1, y: 4, undiscovered: true, px: 4, py: 4},
		{name: "miss", x: 1, y: 4, miss: true, px: 4, py: 4},
		{name: "immovable", x: 1, y: 4, immovable: true, want: true, px: 4, py: 4},
		{name: "trap", x: 1, y: 4, trap: true, want: true, px: 3, py: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := wave5APullGame(!tc.miss)
			addTestEnemy(g, 93, tc.x, tc.y)
			g.state.Enemies[0].PlayerDiscovered = !tc.undiscovered
			if tc.block {
				g.state.Map[4][2] = Tile{Type: "wall", Blocked: true}
			}
			if tc.enemy {
				addTestEnemy(g, 0, 2, 4)
			}
			if tc.immovable {
				equip(g, testRing(AbilityImmovable))
			}
			stepped := 0
			if tc.trap {
				g.state.MapTraps = []MapTrap{{X: 3, Y: 4, Name: "試験の罠", Effect: func(*Game) { stepped++ }}}
			}
			if got := MonsterDefinitions[93].Behavior(g, 0); got != tc.want {
				t.Fatalf("behavior %v", got)
			}
			if g.state.Player.X != 4 || g.state.Player.Y != 4 {
				t.Fatal("immediate pull")
			}
			runQueuedActions(g)
			if g.state.Player.X != tc.px || g.state.Player.Y != tc.py || (tc.trap && stepped != 1) {
				t.Fatalf("position %d,%d trap %d", g.state.Player.X, g.state.Player.Y, stepped)
			}
		})
	}
}

func TestWave5AKnockShield(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		curse, wall, guard, noShield bool
	}{
		{name: "normal"}, {name: "cursed", curse: true}, {name: "wall", wall: true}, {name: "guard", guard: true}, {name: "no shield", noShield: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newFoundationTestGame()
			g.state.Map = makeTestFloorMap(12, 9)
			addTestEnemy(g, 96, 3, 4)
			armor := testArmor()
			armor.Cursed = tc.curse
			if !tc.noShield {
				equip(g, armor)
			}
			defense := g.state.Player.DefensePower
			if tc.wall {
				g.state.Map[4][7] = Tile{Type: "wall", Blocked: true}
			}
			if tc.guard {
				withStatusImmunityRing(t, g, StatusKnockOff)
			}
			MonsterDefinitions[96].SpecialAttack(&g.state.Enemies[0], g)
			if !tc.noShield && g.state.Player.EquippedArmor != armor {
				t.Fatal("immediate knockoff")
			}
			runQueuedActions(g)
			if tc.noShield {
				if g.state.Player.Health >= 40 {
					t.Fatal("no normal attack")
				}
				return
			}
			if tc.guard {
				if g.state.Player.EquippedArmor != armor || g.state.Player.DefensePower != defense {
					t.Fatal("guard failed")
				}
				return
			}
			x := 8
			if tc.wall {
				x = 6
			}
			if g.state.Player.EquippedArmor != nil || g.state.Player.DefensePower != defense-armor.DefensePower || len(g.state.Player.Inventory) != 0 || armor.Cursed != tc.curse {
				t.Fatal("equipment not removed")
			}
			if items := floorItemsAt(g, x, 4); len(items) != 1 || items[0] != armor {
				t.Fatalf("landing %v", g.state.Items)
			}
		})
	}
}

func TestWave5ASealAndBackward(t *testing.T) {
	for id := 90; id <= 98; id++ {
		for _, backward := range []bool{false, true} {
			g := newFoundationTestGame()
			addTestEnemy(g, id, 3, 4)
			armor := testArmor()
			equip(g, armor)
			if backward {
				g.state.Enemies[0].StatusAilments.Backward = 2
			} else {
				g.state.Enemies[0].StatusAilments.Seal = true
			}
			g.actEnemy(0)
			message := runQueuedActions(g)
			if g.state.Player.X != 4 || g.state.Player.EquippedArmor != armor || strings.Contains(message, "体当たり") || strings.Contains(message, "疑似餌") || strings.Contains(message, "はじいた") {
				t.Fatalf("id %d special not blocked: %s", id, message)
			}
		}
	}
	g := wave5APullGame(true)
	addTestEnemy(g, 93, 1, 4)
	g.state.Enemies[0].StatusAilments.Seal = true
	g.actEnemy(0)
	runQueuedActions(g)
	if g.state.Player.X != 4 {
		t.Fatal("sealed ranged pull")
	}
}

func TestWave5AShellSlayer(t *testing.T) {
	for id := 96; id <= 98; id++ {
		e := CreateEnemyByID(id, 3, 4)
		if multiplier, matched := slayerMultiplier([]EquipmentAbilityID{AbilityShellSlayer}, e.Traits); multiplier != 1.5 || !matched {
			t.Fatalf("slayer %d", id)
		}
	}
}

func TestWave5AUpperAbilitiesAfterSave(t *testing.T) {
	for id := 90; id <= 98; id++ {
		g := wave5APullGame(true)
		x := 3
		if id >= 93 && id <= 95 {
			x = 1
		}
		e := CreateEnemyByID(id, x, 4)
		e.PlayerDiscovered = true
		g.state.Enemies = []Enemy{savedToEnemy(enemyToSaved(&e))}
		if id >= 93 && id <= 95 {
			if !MonsterDefinitions[g.state.Enemies[0].ID].Behavior(g, 0) {
				t.Fatalf("restored pull %d", id)
			}
			runQueuedActions(g)
			if g.state.Player.X != 2 {
				t.Fatalf("pull %d", id)
			}
		} else {
			armor := testArmor()
			equip(g, armor)
			g.state.Enemies[0].SpecialAttack(&g.state.Enemies[0], g)
			runQueuedActions(g)
			if id >= 96 {
				if g.state.Player.EquippedArmor != nil {
					t.Fatalf("knockoff %d", id)
				}
			} else if g.state.Player.Health != 40-umiDangoRamDamage[id] {
				t.Fatalf("ram %d", id)
			}
		}
	}
}

func TestWave5ARamCollisionEnemyAndDeathSource(t *testing.T) {
	g := newFoundationTestGame()
	addTestEnemy(g, 90, 3, 4)
	addTestEnemy(g, 0, 6, 4)
	MonsterDefinitions[90].SpecialAttack(&g.state.Enemies[0], g)
	runQueuedActions(g)
	if g.state.Player.X != 5 || g.state.Player.Health != 29 {
		t.Fatal("enemy collision")
	}
	g = newFoundationTestGame()
	addTestEnemy(g, 90, 3, 4)
	g.state.Player.Health = 10
	g.state.Map[4][6] = Tile{Type: "wall", Blocked: true}
	MonsterDefinitions[90].SpecialAttack(&g.state.Enemies[0], g)
	message := runQueuedActions(g)
	if !g.playerDead || !strings.Contains(message, "体当たり") {
		t.Fatalf("collision death: %s", message)
	}
}

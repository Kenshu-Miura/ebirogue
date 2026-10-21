//go:build !test

package main

import (
	"strings"
	"testing"
)

func TestWave5BTablesAndSave(t *testing.T) {
	names := []string{"サビヒトデ", "クサレヒトデ", "ドロヒトデ", "イタズラカワハギ", "ワルサカワハギ", "アクタレカワハギ", "オオハサミガニ", "ギロチンガニ", "ダンザイガニ"}
	types := []string{"SabiHitode", "KusareHitode", "DoroHitode", "ItazuraKawahagi", "WarusaKawahagi", "AkutareKawahagi", "OohasamiGani", "GuillotineGani", "DanzaiGani"}
	stats := [][4]int{{30, 6, 5, 18}, {55, 11, 8, 45}, {94, 22, 10, 112}, {22, 4, 3, 12}, {40, 8, 6, 32}, {68, 16, 8, 80}, {36, 10, 6, 26}, {64, 17, 9, 65}, {109, 34, 11, 162}}
	bands := [][2]int{{5, 11}, {12, 19}, {20, 30}, {3, 9}, {10, 17}, {18, 30}, {6, 12}, {13, 20}, {21, 30}}
	for k, want := range stats {
		id := 99 + k
		d := MonsterDefinitions[id]
		if d.Name != names[k] || d.Type != types[k] || [4]int{d.Health, d.AttackPower, d.DefensePower, d.ExperiencePoints} != want || d.MaxHealth != want[0] {
			t.Fatalf("definition %d: %+v", id, d)
		}
		if k%3 < 2 && MonsterLevelUpTable[id] != id+1 {
			t.Fatalf("level up %d", id)
		}
		if id < 105 {
			if hasEnemyTrait(d.Traits, EnemyTraitAquatic) != (id >= 102) || d.CriticalPercent != 0 || d.SpecialAttack == nil || d.SpecialAttackProbability != 0.3 {
				t.Fatalf("abilities %d", id)
			}
		} else {
			if len(d.Traits) != 1 || d.Traits[0] != EnemyTraitShell || d.CriticalPercent != 15 || d.SpecialAttack != nil {
				t.Fatalf("crab %d", id)
			}
		}
		if (d.OnDamaged != nil) != (id <= 101) {
			t.Fatalf("OnDamaged %d", id)
		}
		if d.Char != []rune{'s', 'k', 'g'}[k/3] {
			t.Fatalf("char %d", id)
		}
		for floor := 1; floor <= 30; floor++ {
			weight := 0
			for _, e := range FloorSpawnTables[floor] {
				if e.MonsterID == id {
					weight += e.Weight
				}
			}
			expected := 0
			if floor >= bands[k][0] && floor <= bands[k][1] {
				expected = 6
			}
			if weight != expected {
				t.Fatalf("spawn %d/%d = %d", id, floor, weight)
			}
		}
		e := CreateEnemyByID(id, 5, 4)
		e.Health--
		restored := savedToEnemy(enemyToSaved(&e))
		if restored.ID != id || restored.Health != e.Health || restored.AttackPower != e.AttackPower || restored.DefensePower != e.DefensePower || (restored.SpecialAttack != nil) != (e.SpecialAttack != nil) {
			t.Fatalf("save %d", id)
		}
	}
	for _, p := range helpPages() {
		if p.Title == "新しい敵（4）" {
			if !strings.Contains(strings.Join(p.Lines, ""), "分身は経験値が無い") {
				t.Fatal("clone help")
			}
			return
		}
	}
	t.Fatal("help page missing")
}

func TestWave5BRust(t *testing.T) {
	for _, id := range []int{99, 100, 101} {
		for _, armor := range []bool{false, true} {
			g := newFoundationTestGame()
			w, a := testWeapon(), testArmor()
			if armor {
				equip(g, a)
			} else {
				equip(g, w)
			}
			addTestEnemy(g, id, 5, 4)
			attack, defense := g.state.Player.AttackPower, g.state.Player.DefensePower
			MonsterDefinitions[id].SpecialAttack(&g.state.Enemies[0], g)
			if w.Sharpness != 0 || a.Sharpness != 0 {
				t.Fatal("immediate rust")
			}
			runQueuedActions(g)
			if armor {
				if a.Sharpness != -1 || g.state.Player.DefensePower != defense-1 {
					t.Fatal("armor rust")
				}
			} else {
				if w.Sharpness != -1 || g.state.Player.AttackPower != attack-1 {
					t.Fatal("weapon rust")
				}
			}
		}
	}
	for _, mode := range []string{"proof", "gold", "ring"} {
		for _, armor := range []bool{false, true} {
			t.Run(mode+map[bool]string{false: "weapon", true: "armor"}[armor], func(t *testing.T) {
				g := newFoundationTestGame()
				w, a := testWeapon(), testArmor()
				if mode == "proof" {
					w.RustProof = true
					a.RustProof = true
				}
				if mode == "gold" {
					w.Abilities = []EquipmentAbilityID{AbilityRustImmune}
					a.Abilities = []EquipmentAbilityID{AbilityRustImmune}
				}
				if armor {
					equip(g, a)
				} else {
					equip(g, w)
				}
				if mode == "ring" {
					withStatusImmunityRing(t, g, StatusRust)
				}
				addTestEnemy(g, 99, 5, 4)
				sabiHitodeAttack(&g.state.Enemies[0], g)
				runQueuedActions(g)
				if w.Sharpness != 0 || a.Sharpness != 0 {
					t.Fatal("guard failed")
				}
			})
		}
	}
	g := newFoundationTestGame()
	addTestEnemy(g, 99, 5, 4)
	sabiHitodeAttack(&g.state.Enemies[0], g)
	runQueuedActions(g)
	if g.state.Player.Health >= 40 {
		t.Fatal("no equipment must attack")
	}
}

// 最初の100択が指定値になるシードを固定し、20%の両側を実際の被弾経路で検査する。
func wave5BRollSeed(want int) int64 {
	for seed := int64(0); ; seed++ {
		if newTurnRNG(seed).Intn(100) == want {
			return seed
		}
	}
}

func TestWave5BSplit(t *testing.T) {
	for _, id := range []int{99, 100, 101} {
		for _, mode := range []string{"split", "miss", "dead", "seal", "limit", "blocked"} {
			g := newFoundationTestGame()
			addTestEnemy(g, id, 5, 4)
			g.state.Enemies[0].Health = 20
			roll := 19
			if mode == "miss" {
				roll = 20
			}
			g.rng = newTurnRNG(wave5BRollSeed(roll))
			if mode == "seal" {
				g.state.Enemies[0].StatusAilments.Seal = true
			}
			if mode == "limit" {
				for len(g.state.Enemies) < MaxEnemiesOnFloor {
					addTestEnemy(g, 0, 1, 1)
				}
			}
			if mode == "blocked" {
				for _, o := range nearestFreeOffsets(1)[1:] {
					g.state.Map[4+o.Y][5+o.X] = Tile{Type: "wall", Blocked: true}
				}
			}
			before := len(g.state.Enemies)
			damage := 5
			if mode == "dead" {
				damage = 20
			}
			g.applyDamageToEnemy(0, damage)
			expected := before
			if mode == "split" {
				expected++
			}
			if mode == "dead" {
				expected--
			}
			if len(g.state.Enemies) != expected {
				t.Fatalf("%d %s: enemies %d", id, mode, len(g.state.Enemies))
			}
			if mode == "split" {
				c := g.state.Enemies[1]
				if c.ID != id || c.Health != 15 || c.ExperiencePoints != 0 || max(abs(c.X-5), abs(c.Y-4)) != 1 {
					t.Fatalf("clone %+v", c)
				}
				if savedToEnemy(enemyToSaved(&c)).ExperiencePoints != 0 {
					t.Fatal("clone save")
				}
			}
		}
	}
}

func TestWave5BToss(t *testing.T) {
	for _, id := range []int{102, 103, 104} {
		for _, mode := range []string{"cane", "food", "empty", "guard", "equipped", "stuck"} {
			t.Run(MonsterDefinitions[id].Name+mode, func(t *testing.T) {
				g := newFoundationTestGame()
				g.state.Map = makeTestFloorMap(25, 25)
				addTestEnemy(g, id, 5, 4)
				var target Item
				switch mode {
				case "cane":
					target = buildItemFromTemplate(16, 0, 0)
					g.state.Player.Inventory = []Item{buildItemFromTemplate(1, 0, 0), target}
				case "food", "guard":
					target = buildItemFromTemplate(1, 0, 0)
					g.state.Player.Inventory = []Item{target}
					if mode == "guard" {
						withStatusImmunityRing(t, g, StatusItemToss)
					}
				case "equipped":
					equip(g, testWeapon(), testArmor())
				case "stuck":
					for _, d := range itemTemplates {
						if d.Name == "聖域のカード" {
							c := buildItemFromTemplate(d.ID, 2, 2).(*Card)
							c.Stuck = true
							target = c
							g.state.Items = []Item{c}
							g.state.Player.Inventory = []Item{c}
							break
						}
					}
					if target == nil {
						t.Fatal("sanctuary missing")
					}
				}
				before := len(g.state.Player.Inventory)
				MonsterDefinitions[id].SpecialAttack(&g.state.Enemies[0], g)
				if len(g.state.Player.Inventory) != before {
					t.Fatal("immediate toss")
				}
				msg := runQueuedActions(g)
				if mode == "cane" || mode == "food" {
					if len(g.state.Player.Inventory) != before-1 || len(g.state.Items) != 1 || g.state.Items[0] != target {
						t.Fatal("toss did not move target")
					}
					if g.state.Enemies[0].X != 5 || g.state.Enemies[0].Y != 4 || g.state.Enemies[0].Fleeing {
						t.Fatal("enemy fled")
					}
				} else {
					if len(g.state.Player.Inventory) != before {
						t.Fatal("protected target tossed")
					}
					if mode == "empty" && !strings.Contains(msg, "何も取られなかった") {
						t.Fatal(msg)
					}
				}
			})
		}
	}
}

func TestWave5BSealedAndBackward(t *testing.T) {
	for id := 99; id <= 107; id++ {
		for _, backward := range []bool{false, true} {
			g := newFoundationTestGame()
			w := testWeapon()
			equip(g, w)
			food := buildItemFromTemplate(1, 0, 0)
			g.state.Player.Inventory = append(g.state.Player.Inventory, food)
			addTestEnemy(g, id, 5, 4)
			if backward {
				g.state.Enemies[0].StatusAilments.Backward = 2
			} else {
				g.state.Enemies[0].StatusAilments.Seal = true
			}
			g.randHooks.enemyCritical = func(int) int { return 0 }
			g.AttackFromEnemy(0)
			msg := runQueuedActions(g)
			if w.Sharpness != 0 || len(g.state.Player.Inventory) != 2 || strings.Contains(msg, "痛恨の一撃") {
				t.Fatalf("sealed/backward %d: %s", id, msg)
			}
			if backward && g.state.Player.Health != 40 {
				t.Fatal("backward hurt player")
			}
		}
	}
}

func TestWave5BCriticalAndSlayer(t *testing.T) {
	for _, id := range []int{105, 106, 107} {
		for _, roll := range []int{14, 15} {
			g := newFoundationTestGame()
			g.state.Player.Health = 1000
			addTestEnemy(g, id, 5, 4)
			g.randHooks.enemyCritical = func(int) int { return roll }
			g.AttackFromEnemy(0)
			msg := runQueuedActions(g)
			multiplier := 1.0
			if roll == 14 {
				multiplier = enemyCriticalMultiplier
			}
			want := rollEnemyAttackDamageWith(g.state.Enemies[0].AttackPower, g.state.Player.DefensePower, multiplier, g.damageIntn())
			if 1000-g.state.Player.Health != want || strings.Contains(msg, "痛恨の一撃") != (roll == 14) {
				t.Fatalf("critical %d/%d: %s", id, roll, msg)
			}
			if mult, ok := slayerMultiplier([]EquipmentAbilityID{AbilityShellSlayer}, g.state.Enemies[0].Traits); !ok || mult != 1.5 {
				t.Fatal("shell slayer")
			}
		}
	}
}

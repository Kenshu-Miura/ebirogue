//go:build !test

package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestWave6ATablesAndSave(t *testing.T) {
	cases := []struct {
		id                                  int
		name, typ                           string
		hp, atk, def, exp, from, to, weight int
		char                                rune
		explosive                           bool
	}{
		{108, "フクレフグ", "FukureFugu", 32, 7, 5, 24, 6, 12, 6, 'p', true},
		{109, "バクハツフグ", "BakuhatsuFugu", 56, 12, 8, 60, 13, 20, 6, 'p', true},
		{110, "ダイバクハツフグ", "DaibakuhatsuFugu", 95, 24, 10, 150, 21, 30, 6, 'p', true},
		{114, "シビレエイ", "ShibireEi", 38, 8, 7, 30, 8, 14, 5, 'e', false},
		{115, "デンキエイ", "DenkiEi", 66, 14, 10, 75, 15, 22, 5, 'e', false},
		{116, "ライテイエイ", "RaiteiEi", 112, 28, 12, 188, 23, 30, 5, 'e', false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d, ok := MonsterDefinitions[c.id]
			if !ok || d.Name != c.name || d.Type != c.typ || d.Char != c.char || d.Health != c.hp || d.MaxHealth != c.hp || d.AttackPower != c.atk || d.DefensePower != c.def || d.ExperiencePoints != c.exp || d.OnDamaged == nil || d.Behavior == nil {
				t.Fatalf("definition: %+v", d)
			}
			multiplier, matched := slayerMultiplier([]EquipmentAbilityID{AbilityExplosiveSlayer}, d.Traits)
			if matched != c.explosive || (matched && multiplier != 1.5) {
				t.Fatalf("slayer: %v %v", multiplier, matched)
			}
			if !hasEnemyTrait(d.Traits, EnemyTraitAquatic) || c.explosive && len(d.Traits) != 2 || !c.explosive && len(d.Traits) != 1 {
				t.Fatalf("traits: %v", d.Traits)
			}
			for floor := 1; floor <= 30; floor++ {
				count := 0
				for _, entry := range FloorSpawnTables[floor] {
					if entry.MonsterID == c.id {
						count++
						if entry.Weight != c.weight {
							t.Fatal("weight")
						}
					}
				}
				want := 0
				if floor >= c.from && floor <= c.to {
					want = 1
				}
				if count != want {
					t.Fatalf("floor %d: %d", floor, count)
				}
			}
			e := CreateEnemyByID(c.id, 5, 4)
			e.Health--
			e.Counter, e.Timer, e.Flag = 1, 7, true
			restored := savedToEnemy(enemyToSaved(&e))
			if restored.ID != e.ID || restored.Health != e.Health || restored.Counter != 1 || restored.Timer != 7 || !restored.Flag {
				t.Fatalf("saved: %+v", restored)
			}
			g := newFoundationTestGame()
			g.state.Enemies = []Enemy{restored}
			if c.explosive {
				if !MonsterDefinitions[restored.ID].Behavior(g, 0) {
					t.Fatal("restored fugu")
				}
			} else {
				MonsterDefinitions[restored.ID].Behavior(g, 0)
				if g.state.Enemies[0].Timer != 6 {
					t.Fatal("restored ray")
				}
			}
		})
	}
	for from, to := range map[int]int{108: 109, 109: 110, 114: 115, 115: 116} {
		if MonsterLevelUpTable[from] != to {
			t.Fatal("level up", from)
		}
	}
}

func wave6AGame(id, x int) (*Game, int) {
	g := newFoundationTestGame()
	g.state.Player.Health, g.state.Player.MaxHealth = 200, 200
	return g, addTestEnemy(g, id, x, 4)
}

func TestWave6AFuguThresholds(t *testing.T) {
	for _, id := range []int{108, 109, 110} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			g, i := wave6AGame(id, 5)
			e := &g.state.Enemies[i]
			half := e.MaxHealth / 2
			g.applyDamageToEnemy(i, e.Health-half-1)
			if e.Flag {
				t.Fatal("above half")
			}
			g.applyDamageToEnemy(i, 1)
			if !e.Flag || !MonsterDefinitions[id].Behavior(g, i) {
				t.Fatal("half should inflate")
			}
			g.actEnemy(i)
			runQueuedActions(g)
			if e.X != 5 || g.state.Player.Health != 200 {
				t.Fatal("inflated should hold")
			}
			g.applyDamageToEnemy(i, e.Health-e.MaxHealth/4)
			g.applyDamageToEnemy(i, 1)
			if e.Counter != 1 {
				t.Fatal("reservation")
			}
			messages := runQueuedActions(g)
			if strings.Count(messages, "が破裂した！") != 1 || len(g.state.Enemies) != 0 || g.state.Player.Health != 200-fukureFuguExplosionDamage[id] || g.state.Player.ExperiencePoints != 0 {
				t.Fatalf("explosion: %s HP %d", messages, g.state.Player.Health)
			}
			g, i = wave6AGame(id, 5)
			g.applyDamageToEnemy(i, g.state.Enemies[i].Health-g.state.Enemies[i].MaxHealth/4)
			if g.state.Enemies[i].Flag {
				t.Fatal("skip inflation")
			}
			if strings.Contains(runQueuedActions(g), "膨らんだ") {
				t.Fatal("inflation message")
			}
			g, i = wave6AGame(id, 5)
			g.applyDamageToEnemy(i, g.state.Enemies[i].Health)
			if strings.Contains(runQueuedActions(g), "破裂") || g.state.Player.Health < 200 {
				t.Fatal("lethal hit must not explode")
			}
		})
	}
}

func TestWave6AFuguBlastAndChain(t *testing.T) {
	g, i := wave6AGame(108, 5)
	equip(g, testArmor(AbilityExplosionResistance))
	g.state.Items = []Item{buildItemFromTemplate(1, 5, 3), buildItemFromTemplate(1, 7, 7)}
	j := addTestEnemy(g, 108, 6, 4)
	g.state.Enemies[j].Health = 20
	k := addTestEnemy(g, 0, 5, 3)
	g.state.Enemies[k].Health = 1
	survivor := addTestEnemy(g, 1, 6, 3)
	g.state.Enemies[survivor].Health = 100
	g.applyDamageToEnemy(i, 24)
	messages := runQueuedActions(g)
	if strings.Count(messages, "が破裂した！") != 2 || len(g.state.Enemies) != 1 || g.state.Enemies[0].Health != 72 || len(g.state.Items) != 1 || g.state.Player.ExperiencePoints != 0 || g.state.Player.Health != 193 {
		t.Fatalf("chain: %s enemies %+v HP %d", messages, g.state.Enemies, g.state.Player.Health)
	}
	if g.lastDamageSource.Label() != "フクレフグの爆発" {
		t.Fatal(g.lastDamageSource.Label())
	}
}

func TestWave6ASeal(t *testing.T) {
	for _, id := range []int{108, 109, 110, 114, 115, 116} {
		g, i := wave6AGame(id, 6)
		g.state.Enemies[i].StatusAilments.Seal = true
		g.applyDamageToEnemy(i, g.state.Enemies[i].Health-1)
		runQueuedActions(g)
		if g.state.Enemies[i].Flag || g.state.Enemies[i].Counter != 0 {
			t.Fatal("sealed damage", id)
		}
		g.state.Enemies[i].Flag, g.state.Enemies[i].Timer = true, 10
		if g.runEnemyBehavior(i) || g.state.Enemies[i].Timer != 10 {
			t.Fatal("sealed behavior", id)
		}
		g.state.Enemies[i].PlayerDiscovered = true
		g.actEnemy(i)
		runQueuedActions(g)
		if g.state.Enemies[i].X == 6 {
			t.Fatal("sealed enemy should move", id)
		}
	}
}

func wave6ALightningSeed() int64 {
	for seed := int64(0); ; seed++ {
		r := newTurnRNG(seed)
		if r.Intn(3) == 0 {
			return seed
		}
	}
}

func TestWave6ARayChargeAndDischarge(t *testing.T) {
	for _, id := range []int{114, 115, 116} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			for _, ability := range []EquipmentAbilityID{"", AbilityMagicResistance, AbilityEvasion, AbilityReflection} {
				g, i := wave6AGame(id, 5)
				if ability != "" {
					equip(g, testArmor(ability))
				}
				g.randHooks.shieldDefense = func(int) int { return 0 }
				for hit := 1; hit <= 3; hit++ {
					g.applyDamageToEnemy(i, 1)
					if g.state.Enemies[i].Flag != (hit == 3) {
						t.Fatal("charge", hit)
					}
				}
				if g.state.Enemies[i].Timer != 10 || g.state.Enemies[i].Counter != 0 {
					t.Fatal("charge state")
				}
				runQueuedActions(g)
				if g.state.Player.Health != 200 {
					t.Fatal("third hit should only charge")
				}
				g.applyDamageToEnemy(i, 1)
				runQueuedActions(g)
				want := shibireEiDischargeDamage[id]
				if ability == AbilityMagicResistance {
					want /= 2
				}
				if g.state.Player.Health != 200-want || !g.state.Enemies[i].Flag || g.state.Enemies[i].Timer != 10 {
					t.Fatal("discharge", ability, g.state.Player.Health)
				}
				g.state.Enemies[i].X = 6
				g.applyDamageToEnemy(i, 1)
				if strings.Contains(runQueuedActions(g), "放電") {
					t.Fatal("distant discharge")
				}
			}
			g, i := wave6AGame(id, 6)
			g.state.Enemies[i].Flag, g.state.Enemies[i].Timer = true, 10
			for turn := 1; turn <= 10; turn++ {
				if MonsterDefinitions[id].Behavior(g, i) {
					t.Fatal("undiscovered lightning")
				}
				if g.state.Enemies[i].Flag != (turn < 10) {
					t.Fatal("expiry", turn)
				}
			}
			if !strings.Contains(runQueuedActions(g), "帯電が消えた") {
				t.Fatal("expiry message")
			}
		})
	}
}

func TestWave6ARayLightning(t *testing.T) {
	for _, id := range []int{114, 115, 116} {
		for _, ability := range []EquipmentAbilityID{"", AbilityMagicResistance, AbilityReflection, AbilityEvasion} {
			t.Run(fmt.Sprintf("%d/%s", id, ability), func(t *testing.T) {
				g, i := wave6AGame(id, 6)
				g.rooms = []Room{{X: 1, Y: 1, Width: 7, Height: 7}}
				e := &g.state.Enemies[i]
				e.Flag, e.Timer, e.PlayerDiscovered = true, 10, true
				if ability != "" {
					equip(g, testArmor(ability))
				}
				g.randHooks.shieldDefense = func(int) int { return 0 }
				g.rng = newTurnRNG(wave6ALightningSeed())
				before := e.Health
				if !MonsterDefinitions[id].Behavior(g, i) {
					t.Fatal("lightning")
				}
				messages := runQueuedActions(g)
				want := shibireEiLightningDamage[id]
				if ability == AbilityMagicResistance {
					want /= 2
				}
				if ability == AbilityReflection || ability == AbilityEvasion {
					want = 0
				}
				if g.state.Player.Health != 200-want || !strings.Contains(messages, "雷を落とした") {
					t.Fatal("lightning damage", messages, g.state.Player.Health)
				}
				if ability == AbilityReflection && g.state.Enemies[i].Health != before-shibireEiLightningDamage[id] {
					t.Fatal("reflection")
				}
			})
		}
	}
	for _, discovered := range []bool{false, true} {
		g, i := wave6AGame(114, 6)
		g.state.Enemies[i].Flag, g.state.Enemies[i].Timer, g.state.Enemies[i].PlayerDiscovered = true, 10, discovered
		if !discovered {
			g.rooms = []Room{{X: 1, Y: 1, Width: 7, Height: 7}}
		} else {
			g.rooms = []Room{{X: 1, Y: 1, Width: 4, Height: 7}, {X: 5, Y: 1, Width: 3, Height: 7}}
		}
		g.rng = newTurnRNG(wave6ALightningSeed())
		if MonsterDefinitions[114].Behavior(g, i) || len(g.ActionQueue.Queue) != 0 {
			t.Fatal("room/discovery guard")
		}
	}
}

// 実際の投擲経路でも被弾フックが働き、離れた投げ手には放電しない。
func TestWave6ARayThrownHitAndRestoredCharge(t *testing.T) {
	g := newProjectileTestGame()
	i := addTestEnemy(g, 114, 6, 4)
	e := &g.state.Enemies[i]
	e.Counter = 2
	restored := savedToEnemy(enemyToSaved(e))
	g.state.Enemies[i] = restored
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}
	g.throwWithCallbacks(food, 10)
	messages := finishThrow(t, g)
	if !g.state.Enemies[i].Flag || g.state.Enemies[i].Timer != 10 || !strings.Contains(messages, "帯電した") {
		t.Fatal("restored third hit", messages)
	}
	food = buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}
	g.throwWithCallbacks(food, 10)
	messages = finishThrow(t, g)
	if strings.Contains(messages, "放電した") || g.state.Player.Health != 40 {
		t.Fatal("distant throw retaliation", messages)
	}
}

func TestWave6ARaySealAndLethalHit(t *testing.T) {
	for _, id := range []int{114, 115, 116} {
		g, i := wave6AGame(id, 5)
		g.state.Enemies[i].StatusAilments.Seal = true
		for range 4 {
			g.applyDamageToEnemy(i, 1)
		}
		if g.state.Enemies[i].Counter != 0 || g.state.Enemies[i].Flag {
			t.Fatal("sealed charge", id)
		}
		g.state.Enemies[i].StatusAilments.Seal = false
		g.state.Enemies[i].Flag, g.state.Enemies[i].Timer = true, 10
		g.applyDamageToEnemy(i, g.state.Enemies[i].Health)
		if strings.Contains(runQueuedActions(g), "放電した") {
			t.Fatal("lethal retaliation", id)
		}
	}
}

//go:build !test

package main

import (
	"fmt"
	"testing"
)

func TestWave6BDefinitionsAndSave(t *testing.T) {
	rows := []struct {
		id                                         int
		name, typ                                  string
		hp, attack, defense, exp, from, to, weight int
	}{
		{111, "アナジャコ", "AnaJako", 36, 8, 7, 26, 7, 13, 5},
		{112, "オオアナジャコ", "OoAnaJako", 63, 14, 10, 65, 14, 21, 5},
		{113, "ジゴクアナジャコ", "JigokuAnaJako", 107, 28, 12, 162, 22, 30, 5},
		{120, "コバンザメ", "Kobanzame", 28, 6, 4, 20, 5, 11, 6},
		{121, "オオバンザメ", "Oobanzame", 50, 11, 7, 50, 12, 19, 6},
		{122, "センリョウザメ", "Senryouzame", 85, 22, 9, 125, 20, 30, 6},
	}
	for _, r := range rows {
		t.Run(fmt.Sprint(r.id), func(t *testing.T) {
			d, ok := MonsterDefinitions[r.id]
			if !ok || d.Name != r.name || d.Type != r.typ || d.Health != r.hp || d.MaxHealth != r.hp || d.AttackPower != r.attack || d.DefensePower != r.defense || d.ExperiencePoints != r.exp || len(d.Traits) != 0 || d.Behavior == nil {
				t.Fatalf("definition: %+v", d)
			}
			char := 'a'
			if r.id >= 120 {
				char = 'z'
			}
			if d.Char != char {
				t.Fatal("char")
			}
			for floor := 1; floor <= 30; floor++ {
				found := 0
				for _, e := range FloorSpawnTables[floor] {
					if e.MonsterID == r.id {
						found++
						if e.Weight != r.weight {
							t.Fatal("weight")
						}
					}
				}
				want := 0
				if floor >= r.from && floor <= r.to {
					want = 1
				}
				if found != want {
					t.Fatalf("floor %d: %d", floor, found)
				}
			}
			e := CreateEnemyByID(r.id, 2, 4)
			e.Health--
			e.Alerted = true
			restored := savedToEnemy(enemyToSaved(&e))
			if restored.ID != e.ID || restored.Health != e.Health || !restored.Alerted || MonsterDefinitions[restored.ID].Behavior == nil {
				t.Fatalf("save: %+v", restored)
			}
		})
	}
	for _, pair := range [][2]int{{111, 112}, {112, 113}, {120, 121}, {121, 122}} {
		if MonsterLevelUpTable[pair[0]] != pair[1] {
			t.Fatal("level up", pair)
		}
	}
}

func TestWave6BAnaJakoBehavior(t *testing.T) {
	for _, id := range []int{111, 112, 113} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			g := newDigTestGame()
			g.state.Player.X, g.state.Player.Y = 10, 4
			i := addTestEnemy(g, id, 2, 4)
			for turn := 0; turn < 10; turn++ {
				g.state.Enemies[i].X, g.state.Enemies[i].Y = 2, 4
				if !MonsterDefinitions[id].Behavior(g, i) || g.state.Enemies[i].Alerted {
					t.Fatal("far enemy alerted")
				}
			}
			g = newDigTestGame()
			g.state.Player.X, g.state.Player.Y = 5, 4
			i = addTestEnemy(g, id, 2, 4)
			g.state.Enemies[i].Health--
			if !MonsterDefinitions[id].Behavior(g, i) || !g.state.Enemies[i].Alerted || g.state.Enemies[i].X != 3 || g.state.Map[4][3].Type != "corridor" {
				t.Fatal("damaged enemy must dig toward player")
			}
			g.state.Player.X = 10
			if !MonsterDefinitions[id].Behavior(g, i) || !g.state.Enemies[i].Alerted {
				t.Fatal("alert must persist")
			}
			g = newDigTestGame()
			g.state.Player.X, g.state.Player.Y = 4, 4
			i = addTestEnemy(g, id, 2, 4)
			if !MonsterDefinitions[id].Behavior(g, i) || !g.state.Enemies[i].Alerted {
				t.Fatal("proximity")
			}
			g.state.Player.X, g.state.Player.Y = g.state.Enemies[i].X+1, g.state.Enemies[i].Y
			if MonsterDefinitions[id].Behavior(g, i) {
				t.Fatal("adjacent must attack normally")
			}
			g = newDigTestGame()
			g.state.Player.X = 8
			i = addTestEnemy(g, id, 2, 4)
			if g.stepEnemyDigging(i, 1, 1) || g.state.Map[5][3].Type != "wall" {
				t.Fatal("diagonal wall")
			}
			g.state.Enemies[i].X = 1
			if g.stepEnemyDigging(i, -1, 0) || g.state.Map[4][0].Type != "wall" {
				t.Fatal("outer wall")
			}
			g.state.Enemies[i].X = 2
			g.state.Enemies[i].StatusAilments.Seal = true
			if g.runEnemyBehavior(i) || g.state.Map[4][3].Type != "wall" {
				t.Fatal("sealed")
			}
		})
	}
}

func wave6BCoinGame(id int) (*Game, int) {
	g := newFoundationTestGame()
	g.state.Map = makeTestFloorMap(14, 9)
	g.state.Player.X, g.state.Player.Y = 6, 4
	g.state.Player.Cash = 100
	g.state.Player.DefensePower = 999
	g.randHooks.shieldDefense = func(int) int { return 99 }
	i := addTestEnemy(g, id, 3, 4)
	return g, i
}

// 行動の成否を乱数系列で確かめつつ、成功するシードを決定的に選ぶ。
func wave6BThrow(g *Game, i int) bool {
	for seed := int64(1); seed <= 100; seed++ {
		g.rng = newTurnRNG(seed)
		if MonsterDefinitions[g.state.Enemies[i].ID].Behavior(g, i) {
			return true
		}
	}
	return false
}

func TestWave6BCoinAttack(t *testing.T) {
	for _, r := range []struct{ id, cost, damage int }{{120, 10, 6}, {121, 24, 14}, {122, 50, 30}} {
		t.Run(fmt.Sprint(r.id), func(t *testing.T) {
			g, i := wave6BCoinGame(r.id)
			if !wave6BThrow(g, i) {
				t.Fatal("no throw")
			}
			if g.state.Player.Cash != 100 || g.state.Player.Health != 40 {
				t.Fatal("effect before Execute")
			}
			runQueuedActions(g)
			if g.state.Player.Cash != 100-r.cost || g.state.Player.Health != 40-r.damage {
				t.Fatalf("cash/hp %d/%d", g.state.Player.Cash, g.state.Player.Health)
			}
			g, i = wave6BCoinGame(r.id)
			g.state.Player.Health = 1
			wave6BThrow(g, i)
			runQueuedActions(g)
			if !g.playerDead || g.lastDamageSource.Label() != MonsterDefinitions[r.id].Name+"の小銭" {
				t.Fatal("death cause", g.lastDamageSource)
			}
			g, i = wave6BCoinGame(r.id)
			g.state.Player.Cash = r.cost - 1
			if wave6BThrow(g, i) {
				t.Fatal("insufficient cash")
			}
			g, i = wave6BCoinGame(r.id)
			g.state.Player.Cash = r.cost
			if !wave6BThrow(g, i) {
				t.Fatal("exact cash")
			}
			g.state.Player.Cash = 0
			runQueuedActions(g)
			if g.state.Player.Cash != 0 {
				t.Fatal("negative cash")
			}
		})
	}
}

func TestWave6BCoinGuardsAndGeometry(t *testing.T) {
	for _, ability := range []EquipmentAbilityID{AbilityTheftResistance, AbilityEvasion, AbilityReflection} {
		t.Run(string(ability), func(t *testing.T) {
			g, i := wave6BCoinGame(120)
			equip(g, testArmor(ability))
			g.randHooks.shieldDefense = func(int) int { return 0 }
			thrown := wave6BThrow(g, i)
			if ability == AbilityTheftResistance {
				if thrown || len(g.ActionQueue.Queue) != 0 {
					t.Fatal("theft guard")
				}
				return
			}
			if !thrown {
				t.Fatal("no throw")
			}
			hp := g.state.Enemies[i].Health
			runQueuedActions(g)
			if g.state.Player.Cash != 90 || g.state.Player.Health != 40 {
				t.Fatal("evade/reflect must consume coins")
			}
			if ability == AbilityReflection && g.state.Enemies[i].Health != hp-6 {
				t.Fatal("reflection")
			}
		})
	}
	for _, name := range []string{"adjacent", "offLine", "tooFar", "wall", "enemy", "undiscovered", "sealed"} {
		t.Run(name, func(t *testing.T) {
			g, i := wave6BCoinGame(120)
			switch name {
			case "adjacent":
				g.state.Player.X = 4
			case "offLine":
				g.state.Player.Y = 5
			case "tooFar":
				g.state.Player.X = 9
			case "wall":
				g.state.Map[4][4] = Tile{Type: "wall", Blocked: true, BlockSight: true}
			case "enemy":
				addTestEnemy(g, 0, 4, 4)
			case "undiscovered":
				g.state.Enemies[i].PlayerDiscovered = false
			case "sealed":
				g.state.Enemies[i].StatusAilments.Seal = true
				if g.runEnemyBehavior(i) {
					t.Fatal("sealed behavior")
				}
				return
			}
			if wave6BThrow(g, i) || len(g.ActionQueue.Queue) != 0 {
				t.Fatal("invalid throw")
			}
		})
	}
}

func TestWave6BStatePriorityAndCoinChance(t *testing.T) {
	for _, id := range []int{111, 120} {
		for _, status := range []StatusAilments{{Sleep: 3}, {Paralysis: true}, {Confusion: 3}, {Blind: 3}} {
			g, i := wave6BCoinGame(id)
			g.state.Enemies[i].StatusAilments = status
			g.actEnemy(i)
			if len(g.ActionQueue.Queue) != 0 || g.state.Player.Cash != 100 || g.state.Enemies[i].Alerted {
				t.Fatalf("id %d: special action despite status %+v", id, status)
			}
		}
	}
	g, i := wave6BCoinGame(120)
	missed := false
	for seed := int64(1); seed <= 100; seed++ {
		g.rng = newTurnRNG(seed)
		if g.randIntn(2) == 1 {
			g.rng = newTurnRNG(seed)
			if MonsterDefinitions[120].Behavior(g, i) || len(g.ActionQueue.Queue) != 0 {
				t.Fatal("chance failure consumed action")
			}
			missed = true
			break
		}
	}
	if !missed {
		t.Fatal("no chance failure seed")
	}
	g, i = wave6BCoinGame(120)
	g.state.Player.X, g.state.Player.Y = 6, 7
	if !wave6BThrow(g, i) {
		t.Fatal("diagonal straight line should throw")
	}
	runQueuedActions(g)
	if g.state.Player.Health != 34 {
		t.Fatal("diagonal damage")
	}

	g = newDigTestGame()
	g.state.Player.X, g.state.Player.Y = 8, 4
	i = addTestEnemy(g, 111, 2, 4)
	g.state.Enemies[i].Alerted = true
	g.state.Enemies[i].PlayerDiscovered = false
	if MonsterDefinitions[111].Behavior(g, i) || g.state.Map[4][3].Type != "wall" {
		t.Fatal("undiscovered player must not be pursued")
	}
}

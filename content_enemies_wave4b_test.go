//go:build !test

package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestWave4BDefinitionsAndSave(t *testing.T) {
	rows := []struct {
		name, kind               string
		char                     rune
		hp, attack, defense, exp int
	}{
		{"ヒバナウミウシ", "HibanaUmiushi", 'A', 26, 6, 3, 16}, {"カエンウミウシ", "KaenUmiushi", 'A', 48, 11, 6, 40}, {"ゴウカウミウシ", "GoukaUmiushi", 'A', 82, 22, 8, 100},
		{"テッポウエビ", "TeppouEbi", 'P', 30, 7, 4, 18}, {"タイホウエビ", "TaihouEbi", 'P', 54, 12, 7, 45}, {"ゴウホウエビ", "GouhouEbi", 'P', 92, 24, 9, 112},
		{"ニゲハゼ", "NigeHaze", 'Z', 14, 2, 2, 24}, {"トンズラハゼ", "TonzuraHaze", 'Z', 26, 4, 5, 60}, {"キンピカハゼ", "KinpikaHaze", 'Z', 44, 8, 7, 150},
		{"ヨビコミボラ", "YobikomiBora", 'Y', 28, 5, 6, 22}, {"ヨビアツメボラ", "YobiatsumeBora", 'Y', 50, 9, 9, 55}, {"ダイショウカンボラ", "DaishoukanBora", 'Y', 85, 18, 11, 138},
	}
	for k, row := range rows {
		t.Run(row.kind, func(t *testing.T) {
			id := 66 + k
			d, ok := MonsterDefinitions[id]
			if !ok || d.Name != row.name || d.Type != row.kind || d.Char != row.char || d.Health != row.hp || d.MaxHealth != row.hp || d.AttackPower != row.attack || d.DefensePower != row.defense || d.ExperiencePoints != row.exp || hasEnemyTrait(d.Traits, EnemyTraitAquatic) != (id >= 72) {
				t.Fatalf("definition %d = %+v", id, d)
			}
			if k%3 != 2 && MonsterLevelUpTable[id] != id+1 {
				t.Fatal("missing level-up link")
			}
			e := CreateEnemyByID(id, 2, 3)
			e.Health--
			e.PlayerDiscovered = true
			e.StatusAilments.Sleep = 2
			restored := savedToEnemy(enemyToSaved(&e))
			if !reflect.DeepEqual(enemyToSaved(&e), enemyToSaved(&restored)) || restored.Name != row.name || restored.Type != row.kind || restored.SpecialMovement != e.SpecialMovement {
				t.Fatal("save round trip changed enemy")
			}
			if id <= 68 {
				if restored.SpecialAttack == nil || restored.SpecialAttackProbability != 0.3 {
					t.Fatal("fire ability lost")
				}
			} else if MonsterDefinitions[restored.ID].Behavior == nil {
				t.Fatal("behavior lost")
			}
			if id >= 72 && id <= 74 && restored.SpecialMovement != SpecialMovementDoubleSpeed {
				t.Fatal("double speed lost")
			}
		})
	}
}

func TestWave4BSpawnRanges(t *testing.T) {
	ranges := []monsterSpawnRange{{66, 3, 4, 5}, {66, 5, 8, 8}, {67, 7, 8, 4}, {67, 9, 16, 7}, {68, 15, 16, 4}, {68, 17, 30, 7}, {69, 4, 5, 5}, {69, 6, 10, 8}, {70, 10, 17, 7}, {71, 18, 30, 7}, {72, 2, 9, 4}, {73, 8, 16, 4}, {74, 15, 30, 3}, {75, 5, 11, 5}, {76, 12, 19, 5}, {77, 20, 30, 5}}
	for id := 66; id <= 77; id++ {
		for floor := 1; floor <= 30; floor++ {
			want := 0
			for _, r := range ranges {
				if r.MonsterID == id && floor >= r.From && floor <= r.To {
					want = r.Weight
				}
			}
			got, n := 0, 0
			for _, e := range FloorSpawnTables[floor] {
				if e.MonsterID == id {
					got = e.Weight
					n++
				}
			}
			if got != want || n > 1 {
				t.Fatalf("id %d floor %d weight %d (%d entries), want %d", id, floor, got, n, want)
			}
		}
	}
}

// ゲームの乱数源を差し替えず、最初の判定が目的の結果になるシードを選ぶ。
func wave4BRNG(n int, accept func(int) bool) *gameRNG {
	for seed := int64(0); ; seed++ {
		if accept(newTurnRNG(seed).Intn(n)) {
			return newTurnRNG(seed)
		}
	}
}

func TestWave4BHibanaFire(t *testing.T) {
	for _, id := range []int{66, 67, 68} {
		for _, defense := range []int{0, 999} {
			t.Run(fmt.Sprintf("%d/defense%d", id, defense), func(t *testing.T) {
				g := newFoundationTestGame()
				g.state.Player.DefensePower = defense
				addTestEnemy(g, id, 5, 4)
				g.rng = wave4BRNG(100, func(v int) bool { return v >= 25 })
				g.state.Enemies[0].SpecialAttack(&g.state.Enemies[0], g)
				if g.state.Player.Health != 40 {
					t.Fatal("damage before action execution")
				}
				messages := runQueuedActions(g)
				if g.state.Player.Health != 40-hibanaFireDamage[id] || !strings.Contains(messages, "火を吹いた") {
					t.Fatalf("HP=%d messages=%s", g.state.Player.Health, messages)
				}
				if g.lastDamageSource != enemyRangedSource(id, MonsterDefinitions[id].Name, "炎") {
					t.Fatalf("fire damage source = %+v", g.lastDamageSource)
				}
			})
		}
	}
	for _, mode := range []string{"misfire", "resist", "reflect", "sealed", "blind", "sanctuary"} {
		t.Run(mode, func(t *testing.T) {
			g := newFoundationTestGame()
			addTestEnemy(g, 66, 5, 4)
			g.rng = wave4BRNG(100, func(v int) bool { return v >= 25 })
			want := 32
			switch mode {
			case "misfire":
				g.rng = wave4BRNG(100, func(v int) bool { return v < 25 })
				want = 40
			case "resist":
				equip(g, testArmor(AbilityFireResistance))
				want = 36
			case "reflect":
				equip(g, testArmor(AbilityReflection))
				want = 40
			case "sealed":
				g.state.Enemies[0].StatusAilments.Seal = true
				g.actEnemy(0)
				messages := runQueuedActions(g)
				if strings.Contains(messages, "火を吹") || g.state.Player.Health >= 40 {
					t.Fatalf("sealed attack: %s", messages)
				}
				return
			case "blind":
				g.state.Player.StatusAilments.Blind = 3
			case "sanctuary":
				card := buildItemFromTemplate(55, 4, 4).(*Card)
				card.Stuck = true
				g.state.Items = append(g.state.Items, card)
				want = 40
			}
			g.state.Enemies[0].SpecialAttack(&g.state.Enemies[0], g)
			messages := runQueuedActions(g)
			if g.state.Player.Health != want {
				t.Fatalf("HP=%d want=%d", g.state.Player.Health, want)
			}
			if mode == "blind" && !strings.Contains(messages, "何者") {
				t.Fatal(messages)
			}
			if mode == "reflect" && g.state.Enemies[0].Health != 18 {
				t.Fatal("fire not reflected")
			}
		})
	}
}

func TestWave4BTeppou(t *testing.T) {
	cases := []struct {
		name                                string
		x, y                                int
		wall, enemy, corner, sealed, unseen bool
		attack                              bool
	}{
		{name: "vertical", x: 4, y: 2, attack: true}, {name: "horizontal", x: 2, y: 4, attack: true}, {name: "diagonal", x: 2, y: 2, attack: true},
		{name: "wall", x: 4, y: 2, wall: true}, {name: "enemy", x: 4, y: 2, enemy: true}, {name: "three", x: 4, y: 1}, {name: "nonline", x: 2, y: 3},
		{name: "corner", x: 2, y: 2, corner: true, attack: true}, {name: "sealed", x: 4, y: 2, sealed: true}, {name: "unseen", x: 4, y: 2, unseen: true},
	}
	for _, id := range []int{69, 70, 71} {
		for _, c := range cases {
			t.Run(fmt.Sprintf("%d/%s", id, c.name), func(t *testing.T) {
				g := newFoundationTestGame()
				addTestEnemy(g, id, c.x, c.y)
				e := &g.state.Enemies[0]
				e.StatusAilments.Seal = c.sealed
				e.PlayerDiscovered = !c.unseen
				if c.wall {
					g.state.Map[3][4].Blocked = true
				}
				if c.enemy {
					addTestEnemy(g, 0, 4, 3)
				}
				if c.corner {
					g.state.Map[2][3].Blocked = true
					g.state.Map[3][2].Blocked = true
				}
				if got := g.runEnemyBehavior(0); got != c.attack {
					t.Fatalf("behavior=%v want %v", got, c.attack)
				}
				before := g.state.Player.Health
				runQueuedActions(g)
				if (g.state.Player.Health < before) != c.attack {
					t.Fatal("unexpected damage")
				}
				if g.state.Enemies[0].X != c.x || g.state.Enemies[0].Y != c.y {
					t.Fatal("ranged attack moved enemy")
				}
			})
		}
	}
	g := newFoundationTestGame()
	addTestEnemy(g, 69, 4, 2)
	g.state.Enemies[0].StatusAilments.Seal = true
	g.actEnemy(0)
	if len(g.ActionQueue.Queue) != 0 {
		t.Fatal("sealed enemy attacked at range")
	}
}

func TestWave4BNigeHaze(t *testing.T) {
	for _, id := range []int{72, 73, 74} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			g := newFoundationTestGame()
			addTestEnemy(g, id, 5, 4)
			g.actEnemy(0)
			e := g.state.Enemies[0]
			if len(g.ActionQueue.Queue) != 0 || abs(e.X-4)+abs(e.Y-4) <= 1 {
				t.Fatal("haze failed to flee without attacking")
			}
			g = newFoundationTestGame()
			addTestEnemy(g, id, 5, 4)
			g.state.Enemies[0].StatusAilments.Seal = true
			g.actEnemy(0)
			if len(g.ActionQueue.Queue) == 0 {
				t.Fatal("sealed haze did not attack")
			}
			g = newFoundationTestGame()
			addTestEnemy(g, id, 5, 4)
			g.MoveEnemies()
			e = g.state.Enemies[0]
			if max(abs(e.X-5), abs(e.Y-4)) != 2 || len(g.ActionQueue.Queue) != 0 {
				t.Fatalf("double speed position=%d,%d", e.X, e.Y)
			}
			g = newFoundationTestGame()
			addTestEnemy(g, id, 5, 4)
			for y := 3; y <= 5; y++ {
				for x := 4; x <= 6; x++ {
					if x != 5 || y != 4 {
						g.state.Map[y][x].Blocked = true
					}
				}
			}
			g.actEnemy(0)
			if len(g.ActionQueue.Queue) != 0 || g.state.Enemies[0].X != 5 || g.state.Enemies[0].Y != 4 {
				t.Fatal("trapped haze attacked or moved")
			}
			g = newFoundationTestGame()
			addTestEnemy(g, id, 1, 1)
			g.state.Enemies[0].PlayerDiscovered = false
			g.rooms = []Room{{X: 0, Y: 0, Width: 3, Height: 3}}
			g.runEnemyBehavior(0)
			if len(g.ActionQueue.Queue) != 0 {
				t.Fatal("undiscovered haze attacked")
			}
		})
	}
}

func wave4BBoraGame(id int) *Game {
	g := newFoundationTestGame()
	g.Floor = 20
	g.rooms = []Room{{X: 0, Y: 0, Width: 9, Height: 9}}
	addTestEnemy(g, id, 5, 4)
	g.rng = wave4BRNG(4, func(v int) bool { return v == 0 })
	return g
}

func TestWave4BYobikomi(t *testing.T) {
	for _, id := range []int{75, 76, 77} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			g := wave4BBoraGame(id)
			if !g.runEnemyBehavior(0) {
				t.Fatal("did not call")
			}
			if len(g.state.Enemies) != 1 {
				t.Fatal("summoned before action execution")
			}
			messages := runQueuedActions(g)
			want := 2
			if id == 77 {
				want = 3
			}
			if len(g.state.Enemies) != want || !strings.Contains(messages, "ほら貝") {
				t.Fatalf("summoned=%d %s", len(g.state.Enemies)-1, messages)
			}
			for _, e := range g.state.Enemies[1:] {
				if !e.PlayerDiscovered || e.ID >= 75 && e.ID <= 77 || max(abs(e.X-5), abs(e.Y-4)) > 2 {
					t.Fatalf("invalid summon %+v", e)
				}
			}
		})
	}
	for _, mode := range []string{"miss", "otherRoom", "unseen", "crowded", "sealed", "genocide", "limit", "noSpace"} {
		t.Run(mode, func(t *testing.T) {
			g := wave4BBoraGame(77)
			wantCall := false
			switch mode {
			case "miss":
				g.rng = wave4BRNG(4, func(v int) bool { return v != 0 })
			case "otherRoom":
				g.rooms = []Room{{X: 0, Y: 0, Width: 5, Height: 9}}
			case "unseen":
				g.state.Enemies[0].PlayerDiscovered = false
			case "crowded":
				addTestEnemy(g, 0, 5, 3)
				addTestEnemy(g, 0, 6, 3)
				addTestEnemy(g, 0, 6, 4)
			case "sealed":
				g.state.Enemies[0].StatusAilments.Seal = true
			case "genocide":
				wantCall = true
				g.genocidedMonsterIDs = map[int]bool{}
				for _, e := range FloorSpawnTables[g.Floor] {
					for _, familyID := range monsterFamilyIDs(e.MonsterID) {
						g.genocidedMonsterIDs[familyID] = true
					}
				}
			case "limit":
				wantCall = true
				for y := 0; y < 9 && len(g.state.Enemies) < MaxEnemiesOnFloor; y++ {
					for x := 0; x < 9 && len(g.state.Enemies) < MaxEnemiesOnFloor; x++ {
						if max(abs(x-5), abs(y-4)) > 2 {
							addTestEnemy(g, 0, x, y)
						}
					}
				}
			case "noSpace":
				wantCall = true
				for y := 2; y <= 6; y++ {
					for x := 3; x <= 7; x++ {
						if x != 5 || y != 4 {
							g.state.Map[y][x].Blocked = true
						}
					}
				}
			}
			before := len(g.state.Enemies)
			if got := g.runEnemyBehavior(0); got != wantCall {
				t.Fatalf("call=%v want=%v", got, wantCall)
			}
			messages := runQueuedActions(g)
			if len(g.state.Enemies) != before {
				t.Fatal("unexpected summons")
			}
			if wantCall && !strings.Contains(messages, "しかし何も来なかった") {
				t.Fatal(messages)
			}
		})
	}
	// 1枠だけ空いているとき、第3段階も上限を越えない。
	g := wave4BBoraGame(77)
	for y := 0; y < 9 && len(g.state.Enemies) < MaxEnemiesOnFloor-1; y++ {
		for x := 0; x < 9 && len(g.state.Enemies) < MaxEnemiesOnFloor-1; x++ {
			if max(abs(x-5), abs(y-4)) > 2 {
				addTestEnemy(g, 0, x, y)
			}
		}
	}
	g.runEnemyBehavior(0)
	runQueuedActions(g)
	if len(g.state.Enemies) != MaxEnemiesOnFloor {
		t.Fatal("summon exceeded or missed cap")
	}
}

func TestWave4BCommonStatusAndGenocide(t *testing.T) {
	for _, id := range []int{66, 69, 72, 75} {
		for _, status := range []StatusAilments{{Sleep: 3}, {Paralysis: true}} {
			g := wave4BBoraGame(id)
			g.state.Enemies[0].StatusAilments = status
			g.actEnemy(0)
			if len(g.ActionQueue.Queue) != 0 || g.state.Enemies[0].X != 5 || g.state.Enemies[0].Y != 4 {
				t.Fatalf("id %d acted while incapacitated", id)
			}
		}
	}
	g := wave4BBoraGame(75)
	g.state.Enemies[0].StatusAilments.Seal = true
	g.actEnemy(0)
	if messages := runQueuedActions(g); strings.Contains(messages, "ほら貝") || len(g.state.Enemies) != 1 || g.state.Player.Health == 40 {
		t.Fatal("sealed bora did not use a normal attack")
	}
	// 召喚候補をテスト中だけ絞り、封じた系統を除いた残りから選ばれることを確認する。
	previous := FloorSpawnTables[20]
	t.Cleanup(func() { FloorSpawnTables[20] = previous })
	FloorSpawnTables[20] = []MonsterSpawnEntry{{MonsterID: 66, Weight: 100}, {MonsterID: 67, Weight: 100}, {MonsterID: 68, Weight: 100}, {MonsterID: 75, Weight: 100}, {MonsterID: 0, Weight: 1}}
	g = wave4BBoraGame(77)
	g.applyGenocide(67)
	g.runEnemyBehavior(0)
	runQueuedActions(g)
	if len(g.state.Enemies) != 3 || g.state.Enemies[1].ID != 0 || g.state.Enemies[2].ID != 0 {
		t.Fatal("summon selected a genocided or summoner family")
	}
}

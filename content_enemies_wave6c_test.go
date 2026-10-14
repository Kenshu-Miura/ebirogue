//go:build !test

package main

import (
	"strings"
	"testing"
)

func TestWave6CTablesAndSave(t *testing.T) {
	// 指示書の数値を固定し、表から登録した結果との一致だけで終わらせない。
	want := []monsterStage{
		{117, "MimicDako", "ミミックダコ", 34, 8, 6, 30, "img/mimic_dako.png"},
		{118, "HengeDako", "ヘンゲダコ", 60, 14, 9, 75, "img/henge_dako.png"},
		{119, "SenmenDako", "センメンダコ", 102, 28, 11, 188, "img/senmen_dako.png"},
		{123, "UchikaeshiEbi", "ウチカエシエビ", 30, 7, 6, 24, "img/uchikaeshi_ebi.png"},
		{124, "HajikikaeshiEbi", "ハジキカエシエビ", 54, 12, 9, 60, "img/hajikikaeshi_ebi.png"},
		{125, "ZenbukaeshiEbi", "ゼンブカエシエビ", 92, 24, 11, 150, "img/zenbukaeshi_ebi.png"},
	}
	for _, s := range want {
		d := MonsterDefinitions[s.ID]
		if d.Type != s.Type || d.Name != s.Name || d.Health != s.Health || d.AttackPower != s.Attack || d.DefensePower != s.Defense || d.ExperiencePoints != s.Experience || upperEnemyImagePaths[s.Type] != s.Image {
			t.Fatalf("specification mismatch for %d: %+v", s.ID, d)
		}
	}
	for _, family := range expansionC3MonsterFamilies {
		stages := []monsterStage{{family.Base.ID, family.Base.Type, family.Base.Name, family.Base.Health, family.Base.AttackPower, family.Base.DefensePower, family.Base.ExperiencePoints, family.BaseImage}, family.Second, family.Third}
		for n, stage := range stages {
			d, ok := MonsterDefinitions[stage.ID]
			if !ok || d.Name != stage.Name || d.Type != stage.Type || d.Health != stage.Health || d.MaxHealth != stage.Health || d.AttackPower != stage.Attack || d.DefensePower != stage.Defense || d.ExperiencePoints != stage.Experience || len(d.Traits) != 0 || d.Char != family.Base.Char {
				t.Fatalf("stage %d: %+v", stage.ID, d)
			}
			if stage.ID < 120 {
				if d.Disguise != EnemyDisguiseEnemy || (d.Behavior != nil) != (n > 0) {
					t.Fatalf("mimic definition: %+v", d)
				}
			} else if d.ProjectileReturnPercent != 40+20*n || d.Disguise != EnemyDisguiseNone {
				t.Fatalf("return definition: %+v", d)
			}
			if n < 2 && MonsterLevelUpTable[stage.ID] != stages[n+1].ID {
				t.Fatal("level-up chain")
			}
			spawn := family.Spawns[n]
			for floor := 1; floor <= 31; floor++ {
				weight := 0
				for _, entry := range FloorSpawnTables[floor] {
					if entry.MonsterID == stage.ID {
						weight += entry.Weight
					}
				}
				want := 0
				if floor >= spawn.From && floor <= spawn.To {
					want = 5
				}
				if weight != want {
					t.Fatalf("ID %d floor %d weight %d want %d", stage.ID, floor, weight, want)
				}
			}
			e := CreateEnemyByID(stage.ID, 6, 4)
			e.Health--
			restored := savedToEnemy(enemyToSaved(&e))
			if restored.ID != e.ID || restored.Health != e.Health || restored.AttackPower != e.AttackPower || restored.DefensePower != e.DefensePower || restored.Disguise != e.Disguise || enemyProjectileReturnPercent(restored) != d.ProjectileReturnPercent {
				t.Fatalf("restored: %+v", restored)
			}
		}
	}
}

func TestWave6CMimicSpawnAndMelee(t *testing.T) {
	for floor := 6; floor <= 12; floor++ {
		e := CreateEnemyByID(117, 5, 4)
		prepareSpawnedEnemy(&e, floor, func(int) int { return 0 })
		d := MonsterDefinitions[e.DisguiseID]
		if !enemyMasquerading(e) || d.IsBoss || d.Disguise != EnemyDisguiseNone {
			t.Fatalf("floor %d disguise %+v", floor, e)
		}
		found := false
		for _, entry := range FloorSpawnTables[floor] {
			found = found || entry.MonsterID == e.DisguiseID
		}
		if !found {
			t.Fatal("disguise not on floor")
		}
		g := newProjectileTestGame()
		g.state.Enemies = []Enemy{e}
		g.CheckForEnemies(1, 0)
		runQueuedActions(g)
		if !g.state.Enemies[0].Revealed {
			t.Fatal("melee did not reveal")
		}
	}
}

func TestWave6CRedisguise(t *testing.T) {
	for _, id := range []int{118, 119} {
		g := newFoundationTestGame()
		g.rooms = []Room{{X: 1, Y: 1, Width: 7, Height: 7}}
		g.Floor = 13
		i := addTestEnemy(g, id, 6, 4)
		g.state.Enemies[i].Revealed = true
		for turn := 1; turn <= 9; turn++ {
			if g.runEnemyBehavior(i) {
				t.Fatalf("early redisguise %d", turn)
			}
		}
		g.state.Enemies[i].X = 5
		if g.runEnemyBehavior(i) {
			t.Fatal("redisguised adjacent to player")
		}
		g.state.Enemies[i].X = 6
		if !g.runEnemyBehavior(i) {
			t.Fatal("did not redisguise after adjacency ended")
		}
		messages := runQueuedActions(g)
		if !enemyMasquerading(g.state.Enemies[i]) || g.state.Enemies[i].Timer != 0 || !strings.Contains(messages, MonsterDefinitions[id].Name+"はまた姿を変えた") {
			t.Fatalf("redisguise: %+v %q", g.state.Enemies[i], messages)
		}
		g.state.Enemies[i].Timer = 7
		g.runEnemyBehavior(i)
		e := savedToEnemy(enemyToSaved(&g.state.Enemies[i]))
		if e.Timer != 7 || e.DisguiseID != g.state.Enemies[i].DisguiseID || !enemyMasquerading(e) {
			t.Fatalf("saved disguise %+v", e)
		}
		g.state.Enemies[i].Revealed = true
		g.state.Enemies[i].StatusAilments.Seal = true
		for n := 0; n < 12; n++ {
			if g.runEnemyBehavior(i) {
				t.Fatal("sealed redisguise")
			}
		}
		if g.state.Enemies[i].Timer != 7 {
			t.Fatal("sealed timer advanced")
		}
		g.state.Enemies[i].StatusAilments.Seal = false
		g.state.Enemies[i].Timer = 9
		g.Floor = 31 // 候補がない階では通常行動を続ける。
		if g.runEnemyBehavior(i) || g.state.Enemies[i].Timer != 10 {
			t.Fatal("no-candidate behavior")
		}
	}
	g := newFoundationTestGame()
	i := addTestEnemy(g, 117, 6, 4)
	g.state.Enemies[i].Revealed = true
	for n := 0; n < 20; n++ {
		if g.runEnemyBehavior(i) {
			t.Fatal("base redisguised")
		}
	}
	if g.state.Enemies[i].Timer != 0 {
		t.Fatal("base timer advanced")
	}
	// 隣接しない10回目で化け直す。
	g = newFoundationTestGame()
	g.Floor = 13
	i = addTestEnemy(g, 118, 6, 4)
	g.state.Enemies[i].Revealed = true
	for n := 0; n < 9; n++ {
		g.runEnemyBehavior(i)
	}
	if !g.runEnemyBehavior(i) {
		t.Fatal("10th action")
	}
	runQueuedActions(g)
}

func TestWave6CReturnProjectiles(t *testing.T) {
	for _, id := range []int{123, 124, 125} {
		percent := MonsterDefinitions[id].ProjectileReturnPercent
		for _, roll := range []int{percent - 1, percent} {
			g := newProjectileTestGame()
			g.randHooks.projectile = func(int) int { return roll }
			i := addTestEnemy(g, id, 6, 4)
			hp := g.state.Enemies[i].Health
			food := buildItemFromTemplate(1, 0, 0)
			g.state.Player.Inventory = []Item{food}
			g.throwWithCallbacks(food, 10)
			finishThrow(t, g)
			if roll < percent {
				if g.state.Player.Health >= 40 || g.state.Enemies[i].Health != hp || g.lastDamageSource.Label() != MonsterDefinitions[id].Name+"の打ち返し" {
					t.Fatal("returned throw")
				}
			} else if g.state.Player.Health != 40 || g.state.Enemies[i].Health >= hp {
				t.Fatal("ordinary hit")
			}
		}
		for _, kind := range []string{"arrow", "bolt", "sleep", "seal", "paralysis"} {
			g := newProjectileTestGame()
			i := addTestEnemy(g, id, 6, 4)
			hp := g.state.Enemies[i].Health
			switch kind {
			case "arrow":
				arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
				arrow.ShotCount = 3
				equip(g, arrow)
				g.shootArrow(arrow, 10)
			case "bolt":
				cane := buildItemFromTemplate(16, 0, 0).(*Cane)
				cane.Uses = 3
				g.state.Player.Inventory = []Item{cane}
				g.useCane(cane)
			default:
				if kind == "sleep" {
					g.state.Enemies[i].StatusAilments.Sleep = 3
				}
				if kind == "seal" {
					g.state.Enemies[i].StatusAilments.Seal = true
				}
				if kind == "paralysis" {
					g.state.Enemies[i].StatusAilments.Paralysis = true
				}
				food := buildItemFromTemplate(1, 0, 0)
				g.state.Player.Inventory = []Item{food}
				g.throwWithCallbacks(food, 10)
			}
			messages := finishThrow(t, g)
			if kind == "arrow" || kind == "bolt" {
				if !strings.Contains(messages, "打ち返した") || g.state.Enemies[i].Health != hp || g.state.Enemies[i].StatusAilments.Seal {
					t.Fatalf("%s: %q", kind, messages)
				}
				if kind == "arrow" && g.state.Player.Health >= 40 {
					t.Fatal("arrow damage")
				}
				if kind == "bolt" && g.state.Player.Health != 40 {
					t.Fatal("bolt affected player")
				}
			} else if strings.Contains(messages, "打ち返した") || g.state.Enemies[i].Health >= hp || g.state.Player.Health != 40 {
				t.Fatalf("%s: %q", kind, messages)
			}
		}
	}
}

func TestWave6CMimicReturnerAppearance(t *testing.T) {
	g := newProjectileTestGame()
	g.Floor = 13
	i := addTestEnemy(g, 118, 6, 4)
	// 実際の13階の候補からハジキカエシエビを選ぶ。
	roll := 0
	for _, entry := range FloorSpawnTables[13] {
		d := MonsterDefinitions[entry.MonsterID]
		if d.IsBoss || d.Disguise != EnemyDisguiseNone {
			continue
		}
		if entry.MonsterID == 124 {
			break
		}
		roll += entry.Weight
	}
	prepareSpawnedEnemy(&g.state.Enemies[i], 13, func(int) int { return roll })
	if g.state.Enemies[i].DisguiseID != 124 {
		t.Fatal("returner appearance not selected")
	}
	hp := g.state.Enemies[i].Health
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}
	g.throwWithCallbacks(food, 10)
	messages := finishThrow(t, g)
	if strings.Contains(messages, "打ち返した") || g.state.Player.Health != 40 || g.state.Enemies[i].Health >= hp || !g.state.Enemies[i].Revealed {
		t.Fatalf("mimic: %q", messages)
	}
}

func TestWave6CReturnDeathAndUnseenRedisguise(t *testing.T) {
	g := newProjectileTestGame()
	addTestEnemy(g, 123, 6, 4)
	g.state.Player.Health = 1
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}
	g.throwWithCallbacks(food, 10)
	finishThrow(t, g)
	if g.state.Player.Health > 0 || g.lastDamageSource.Label() != "ウチカエシエビの打ち返し" {
		t.Fatal("returned projectile death cause")
	}
	g = newFoundationTestGame() // 同じ部屋の情報がないためメッセージは表示しない。
	g.Floor = 13
	i := addTestEnemy(g, 118, 6, 4)
	g.state.Enemies[i].Revealed = true
	g.state.Enemies[i].Timer = 9
	if !g.runEnemyBehavior(i) || enemyMasquerading(g.state.Enemies[i]) {
		t.Fatal("redisguise should be queued")
	}
	if messages := runQueuedActions(g); messages != "" || !enemyMasquerading(g.state.Enemies[i]) {
		t.Fatalf("unseen redisguise: %q", messages)
	}
}

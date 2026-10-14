//go:build !test

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestWave4CTablesAndSave(t *testing.T) {
	names := []string{"イヤシナマコ", "ジヒナマコ", "セイボナマコ", "ゼニガタアザラシ", "キンカアザラシ", "センリョウアザラシ", "スナモグリ", "スナカケモグリ", "スナアラシモグリ", "ヨロイエビ", "カブトエビ", "ムシャエビ"}
	types := []string{"IyashiNamako", "JihiNamako", "SeiboNamako", "ZenigataAzarashi", "KinkaAzarashi", "SenryouAzarashi", "Sunamoguri", "SunakakeMoguri", "SunaarashiMoguri", "YoroiEbi", "KabutoEbi", "MushaEbi"}
	stats := [][4]int{{30, 3, 4, 20}, {54, 6, 7, 50}, {92, 12, 9, 125}, {28, 6, 4, 18}, {50, 11, 7, 45}, {85, 22, 9, 112}, {32, 7, 5, 22}, {58, 12, 8, 55}, {98, 24, 10, 138}, {40, 9, 8, 30}, {70, 15, 11, 70}, {119, 30, 13, 175}}
	ranges := [][3]int{{4, 10, 5}, {11, 18, 5}, {19, 30, 5}, {3, 8, 6}, {9, 16, 6}, {17, 30, 6}, {5, 11, 7}, {12, 19, 7}, {20, 30, 7}, {6, 12, 7}, {13, 20, 7}, {21, 30, 7}}
	for k, want := range stats {
		id := 78 + k
		d, ok := MonsterDefinitions[id]
		if !ok || d.Name != names[k] || d.Type != types[k] || [4]int{d.Health, d.AttackPower, d.DefensePower, d.ExperiencePoints} != want {
			t.Fatalf("definition %d: %+v", id, d)
		}
		shell := id >= 84
		if (len(d.Traits) == 1 && d.Traits[0] == EnemyTraitShell) != shell {
			t.Fatalf("traits %d: %v", id, d.Traits)
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
				expected = ranges[k][2]
			}
			if weight != expected {
				t.Fatalf("spawn %d floor %d: %d", id, floor, weight)
			}
		}
		e := CreateEnemyByID(id, 2, 2)
		e.Health--
		saved := enemyToSaved(&e)
		restored := savedToEnemy(saved)
		if restored.ID != id || restored.Health != e.Health {
			t.Fatalf("save %d", id)
		}
	}
}

func TestWave4CNamako(t *testing.T) {
	for _, id := range []int{78, 79, 80} {
		g := newFoundationTestGame()
		addTestEnemy(g, id, 5, 4)
		j := addTestEnemy(g, 0, 6, 4)
		g.state.Enemies[j].Health = 1
		g.actEnemy(0)
		if g.state.Enemies[j].Health != 1 {
			t.Fatal("heal was immediate")
		}
		runQueuedActions(g)
		if g.state.Enemies[j].Health != min(g.state.Enemies[j].MaxHealth, 1+[]int{15, 30, 60}[id-78]) || g.state.Player.Health != 40 {
			t.Fatal("healing or nonattack")
		}
	}
	g := newFoundationTestGame()
	addTestEnemy(g, 78, 1, 1)
	j := addTestEnemy(g, 0, 3, 3)
	g.state.Enemies[j].Health = 1
	g.actEnemy(0)
	if g.state.Enemies[0].X != 2 || g.state.Enemies[0].Y != 2 {
		t.Fatal("did not approach")
	}
	g = newFoundationTestGame()
	addTestEnemy(g, 78, 5, 4)
	j = addTestEnemy(g, 42, 6, 4)
	g.state.Enemies[j].Health = 1
	g.actEnemy(0)
	runQueuedActions(g)
	if g.state.Enemies[j].Health != 1 || g.state.Player.Health != 40 {
		t.Fatal("boss healed or player attacked")
	}
	g = newFoundationTestGame()
	addTestEnemy(g, 78, 5, 4)
	g.state.Enemies[0].StatusAilments.Seal = true
	g.actEnemy(0)
	if !strings.Contains(runQueuedActions(g), "ダメージ") {
		t.Fatal("sealed healer did not attack")
	}
	// 回復対象がキュー実行前に消えても、別の敵を回復しない。
	g = newFoundationTestGame()
	addTestEnemy(g, 78, 5, 4)
	j = addTestEnemy(g, 0, 6, 4)
	g.state.Enemies[j].Health = 1
	g.actEnemy(0)
	g.state.Enemies = g.state.Enemies[:1]
	runQueuedActions(g)
}

func TestWave4CCashTheft(t *testing.T) {
	for k, cap := range []int{200, 500, 1000} {
		g := newFoundationTestGame()
		g.rooms = []Room{{X: 3, Y: 3, Width: 3, Height: 3}}
		g.state.Player.Cash = 1500
		addTestEnemy(g, 81+k, 5, 4)
		sealCashAttack(&g.state.Enemies[0], g)
		if g.state.Player.Cash != 1500 {
			t.Fatal("immediate theft")
		}
		runQueuedActions(g)
		e := &g.state.Enemies[0]
		if g.state.Player.Cash != 1500-cap || !e.Fleeing || g.IsInPlayerVisibleRoom(e.X, e.Y) || e.Animating {
			t.Fatalf("theft: %+v", e)
		}
		restored := savedToEnemy(enemyToSaved(e))
		if !restored.Fleeing || restored.HeldItem.(*Money).Amount != cap {
			t.Fatal("lost stolen cash")
		}
		g.applyDamageToEnemy(0, 9999)
		if len(g.state.Items) != 1 || g.state.Items[0].(*Money).Amount != cap {
			t.Fatal("cash not dropped")
		}
		g.PickUpItem(g.state.Items[0], 0)
		if g.state.Player.Cash != 1500 {
			t.Fatal("cash not returned")
		}
	}
	g := newFoundationTestGame()
	g.state.Player.Cash = 50
	addTestEnemy(g, 81, 5, 4)
	equip(g, testArmor(AbilityTheftResistance))
	sealCashAttack(&g.state.Enemies[0], g)
	runQueuedActions(g)
	if g.state.Player.Cash != 50 || g.state.Enemies[0].HeldItem != nil {
		t.Fatal("shield did not protect")
	}
	for _, held := range []bool{false, true} {
		g = newFoundationTestGame()
		addTestEnemy(g, 81, 5, 4)
		if held {
			g.state.Player.Cash = 50
			g.state.Enemies[0].HeldItem = newStolenCash(10)
		}
		sealCashAttack(&g.state.Enemies[0], g)
		if !strings.Contains(runQueuedActions(g), "ダメージ") {
			t.Fatal("no normal attack fallback")
		}
	}
	g = newFoundationTestGame()
	g.rooms = []Room{{X: 0, Y: 0, Width: 9, Height: 9}}
	g.state.Player.Cash = 50
	addTestEnemy(g, 81, 5, 4)
	sealCashAttack(&g.state.Enemies[0], g)
	runQueuedActions(g)
	if g.state.Player.Cash != 0 || g.state.Enemies[0].X != 5 || g.state.Enemies[0].HeldItem.(*Money).Amount != 50 {
		t.Fatal("no warp candidate theft")
	}
}

func TestWave4CSand(t *testing.T) {
	// 同一乱数系列の最初の判定が30未満/以上になるシードを使い、両境界側を通す。
	for _, roll := range []bool{true, false} {
		for seed := int64(1); ; seed++ {
			rng := newTurnRNG(seed)
			if (rng.Intn(100) < 30) != roll {
				continue
			}
			g := newFoundationTestGame()
			g.rng = newTurnRNG(seed)
			g.state.Player.DefensePower = 100
			addTestEnemy(g, 84, 5, 4)
			sandMoguriAttack(&g.state.Enemies[0], g)
			runQueuedActions(g)
			blind := 0
			if roll {
				blind = 5
			}
			if g.state.Player.Health != 34 || g.state.Player.StatusAilments.Blind != blind {
				t.Fatal("sand damage or blindness")
			}
			if roll {
				g = newFoundationTestGame()
				g.rng = newTurnRNG(seed)
				withTestImmunity(t, "sand_guard", StatusBlind)
				equip(g, testRing("sand_guard"))
				addTestEnemy(g, 84, 5, 4)
				sandMoguriAttack(&g.state.Enemies[0], g)
				runQueuedActions(g)
				if g.state.Player.StatusAilments.Blind != 0 {
					t.Fatal("immunity failed")
				}
			}
			break
		}
	}
	for _, evaded := range []bool{false, true} {
		g := newFoundationTestGame()
		addTestEnemy(g, 84, 5, 4)
		value := 15
		if evaded {
			value = 14
		}
		g.randHooks.enemyEvasion = func(int) int { return value }
		g.CheckForEnemies(1, 0)
		runQueuedActions(g)
		if (g.state.Enemies[0].Health == 32) != evaded {
			t.Fatal("evasion boundary")
		}
	}
	withTestAbilityHooks(t, "sand_evade", AbilityHooks{ModifyIncomingDamage: func(_ *Game, d *IncomingDamage, _ int) { d.Evaded = true }})
	g := newFoundationTestGame()
	equip(g, testRing("sand_evade"))
	addTestEnemy(g, 84, 5, 4)
	sandMoguriAttack(&g.state.Enemies[0], g)
	runQueuedActions(g)
	if g.state.Player.Health != 40 || g.state.Player.StatusAilments.Blind != 0 {
		t.Fatal("evaded sand had effect")
	}
}

func TestWave4CArmoredSpirit(t *testing.T) {
	for k, spirit := range []int{11, 29, 56} {
		for _, mode := range []string{"normal", "sealed", "genocide", "enemy"} {
			g := newFoundationTestGame()
			addTestEnemy(g, 87+k, 2, 2)
			if mode == "sealed" {
				g.state.Enemies[0].StatusAilments.Seal = true
			}
			if mode == "genocide" {
				g.genocidedMonsterIDs = map[int]bool{spirit: true}
			}
			if mode == "enemy" {
				addTestEnemy(g, 0, 3, 2)
				g.defeatEnemyByEnemy(1, 0)
			} else {
				g.applyDamageToEnemy(0, 9999)
			}
			g.processPendingEnemySpawns()
			g.processPendingEnemySpawns()
			for _, e := range g.state.Enemies {
				if e.ID == spirit {
					t.Fatal("early spawn")
				}
			}
			g.processPendingEnemySpawns()
			found := false
			for _, e := range g.state.Enemies {
				if e.ID == spirit {
					found = true
					if max(abs(e.X-2), abs(e.Y-2)) > 2 {
						t.Fatal("distant spawn")
					}
				}
			}
			if found != (mode != "sealed" && mode != "genocide") {
				t.Fatalf("spawn %d mode %s", spirit, mode)
			}
		}
	}
	g, _ := newHeadlessGame(77, DifficultyAdvanced)
	g.state.Enemies = nil
	addTestEnemy(g, 87, 2, 2)
	g.applyDamageToEnemy(0, 9999)
	data, err := json.Marshal(g.buildSaveData())
	if err != nil {
		t.Fatal(err)
	}
	save, err := decodeSaveData(data)
	if err != nil {
		t.Fatal(err)
	}
	restored := &Game{}
	if err = restored.applySaveData(save); err != nil {
		t.Fatal(err)
	}
	if len(restored.pendingEnemySpawns) != 1 || restored.pendingEnemySpawns[0].Turns != 3 {
		t.Fatal("reservation save")
	}
	g.fadingOut, g.fadeAlpha, g.frameCounter = true, 1, 0
	g.handleFadingOut()
	if len(g.pendingEnemySpawns) != 0 {
		t.Fatal("reservation survived floor")
	}
}

func TestWave4CStatusAndUpperSand(t *testing.T) {
	for k, damage := range []int{6, 10, 18} {
		g := newFoundationTestGame()
		addTestEnemy(g, 84+k, 5, 4)
		g.state.Player.DefensePower = 1000
		sandMoguriAttack(&g.state.Enemies[0], g)
		runQueuedActions(g)
		if g.state.Player.Health != 40-damage {
			t.Fatal("upper sand damage")
		}
	}
	for _, status := range []string{"seal", "sleep", "paralysis"} {
		g := newFoundationTestGame()
		addTestEnemy(g, 84, 5, 4)
		e := &g.state.Enemies[0]
		switch status {
		case "seal":
			e.StatusAilments.Seal = true
		case "sleep":
			e.StatusAilments.Sleep = 3
		case "paralysis":
			e.StatusAilments.Paralysis = true
		}
		g.randHooks.enemyEvasion = func(int) int { return 0 }
		g.CheckForEnemies(1, 0)
		runQueuedActions(g)
		if g.state.Enemies[0].Health == 32 {
			t.Fatalf("inactive evasion: %s", status)
		}
	}
	for _, id := range []int{78, 81, 84} {
		g := newFoundationTestGame()
		g.state.Player.Cash = 200
		addTestEnemy(g, id, 5, 4)
		g.state.Enemies[0].StatusAilments.Seal = true
		g.actEnemy(0)
		runQueuedActions(g)
		if g.state.Player.Cash != 200 || g.state.Player.StatusAilments.Blind != 0 || g.state.Player.Health == 40 {
			t.Fatalf("sealed normal action %d", id)
		}
	}
	for _, id := range []int{78, 79, 80} {
		g := newFoundationTestGame()
		addTestEnemy(g, id, 5, 4)
		j := addTestEnemy(g, 89, 6, 4)
		g.state.Enemies[j].Health = 1
		g.actEnemy(0)
		runQueuedActions(g)
		if g.state.Enemies[j].Health != 1+[]int{15, 30, 60}[id-78] {
			t.Fatal("healing amount")
		}
		g.state.Enemies[j].Health = g.state.Enemies[j].MaxHealth - 1
		g.actEnemy(0)
		runQueuedActions(g)
		if g.state.Enemies[j].Health != g.state.Enemies[j].MaxHealth {
			t.Fatal("overheal")
		}
	}
	g := newFoundationTestGame()
	g.rooms = []Room{{X: 3, Y: 3, Width: 3, Height: 3}}
	g.state.Player.Cash = 100
	addTestEnemy(g, 81, 5, 4)
	sealCashAttack(&g.state.Enemies[0], g)
	runQueuedActions(g)
	e := g.state.Enemies[0]
	g.moveFleeingEnemy(0)
	if !g.state.Enemies[0].Fleeing || g.state.Enemies[0].HeldItem == nil {
		t.Fatal("fleeing lost cash")
	}
	data, err := json.Marshal(enemyToSaved(&e))
	if err != nil {
		t.Fatal(err)
	}
	var saved SavedEnemy
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	restored := savedToEnemy(saved)
	if restored.HeldItem.(*Money).Amount != 100 {
		t.Fatal("json cash save")
	}
}

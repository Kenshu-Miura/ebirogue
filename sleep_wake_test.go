//go:build !test

package main

import (
	"bytes"
	"testing"
)

func TestSleepingRoomStayDoesNotReroll(t *testing.T) {
	g := newFoundationTestGame()
	g.rooms = []Room{{ID: 1, X: 0, Y: 0, Width: 9, Height: 9}}
	g.state.Enemies = []Enemy{{Entity: Entity{X: 6, Y: 6}, Health: 100, StatusAilments: StatusAilments{Sleep: -1}}}
	// 入室抽選が外れるシードを選び、その後は乱数が消費されないことも確認する。
	for seed := int64(1); ; seed++ {
		g.rng = newTurnRNG(seed)
		if g.random().Float64() >= 0.5 {
			g.rng = newTurnRNG(seed)
			break
		}
	}
	g.CheckSleepingEnemyWakeUp()
	other := addTestEnemy(g, 0, 5, 4)
	g.state.Enemies[other].Health = 100
	before, _ := g.rng.MarshalState()
	for i := 0; i < 20; i++ {
		g.applyDamageToEnemy(other, 1) // 同室の別の敵を攻撃しても入室抽選は増えない。
		g.CheckSleepingEnemyWakeUp()
	}
	after, _ := g.rng.MarshalState()
	if !bytes.Equal(before, after) {
		t.Fatal("staying in room consumed wake randomness")
	}
	if g.state.Enemies[0].StatusAilments.Sleep != -1 {
		t.Fatal("staying in room woke sleeping enemy")
	}
}

func TestSleepingFloorArrivalRollsOnce(t *testing.T) {
	g := newFoundationTestGame()
	g.rooms = []Room{{ID: 0, X: 0, Y: 0, Width: 9, Height: 9}}
	i := addTestEnemy(g, 0, 6, 6)
	g.state.Enemies[i].StatusAilments.Sleep = -1
	seed := int64(1)
	for newTurnRNG(seed).Float64() >= 0.5 {
		seed++
	}
	g.rng = newTurnRNG(seed)
	g.wakeSleepingEnemiesOnArrival()
	if g.state.Enemies[i].StatusAilments.Sleep != 0 {
		t.Fatal("arrival did not roll")
	}
	before, _ := g.rng.MarshalState()
	g.CheckSleepingEnemyWakeUp()
	after, _ := g.rng.MarshalState()
	if !bytes.Equal(before, after) {
		t.Fatal("arrival rolled again on first turn")
	}
}

func TestSleepingReentryAndRoomChange(t *testing.T) {
	g := newFoundationTestGame()
	g.rooms = []Room{{ID: 0, X: 0, Y: 0, Width: 9, Height: 9}, {ID: 1, X: 10, Y: 0, Width: 9, Height: 9}}
	g.state.Enemies = []Enemy{{Entity: Entity{X: 6, Y: 6}, Health: 100, StatusAilments: StatusAilments{Sleep: -1}}, {Entity: Entity{X: 16, Y: 6}, Health: 100, StatusAilments: StatusAilments{Sleep: -1}}}
	g.previousPlayerRoomID = 0
	g.CheckSleepingEnemyWakeUp()
	g.state.Player.X = 9
	g.CheckSleepingEnemyWakeUp()
	if g.previousPlayerRoomID != -1 {
		t.Fatal("corridor must be -1")
	}
	hitSeed := int64(1)
	for newTurnRNG(hitSeed).Float64() >= 0.5 {
		hitSeed++
	}
	g.rng = newTurnRNG(hitSeed)
	g.state.Player.X = 4
	g.CheckSleepingEnemyWakeUp()
	if g.state.Enemies[0].StatusAilments.Sleep != 0 {
		t.Fatal("reentry did not roll")
	}
	g.rng = newTurnRNG(hitSeed)
	g.state.Player.X = 14
	g.CheckSleepingEnemyWakeUp()
	if g.state.Enemies[1].StatusAilments.Sleep != 0 {
		t.Fatal("room A to B did not roll")
	}
	if g.roomIDAt(10, 4) != -1 {
		t.Fatal("wall counted as room")
	}
}

func TestSleepingProximityStillRerolls(t *testing.T) {
	g := newFoundationTestGame()
	i := addTestEnemy(g, 0, 5, 4)
	g.state.Enemies[i].StatusAilments.Sleep = -1
	missSeed := int64(1)
	for newTurnRNG(missSeed).Float64() < 0.5 {
		missSeed++
	}
	g.rng = newTurnRNG(missSeed)
	g.WakeUpSleepingEnemyByProximity(i)
	if g.state.Enemies[i].StatusAilments.Sleep != -1 {
		t.Fatal("failed proximity roll woke enemy")
	}
	hitSeed := int64(1)
	for newTurnRNG(hitSeed).Float64() >= 0.5 {
		hitSeed++
	}
	g.rng = newTurnRNG(hitSeed)
	g.WakeUpSleepingEnemyByProximity(i)
	if g.state.Enemies[i].StatusAilments.Sleep != 0 {
		t.Fatal("second adjacent roll did not wake enemy")
	}
}

func TestSleepingPlayerAttackPaths(t *testing.T) {
	for _, path := range []string{"melee", "throw", "bolt", "card"} {
		t.Run(path, func(t *testing.T) {
			g := newProjectileTestGame()
			x := 6
			if path == "melee" || path == "card" {
				x = 5
			}
			i := addTestEnemy(g, 0, x, 4)
			g.state.Enemies[i].Health = 100
			g.state.Enemies[i].StatusAilments.Sleep = -1
			switch path {
			case "melee":
				g.CheckForEnemies(1, 0)
				runQueuedActions(g)
			case "throw":
				food := buildItemFromTemplate(1, 0, 0)
				g.state.Player.Inventory = []Item{food}
				g.throwWithCallbacks(food, 10)
				finishThrow(t, g)
			case "bolt":
				// 現在の杖は非ダメージ効果だけ。魔法弾の実経路にダメージ効果を差し込んで検証する。
				cane := buildItemFromTemplate(16, 0, 0).(*Cane)
				cane.Name = "テストのダメージ杖"
				cane.Uses = 1
				cane.UseActions = map[string]UseAction{"CaneEffect": func(g *Game) {
					index := g.TargetEnemyIndex
					g.Enqueue(Action{Execute: func(g *Game) { g.applyDamageToEnemy(index, 1) }})
				}}
				g.state.Player.Inventory = []Item{cane}
				g.useCane(cane)
				finishThrow(t, g)
			case "card":
				card := buildItemFromTemplate(7, 0, 0)
				g.state.Player.Inventory = []Item{card}
				g.useConsumable(card)
				finishThrow(t, g)
			}
			if g.state.Enemies[i].StatusAilments.Sleep != 0 {
				t.Fatal("player attack did not wake nap")
			}
		})
	}
}

func TestSleepingResumeDoesNotRoll(t *testing.T) {
	g, _ := newHeadlessGame(10, DifficultyAdvanced)
	i := addTestEnemy(g, 0, g.state.Player.X+2, g.state.Player.Y)
	g.state.Enemies[i].StatusAilments.Sleep = -1
	saved := g.buildSaveData()
	resumed, _ := newHeadlessGame(20, DifficultyAdvanced)
	if err := resumed.applySaveData(saved); err != nil {
		t.Fatal(err)
	}
	before, _ := resumed.rng.MarshalState()
	resumed.CheckSleepingEnemyWakeUp()
	after, _ := resumed.rng.MarshalState()
	if !bytes.Equal(before, after) || resumed.state.Enemies[i].StatusAilments.Sleep != -1 {
		t.Fatal("resume rerolled room wake")
	}
}

func TestSleepingDamageWakesOnlyNap(t *testing.T) {
	for _, sleep := range []int{-1, 5} {
		g := newFoundationTestGame()
		g.state.Enemies = []Enemy{{Health: 100, StatusAilments: StatusAilments{Sleep: sleep}}}
		g.applyDamageToEnemy(0, 1)
		want := sleep
		if sleep == -1 {
			want = 0
		}
		if got := g.state.Enemies[0].StatusAilments.Sleep; got != want {
			t.Fatalf("sleep=%d got %d want %d", sleep, got, want)
		}
	}
}

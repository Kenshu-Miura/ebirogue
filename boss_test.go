//go:build !test

package main

import (
	"strings"
	"testing"
)

// moveToStairs はプレイヤーを階段の上へ移動させる。
func moveToStairs(t *testing.T, g *Game) {
	t.Helper()
	for y, row := range g.state.Map {
		for x, tile := range row {
			if tile.Type == "stairs" {
				g.state.Player.X, g.state.Player.Y = x, y
				return
			}
		}
	}
	t.Fatal("no stairs on this floor")
}

// descend は階段を降りて次の階へ進む。
func descend(t *testing.T, g *Game, v *VirtualInput) {
	t.Helper()
	floor := g.Floor
	moveToStairs(t, g)
	if err := g.RunUntilInputReady(100); err != nil {
		t.Fatal(err)
	}
	if err := g.StepFrame(); err != nil {
		t.Fatal(err)
	}
	if !g.showStairsPrompt {
		t.Fatalf("stairs prompt did not open on floor %d", floor)
	}
	tapHeadless(t, g, v, KeyZ)
	if g.Floor != floor+1 {
		t.Fatalf("floor = %d, want %d", g.Floor, floor+1)
	}
}

func bossIndex(g *Game) int {
	for i, enemy := range g.state.Enemies {
		if MonsterDefinitions[enemy.ID].IsBoss {
			return i
		}
	}
	return -1
}

// gotoBossFloor は floor-1 階から階段で floor 階（ボス階）へ移動する。
func gotoBossFloor(t *testing.T, difficulty Difficulty, floor int) (*Game, *VirtualInput) {
	t.Helper()
	g, v := newHeadlessGame(17, difficulty)
	g.Floor = floor - 1
	descend(t, g, v)
	if i := bossIndex(g); i < 0 {
		t.Fatalf("no boss on floor %d", floor)
	} else if want, _ := bossIDForFloor(floor); g.state.Enemies[i].ID != want {
		t.Fatalf("boss ID = %d, want %d", g.state.Enemies[i].ID, want)
	}
	return g, v
}

func defeatBoss(t *testing.T, g *Game) {
	t.Helper()
	i := bossIndex(g)
	g.applyDamageToEnemy(i, g.state.Enemies[i].Health)
	if err := g.RunUntilInputReady(600); err != nil {
		t.Fatal(err)
	}
}

func TestBossFloorLocksStairsUntilBossDefeated(t *testing.T) {
	g, v := gotoBossFloor(t, DifficultyAdvanced, 10)
	moveToStairs(t, g)
	for i := 0; i < 3; i++ {
		if err := g.StepFrame(); err != nil {
			t.Fatal(err)
		}
	}
	if g.showStairsPrompt {
		t.Fatal("stairs must stay locked while the boss is alive")
	}
	if !strings.Contains(strings.Join(g.messageLog.Messages, "\n"), "ボスを倒すまで階段は使えない") {
		t.Fatal("locked stairs message was not shown")
	}

	defeatBoss(t, g)
	if g.RunEnded() || g.cleared || !g.bossDefeated {
		t.Fatal("an earlier boss must unlock the stairs without clearing the run")
	}
	g.ignoreStairs = false // 階段から一度離れたことにする
	descend(t, g, v)
	if bossIndex(g) >= 0 {
		t.Fatal("floor 11 should not have a boss")
	}
}

func TestDefeatingGoalBossClearsRun(t *testing.T) {
	for _, tt := range []struct {
		difficulty Difficulty
		floor      int
	}{
		{DifficultyBeginner, 10},
		{DifficultyIntermediate, 20},
		{DifficultyAdvanced, 30},
	} {
		g, _ := gotoBossFloor(t, tt.difficulty, tt.floor)
		defeatBoss(t, g)
		result := g.LastRunResult()
		if !g.RunEnded() || !g.cleared || result == nil || !result.Cleared {
			t.Fatalf("%v: defeating the floor %d boss should clear the run (result %+v)", tt.difficulty, tt.floor, result)
		}
		if result.Difficulty != tt.difficulty || result.Floor != tt.floor || result.DeathMessage != "" {
			t.Fatalf("%v: result = %+v", tt.difficulty, result)
		}
	}
}

func TestIntermediateContinuesPastFirstBoss(t *testing.T) {
	g, _ := gotoBossFloor(t, DifficultyIntermediate, 10)
	defeatBoss(t, g)
	if g.RunEnded() || g.LastRunResult() != nil {
		t.Fatal("intermediate run must continue after the floor 10 boss")
	}
}

func TestCannotDescendBelowFinalFloor(t *testing.T) {
	g, _ := gotoBossFloor(t, DifficultyAdvanced, 30)
	g.difficulty = DifficultyAdvanced
	g.bossDefeated = true // ボスを倒した後でも30階より下へは進めない
	moveToStairs(t, g)
	for i := 0; i < 3; i++ {
		if err := g.StepFrame(); err != nil {
			t.Fatal(err)
		}
	}
	if g.showStairsPrompt || g.fadingOut {
		t.Fatal("must not descend below floor 30")
	}
}

func TestBossesOnlyAppearOnBossFloors(t *testing.T) {
	for floor := 1; floor <= finalFloor; floor++ {
		player := newInitialPlayer()
		_, enemies, _, _, _, _ := GenerateRandomMap(70, 70, floor-1, &player, newFloorRNG(99, floor))
		bosses := 0
		for _, enemy := range enemies {
			if MonsterDefinitions[enemy.ID].IsBoss && !isShopMonsterID(enemy.ID) { // 店主はボスと同じ例外だがボスではない
				bosses++
				if enemy.StatusAilments.Sleep != 0 {
					t.Fatalf("floor %d boss should be awake", floor)
				}
			}
		}
		if want := map[bool]int{true: 1, false: 0}[isBossFloor(floor)]; bosses != want {
			t.Fatalf("floor %d has %d bosses, want %d", floor, bosses, want)
		}
	}
	for _, id := range []int{42, 43, 44} {
		definition := MonsterDefinitions[id]
		if !definition.IsBoss {
			t.Fatalf("monster %d should be a boss", id)
		}
		if _, ok := MonsterLevelUpTable[id]; ok {
			t.Fatalf("boss %d must not level up", id)
		}
		for _, entries := range FloorSpawnTables {
			for _, entry := range entries {
				if entry.MonsterID == id {
					t.Fatalf("boss %d must not be in spawn tables", id)
				}
			}
		}
	}
}

func TestBossLevelUpIsBlocked(t *testing.T) {
	g := &Game{state: GameState{Enemies: []Enemy{CreateEnemyByID(42, 1, 1)}}}
	if g.levelUpEnemy(0) {
		t.Fatal("boss must not level up")
	}
}

func TestSaveKeepsDifficultyAndBossState(t *testing.T) {
	g, _ := gotoBossFloor(t, DifficultyIntermediate, 10)
	defeatBoss(t, g)
	restored := &Game{}
	if err := restored.applySaveData(g.buildSaveData()); err != nil {
		t.Fatal(err)
	}
	if restored.difficulty != DifficultyIntermediate || !restored.bossDefeated || restored.Floor != 10 {
		t.Fatalf("restored difficulty %v, boss defeated %v, floor %d", restored.difficulty, restored.bossDefeated, restored.Floor)
	}
}

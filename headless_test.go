//go:build !test

package main

import (
	"testing"
	"time"
)

// tapHeadless はキーを1フレームだけ押して離し、入力待ちに戻るまで進める。
func tapHeadless(t *testing.T, g *Game, v *VirtualInput, keys ...Key) {
	t.Helper()
	v.Press(keys...)
	if err := g.StepFrame(); err != nil {
		t.Fatal(err)
	}
	v.Release(keys...)
	if err := g.RunUntilInputReady(600); err != nil {
		t.Fatal(err)
	}
}

func TestHeadlessRunsThousandTurnsQuickly(t *testing.T) {
	start := time.Now()
	g, v := newHeadlessGame(7, DifficultyAdvanced)
	// 時間内に1000ターン歩き回れるよう、死なない程度に強くしておく
	g.state.Player.MaxHealth, g.state.Player.Health = 1_000_000, 1_000_000
	if err := g.RunUntilInputReady(600); err != nil {
		t.Fatal(err)
	}
	directions := []Key{KeyUp, KeyRight, KeyDown, KeyLeft}
	dir := 0
	for steps := 0; g.turnCount < 1000; steps++ {
		if steps > 20000 {
			t.Fatalf("too many inputs: turn %d", g.turnCount)
		}
		if g.RunEnded() {
			t.Fatalf("run ended unexpectedly at turn %d", g.turnCount)
		}
		if g.showStairsPrompt {
			tapHeadless(t, g, v, KeyX) // 階段は降りずに歩き続ける
			continue
		}
		x, y := g.state.Player.X, g.state.Player.Y
		turn := g.turnCount
		tapHeadless(t, g, v, directions[dir])
		if g.state.Player.X == x && g.state.Player.Y == y && g.turnCount == turn {
			dir = g.randIntn(len(directions)) // 壁にぶつかったら向きを変える
		}
		if g.randIntn(8) == 0 {
			tapHeadless(t, g, v, KeyZ) // ときどき正面を攻撃する
		}
	}
	if elapsed := time.Since(start); elapsed > 10*time.Second {
		t.Fatalf("1000 turns took %v", elapsed)
	} else {
		t.Logf("1000 turns in %v (frames: %d)", elapsed, g.inputFrame)
	}
}

func TestHeadlessDeathEndsRunWithoutReset(t *testing.T) {
	g, _ := newHeadlessGame(3, DifficultyBeginner)
	g.state.Player.Health = 0
	g.state.Player.checkDeath(g)
	if err := g.RunUntilInputReady(600); err != nil {
		t.Fatal(err)
	}
	if !g.RunEnded() {
		t.Fatal("death should end the headless run")
	}
	if g.state.Player.Health > 0 || g.seed != 3 {
		t.Fatal("headless death must not restart the run")
	}
}

func TestHeadlessStairsUseNoRealTime(t *testing.T) {
	g, v := newHeadlessGame(11, DifficultyAdvanced)
	// 階段の上に立たせて降りる
	for y, row := range g.state.Map {
		for x, tile := range row {
			if tile.Type == "stairs" {
				g.state.Player.X, g.state.Player.Y = x, y
			}
		}
	}
	if err := g.RunUntilInputReady(10); err != nil {
		t.Fatal(err)
	}
	if err := g.StepFrame(); err != nil { // 階段の確認が開く
		t.Fatal(err)
	}
	if !g.showStairsPrompt {
		t.Fatal("stairs prompt should open")
	}
	tapHeadless(t, g, v, KeyZ)
	if g.Floor != 2 {
		t.Fatalf("floor = %d, want 2", g.Floor)
	}
}

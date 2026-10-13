//go:build !test

package main

import "testing"

// newScenarioGame は 12x5 の部屋だけのフロアにプレイヤーと眠った敵を置いた、仮想入力のゲームを作る。
func newScenarioGame(t *testing.T, input InputSource) *Game {
	t.Helper()
	g := &Game{input: input, customNames: map[int]string{}}
	g.startNewRun(1)
	g.state.Map = makeTestFloorMap(12, 5)
	g.rooms = []Room{{ID: 0, X: 0, Y: 0, Width: 12, Height: 5}}
	g.state.Player.X, g.state.Player.Y = 2, 2
	g.state.Player.Direction = Right
	enemy := CreateEnemyByID(0, 6, 2)
	enemy.StatusAilments.Sleep = 50 // 通常の睡眠は近づいても起きない
	g.state.Enemies = []Enemy{enemy}
	g.state.Items = nil
	g.state.MapTraps = nil
	g.fadeInProgress = 1.0
	g.randHooks.attackHit = func(int) int { return 0 } // 必ず命中させる
	return g
}

// runUntilIdle は入力を受け付けられる状態になるまでフレームを進める。
func runUntilIdle(t *testing.T, g *Game, maxFrames int) {
	t.Helper()
	for i := 0; i < maxFrames; i++ {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		if g.isInputReady() {
			return
		}
	}
	t.Fatalf("game did not become idle within %d frames", maxFrames)
}

// tapKey はキーを1フレームだけ押して離す。
func tapKey(t *testing.T, g *Game, v *VirtualInput, k Key) {
	t.Helper()
	v.Press(k)
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	v.Release(k)
	runUntilIdle(t, g, 600)
}

func TestVirtualInputMoveThreeStepsAndAttack(t *testing.T) {
	v := NewVirtualInput()
	recorder := NewInputRecorder(v)
	g := newScenarioGame(t, recorder)

	for i := 0; i < 3; i++ {
		tapKey(t, g, v, KeyRight)
	}
	if g.state.Player.X != 5 || g.state.Player.Y != 2 {
		t.Fatalf("player at (%d, %d), want (5, 2)", g.state.Player.X, g.state.Player.Y)
	}
	if g.turnCount != 3 {
		t.Fatalf("turnCount = %d, want 3", g.turnCount)
	}

	before := g.state.Enemies[0].Health
	tapKey(t, g, v, KeyZ)
	if g.state.Player.X != 5 {
		t.Fatal("attacking must not move the player")
	}
	if len(g.state.Enemies) > 0 && g.state.Enemies[0].Health >= before {
		t.Fatalf("enemy HP %d, want damage from %d", g.state.Enemies[0].Health, before)
	}
	if g.turnCount != 4 {
		t.Fatalf("turnCount after attack = %d, want 4", g.turnCount)
	}

	// 同じシナリオを記録した入力で再生すると同じ結果になる
	events := recorder.Events()
	frames := g.inputFrame
	replay := newScenarioGame(t, NewReplayInput(events))
	for i := 0; i < frames; i++ {
		if err := replay.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if replay.state.Player.X != g.state.Player.X || replay.turnCount != g.turnCount ||
		len(replay.state.Enemies) != len(g.state.Enemies) ||
		(len(g.state.Enemies) > 0 && replay.state.Enemies[0].Health != g.state.Enemies[0].Health) {
		t.Fatal("replaying the recorded input produced a different result")
	}
}

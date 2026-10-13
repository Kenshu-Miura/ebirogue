//go:build !test

package main

import "testing"

// ターン進行の一本化: 行動を消費する操作はどれも CheckCombatState で1回だけ AdvanceTurn を呼び、
// turnCount と floorTurns がちょうど1増える。行動を消費しない操作では増えない。

// newTurnTestGame は 9x7 の部屋にプレイヤーだけを置いたヘッドレスゲームと、操作用の BOT を作る。
func newTurnTestGame(t *testing.T) (*Game, *bot) {
	t.Helper()
	g, in := newHeadlessGame(3, DifficultyAdvanced)
	g.state.Map = makeTestFloorMap(9, 7)
	g.rooms = []Room{{ID: 0, X: 0, Y: 0, Width: 9, Height: 7, Center: Coordinate{X: 4, Y: 3}}}
	g.state.Enemies, g.state.Items, g.state.MapTraps = nil, nil, nil
	g.state.Player.X, g.state.Player.Y, g.state.Player.Direction = 4, 3, Right
	g.randHooks = randHooks{attackHit: func(int) int { return 0 }, damage: func(n int) int { return 0 }}
	if err := g.RunUntilInputReady(100); err != nil {
		t.Fatal(err)
	}
	return g, newBot(g, in, personaAggressive, 3)
}

// expectTurns は op の前後で turnCount と floorTurns が want だけ増えたことを確かめる。
func expectTurns(t *testing.T, g *Game, want int, op func() error) {
	t.Helper()
	turn, floorTurns := g.turnCount, g.floorTurns
	if err := op(); err != nil {
		t.Fatal(err)
	}
	if err := g.RunUntilInputReady(2000); err != nil {
		t.Fatal(err)
	}
	if got := g.turnCount - turn; got != want {
		t.Fatalf("turnCount increased by %d, want %d", got, want)
	}
	if got := g.floorTurns - floorTurns; got != want {
		t.Fatalf("floorTurns increased by %d, want %d", got, want)
	}
}

func TestTurnAdvancesOncePerAction(t *testing.T) {
	tests := []struct {
		name  string
		setup func(g *Game)
		op    func(g *Game, b *bot) error
	}{
		{"move", nil, func(g *Game, b *bot) error { return b.move(1, 0) }},
		{"attack enemy", func(g *Game) {
			enemy := CreateEnemyByID(0, 5, 3)
			enemy.Health, enemy.MaxHealth = 1000, 1000
			enemy.StatusAilments.Sleep = 50
			g.state.Enemies = []Enemy{enemy}
		}, func(g *Game, b *bot) error { return b.tap(KeyZ) }},
		{"attack empty air", nil, func(g *Game, b *bot) error { return b.tap(KeyZ) }},
		{"step in place (X+Z)", nil, func(g *Game, b *bot) error { return b.stepInPlace() }},
		{"shoot arrow (D)", func(g *Game) {
			arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
			arrow.ShotCount = 5
			g.state.Player.Inventory = []Item{arrow}
			g.state.Player.EquippedArrow = arrow
		}, func(g *Game, b *bot) error { return b.tap(KeyD) }},
		{"use item from menu", func(g *Game) {
			g.state.Player.Inventory = []Item{buildItemFromTemplate(1, 0, 0)}
		}, func(g *Game, b *bot) error { return useOrFail(b, 0, "使う") }},
		{"throw item from menu", func(g *Game) {
			g.state.Player.Inventory = []Item{buildItemFromTemplate(1, 0, 0)}
		}, func(g *Game, b *bot) error { return useOrFail(b, 0, "投げる") }},
		{"equip weapon from menu", func(g *Game) {
			g.state.Player.Inventory = []Item{buildItemFromTemplate(20, 0, 0)}
		}, func(g *Game, b *bot) error { return useOrFail(b, 0, "装備") }},
		{"use ground item", func(g *Game) {
			g.state.Items = []Item{buildItemFromTemplate(1, 4, 3)}
		}, func(g *Game, b *bot) error {
			g.currentGroundItem = g.state.Items[0]
			g.ShowGroundItem, g.GroundItemActioned = true, true
			g.selectedGroundActionIndex = 2 // 使う
			g.executeGroundItemAction()
			return nil
		}},
		{"open door", func(g *Game) {
			g.state.Map[3][5] = Tile{Type: "door", Blocked: true, BlockSight: true}
		}, func(g *Game, b *bot) error { return b.tap(KeySpace) }},
		{"sleeping auto action", func(g *Game) {
			g.state.Player.StatusAilments.Sleep = 1
		}, func(g *Game, b *bot) error { return g.StepFrame() }},
		{"slow status still one turn", func(g *Game) {
			g.state.Player.StatusAilments.Slow = 5
		}, func(g *Game, b *bot) error { return b.move(1, 0) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, b := newTurnTestGame(t)
			if tt.setup != nil {
				tt.setup(g)
			}
			expectTurns(t, g, 1, func() error { return tt.op(g, b) })
		})
	}
}

func useOrFail(b *bot, index int, action string) error {
	used, err := b.useInventoryItem(index, action)
	if err == nil && !used {
		b.g.state.Player.Inventory = nil // 行動が選べなかったことをテスト側で検出できるようにする
	}
	return err
}

func TestNonActionsDoNotAdvanceTurn(t *testing.T) {
	tests := []struct {
		name string
		op   func(g *Game, b *bot) error
	}{
		{"turn with A+direction", func(g *Game, b *bot) error { return b.tap(KeyA, KeyUp) }},
		{"turn diagonally with A+two directions", func(g *Game, b *bot) error { return b.tap(KeyA, KeyDown, KeyLeft) }},
		{"hold A alone", func(g *Game, b *bot) error {
			b.in.Press(KeyA)
			for i := 0; i < 30; i++ {
				if err := g.StepFrame(); err != nil {
					return err
				}
			}
			b.in.Release(KeyA)
			return g.StepFrame()
		}},
		{"open and close menu", func(g *Game, b *bot) error {
			if err := b.tap(KeyC); err != nil {
				return err
			}
			return b.tap(KeyC)
		}},
		{"space without door", func(g *Game, b *bot) error { return b.tap(KeySpace) }},
		{"walk into wall", func(g *Game, b *bot) error {
			g.state.Player.X = 1
			return b.move(-1, 0)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, b := newTurnTestGame(t)
			x, y := g.state.Player.X, g.state.Player.Y
			if tt.name == "walk into wall" {
				x = 1
			}
			expectTurns(t, g, 0, func() error { return tt.op(g, b) })
			if g.state.Player.X != x || g.state.Player.Y != y {
				t.Fatalf("player moved to (%d, %d)", g.state.Player.X, g.state.Player.Y)
			}
		})
	}
}

// A キーの方向転換が足踏みや移動と干渉しないこと（A を押したまま方向を変えても位置とターンは変わらない）。
func TestDirectionChangeDoesNotInterfere(t *testing.T) {
	g, b := newTurnTestGame(t)
	expectTurns(t, g, 0, func() error { return b.tap(KeyA, KeyUp) })
	if g.state.Player.Direction != Up {
		t.Fatalf("direction = %d, want Up", g.state.Player.Direction)
	}
	// A を押したまま Z を押しても攻撃・足踏みにならない
	expectTurns(t, g, 0, func() error { return b.tap(KeyA, KeyZ) })
	// 方向転換の直後の Z は新しい向きへの攻撃（空振り）で1ターン
	expectTurns(t, g, 1, func() error { return b.tap(KeyZ) })
	if g.state.Player.Direction != Up || g.state.Player.X != 4 || g.state.Player.Y != 3 {
		t.Fatalf("after attack: direction %d at (%d, %d)", g.state.Player.Direction, g.state.Player.X, g.state.Player.Y)
	}
}

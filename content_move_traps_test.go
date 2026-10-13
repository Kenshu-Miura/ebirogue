//go:build !test

package main

import (
	"encoding/json"
	"image/png"
	"testing"
)

func TestBearTrapPlayableActions(t *testing.T) {
	for _, op := range []string{"struggle", "wait", "attack", "item"} {
		t.Run(op, func(t *testing.T) {
			g, b := newTurnTestGame(t)
			g.state.Player.StatusAilments.BearTrap = 2
			g.state.Player.Inventory = []Item{buildItemFromTemplate(1, 0, 0)}
			x, y := g.state.Player.X, g.state.Player.Y
			expectTurns(t, g, 1, func() error {
				switch op {
				case "struggle":
					return b.move(1, 0)
				case "wait":
					return b.stepInPlace()
				case "attack":
					return b.tap(KeyZ)
				default:
					return useOrFail(b, 0, "使う")
				}
			})
			if g.state.Player.X != x || g.state.Player.Y != y || g.state.Player.StatusAilments.BearTrap != 1 {
				t.Fatal("movement restraint or turn decrement failed")
			}
		})
	}
}

func TestPitfallAutoSave(t *testing.T) {
	t.Chdir(t.TempDir())
	g, _ := newHeadlessGame(12, DifficultyAdvanced)
	g.persistent = true
	g.floorTurns = 500
	g.pickupBanned = true
	pitfallTrapEffect(g)
	executeMoveTrapActions(g)
	g.handleFadingOut()
	data, err := storageRead(saveFileName)
	if err != nil {
		t.Fatal(err)
	}
	save, err := decodeSaveData(data)
	if err != nil {
		t.Fatal(err)
	}
	if save.Floor != 2 || g.floorTurns != 0 || g.pickupBanned {
		t.Fatal("arrival auto-save/reset failed")
	}
}

func TestMoveTrapImages(t *testing.T) {
	for _, name := range []string{"warp", "pitfall", "bear"} {
		f, err := imgFS.Open("img/" + name + "_trap.png")
		if err != nil {
			t.Fatal(err)
		}
		im, err := png.Decode(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if im.Bounds().Dx() != 30 || im.Bounds().Dy() != 30 {
			t.Fatal("wrong image size")
		}
		for _, p := range []Coordinate{{X: 0, Y: 0}, {X: 29, Y: 0}, {X: 0, Y: 29}, {X: 29, Y: 29}} {
			_, _, _, a := im.At(p.X, p.Y).RGBA()
			if a != 0 {
				t.Fatal("opaque corner")
			}
		}
	}
}

func executeMoveTrapActions(g *Game) {
	for len(g.ActionQueue.Queue) > 0 {
		a := g.ActionQueue.Queue[0]
		g.ActionQueue.Queue = g.ActionQueue.Queue[1:]
		if a.Execute != nil {
			a.Execute(g)
		}
	}
}

func TestMoveTrapsActivationAndFailure(t *testing.T) {
	for id := 5; id <= 7; id++ {
		for _, fail := range []bool{false, true} {
			g, _ := newHeadlessGame(17, DifficultyAdvanced)
			g.state.Map = makeTestFloorMap(8, 8)
			g.state.Player.X, g.state.Player.Y = 2, 2
			g.state.Enemies = nil
			g.state.Items = nil
			g.rooms = []Room{{X: 0, Y: 0, Width: 8, Height: 8}}
			trap := createMapTrapByID(id, 2, 2)
			trap.FailRate = 0
			if fail {
				trap.FailRate = 100
			}
			g.state.MapTraps = []MapTrap{trap}
			health := g.state.Player.Health
			g.stepOnTrap(0)
			if g.state.Player.Health != health || g.state.Player.StatusAilments.BearTrap != 0 {
				t.Fatal("effect executed before queue")
			}
			executeMoveTrapActions(g)
			if !g.state.MapTraps[0].Discovered {
				t.Fatal("trap not discovered")
			}
			moved := g.state.Player.X != 2 || g.state.Player.Y != 2
			if fail {
				if moved || g.fadingOut || g.state.Player.StatusAilments.BearTrap != 0 || g.state.Player.Health != health {
					t.Fatalf("failed trap %d applied effect", id)
				}
			} else {
				switch id {
				case 5:
					if !moved || g.Floor != 1 {
						t.Fatal("warp failed")
					}
				case 6:
					if !g.fadingOut || g.state.Player.Health >= health {
						t.Fatal("fall failed")
					}
					g.handleFadingOut()
					if g.Floor != 2 || g.floorTurns != 0 || g.bossDefeated {
						t.Fatal("floor initialization failed")
					}
				case 7:
					if g.state.Player.StatusAilments.BearTrap != 6 {
						t.Fatal("bear trap failed")
					}
				}
			}
		}
	}
}

func TestBearTrapMovementAndSave(t *testing.T) {
	g, _ := newHeadlessGame(22, DifficultyAdvanced)
	g.state.MapTraps = []MapTrap{createMapTrapByID(5, 2, 2), createMapTrapByID(6, 3, 2), createMapTrapByID(7, 4, 2)}
	g.state.MapTraps[0].Discovered = true
	g.state.Player.StatusAilments.BearTrap = 2
	data, err := json.Marshal(g.buildSaveData())
	if err != nil {
		t.Fatal(err)
	}
	save, err := decodeSaveData(data)
	if err != nil {
		t.Fatal(err)
	}
	resumed, _ := newHeadlessGame(99, DifficultyAdvanced)
	if err := resumed.applySaveData(save); err != nil {
		t.Fatal(err)
	}
	if resumed.state.Player.StatusAilments.BearTrap != 2 {
		t.Fatal("status lost")
	}
	for i, trap := range resumed.state.MapTraps {
		if trap.TemplateID != i+5 || trap.Effect == nil {
			t.Fatal("trap lost")
		}
	}
	if !resumed.state.MapTraps[0].Discovered {
		t.Fatal("discovery lost")
	}
	x, y := resumed.state.Player.X, resumed.state.Player.Y
	resumed.state.Player.StatusAilments.Confusion = 2
	if !resumed.MovePlayer(1, 0) || !resumed.isActioned || resumed.state.Player.X != x || resumed.state.Player.Y != y {
		t.Fatal("struggling must consume turn without moving")
	}
	resumed.decrementStatusAilments()
	if resumed.state.Player.StatusAilments.BearTrap != 1 {
		t.Fatal("status not decremented")
	}
	resumed.IncrementMoveCount() // 足踏み・攻撃・道具と共通のターン処理
	if resumed.state.Player.StatusAilments.BearTrap != 0 {
		t.Fatal("status not released")
	}
}

func TestPitfallBossSafetyAndDeath(t *testing.T) {
	for _, floor := range []int{9, 10, 19, 20, 29, 30, 31} {
		g, _ := newHeadlessGame(7, DifficultyAdvanced)
		g.Floor = floor
		g.bossDefeated = true // 撃破後にも落とし穴は禁止
		pitfallTrapEffect(g)
		executeMoveTrapActions(g)
		allowed := !isBossFloor(floor) && floor < finalFloor
		if g.fadingOut != allowed {
			t.Fatalf("floor %d fall = %v", floor, g.fadingOut)
		}
		for _, id := range availableTrapIDs(randomMapTrapIDs, floor) {
			if id == 6 && !allowed {
				t.Fatal("boss trap generation allowed")
			}
		}
		if allowed {
			g.handleFadingOut()
			if g.Floor != floor+1 {
				t.Fatal("wrong arrival floor")
			}
		}
	}
	g, _ := newHeadlessGame(7, DifficultyAdvanced)
	g.state.Player.Health = 1
	g.steppedTrapSource = trapSource(pitfallTrapID, "落とし穴")
	pitfallTrapEffect(g)
	executeMoveTrapActions(g)
	if !g.playerDead || g.fadingOut || g.lastDamageSource.TrapID != 6 {
		t.Fatalf("fall death incorrectly recorded: %+v", g.lastDamageSource)
	}
}

//go:build !test

package main

import "testing"

func TestTrapInteractionWalk(t *testing.T) {
	g := newProjectileTestGame()
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 2, 2)}
	trap := createMapTrapByID(0, 3, 2)
	trap.FailRate = 0
	g.state.MapTraps = []MapTrap{trap}
	g.tryMoveEnemy(0, 1, 0)
	runQueuedActions(g)
	if g.state.Enemies[0].StatusAilments.Sleep != 10 {
		t.Fatal("歩いた敵が睡眠ガスにかからない")
	}
	if g.state.Player.StatusAilments.Sleep != 0 {
		t.Fatal("敵の罠が海老さんに効いた")
	}
}

func trapInteractionGame(id int) *Game {
	g := newProjectileTestGame()
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 3, 2)}
	g.state.Enemies[0].Health = 100
	trap := createMapTrapByID(id, 3, 2)
	trap.FailRate = 0
	g.state.MapTraps = []MapTrap{trap}
	return g
}

func TestTrapInteractionEffects(t *testing.T) {
	for id := 0; id <= 14; id++ {
		t.Run(mapTrapTemplates[id].Name, func(t *testing.T) {
			g := trapInteractionGame(id)
			p := g.state.Player
			g.enemyTrapAfterMove(0, 2, 2)
			runQueuedActions(g)
			if g.state.Player.Health != p.Health || g.state.Player.Power != p.Power || g.state.Player.StatusAilments != p.StatusAilments {
				t.Fatal("敵の罠が海老さんに効いた")
			}
			switch id {
			case 0:
				if g.state.Enemies[0].StatusAilments.Sleep != 10 {
					t.Fatal("sleep")
				}
			case 2:
				if g.state.Enemies[0].StatusAilments.Slow != 10 {
					t.Fatal("slow")
				}
			case 7:
				if g.state.Enemies[0].StatusAilments.BearTrap != 6 {
					t.Fatal("bear")
				}
			case 1, 3, 8, 9, 10, 11:
				if g.state.Enemies[0].Health >= 100 {
					t.Fatal("damage")
				}
			case 5:
				if g.state.Enemies[0].X == 3 && g.state.Enemies[0].Y == 2 {
					t.Fatal("warp")
				}
			case 6:
				if len(g.state.Enemies) != 0 {
					t.Fatal("pitfall")
				}
			default:
				if g.state.Enemies[0].Health != 100 || len(g.state.MapTraps) != 1 {
					t.Fatal("無効の罠は残す")
				}
			}
		})
	}
}

func TestTrapInteractionImmunityAndFailure(t *testing.T) {
	for _, kind := range []string{"floating", "boss", "setter", "wall", "disguise", "fail"} {
		t.Run(kind, func(t *testing.T) {
			g := trapInteractionGame(0)
			switch kind {
			case "floating":
				g.state.Enemies[0].Traits = []EnemyTrait{EnemyTraitFloating}
			case "boss":
				g.state.Enemies[0] = CreateEnemyByID(42, 3, 2)
			case "setter":
				g.state.Enemies[0].SpecialMovement = SpecialMovementCreateTrap
			case "wall":
				g.state.Map[2][3].Blocked = true
			case "disguise":
				g.state.Enemies[0].Disguise = EnemyDisguiseItem
				g.state.Enemies[0].Revealed = false
			case "fail":
				g.state.MapTraps[0].FailRate = 100
			}
			g.enemyTrapAfterMove(0, 2, 2)
			runQueuedActions(g)
			if g.state.Enemies[0].StatusAilments.Sleep > 0 {
				t.Fatal("免除または不発")
			}
		})
	}
}

func TestTrapInteractionDiscovery(t *testing.T) {
	for _, visible := range []bool{false, true} {
		g := trapInteractionGame(0)
		if visible {
			g.state.Player.X, g.state.Player.Y = 3, 3
		}
		g.enemyTrapAfterMove(0, 2, 2)
		runQueuedActions(g)
		if g.state.MapTraps[0].Discovered != visible {
			t.Fatalf("visible=%v", visible)
		}
	}
}

func TestTrapInteractionCredit(t *testing.T) {
	for _, credit := range []bool{false, true} {
		g := trapInteractionGame(1)
		g.state.Enemies[0].Health = 1
		g.queueLandedTrap(3, 2, credit)
		runQueuedActions(g)
		if len(g.state.Enemies) != 0 {
			t.Fatal("enemy remains")
		}
		if (g.state.Player.ExperiencePoints > 0) != credit {
			t.Fatalf("credit=%v xp=%d", credit, g.state.Player.ExperiencePoints)
		}
	}
}

func TestTrapInteractionProjectileLanding(t *testing.T) {
	for _, kind := range []ProjectileKind{ProjectileThrow, ProjectileShot, ProjectileBolt} {
		g := newProjectileTestGame()
		trap := createMapTrapByID(0, 6, 4)
		trap.FailRate = 0
		g.state.MapTraps = []MapTrap{trap}
		item := buildItemFromTemplate(1, 0, 0)
		p := g.newPlayerProjectile(item, kind, 2)
		g.ThrowItem(p)
		finishThrow(t, g)
		if g.state.MapTraps[0].Discovered != (kind != ProjectileBolt) {
			t.Fatalf("kind=%v", kind)
		}
	}
	g := newProjectileTestGame()
	trap := createMapTrapByID(0, 5, 4)
	trap.FailRate = 0
	g.state.MapTraps = []MapTrap{trap}
	g.ThrowItem(g.newPlayerProjectile(buildItemFromTemplate(1, 0, 0), ProjectileThrow, 2))
	finishThrow(t, g)
	if g.state.MapTraps[0].Discovered {
		t.Fatal("通過では作動しない")
	}
}

func TestTrapInteractionExplosionDepth(t *testing.T) {
	g := newProjectileTestGame()
	g.state.Player.X, g.state.Player.Y = 1, 7
	g.state.MapTraps = nil
	for x := 1; x <= 7; x++ {
		g.state.MapTraps = append(g.state.MapTraps, createMapTrapByID(11, x, 1))
	}
	g.explodeTraps(1, 1, 0, true, 0)
	runQueuedActions(g)
	// 深さ1,2,3が爆発し、その先の地雷は破壊だけ。5マス目以遠は残る。
	if len(g.state.MapTraps) != 3 || g.state.MapTraps[0].X != 5 {
		t.Fatalf("chain=%+v", g.state.MapTraps)
	}
	g.state.MapTraps = []MapTrap{createMapTrapByID(0, 2, 2)}
	g.explodeTraps(2, 2, 0, false, 0)
	if len(g.state.MapTraps) != 0 {
		t.Fatal("非地雷は消滅")
	}
}

func TestTrapInteractionSaveAndInvariants(t *testing.T) {
	for _, id := range []int{5, 6, 7} {
		g, _ := newHeadlessGame(77, DifficultyBeginner)
		g.state.Enemies = []Enemy{CreateEnemyByID(0, g.state.Player.X+1, g.state.Player.Y)}
		// 実際の空き床を選び、地形の不変条件も検査する。
		for y, row := range g.state.Map {
			for x, tile := range row {
				if tile.Type == "floor" && !tile.Blocked && !isOccupied(g, x, y) {
					g.state.Enemies[0].X, g.state.Enemies[0].Y = x, y
				}
			}
		}
		e := g.state.Enemies[0]
		trap := createMapTrapByID(id, e.X, e.Y)
		trap.FailRate = 0
		g.state.MapTraps = []MapTrap{trap}
		g.enemyTrapAfterMove(0, -1, -1)
		runQueuedActions(g)
		if v := checkInvariants(g); len(v) > 0 {
			t.Fatalf("before save: %v", v)
		}
		restored, v := saveRoundTrip(g)
		if len(v) > 0 {
			t.Fatal(v)
		}
		if v := checkInvariants(restored); len(v) > 0 {
			t.Fatalf("restored: %v", v)
		}
		if id == 7 && restored.state.Enemies[0].StatusAilments.BearTrap != 6 {
			t.Fatal("罠の状態が復元されない")
		}
	}
}

func TestTrapInteractionDefeatHook(t *testing.T) {
	called := 0
	withTestMonster(t, 9900, 0, func(d *MonsterDefinition) { d.OnDefeated = func(g *Game, e Enemy) { called++ } })
	for _, id := range []int{1, 6} {
		g := trapInteractionGame(id)
		g.state.Enemies[0] = CreateEnemyByID(9900, 3, 2)
		g.state.Enemies[0].Health = 1
		g.enemyTrapAfterMove(0, 2, 2)
		runQueuedActions(g)
		if g.state.Player.ExperiencePoints != 0 {
			t.Fatal("自然作動で経験値が入った")
		}
	}
	if called != 2 {
		t.Fatalf("OnDefeated=%d", called)
	}
}

func TestTrapInteractionKnockbackAndNoRepeatedStep(t *testing.T) {
	g := trapInteractionGame(0)
	g.state.Enemies[0].X = 2
	g.knockbackEnemy(0, 1, 0, 1, 0)
	if g.state.Enemies[0].StatusAilments.Sleep != 0 {
		t.Fatal("Execute より前に作動した")
	}
	runQueuedActions(g)
	if g.state.Enemies[0].StatusAilments.Sleep != 10 {
		t.Fatal("吹き飛ばしの着地で作動しない")
	}
	g.clearEnemyStatus(0, StatusSleep)
	g.enemyTrapAfterMove(0, 3, 2)
	runQueuedActions(g)
	if g.state.Enemies[0].StatusAilments.Sleep != 0 {
		t.Fatal("同じ場所では踏み直さない")
	}
}

func TestTrapInteractionExplosionEntrypoints(t *testing.T) {
	for _, kind := range []string{"enemy", "ring", "largeMine"} {
		t.Run(kind, func(t *testing.T) {
			g := newProjectileTestGame()
			g.state.MapTraps = []MapTrap{createMapTrapByID(11, 5, 4), createMapTrapByID(0, 6, 4)}
			switch kind {
			case "enemy":
				g.state.Enemies = []Enemy{CreateEnemyByID(0, 4, 4)}
				g.state.Player.X, g.state.Player.Y = 1, 1
				g.enqueueEnemyExplosion(EnemyExplosion{BomberIndex: 0, CenterX: 4, CenterY: 4, Radius: 1, Damage: 1})
			case "ring":
				g.Enqueue(Action{Execute: explodeRing})
			case "largeMine":
				g.steppedTrapSource = trapSource(11, "大型地雷")
				largeMineTrapEffect(g)
			}
			runQueuedActions(g)
			if len(g.state.MapTraps) != 0 {
				t.Fatalf("罠が残った: %+v", g.state.MapTraps)
			}
		})
	}
}

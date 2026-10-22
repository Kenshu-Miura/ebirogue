//go:build !test

package main

import (
	"strings"
	"testing"
)

// 敵は罠を踏まない（開発者の決定、2026-10-22）: 歩いても効果・発見・メッセージは無く、罠は残る。
func TestEnemyDoesNotTriggerTraps(t *testing.T) {
	for id := range mapTrapTemplates {
		t.Run(mapTrapTemplates[id].Name, func(t *testing.T) {
			g := trapInteractionGame(id)
			g.state.Enemies[0].X = 2
			g.state.Player.X, g.state.Player.Y = 3, 3 // 罠のすぐ隣で見ている
			p := g.state.Player
			if !g.tryMoveEnemy(0, 1, 0) {
				t.Fatal("敵が罠のマスへ動けない")
			}
			runQueuedActions(g)
			assertEnemyIgnoredTrap(t, g, p, 100)
		})
	}
}

// 吹き飛ばしの着地も同じ。
func TestKnockedBackEnemyDoesNotTriggerTrap(t *testing.T) {
	g := trapInteractionGame(0)
	g.state.Enemies[0].X = 2
	p := g.state.Player
	g.knockbackEnemy(0, 1, 0, 1, 0)
	runQueuedActions(g)
	if g.state.Enemies[0].X != 3 {
		t.Fatalf("吹き飛ばない: %+v", g.state.Enemies[0])
	}
	assertEnemyIgnoredTrap(t, g, p, 100)
}

func assertEnemyIgnoredTrap(t *testing.T, g *Game, p Player, hp int) {
	t.Helper()
	if len(g.state.Enemies) != 1 {
		t.Fatal("敵が消えた")
	}
	e := g.state.Enemies[0]
	if e.X != 3 || e.Y != 2 || e.Health != hp || e.StatusAilments != (StatusAilments{}) {
		t.Fatalf("罠が敵に作動した: %+v", e)
	}
	if len(g.state.MapTraps) != 1 || g.state.MapTraps[0].Discovered {
		t.Fatalf("罠が消えたか発見済みになった: %+v", g.state.MapTraps)
	}
	if g.state.Player.Health != p.Health || g.state.Player.Power != p.Power || g.state.Player.StatusAilments != p.StatusAilments {
		t.Fatal("敵の罠が海老さんに効いた")
	}
	for _, m := range g.messageLog.Messages {
		if strings.Contains(m, "罠") || strings.Contains(m, "地雷") || strings.Contains(m, "落とし穴") || strings.Contains(m, "トラバサミ") {
			t.Fatalf("罠のメッセージが出た: %q", m)
		}
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

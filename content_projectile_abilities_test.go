//go:build !test

package main

// 飛ぶ道具の能力で変える項目（Projectile のフィールドと ModifyProjectile のフック）のテスト。

import (
	"strings"
	"testing"
)

// equipProjectileRing はテスト中だけ、飛ぶ道具を modify で変える指輪を装備させる。
func equipProjectileRing(t *testing.T, g *Game, modify func(g *Game, p *Projectile, stacks int)) {
	t.Helper()
	id := EquipmentAbilityID("test_projectile_" + t.Name())
	withTestAbilityHooks(t, id, AbilityHooks{ModifyProjectile: modify})
	equip(g, testRing(id))
}

func throwTestFood(t *testing.T, g *Game) (Item, string) {
	t.Helper()
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = append(g.state.Player.Inventory, food)
	g.throwWithCallbacks(food, 10)
	return food, finishThrow(t, g)
}

func TestProjectileMissFliesPastEnemy(t *testing.T) {
	g := newProjectileTestGame()
	index := addTestEnemy(g, 0, 6, 4)
	g.state.Enemies[index].Health = 20
	equipProjectileRing(t, g, func(_ *Game, p *Projectile, _ int) { p.HitPercent = 0 })

	food, messages := throwTestFood(t, g)

	if g.state.Enemies[index].Health != 20 || !strings.Contains(messages, "当たらなかった") {
		t.Fatalf("health = %d, messages = %q", g.state.Enemies[index].Health, messages)
	}
	if items := floorItemsAt(g, 7, 4); len(items) != 1 || items[0] != food {
		t.Fatalf("items = %v, want the food behind the enemy at (7,4)", g.state.Items)
	}
}

func TestProjectileHitPercentUsesProjectileRandom(t *testing.T) {
	for _, tc := range []struct {
		roll    int
		sure    bool
		wantHit bool
	}{{roll: 49, wantHit: true}, {roll: 50, wantHit: false}, {roll: 99, sure: true, wantHit: true}} {
		g := newProjectileTestGame()
		index := addTestEnemy(g, 0, 6, 4)
		g.state.Enemies[index].Health = 20
		g.randHooks.projectile = func(int) int { return tc.roll }
		equipProjectileRing(t, g, func(_ *Game, p *Projectile, _ int) {
			p.HitPercent = 50
			p.SureHit = tc.sure
		})

		throwTestFood(t, g)

		if hit := g.state.Enemies[index].Health < 20; hit != tc.wantHit {
			t.Fatalf("roll %d sure %v: hit = %v, want %v", tc.roll, tc.sure, hit, tc.wantHit)
		}
	}
}

func TestProjectileBeforeHitCanMakeItMiss(t *testing.T) {
	g := newProjectileTestGame()
	index := addTestEnemy(g, 0, 6, 4)
	g.state.Enemies[index].Health = 20
	equipProjectileRing(t, g, func(_ *Game, p *Projectile, _ int) {
		p.BeforeHit = append(p.BeforeHit, func(_ *Game, _ *Projectile, hit *ProjectileHit) { hit.HitPercent = 0 })
	})

	throwTestFood(t, g)

	if g.state.Enemies[index].Health != 20 {
		t.Fatal("BeforeHit should be able to make the item miss")
	}
}

func TestProjectilePierceTargetsHitsEveryEnemy(t *testing.T) {
	g := newProjectileTestGame()
	first := addTestEnemy(g, 0, 5, 4)
	second := addTestEnemy(g, 0, 6, 4)
	g.state.Enemies[first].Health = 20
	g.state.Enemies[second].Health = 20
	equipProjectileRing(t, g, func(_ *Game, p *Projectile, _ int) { p.PierceTargets = true })

	throwTestFood(t, g)

	if g.state.Enemies[first].Health != 17 || g.state.Enemies[second].Health != 17 {
		t.Fatalf("health = %d, %d; want both 17", g.state.Enemies[first].Health, g.state.Enemies[second].Health)
	}
	if len(g.state.Items) != 0 {
		t.Fatalf("items = %v, an item that hit something should vanish", g.state.Items)
	}
}

func TestProjectilePierceWallsLandsOnLastFloor(t *testing.T) {
	for _, tc := range []struct {
		name     string
		wallsAt  []int
		wantLand Coordinate
	}{
		{name: "through a wall", wallsAt: []int{6}, wantLand: Coordinate{X: 14, Y: 4}},
		{name: "ends inside a wall", wallsAt: []int{6, 13, 14}, wantLand: Coordinate{X: 12, Y: 4}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newProjectileTestGame()
			g.state.Map = makeTestFloorMap(20, 9)
			for _, x := range tc.wallsAt {
				g.state.Map[4][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
			}
			equipProjectileRing(t, g, func(_ *Game, p *Projectile, _ int) { p.PierceWalls = true })

			throwTestFood(t, g)

			if items := floorItemsAt(g, tc.wantLand.X, tc.wantLand.Y); len(items) != 1 {
				t.Fatalf("items = %v, want the food at %v", g.state.Items, tc.wantLand)
			}
		})
	}
}

func TestProjectileBounceComesBackPastThrower(t *testing.T) {
	g := newProjectileTestGame()
	equipProjectileRing(t, g, func(_ *Game, p *Projectile, _ int) { p.Bounces = 1 })

	throwTestFood(t, g)

	// 5,6,7 と進んで右の壁で跳ね返り、海老さん（4）を素通りして左の壁の手前（1）に落ちる
	if items := floorItemsAt(g, 1, 4); len(items) != 1 {
		t.Fatalf("items = %v, want the food at (1,4)", g.state.Items)
	}
	if g.state.Player.Health != 40 {
		t.Fatal("a bounced item should not hit the thrower")
	}
}

func TestProjectileBounceHitsEnemyBehindThrower(t *testing.T) {
	g := newProjectileTestGame()
	index := addTestEnemy(g, 0, 2, 4)
	g.state.Enemies[index].Health = 20
	equipProjectileRing(t, g, func(_ *Game, p *Projectile, _ int) { p.Bounces = 1 })

	throwTestFood(t, g)

	if g.state.Enemies[index].Health != 17 {
		t.Fatalf("health = %d, want 17", g.state.Enemies[index].Health)
	}
}

func TestProjectileOnTileCanStopFlight(t *testing.T) {
	g := newProjectileTestGame()
	var visited []Coordinate
	equipProjectileRing(t, g, func(_ *Game, p *Projectile, _ int) {
		p.OnTile = append(p.OnTile, func(_ *Game, p *Projectile, x, y int) {
			visited = append(visited, Coordinate{X: x, Y: y})
			if p.Traveled() == 2 {
				p.Stop()
			}
		})
	})

	throwTestFood(t, g)

	if len(visited) != 2 || visited[1] != (Coordinate{X: 6, Y: 4}) {
		t.Fatalf("visited = %v", visited)
	}
	if items := floorItemsAt(g, 6, 4); len(items) != 1 {
		t.Fatalf("items = %v, want the food at (6,4)", g.state.Items)
	}
}

func TestProjectileHookSeesKind(t *testing.T) {
	g := newProjectileTestGame()
	var kinds []ProjectileKind
	equipProjectileRing(t, g, func(_ *Game, p *Projectile, _ int) { kinds = append(kinds, p.Kind) })
	arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
	arrow.ShotCount = 2
	cane := buildItemFromTemplate(16, 0, 0).(*Cane)
	cane.Uses = 1
	g.state.Player.Inventory = append(g.state.Player.Inventory, arrow, cane)

	throwTestFood(t, g)
	g.shootArrow(arrow, 10)
	finishThrow(t, g)
	g.useCane(cane)
	finishThrow(t, g)

	want := []ProjectileKind{ProjectileThrow, ProjectileShot, ProjectileBolt}
	if len(kinds) != 3 || kinds[0] != want[0] || kinds[1] != want[1] || kinds[2] != want[2] {
		t.Fatalf("kinds = %v, want %v", kinds, want)
	}
}

func TestShootArrowVolleyShootsSeveralArrows(t *testing.T) {
	for _, tc := range []struct {
		arrows, volley, wantShots int
	}{{arrows: 5, volley: 2, wantShots: 2}, {arrows: 5, volley: 3, wantShots: 3}, {arrows: 1, volley: 3, wantShots: 1}} {
		g := newProjectileTestGame()
		index := addTestEnemy(g, 0, 6, 4)
		g.state.Enemies[index].Health = 999
		arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
		arrow.ShotCount = tc.arrows
		equip(g, arrow)
		equipProjectileRing(t, g, func(_ *Game, p *Projectile, _ int) {
			if p.Kind == ProjectileShot {
				p.Volley = tc.volley
			}
		})
		p := g.state.Player
		oneShot := rollPlayerAttackDamage(playerAttackTotal(p.AttackPower, p.Power, p.Level)+arrow.AttackPower,
			g.state.Enemies[index].DefensePower, 1.0, g.damageIntn()).Damage

		g.shootArrow(arrow, 10)
		finishThrow(t, g)

		if got := 999 - g.state.Enemies[index].Health; got != oneShot*tc.wantShots {
			t.Fatalf("arrows %d volley %d: damage = %d, want %d shots of %d", tc.arrows, tc.volley, got, tc.wantShots, oneShot)
		}
		if arrow.ShotCount != tc.arrows-tc.wantShots {
			t.Fatalf("arrows left = %d, want %d", arrow.ShotCount, tc.arrows-tc.wantShots)
		}
		if g.stats.counts("Arrow").Shot != tc.wantShots {
			t.Fatalf("shot count stat = %d", g.stats.counts("Arrow").Shot)
		}
	}
}

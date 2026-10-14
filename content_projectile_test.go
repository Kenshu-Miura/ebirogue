//go:build !test

package main

// 投擲・射撃・杖の魔法弾の飛び方（projectile.go）のテスト。
// 前半は共通化の前の ThrowItem の挙動をそのまま固定したもの（共通化で挙動を変えていないことの確認）。

import (
	"strings"
	"testing"
)

// newProjectileTestGame は 9x9 の部屋（外周が壁）の (4, 4) に右向きの海老さんを置く。
func newProjectileTestGame() *Game {
	g := newFoundationTestGame()
	g.state.Player.Inventory = nil
	return g
}

// finishThrow は積まれた Action を実行し、飛んでいる道具が着地するまで進める。
func finishThrow(t *testing.T, g *Game) string {
	t.Helper()
	var messages []string
	for i := 0; i < 20; i++ {
		messages = append(messages, runQueuedActions(g))
		for frame := 0; frame < 200 && g.ThrownItem.Item != nil; frame++ {
			g.UpdateThrownItem()
		}
		if g.ThrownItem.Item != nil {
			t.Fatal("thrown item did not land")
		}
		if len(g.ActionQueue.Queue) == 0 {
			return strings.Join(messages, "\n")
		}
	}
	t.Fatal("actions kept being queued")
	return ""
}

func floorItemsAt(g *Game, x, y int) []Item {
	var items []Item
	for _, item := range g.state.Items {
		if ix, iy := item.GetPosition(); ix == x && iy == y {
			items = append(items, item)
		}
	}
	return items
}

func TestThrowLandsBeforeWall(t *testing.T) {
	g := newProjectileTestGame()
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}

	g.throwWithCallbacks(food, 10)
	messages := finishThrow(t, g)

	if !strings.Contains(messages, "ウインナーを投げた") {
		t.Fatalf("messages = %q", messages)
	}
	if len(g.state.Player.Inventory) != 0 {
		t.Fatalf("inventory = %d, want thrown item removed", len(g.state.Player.Inventory))
	}
	if items := floorItemsAt(g, 7, 4); len(items) != 1 || items[0] != food {
		t.Fatalf("items at (7,4) = %v, want the thrown food", items)
	}
	if !g.isActioned {
		t.Fatal("throw should consume the turn")
	}
}

func TestThrowStopsAtRangeEnd(t *testing.T) {
	g := newProjectileTestGame()
	g.state.Map = makeTestFloorMap(20, 9)
	g.state.Player.X = 2
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}

	g.throwWithCallbacks(food, 3)
	finishThrow(t, g)

	if items := floorItemsAt(g, 5, 4); len(items) != 1 {
		t.Fatalf("items = %v, want the food at range end (5,4)", g.state.Items)
	}
}

func TestThrowDiagonalLandsBeforeWall(t *testing.T) {
	g := newProjectileTestGame()
	g.state.Player.Direction = DownRight
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}

	g.throwWithCallbacks(food, 10)
	finishThrow(t, g)

	if items := floorItemsAt(g, 7, 7); len(items) != 1 {
		t.Fatalf("items = %v, want the food at (7,7)", g.state.Items)
	}
}

func TestThrowStopsAtMapEdge(t *testing.T) {
	g := newProjectileTestGame()
	m := make([][]Tile, 5)
	for y := range m {
		m[y] = make([]Tile, 6)
		for x := range m[y] {
			m[y][x] = Tile{Type: "floor"}
		}
	}
	g.state.Map = m
	g.state.Player.X, g.state.Player.Y = 3, 2
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}

	g.throwWithCallbacks(food, 10)
	finishThrow(t, g)

	if items := floorItemsAt(g, 5, 2); len(items) != 1 {
		t.Fatalf("items = %v, want the food on the last tile (5,2)", g.state.Items)
	}
}

func TestThrowLandingOnItemMovesToNeighbor(t *testing.T) {
	g := newProjectileTestGame()
	existing := buildItemFromTemplate(2, 7, 4)
	g.state.Items = []Item{existing}
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}

	g.throwWithCallbacks(food, 10)
	finishThrow(t, g)

	// 着地点に道具があれば、周囲8マスを左上から順に探して空いたマスに置く
	if items := floorItemsAt(g, 6, 3); len(items) != 1 || items[0] != food {
		t.Fatalf("items = %v, want the food moved to (6,3)", g.state.Items)
	}
}

func TestThrowFromGroundRemovesGroundItem(t *testing.T) {
	g := newProjectileTestGame()
	food := buildItemFromTemplate(1, 4, 4)
	g.state.Items = []Item{food}

	g.throwWithCallbacks(food, 10)
	finishThrow(t, g)

	if len(g.state.Items) != 1 || floorItemsAt(g, 7, 4) == nil {
		t.Fatalf("items = %v, want only the food at (7,4)", g.state.Items)
	}
}

func TestThrowHitsFirstEnemyOnly(t *testing.T) {
	g := newProjectileTestGame()
	first := addTestEnemy(g, 0, 6, 4)
	second := addTestEnemy(g, 0, 7, 4)
	g.state.Enemies[first].Health = 20
	g.state.Enemies[second].Health = 20
	food := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{food}

	g.throwWithCallbacks(food, 10)
	messages := finishThrow(t, g)

	// 投げた道具のダメージは 1〜3（乱数を最大に固定して3）
	if g.state.Enemies[first].Health != 17 || g.state.Enemies[second].Health != 20 {
		t.Fatalf("health = %d, %d; want 17, 20", g.state.Enemies[first].Health, g.state.Enemies[second].Health)
	}
	if !strings.Contains(messages, "3のダメージを与えた") {
		t.Fatalf("messages = %q", messages)
	}
	if len(g.state.Items) != 0 || len(g.state.Player.Inventory) != 0 {
		t.Fatalf("items = %v, inventory = %v; the item should vanish on hit", g.state.Items, g.state.Player.Inventory)
	}
	if g.ThrownItemDestination != (Coordinate{}) || g.TargetEnemy != nil {
		t.Fatal("thrown item state should be reset after landing")
	}
}

func TestThrownPotionAppliesToEnemy(t *testing.T) {
	g := newProjectileTestGame()
	index := addTestEnemy(g, 0, 6, 4)
	potion := buildItemFromTemplate(17, 0, 0) // 睡眠薬
	g.state.Player.Inventory = []Item{potion}

	g.throwWithCallbacks(potion, 10)
	finishThrow(t, g)

	if g.state.Enemies[index].StatusAilments.Sleep != 10 {
		t.Fatalf("sleep = %d, want 10", g.state.Enemies[index].StatusAilments.Sleep)
	}
}

func TestShotArrowHitsEnemyWithAttackFormula(t *testing.T) {
	g := newProjectileTestGame()
	index := addTestEnemy(g, 0, 6, 4)
	g.state.Enemies[index].Health = 99
	arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
	arrow.ShotCount = 3
	equip(g, arrow)
	p := g.state.Player
	want := rollPlayerAttackDamage(playerAttackTotal(p.AttackPower, p.Power, p.Level)+arrow.AttackPower,
		g.state.Enemies[index].DefensePower, 1.0, g.damageIntn()).Damage

	if !g.shootArrow(arrow, 10) {
		t.Fatal("shootArrow failed")
	}
	messages := finishThrow(t, g)

	if !strings.Contains(messages, "銀の弓矢を撃った") {
		t.Fatalf("messages = %q", messages)
	}
	if got := 99 - g.state.Enemies[index].Health; got != want {
		t.Fatalf("damage = %d, want %d", got, want)
	}
	if arrow.ShotCount != 2 || len(g.state.Items) != 0 {
		t.Fatalf("arrows left = %d, floor = %v", arrow.ShotCount, g.state.Items)
	}
	if g.dPressed {
		t.Fatal("shooting state should be cleared after landing")
	}
}

func TestShotArrowLandsAsOneArrow(t *testing.T) {
	g := newProjectileTestGame()
	arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
	arrow.ShotCount = 3
	equip(g, arrow)

	g.shootArrow(arrow, 10)
	finishThrow(t, g)

	items := floorItemsAt(g, 7, 4)
	if len(items) != 1 {
		t.Fatalf("items = %v, want one arrow at (7,4)", g.state.Items)
	}
	if landed, ok := items[0].(*Arrow); !ok || landed.ShotCount != 1 || landed == arrow {
		t.Fatalf("landed = %#v, want a separate arrow with count 1", items[0])
	}
	if arrow.ShotCount != 2 || g.state.Player.Inventory[0] != arrow {
		t.Fatalf("inventory arrow count = %d", arrow.ShotCount)
	}
}

func TestCaneBoltReachesWallTile(t *testing.T) {
	g := newProjectileTestGame()
	cane := buildItemFromTemplate(16, 0, 0).(*Cane) // 封印の杖
	cane.Uses = 3
	g.state.Player.Inventory = []Item{cane}

	g.useCane(cane)
	finishThrow(t, g)

	if len(g.state.Items) != 0 {
		t.Fatalf("items = %v, the bolt should not land", g.state.Items)
	}
	if len(g.state.Player.Inventory) != 1 || cane.Uses != 2 {
		t.Fatalf("inventory = %v, uses = %d", g.state.Player.Inventory, cane.Uses)
	}
}

func TestCaneBoltSealsEnemy(t *testing.T) {
	g := newProjectileTestGame()
	index := addTestEnemy(g, 0, 6, 4)
	cane := buildItemFromTemplate(16, 0, 0).(*Cane)
	cane.Uses = 3
	g.state.Player.Inventory = []Item{cane}

	g.useCane(cane)
	finishThrow(t, g)

	if !g.state.Enemies[index].StatusAilments.Seal {
		t.Fatal("enemy should be sealed")
	}
	if len(g.state.Player.Inventory) != 1 || len(g.state.Items) != 0 {
		t.Fatalf("inventory = %v, items = %v", g.state.Player.Inventory, g.state.Items)
	}
}

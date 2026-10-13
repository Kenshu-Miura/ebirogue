//go:build !test

package main

import (
	"fmt"
	"testing"
)

// BOT のテストプレイ（TestSimSmoke / TestSimBatch）で見つかった不具合の回帰テスト。

// 所持品が1個だけのときに「使う」と、空になった所持品を参照して panic していた。
func TestUsingOnlyInventoryItemDoesNotPanic(t *testing.T) {
	for _, id := range randomItemTemplateIDs {
		id := id
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			g, _ := newHeadlessGame(1, DifficultyAdvanced)
			item := g.createItemByID(id, 0, 0)
			if _, ok := item.(Equipable); ok {
				return
			}
			if _, ok := item.(*Pot); ok {
				return
			}
			g.state.Player.Inventory = []Item{item}
			g.showInventory, g.showItemActions = true, true
			g.selectedItemIndex, g.selectedActionIndex = 0, 0
			g.executeAction()
			if err := g.RunUntilInputReady(2000); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// 部屋の数が少ないと、敵がプレイヤーと同じマスに配置されることがあった。
func TestGeneratedEnemiesNeverStartOnPlayer(t *testing.T) {
	for seed := int64(0); seed < 200; seed++ {
		for _, floor := range []int{1, 4, 10} {
			player := newInitialPlayer()
			_, enemies, _, _, _, _ := GenerateRandomMap(70, 70, floor-1, &player, newFloorRNG(seed, floor))
			for _, enemy := range enemies {
				if enemy.X == player.X && enemy.Y == player.Y {
					t.Fatalf("seed %d floor %d: %s placed on the player", seed, floor, enemy.Name)
				}
			}
		}
	}
}

// 通路の外側（壁ではない通れないマス）を投擲物が突き抜け、マップ外を参照して panic していた。
func TestThrowStopsAtBlockedTilesAndMapEdge(t *testing.T) {
	g, _ := newHeadlessGame(1, DifficultyAdvanced)
	m := make([][]Tile, 5)
	for y := range m {
		m[y] = make([]Tile, 5)
		for x := range m[y] {
			m[y][x] = Tile{Type: "other", Blocked: true, BlockSight: true}
		}
	}
	m[2][1], m[2][2], m[2][3] = Tile{Type: "corridor"}, Tile{Type: "corridor"}, Tile{Type: "corridor"}
	g.state.Map = m
	g.rooms = nil
	g.state.Enemies, g.state.Items, g.state.MapTraps = nil, nil, nil
	g.state.Player.X, g.state.Player.Y, g.state.Player.Direction = 2, 2, Up
	item := buildItemFromTemplate(1, 0, 0)
	g.state.Player.Inventory = []Item{item}
	g.selectedItemIndex = 0
	g.throwWithCallbacks(item, 10)
	if err := g.RunUntilInputReady(2000); err != nil {
		t.Fatal(err)
	}
	for _, floorItem := range g.state.Items {
		x, y := floorItem.GetPosition()
		if y < 0 || y >= len(m) || x < 0 || x >= len(m[y]) || m[y][x].Blocked {
			t.Fatalf("thrown item landed at (%d, %d)", x, y)
		}
	}
}

// あやつりクラゲの「勝手に移動」が、予約後に別の敵が入ったマスへプレイヤーを動かしていた。
func TestForcedMoveRechecksDestination(t *testing.T) {
	g, _ := newHeadlessGame(1, DifficultyAdvanced)
	g.state.Map = makeTestFloorMap(5, 5)
	g.rooms = []Room{{ID: 0, X: 0, Y: 0, Width: 5, Height: 5}}
	g.state.Items, g.state.MapTraps = nil, nil
	g.state.Player.X, g.state.Player.Y = 2, 2
	jelly := CreateEnemyByID(10, 1, 1)
	g.state.Enemies = []Enemy{jelly}
	enqueueForcedMove(&g.state.Enemies[0], g, []Coordinate{{X: 3, Y: 2}}, func(int) int { return 0 })
	// 実行前に別の敵が移動先へ入る
	g.state.Enemies = append(g.state.Enemies, CreateEnemyByID(0, 3, 2))
	if err := g.RunUntilInputReady(2000); err != nil {
		t.Fatal(err)
	}
	if g.state.Player.X != 2 || g.state.Player.Y != 2 {
		t.Fatalf("player was forced onto an occupied tile: (%d, %d)", g.state.Player.X, g.state.Player.Y)
	}
}

// 足元のアイテムを投げると、足元ではなく所持品の選択位置のアイテムが消えていた（所持品が空なら panic）。
// FuzzInputSequence が見つけた不具合。
func TestThrowingGroundItemKeepsInventory(t *testing.T) {
	for _, inventorySize := range []int{0, 1} {
		g, _ := newHeadlessGame(1, DifficultyAdvanced)
		g.state.Map = makeTestFloorMap(7, 5)
		g.rooms = []Room{{ID: 0, X: 0, Y: 0, Width: 7, Height: 5}}
		g.state.Enemies, g.state.MapTraps = nil, nil
		g.state.Player.X, g.state.Player.Y, g.state.Player.Direction = 2, 2, Right
		owned := buildItemFromTemplate(2, 0, 0)
		g.state.Player.Inventory = []Item{owned}[:inventorySize]
		ground := buildItemFromTemplate(1, 2, 2)
		g.state.Items = []Item{ground}
		g.currentGroundItem = ground
		g.ShowGroundItem, g.GroundItemActioned = true, true
		g.selectedGroundActionIndex = 3 // 投げる
		g.executeGroundItemAction()
		if err := g.RunUntilInputReady(2000); err != nil {
			t.Fatal(err)
		}
		if len(g.state.Player.Inventory) != inventorySize {
			t.Fatalf("inventory size %d -> %d", inventorySize, len(g.state.Player.Inventory))
		}
		if len(g.state.Items) != 1 {
			t.Fatalf("ground items = %d, want only the thrown item", len(g.state.Items))
		}
		if x, y := g.state.Items[0].GetPosition(); x == 2 && y == 2 {
			t.Fatal("thrown item should have left the player's tile")
		}
	}
}

// 足元のお金を「使う」と、所持品の選択位置をお金として扱い panic していた。FuzzInputSequence が発見。
func TestUsingGroundMoneyAddsCash(t *testing.T) {
	for _, inventorySize := range []int{0, 1} {
		g, _ := newHeadlessGame(1, DifficultyAdvanced)
		g.state.Enemies, g.state.MapTraps = nil, nil
		px, py := g.state.Player.X, g.state.Player.Y
		owned := buildItemFromTemplate(1, 0, 0)
		g.state.Player.Inventory = []Item{owned}[:inventorySize]
		var moneyID int
		for _, id := range randomItemTemplateIDs {
			if _, ok := buildItemFromTemplate(id, 0, 0).(*Money); ok {
				moneyID = id
			}
		}
		cash := buildItemFromTemplate(moneyID, px, py).(*Money)
		cash.Amount = 123
		g.state.Items = []Item{cash}
		g.currentGroundItem = cash
		g.ShowGroundItem, g.GroundItemActioned = true, true
		g.selectedGroundActionIndex = 2 // 使う
		g.executeGroundItemAction()
		if err := g.RunUntilInputReady(2000); err != nil {
			t.Fatal(err)
		}
		if g.state.Player.Cash != 123 || len(g.state.Items) != 0 || len(g.state.Player.Inventory) != inventorySize {
			t.Fatalf("cash %d, ground items %d, inventory %d", g.state.Player.Cash, len(g.state.Items), len(g.state.Player.Inventory))
		}
	}
}

// 目潰し中の敵どうしの攻撃が予約時の添字とポインタを使っていたため、先の攻撃で敵が倒れて並びが変わると、
// ダメージが別の敵に入ったり HP 0 以下の敵が残ったりしていた。TestSimBatch が発見。
func TestBlindEnemyAttacksResolveTargetsAtExecution(t *testing.T) {
	g, _ := newHeadlessGame(1, DifficultyAdvanced)
	g.state.Map = makeTestFloorMap(9, 7)
	g.state.Items, g.state.MapTraps = nil, nil
	g.state.Player.X, g.state.Player.Y = 7, 5
	g.randHooks.damage = func(n int) int { return n - 1 }
	attacker1, victim1 := CreateEnemyByID(5, 1, 1), CreateEnemyByID(0, 2, 1)
	attacker2, victim2 := CreateEnemyByID(5, 1, 3), CreateEnemyByID(0, 2, 3)
	victim1.Health = 1
	victim2.Health, victim2.MaxHealth = 500, 500
	g.state.Enemies = []Enemy{attacker1, victim1, attacker2, victim2}
	g.AttackEnemyFromBlindEnemy(0, 1)
	g.AttackEnemyFromBlindEnemy(2, 3)
	if err := g.RunUntilInputReady(2000); err != nil {
		t.Fatal(err)
	}
	for _, enemy := range g.state.Enemies {
		if enemy.Health <= 0 {
			t.Fatalf("%s remains with HP %d", enemy.Name, enemy.Health)
		}
	}
	if i := g.enemyIndexAt(2, 3); i < 0 || g.state.Enemies[i].Health >= 500 {
		t.Fatal("the second attack should damage its own target")
	}
	if g.enemyIndexAt(2, 1) >= 0 {
		t.Fatal("the first victim should be defeated")
	}
}

// 炸裂装甲のカードの撃破が予約時の添字で行われ、同じ敵ターンに複数の敵が攻撃すると何度も発動して panic していた。
// expert の BOT が発見。罠は1回で消費する。
func TestExplosiveArmorTrapTriggersOnce(t *testing.T) {
	g, _ := newHeadlessGame(1, DifficultyAdvanced)
	g.state.Map = makeTestFloorMap(7, 7)
	g.state.Items, g.state.MapTraps = nil, nil
	g.state.Player.X, g.state.Player.Y = 3, 3
	g.state.Player.SetTrap = buildItemFromTemplate(8, 0, 0)
	g.state.Enemies = []Enemy{CreateEnemyByID(0, 2, 3), CreateEnemyByID(0, 4, 3), CreateEnemyByID(0, 3, 2)}
	for i := range g.state.Enemies {
		g.AttackFromEnemy(i)
	}
	if err := g.RunUntilInputReady(2000); err != nil {
		t.Fatal(err)
	}
	if len(g.state.Enemies) != 2 || g.state.Player.SetTrap != nil {
		t.Fatalf("enemies left %d (want 2), trap %v", len(g.state.Enemies), g.state.Player.SetTrap)
	}
}

// 足元の道具を拾う行動がキューに残ったまま落とし穴で階を移動すると、
// 古い階の添字で新しい階の道具一覧を削除して panic していた（seed=121 itemUser 中級）。
func TestPitfallWaitsForQueuedPickup(t *testing.T) {
	g, _ := newHeadlessGame(121, DifficultyIntermediate)
	px, py := g.state.Player.X, g.state.Player.Y
	// 足元のウインナー（ID 1）を新しい階の道具数より後ろの添字に置く
	for i := 0; i < 40; i++ {
		g.state.Items = append(g.state.Items, g.createItemByID(1, 0, 0))
	}
	item := g.createItemByID(1, px, py)
	g.state.Items = append(g.state.Items, item)
	g.state.Player.Inventory = nil
	floor := g.Floor

	// 実際の順序どおり、罠の発動の後に拾う行動が積まれる
	g.steppedTrapSource = trapSource(pitfallTrapID, "落とし穴")
	pitfallTrapEffect(g)
	g.PickupItem()
	g.isActioned = true
	if err := g.RunUntilInputReady(3000); err != nil {
		t.Fatal(err)
	}
	if g.Floor != floor+1 {
		t.Fatalf("floor = %d, want %d", g.Floor, floor+1)
	}
	if len(g.state.Player.Inventory) != 1 || g.state.Player.Inventory[0] != item {
		t.Fatalf("pickup before falling failed: %v", g.state.Player.Inventory)
	}
}

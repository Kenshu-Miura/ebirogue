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

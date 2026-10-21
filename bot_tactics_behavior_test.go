//go:build !test

package main

import "testing"

// expert の立ち回りと道具の使いどころのテスト（bot_tactics_test.go・bot_expert_items_test.go）。

func newActiveEnemy(id, x, y int) Enemy {
	e := CreateEnemyByID(id, x, y)
	e.StatusAilments.Sleep = 0
	e.PlayerDiscovered = true
	return e
}

// 部屋で2体以上の敵が近づいてきたら、通路の入口へ下がる。
func TestExpertRetreatsToCorridor(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	m := makeTestFloorMap(12, 7)
	for y := 0; y < 7; y++ {
		for x := 8; x < 12; x++ {
			if y != 3 || x == 11 {
				m[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
			} else {
				m[y][x] = Tile{Type: "corridor"}
			}
		}
	}
	g.state.Map = m
	g.state.Player.X, g.state.Player.Y = 6, 3
	g.state.Enemies = []Enemy{newActiveEnemy(0, 3, 1), newActiveEnemy(0, 3, 5)}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if b.act != "retreat" || g.state.Player.X != 7 {
		t.Fatalf("did not retreat toward the corridor: act=%s player=(%d,%d)", b.act, g.state.Player.X, g.state.Player.Y)
	}
}

// 距離2の敵は足踏みで待つ。
func TestExpertWaitsForEnemyAtDistanceTwo(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	g.state.Player.X = 4
	g.state.Enemies = []Enemy{newActiveEnemy(0, 6, 3)}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if b.act != "ambush" || g.state.Player.X != 4 {
		t.Fatalf("did not wait: act=%s player=(%d,%d)", b.act, g.state.Player.X, g.state.Player.Y)
	}
}

// 殴り合うと倒れると見込んだら、止める道具を早めに使う。
func TestExpertUsesItemWhenLosing(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	player := &g.state.Player
	player.X = 4
	player.Health, player.MaxHealth = 100, 100
	player.Inventory = []Item{buildItemFromTemplate(12, 0, 0)} // 睡眠のカード
	for _, pos := range [][2]int{{5, 3}, {5, 2}} {
		e := newActiveEnemy(0, pos[0], pos[1])
		e.Health, e.MaxHealth, e.AttackPower = 300, 300, 40
		g.state.Enemies = append(g.state.Enemies, e)
	}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if b.act != "saveItem" || countInventoryID(g, 12) != 0 {
		t.Fatalf("did not use the sleep card: act=%s", b.act)
	}
}

// 良い指輪だけを付け、両方の枠が埋まっていれば価値の低い指輪と替える。
func TestExpertEquipsGoodRings(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	player := &g.state.Player
	bad := buildItemFromTemplate(177, 0, 0).(*Accessory) // 胃縮小（呪い）
	low1 := buildItemFromTemplate(185, 0, 0).(*Accessory)
	low2 := buildItemFromTemplate(187, 0, 0).(*Accessory)
	good := buildItemFromTemplate(175, 0, 0).(*Accessory) // 回復
	player.Inventory = []Item{bad, low1, low2}
	stepUntil(t, b, 5, func() bool { return player.EquippedAccessories[1] != nil })
	if player.IsEquipped(bad) || !player.IsEquipped(low1) || !player.IsEquipped(low2) {
		t.Fatalf("wrong rings: %v %v", player.EquippedAccessories[0], player.EquippedAccessories[1])
	}
	player.Inventory = append(player.Inventory, good)
	stepUntil(t, b, 5, func() bool { return player.IsEquipped(good) })
	if !player.IsEquipped(good) || player.IsEquipped(bad) {
		t.Fatal("did not swap in the good ring")
	}
}

// 空腹で食料が無ければ、価値の一番低い道具をウインナー化のカードで食料にする。
func TestExpertSausageCardWhenStarving(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	player := &g.state.Player
	player.Satiety = 10
	player.Inventory = []Item{buildItemFromTemplate(cardSausageID, 0, 0), buildItemFromTemplate(37, 0, 0), buildItemFromTemplate(7, 0, 0)}
	stepUntil(t, b, 4, func() bool { return countInventoryID(g, cardSausageID) == 0 })
	if countInventoryID(g, cardSausageID) != 0 || countInventoryID(g, 37) != 0 || countInventoryID(g, 7) != 1 {
		t.Fatalf("wrong target: inventory=%d act=%s", len(player.Inventory), b.act)
	}
	if b.windowOpen() {
		t.Fatal("window left open")
	}
}

// 白紙のカードは、近くに敵がいなければ書き込む（回復の道具が無ければ完全回復を選ぶ）。
func TestExpertWritesBlankCard(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	g.state.Player.Inventory = []Item{buildItemFromTemplate(cardBlankID, 0, 0)}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if countInventoryID(g, 44) != 1 || len(g.state.Player.Inventory) != 1 || b.windowOpen() {
		t.Fatalf("full-heal card not written: id=%d act=%s", g.state.Player.Inventory[0].GetID(), b.act)
	}
}

// 予備の食料を保存の壺へ入れ、空腹になったら出して食べる。
func TestExpertStoresFoodInPot(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	player := &g.state.Player
	pot := buildItemFromTemplate(storagePotID, 0, 0).(*Pot)
	player.Inventory = []Item{pot, buildItemFromTemplate(testFoodID, 0, 0), buildItemFromTemplate(testFoodID, 0, 0)}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if len(pot.Contents) != 1 || countInventoryID(g, testFoodID) != 1 {
		t.Fatalf("spare food not stored: pot=%d act=%s", len(pot.Contents), b.act)
	}
	player.Inventory = []Item{pot}
	player.Satiety = 10
	stepUntil(t, b, 4, func() bool { return player.Satiety > 10 })
	if player.Satiety <= 10 {
		t.Fatalf("did not eat the stored food: act=%s", b.act)
	}
}

// 所持品が満杯でも、床の道具の方が価値が高ければ一番価値の低い道具と入れ替える。不利なカードは拾いに行かない。
func TestExpertSwapsForBetterItem(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	player := &g.state.Player
	for len(player.Inventory) < player.MaxInventory-1 {
		player.Inventory = append(player.Inventory, buildItemFromTemplate(7, 0, 0)) // 黒炎弾
	}
	player.Inventory = append(player.Inventory, buildItemFromTemplate(37, 0, 0)) // 地図忘却（不利）を最後に
	placeVisibleItem(g, buildItemFromTemplate(2, 4, 3))                          // 回復の薬
	stepUntil(t, b, 10, func() bool { return countInventoryID(g, 2) > 0 })
	if countInventoryID(g, 2) != 1 || countInventoryID(g, 37) != 0 || countInventoryID(g, 7) != player.MaxInventory-1 {
		t.Fatalf("did not swap: heal=%d harmful=%d act=%s", countInventoryID(g, 2), countInventoryID(g, 37), b.act)
	}
	if b.windowOpen() || g.GroundItemActioned {
		t.Fatalf("window left open or ground action state left: %v", g.GroundItemActioned)
	}
	if b.expertItemGoal(4, 3) {
		t.Fatal("wants to pick the harmful card back up")
	}
}

// 水路の向こうでたどり着けない敵は「近づいてくる敵」と数えず、通路へ下がらない（段9の統合で追加。
// 以前は下がる→部屋へ出る→下がるを繰り返して突風で倒れていた）。
func TestExpertIgnoresEnemiesAcrossWater(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	m := makeTestFloorMap(12, 7)
	for y := 0; y < 7; y++ {
		for x := 8; x < 12; x++ {
			if y != 3 || x == 11 {
				m[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
			} else {
				m[y][x] = Tile{Type: "corridor"}
			}
		}
		if y > 0 && y < 6 {
			m[y][4] = Tile{Type: "water", Blocked: true}
		}
	}
	g.state.Map = m
	g.state.Player.X, g.state.Player.Y = 6, 3
	g.state.Enemies = []Enemy{newActiveEnemy(0, 3, 1), newActiveEnemy(0, 3, 5)}
	if n := len(b.approachingEnemies(expertThreatRange)); n != 0 {
		t.Fatalf("enemies across water counted as approaching: %d", n)
	}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if b.act == "retreat" {
		t.Fatal("retreated from enemies that cannot reach")
	}
}

// 金縛りの敵が1マスの隙間をふさいでいれば、その後ろの敵も近づいてこない。
func TestExpertIgnoresEnemiesBehindParalyzedBlocker(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	for y := 1; y < 6; y++ {
		if y != 3 {
			g.state.Map[y][4] = Tile{Type: "wall", Blocked: true, BlockSight: true}
		}
	}
	g.state.Player.X, g.state.Player.Y = 7, 3
	blocker := newActiveEnemy(0, 4, 3)
	blocker.StatusAilments.Paralysis = true
	g.state.Enemies = []Enemy{blocker, newActiveEnemy(0, 3, 2), newActiveEnemy(0, 3, 4)}
	if n := len(b.approachingEnemies(expertThreatRange)); n != 0 {
		t.Fatalf("blocked enemies counted as approaching: %d", n)
	}
	g.state.Enemies = g.state.Enemies[1:] // 隙間が空けば近づいてくる
	if n := len(b.approachingEnemies(expertThreatRange)); n != 2 {
		t.Fatalf("approaching = %d, want 2", n)
	}
}

// 隣に回復役（イヤシナマコ系）がいれば先に殴る。
func TestExpertAttacksHealerFirst(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	g.state.Player.X, g.state.Player.Y = 4, 3
	g.state.Player.Health, g.state.Player.MaxHealth = 200, 200
	g.state.Enemies = []Enemy{newActiveEnemy(10, 4, 2), newActiveEnemy(78, 4, 4)}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if b.act != "melee" || g.state.Player.Direction != determineDirection(0, 1) {
		t.Fatalf("did not attack the healer: act=%s dir=%v", b.act, g.state.Player.Direction)
	}
}

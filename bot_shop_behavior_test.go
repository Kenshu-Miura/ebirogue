//go:build !test

package main

import "testing"

// wave11-C: BOT の店の売買・泥棒をしない工夫・階段の前の休憩の小さな盤面のテスト。
// 盤面は content_shop_test.go の newShopTestGame（店 x0〜9・店主 (8,3)・出口 (9,4)、隣の部屋の階段 (17,4)）。

func shopTestPricedItem(g *Game, id, x, y int) Item {
	item := buildItemFromTemplate(id, x, y)
	setItemShopPrice(item, itemBuyPrice(item))
	g.state.Items = append(g.state.Items, item)
	return item
}

// runBotSteps は BOT を最大 steps 回動かし、done が true になったら止める。
func runBotSteps(t *testing.T, b *bot, steps int, done func() bool) {
	t.Helper()
	for i := 0; i < steps && !done() && !b.g.RunEnded(); i++ {
		if err := b.step(); err != nil {
			t.Fatalf("step %d: %v at (%d,%d)", i, err, b.g.state.Player.X, b.g.state.Player.Y)
		}
	}
}

// enableBotFeature は既定で切ってある改善 name をこのテストの間だけ入れる。
func enableBotFeature(t *testing.T, name string) {
	t.Helper()
	botFeature(name) // 既定の設定を読み込ませる
	was := botOffSet[name]
	delete(botOffSet, name)
	t.Cleanup(func() {
		if was {
			botOffSet[name] = true
		}
	})
}

func assertNoTheft(t *testing.T, g *Game) {
	t.Helper()
	if g.stats.ShopThefts != 0 {
		t.Fatal("bot stole from the shop")
	}
	if k := g.shopkeeperIndex(); k >= 0 && g.Floor == 5 && !g.shopOpen() {
		t.Fatal("bot angered the keeper")
	}
}

// 店の中で価値の低い道具（自分に不利なカード）を床に置いて売る。
func TestExpertSellsJunkInShop(t *testing.T) {
	g := newShopTestGame(t)
	card := buildItemFromTemplate(35, 0, 0) // モンスターハウスのカード（不利・価値0）
	g.state.Player.Inventory = []Item{card}
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	runBotSteps(t, b, 30, func() bool { return g.stats.ShopSold > 0 })
	if g.stats.ShopSold != 1 || g.state.Player.Cash != itemSellPrice(card) || !g.itemOnFloor(card) || itemShopPrice(card) == 0 {
		t.Fatalf("sold %d cash %d price %d", g.stats.ShopSold, g.state.Player.Cash, itemShopPrice(card))
	}
	assertNoTheft(t, g)
}

// 所持金で買える食料を拾い、店主に話しかけて払う。
func TestExpertBuysFoodInShop(t *testing.T) {
	g := newShopTestGame(t)
	food := shopTestPricedItem(g, 1, 2, 6)
	price := itemShopPrice(food)
	g.state.Player.Cash = price + 50
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	runBotSteps(t, b, 80, func() bool { return g.stats.ShopBought > 0 })
	if g.stats.ShopBought != 1 || g.state.Player.Cash != 50 || itemShopPrice(food) != 0 || g.shopUnpaidTotal() != 0 {
		t.Fatalf("bought %d cash %d price %d unpaid %d", g.stats.ShopBought, g.state.Player.Cash, itemShopPrice(food), g.shopUnpaidTotal())
	}
	if b.findInventory(func(item Item) bool { return item == food }) < 0 {
		t.Fatal("food is not in the inventory")
	}
	assertNoTheft(t, g)
}

// 所持品が満杯なら、先に価値の低い道具を売って空けてから買う。
func TestExpertSellsBeforeBuyingWhenFull(t *testing.T) {
	g := newShopTestGame(t)
	g.state.Items = nil
	food := shopTestPricedItem(g, 1, 2, 6)
	inv := []Item{buildItemFromTemplate(35, 0, 0)}
	for len(inv) < g.state.Player.MaxInventory {
		inv = append(inv, buildItemFromTemplate(7, 0, 0)) // 黒炎弾のカード（価値70）
	}
	g.state.Player.Inventory = inv
	g.state.Player.Cash = itemShopPrice(food)
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	runBotSteps(t, b, 120, func() bool { return g.stats.ShopBought > 0 })
	if g.stats.ShopSold != 1 || g.stats.ShopBought != 1 || itemShopPrice(food) != 0 {
		t.Fatalf("sold %d bought %d", g.stats.ShopSold, g.stats.ShopBought)
	}
	assertNoTheft(t, g)
}

// 払えない商品は店の中に置いて返し、同じ階では買いに戻らない。
func TestExpertReturnsUnaffordableItem(t *testing.T) {
	g := newShopTestGame(t)
	product := pickUpShopProduct(t, g) // こん棒 200円、所持金0
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	runBotSteps(t, b, 5, func() bool { return g.itemOnFloor(product) })
	if !g.itemOnFloor(product) || itemShopPrice(product) == 0 || b.unpaidTotal() != 0 {
		t.Fatal("unpaid item was not returned")
	}
	g.state.Player.Cash = 100 // まだ足りない
	runBotSteps(t, b, 40, func() bool { return g.Floor != 5 || !g.itemOnFloor(product) })
	if g.Floor == 5 && !g.itemOnFloor(product) || b.findInventory(func(item Item) bool { return item == product }) >= 0 {
		t.Fatal("bot picked up the rejected item again")
	}
	assertNoTheft(t, g)
}

// 未払いの品を持ったまま店の外へ出る1歩は踏まない（払えるなら払ってから出る）。
func TestExpertDoesNotLeaveShopUnpaid(t *testing.T) {
	g := newShopTestGame(t)
	product := pickUpShopProduct(t, g)
	g.state.Player.Cash = 1000
	g.state.Player.X, g.state.Player.Y = 9, 4 // 出口
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	if !b.leavesShopUnpaid(1, 0) || b.leavesShopUnpaid(-1, 0) {
		t.Fatal("leaving check")
	}
	if err := b.move(1, 0); err != nil {
		t.Fatal(err)
	}
	if g.state.Player.X != 9 || g.stats.ShopThefts != 0 {
		t.Fatal("bot stepped out of the shop with an unpaid item")
	}
	runBotSteps(t, b, 40, func() bool { return itemShopPrice(product) == 0 })
	if itemShopPrice(product) != 0 || g.state.Player.Cash != 1000-200 {
		t.Fatalf("bot did not pay: cash %d", g.state.Player.Cash)
	}
	assertNoTheft(t, g)
}

// 売買のある店で何度遊んでも泥棒にならず、店主を怒らせずに階を降りる。
func TestExpertShopTradeDoesNotSteal(t *testing.T) {
	for seed := int64(1); seed <= 5; seed++ {
		g := newShopTestGame(t)
		for i, xy := range []Coordinate{{2, 2}, {5, 2}, {2, 6}, {6, 6}} {
			shopTestPricedItem(g, []int{1, 2, 57, 7}[i], xy.X, xy.Y)
		}
		g.state.Player.Inventory = []Item{buildItemFromTemplate(35, 0, 0), buildItemFromTemplate(36, 0, 0)}
		g.state.Player.Cash = 1500
		b := newBot(g, g.input.(*VirtualInput), personaExpert, seed)
		runBotSteps(t, b, 600, func() bool { return g.Floor != 5 })
		if g.Floor == 5 {
			t.Fatalf("seed %d: bot did not leave the shop floor", seed)
		}
		if g.stats.ShopSold == 0 || g.stats.ShopBought == 0 {
			t.Fatalf("seed %d: sold %d bought %d", seed, g.stats.ShopSold, g.stats.ShopBought)
		}
		assertNoTheft(t, g)
	}
}

// 直線上の敵の後ろに店主がいれば撃たない・投げない（外れた矢・貫通する矢が店主に当たる）。
func TestBotDoesNotShootTowardKeeper(t *testing.T) {
	g := newShopTestGame(t)
	g.state.Enemies = append(g.state.Enemies, CreateEnemyByID(0, 4, 3))
	g.state.Player.X, g.state.Player.Y = 2, 3
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	if !b.lineHitsShopkeeper(1, 0) {
		t.Fatal("keeper behind the enemy must be detected")
	}
	if _, ok := b.enemyInLine(1, 0, 8); ok {
		t.Fatal("must not target an enemy with the keeper behind it")
	}
	g.state.Player.X, g.state.Player.Y = 4, 5
	g.state.Enemies[1].X, g.state.Enemies[1].Y = 4, 3
	if _, ok := b.enemyInLine(0, -1, 8); !ok || b.lineHitsShopkeeper(0, -1) {
		t.Fatal("a line without the keeper is fine")
	}
}

// 三方向の武器の横の振りが店主に当たるなら振らない。
func TestBotThreeWaySwingAvoidsKeeper(t *testing.T) {
	g := newShopTestGame(t)
	naginata := buildItemFromTemplate(57, 0, 0).(*Weapon)
	g.state.Player.Inventory = []Item{naginata}
	g.state.Player.EquippedWeapon = naginata
	g.state.Player.X, g.state.Player.Y = 7, 4
	g.state.Enemies = append(g.state.Enemies, CreateEnemyByID(0, 7, 5))
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	// 下を向いて振ると、横の振りは (8,5) と (6,5)。右を向くと (8,3) の店主に当たる
	if b.swingHitsShopkeeper(0, 1) || !b.swingHitsShopkeeper(1, 0) {
		t.Fatal("swing check")
	}
	if err := b.attack(1, 0); err != nil {
		t.Fatal(err)
	}
	keeper := g.state.Enemies[g.shopkeeperIndex()]
	if keeper.Health != keeper.MaxHealth || !g.shopOpen() || g.stats.ShopThefts != 0 {
		t.Fatal("bot hit the keeper with a three-way swing")
	}
}

// 店主の隣の調べていないマスへは、踏み込む前に Z で罠を調べる（大型地雷の爆風が店主に当たるため）。
func TestBotSearchesTrapNextToKeeper(t *testing.T) {
	g := newShopTestGame(t)
	g.state.MapTraps = []MapTrap{createMapTrapByID(11, 7, 3)}
	g.state.Player.X, g.state.Player.Y = 6, 3
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	turn := g.turnCount
	if err := b.move(1, 0); err != nil {
		t.Fatal(err)
	}
	if g.state.Player.X != 6 || !g.state.MapTraps[0].Discovered || g.turnCount == turn {
		t.Fatalf("bot must search first: pos %d discovered %v", g.state.Player.X, g.state.MapTraps[0].Discovered)
	}
	// 見つけた大型地雷は、敵・罠を押しのける経路でも踏まない
	if dx, dy, ok := b.pathStepThrough(func(x, y int) bool { return x == 7 && y == 3 }, true); ok {
		t.Fatalf("path onto the mine next to the keeper: %d,%d", dx, dy)
	}
	assertNoTheft(t, g)
}

// 降りる前に、敵がいず満腹度に余裕があれば階段の上で HP を戻してから降りる（-bot.on=restbeforestairs のとき）。
func TestExpertRestsOnStairsBeforeDescending(t *testing.T) {
	enableBotFeature(t, "restbeforestairs") // 計測で下がったので既定では切ってある
	g := newShopTestGame(t)
	p := &g.state.Player
	p.X, p.Y = 15, 4 // 2歩で階段（着くと確認の窓が開く）
	p.Health = p.MaxHealth * 7 / 10
	p.Satiety = 80
	g.floorTurns = 800 // 長居したので降りたい
	if err := g.RunUntilInputReady(100); err != nil {
		t.Fatal(err)
	}
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	runBotSteps(t, b, 10, func() bool { return p.X == 17 && b.stairsRest > 0 })
	start := p.Health
	runBotSteps(t, b, 5, func() bool { return false })
	if g.Floor != 5 || p.Health <= start {
		t.Fatalf("bot must rest on the stairs: floor %d hp %d→%d", g.Floor, start, p.Health)
	}
	runBotSteps(t, b, 100, func() bool { return g.Floor != 5 })
	if g.Floor == 5 {
		t.Fatal("bot did not descend after resting")
	}
	if b.stairsRest == 0 || b.stairsRest > botStairsRestMax {
		t.Fatalf("rest turns %d", b.stairsRest)
	}
}

// 満腹度が少なければ階段の上で休まず降りる。
func TestExpertDoesNotRestOnStairsWhenHungry(t *testing.T) {
	enableBotFeature(t, "restbeforestairs")
	g := newShopTestGame(t)
	p := &g.state.Player
	p.X, p.Y = 15, 4
	p.Health = p.MaxHealth * 7 / 10
	p.Satiety = botStairsRestSatiety
	g.floorTurns = 800
	if err := g.RunUntilInputReady(100); err != nil {
		t.Fatal(err)
	}
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	runBotSteps(t, b, 6, func() bool { return g.Floor != 5 })
	if g.Floor == 5 || b.stairsRest != 0 {
		t.Fatal("hungry bot must not rest on the stairs")
	}
}

// 階段を見つけていて、拾う価値のある道具が見えず、長居していれば探索を切り上げる。
func TestExpertCutsExploration(t *testing.T) {
	g := newShopTestGame(t)
	g.state.Map[4][17].Visited = true
	g.state.Map[4][15].Visited = false // 未探索のマスを残す
	g.state.Player.Satiety = 100
	g.state.Player.Inventory = []Item{buildItemFromTemplate(1, 0, 0)}
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	g.floorTurns = 100
	if b.expertCutExploration(50) {
		t.Fatal("must keep exploring early on the floor")
	}
	g.floorTurns = botLeaveFloorTurns + 1
	if !b.expertCutExploration(50) {
		t.Fatal("must leave after a long stay")
	}
	g.floorTurns = 100
	g.state.Player.Satiety = 40
	if !b.expertCutExploration(50) {
		t.Fatal("must leave when food is short")
	}
	g.state.Map[4][17].Visited = false
	if b.expertCutExploration(50) {
		t.Fatal("must not leave before finding the stairs")
	}
}

// 遠投の指輪（壁を抜けて飛ぶ）では壁の向こうの店主も、火迅の太刀（HP 満タンで直線の炎）では
// 正面の敵の後ろの店主も巻き込むとみなす。
func TestBotKeeperLineWithLongThrowAndFlameBlade(t *testing.T) {
	g := newShopTestGame(t)
	p := &g.state.Player
	p.X, p.Y = 15, 3 // 隣の部屋。店主 (8,3) との間は壁
	b := newBot(g, g.input.(*VirtualInput), personaExpert, 1)
	if b.lineHitsShopkeeper(-1, 0) {
		t.Fatal("a wall stops normal projectiles")
	}
	ring := buildItemFromTemplate(189, 0, 0).(*Accessory)
	p.Inventory = []Item{ring}
	p.EquippedAccessories[0] = ring
	if !b.lineHitsShopkeeper(-1, 0) {
		t.Fatal("long throw pierces walls")
	}
	p.EquippedAccessories[0] = nil
	p.Inventory = nil

	tachi := buildItemFromTemplate(146, 0, 0).(*Weapon)
	p.Inventory = []Item{tachi}
	p.EquippedWeapon = tachi
	p.X, p.Y = 5, 3
	g.state.Enemies = append(g.state.Enemies, CreateEnemyByID(0, 6, 3))
	p.Health = p.MaxHealth
	if !b.swingHitsShopkeeper(1, 0) {
		t.Fatal("the flame flies past the front enemy to the keeper")
	}
	p.Health = p.MaxHealth - 1
	if b.swingHitsShopkeeper(1, 0) {
		t.Fatal("no flame without full HP")
	}
}

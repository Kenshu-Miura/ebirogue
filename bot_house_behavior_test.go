//go:build !test

package main

import "testing"

// wave11-A の expert の判断（bot_house_test.go・bot_terrain_test.go）を小さな盤面で確かめる。

// markAllVisited は盤面のすべてのマスを見たことにする（探索の目標を消す）。
func markAllVisited(g *Game) {
	for y := range g.state.Map {
		for x := range g.state.Map[y] {
			g.state.Map[y][x].Visited = true
		}
	}
}

// newHouseTestGame は 15x9 の部屋（右の壁の真ん中から通路が右へ伸びる）に、起きている敵 n 体を置く。
// 部屋は ID 0、通路は x=15〜19, y=4。
func newHouseTestGame(t *testing.T, n int) (*Game, *bot) {
	t.Helper()
	g, b := newBotFoodTestGame(t, personaExpert)
	m := makeTestFloorMap(21, 9)
	for y := 0; y < 9; y++ {
		for x := 14; x < 21; x++ {
			switch {
			case y == 4 && x < 20:
				m[y][x] = Tile{Type: "corridor"}
			default:
				m[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
			}
		}
	}
	g.state.Map = m
	g.rooms = []Room{{ID: 0, X: 0, Y: 0, Width: 15, Height: 9, Center: Coordinate{X: 7, Y: 4}}}
	markAllVisited(g)
	p := &g.state.Player
	p.X, p.Y = 4, 4
	p.Health, p.MaxHealth = 60, 60
	g.state.Enemies = nil
	for i := 0; i < n; i++ {
		g.state.Enemies = append(g.state.Enemies, newActiveEnemy(0, 1+i%3, 1+i/3))
	}
	return g, b
}

// 「モンスターハウスだ！」の通知が出たら、今いる部屋をモンスターハウスとして覚える。
func TestExpertNoticesMonsterHouseMessage(t *testing.T) {
	g, b := newHouseTestGame(t, 2)
	b.newMessages() // ここまでのメッセージは読んだことにする
	g.messageLog.Add(houseNotice)
	b.updateHouse()
	if b.house.floor != g.Floor || b.house.room != 0 {
		t.Fatalf("house not noticed: %+v", b.house)
	}
	// 次の階では忘れる
	g.Floor++
	b.updateHouse()
	if b.house.floor != 0 {
		t.Fatalf("house kept on the next floor: %+v", b.house)
	}
}

// 通知を見逃しても、部屋に起きている敵が多く見えればモンスターハウスとみなす。少なければみなさない。
func TestExpertNoticesCrowdedRoom(t *testing.T) {
	g, b := newHouseTestGame(t, houseCrowdMin-1)
	b.updateHouse()
	if b.house.floor != 0 {
		t.Fatalf("a few enemies counted as a house: %+v", b.house)
	}
	g.state.Enemies = append(g.state.Enemies, newActiveEnemy(0, 3, 7))
	b.updateHouse()
	if b.house.floor != g.Floor {
		t.Fatal("crowded room not noticed")
	}
}

// 部屋に敵が多ければ、全滅のカードを先に使う。
func TestExpertHouseUsesAnnihilationCard(t *testing.T) {
	g, b := newHouseTestGame(t, 6)
	g.state.Player.Inventory = []Item{buildItemFromTemplate(56, 0, 0)}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if b.act != "houseSweep" || countInventoryID(g, 56) != 0 {
		t.Fatalf("did not use the annihilation card: act=%s", b.act)
	}
}

// 片づける道具が無ければ、ふだん（5歩）より遠い通路の入口まで下がる。
func TestExpertHouseRetreatsToFarCorridor(t *testing.T) {
	g, b := newHouseTestGame(t, 6)
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if b.act != "houseRetreat" || g.state.Player.X != 5 {
		t.Fatalf("did not retreat: act=%s player=(%d,%d)", b.act, g.state.Player.X, g.state.Player.Y)
	}
	t.Run("off", func(t *testing.T) {
		g, b := newHouseTestGame(t, 6)
		botFeature("house") // -bot.off の読み込みを済ませてから一時的に切る
		prev := botOffSet
		botOffSet = map[string]bool{"house": true}
		defer func() { botOffSet = prev }()
		if err := b.step(); err != nil {
			t.Fatal(err)
		}
		if b.act == "houseRetreat" {
			t.Fatalf("house tactics used while off: player=(%d,%d)", g.state.Player.X, g.state.Player.Y)
		}
	})
}

// 階段が近く、敵が届かなければ階段へ向かい、着いたら降りる。
func TestExpertHouseTakesNearStairs(t *testing.T) {
	g, b := newHouseTestGame(t, 6)
	g.state.Map[4][7] = Tile{Type: "stairs", Visited: true}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if b.act != "houseStairs" || g.state.Player.X != 5 {
		t.Fatalf("did not head for the stairs: act=%s player=(%d,%d)", b.act, g.state.Player.X, g.state.Player.Y)
	}
	if !b.expertWantsToDescend() {
		t.Fatal("does not want to descend while the house is awake")
	}
	// 敵が階段の近くにいれば向かわない
	g, b = newHouseTestGame(t, 6)
	g.state.Map[4][7] = Tile{Type: "stairs", Visited: true}
	g.state.Enemies[0].X, g.state.Enemies[0].Y = 8, 3
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if b.act == "houseStairs" {
		t.Fatal("headed for stairs the enemies can reach first")
	}
}

// 部屋に起きている敵が残っている間は、部屋の道具を拾いに行かない。片づけたら拾う。
func TestExpertHouseItemsWaitUntilCleared(t *testing.T) {
	g, b := newHouseTestGame(t, 6)
	b.house = botHouse{floor: g.Floor, room: 0}
	placeVisibleItem(g, buildItemFromTemplate(7, 6, 6))
	if b.expertItemGoalFunc()(6, 6) {
		t.Fatal("wants the item while the house is awake")
	}
	g.state.Enemies = nil
	if !b.expertItemGoalFunc()(6, 6) {
		t.Fatal("does not want the item after clearing the house")
	}
}

// 割れた壁を壊すと近道になるなら、通常攻撃で壊して通る。店の壁の割れた壁は壊さない。
func TestExpertBreaksCrackedWallShortcut(t *testing.T) {
	newGame := func(t *testing.T) (*Game, *bot) {
		g, b := newBotFoodTestGame(t, personaExpert)
		// 縦の壁 x=4 で左右に分かれ、下の端 y=5 だけがつながっている。y=1 の壁が割れている
		m := makeTestFloorMap(9, 7)
		for y := 1; y < 5; y++ {
			m[y][4] = Tile{Type: "wall", Blocked: true, BlockSight: true}
		}
		m[1][4].Type = "cracked_wall"
		g.state.Map = m
		markAllVisited(g)
		g.state.Map[1][7].Visited = false // 右上の未探索のマス
		g.state.Player.X, g.state.Player.Y = 3, 1
		return g, b
	}
	g, b := newGame(t)
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if g.state.Map[1][4].Type != "corridor" || g.state.Player.X != 3 {
		t.Fatalf("did not break the cracked wall: tile=%s act=%s player=(%d,%d)", g.state.Map[1][4].Type, b.act, g.state.Player.X, g.state.Player.Y)
	}
	t.Run("shop", func(t *testing.T) {
		g, b := newGame(t)
		g.rooms = append(g.rooms, Room{ID: 1, X: 4, Y: 0, Width: 5, Height: 4, Role: RoomRoleShop})
		if b.crackedWallUsable(4, 1) {
			t.Fatal("shop wall counted as breakable")
		}
		if err := b.step(); err != nil {
			t.Fatal(err)
		}
		if g.state.Map[1][4].Type != "cracked_wall" {
			t.Fatal("broke the shop wall")
		}
	})
	t.Run("unseen", func(t *testing.T) {
		g, b := newGame(t)
		g.state.Map[1][4].Visited = false
		if b.crackedWallUsable(4, 1) {
			t.Fatal("unseen cracked wall counted as breakable")
		}
	})
}

// 行き先が無ければ、部屋の壁ぎわを調べて隠し通路を探す。見つけたら通る。
func TestExpertSearchesHiddenCorridorWhenStuck(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	m := makeTestFloorMap(13, 7)
	for y := 0; y < 7; y++ {
		for x := 8; x < 13; x++ {
			m[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
		}
	}
	m[3][8] = Tile{Type: "hidden_corridor", Blocked: true, BlockSight: true}
	m[3][9] = Tile{Type: "hidden_corridor", Blocked: true, BlockSight: true}
	m[3][10] = Tile{Type: "corridor"}
	m[3][11] = Tile{Type: "stairs"}
	g.state.Map = m
	g.rooms = []Room{{ID: 0, X: 0, Y: 0, Width: 9, Height: 7, Center: Coordinate{X: 4, Y: 3}}}
	markAllVisited(g)
	g.state.Player.X, g.state.Player.Y = 7, 3
	stepUntil(t, b, 60, func() bool { return g.state.Map[3][8].Type == "corridor" })
	if g.state.Map[3][8].Type != "corridor" {
		t.Fatalf("hidden corridor not found: act=%s searches=%d", b.act, b.searchCount)
	}
	if b.searchCount > hiddenSearchMax {
		t.Fatalf("searched too many times: %d", b.searchCount)
	}
}

// 逃げ回るだけのニゲハゼ系は、近くにいても脅威に数えず、自分からは追わない。
func TestExpertIgnoresRunawayFish(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	g.state.Player.X = 4
	g.state.Enemies = []Enemy{newActiveEnemy(73, 6, 1), newActiveEnemy(73, 6, 5)}
	if n := len(b.approachingEnemies(expertThreatRange)); n != 0 {
		t.Fatalf("runaway fish counted as approaching: %d", n)
	}
	if b.worthChasing(g.state.Enemies[0]) {
		t.Fatal("wants to chase a runaway fish")
	}
}

// 追う敵は歩いて届く距離で選ぶ（水路の向こうの敵は追わない）。
func TestExpertChasesOnlyReachableEnemies(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	for y := 0; y < 7; y++ {
		g.state.Map[y][5] = Tile{Type: "water", Blocked: true}
	}
	g.state.Player.X = 2
	enemy := newActiveEnemy(0, 7, 3)
	g.state.Enemies = []Enemy{enemy}
	if b.worthChasing(enemy) {
		t.Fatal("wants to chase an enemy across the water")
	}
	g.state.Map[3][5] = Tile{Type: "floor"}
	if !b.worthChasing(enemy) {
		t.Fatal("does not chase a reachable enemy")
	}
}

// 商売中の店がある階では、道具寄せのカードを使わない（商品が店の外へ集まって動けなくなる）。
func TestExpertSkipsGatherCardWithOpenShop(t *testing.T) {
	g, b := newBotFoodTestGame(t, personaExpert)
	placeVisibleItem(g, buildItemFromTemplate(7, 7, 5))
	g.state.Player.X, g.state.Player.Y = 1, 1
	if !b.expertBuffUsable(171) {
		t.Fatal("gather card not usable without a shop")
	}
	g.state.Enemies = []Enemy{CreateEnemyByID(shopkeeperMonsterID, 7, 1)}
	if !g.shopOpen() {
		t.Fatal("shop not open in the test")
	}
	if b.expertBuffUsable(171) {
		t.Fatal("gather card used while a shop is open")
	}
}

// 部屋の敵を眠らせた後は、隣の眠った敵を殴って起こさず、階段へ向かう（階段が無ければ部屋の外へ出る）。
func TestExpertHouseSneaksPastSleepers(t *testing.T) {
	g, b := newHouseTestGame(t, 6)
	b.house = botHouse{floor: g.Floor, room: 0}
	for i := range g.state.Enemies {
		g.state.Enemies[i].StatusAilments.Sleep = 50
	}
	g.state.Enemies[0].X, g.state.Enemies[0].Y = 5, 4 // 隣で眠っている
	g.state.Map[4][18] = Tile{Type: "stairs", Visited: true}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if b.act != "houseSneakStairs" {
		t.Fatalf("did not sneak toward the stairs: act=%s", b.act)
	}
	if g.state.Enemies[0].StatusAilments.Sleep == 0 {
		t.Fatal("woke the sleeping enemy")
	}
	// 階段が無ければ部屋の外へ出る
	g, b = newHouseTestGame(t, 6)
	b.house = botHouse{floor: g.Floor, room: 0}
	for i := range g.state.Enemies {
		g.state.Enemies[i].StatusAilments.Sleep = 50
	}
	if err := b.step(); err != nil {
		t.Fatal(err)
	}
	if b.act != "houseSneakOut" {
		t.Fatalf("did not leave the room: act=%s", b.act)
	}
	if b.worthChasing(g.state.Enemies[1]) {
		t.Fatal("wants to chase an enemy in the crowded house")
	}
}

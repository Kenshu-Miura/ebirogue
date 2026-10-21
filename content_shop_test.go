//go:build !test

package main

import (
	"testing"
)

// 店・売買・店主・泥棒（wave10-B）のテスト。

// newShopTestGame は左に店（10x9、出口は (9,4)）、通路を挟んで右に普通の部屋（階段 (17,4)）を置いたゲームを作る。
// 店主は出口の内側 (8,4) の上 (8,3)、商品はこん棒（200円）を (3,3) に置く。海老さんは店の (4,4)。
func newShopTestGame(t *testing.T) *Game {
	t.Helper()
	g, _ := newHeadlessGame(7, DifficultyAdvanced)
	width, height := 21, 9
	m := make([][]Tile, height)
	for y := range m {
		m[y] = make([]Tile, width)
		for x := range m[y] {
			m[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
		}
	}
	shop := Room{ID: 0, X: 0, Y: 0, Width: 10, Height: 9, Center: Coordinate{X: 4, Y: 4}, Role: RoomRoleShop}
	other := Room{ID: 1, X: 13, Y: 0, Width: 8, Height: 9, Center: Coordinate{X: 16, Y: 4}}
	for _, r := range []Room{shop, other} {
		for y := r.Y + 1; y < r.Y+r.Height-1; y++ {
			for x := r.X + 1; x < r.X+r.Width-1; x++ {
				m[y][x] = Tile{Type: "floor"}
			}
		}
	}
	for x := 9; x <= 13; x++ {
		m[4][x] = Tile{Type: "corridor"}
	}
	m[4][17] = Tile{Type: "stairs"}
	g.state.Map = m
	g.rooms = []Room{shop, other}
	g.state.Enemies = []Enemy{CreateEnemyByID(shopkeeperMonsterID, 8, 3)}
	g.state.MapTraps = nil
	g.state.Items = nil
	product := shopTestWeapon(0)
	product.SetPosition(3, 3)
	setItemShopPrice(product, itemBuyPrice(product))
	g.state.Items = []Item{product}
	g.state.Player.X, g.state.Player.Y = 4, 4
	g.state.Player.Inventory = nil
	g.state.Player.Cash = 0
	g.Floor = 5
	if err := g.RunUntilInputReady(100); err != nil {
		t.Fatal(err)
	}
	return g
}

// shopTestWeapon はこん棒（基準 200円）を強化値 sharpness で作る。
func shopTestWeapon(sharpness int) *Weapon {
	w := buildItemFromTemplate(20, 0, 0).(*Weapon)
	w.Sharpness, w.Cursed = sharpness, sharpness < 0
	return w
}

func pickUpShopProduct(t *testing.T, g *Game) Item {
	t.Helper()
	product := g.state.Items[0]
	g.PickUpItem(product, 0)
	if err := g.RunUntilInputReady(100); err != nil {
		t.Fatal(err)
	}
	return product
}

func TestItemPriceTableCoversAllTemplates(t *testing.T) {
	for id, template := range itemTemplates {
		price, ok := itemBasePrices[id]
		if !ok {
			t.Errorf("アイテム %d（%s）に値段が無い（item_prices.go の itemBasePrices に追加する）", id, template.Name)
			continue
		}
		if template.ItemType == "Money" {
			if price != 0 {
				t.Errorf("小銭 %d は売買しないので値段0にする", id)
			}
			continue
		}
		if price <= 0 {
			t.Errorf("アイテム %d（%s）の値段が %d", id, template.Name, price)
		}
	}
	for id := range itemBasePrices {
		if _, ok := itemTemplates[id]; !ok {
			t.Errorf("値段表に存在しないアイテム %d がある", id)
		}
	}
}

func TestItemPricesBySharpnessCurseUsesCapacity(t *testing.T) {
	cases := []struct {
		name string
		item Item
		buy  int
	}{
		{"こん棒+0", shopTestWeapon(0), 200},
		{"こん棒+3", shopTestWeapon(3), 320},
		{"呪われたこん棒-1", shopTestWeapon(-1), 80},
	}
	cane := buildItemFromTemplate(9, 0, 0).(*Cane) // シフトチェンジの杖 800
	cane.Uses = 5
	cases = append(cases, struct {
		name string
		item Item
		buy  int
	}{"杖5回", cane, 1200})
	emptyCane := buildItemFromTemplate(9, 0, 0).(*Cane)
	emptyCane.Uses = 0
	cases = append(cases, struct {
		name string
		item Item
		buy  int
	}{"杖0回", emptyCane, 800})
	pot := buildItemFromTemplate(46, 0, 0).(*Pot) // 保存の壺 500
	pot.Capacity = 3
	cases = append(cases, struct {
		name string
		item Item
		buy  int
	}{"壺の容量3", pot, 800})
	arrow := buildItemFromTemplate(6, 0, 0).(*Arrow) // 1本20
	arrow.ShotCount = 10
	cases = append(cases, struct {
		name string
		item Item
		buy  int
	}{"矢10本", arrow, 200})
	ring := buildItemFromTemplate(175, 0, 0).(*Accessory) // 回復の指輪 3000
	ring.Cursed = true
	cases = append(cases, struct {
		name string
		item Item
		buy  int
	}{"呪われた指輪", ring, 1500})
	for _, c := range cases {
		if got := itemBuyPrice(c.item); got != c.buy {
			t.Errorf("%s: 買値 %d, want %d", c.name, got, c.buy)
		}
		if got := itemSellPrice(c.item); got != c.buy/2 {
			t.Errorf("%s: 売値 %d, want %d", c.name, got, c.buy/2)
		}
	}
	if itemBuyPrice(buildItemFromTemplate(0, 0, 0)) != 0 {
		t.Error("小銭は売買しない")
	}
	// 壺の売値には自分の中身の売値も入る
	pot.Contents = []Item{buildItemFromTemplate(1, 0, 0)} // ウインナー 100 → 売値50
	if got := itemSellPrice(pot); got != 400+50 {
		t.Errorf("中身入りの壺の売値 %d, want 450", got)
	}
	// 強化値がとても低くても基準の20%より安くならない
	if got := itemBuyPrice(shopTestWeapon(-10)); got != 20 {
		t.Errorf("強化値-10の買値 %d, want 20（基準の20%%の半額）", got)
	}
}

func TestShopChanceByFloor(t *testing.T) {
	for floor := 1; floor <= finalFloor; floor++ {
		want := shopChancePercent
		if floor < shopFirstFloor || isBossFloor(floor) {
			want = 0
		}
		if got := shopChance(floor); got != want {
			t.Errorf("floor %d chance %d, want %d", floor, got, want)
		}
	}
}

func TestShopGeneration(t *testing.T) {
	shops, eligible := 0, 0
	for seed := int64(1); seed <= 60; seed++ {
		for floor := 1; floor <= finalFloor; floor++ {
			player := newInitialPlayer()
			mapGrid, enemies, items, _, rooms, traps := GenerateRandomMapFor(70, 70, floor-1, &player, newFloorRNG(seed, floor), DifficultyAdvanced)
			if shopChance(floor) > 0 {
				eligible++
			}
			var shop Room
			count := 0
			for _, room := range rooms {
				if room.Role == RoomRoleShop {
					shop = room
					count++
				}
			}
			if count == 0 {
				for _, enemy := range enemies {
					if isShopMonsterID(enemy.ID) {
						t.Fatalf("seed %d floor %d: shop monster without a shop", seed, floor)
					}
				}
				continue
			}
			shops++
			if count > 1 || shopChance(floor) == 0 {
				t.Fatalf("seed %d floor %d: %d shops (chance %d)", seed, floor, count, shopChance(floor))
			}
			if roomContainsWithWalls(shop, player.X, player.Y) {
				t.Fatalf("seed %d floor %d: shop contains the start", seed, floor)
			}
			products := 0
			for _, item := range items {
				x, y := item.GetPosition()
				inShop := roomContainsWithWalls(shop, x, y)
				price := itemShopPrice(item)
				if inShop != (price > 0) {
					t.Fatalf("seed %d floor %d: item %d at (%d,%d) inShop=%v price=%d", seed, floor, item.GetID(), x, y, inShop, price)
				}
				if price > 0 {
					products++
					if !roomContains(shop, x, y) || price != itemBuyPrice(item) || itemTemplates[item.GetID()].ItemType == "Money" {
						t.Fatalf("seed %d floor %d: bad product %d price %d", seed, floor, item.GetID(), price)
					}
				}
			}
			if products < shopMinItems || products > shopMaxItems {
				t.Fatalf("seed %d floor %d: %d products", seed, floor, products)
			}
			keepers := 0
			for _, enemy := range enemies {
				if roomContainsWithWalls(shop, enemy.X, enemy.Y) && enemy.ID != shopkeeperMonsterID {
					t.Fatalf("seed %d floor %d: enemy %d inside the shop", seed, floor, enemy.ID)
				}
				if enemy.ID == shopkeeperMonsterID {
					keepers++
					// 店主は出入口の内側の隣に立ち、どの出入口の内側もふさがない
					adjacentExit, blocking := false, false
					for _, exit := range roomExits(mapGrid, shop) {
						door := shopDoorway(shop, exit)
						blocking = blocking || door == (Coordinate{X: enemy.X, Y: enemy.Y})
						adjacentExit = adjacentExit || abs(door.X-enemy.X)+abs(door.Y-enemy.Y) == 1
					}
					if blocking {
						t.Fatalf("seed %d floor %d: keeper blocks a doorway", seed, floor)
					}
					if !roomContains(shop, enemy.X, enemy.Y) || !adjacentExit || enemy.Alerted || enemy.StatusAilments.Sleep != 0 {
						t.Fatalf("seed %d floor %d: keeper at (%d,%d)", seed, floor, enemy.X, enemy.Y)
					}
				}
			}
			if keepers != 1 {
				t.Fatalf("seed %d floor %d: %d keepers", seed, floor, keepers)
			}
			for _, trap := range traps {
				if roomContainsWithWalls(shop, trap.X, trap.Y) {
					t.Fatalf("seed %d floor %d: trap in the shop", seed, floor)
				}
			}
			// 店の床には地形を置かない（壁の割れ目は地形の担当の規則のまま。壊して出れば泥棒になる）
			for y := shop.Y + 1; y < shop.Y+shop.Height-1; y++ {
				for x := shop.X + 1; x < shop.X+shop.Width-1; x++ {
					if typ := mapGrid[y][x].Type; typ != "floor" {
						t.Fatalf("seed %d floor %d: %s in the shop", seed, floor, typ)
					}
				}
			}
		}
	}
	// 10% の目安（対象 60×26 階＝1560）。実際の割合は候補の部屋が無い階のぶん少し下がる
	if rate := float64(shops) / float64(eligible); rate < 0.05 || rate > 0.15 {
		t.Fatalf("shop rate %.3f (%d/%d)", rate, shops, eligible)
	}
}

func TestShopBuyWithEnoughCash(t *testing.T) {
	g := newShopTestGame(t)
	product := pickUpShopProduct(t, g)
	if g.shopUnpaidTotal() != 200 || itemShopPrice(product) != 200 {
		t.Fatalf("unpaid %d", g.shopUnpaidTotal())
	}
	g.state.Player.Cash = 500
	if !g.payShop() {
		t.Fatal("payment failed")
	}
	if g.state.Player.Cash != 300 || itemShopPrice(product) != 0 || g.shopUnpaidTotal() != 0 || g.stats.ShopBought != 1 {
		t.Fatalf("cash %d price %d unpaid %d bought %d", g.state.Player.Cash, itemShopPrice(product), g.shopUnpaidTotal(), g.stats.ShopBought)
	}
	// 払った品を持って出ても泥棒にならない
	g.state.Player.X, g.state.Player.Y = 11, 4
	g.updateShop()
	if g.stats.ShopThefts != 0 || !g.shopOpen() {
		t.Fatal("paid item caused theft")
	}
}

func TestShopBuyWithoutEnoughCash(t *testing.T) {
	g := newShopTestGame(t)
	product := pickUpShopProduct(t, g)
	g.state.Player.Cash = 150
	if g.payShop() {
		t.Fatal("payment must fail")
	}
	if g.state.Player.Cash != 150 || itemShopPrice(product) != 200 || g.shopUnpaidTotal() != 200 {
		t.Fatal("failed payment changed state")
	}
	// 店の床へ戻せば未払いは消える
	g.state.Player.Inventory = nil
	product.SetPosition(5, 5)
	g.state.Items = append(g.state.Items, product)
	g.refreshShopLedger()
	if g.shopUnpaidTotal() != 0 || itemShopPrice(product) != 200 {
		t.Fatalf("returned item still unpaid: %d", g.shopUnpaidTotal())
	}
}

func TestShopUsedUnpaidItemIsDebt(t *testing.T) {
	g := newShopTestGame(t)
	pickUpShopProduct(t, g)
	g.state.Player.Inventory = nil // 使って無くなった
	g.refreshShopLedger()
	if g.shopDebt != 200 || g.shopUnpaidTotal() != 200 {
		t.Fatalf("debt %d", g.shopDebt)
	}
	g.state.Player.Cash = 200
	if !g.payShop() || g.state.Player.Cash != 0 || g.shopDebt != 0 {
		t.Fatal("debt payment")
	}
}

func TestShopTalkAndPayByInput(t *testing.T) {
	g := newShopTestGame(t)
	in := g.input.(*VirtualInput)
	b := newBot(g, in, personaCautious, 1)
	pickUpShopProduct(t, g)
	g.state.Player.Cash = 1000
	g.state.Player.X, g.state.Player.Y = 7, 3 // 店主の左隣
	// 右を向いて Z で話しかける（攻撃しない・ターンは進まない）
	turn := g.turnCount
	if err := b.tap(KeyA, KeyRight); err != nil {
		t.Fatal(err)
	}
	if err := b.tap(KeyZ); err != nil {
		t.Fatal(err)
	}
	if g.shopPrompt == nil || g.shopPrompt.Kind != shopPromptPay || g.shopPrompt.Price != 200 {
		t.Fatalf("prompt %+v", g.shopPrompt)
	}
	keeper := g.state.Enemies[g.shopkeeperIndex()]
	if keeper.Health != keeper.MaxHealth || keeper.Alerted || g.turnCount != turn {
		t.Fatal("talking must not attack or use a turn")
	}
	if err := b.tap(KeyZ); err != nil {
		t.Fatal(err)
	}
	if g.shopPrompt != nil || g.state.Player.Cash != 800 || g.shopUnpaidTotal() != 0 {
		t.Fatalf("cash %d unpaid %d", g.state.Player.Cash, g.shopUnpaidTotal())
	}
}

func TestShopExitPrompt(t *testing.T) {
	g := newShopTestGame(t)
	pickUpShopProduct(t, g)
	g.state.Player.X, g.state.Player.Y = 9, 4 // 出口
	g.updateShop()
	if g.shopPrompt == nil || g.shopPrompt.Kind != shopPromptPay || g.stats.ShopThefts != 0 {
		t.Fatal("exit must offer payment")
	}
	g.shopPrompt = nil // やめる
	g.updateShop()
	if g.shopPrompt != nil {
		t.Fatal("prompt must not repeat until leaving the exit")
	}
}

func TestShopSell(t *testing.T) {
	g := newShopTestGame(t)
	own := shopTestWeapon(3) // 買値 320・売値 160
	g.state.Player.Inventory = []Item{own}
	g.state.Player.X, g.state.Player.Y = 5, 5
	own.SetPosition(5, 5)
	g.state.Player.Inventory = nil
	g.state.Items = append(g.state.Items, own)
	g.onItemPlacedInShop(own)
	if g.shopPrompt == nil || g.shopPrompt.Kind != shopPromptSell || g.shopPrompt.Price != 160 {
		t.Fatalf("sell prompt %+v", g.shopPrompt)
	}
	g.shopPrompt = nil
	if !g.sellToShop(own, 160) {
		t.Fatal("sale failed")
	}
	if g.state.Player.Cash != 160 || itemShopPrice(own) != 320 || g.stats.ShopSold != 1 {
		t.Fatalf("cash %d price %d", g.state.Player.Cash, itemShopPrice(own))
	}
	// 店の外では売れない
	outside := shopTestWeapon(0)
	outside.SetPosition(15, 4)
	g.state.Items = append(g.state.Items, outside)
	g.onItemPlacedInShop(outside)
	if g.shopPrompt != nil || g.sellToShop(outside, 100) {
		t.Fatal("must not sell outside the shop")
	}
}

func TestShopSellByDroppingWithInput(t *testing.T) {
	g := newShopTestGame(t)
	in := g.input.(*VirtualInput)
	b := newBot(g, in, personaCautious, 1)
	own := shopTestWeapon(0) // 売値 100
	g.state.Player.Inventory = []Item{own}
	g.state.Player.X, g.state.Player.Y = 5, 5
	if err := b.openItemActions(0); err != nil {
		t.Fatal(err)
	}
	if ok, err := b.chooseItemAction("置く"); !ok || err != nil {
		t.Fatalf("drop: %v %v", ok, err)
	}
	if g.shopPrompt == nil || g.shopPrompt.Price != 100 {
		t.Fatalf("prompt %+v", g.shopPrompt)
	}
	if err := b.tap(KeyZ); err != nil {
		t.Fatal(err)
	}
	if g.state.Player.Cash != 100 || itemShopPrice(own) != 200 {
		t.Fatalf("cash %d price %d", g.state.Player.Cash, itemShopPrice(own))
	}
}

func assertThief(t *testing.T, g *Game, product Item) {
	t.Helper()
	keeper := g.shopkeeperIndex()
	if g.stats.ShopThefts != 1 || itemShopPrice(product) != 0 || keeper < 0 || !g.state.Enemies[keeper].Alerted || !g.shopAlarm {
		t.Fatalf("thefts %d price %d alarm %v", g.stats.ShopThefts, itemShopPrice(product), g.shopAlarm)
	}
	if !statusActive(g.state.Enemies[keeper].StatusAilments, StatusHaste) {
		t.Fatal("angry keeper must be hasted")
	}
	dogs := 0
	for _, e := range g.state.Enemies {
		if e.ID == guardDogMonsterID {
			dogs++
		}
	}
	if dogs == 0 {
		t.Fatal("guard dogs must appear")
	}
	if g.spawnInterval > shopAlarmSpawnInterval(MaxSpawnInterval) {
		t.Fatalf("spawn interval %d after theft", g.spawnInterval)
	}
}

func TestShopTheftByWalkingOut(t *testing.T) {
	g := newShopTestGame(t)
	product := pickUpShopProduct(t, g)
	g.state.Player.X, g.state.Player.Y = 10, 4 // 出口の外の通路
	g.updateShop()
	assertThief(t, g, product)
}

func TestShopTheftByWarp(t *testing.T) {
	g := newShopTestGame(t)
	product := pickUpShopProduct(t, g)
	g.state.Player.X, g.state.Player.Y = 15, 6 // ワープで隣の部屋へ
	g.updateShop()
	assertThief(t, g, product)
}

func TestShopTheftByFallingAndStairs(t *testing.T) {
	for _, viaStairs := range []bool{false, true} {
		g := newShopTestGame(t)
		product := pickUpShopProduct(t, g)
		if viaStairs {
			g.state.Player.X, g.state.Player.Y = 17, 4 // 階段まで歩いた（店を出た時点で泥棒）
			g.updateShop()
			assertThief(t, g, product)
			g.beginFloorDescent()
		} else {
			g.requestFloorDescent() // 店の中の落とし穴
		}
		if err := g.StepFrame(); err != nil {
			t.Fatal(err)
		}
		if err := g.RunUntilInputReady(600); err != nil {
			t.Fatal(err)
		}
		if g.Floor != 6 || g.stats.ShopThefts != 1 || itemShopPrice(product) != 0 || g.shopAlarm || g.shopUnpaidTotal() != 0 {
			t.Fatalf("stairs=%v floor %d thefts %d price %d alarm %v", viaStairs, g.Floor, g.stats.ShopThefts, itemShopPrice(product), g.shopAlarm)
		}
	}
}

func TestShopNoTheftWithoutUnpaidItems(t *testing.T) {
	g := newShopTestGame(t)
	g.state.Player.X, g.state.Player.Y = 11, 4
	g.updateShop()
	g.requestFloorDescent()
	if err := g.StepFrame(); err != nil {
		t.Fatal(err)
	}
	if err := g.RunUntilInputReady(600); err != nil {
		t.Fatal(err)
	}
	if g.stats.ShopThefts != 0 {
		t.Fatal("leaving without goods is not theft")
	}
}

func TestShopAttackingKeeperIsTheft(t *testing.T) {
	g := newShopTestGame(t)
	keeper := g.shopkeeperIndex()
	g.applyDamageToEnemy(keeper, 5)
	if g.stats.ShopThefts != 1 || !g.state.Enemies[g.shopkeeperIndex()].Alerted || !g.shopAlarm {
		t.Fatal("attack must count as theft")
	}
	// 商品は店主が怒っている間に拾うと自分の物
	product := pickUpShopProduct(t, g)
	if itemShopPrice(product) != 0 {
		t.Fatal("goods picked up after theft are free")
	}
}

func TestShopThrowingAtKeeperIsTheft(t *testing.T) {
	g := newShopTestGame(t)
	keeper := g.shopkeeperIndex()
	food := buildItemFromTemplate(1, 0, 0)
	g.hitEnemyWithProjectile(&Projectile{Item: food, Kind: ProjectileThrow, FromPlayer: true, X: 8, Y: 3}, keeper)
	if g.stats.ShopThefts != 1 || !g.state.Enemies[g.shopkeeperIndex()].Alerted {
		t.Fatal("throwing at the keeper must count as theft")
	}
}

func TestShopkeeperIsUntouchableByEffects(t *testing.T) {
	g := newShopTestGame(t)
	keeper := g.shopkeeperIndex()
	for _, kind := range []StatusKind{StatusSleep, StatusConfusion, StatusParalysis, StatusSeal, StatusSlow, StatusBackward, StatusDecoy} {
		if g.applyEnemyStatus(keeper, kind, 5) {
			t.Errorf("keeper accepted %s", kind)
		}
	}
	x, y := g.state.Enemies[keeper].X, g.state.Enemies[keeper].Y
	g.knockbackEnemy(keeper, -1, 0, 3, 5)
	if err := g.RunUntilInputReady(100); err != nil {
		t.Fatal(err)
	}
	if k := g.state.Enemies[g.shopkeeperIndex()]; k.X != x || k.Y != y {
		t.Fatal("keeper must not be knocked back")
	}
	// 怒っていない店主は動かず攻撃しない
	g.state.Player.X, g.state.Player.Y = 7, 3
	hp := g.state.Player.Health
	for i := 0; i < 5; i++ {
		g.MoveEnemies()
		if err := g.RunUntilInputReady(100); err != nil {
			t.Fatal(err)
		}
	}
	if k := g.state.Enemies[g.shopkeeperIndex()]; k.X != x || k.Y != y || g.state.Player.Health != hp {
		t.Fatal("peaceful keeper moved or attacked")
	}
}

func TestShopKeeperDefeatedFreesGoods(t *testing.T) {
	g := newShopTestGame(t)
	product := g.state.Items[0]
	g.applyDamageToEnemy(g.shopkeeperIndex(), 1) // 怒らせる
	g.applyDamageToEnemy(g.shopkeeperIndex(), 10000)
	if err := g.RunUntilInputReady(200); err != nil {
		t.Fatal(err)
	}
	if g.shopkeeperIndex() >= 0 || itemShopPrice(product) != 0 {
		t.Fatal("goods must become free after the keeper is defeated")
	}
	if !g.shopAlarm {
		t.Fatal("the floor stays alarmed")
	}
}

func TestShopSaveRestore(t *testing.T) {
	g := newShopTestGame(t)
	product := pickUpShopProduct(t, g)
	g.shopDebt = 50
	restored, vs := saveRoundTrip(g)
	if len(vs) > 0 {
		t.Fatalf("round trip: %+v", vs)
	}
	if room, ok := restored.shopRoom(); !ok || room.ID != 0 {
		t.Fatal("shop room lost")
	}
	if len(restored.state.Player.Inventory) != 1 || itemShopPrice(restored.state.Player.Inventory[0]) != itemShopPrice(product) {
		t.Fatal("unpaid item lost")
	}
	if restored.shopUnpaidTotal() != 250 {
		t.Fatalf("unpaid %d after restore", restored.shopUnpaidTotal())
	}
	// 泥棒の状態も残る
	restored.applyDamageToEnemy(restored.shopkeeperIndex(), 1)
	again, vs := saveRoundTrip(restored)
	if len(vs) > 0 {
		t.Fatalf("round trip: %+v", vs)
	}
	if !again.shopAlarm || again.shopOpen() || !again.state.Enemies[again.shopkeeperIndex()].Alerted {
		t.Fatal("theft state lost")
	}
	// 古いセーブ（店の項目なし）も読める
	save := g.buildSaveData()
	save.Shop = nil
	for i := range save.Rooms {
		save.Rooms[i].Role = ""
	}
	if err := (&Game{customNames: map[int]string{}}).applySaveData(save); err != nil {
		t.Fatal(err)
	}
}

// TestBotDoesNotStealFromShop は BOT（random 以外）が店の中から始めても泥棒せず、店主を攻撃しないことを確かめる。
func TestBotDoesNotStealFromShop(t *testing.T) {
	for _, p := range []persona{personaExpert, personaAggressive, personaCautious, personaItemUser, personaDiver} {
		for seed := int64(1); seed <= 3; seed++ {
			g := newShopTestGame(t)
			// 商品を増やし、売れる道具と回復薬を持たせる
			for i, xy := range []Coordinate{{2, 2}, {5, 2}, {2, 6}, {6, 6}} {
				item := buildItemFromTemplate([]int{2, 7, 24, 175}[i], xy.X, xy.Y)
				setItemShopPrice(item, itemBuyPrice(item))
				g.state.Items = append(g.state.Items, item)
			}
			b := newBot(g, g.input.(*VirtualInput), p, seed)
			for step := 0; step < 300 && g.Floor == 5 && !g.RunEnded(); step++ {
				if err := b.step(); err != nil {
					t.Fatalf("%s seed %d: %v at (%d,%d)", p, seed, err, g.state.Player.X, g.state.Player.Y)
				}
			}
			if g.Floor == 5 {
				t.Fatalf("%s seed %d: bot did not leave the shop floor", p, seed)
			}
			if g.stats.ShopThefts != 0 {
				t.Fatalf("%s seed %d: bot stole from the shop", p, seed)
			}
			if g.Floor == 5 && !g.shopOpen() {
				t.Fatalf("%s seed %d: bot angered the keeper", p, seed)
			}
		}
	}
}

// TestShopEnemyTrapDoesNotAngerKeeper は敵が踏んだ罠の爆発で店主が怒らないことを確かめる
// （TestSimBatch で、敵の踏んだ地雷の爆風が店主に当たって泥棒扱いになっていた）。
// 海老さんが起こした罠（投げた道具の着地など）なら泥棒と同じ扱い。
func TestShopEnemyTrapDoesNotAngerKeeper(t *testing.T) {
	g := newShopTestGame(t)
	keeper := g.shopkeeperIndex()
	g.trapEnemyDamage(keeper, 30, false)
	if k := g.state.Enemies[g.shopkeeperIndex()]; k.Health != k.MaxHealth || k.Alerted || g.stats.ShopThefts != 0 {
		t.Fatal("an enemy-triggered trap must not hurt or anger the keeper")
	}
	g.trapEnemyDamage(keeper, 30, true)
	if g.stats.ShopThefts != 1 || !g.state.Enemies[g.shopkeeperIndex()].Alerted {
		t.Fatal("a player-triggered trap counts as theft")
	}
}

// 店の中でモンスターハウスのカードを使うと、出た道具は商品になる（開発者の決定、2026-10-21）。
// 店主が怒っていれば値段は付かない。
func TestMonsterHouseCardInShopMakesGoods(t *testing.T) {
	for _, angry := range []bool{false, true} {
		g := newShopTestGame(t)
		if angry {
			g.provokeShopkeeper(g.shopkeeperIndex())
		}
		product := g.state.Items[0]
		card := buildItemFromTemplate(35, 0, 0)
		g.state.Player.Inventory = []Item{card}
		g.selectedItemIndex = 0
		card.Use(g)
		if err := g.RunUntilInputReady(200); err != nil {
			t.Fatal(err)
		}
		spawned := 0
		for _, item := range g.state.Items {
			if item == product {
				continue
			}
			spawned++
			x, y := item.GetPosition()
			if !g.inShopInterior(x, y) {
				t.Fatalf("item at (%d,%d) outside the shop", x, y)
			}
			price, want := itemShopPrice(item), itemBuyPrice(item)
			if angry {
				want = 0
			}
			if price != want {
				t.Fatalf("angry=%v: %s price %d, want %d", angry, item.GetName(), price, want)
			}
		}
		if spawned < 2 {
			t.Fatalf("angry=%v: spawned items = %d, want 2..3", angry, spawned)
		}
	}
}

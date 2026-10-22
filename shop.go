//go:build !test

package main

import "fmt"

// 店・売買・店主・泥棒（wave10-B）。値段の表と計算は item_prices.go、部屋の選び方は special_rooms.go。
//
// 状態の持ち方:
//   - 店の部屋は Room.Role == RoomRoleShop（1階に1つまで）。
//   - 商品は床のアイテムの BaseItem.ShopPrice > 0。持ち物に入った ShopPrice > 0 の品は代金未払い。
//   - 店主（126）が怒っているか（泥棒をしたか）は店主の Enemy.Alerted。
//   - g.shopAlarm はこの階で泥棒をした（店主を倒した後も敵が湧きやすいまま）。
//   - g.shopDebt は未払いのまま使った・失った品の代金。g.shopCarried は未払いの品をポインタで追う一覧（保存しない）。

const (
	shopkeeperMonsterID    = 126
	guardDogMonsterID      = 127
	eliteGuardDogMonsterID = 128
)

// isShopMonsterID は店主・番犬の ID か。ランダム出現の対象から外す。
func isShopMonsterID(id int) bool {
	return id == shopkeeperMonsterID || id == guardDogMonsterID || id == eliteGuardDogMonsterID
}

// addShopMonsters は店主と番犬を定義の表へ加える（buildMonsterDefinitions から呼ぶ）。
// 店主は階に関係なく同じ強さで、最も強い第3段階の敵（深淵海竜 HP184・攻撃44・防御16）より強くしてある。
// 店主はボスと同じ例外（IsBoss: ランダム出現・上位種化・即死・吹き飛ばし・場所替え・罠の対象外）にする。
// 番犬は泥棒をしたときだけ現れる1段階の敵で、上位種化しない（MonsterLevelUpTable に入れない）。
// 階の深さは16階からの強い番犬（128）で表す。
func addShopMonsters(definitions map[int]MonsterDefinition) {
	definitions[shopkeeperMonsterID] = MonsterDefinition{ID: shopkeeperMonsterID, Type: "Shopkeeper", Name: "店主", Char: 'K',
		Description: "店の主人。ふだんは何もしないが、泥棒や攻撃をすると怒って襲ってくる。とても強い。",
		Health:      400, MaxHealth: 400, AttackPower: 50, DefensePower: 20, ExperiencePoints: 1000, IsBoss: true}
	definitions[guardDogMonsterID] = MonsterDefinition{ID: guardDogMonsterID, Type: "GuardDog", Name: "番犬", Char: 'd',
		Description: "泥棒をすると海老さんの近くに現れる店の番犬。",
		Health:      60, MaxHealth: 60, AttackPower: 18, DefensePower: 8, ExperiencePoints: 60}
	definitions[eliteGuardDogMonsterID] = MonsterDefinition{ID: eliteGuardDogMonsterID, Type: "EliteGuardDog", Name: "猛番犬", Char: 'd',
		Description: "泥棒をすると海老さんの近くに現れる強い番犬。16階から番犬の代わりに現れる。",
		Health:      150, MaxHealth: 150, AttackPower: 38, DefensePower: 14, ExperiencePoints: 200}
}

func init() {
	updateMonsterDefinitions(func(d *MonsterDefinition) {
		d.Behavior = shopkeeperBehavior
		d.OnDefeated = shopkeeperDefeated
	}, shopkeeperMonsterID)
	priceItemIfInShop = priceSpawnedShopItem
	upperEnemyImagePaths["Shopkeeper"] = "img/shopkeeper.png"
	upperEnemyImagePaths["GuardDog"] = "img/guard_dog.png"
	upperEnemyImagePaths["EliteGuardDog"] = "img/elite_guard_dog.png"
	expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "店", Lines: []string{
		"3階から、ときどき店のある階がある。",
		"商品を拾うと代金が未払いになる。",
		"店主に話しかける（隣でZ）か、出口で払う。",
		"自分の道具を店の床に置くと売れる。",
		"未払いのまま店を出ると泥棒になる。",
		"　店主が怒り、番犬が現れ、敵が湧きやすい。",
		"　店主を倒すと商品は自分の物になる。",
		"店主を攻撃しても泥棒と同じ扱い。",
		"道具寄せで商品を店の外へ出しても泥棒。",
		"値段は強化値・杖の回数・壺の容量で変わる。",
		"呪われた品は半額。売値は買値の半分。",
	}})
}

// --- 品物の値段 ---

// priceItemIfInShop は床に新しく出した道具が商売中の店の中なら商品にする（モンスターハウスのカード）。
// アイテムの表から呼ぶので、初期化の循環を避けて init で代入する。
var priceItemIfInShop = func(*Game, Item) {}

func priceSpawnedShopItem(g *Game, item Item) {
	if item == nil {
		return
	}
	x, y := item.GetPosition()
	if g.shopOpen() && g.inShopInterior(x, y) {
		setItemShopPrice(item, itemBuyPrice(item)) // お金など値段の無い品は0のまま
	}
}

type shopPricedItem interface{ shopBaseItem() *BaseItem }

func (b *BaseItem) shopBaseItem() *BaseItem { return b }

// itemShopPrice は品物の店の値段（0 は自分の物）。
func itemShopPrice(item Item) int {
	if priced, ok := item.(shopPricedItem); ok && item != nil {
		return priced.shopBaseItem().ShopPrice
	}
	return 0
}

// setItemShopPrice は品物の店の値段を設定する（0 で自分の物にする）。
func setItemShopPrice(item Item, price int) {
	if priced, ok := item.(shopPricedItem); ok && item != nil {
		priced.shopBaseItem().ShopPrice = max(price, 0)
	}
}

// itemPriceFactorsOf は値段の計算に使う品物の値を集める。
func itemPriceFactorsOf(item Item) itemPriceFactors {
	f := itemPriceFactors{Category: itemTemplates[item.GetID()].ItemType}
	switch v := item.(type) {
	case *Weapon:
		f.Sharpness, f.Cursed = v.Sharpness, v.Cursed
	case *Armor:
		f.Sharpness, f.Cursed = v.Sharpness, v.Cursed
	case *Accessory:
		f.Cursed = v.Cursed
	case *Arrow:
		f.Count, f.Cursed = v.ShotCount, v.Cursed
	case *Cane:
		f.Uses = v.Uses
	case *Pot:
		f.Capacity = v.Capacity
	}
	return f
}

// itemBuyPrice は品物の買値（店で買うときの値段）。小銭は0。
func itemBuyPrice(item Item) int {
	if item == nil {
		return 0
	}
	return itemBuyPriceFrom(itemBasePrices[item.GetID()], itemPriceFactorsOf(item))
}

// itemSellPrice は品物の売値（買値の半分）。壺は中身（自分の物だけ）の売値も足す。換金の壺もこれを使う。
func itemSellPrice(item Item) int {
	if item == nil {
		return 0
	}
	price := itemSellPriceFromBuy(itemBuyPrice(item))
	if pot, ok := item.(*Pot); ok {
		for _, content := range pot.Contents {
			if itemShopPrice(content) == 0 {
				price += itemSellPrice(content)
			}
		}
	}
	return price
}

// --- 店の生成（GenerateRandomMapFor から呼ぶ） ---

// placeShop は確率で1部屋を店にし、商品と店主を置く。店にした部屋の役割を書き換え、
// 部屋の中にあった道具・罠・敵は取り除く。地形を置かないマスを protect に加える。
func placeShop(mapGrid [][]Tile, rooms []Room, enemies []Enemy, items []Item, traps []MapTrap, start, stairs Coordinate,
	floor int, difficulty Difficulty, rng *gameRNG, protect map[Coordinate]bool) ([]Enemy, []Item, []MapTrap) {
	if rng.Intn(100) >= shopChance(floor) {
		return enemies, items, traps
	}
	index := selectSpecialRoom(rooms, start, stairs, rng.Intn)
	if index < 0 {
		return enemies, items, traps
	}
	room := rooms[index]
	exits := roomExits(mapGrid, room)
	if len(exits) == 0 {
		return enemies, items, traps
	}
	// 出入口の内側のマスには何も置かない（通り道をふさがない）。店主は最初に置ける出入口の内側の隣
	reserved := map[Coordinate]bool{}
	for _, exit := range exits {
		reserved[shopDoorway(room, exit)] = true
	}
	var keeper Coordinate
	placed := false
	for _, exit := range exits {
		if keeper, placed = shopKeeperSpot(room, exit, reserved); placed {
			break
		}
	}
	if !placed {
		return enemies, items, traps
	}
	reserved[keeper] = true
	rooms[index].Role = RoomRoleShop

	// 部屋の中の道具・罠・敵を取り除く（商品と混ざらないように）
	items = filterItems(items, func(item Item) bool { x, y := item.GetPosition(); return !roomContainsWithWalls(room, x, y) })
	keptTraps := traps[:0]
	for _, trap := range traps {
		if !roomContainsWithWalls(room, trap.X, trap.Y) {
			keptTraps = append(keptTraps, trap)
		}
	}
	traps = keptTraps
	keptEnemies := enemies[:0]
	for _, enemy := range enemies {
		if !roomContainsWithWalls(room, enemy.X, enemy.Y) {
			keptEnemies = append(keptEnemies, enemy)
		}
	}
	enemies = keptEnemies

	for y := room.Y; y <= room.Y+room.Height-1; y++ {
		for x := room.X; x <= room.X+room.Width-1; x++ {
			protect[Coordinate{X: x, Y: y}] = true // 店の中には地形を置かない
		}
	}
	cells := pickFreeCellsInRoom(room, func(x, y int) bool { return !reserved[Coordinate{X: x, Y: y}] }, room.Width*room.Height, rng.Intn)
	count := rollShopItemCount(len(cells), rng.Intn)
	for _, cell := range cells[:count] {
		item := createShopItem(cell.X, cell.Y, floor, difficulty, rng.Intn)
		if item == nil {
			continue
		}
		items = append(items, item)
	}
	enemies = append(enemies, CreateEnemyByID(shopkeeperMonsterID, keeper.X, keeper.Y))
	return enemies, items, traps
}

// createShopItem は商品を1つ作る（床と同じ階層別の抽選。小銭は引き直す）。値段を付けて返す。
func createShopItem(x, y, floor int, difficulty Difficulty, intn func(int) int) Item {
	for i := 0; i < shopItemRerollMax; i++ {
		id := pickFloorItemID(floor, difficulty, intn)
		if itemBasePrices[id] <= 0 {
			continue
		}
		item := rollItemFromTemplate(id, x, y, intn)
		setItemShopPrice(item, itemBuyPrice(item))
		return item
	}
	return nil
}

func filterItems(items []Item, keep func(Item) bool) []Item {
	out := items[:0]
	for _, item := range items {
		if keep(item) {
			out = append(out, item)
		}
	}
	return out
}

// --- 店の部屋と店主 ---

// shopRoom は今の階の店の部屋を返す。
func (g *Game) shopRoom() (Room, bool) {
	for _, room := range g.rooms {
		if room.Role == RoomRoleShop {
			return room, true
		}
	}
	return Room{}, false
}

// inShopInterior は (x, y) が店の部屋の床（壁の線より内側）か。
func (g *Game) inShopInterior(x, y int) bool {
	room, ok := g.shopRoom()
	return ok && roomContains(room, x, y)
}

// shopkeeperIndex は店主の添字（いなければ -1）。
func (g *Game) shopkeeperIndex() int {
	for i := range g.state.Enemies {
		if g.state.Enemies[i].ID == shopkeeperMonsterID {
			return i
		}
	}
	return -1
}

// isPeacefulShopkeeper は敵 i が怒っていない店主か。
func (g *Game) isPeacefulShopkeeper(i int) bool {
	return i >= 0 && i < len(g.state.Enemies) && peacefulShopkeeper(g.state.Enemies[i])
}

func peacefulShopkeeper(e Enemy) bool {
	return e.ID == shopkeeperMonsterID && !e.Alerted
}

// shopOpen は商売ができる（怒っていない店主がいる）か。
func (g *Game) shopOpen() bool {
	return g.isPeacefulShopkeeper(g.shopkeeperIndex())
}

// shopkeeperBehavior は店主の行動。怒っていなければ何もしない（攻撃も移動もしない）。
func shopkeeperBehavior(g *Game, i int) bool {
	return !g.state.Enemies[i].Alerted
}

// shopkeeperDefeated は店主を倒したとき、店の品物をすべて海老さんの物にする。
func shopkeeperDefeated(g *Game, _ Enemy) {
	for _, item := range g.state.Items {
		setItemShopPrice(item, 0)
	}
	g.clearInventoryShopPrices()
	g.shopCarried, g.shopDebt = nil, 0
	g.EnqueueMessage("店の品物は海老さんの物になった", 0.8)
}

// --- 未払いの品 ---

// forEachInventoryItem は持ち物と壺の中身（入れ子も）を順に渡す。
func forEachItemIn(items []Item, visit func(Item)) {
	for _, item := range items {
		visit(item)
		if pot, ok := item.(*Pot); ok {
			forEachItemIn(pot.Contents, visit)
		}
	}
}

func (g *Game) clearInventoryShopPrices() {
	forEachItemIn(g.state.Player.Inventory, func(item Item) { setItemShopPrice(item, 0) })
}

// refreshShopLedger は未払いの品の一覧を今の状態に合わせる。店に戻した品は一覧から外し、
// 使った・失った品は代金を g.shopDebt へ移す。
func (g *Game) refreshShopLedger() {
	carried := map[Item]bool{}
	forEachItemIn(g.state.Player.Inventory, func(item Item) {
		if itemShopPrice(item) > 0 {
			carried[item] = true
		}
	})
	onFloor := map[Item]Coordinate{} // 床の道具と、床の壺の中身（壺の位置）
	for _, item := range g.state.Items {
		x, y := item.GetPosition()
		forEachItemIn([]Item{item}, func(inner Item) { onFloor[inner] = Coordinate{X: x, Y: y} })
	}
	var tracked []Item
	seen := map[Item]bool{}
	track := func(item Item) {
		if seen[item] || itemShopPrice(item) == 0 {
			return
		}
		seen[item] = true
		switch position, floor := onFloor[item]; {
		case carried[item]:
			tracked = append(tracked, item)
		case floor && g.inShopInterior(position.X, position.Y):
			// 店の床へ戻した（商品に戻る）
		case floor:
			tracked = append(tracked, item) // 店の外へ投げた・置いた品は未払いのまま
		default:
			g.shopDebt += itemShopPrice(item) // 使った・壺に入れて消えた・盗まれた
			setItemShopPrice(item, 0)
		}
	}
	for _, item := range g.shopCarried {
		track(item)
	}
	forEachItemIn(g.state.Player.Inventory, track)
	g.shopCarried = tracked
}

// shopUnpaidTotal は未払いの代金の合計。
func (g *Game) shopUnpaidTotal() int {
	total := g.shopDebt
	for _, item := range g.shopCarried {
		total += itemShopPrice(item)
	}
	return total
}

// onShopItemPickedUp は拾った品が商品なら未払いにする（店主が怒っている・いないなら自分の物になる）。
func (g *Game) onShopItemPickedUp(item Item) {
	price := itemShopPrice(item)
	if price <= 0 {
		return
	}
	if !g.shopOpen() {
		setItemShopPrice(item, 0)
		return
	}
	g.shopCarried = append(g.shopCarried, item)
	name, _ := displayItemName(item)
	g.EnqueueMessage(fmt.Sprintf("店主「%sは%d円です」", name, price), 0.6)
}

// onItemPlacedInShop は店の床に置いた品を扱う。商品なら棚に戻し、自分の物なら買い取りを持ちかける。
func (g *Game) onItemPlacedInShop(item Item) {
	x, y := item.GetPosition()
	if !g.shopOpen() || !g.inShopInterior(x, y) {
		return
	}
	name, _ := displayItemName(item)
	if itemShopPrice(item) > 0 {
		g.EnqueueMessage(fmt.Sprintf("%sを店に戻した", name), 0.5)
		return
	}
	price := itemSellPrice(item)
	if price <= 0 {
		g.EnqueueMessage("店主「それは買い取れません」", 0.6)
		return
	}
	g.openShopPrompt(&shopPromptState{Kind: shopPromptSell, Item: item, Price: price,
		Text: fmt.Sprintf("店主「%sなら%d円で買い取ります」", name, price)})
}

// --- 支払い・売却 ---

// talkToShopkeeper は隣の店主に話しかける（Z）。未払いがあれば支払いを持ちかける。ターンは進まない。
func (g *Game) talkToShopkeeper() {
	g.refreshShopLedger()
	if total := g.shopUnpaidTotal(); total > 0 {
		g.openShopPrompt(&shopPromptState{Kind: shopPromptPay, Price: total,
			Text: fmt.Sprintf("店主「お代は%d円です」", total)})
		return
	}
	g.EnqueueMessage("店主「いらっしゃいませ。品物を拾ってから声をかけてください」", 0.8)
}

// payShop は未払いの代金を払う。所持金が足りなければ払わない。
func (g *Game) payShop() bool {
	g.refreshShopLedger()
	total := g.shopUnpaidTotal()
	if total <= 0 {
		return false
	}
	if g.state.Player.Cash < total {
		g.EnqueueMessage(fmt.Sprintf("所持金が足りない（%d円）。店主「品物を戻してください」", g.state.Player.Cash), 0.8)
		return false
	}
	g.state.Player.Cash -= total
	g.stats.ShopBought += len(g.shopCarried)
	g.stats.ShopSpent += total
	for _, item := range g.shopCarried {
		setItemShopPrice(item, 0)
	}
	g.shopCarried, g.shopDebt = nil, 0
	g.EnqueueMessage(fmt.Sprintf("%d円を払った。店主「まいどあり！」", total), 0.8)
	return true
}

// sellToShop は店の床に置いた自分の品を売る。売った品は商品になる。
func (g *Game) sellToShop(item Item, price int) bool {
	x, y := item.GetPosition()
	if !g.shopOpen() || !g.inShopInterior(x, y) || itemShopPrice(item) > 0 || !g.itemOnFloor(item) {
		return false
	}
	g.state.Player.Cash += price
	g.stats.ShopSold++
	g.stats.ShopEarned += price
	setItemShopPrice(item, max(itemBuyPrice(item), 1))
	name, _ := displayItemName(item)
	g.EnqueueMessage(fmt.Sprintf("%sを%d円で売った", name, price), 0.6)
	return true
}

func (g *Game) itemOnFloor(item Item) bool {
	for _, floorItem := range g.state.Items {
		if floorItem == item {
			return true
		}
	}
	return false
}

// --- 泥棒 ---

// becomeShopThief は泥棒になる。未払いの品は自分の物になり、店主が怒る。leaving なら階を離れるところなので、
// 番犬と湧きの増加は省く（冒険の記録には残す）。
func (g *Game) becomeShopThief(leaving bool) {
	keeper := g.shopkeeperIndex()
	if !g.isPeacefulShopkeeper(keeper) {
		return
	}
	g.stats.ShopThefts++
	g.clearInventoryShopPrices()
	for _, item := range g.shopCarried {
		setItemShopPrice(item, 0)
	}
	g.shopCarried, g.shopDebt = nil, 0
	if leaving {
		g.EnqueueMessage("代金を払わずに店を去った（泥棒）", 0.8)
		return
	}
	g.shopAlarm = true
	g.state.Enemies[keeper].Alerted = true
	g.applyEnemyStatus(keeper, StatusHaste, shopKeeperHasteTrn)
	g.EnqueueMessage("店主「泥棒だー！」", 0.8)
	g.spawnGuardDogs()
	g.spawnInterval = min(g.spawnInterval, shopAlarmSpawnInterval(g.spawnInterval))
}

// provokeShopkeeper は海老さんが店主に攻撃・投擲・杖を当てたときに呼ぶ（泥棒と同じ扱い）。
func (g *Game) provokeShopkeeper(i int) {
	if g.isPeacefulShopkeeper(i) {
		g.becomeShopThief(false)
	}
}

// spawnGuardDogs は海老さんの近く（2〜4マス）の空いたマスに番犬を出す。
func (g *Game) spawnGuardDogs() {
	id := guardDogMonsterID
	if g.Floor >= shopEliteDogFloor {
		id = eliteGuardDogMonsterID
	}
	p := g.state.Player
	var cells []Coordinate
	for radius := 2; radius <= 4 && len(cells) < shopGuardDogCount; radius++ {
		for y := p.Y - radius; y <= p.Y+radius; y++ {
			for x := p.X - radius; x <= p.X+radius; x++ {
				if max(abs(x-p.X), abs(y-p.Y)) != radius || !g.inMapBounds(x, y) || g.state.Map[y][x].Blocked ||
					!g.terrainOpenAt(x, y) || isOccupied(g, x, y) || g.state.Map[y][x].Type == "stairs" {
					continue
				}
				cells = append(cells, Coordinate{X: x, Y: y})
			}
		}
	}
	for i := len(cells) - 1; i > 0; i-- {
		j := g.randIntn(i + 1)
		cells[i], cells[j] = cells[j], cells[i]
	}
	spawned := 0
	for _, cell := range cells {
		if spawned >= shopGuardDogCount {
			break
		}
		g.state.Enemies = append(g.state.Enemies, CreateEnemyByID(id, cell.X, cell.Y))
		spawned++
	}
	if spawned > 0 {
		g.miniMapDirty = true
		g.EnqueueMessage(fmt.Sprintf("%sが現れた！", MonsterDefinitions[id].Name), 0.6)
	}
}

// updateShop は毎フレーム（行動の処理が終わっているとき）呼び、未払いのまま店を出たら泥棒にし、
// 出口に立ったら支払いを持ちかける。
func (g *Game) updateShop() {
	if g.playerDead || g.cleared || g.fadingOut || g.fadingIn || len(g.ActionQueue.Queue) > 0 || g.isActioned || g.shopPrompt != nil {
		return
	}
	room, ok := g.shopRoom()
	if !ok || !g.shopOpen() {
		return
	}
	g.refreshShopLedger()
	p := g.state.Player
	unpaid := g.shopUnpaidTotal()
	if unpaid > 0 && !roomContainsWithWalls(room, p.X, p.Y) {
		g.becomeShopThief(false)
		return
	}
	onExit := roomContainsWithWalls(room, p.X, p.Y) && !roomContains(room, p.X, p.Y)
	if !onExit {
		g.shopExitPrompted = false
		return
	}
	if unpaid > 0 && !g.shopExitPrompted && !g.anyWindowOpen() {
		g.shopExitPrompted = true
		g.openShopPrompt(&shopPromptState{Kind: shopPromptPay, Price: unpaid,
			Text: fmt.Sprintf("店主「お代は%d円です。払っていきますか？」", unpaid)})
	}
}

// anyWindowOpen は行動入力を止める窓が開いているか（出口での支払いの窓を重ねないため）。
func (g *Game) anyWindowOpen() bool {
	return g.showInventory || g.ShowGroundItem || g.showMenu || g.showStairsPrompt || g.showMessageLog || g.showSettings ||
		g.showSuspendPrompt || g.showHelp || g.showBlankCardMenu || g.showItemActions || g.useIdentifyItem || g.usePotInsert ||
		g.itemTargetEffect != nil
}

// leaveShopFloor は階を移るとき（階段・落とし穴など）に呼ぶ。未払いのまま去れば泥棒として記録し、店の状態を消す。
func (g *Game) leaveShopFloor() {
	if g.shopOpen() {
		g.refreshShopLedger()
		if g.shopUnpaidTotal() > 0 {
			g.becomeShopThief(true)
		}
	}
	g.clearInventoryShopPrices()
	g.resetShopState()
}

// resetShopState は店の状態を消す（階の移動・新しい冒険）。
func (g *Game) resetShopState() {
	g.shopAlarm, g.shopDebt, g.shopCarried, g.shopPrompt, g.shopExitPrompted = false, 0, nil, nil, false
}

// shopPriceNote は商品・未払いの品の値段の注記（店が開いていなければ空）。
func (g *Game) shopPriceNote(item Item) string {
	price := itemShopPrice(item)
	if price <= 0 || !g.shopOpen() {
		return ""
	}
	return fmt.Sprintf("（%d円）", price)
}

// shopPriceDescription は説明画面に付ける値札（店が開いていれば、商品・未払いは値段、自分の物は売値）。
func (g *Game) shopPriceDescription(item Item) string {
	if item == nil || !g.shopOpen() {
		return ""
	}
	if price := itemShopPrice(item); price > 0 {
		return fmt.Sprintf("\n値段：%d円", price)
	}
	if price := itemSellPrice(item); price > 0 {
		return fmt.Sprintf("\n売値：%d円", price)
	}
	return ""
}

// --- 支払い・売却の窓 ---

type shopPromptKind int

const (
	shopPromptPay shopPromptKind = iota
	shopPromptSell
)

type shopPromptState struct {
	Kind        shopPromptKind
	Item        Item // 売る品
	Price       int
	Text        string
	Selected    int // 0: 払う・売る、1: やめる
	openedFrame int
}

func (s *shopPromptState) options() []string {
	if s.Kind == shopPromptSell {
		return []string{"売る", "やめる"}
	}
	return []string{"払う", "やめる"}
}

func (g *Game) openShopPrompt(prompt *shopPromptState) {
	prompt.openedFrame = g.inputFrame
	g.shopPrompt = prompt
}

// handleShopPromptInput は支払い・売却の窓の入力（左右で選び Z で決定、X でやめる）。
func (g *Game) handleShopPromptInput() {
	prompt := g.shopPrompt
	if prompt == nil || prompt.openedFrame == g.inputFrame {
		return // 開いたのと同じフレームの Z では決定しない
	}
	if g.keys().JustPressed(KeyLeft) || g.keys().JustPressed(KeyRight) || g.keys().JustPressed(KeyUp) || g.keys().JustPressed(KeyDown) {
		prompt.Selected = 1 - prompt.Selected
	}
	if g.keys().JustPressed(KeyX) {
		g.shopPrompt = nil
		return
	}
	if !g.keys().JustPressed(KeyZ) {
		return
	}
	g.shopPrompt = nil
	if prompt.Selected != 0 {
		return
	}
	switch prompt.Kind {
	case shopPromptPay:
		g.payShop()
	case shopPromptSell:
		g.sellToShop(prompt.Item, prompt.Price)
	}
}

// --- 中断セーブ ---

func (g *Game) savedShopState() *SavedShopState {
	if !g.shopAlarm && g.shopDebt == 0 {
		return nil
	}
	return &SavedShopState{Alarm: g.shopAlarm, Debt: g.shopDebt}
}

func (g *Game) applySavedShopState(s *SavedShopState) {
	g.resetShopState()
	if s != nil {
		g.shopAlarm, g.shopDebt = s.Alarm, max(s.Debt, 0)
	}
	forEachItemIn(g.state.Player.Inventory, func(item Item) {
		if itemShopPrice(item) > 0 {
			g.shopCarried = append(g.shopCarried, item)
		}
	})
}

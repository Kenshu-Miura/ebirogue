//go:build !test

package main

// BOT と店（wave10-B・wave11-C）。
//
// 全性格: 商品を拾いに行かず（経路でも商品のマスを通らず、行き先にもしない）、怒っていない店主を敵として扱わない。
// うっかり拾った商品は次の行動で店の床へ置いて返す。店主を巻き込む行動（店主が後ろにいる直線へ撃つ・投げる、
// 三方向の武器で店主を巻き込む振り、店主の隣の調べていないマスへ踏み込む〔大型地雷の爆風が店主に当たる〕、
// 未払いのまま店を出る1歩）はしない（botFeature "shopsafe"）。
//
// expert だけ:
//   - shopsell: 店の中で価値の低い道具（botSellMaxValue 以下）を床に置いて売る。
//   - shopbuy: 所持金（売る分を含む）で買える商品のうち、食料・回復薬・今より強い武器・盾・価値の高い道具を拾い、
//     店主に話しかけて払う。払えない品は店の中に置いて返す。満杯なら先に一番価値の低い道具を売る。
//   - money: 所持品が満杯で床のお金へ行けるとき、価値の低い道具を置いてお金を拾う（満杯だとお金も拾えない）。

const (
	botSellMaxValue   = 10 // 価値（itemValue）がこれ以下の道具は店で売る
	botBuyFoodMax     = 3  // 食料（保存の壺の中を含む）がこれ未満なら買う
	botBuyHealMax     = 2  // 回復の道具がこれ未満なら買う
	botBuyMinValue    = 50 // 食料・回復・装備以外は価値がこれ以上なら買う
	botShopEnemyRange = 5  // この距離以内に起きている敵がいたら売買しない
)

// expertHealItemIDs は expert が HP 40% 未満で使う回復の道具（bot_expert_test.go の回復の順番と同じ）。
var expertHealItemIDs = []int{173, 2, 28, 3, 29, 44}

// botIgnoresEnemy は BOT が敵として数えない（攻撃・逃走・射撃の対象にしない）相手か。怒っていない店主。
func botIgnoresEnemy(e Enemy) bool {
	return peacefulShopkeeper(e)
}

// botAvoidsItem は BOT が拾いに行かない床の道具か。値段の付いた商品。
func botAvoidsItem(item Item) bool {
	return itemShopPrice(item) > 0
}

// shopGoodsMask は経路探索で通らない・行き先にしない商品のマス（踏むと拾ってしまうため）。
// 買うと決めた商品（b.buyTarget）だけは行き先にできる。shopsafe を切ると nil（従来どおり通らないだけ）。
func (b *bot) shopGoodsMask(width, height int) []bool {
	if !botFeature("shopsafe") {
		return nil
	}
	mask := make([]bool, width*height)
	for _, item := range b.g.state.Items {
		if !botAvoidsItem(item) || item == b.buyTarget {
			continue
		}
		if x, y := item.GetPosition(); x >= 0 && y >= 0 && x < width && y < height {
			mask[y*width+x] = true
		}
	}
	// 店の中では見つけた罠のマスも行き先にしない（未払いのままワープ・落とし穴を踏むと泥棒になる）
	if room, ok := b.g.shopRoom(); ok && b.g.shopOpen() {
		for _, trap := range b.g.state.MapTraps {
			if trap.Discovered && roomContainsWithWalls(room, trap.X, trap.Y) && trap.X >= 0 && trap.Y >= 0 && trap.X < width && trap.Y < height {
				mask[trap.Y*width+trap.X] = true
			}
		}
	}
	return mask
}

// blockShopItems は経路探索で商品のマスを通れないものとする（踏むと拾ってしまうため）。
func (b *bot) blockShopItems(blocked []bool, width, height int) {
	for _, item := range b.g.state.Items {
		if !botAvoidsItem(item) || item == b.buyTarget {
			continue
		}
		if x, y := item.GetPosition(); x >= 0 && y >= 0 && x < width && y < height {
			blocked[y*width+x] = true
		}
	}
}

// --- 未払いの品 ---

// unpaidTotal は未払いの代金（持ち物と壺の中の未払いの品＋使った品の代金）。店が開いていなければ0。
func (b *bot) unpaidTotal() int {
	g := b.g
	if !g.shopOpen() {
		return 0
	}
	total := g.shopDebt
	forEachItemIn(g.state.Player.Inventory, func(item Item) { total += itemShopPrice(item) })
	return total
}

// shopBuying は expert が買い物をする（払える未払いの品を持ったままでよい）か。
func (b *bot) shopBuying() bool {
	return b.persona == personaExpert && botFeature("shopbuy")
}

// returnUnpaidItem は商品をうっかり拾ってしまったら、その場（店の床）へ置いて返す。置いたら true。
// 店を出ると泥棒になるので、拾った次の行動で返す。expert が買い物をするときは、払えない分だけ返す。
func (b *bot) returnUnpaidItem() (bool, error) {
	g := b.g
	p := g.state.Player
	if !g.shopOpen() || !g.inShopInterior(p.X, p.Y) || b.itemUnderPlayer() != nil {
		return false, nil
	}
	unpaid := b.unpaidTotal()
	if unpaid == 0 || (b.shopBuying() && unpaid <= p.Cash) {
		return false, nil
	}
	best, bestPrice := -1, 0
	for i, item := range p.Inventory {
		if price := itemShopPrice(item); price > bestPrice {
			best, bestPrice = i, price
		}
	}
	if best < 0 {
		return false, nil
	}
	if b.shopRejected == nil {
		b.shopRejected = map[Item]bool{}
	}
	b.shopRejected[p.Inventory[best]] = true // 同じ品を買いに戻らない
	b.buyTarget = nil
	b.act = "returnUnpaid"
	return b.useInventoryItem(best, "置く")
}

// leavesShopUnpaid は (dx, dy) へ1歩動くと未払いのまま店の部屋（壁の線を含む）の外に出るか。
func (b *bot) leavesShopUnpaid(dx, dy int) bool {
	g := b.g
	room, ok := g.shopRoom()
	p := g.state.Player
	return ok && b.unpaidTotal() > 0 && roomContainsWithWalls(room, p.X, p.Y) && !roomContainsWithWalls(room, p.X+dx, p.Y+dy)
}

// shopPromptKey は支払い・売却の窓で押すキー。expert は払える支払いと売るつもりで置いた品の売却を受け、
// それ以外は「やめる」（X）。
func (b *bot) shopPromptKey() Key {
	g := b.g
	prompt := g.shopPrompt
	if prompt == nil || b.persona != personaExpert {
		return KeyX
	}
	accept := false
	switch prompt.Kind {
	case shopPromptPay:
		accept = botFeature("shopbuy") && g.state.Player.Cash >= prompt.Price
	case shopPromptSell:
		accept = botFeature("shopsell")
	}
	if !accept {
		return KeyX
	}
	if prompt.Selected != 0 {
		return KeyLeft // 「払う・売る」へ戻してから次の入力で決める
	}
	return KeyZ
}

// --- 店主を巻き込まない（shopsafe） ---

// lineHitsShopkeeper は向き dx, dy へ飛ぶ道具（矢・投げた道具・魔法弾）の通り道に怒っていない店主がいるか。
// 矢は外れても後ろへ飛び続け、貫通する矢もあるので、最初の敵の後ろも壁まで（遠投なら壁の先も）調べる。
func (b *bot) lineHitsShopkeeper(dx, dy int) bool {
	if !botFeature("shopsafe") || !b.g.shopOpen() {
		return false
	}
	g := b.g
	dx, dy = sign(dx), sign(dy)
	if dx == 0 && dy == 0 {
		return false
	}
	x, y := g.state.Player.X, g.state.Player.Y
	pierceWalls := g.playerHasAbility(AbilityLongThrow) // 遠投は壁も抜ける
	for i := 0; i < 80; i++ {
		x, y = x+dx, y+dy
		if !g.inMapBounds(x, y) || (!pierceWalls && terrainStopsProjectile(g.state.Map[y][x])) {
			return false
		}
		if g.isPeacefulShopkeeper(g.enemyIndexAt(x, y)) {
			return true
		}
	}
	return false
}

// swingHitsShopkeeper は (dx, dy) へ振ると三方向の武器の横の振りが怒っていない店主に当たるか
// （正面の店主には攻撃せず話しかけるので数えない）。
func (b *bot) swingHitsShopkeeper(dx, dy int) bool {
	g := b.g
	if !botFeature("shopsafe") || !g.shopOpen() {
		return false
	}
	p := g.state.Player
	var abilities []EquipmentAbilityID
	if p.EquippedWeapon != nil {
		abilities = p.EquippedWeapon.Abilities
	}
	for i, d := range playerAttackDeltas(dx, dy, abilities) {
		if i > 0 && g.isPeacefulShopkeeper(g.enemyIndexAt(p.X+d.X, p.Y+d.Y)) {
			return true
		}
	}
	// 太刀の炎は HP が満タンのとき、振った後に直線の先の敵へ飛ぶ。正面の敵を殴って倒した後や空振り
	// （罠を調べる）では、その先の店主に当たる
	return g.playerHasAbility(AbilityFlameBlade) && p.Health >= p.MaxHealth && b.lineHitsShopkeeper(dx, dy)
}

// keeperSearchKey は店主の隣のマスを調べたかの記録（階ごと）。
type keeperSearchKey struct{ floor, x, y int }

// needsKeeperSearch は (x, y) へ踏み込む前に Z で罠を調べるか。店の中や店主の隣で地雷を踏むと、
// 爆風が店主に当たって泥棒になる（店の中にも敵が罠を置く）。未払いの品を持っている間はどのマスも調べる。
// 隣に敵がいるときは調べない。
func (b *bot) needsKeeperSearch(x, y int) bool {
	g := b.g
	if !botFeature("shopsafe") || !g.shopOpen() || b.keeperSearched[keeperSearchKey{g.Floor, x, y}] {
		return false
	}
	k := g.state.Enemies[g.shopkeeperIndex()]
	room, _ := g.shopRoom()
	// 店の部屋（壁の線を含む）の中のマスと店主の隣のマスは調べてから踏む（地雷の爆風が店主や商品に当たる）。
	// 未払いの品を持っている間はどのマスも調べる（ワープ・落とし穴で店の外へ出ると泥棒になる）
	near := max(abs(k.X-x), abs(k.Y-y)) == 1 || roomContainsWithWalls(room, x, y)
	if (!near && b.unpaidTotal() == 0) || g.mapTrapIndexAt(x, y) >= 0 && g.state.MapTraps[g.mapTrapIndexAt(x, y)].Discovered {
		return false
	}
	return len(b.adjacentEnemies()) == 0
}

// searchBeforeStep は (dx, dy) の先が調べるべきマスなら、向いて Z で調べる（1ターン）。調べたら true。
func (b *bot) searchBeforeStep(dx, dy int) (bool, error) {
	g := b.g
	p := g.state.Player
	x, y := p.X+dx, p.Y+dy
	if !b.needsKeeperSearch(x, y) {
		return false, nil
	}
	if b.keeperSearched == nil {
		b.keeperSearched = map[keeperSearchKey]bool{}
	}
	b.keeperSearched[keeperSearchKey{g.Floor, x, y}] = true
	if isOccupied(g, x, y) || b.swingHitsShopkeeper(dx, dy) {
		return false, nil
	}
	b.act = "searchKeeper"
	if err := b.tap(append([]Key{KeyA}, directionKeys(dx, dy)...)...); err != nil {
		return true, err
	}
	return true, b.tap(KeyZ)
}

// mineNearKeeper は (x, y) が発見済みの地雷・大型地雷で、怒っていない店主の隣か（踏むと泥棒になる）。
func (b *bot) mineNearKeeper(x, y int) bool {
	g := b.g
	if !botFeature("shopsafe") || !g.shopOpen() {
		return false
	}
	i := g.mapTrapIndexAt(x, y)
	if i < 0 || !g.state.MapTraps[i].Discovered {
		return false
	}
	if id := g.state.MapTraps[i].TemplateID; id != 3 && id != 11 {
		return false
	}
	k := g.state.Enemies[g.shopkeeperIndex()]
	return max(abs(k.X-x), abs(k.Y-y)) <= 1
}

// --- 売る（shopsell） ---

// shopBusy は売買をしない（近くに起きている敵がいる・店が開いていない）か。
func (b *bot) shopBusy() bool {
	return !b.g.shopOpen() || len(b.enemiesWithin(botShopEnemyRange)) > 0
}

// sellable は所持品 item を店で売れるか（装備中・未払い・値段0・手放さない道具は売らない）。
func (b *bot) sellable(item Item) (int, bool) {
	v, ok := b.itemValue(item)
	if !ok || itemShopPrice(item) > 0 || itemSellPrice(item) <= 0 {
		return 0, false
	}
	return v, true
}

// sellCandidate は売る道具の添字。価値の低い道具か、買う商品のために空けるなら一番価値の低い道具。
func (b *bot) sellCandidate() int {
	best, bestValue := -1, 1<<30
	for i, item := range b.g.state.Player.Inventory {
		if v, ok := b.sellable(item); ok && v < bestValue {
			best, bestValue = i, v
		}
	}
	if best < 0 {
		return -1
	}
	if bestValue <= botSellMaxValue {
		return best
	}
	// 満杯で買う商品があれば、それより十分に価値の低い道具を売って空ける
	if b.buyTarget != nil && b.inventoryFull() && bestValue+swapValueMargin <= b.buyValue(b.buyTarget) {
		return best
	}
	return -1
}

// freeShopTile は (x, y) が道具を置ける店の床（道具・罠・階段・店主・敵が無い）か。
func (b *bot) freeShopTile(x, y int) bool {
	g := b.g
	if !g.inShopInterior(x, y) || g.state.Map[y][x].Type == "stairs" || g.mapTrapIndexAt(x, y) >= 0 || !g.itemDropFree(x, y) {
		return false
	}
	for _, item := range g.state.Items {
		if ix, iy := item.GetPosition(); ix == x && iy == y {
			return false
		}
	}
	// 出入口のそばには置かない（商品のマスは通らないので、出入口をふさぐと出られなくなる）
	room, _ := g.shopRoom()
	for _, s := range botSteps {
		nx, ny := x+s[0], y+s[1]
		if g.inMapBounds(nx, ny) && !g.state.Map[ny][nx].Blocked && roomContainsWithWalls(room, nx, ny) && !roomContains(room, nx, ny) {
			return false
		}
	}
	p := g.state.Player
	if !(x == p.X && y == p.Y) && isOccupied(g, x, y) {
		return false
	}
	return b.shopStaysConnected(room, x, y)
}

// shopStaysConnected は (x, y) に商品を置いても、店の空いた床と出入口がすべてつながったままか
// （商品のマスは通らないので、売った品で自分や通り道を囲まないため）。
func (b *bot) shopStaysConnected(room Room, x, y int) bool {
	g := b.g
	goods := map[Coordinate]bool{{X: x, Y: y}: true}
	for _, item := range g.state.Items {
		if itemShopPrice(item) > 0 {
			ix, iy := item.GetPosition()
			goods[Coordinate{X: ix, Y: iy}] = true
		}
	}
	keeper := g.state.Enemies[g.shopkeeperIndex()]
	open := map[Coordinate]bool{}
	var start Coordinate
	found := false
	for ty := room.Y; ty < room.Y+room.Height; ty++ {
		for tx := room.X; tx < room.X+room.Width; tx++ {
			c := Coordinate{X: tx, Y: ty}
			if !g.inMapBounds(tx, ty) || g.state.Map[ty][tx].Blocked || goods[c] || (tx == keeper.X && ty == keeper.Y) {
				continue
			}
			open[c] = true
			if !found && !roomContains(room, tx, ty) {
				start, found = c, true // 出入口から数える
			}
		}
	}
	if !found {
		return false
	}
	seen := map[Coordinate]bool{start: true}
	queue := []Coordinate{start}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, s := range botSteps {
			n := Coordinate{X: c.X + s[0], Y: c.Y + s[1]}
			if open[n] && !seen[n] && b.canStep(c.X, c.Y, s[0], s[1]) {
				seen[n] = true
				queue = append(queue, n)
			}
		}
	}
	return len(seen) == len(open)
}

// expertSell は店の中で売る道具があれば、空いた床で置いて売る（売却の窓は shopPromptKey が受ける）。
func (b *bot) expertSell() (bool, error) {
	g := b.g
	p := g.state.Player
	if !botFeature("shopsell") || b.shopBusy() || !g.inShopInterior(p.X, p.Y) || b.unpaidTotal() > 0 {
		return false, nil
	}
	i := b.sellCandidate()
	if i < 0 {
		return false, nil
	}
	if !b.freeShopTile(p.X, p.Y) {
		if dx, dy, ok := b.pathStep(func(x, y int) bool { return b.freeShopTile(x, y) && !b.needsKeeperSearch(x, y) }); ok {
			b.act = "toSell"
			return true, b.move(dx, dy)
		}
		return false, nil
	}
	b.act = "sell"
	return b.useInventoryItem(i, "置く")
}

// --- 買う（shopbuy） ---

// buyValue は商品を買う価値（0 は買わない）。
func (b *bot) buyValue(item Item) int {
	p := &b.g.state.Player
	switch v := item.(type) {
	case *Food:
		if v.GetID() != rottenSausageItemID && b.foodCarried()+b.potFood() < botBuyFoodMax {
			return 100
		}
		return 0
	case *Weapon:
		if v.Cursed || v.Sharpness < 0 || satietyHarmfulEquipment(v.GetID()) {
			return 0
		}
		if p.EquippedWeapon == nil || (!p.EquippedWeapon.Cursed && v.AttackPower+v.Sharpness > p.EquippedWeapon.AttackPower+p.EquippedWeapon.Sharpness+1) {
			return 80
		}
		return 0
	case *Armor:
		if v.Cursed || v.Sharpness < 0 || satietyHarmfulEquipment(v.GetID()) || v.GetID() == disposableShieldID {
			return 0
		}
		cur := p.EquippedArmor
		if cur == nil {
			return 80
		}
		margin := 1
		if len(cur.Abilities) > 0 && cur.GetID() != disposableShieldID {
			margin = 2
		}
		if !cur.Cursed && v.DefensePower+v.Sharpness >= cur.DefensePower+cur.Sharpness+margin+1 {
			return 80
		}
		return 0
	}
	for _, id := range expertHealItemIDs {
		if item.GetID() == id {
			if b.healCarried() < botBuyHealMax {
				return 90
			}
			return 0
		}
	}
	if itemRoles[item.GetID()] == roleHarmful {
		return 0
	}
	if value := b.keepValue(item); value >= botBuyMinValue && value < 100 {
		return value
	}
	return 0
}

// healCarried は expert が使う回復の道具の数。
func (b *bot) healCarried() int {
	n := 0
	for _, item := range b.g.state.Player.Inventory {
		for _, id := range expertHealItemIDs {
			if item.GetID() == id {
				n++
			}
		}
	}
	return n
}

// shopBudget は買い物に使える額（所持金－未払い＋店で売る道具の売値）。
func (b *bot) shopBudget() int {
	p := &b.g.state.Player
	budget := p.Cash - b.unpaidTotal()
	if botFeature("shopsell") {
		for _, item := range p.Inventory {
			if v, ok := b.sellable(item); ok && v <= botSellMaxValue {
				budget += itemSellPrice(item)
			}
		}
	}
	return budget
}

// bestPurchase は買う商品（無ければ nil）。見えている店の商品のうち、買える中で価値の一番高いもの。
func (b *bot) bestPurchase() Item {
	g := b.g
	budget := b.shopBudget()
	full := b.inventoryFull()
	worstValue := 1 << 30
	if full {
		for _, item := range g.state.Player.Inventory {
			if v, ok := b.sellable(item); ok && v < worstValue {
				worstValue = v
			}
		}
	}
	var best Item
	bestValue, bestPrice := 0, 0
	for _, item := range g.state.Items {
		price := itemShopPrice(item)
		x, y := item.GetPosition()
		if price <= 0 || !g.inShopInterior(x, y) || !item.GetShowOnMiniMap() || b.shopRejected[item] {
			continue
		}
		v := b.buyValue(item)
		if v <= 0 || price > budget || (full && (worstValue == 1<<30 || worstValue+swapValueMargin > v)) {
			continue
		}
		if v > bestValue || (v == bestValue && price < bestPrice) {
			best, bestValue, bestPrice = item, v, price
		}
	}
	return best
}

// expertPay は払える未払いがあれば店主の隣へ行き、話しかけて払う（支払いの窓は shopPromptKey が受ける）。
func (b *bot) expertPay() (bool, error) {
	g := b.g
	p := g.state.Player
	unpaid := b.unpaidTotal()
	if unpaid == 0 || unpaid > p.Cash || len(b.adjacentEnemies()) > 0 {
		return false, nil
	}
	k := g.state.Enemies[g.shopkeeperIndex()]
	if dx, dy := k.X-p.X, k.Y-p.Y; max(abs(dx), abs(dy)) == 1 {
		b.act = "pay"
		if err := b.tap(append([]Key{KeyA}, directionKeys(dx, dy)...)...); err != nil {
			return true, err
		}
		return true, b.tap(KeyZ)
	}
	nextToKeeper := func(x, y int) bool {
		return max(abs(k.X-x), abs(k.Y-y)) == 1 && g.inShopInterior(x, y)
	}
	if dx, dy, ok := b.pathStep(nextToKeeper); ok {
		b.act = "toPay"
		return true, b.move(dx, dy)
	}
	return false, nil
}

// expertShop は店での売買の1行動（払う → 売る → 買いに行く）。行動したら true。
func (b *bot) expertShop() (bool, error) {
	g := b.g
	if b.buyTarget != nil && (!g.itemOnFloor(b.buyTarget) || itemShopPrice(b.buyTarget) == 0) {
		b.buyTarget = nil // 拾った・ほかの理由で無くなった
	}
	if !g.shopOpen() {
		b.buyTarget = nil
		return false, nil
	}
	if b.shopBuying() {
		if done, err := b.expertPay(); done || err != nil {
			return done, err
		}
	}
	if b.shopBusy() || b.unpaidTotal() > 0 {
		return false, nil
	}
	if b.shopBuying() {
		b.buyTarget = b.bestPurchase()
	}
	if done, err := b.expertSell(); done || err != nil {
		return done, err
	}
	if b.buyTarget == nil || b.inventoryFull() {
		b.buyTarget = nil
		return false, nil
	}
	tx, ty := b.buyTarget.GetPosition()
	if dx, dy, ok := b.pathStep(func(x, y int) bool { return x == tx && y == ty }); ok {
		b.act = "toBuy"
		return true, b.move(dx, dy)
	}
	b.buyTarget = nil
	return false, nil
}

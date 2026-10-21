//go:build !test

package main

// BOT と店（wave10-B）。売買は wave11 で考える。今は商品を拾わず、怒っていない店主を敵として扱わず、
// 支払い・売却の窓は「やめる」（X）で閉じる（windowOpen / closeWindows）。

// botIgnoresEnemy は BOT が敵として数えない（攻撃・逃走・射撃の対象にしない）相手か。怒っていない店主。
func botIgnoresEnemy(e Enemy) bool {
	return peacefulShopkeeper(e)
}

// botAvoidsItem は BOT が拾いに行かない床の道具か。値段の付いた商品。
func botAvoidsItem(item Item) bool {
	return itemShopPrice(item) > 0
}

// blockShopItems は経路探索で商品のマスを通れないものとする（踏むと拾ってしまうため）。
func (b *bot) blockShopItems(blocked []bool, width, height int) {
	for _, item := range b.g.state.Items {
		if !botAvoidsItem(item) {
			continue
		}
		if x, y := item.GetPosition(); x >= 0 && y >= 0 && x < width && y < height {
			blocked[y*width+x] = true
		}
	}
}

// returnUnpaidItem は商品をうっかり拾ってしまったら、その場（店の床）へ置いて返す。置いたら true。
// 店を出ると泥棒になるので、拾った次の行動で返す。
func (b *bot) returnUnpaidItem() (bool, error) {
	g := b.g
	if !g.shopOpen() || !g.inShopInterior(g.state.Player.X, g.state.Player.Y) || b.itemUnderPlayer() != nil {
		return false, nil
	}
	for i, item := range g.state.Player.Inventory {
		if itemShopPrice(item) > 0 {
			return b.useInventoryItem(i, "置く")
		}
	}
	return false, nil
}

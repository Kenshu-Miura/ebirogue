//go:build !test

package main

// eatRottenSausage は通常の食料と違い、満腹時にも最大値を増やさない。
var eatRottenSausage = func(g *Game) {
	item, inventory := determineItemSource(g)
	g.Enqueue(Action{Duration: 0.4, Message: item.GetName() + "を食べた", Execute: func(g *Game) {
		player := &g.state.Player
		player.Satiety = recoveredValue(player.Satiety, player.MaxSatiety, 10, false)
		player.StatusAilments.Poison = max(player.StatusAilments.Poison, 8)
		// 通常の食料と同じ消費経路を使い、副作用を遅延実行する。
		for i, owned := range player.Inventory {
			if inventory && owned == item {
				g.selectedItemIndex = i
				removeUsedItem(g, true)
				g.selectedItemIndex = 0
				g.EnqueueMessage("満腹度が少し回復したが、毒状態になった", 0.4)
				return
			}
		}
		if !inventory {
			for i, ground := range g.state.Items {
				if ground == item {
					g.selectedGroundItemIndex = i
					removeUsedItem(g, false)
					break
				}
			}
		}
		g.EnqueueMessage("満腹度が少し回復したが、毒状態になった", 0.4)
	}})
}

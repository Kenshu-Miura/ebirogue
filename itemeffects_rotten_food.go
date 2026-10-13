//go:build !test

package main

const (
	rottenSausageItemID    = 71 // 腐ったウインナー（腐敗の罠が食料を変える先）
	rottenFoodPoisonTurns  = 8  // 腐った食料を食べたときの毒のターン数
	rottenFoodEatenMessage = "満腹度が少し回復したが、毒状態になった"
)

// eatRottenSausage は通常の食料と違い、満腹時にも最大値を増やさない。
var eatRottenSausage = func(g *Game) {
	item, _ := determineItemSource(g)
	g.Enqueue(Action{Duration: 0.4, Message: item.GetName() + "を食べた", Execute: func(g *Game) {
		player := &g.state.Player
		recovery := 0
		if food, ok := item.(*Food); ok {
			recovery = food.Satiety
		}
		player.Satiety = recoveredValue(player.Satiety, player.MaxSatiety, recovery, false)
		player.StatusAilments.Poison = max(player.StatusAilments.Poison, rottenFoodPoisonTurns)
		// 実行までに並びが変わっても、所持品か足元から同じオブジェクトを消費する。
		g.removeThrownItemFromSource(item)
		g.selectedItemIndex = 0
		g.EnqueueMessage(rottenFoodEatenMessage, 0.4)
	}})
}

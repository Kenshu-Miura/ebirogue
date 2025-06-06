//go:build !test
// +build !test

package main

import (
	"fmt"
)

func determineItemSource(g *Game) (item Item, isInventoryItem bool) {
	if g.GroundItemActioned {
		// 地面からのアイテムの場合
		item = g.state.Items[g.selectedGroundItemIndex]
	} else {
		// インベントリからのアイテムの場合
		item = g.state.Player.Inventory[g.selectedItemIndex]
		isInventoryItem = true
	}
	return item, isInventoryItem
}

func removeUsedItem(g *Game, isInventoryItem bool) {
	if isInventoryItem {
		// インベントリからアイテムを削除
		g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
	} else {
		// 地面からアイテムを削除
		g.state.Items = append(g.state.Items[:g.selectedGroundItemIndex], g.state.Items[g.selectedGroundItemIndex+1:]...)
	}
}

var restoreSatiety50 = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)

	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを食べた", item.GetName()),
		Execute: func(g *Game) {
		},
	}
	g.Enqueue(action)
	if g.state.Player.Satiety == g.state.Player.MaxSatiety {
		action := Action{
			Duration: 0.4,
			Message:  "最大満腹度が1上昇した。",
			Execute: func(g *Game) {
				g.state.Player.MaxSatiety++
			},
		}
		g.Enqueue(action)
	} else {
		if foodItem, ok := item.(*Food); ok {
			action := Action{
				Duration: 0.4,
				Message:  fmt.Sprintf("満腹度が%d回復した。", foodItem.Satiety),
				Execute: func(g *Game) {
					g.state.Player.Satiety += foodItem.Satiety
					if g.state.Player.Satiety > g.state.Player.MaxSatiety {
						g.state.Player.Satiety = g.state.Player.MaxSatiety
					}
				},
			}
			g.Enqueue(action)
		}
	}
	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

var restoreHP30 = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)

	// アクションの生成
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを食べた", item.GetName()),
		Execute:  func(g *Game) {},
	}
	g.Enqueue(action)

	// HP回復の処理
	if g.state.Player.Health == g.state.Player.MaxHealth {
		action := Action{
			Duration: 0.4,
			Message:  "最大HPが1上昇した。",
			Execute: func(g *Game) {
				g.state.Player.MaxHealth++
			},
		}
		g.Enqueue(action)
	} else {
		if potionItem, ok := item.(*Potion); ok {
			action := Action{
				Duration: 0.4,
				Message:  fmt.Sprintf("HPが%d回復した。", potionItem.Health),
				Execute: func(g *Game) {
					g.state.Player.Health += potionItem.Health
					if g.state.Player.Health > g.state.Player.MaxHealth {
						g.state.Player.Health = g.state.Player.MaxHealth
					}
				},
			}
			g.Enqueue(action)
		}
	}

	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

var restoreHP100 = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)

	// アクションの生成
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを食べた", item.GetName()),
		Execute:  func(g *Game) {},
	}
	g.Enqueue(action)

	// HP回復の処理
	if g.state.Player.Health == g.state.Player.MaxHealth {
		action := Action{
			Duration: 0.4,
			Message:  "最大HPが2上昇した。",
			Execute: func(g *Game) {
				g.state.Player.MaxHealth += 2
			},
		}
		g.Enqueue(action)
	} else {
		if potionItem, ok := item.(*Potion); ok {
			action := Action{
				Duration: 0.4,
				Message:  fmt.Sprintf("HPが%d回復した。", potionItem.Health),
				Execute: func(g *Game) {
					g.state.Player.Health += potionItem.Health
					if g.state.Player.Health > g.state.Player.MaxHealth {
						g.state.Player.Health = g.state.Player.MaxHealth
					}
				},
			}
			g.Enqueue(action)
		}
	}

	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

var damageHP30 = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使った。", item.GetName()),
		Execute: func(g *Game) {
		},
	}
	g.Enqueue(action)

	action = Action{
		Duration: 0.4,
		Message:  "",
		Execute: func(g *Game) {
			var targetX, targetY int
			switch g.state.Player.Direction {
			case Up:
				targetX, targetY = g.state.Player.X, g.state.Player.Y-1
			case Down:
				targetX, targetY = g.state.Player.X, g.state.Player.Y+1
			case Left:
				targetX, targetY = g.state.Player.X-1, g.state.Player.Y
			case Right:
				targetX, targetY = g.state.Player.X+1, g.state.Player.Y
			case UpRight:
				targetX, targetY = g.state.Player.X+1, g.state.Player.Y-1
			case DownRight:
				targetX, targetY = g.state.Player.X+1, g.state.Player.Y+1
			case UpLeft:
				targetX, targetY = g.state.Player.X-1, g.state.Player.Y-1
			case DownLeft:
				targetX, targetY = g.state.Player.X-1, g.state.Player.Y+1
			}
			for i, enemy := range g.state.Enemies {
				if enemy.X == targetX && enemy.Y == targetY {
					action := Action{
						Duration: 0.5,
						Message:  fmt.Sprintf("%sに30ダメージを与えた。", g.state.Enemies[i].Name),
						Execute: func(g *Game) {
							g.state.Enemies[i].Health -= 30
							if g.state.Enemies[i].Health <= 0 {
								// 敵のHealthが0以下の場合、敵を配列から削除
								defeatAction := Action{
									Duration: 0.5,
									Message:  fmt.Sprintf("%sを倒した。", g.state.Enemies[i].Name),
									Execute:  func(g *Game) {},
								}
								g.Enqueue(defeatAction)

								g.state.Enemies = append(g.state.Enemies[:i], g.state.Enemies[i+1:]...)

								// 敵の経験値をプレイヤーの所持経験値に加える
								g.state.Player.ExperiencePoints += enemy.ExperiencePoints

								g.state.Player.checkLevelUp() // レベルアップをチェック
							}
						},
					}
					g.Enqueue(action)
					break
				}
			}
		},
	}
	g.Enqueue(action)
	removeUsedItem(g, isInventoryItem)
}

var money = func(g *Game) {
	moneyItem := g.state.Player.Inventory[g.selectedItemIndex].(*Money)
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%dを入手した。", moneyItem.Amount),
		Execute: func(g *Game) {
			g.state.Player.Cash += moneyItem.Amount
		},
	}
	g.Enqueue(action)
	g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
}

var setTrap = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	if trapItem, ok := item.(*Trap); ok {
		action := Action{
			Duration: 0.4,
			Message:  fmt.Sprintf("%sをセットした。", trapItem.GetName()),
			Execute: func(g *Game) {
				g.state.Player.SetTrap = trapItem // Set the trap
			},
		}
		g.Enqueue(action)
	}

	removeUsedItem(g, isInventoryItem)
}

var shiftChange = func(g *Game) {
	//プレイヤーとインデックスの敵の位置を入れ替える
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sと入れ替わった", g.state.Enemies[g.TargetEnemyIndex].GetName()),
		Execute: func(g *Game) {
			g.state.Player.X, g.state.Player.Y, g.state.Enemies[g.TargetEnemyIndex].X, g.state.Enemies[g.TargetEnemyIndex].Y = g.state.Enemies[g.TargetEnemyIndex].X, g.state.Enemies[g.TargetEnemyIndex].Y, g.state.Player.X, g.state.Player.Y
			g.TargetEnemyIndex = -1
		},
	}
	g.Enqueue(action)

}

var identifyItem = func(g *Game) {
	_, isInventoryItem := determineItemSource(g)

	if isInventoryItem {
		g.tmpselectedItemIndex = g.selectedItemIndex
	} else {
		g.tmpselectedItemIndex = g.selectedGroundItemIndex
	}

	g.useidentifyItem = true
	g.showInventory = true
}

func (g *Game) executeItemIdentify() {
	g.showInventory = false
	item, _ := determineItemSource(g)

	if identifiableItem, ok := item.(Identifiable); ok {

		action := Action{
			Duration: 0.5,
			ItemName: identifiableItem.GetName(),
			Message:  fmt.Sprintf("%sを識別した。", identifiableItem.GetName()),
			Execute: func(g *Game) {
			},
			IsIdentified: identifiableItem.GetIdentified(),
		}
		g.Enqueue(action)

		identifiableItem.SetIdentified(true)

	}

	action := Action{
		Duration: 0.5,
		ItemName: getItemNameWithSharpness(item),
		Message:  fmt.Sprintf("アイテムの正体は%sだった。", getItemNameWithSharpness(item)),
		Execute: func(g *Game) {
		},
		IsIdentified: true,
	}
	g.Enqueue(action)

	_, isInventoryItem := determineItemSource(g)

	if isInventoryItem {
		// インベントリからアイテムを削除
		g.state.Player.Inventory = append(g.state.Player.Inventory[:g.tmpselectedItemIndex], g.state.Player.Inventory[g.tmpselectedItemIndex+1:]...)
	} else {
		// 地面からアイテムを削除
		g.state.Items = append(g.state.Items[:g.tmpselectedItemIndex], g.state.Items[g.tmpselectedItemIndex+1:]...)
	}

	g.tmpselectedItemIndex = -1
	g.selectedItemIndex = 0
	g.useidentifyItem = false
}

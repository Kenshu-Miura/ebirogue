package main

import "fmt"

var restoreSatiety50 = func(g *Game) {
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを食べた", g.state.Player.Inventory[g.selectedItemIndex].GetName()),
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
		foodItem := g.state.Player.Inventory[g.selectedItemIndex].(*Food) // Assumes item is of type *Food
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
	g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
}

var restoreHP30 = func(g *Game) {
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを食べた", g.state.Player.Inventory[g.selectedItemIndex].GetName()),
		Execute: func(g *Game) {
		},
	}
	g.Enqueue(action)
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
		potionItem := g.state.Player.Inventory[g.selectedItemIndex].(*Potion) // Assumes item is of type *Food
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
	g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
}

var restoreHP100 = func(g *Game) {
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを食べた", g.state.Player.Inventory[g.selectedItemIndex].GetName()),
		Execute: func(g *Game) {
		},
	}
	g.Enqueue(action)
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
		potionItem := g.state.Player.Inventory[g.selectedItemIndex].(*Potion) // Assumes item is of type *Food
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
	g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
}

var damageHP30 = func(g *Game) {
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使った。", g.state.Player.Inventory[g.selectedItemIndex].GetName()),
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
	g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
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

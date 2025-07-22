//go:build !test

package main

import (
	"fmt"
	"log"
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
					// 目潰し状態の場合は敵の名前を「何者」に変更
					enemyDisplayName := g.state.Enemies[i].Name
					if g.state.Player.StatusAilments.Blind > 0 {
						enemyDisplayName = "何者"
					}
					
					action := Action{
						Duration: 0.5,
						Message:  fmt.Sprintf("%sに30ダメージを与えた。", enemyDisplayName),
						Execute: func(g *Game) {
							g.state.Enemies[i].Health -= 30
							
							// ダメージを受けた敵の金縛り状態を解除
							if g.state.Enemies[i].StatusAilments.Paralysis {
								g.state.Enemies[i].StatusAilments.Paralysis = false
							}
							
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

var sealEnemy = func(g *Game) {
	//インデックスの敵を封印状態にする
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを封印状態にした", g.state.Enemies[g.TargetEnemyIndex].GetName()),
		Execute: func(g *Game) {
			g.state.Enemies[g.TargetEnemyIndex].StatusAilments.Seal = true
			g.TargetEnemyIndex = -1
		},
	}
	g.Enqueue(action)

}

var identifyItem = func(g *Game) {
	_, isInventoryItem := determineItemSource(g)

	if isInventoryItem {
		g.tmpSelectedItemIndex = g.selectedItemIndex
	} else {
		g.tmpSelectedItemIndex = g.selectedGroundItemIndex
	}

	g.useIdentifyItem = true
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
		g.state.Player.Inventory = append(g.state.Player.Inventory[:g.tmpSelectedItemIndex], g.state.Player.Inventory[g.tmpSelectedItemIndex+1:]...)
	} else {
		// 地面からアイテムを削除
		g.state.Items = append(g.state.Items[:g.tmpSelectedItemIndex], g.state.Items[g.tmpSelectedItemIndex+1:]...)
	}

	g.tmpSelectedItemIndex = -1
	g.selectedItemIndex = 0
	g.useIdentifyItem = false
}

// 睡眠効果関数 - 部屋内では同じ部屋の敵全員、通路では周囲1マスの敵を睡眠状態にする
var sleepAllEnemiesInRoom = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した", item.GetName()),
		Execute: func(g *Game) {
			playerX, playerY := g.state.Player.GetPosition()
			enemiesPutToSleep := 0
			
			// プレイヤーが部屋内にいるかどうかを判定
			if isInsideRoom(playerX, playerY, g.rooms) {
				// 部屋内の場合：同じ部屋にいる敵を全員睡眠状態にする
				for i := range g.state.Enemies {
					enemyX, enemyY := g.state.Enemies[i].GetPosition()
					if isSameRoom(playerX, playerY, enemyX, enemyY, g.rooms) {
						g.state.Enemies[i].StatusAilments.Sleep = 10 // 10ターン睡眠
						// 個別の敵に対してメッセージを追加
						action := Action{
							Duration: 0.4,
							Message:  fmt.Sprintf("%sは眠った", g.state.Enemies[i].Name),
							Execute:  func(g *Game) {},
						}
						g.Enqueue(action)
						enemiesPutToSleep++
					}
				}
			} else {
				// 通路の場合：周囲1マスの敵を睡眠状態にする
				for i := range g.state.Enemies {
					enemyX, enemyY := g.state.Enemies[i].GetPosition()
					// 周囲1マス以内の判定
					if abs(enemyX-playerX) <= 1 && abs(enemyY-playerY) <= 1 {
						g.state.Enemies[i].StatusAilments.Sleep = 10 // 10ターン睡眠
						// 個別の敵に対してメッセージを追加
						action := Action{
							Duration: 0.4,
							Message:  fmt.Sprintf("%sは眠った", g.state.Enemies[i].Name),
							Execute:  func(g *Game) {},
						}
						g.Enqueue(action)
						enemiesPutToSleep++
					}
				}
			}
		},
	}
	g.Enqueue(action)
	
	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

// 混乱効果関数 - 部屋内では同じ部屋の敵全員、通路では周囲1マスの敵を混乱状態にする
var confuseAllEnemiesInRoom = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した", item.GetName()),
		Execute: func(g *Game) {
			playerX, playerY := g.state.Player.GetPosition()
			enemiesConfused := 0
			
			// プレイヤーが部屋内にいるかどうかを判定
			if isInsideRoom(playerX, playerY, g.rooms) {
				// 部屋内の場合：同じ部屋にいる敵を全員混乱状態にする
				for i := range g.state.Enemies {
					enemyX, enemyY := g.state.Enemies[i].GetPosition()
					if isSameRoom(playerX, playerY, enemyX, enemyY, g.rooms) {
						g.state.Enemies[i].StatusAilments.Confusion = 10 // 10ターン混乱
						// 個別の敵に対してメッセージを追加
						action := Action{
							Duration: 0.4,
							Message:  fmt.Sprintf("%sは混乱した", g.state.Enemies[i].Name),
							Execute:  func(g *Game) {},
						}
						g.Enqueue(action)
						enemiesConfused++
					}
				}
			} else {
				// 通路の場合：周囲1マスの敵を混乱状態にする
				for i := range g.state.Enemies {
					enemyX, enemyY := g.state.Enemies[i].GetPosition()
					// 周囲1マス以内の判定
					if abs(enemyX-playerX) <= 1 && abs(enemyY-playerY) <= 1 {
						g.state.Enemies[i].StatusAilments.Confusion = 10 // 10ターン混乱
						// 個別の敵に対してメッセージを追加
						action := Action{
							Duration: 0.4,
							Message:  fmt.Sprintf("%sは混乱した", g.state.Enemies[i].Name),
							Execute:  func(g *Game) {},
						}
						g.Enqueue(action)
						enemiesConfused++
					}
				}
			}
		},
	}
	g.Enqueue(action)
	
	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

// 目潰し効果関数 - 部屋内では同じ部屋の敵全員、通路では周囲1マスの敵を目潰し状態にする
var blindAllEnemiesInRoom = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した", item.GetName()),
		Execute: func(g *Game) {
			playerX, playerY := g.state.Player.GetPosition()
			enemiesBlinded := 0
			
			// プレイヤーが部屋内にいるかどうかを判定
			if isInsideRoom(playerX, playerY, g.rooms) {
				// 部屋内の場合：同じ部屋にいる敵を全員目潰し状態にする
				for i := range g.state.Enemies {
					enemyX, enemyY := g.state.Enemies[i].GetPosition()
					if isSameRoom(playerX, playerY, enemyX, enemyY, g.rooms) {
						g.state.Enemies[i].StatusAilments.Blind = 10 // 10ターン目潰し状態
						// 個別の敵に対してメッセージを追加
						action := Action{
							Duration: 0.4,
							Message:  fmt.Sprintf("%sは目が見えなくなった", g.state.Enemies[i].Name),
							Execute:  func(g *Game) {},
						}
						g.Enqueue(action)
						enemiesBlinded++
					}
				}
			} else {
				// 通路の場合：周囲1マスの敵を目潰し状態にする
				for i := range g.state.Enemies {
					enemyX, enemyY := g.state.Enemies[i].GetPosition()
					// 周囲1マス以内の判定
					if abs(enemyX-playerX) <= 1 && abs(enemyY-playerY) <= 1 {
						g.state.Enemies[i].StatusAilments.Blind = 10 // 10ターン目潰し状態
						// 個別の敵に対してメッセージを追加
						action := Action{
							Duration: 0.4,
							Message:  fmt.Sprintf("%sは目が見えなくなった", g.state.Enemies[i].Name),
							Execute:  func(g *Game) {},
						}
						g.Enqueue(action)
						enemiesBlinded++
					}
				}
			}
		},
	}
	g.Enqueue(action)
	
	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

// 金縛り効果関数 - 周囲8マスの敵を金縛り状態にする
var paralyzeAllEnemiesAround = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した", item.GetName()),
		Execute: func(g *Game) {
			playerX, playerY := g.state.Player.GetPosition()
			enemiesParalyzed := 0
			
			// 周囲8マスの敵を金縛り状態にする
			for i := range g.state.Enemies {
				enemyX, enemyY := g.state.Enemies[i].GetPosition()
				// 周囲8マス以内の判定（8方向）
				if abs(enemyX-playerX) <= 1 && abs(enemyY-playerY) <= 1 && !(enemyX == playerX && enemyY == playerY) {
					g.state.Enemies[i].StatusAilments.Paralysis = true // 金縛り状態
					enemiesParalyzed++
				}
			}
			
			if enemiesParalyzed > 0 {
				log.Printf("周囲の敵%d体を金縛り状態にした", enemiesParalyzed)
			} else {
				log.Printf("周囲に敵がいない")
			}
		},
	}
	g.Enqueue(action)
	
	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

// 睡眠ポーション効果関数 - プレイヤーを睡眠状態にする
var sleepPotion = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを飲んだ", item.GetName()),
		Execute: func(g *Game) {
			g.state.Player.StatusAilments.Sleep = 10 // 10ターン睡眠
		},
	}
	g.Enqueue(action)
	
	action = Action{
		Duration: 0.4,
		Message:  "海老さんは眠った",
		Execute: func(g *Game) {},
	}
	g.Enqueue(action)
	
	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

// 混乱ポーション効果関数 - プレイヤーを混乱状態にする
var confusionPotion = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを飲んだ", item.GetName()),
		Execute: func(g *Game) {
			g.state.Player.StatusAilments.Confusion = 10 // 10ターン混乱
		},
	}
	g.Enqueue(action)
	
	action = Action{
		Duration: 0.4,
		Message:  "海老さんは混乱した",
		Execute: func(g *Game) {},
	}
	g.Enqueue(action)
	
	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

// 目潰しポーション効果関数 - プレイヤーを目潰し状態にする
var blindPotion = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを飲んだ", item.GetName()),
		Execute: func(g *Game) {
			g.state.Player.StatusAilments.Blind = 30 // 30ターン目潰し状態
		},
	}
	g.Enqueue(action)
	
	action = Action{
		Duration: 0.4,
		Message:  "海老さんは目が見えなくなった",
		Execute: func(g *Game) {},
	}
	g.Enqueue(action)
	
	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

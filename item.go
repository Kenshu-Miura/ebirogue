package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
	"math/rand"
)

func (g *Game) UpdateThrownItem() {
	if g.ThrownItem.Image != nil {
		g.ThrownItem.X += g.ThrownItem.DX
		g.ThrownItem.Y += g.ThrownItem.DY
		// 必要に応じてアイテムが目的地に到達したかどうかをチェックし、到達したらリストから削除
		if (g.ThrownItem.DX >= 0 && g.ThrownItem.X*tileSize >= g.ThrownItemDestination.X*tileSize) || (g.ThrownItem.DX < 0 && g.ThrownItem.X*tileSize <= g.ThrownItemDestination.X*tileSize) {
			if (g.ThrownItem.DY >= 0 && g.ThrownItem.Y*tileSize >= g.ThrownItemDestination.Y*tileSize) || (g.ThrownItem.DY < 0 && g.ThrownItem.Y*tileSize <= g.ThrownItemDestination.Y*tileSize) {
				if g.TargetEnemy != nil {
					// 敵にアイテムが当たった時の処理を実行
					if g.onEnemyHit != nil {
						g.onEnemyHit(g.TargetEnemy, g.ThrownItem.Item, g.TargetEnemyIndex) // g.TargetEnemyIndexは敵のインデックスを指定する仮定の変数です
					}
					g.TargetEnemy = nil
				} else {
					g.state.Items = append(g.state.Items, g.ThrownItem.Item)
				}
				g.ThrownItem = ThrownItem{}
				g.ThrownItemDestination = Coordinate{}
			}
		}
	}
}

func (g *Game) ThrowItem(item Item, throwRange int, character Character, mapState [][]Tile, enemies []Enemy, onWallHit func(Item, Coordinate, int), onTargetHit func(Character, Item, int)) {
	var dx, dy int
	switch character.GetDirection() {
	case Up:
		dx, dy = 0, -1
	case Down:
		dx, dy = 0, 1
	case Left:
		dx, dy = -1, 0
	case Right:
		dx, dy = 1, 0
	case UpRight:
		dx, dy = 1, -1
	case DownRight:
		dx, dy = 1, 1
	case UpLeft:
		dx, dy = -1, -1
	case DownLeft:
		dx, dy = -1, 1
	}

	x, y := character.GetPosition()
	itemName := getItemNameWithSharpness(item)
	action := Action{
		Duration: 0.5,
		Message:  fmt.Sprintf("%sを投げた", itemName),
		Execute: func(g *Game) {
			g.ThrownItem = ThrownItem{
				Item:  item,
				Image: g.getItemImage(item),
				X:     x,
				Y:     y,
				DX:    dx,
				DY:    dy,
			}

			var i int
			for i = 1; i <= throwRange; i++ {
				targetX := x + i*dx
				targetY := y + i*dy
				tile := mapState[targetY][targetX]
				if tile.Type == "wall" {
					position := Coordinate{
						X: x + (i-1)*dx,
						Y: y + (i-1)*dy,
					}
					onWallHit(item, position, g.selectedItemIndex)
					return
				}
				for index, enemy := range enemies {
					if enemy.X == targetX && enemy.Y == targetY {

						g.ThrownItemDestination = Coordinate{
							X: targetX,
							Y: targetY,
						}

						g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)

						g.TargetEnemy = &enemy
						g.TargetEnemyIndex = index
						g.onEnemyHit = onTargetHit

						g.showItemActions = false
						g.showInventory = false

						g.selectedItemIndex = 0
						g.selectedActionIndex = 0
						return
					}
				}
				// Check if the item hits the player
				if targetX == g.state.Player.X && targetY == g.state.Player.Y {
					g.ThrownItemDestination = Coordinate{
						X: targetX,
						Y: targetY,
					}
					onTargetHit(&g.state.Player, item, g.selectedItemIndex) // Passing a pointer to g.state.Player
					return
				}
			}
			if i == throwRange+1 {
				position := Coordinate{
					X: x + (i-1)*dx,
					Y: y + (i-1)*dy,
				}
				onWallHit(item, position, g.selectedItemIndex) // Assuming the item will stop at the maximum range if no wall or enemy is encountered
			}
		},
	}
	g.Enqueue(action)
}

func (g *Game) onWallHit(item Item, position Coordinate, itemIndex int) {
	// Set the position of the item to the position before hitting the wall
	item.SetPosition(position.X, position.Y)

	// Update the ThrownItemDestination to the position before hitting the wall
	g.ThrownItemDestination = position

	// Remove the item from the player's inventory
	g.state.Player.Inventory = append(g.state.Player.Inventory[:itemIndex], g.state.Player.Inventory[itemIndex+1:]...)

	// Update the UI flags
	g.showItemActions = false
	g.showInventory = false
	g.isActioned = true
	g.selectedItemIndex = 0
	g.selectedActionIndex = 0
}

func (g *Game) onTargetHit(target Character, item Item, index int) {
	if potion, ok := item.(*Potion); ok {
		action := Action{
			Duration: 0.5, // Assuming a duration of 0.5 seconds for this action
			Message:  fmt.Sprintf("%sのHPが%d回復した。", target.GetName(), potion.Health),
			Execute: func(*Game) {
				target.SetHealth(target.GetHealth() + potion.Health)
				if target.GetHealth() > target.GetMaxHealth() {
					target.SetHealth(target.GetMaxHealth())
				}
				g.isActioned = true
				// Reset the target character after processing
			},
		}
		g.Enqueue(action)
	} else {
		damage := rand.Intn(3) + 1
		action := Action{
			Duration: 0.5, // Assuming a duration of 0.5 seconds for this action
			Message:  fmt.Sprintf("%sに%dのダメージを与えた。", target.GetName(), damage),
			Execute: func(*Game) {
				target.SetHealth(target.GetHealth() - damage)
				if target.GetHealth() < 0 {
					target.SetHealth(0)
				}
				if enemy, isEnemy := target.(*Enemy); isEnemy && target.GetHealth() <= 0 {
					// 敵のHealthが0以下の場合、敵を配列から削除
					defeatAction := Action{
						Duration: 0.5,
						Message:  fmt.Sprintf("%sを倒した。", target.GetName()),
						Execute:  func(g *Game) {},
					}
					g.Enqueue(defeatAction)

					g.state.Enemies = append(g.state.Enemies[:index], g.state.Enemies[index+1:]...)

					// 敵の経験値をプレイヤーの所持経験値に加える
					g.state.Player.ExperiencePoints += enemy.ExperiencePoints

					g.state.Player.checkLevelUp() // レベルアップをチェック

					// Reset the target enemy after processing
					// (If necessary. This part may need to be adjusted based on your game's logic)
				}
				g.isActioned = true
			},
		}
		g.Enqueue(action)
	}
}

// Additional function to check if item is equipped
func isEquipped(equippedItems []Item, item Equipable) bool {
	for _, equippedItem := range equippedItems {
		if equippedItem == item {
			return true
		}
	}
	return false
}

// UpdatePlayerStats is a method to update player stats when equipping/unequipping an item
// This method needs to be implemented by each equipable item type (Weapon, Armor, Arrow, Accessory)
func (w *Weapon) UpdatePlayerStats(player *Player, equip bool) {
	if equip {
		player.AttackPower += w.AttackPower + w.Sharpness
	} else {
		player.AttackPower -= w.AttackPower + w.Sharpness
	}
}

func (a *Armor) UpdatePlayerStats(player *Player, equip bool) {
	if equip {
		player.DefensePower += a.DefensePower + a.Sharpness
	} else {
		player.DefensePower -= a.DefensePower + a.Sharpness
	}
}

func (ar *Arrow) UpdatePlayerStats(player *Player, equip bool) {
	// Arrows might not affect player stats but can affect other stats like ammo count
	// Implement logic accordingly
}

func (ac *Accessory) UpdatePlayerStats(player *Player, equip bool) {
	// Accessories might affect various stats
	// Implement logic accordingly
}

func getItemNameWithSharpness(item Item) string {
	// Helper function to format sharpness
	formatSharpness := func(sharpness int) string {
		if sharpness > 0 {
			return fmt.Sprintf("+%d", sharpness)
		} else if sharpness < 0 {
			return fmt.Sprintf("%d", sharpness) // Negative sign is included
		}
		return ""
	}

	switch item := item.(type) {
	case *Weapon:
		return fmt.Sprintf("%s%s", item.GetName(), formatSharpness(item.Sharpness))
	case *Armor:
		return fmt.Sprintf("%s%s", item.GetName(), formatSharpness(item.Sharpness))
	case *Money: // Money type case added
		return fmt.Sprintf("%d円", item.Amount) // Format the amount as yen
	case *Arrow: // Arrow type case added
		return fmt.Sprintf("%d本の%s", item.ShotCount, item.GetName()) // Format the arrow item with shot count
	default:
		return item.GetName()
	}
}

func (g *Game) executeItemSwap() {
	playerX, playerY := g.state.Player.X, g.state.Player.Y

	for i, item := range g.state.Items {
		itemX, itemY := item.GetPosition()
		if itemX == playerX && itemY == playerY {
			selectedInventoryItem := g.state.Player.Inventory[g.selectedItemIndex]

			itemName := getItemNameWithSharpness(item)
			selecteditemName := getItemNameWithSharpness(selectedInventoryItem)

			action := Action{
				Duration: 0.5,
				Message:  fmt.Sprintf("%sと%sを交換しました", itemName, selecteditemName),
				Execute: func(g *Game) {
					// Set the position of the selected inventory item to the player's position
					selectedInventoryItem.SetPosition(playerX, playerY)
					// Swap the positions of the items
					g.state.Items[i] = selectedInventoryItem
					g.state.Player.Inventory[g.selectedItemIndex] = item
					g.selectedItemIndex = 0
					g.isActioned = true
				},
			}

			g.ActionQueue.Enqueue(action)
			break
		}
	}
}

func (g *Game) PickUpItem(item Item, i int) {
	if money, isMoney := item.(*Money); isMoney {
		// itemがMoney型である場合、プレイヤーの所持金を増加させる
		g.state.Player.Cash += money.Amount
	} else {
		// それ以外の場合、アイテムをプレイヤーのインベントリに追加
		g.state.Player.Inventory = append(g.state.Player.Inventory, item)
	}
	// アイテムをGameState.Itemsから削除
	g.state.Items = append(g.state.Items[:i], g.state.Items[i+1:]...)
}

func (g *Game) PickupItem() {
	playerX, playerY := g.state.Player.X, g.state.Player.Y // プレイヤーの座標を取得

	if !g.xPressed {
		for i, item := range g.state.Items { // GameStateの全てのアイテムに対してループ
			itemX, itemY := item.GetPosition()        // アイテムの座標を取得
			if itemX == playerX && itemY == playerY { // アイテムの座標とプレイヤーの座標が一致するかチェック

				itemName := getItemNameWithSharpness(item)
				// プレイヤーのインベントリサイズをチェック
				if len(g.state.Player.Inventory) < 20 {
					action := Action{
						Duration: 0.3,
						Message:  fmt.Sprintf("%sを拾った", itemName),
						Execute: func(g *Game) {
							g.PickUpItem(item, i)
						},
					}

					g.Enqueue(action)
					break // 一致するアイテムが見つかったらループを終了
				} else {
					action := Action{
						Duration: 0.5,
						Message:  fmt.Sprintf("持ち物がいっぱいで%sを拾えなかった", itemName),
						Execute: func(g *Game) {

						},
					}
					g.Enqueue(action)
				}

			}
		}
	} else {
		for _, item := range g.state.Items { // GameStateの全てのアイテムに対してループ
			itemX, itemY := item.GetPosition()        // アイテムの座標を取得
			if itemX == playerX && itemY == playerY { // アイテムの座標とプレイヤーの座標が一致するかチェック

				itemName := getItemNameWithSharpness(item)
				action := Action{
					Duration: 0.5,
					Message:  fmt.Sprintf("%sに乗った", itemName),
					Execute: func(g *Game) {
					},
				}
				g.Enqueue(action)
				break // 一致するアイテムが見つかったらループを終了

			}
		}

	}
}

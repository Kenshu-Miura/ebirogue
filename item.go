package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
	"math/rand"
)

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
	default:
		return item.GetName()
	}
}

func (g *Game) hitEnemyWithItem() {
	if potion, ok := g.ThrownItem.Item.(*Potion); ok {
		action := Action{
			Duration: 0.5, // Assuming a duration of 0.5 seconds for this action
			Message:  fmt.Sprintf("%sのHPが%d回復した。", g.state.Enemies[g.TargetEnemyIndex].Name, potion.Health),
			Execute: func(*Game) {
				g.state.Enemies[g.TargetEnemyIndex].Health += potion.Health
				if g.state.Enemies[g.TargetEnemyIndex].Health > g.state.Enemies[g.TargetEnemyIndex].MaxHealth {
					g.state.Enemies[g.TargetEnemyIndex].Health = g.state.Enemies[g.TargetEnemyIndex].MaxHealth
				}
				g.isActioned = true
				g.TargetEnemy = nil // Reset the target enemy after processing
			},
		}
		g.Enqueue(action)
	} else {
		damage := rand.Intn(3) + 1
		action := Action{
			Duration: 0.5, // Assuming a duration of 0.5 seconds for this action
			Message:  fmt.Sprintf("%sに%dのダメージを与えた。", g.state.Enemies[g.TargetEnemyIndex].Name, damage),
			Execute: func(*Game) {
				g.state.Enemies[g.TargetEnemyIndex].Health -= damage
				if g.state.Enemies[g.TargetEnemyIndex].Health < 0 {
					g.state.Enemies[g.TargetEnemyIndex].Health = 0
				}
				if g.state.Enemies[g.TargetEnemyIndex].Health <= 0 {
					// 敵のHealthが0以下の場合、敵を配列から削除
					defeatAction := Action{
						Duration: 0.5,
						Message:  fmt.Sprintf("%sを倒した。", g.state.Enemies[g.TargetEnemyIndex].Name),
						Execute:  func(g *Game) {},
					}
					g.Enqueue(defeatAction)

					g.state.Enemies = append(g.state.Enemies[:g.TargetEnemyIndex], g.state.Enemies[g.TargetEnemyIndex+1:]...)

					// 敵の経験値をプレイヤーの所持経験値に加える
					g.state.Player.ExperiencePoints += g.TargetEnemy.ExperiencePoints

					g.state.Player.checkLevelUp() // レベルアップをチェック

					g.TargetEnemy = nil // Reset the target enemy after processing
				}
				g.isActioned = true
			},
		}
		g.Enqueue(action)
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
					g.isActioned = true
				},
			}

			g.ActionQueue.Enqueue(action)
			break
		}
	}
}

func (g *Game) PickUpItem(item Item, i int) {
	g.state.Player.Inventory = append(g.state.Player.Inventory, item) // アイテムをプレイヤーのインベントリに追加
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

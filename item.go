package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
	"math/rand"
)

func (g *Game) updateItemVisibility() {
	// プレイヤーの座標を取得
	playerX, playerY := g.state.Player.X, g.state.Player.Y

	// 全てのアイテムに対してループを実行
	for _, item := range g.state.Items {
		// アイテムの座標を取得
		itemX, itemY := item.GetPosition()

		// プレイヤーとアイテムが同じ部屋にあるかどうかを確認
		if isSameRoom(playerX, playerY, itemX, itemY, g.rooms) {
			// 同じ部屋にある場合、アイテムのShowOnMiniMapフィールドをtrueに設定
			item.SetShowOnMiniMap(true)
		}
	}
}

func (g *Game) UpdateThrownItem() {
	if g.ThrownItem.Image != nil {
		g.ThrownItem.X += g.ThrownItem.DX
		g.ThrownItem.Y += g.ThrownItem.DY
		// Check if the item has reached its destination
		if (g.ThrownItem.DX >= 0 && g.ThrownItem.X*tileSize >= g.ThrownItemDestination.X*tileSize) ||
			(g.ThrownItem.DX < 0 && g.ThrownItem.X*tileSize <= g.ThrownItemDestination.X*tileSize) {
			if (g.ThrownItem.DY >= 0 && g.ThrownItem.Y*tileSize >= g.ThrownItemDestination.Y*tileSize) ||
				(g.ThrownItem.DY < 0 && g.ThrownItem.Y*tileSize <= g.ThrownItemDestination.Y*tileSize) {

				itemExists := false
				for _, item := range g.state.Items {
					x, y := item.GetPosition()
					if x == g.ThrownItem.X && y == g.ThrownItem.Y {
						itemExists = true
						break
					}
				}

				if itemExists {
					// Check surrounding tiles for placement
					directions := []Coordinate{{-1, -1}, {-1, 0}, {-1, 1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}}
					placed := false
					for _, dir := range directions {
						newX := g.ThrownItem.X + dir.X
						newY := g.ThrownItem.Y + dir.Y
						// Check map boundaries and tile type
						if newX >= 0 && newY >= 0 && newX < len(g.state.Map[0]) && newY < len(g.state.Map) &&
							g.state.Map[newY][newX].Type != "wall" {
							emptyTile := true
							for _, item := range g.state.Items {
								x, y := item.GetPosition()
								if x == newX && y == newY {
									emptyTile = false
									break
								}
							}
							if emptyTile {
								g.state.Items = append(g.state.Items, g.ThrownItem.Item)
								g.ThrownItem.Item.SetPosition(newX, newY)
								placed = true
								break
							}
						}
					}
					if !placed {
						// If no empty tile, do not place the item
					}
				} else {
					// g.ThrownItemがCane型かつTypeが"Effect"の場合、g.state.Itemsにg.ThrownItem.Itemを追加する処理を行わない
					if caneItem, ok := g.ThrownItem.Item.(*Cane); ok && caneItem.BaseItem.Type == "Effect" {
						// Do nothing
					} else {
						// Place the item normally if no item exists at the destination
						g.state.Items = append(g.state.Items, g.ThrownItem.Item)
					}
				}
				g.miniMapDirty = true

				// Reset the thrown item and its destination
				g.dPressed = false
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
	var itemName string
	message := ""
	identified := true

	// アイテムが識別されているかどうかをチェック
	if identifiableItem, ok := item.(Identifiable); ok {
		itemName = getItemNameWithSharpness(item) // 識別されている場合、またはIdentifiableインターフェースを実装している場合
		// 識別されていないアイテムの場合は識別されていないアイテム名を取得
		if !identifiableItem.IsIdentified() {
			itemName = identifiableItem.GetName()
			identified = false
		}
	} else {
		itemName = item.GetName() // Identifiableインターフェースを実装していない場合
	}

	// メッセージの設定
	if caneItem, ok := item.(*Cane); ok && caneItem.BaseItem.Type == "Effect" {
		message = fmt.Sprintf("%sを使った", itemName) // Cane型でかつTypeが"Effect"の場合
	} else if g.dPressed {
		message = fmt.Sprintf("%sを撃った", itemName) // Dキーが押された場合
	} else {
		message = fmt.Sprintf("%sを投げた", itemName) // デフォルトのメッセージ
	}
	action := Action{
		Duration: 0.5,
		Message:  message,
		ItemName: itemName,
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
					//log.Printf("Cane item: %+v", item)
					//log.Printf("Thrown item: %+v", g.ThrownItem)
					// アイテムがCane型であり、BaseItem.Typeが"Effect"であるかチェック
					if caneItem, ok := item.(*Cane); ok && caneItem.BaseItem.Type == "Effect" {
						// 条件に合致した場合のposition
						position := Coordinate{
							X: x + i*dx,
							Y: y + i*dy,
						}
						onWallHit(item, position, g.selectedItemIndex)
						return
					} else {
						// 通常のposition
						position := Coordinate{
							X: x + (i-1)*dx,
							Y: y + (i-1)*dy,
						}
						onWallHit(item, position, g.selectedItemIndex)
						return
					}
				}
				for index, enemy := range enemies {
					if enemy.X == targetX && enemy.Y == targetY {

						g.TargetEnemyIndex = index

						g.ThrownItemDestination = Coordinate{
							X: targetX,
							Y: targetY,
						}

						// Remove the item from the player's inventory
						// Check if the item is of type Arrow and whether the D key was pressed
						if _, ok := item.(*Arrow); ok && g.dPressed {
							// If it's an arrow and D key was pressed, only remove it from inventory if ShotCount is 0
							for i, inventoryItem := range g.state.Player.Inventory {
								if arrow, ok := inventoryItem.(*Arrow); ok && arrow.ShotCount == 0 {
									// Remove the Arrow item with ShotCount of 0 from the inventory
									g.state.Player.Inventory = append(g.state.Player.Inventory[:i], g.state.Player.Inventory[i+1:]...)

									// Adjust the index for the next iteration if an item was removed
									i--
								}
							}
						} else {
							// itemがCane型かつTypeが"Effect"の場合、プレイヤーのインベントリから削除しない
							if caneItem, ok := item.(*Cane); ok && caneItem.BaseItem.Type == "Effect" {
								// Do nothing
							} else {
								if !g.GroundItemActioned {
									// If it's not an arrow or D key wasn't pressed, remove the item from the player's inventory
									g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
								} else {
									// If it's a ground item, remove the item from the map
									g.state.Items = append(g.state.Items[:g.selectedGroundItemIndex], g.state.Items[g.selectedGroundItemIndex+1:]...)
									g.GroundItemActioned = false
									g.selectedGroundActionIndex = 0
								}
							}
						}

						onTargetHit(&enemy, item, index)

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
		IsIdentified: identified,
	}
	g.Enqueue(action)
}

func (g *Game) onWallHit(item Item, position Coordinate, itemIndex int) {
	// Set the position of the item to the position before hitting the wall
	item.SetPosition(position.X, position.Y)

	// Update the ThrownItemDestination to the position before hitting the wall
	g.ThrownItemDestination = position

	// Remove the item from the player's inventory
	// Check if the item is of type Arrow and whether the D key was pressed
	if _, ok := item.(*Arrow); ok && g.dPressed {
		// If it's an arrow and D key was pressed, only remove it from inventory if ShotCount is 0
		for i, inventoryItem := range g.state.Player.Inventory {
			if arrow, ok := inventoryItem.(*Arrow); ok && arrow.ShotCount == 0 {
				// Remove the Arrow item with ShotCount of 0 from the inventory
				g.state.Player.Inventory = append(g.state.Player.Inventory[:i], g.state.Player.Inventory[i+1:]...)

				// Adjust the index for the next iteration if an item was removed
				i--
			}
		}
	} else {
		// itemがCane型かつTypeが"Effect"の場合、プレイヤーのインベントリから削除しない
		if caneItem, ok := item.(*Cane); ok && caneItem.BaseItem.Type == "Effect" {
			// Do nothing
		} else {
			// If it's an item that was on the ground, remove it from the ground
			if g.GroundItemActioned {
				g.state.Items = append(g.state.Items[:g.selectedGroundItemIndex], g.state.Items[g.selectedGroundItemIndex+1:]...)
				g.GroundItemActioned = false
				g.selectedGroundActionIndex = 0
			} else {
				// If it's an item that was in the player's inventory, remove it from the inventory
				g.state.Player.Inventory = append(g.state.Player.Inventory[:itemIndex], g.state.Player.Inventory[itemIndex+1:]...)
			}
		}
	}

	// Update the UI flags
	g.showItemActions = false
	g.showInventory = false
	g.isActioned = true
	g.selectedItemIndex = 0
	g.selectedActionIndex = 0
}

func (g *Game) onTargetHit(target Character, item Item, index int) {
	// Check if the item is of type Cane
	if cane, ok := item.(*Cane); ok {
		cane.Use(g)
	} else if potion, ok := item.(*Potion); ok {
		action := Action{
			Duration: 0.5, // Assuming a duration of 0.5 seconds for this action
			Message:  fmt.Sprintf("%sのHPが%d回復した。", target.GetName(), potion.Health),
			Execute: func(g *Game) {
				// Type assertion to check if target is of type *Player or *Enemy
				if _, ok := target.(*Player); ok {
					// If target is of type *Player
					g.state.Player.Health += potion.Health
					if g.state.Player.Health > g.state.Player.GetMaxHealth() {
						g.state.Player.Health = g.state.Player.GetMaxHealth()
					}
				} else if _, ok := target.(*Enemy); ok && index >= 0 && index < len(g.state.Enemies) {
					// If target is of type *Enemy
					g.state.Enemies[index].Health += potion.Health
					if g.state.Enemies[index].Health > g.state.Enemies[index].GetMaxHealth() {
						g.state.Enemies[index].Health = g.state.Enemies[index].GetMaxHealth()
					}
				}
				g.isActioned = true
				// Reset the target character after processing
			},
		}
		g.Enqueue(action)
	} else {
		damage := 0
		if g.dPressed {
			// Base damage calculation
			damage = g.state.Player.AttackPower + g.state.Player.Power + g.state.Player.Level - target.GetDefensePower() + rand.Intn(3) - 1

			// Check if item is of type Arrow
			if arrow, ok := item.(*Arrow); ok {
				// Add the AttackPower of the Arrow to the damage
				damage += arrow.AttackPower
			}
		} else {
			damage = rand.Intn(3) + 1
		}
		action := Action{
			Duration: 0.5, // Assuming a duration of 0.5 seconds for this action
			Message:  fmt.Sprintf("%sに%dのダメージを与えた。", target.GetName(), damage),
			Execute: func(g *Game) {
				// Type assertion to check if target is of type *Player or *Enemy
				if _, ok := target.(*Player); ok {
					// If target is of type *Player
					g.state.Player.Health -= damage
					if g.state.Player.Health < 0 {
						g.state.Player.Health = 0
					}
				} else if enemy, ok := target.(*Enemy); ok && index >= 0 && index < len(g.state.Enemies) {
					// If target is of type *Enemy
					g.state.Enemies[index].Health -= damage
					if g.state.Enemies[index].Health < 0 {
						g.state.Enemies[index].Health = 0
					}
					if g.state.Enemies[index].Health <= 0 {
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

func getEquippedIndex(equippedItems []Item, item Equipable) int {
	for index, equippedItem := range equippedItems {
		if equippedItem == item {
			return index
		}
	}
	return -1
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

	// Check if the item implements the Identifiable interface
	if identifiable, ok := item.(Identifiable); ok {
		// Check if the item is identified
		if !identifiable.IsIdentified() {
			// For unidentified items, return the base name without sharpness
			return identifiable.GetName()
		} else {
			// Process identified items
			switch item := item.(type) {
			case *Weapon:
				return fmt.Sprintf("%s%s", item.GetName(), formatSharpness(item.Sharpness))
			case *Armor:
				return fmt.Sprintf("%s%s", item.GetName(), formatSharpness(item.Sharpness))
			case *Money:
				return fmt.Sprintf("%d円", item.Amount) // Format the amount as yen
			case *Arrow:
				return fmt.Sprintf("%d本の%s", item.ShotCount, item.GetName()) // Format the arrow item with shot count
			case *Cane:
				return fmt.Sprintf("%s[%d]", item.GetName(), item.Uses) // Format the cane item with uses count
			default:
				return item.GetName()
			}
		}
	} else {
		// If the item does not implement the Identifiable interface, use the default name
		return item.GetName()
	}
}

func (g *Game) executeItemSwap() {
	playerX, playerY := g.state.Player.X, g.state.Player.Y

	for i, item := range g.state.Items {
		itemX, itemY := item.GetPosition()
		if itemX == playerX && itemY == playerY {
			selectedInventoryItem := g.state.Player.Inventory[g.selectedItemIndex]
			//itemName := getItemNameWithSharpness(item) // You might want to adjust this if you have a different way to get the item's name.

			isCursedEquipped := false

			// アイテムが識別されているかどうかをチェック
			identified := true
			var selectedItemName string
			if identifiableItem, ok := selectedInventoryItem.(Identifiable); ok {
				identified = identifiableItem.IsIdentified()
				// 識別されていない場合は識別されていないアイテム名を取得
				if !identified {
					selectedItemName = identifiableItem.GetName()
				}
			}

			// 識別されている場合、またはIdentifiableインターフェースを実装していない場合は、Sharpnessを含む名前を使用
			if identified {
				selectedItemName = getItemNameWithSharpness(selectedInventoryItem)
			}

			// Check if the selected inventory item is Equipable and cursed
			if equipableItem, ok := selectedInventoryItem.(Equipable); ok {
				for _, equippedItem := range g.state.Player.EquippedItems {
					if equippedItem == equipableItem {
						switch v := equipableItem.(type) {
						case *Weapon:
							if v.Cursed {
								isCursedEquipped = true
							}
						case *Armor:
							if v.Cursed {
								isCursedEquipped = true
							}
						}
						break
					}
				}
			}

			if isCursedEquipped {
				// If the selected inventory item is cursed and equipped, do not swap and enqueue an action with a message that it cannot be swapped
				action := Action{
					Duration: 0.4,
					Message:  fmt.Sprintf("%sは呪われていて交換できない", selectedItemName),
					Execute: func(g *Game) {
						// Any additional logic if needed
						g.selectedItemIndex = 0
					},
				}
				g.ActionQueue.Enqueue(action)
			} else {
				// If the item is not cursed or not equipped, proceed with the swap
				action := Action{
					Duration: 0.5,
					Message:  fmt.Sprintf("足元のアイテムと%sを交換しました", selectedItemName),
					ItemName: selectedItemName,
					Execute: func(g *Game) {
						// Check if the item is equipped and unequip if necessary
						if equipableItem, ok := selectedInventoryItem.(Equipable); ok {
							for i, equippedItem := range g.state.Player.EquippedItems {
								if equippedItem == equipableItem {
									g.state.Player.EquippedItems[i] = nil
									equipableItem.UpdatePlayerStats(&g.state.Player, false) // Update player's stats when unequipping
									break
								}
							}
						}
						// Swap the positions of the items
						selectedInventoryItem.SetPosition(playerX, playerY)
						g.state.Items[i] = selectedInventoryItem
						g.state.Player.Inventory[g.selectedItemIndex] = item
						g.selectedItemIndex = 0
						g.isActioned = true
					},
					IsIdentified: identified,
				}
				g.ActionQueue.Enqueue(action)
			}
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

				// アイテムが識別されているかどうかをチェック
				identified := true
				var itemName string
				if identifiableItem, ok := item.(Identifiable); ok {
					identified = identifiableItem.IsIdentified()
					// 識別されていない場合は識別されていないアイテム名を取得
					if !identified {
						itemName = identifiableItem.GetName()
					}
				}

				// 識別されている場合、またはIdentifiableインターフェースを実装していない場合は、Sharpnessを含む名前を使用
				if identified {
					itemName = getItemNameWithSharpness(item)
				}

				// プレイヤーのインベントリサイズをチェック
				if len(g.state.Player.Inventory) < 20 {
					message := fmt.Sprintf("%sを拾った", itemName) // メッセージ全体を作成
					action := Action{
						Duration:     0.3,
						Message:      message,
						ItemName:     itemName,
						Execute:      func(g *Game) { g.PickUpItem(item, i) },
						IsIdentified: identified,
					}
					g.Enqueue(action)
					break // 一致するアイテムが見つかったらループを終了
				} else {
					// インベントリが満杯の場合のメッセージ
					message := fmt.Sprintf("持ち物がいっぱいで%sを拾えなかった", itemName)
					action := Action{
						Duration:     0.5,
						Message:      message,
						ItemName:     itemName,
						Execute:      func(g *Game) {},
						IsIdentified: identified,
					}
					g.Enqueue(action)
				}
			}
		}
	} else {
		for _, item := range g.state.Items { // GameStateの全てのアイテムに対してループ
			itemX, itemY := item.GetPosition()        // アイテムの座標を取得
			if itemX == playerX && itemY == playerY { // アイテムの座標とプレイヤーの座標が一致するかチェック

				// アイテムが識別されているかどうかをチェック
				identified := true
				var itemName string
				if identifiableItem, ok := item.(Identifiable); ok {
					identified = identifiableItem.IsIdentified()
					// 識別されていない場合は識別されていないアイテム名を取得
					if !identified {
						itemName = identifiableItem.GetName()
					}
				}

				// 識別されている場合、またはIdentifiableインターフェースを実装していない場合は、Sharpnessを含む名前を使用
				if identified {
					itemName = getItemNameWithSharpness(item)
				}

				action := Action{
					Duration: 0.5,
					Message:  fmt.Sprintf("%sに乗った", itemName),
					ItemName: itemName,
					Execute: func(g *Game) {
					},
					IsIdentified: identified,
				}
				g.Enqueue(action)
				break // 一致するアイテムが見つかったらループを終了

			}
		}

	}
}

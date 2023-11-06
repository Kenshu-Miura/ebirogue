package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
	"math/rand"
)

func (g *Game) executeGroundItemAction() {
	playerX, playerY := g.state.Player.X, g.state.Player.Y // プレイヤーの座標を取得

	if g.selectedGroundItemIndex == 0 { // Assuming index 0 corresponds to '拾う'
		for i, item := range g.state.Items { // GameStateの全てのアイテムに対してループ
			itemX, itemY := item.GetPosition()        // アイテムの座標を取得
			if itemX == playerX && itemY == playerY { // アイテムの座標とプレイヤーの座標が一致するかチェック
				var itemName string
				if weaponItem, ok := g.state.Items[i].(*Weapon); ok {
					sharpnessSign := ""
					if weaponItem.Sharpness > 0 {
						sharpnessSign = fmt.Sprintf("+%d", weaponItem.Sharpness)
					} else if weaponItem.Sharpness < 0 {
						sharpnessSign = fmt.Sprintf("%d", weaponItem.Sharpness) // Negative sign is included
					}
					itemName = fmt.Sprintf("%s%s", weaponItem.GetName(), sharpnessSign)
				} else {
					itemName = g.state.Items[i].GetName()
				}
				// プレイヤーのインベントリサイズをチェック
				if len(g.state.Player.Inventory) < 20 {
					action := Action{
						Duration: 0.3,
						Message:  fmt.Sprintf("%sを拾った", itemName),
						Execute: func(g *Game) {
							g.PickUpItem(item, i)
							g.isActioned = true
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
		g.ShowGroundItem = false
		g.GroundItemActioned = false
		g.selectedGroundItemIndex = 0
	}

	if g.selectedGroundItemIndex == 1 { // Assuming index 1 corresponds to '交換'
		g.ShowGroundItem = false
		g.showInventory = true
	}
}

func (g *Game) executeAction() {

	if g.selectedActionIndex == 0 { // Assuming index 0 corresponds to '使う' or '装備'
		item := g.state.Player.Inventory[g.selectedItemIndex]
		if foodItem, ok := item.(*Food); ok {
			foodItem.Use(g)
		} else if potionItem, ok := item.(*Potion); ok {
			potionItem.Use(g)
		} else if cardItem, ok := item.(*Card); ok {
			cardItem.Use(g)
		} else if moneyItem, ok := item.(*Money); ok {
			moneyItem.Use(g)
		} else if trapItem, ok := item.(*Trap); ok {
			trapItem.Use(g)
		} else if equipableItem, ok := item.(Equipable); ok { // Check if item is of Equipable type
			var message string
			itemName := getItemNameWithSharpness(equipableItem) // Assume this function can handle Equipable type

			// Find an empty slot or use the last slot
			equipIndex := -1
			for i := 0; i < 4; i++ { // Search for an empty slot in EquippedItems[0] to EquippedItems[3]
				if g.state.Player.EquippedItems[i] == nil {
					equipIndex = i
					break
				}
			}
			if equipIndex == -1 { // If no empty slot found, use the last slot
				equipIndex = 4
			}

			// Check if the item is already equipped
			alreadyEquipped := false
			for i := 0; i < 5; i++ {
				if g.state.Player.EquippedItems[i] == equipableItem {
					alreadyEquipped = true
					equipIndex = i // Update the equipIndex to the slot where the item is already equipped
					break
				}
			}

			if alreadyEquipped {
				// Unequip the item
				message = fmt.Sprintf("%sをはずした。", itemName)
				equipableItem.UpdatePlayerStats(&g.state.Player, false) // Update player's stats when unequipping
				g.state.Player.EquippedItems[equipIndex] = nil          // Remove item from equipped items
			} else {
				// Equip the item
				message = fmt.Sprintf("%sを装備した。", itemName)
				equipableItem.UpdatePlayerStats(&g.state.Player, true)   // Update player's stats when equipping
				equipableItem.SetIdentified(true)                        // Set the item as identified when equipping
				g.state.Player.EquippedItems[equipIndex] = equipableItem // Equip item
			}

			action := Action{
				Duration: 0.5,
				Message:  message,
				Execute: func(g *Game) {
					// The equipped/unequipped item is already set above
				},
			}
			g.Enqueue(action)
		}

		g.showItemActions = false
		g.showInventory = false
		g.isActioned = true
		g.selectedItemIndex = 0
	}

	if g.selectedActionIndex == 1 { // Assuming index 1 corresponds to '投げる'
		item := g.state.Player.Inventory[g.selectedItemIndex]
		throwRange := 10
		character := &g.state.Player
		mapState := g.state.Map
		enemies := g.state.Enemies

		onWallHit := func(item Item, position Coordinate, itemIndex int) {
			g.onWallHit(item, position, itemIndex)
		}

		onTargetHit := func(target Character, item Item, index int) {
			g.onTargetHit(target, item, index)
		}

		// Type assertion to check if item is Equipable
		if equipableItem, ok := item.(Equipable); ok {
			// Check if the item is equipped
			index := getEquippedIndex(g.state.Player.EquippedItems[:], equipableItem)
			if index != -1 {
				// If it is equipped, remove it from the equipped items list
				g.state.Player.EquippedItems[index] = nil
			}
		}

		g.ThrowItem(item, throwRange, character, mapState, enemies, onWallHit, onTargetHit)
	}

	if g.selectedActionIndex == 2 { // Assuming index 2 corresponds to '置く'
		itemExistsAtPlayerPos := false
		playerX, playerY := g.state.Player.X, g.state.Player.Y
		for _, item := range g.state.Items {
			itemX, itemY := item.GetPosition()
			if itemX == playerX && itemY == playerY {
				itemExistsAtPlayerPos = true
				break
			}
		}
		itemName := getItemNameWithSharpness(g.state.Player.Inventory[g.selectedItemIndex])
		if !itemExistsAtPlayerPos {
			action := Action{
				Duration: 0.4, // Assuming a duration of 0.5 seconds for this action
				Message:  fmt.Sprintf("%sを置いた", itemName),
				Execute: func(g *Game) {
					selectedItem := g.state.Player.Inventory[g.selectedItemIndex]

					// Check if the item is equipped and unequip if necessary
					if equipableItem, ok := selectedItem.(Equipable); ok {
						for i, equippedItem := range g.state.Player.EquippedItems {
							if equippedItem == equipableItem {
								g.state.Player.EquippedItems[i] = nil
								equipableItem.UpdatePlayerStats(&g.state.Player, false) // Update player's stats when unequipping
								break
							}
						}
					}

					// Remove the item from inventory
					g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
					// Add the item to the world at the player's current position
					selectedItem.SetPosition(g.state.Player.X, g.state.Player.Y)
					newItem := selectedItem
					g.state.Items = append(g.state.Items, newItem)

					g.selectedItemIndex = 0
					g.selectedActionIndex = 0
					g.showItemActions = false
					g.showInventory = false
					g.isActioned = true
				},
			}
			g.Enqueue(action)
		} else {
			action := Action{
				Duration: 0.4,
				Message:  fmt.Sprintf("ここには%sを置けない", itemName),
				Execute: func(g *Game) {
					g.selectedItemIndex = 0
					g.selectedActionIndex = 0
					g.showItemActions = false
					g.showInventory = false
					g.isActioned = true
				},
			}
			g.Enqueue(action)
		}
	}

	if g.selectedActionIndex == 3 { // Assuming 0-based index and "説明" is at index 3
		selectedItem := g.state.Player.Inventory[g.selectedItemIndex]
		g.itemdescriptionText = selectedItem.GetDescription()
		g.showItemDescription = true
	}

}

func (g *Game) Enqueue(action Action) {
	g.isCombatActive = true
	g.ActionQueue.Queue = append(g.ActionQueue.Queue, action)
}

func (g *Game) processAction(action Action) {
	// 実際のアクションの実行ロジックはアクションオブジェクトのExecuteメソッドに委譲
	action.Execute(g)
	g.ActionDurationCounter = action.Duration // record the duration of the next action
}

// Enqueue adds a new attack to the attack queue
func (aq *ActionQueue) Enqueue(action Action) {
	aq.Queue = append(aq.Queue, action)
}

func (g *Game) AttackFromEnemy(enemyIndex int) {
	enemy := &g.state.Enemies[enemyIndex]

	if trap := g.state.Player.SetTrap; trap != nil && trap.GetName() == "炸裂装甲のカード" {
		// If the player has set a trap and it is the '炸裂装甲のカード', the trap will be triggered
		action := Action{
			Duration: 0.5,
			Message:  fmt.Sprintf("%sの攻撃。", enemy.Name),
			Execute:  func(g *Game) {},
		}
		g.Enqueue(action)

		action = Action{
			Duration: 0.5,
			Message:  fmt.Sprintf("罠カード、%sが発動した。", trap.GetName()),
			Execute:  func(g *Game) {},
		}
		g.Enqueue(action)

		defeatAction := Action{
			Duration: 0.5,
			Message:  fmt.Sprintf("%sを倒した。", enemy.Name),
			Execute: func(g *Game) {
				g.state.Enemies = append(g.state.Enemies[:enemyIndex], g.state.Enemies[enemyIndex+1:]...)

				// 敵の経験値をプレイヤーの所持経験値に加える
				g.state.Player.ExperiencePoints += enemy.ExperiencePoints

				g.state.Player.checkLevelUp() // レベルアップをチェック

				// トラップをリセットする (オプショナル)
				g.state.Player.SetTrap = nil
			},
		}
		g.Enqueue(defeatAction)
		return
	}

	// Generate a random float number between 0 and 1 to compare with specialAttackProbability
	randomValue := rand.Float64()

	// Check if the enemy will perform a special attack
	if enemy.SpecialAttack != nil && randomValue <= enemy.SpecialAttackProbability {
		// Perform the special attack
		enemy.SpecialAttack(enemy, g)
	} else {
		// Perform the normal attack
		netDamage := enemy.AttackPower - g.state.Player.DefensePower + rand.Intn(3) - 1
		if netDamage < 0 { // Ensure damage does not go below 0
			netDamage = 0
		}

		dx, dy := g.state.Player.X-enemy.X, g.state.Player.Y-enemy.Y // プレイヤーと敵の位置の差を計算

		action := Action{
			Duration: 0.5,
			Message:  fmt.Sprintf("%sから%dダメージを受けた", enemy.Name, netDamage),
			Execute: func(g *Game) {
				enemy.AttackTimer = 0.5                            // ここでAttackTimerを設定することで、敵の攻撃アニメーションが実行される
				enemy.AttackDirection = determineDirection(dx, dy) // 敵の攻撃方向を計算
				g.state.Player.Health -= netDamage
				if g.state.Player.Health < 0 {
					g.state.Player.Health = 0 // Ensure health does not go below 0
				}
			},
		}

		g.Enqueue(action)
	}
}

func (g *Game) CheckForEnemies(x, y int) bool {

	g.isFrontEnemy = false

	for i, enemy := range g.state.Enemies {
		if enemy.X == g.state.Player.X+x && enemy.Y == g.state.Player.Y+y {
			g.isFrontEnemy = true
			// Player's AttackPower is considered while dealing damage
			netDamage := g.state.Player.AttackPower + g.state.Player.Power + g.state.Player.Level - enemy.DefensePower + rand.Intn(3) - 1
			if netDamage < 0 { // Ensure damage does not go below 0
				netDamage = 0
			}

			dx, dy := enemy.X-g.state.Player.X, enemy.Y-g.state.Player.Y

			// Determine the direction based on the change in position
			switch {
			case dx == 1 && dy == 0:
				g.state.Player.Direction = Right
			case dx == -1 && dy == 0:
				g.state.Player.Direction = Left
			case dx == 0 && dy == 1:
				g.state.Player.Direction = Down
			case dx == 0 && dy == -1:
				g.state.Player.Direction = Up
			case dx == 1 && dy == 1:
				g.state.Player.Direction = DownRight
			case dx == -1 && dy == 1:
				g.state.Player.Direction = DownLeft
			case dx == 1 && dy == -1:
				g.state.Player.Direction = UpRight
			case dx == -1 && dy == -1:
				g.state.Player.Direction = UpLeft
			}

			g.attackTimer = 0.5 // set timer for 0.5 seconds
			action := Action{
				Duration: 0.5,
				Message:  fmt.Sprintf("%sに%dダメージを与えた。", g.state.Enemies[i].Name, netDamage),
				Execute: func(g *Game) {

					enemyIndex := i // ここでi変数の値を明示的にキャプチャ
					g.state.Enemies[enemyIndex].Health -= netDamage

					if g.state.Enemies[enemyIndex].Health <= 0 {
						// 敵のHealthが0以下の場合、敵を配列から削除
						defeatAction := Action{
							Duration: 0.5,
							Message:  fmt.Sprintf("%sを倒した。", g.state.Enemies[enemyIndex].Name),
							Execute:  func(g *Game) {},
						}
						g.Enqueue(defeatAction)

						g.state.Enemies = append(g.state.Enemies[:enemyIndex], g.state.Enemies[enemyIndex+1:]...)

						// 敵の経験値をプレイヤーの所持経験値に加える
						g.state.Player.ExperiencePoints += enemy.ExperiencePoints

						g.state.Player.checkLevelUp() // レベルアップをチェック
					}
					g.isActioned = true

				},
			}

			g.Enqueue(action)

			return true
		}
	}
	if !g.isFrontEnemy {
		g.attackTimer = 0.5 // set timer for 0.5 seconds
		action := Action{
			Duration: 0.5,
			Message:  "",
			Execute: func(g *Game) {
				g.isActioned = true
			},
		}

		g.Enqueue(action)

		g.isFrontEnemy = false

		return true
	}
	return false
}

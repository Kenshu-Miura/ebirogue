//go:build !test

package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
	"math/rand"
)

func (g *Game) executeGroundItemAction() {
	playerX, playerY := g.state.Player.X, g.state.Player.Y // プレイヤーの座標を取得

	if g.selectedGroundActionIndex == 0 { // Assuming index 0 corresponds to '拾う'
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
					action := Action{
						Duration: 0.8,
						Message:  fmt.Sprintf("%sを拾った", itemName),
						ItemName: itemName,
						Execute: func(g *Game) {
							g.PickUpItem(item, i)
							g.isActioned = true
						},
						IsIdentified: identified,
						NonBlocking:  !g.IsEnemyAdjacent(),
					}
					g.Enqueue(action)
					break // 一致するアイテムが見つかったらループを終了
				} else {
					action := Action{
						Duration: 0.5,
						Message:  fmt.Sprintf("持ち物がいっぱいで%sを拾えなかった", itemName),
						ItemName: itemName,
						Execute: func(g *Game) {

						},
						IsIdentified: identified,
						NonBlocking:  !g.IsEnemyAdjacent(),
					}
					g.Enqueue(action)
				}
			}
		}
		g.ShowGroundItem = false
		g.GroundItemActioned = false
		g.selectedGroundActionIndex = 0
	}

	if g.selectedGroundActionIndex == 1 { // Assuming index 1 corresponds to '交換'
		g.ShowGroundItem = false
		g.showInventory = true
	}

	if g.selectedGroundActionIndex == 2 { // Assuming index 2 corresponds to '使う' or '装備'
		for i, item := range g.state.Items { // GameStateの全てのアイテムに対してループ
			itemX, itemY := item.GetPosition()        // アイテムの座標を取得
			if itemX == playerX && itemY == playerY { // アイテムの座標とプレイヤーの座標が一致するかチェック
				g.selectedGroundItemIndex = i
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
				} else if caneItem, ok := item.(*Cane); ok {

					if caneItem.Uses <= 0 {
						action := Action{
							Duration: 0.5,
							Message:  fmt.Sprintf("%sを使った。しかし何も起こらなかった。", caneItem.GetName()),
							Execute: func(g *Game) {
							},
						}
						g.Enqueue(action)

						g.ShowGroundItem = false
						g.GroundItemActioned = false
						g.selectedGroundActionIndex = 0
						g.isActioned = true
						return
					}

					//caneItemの複製を作成する。

					caneItemCopy := *caneItem

					//caneItemCopyのUsesを1減らす
					caneItem.Uses--

					//caneItemCopyのBaseItem.Typeを"Effect"にする
					caneItemCopy.BaseItem.Type = "Effect"

					throwRange := 30
					character := &g.state.Player
					mapState := g.state.Map
					enemies := g.state.Enemies

					onWallHit := func(item Item, position Coordinate, itemIndex int) {
						g.onWallHit(item, position, itemIndex)
					}

					onTargetHit := func(target Character, item Item, index int) {
						g.onTargetHit(target, item, index)
					}

					// Continue with the throwing logic if the item is not cursed and equipped
					g.ThrowItem(&caneItemCopy, throwRange, character, mapState, enemies, onWallHit, onTargetHit)

				} else if equipableItem, ok := item.(Equipable); ok { // Check if item is of Equipable type

					// インベントリのサイズを確認し、いっぱいの場合はアイテムを拾わない
					if len(g.state.Player.Inventory) >= 20 {
						action := Action{
							Duration: 0.5,
							Message:  fmt.Sprintf("持ち物がいっぱいで%sを拾えなかった", item.GetName()),
							ItemName: item.GetName(),
							Execute: func(g *Game) {

							},
						}
						g.Enqueue(action)
						break
					}

					var message string
					identified := false
					itemName := getItemNameWithSharpness(equipableItem) // Assume this function can handle Equipable type

					// equipableItemがAccessory型の場合はIdentifiedをtrueにしない
					if _, ok := equipableItem.(*Accessory); !ok {
						equipableItem.SetIdentified(true) // Set the item as identified when equipping
						identified = true
						itemName = getItemNameWithSharpness(equipableItem)
					}

					// Use new equipment system
					if equipMessage, err := g.state.Player.EquipItem(equipableItem); err != nil {
						message = fmt.Sprintf("%sを装備できない。", itemName)
					} else {
						message = equipMessage
					}
					g.PickUpItem(item, i)

					action := Action{
						Duration: 0.5,
						Message:  message,
						ItemName: itemName,
						Execute: func(g *Game) {
							// The equipped/unequipped item is already set above
						},
						IsIdentified: identified,
					}
					g.Enqueue(action)
					// Check if the item is cursed after equipping
					cursedMessage := ""
					switch v := equipableItem.(type) {
					case *Weapon:
						if v.Cursed {
							cursedMessage = fmt.Sprintf("%sは呪われていた。", itemName)
						}
					case *Armor:
						if v.Cursed {
							cursedMessage = fmt.Sprintf("%sは呪われていた。", itemName)
						}
					}

					// If the item is cursed, enqueue the cursed message
					if cursedMessage != "" {
						cursedAction := Action{
							Duration: 0.5,
							Message:  cursedMessage,
							ItemName: itemName,
							Execute: func(g *Game) {
								// This can be left empty if no additional behavior is needed other than displaying the message
							},
							IsIdentified: identified,
						}
						g.Enqueue(cursedAction)
					}
				}

				g.ShowGroundItem = false
				g.GroundItemActioned = false
				g.selectedGroundActionIndex = 0
				g.isActioned = true
			}
		}
	}

	if g.selectedGroundActionIndex == 3 { // Assuming index 3 corresponds to '投げる'
		for i, item := range g.state.Items { // GameStateの全てのアイテムに対してループ
			itemX, itemY := item.GetPosition()        // アイテムの座標を取得
			if itemX == playerX && itemY == playerY { // アイテムの座標とプレイヤーの座標が一致するかチェック
				g.selectedGroundItemIndex = i

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

				// Continue with the throwing logic if the item is not cursed and equipped
				g.ThrowItem(item, throwRange, character, mapState, enemies, onWallHit, onTargetHit)

				g.ShowGroundItem = false
				g.isActioned = true
			}
		}
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
		} else if caneItem, ok := item.(*Cane); ok {

			if caneItem.Uses <= 0 {
				action := Action{
					Duration: 0.5,
					Message:  fmt.Sprintf("%sを使った。しかし何も起こらなかった。", caneItem.GetName()),
					Execute: func(g *Game) {
					},
				}
				g.Enqueue(action)
				g.showItemActions = false
				g.showInventory = false
				g.isActioned = true
				g.selectedItemIndex = 0
				g.selectedActionIndex = 0
				return
			}

			//caneItemの複製を作成する。

			caneItemCopy := *caneItem

			//caneItemCopyのUsesを1減らす
			caneItem.Uses--

			//caneItemCopyのBaseItem.Typeを"Effect"にする
			caneItemCopy.BaseItem.Type = "Effect"

			throwRange := 30
			character := &g.state.Player
			mapState := g.state.Map
			enemies := g.state.Enemies

			onWallHit := func(item Item, position Coordinate, itemIndex int) {
				g.onWallHit(item, position, itemIndex)
			}

			onTargetHit := func(target Character, item Item, index int) {
				g.onTargetHit(target, item, index)
			}

			// Continue with the throwing logic if the item is not cursed and equipped
			g.ThrowItem(&caneItemCopy, throwRange, character, mapState, enemies, onWallHit, onTargetHit)

		} else if equipableItem, ok := item.(Equipable); ok { // Check if item is of Equipable type
			var message string
			identified := false
			itemName := getItemNameWithSharpness(equipableItem) // Assume this function can handle Equipable type

			// Check if the item is already equipped
			if g.state.Player.IsEquipped(equipableItem) {
				// Check if the equipped item is cursed
				isCursed := false
				switch v := equipableItem.(type) {
				case *Weapon:
					if v.Cursed {
						isCursed = true
					}
				case *Armor:
					if v.Cursed {
						isCursed = true
					}
				case *Accessory:
					if v.Cursed {
						isCursed = true
					}
				}

				if isCursed {
					// If the item is cursed, update the message and do not unequip
					message = fmt.Sprintf("%sをはずせない。", itemName)
				} else {
					// Unequip the item using new system
					if unequipMessage, err := g.state.Player.UnequipItem(equipableItem); err != nil {
						message = fmt.Sprintf("%sをはずせない。", itemName)
					} else {
						message = unequipMessage
					}
				}
			} else {
				// Equip the item using new system
				if _, ok := equipableItem.(*Accessory); !ok {
					equipableItem.SetIdentified(true) // Set the item as identified when equipping
					identified = true
				}
				itemName = getItemNameWithSharpness(equipableItem)
				
				if equipMessage, err := g.state.Player.EquipItem(equipableItem); err != nil {
					message = fmt.Sprintf("%sを装備できない。", itemName)
				} else {
					message = equipMessage
				}
			}

			action := Action{
				Duration: 0.5,
				Message:  message,
				ItemName: itemName,
				Execute: func(g *Game) {
					// The equipped/unequipped item is already set above
				},
				IsIdentified: identified,
			}
			g.Enqueue(action)
			// Check if the item is cursed after equipping
			cursedMessage := ""
			switch v := equipableItem.(type) {
			case *Weapon:
				if v.Cursed {
					cursedMessage = fmt.Sprintf("%sは呪われていた。", itemName)
				}
			case *Armor:
				if v.Cursed {
					cursedMessage = fmt.Sprintf("%sは呪われていた。", itemName)
				}
			}

			// If the item is cursed, enqueue the cursed message
			if cursedMessage != "" {
				cursedAction := Action{
					Duration: 0.5,
					Message:  cursedMessage,
					ItemName: itemName,
					Execute: func(g *Game) {
						// This can be left empty if no additional behavior is needed other than displaying the message
					},
					IsIdentified: identified,
				}
				g.Enqueue(cursedAction)
			}
		}

		if !g.useIdentifyItem {
			g.showInventory = false
			g.isActioned = true
		}
		g.showItemActions = false
		g.selectedItemIndex = 0
	}

	// 矢の「撃つ」アクション処理
	item := g.state.Player.Inventory[g.selectedItemIndex]
	if arrow, isArrow := item.(*Arrow); isArrow && g.selectedActionIndex == 1 {
		// 矢を撃つ処理
		arrow.ShotCount--
		
		// Create arrow copy for throwing
		arrowCopy := *arrow
		
		// If ShotCount becomes 0, remove from inventory
		if arrow.ShotCount == 0 {
			g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
		}

		// ThrowItem parameters
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

		g.ThrowItem(&arrowCopy, throwRange, character, mapState, enemies, onWallHit, onTargetHit)

		g.showItemActions = false
		g.selectedItemIndex = 0
		g.selectedActionIndex = 0
		return
	}

	// 矢以外のアイテムまたは矢の「投げる」の場合
	actionIndex := g.selectedActionIndex
	if _, isArrow := item.(*Arrow); isArrow && g.selectedActionIndex > 1 {
		// 矢の場合、「撃つ」が1番目に追加されているので、インデックスを調整
		actionIndex = g.selectedActionIndex - 1
	}

	if actionIndex == 1 { // Assuming index 1 corresponds to '投げる' (adjusted for arrows)
		item := g.state.Player.Inventory[g.selectedItemIndex]
		itemName := getItemNameWithSharpness(item) // You might want to adjust this if you have a different way to get the item's name.
		isCursedEquipped := false

		// Type assertion to check if item is Equipable and if it's cursed
		if equipableItem, ok := item.(Equipable); ok {
			for i, equippedItem := range g.state.Player.EquippedItems {
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
					case *Accessory:
						if v.Cursed {
							isCursedEquipped = true
						}
					}
					if isCursedEquipped {
						// If the item is cursed and equipped, do not throw and enqueue an action with a message that it cannot be thrown
						action := Action{
							Duration: 0.4,
							Message:  fmt.Sprintf("%sは呪われていて投げられない", itemName),
							Execute: func(g *Game) {
								// Any additional logic if needed
								g.showItemActions = false
								g.showInventory = false
								g.selectedItemIndex = 0
								g.selectedActionIndex = 0
							},
						}
						g.Enqueue(action)
						break
					}
					// If it is equipped and not cursed, remove it from the equipped items list
					g.state.Player.EquippedItems[i] = nil
					break
				}
			}
		}

		if !isCursedEquipped {
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

			// Continue with the throwing logic if the item is not cursed and equipped
			g.ThrowItem(item, throwRange, character, mapState, enemies, onWallHit, onTargetHit)
			g.isActioned = true

		}
	}

	if actionIndex == 2 { // Assuming index 2 corresponds to '置く' (adjusted for arrows)
		itemExistsAtPlayerPos := false
		playerX, playerY := g.state.Player.X, g.state.Player.Y
		for _, item := range g.state.Items {
			itemX, itemY := item.GetPosition()
			if itemX == playerX && itemY == playerY {
				itemExistsAtPlayerPos = true
				break
			}
		}
		// アイテムが識別されているかどうかをチェック
		identified := true
		var itemName string
		if identifiableItem, ok := g.state.Player.Inventory[g.selectedItemIndex].(Identifiable); ok {
			identified = identifiableItem.IsIdentified()
			// 識別されていない場合は識別されていないアイテム名を取得
			if !identified {
				itemName = identifiableItem.GetName()
			}
		}

		// 識別されている場合、またはIdentifiableインターフェースを実装していない場合は、Sharpnessを含む名前を使用
		if identified {
			itemName = getItemNameWithSharpness(g.state.Player.Inventory[g.selectedItemIndex])
		}

		selectedItem := g.state.Player.Inventory[g.selectedItemIndex]

		// Check if the item is cursed and equipped
		isCursedEquipped := false
		if equipableItem, ok := selectedItem.(Equipable); ok {
			for _, equippedItem := range g.state.Player.EquippedItems {
				if equippedItem == equipableItem {
					// Type assertion to Weapon or Armor to access Cursed property
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
			// If the item is cursed and equipped, enqueue an action with a message that it cannot be placed
			action := Action{
				Duration: 0.4,
				Message:  fmt.Sprintf("%sは呪われていて置けない", itemName),
				Execute: func(g *Game) {
					g.selectedItemIndex = 0
					g.selectedActionIndex = 0
					g.showItemActions = false
					g.showInventory = false
				},
			}
			g.Enqueue(action)
		} else if !itemExistsAtPlayerPos {
			action := Action{
				Duration: 0.4, // Assuming a duration of 0.5 seconds for this action
				Message:  fmt.Sprintf("%sを置いた", itemName),
				ItemName: itemName,
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
				IsIdentified: identified,
			}
			g.Enqueue(action)
		} else {
			action := Action{
				Duration: 0.4,
				Message:  fmt.Sprintf("ここには%sを置けない", itemName),
				ItemName: itemName,
				Execute: func(g *Game) {
					g.selectedItemIndex = 0
					g.selectedActionIndex = 0
					g.showItemActions = false
					g.showInventory = false
				},
				IsIdentified: identified,
			}
			g.Enqueue(action)
		}
	}

	if actionIndex == 3 { // Assuming 0-based index and "説明" is at index 3 (adjusted for arrows)
		selectedItem := g.state.Player.Inventory[g.selectedItemIndex]
		g.itemdescriptionText = selectedItem.GetDescription()
		g.showItemDescription = true
	}

}

func (g *Game) Enqueue(action Action) {
	//log.Printf("Enqueuing action: %+v", action) // ログ出力を追加
	if !action.NonBlocking {
		g.isCombatActive = true
	}
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

func (g *Game) AttackFromEnemyBlind(enemyIndex int) {
	enemy := &g.state.Enemies[enemyIndex]
	
	// 目潰し状態の敵は特技を使用せず、通常攻撃のみ
	netDamage := enemy.AttackPower - g.state.Player.DefensePower + rand.Intn(3) - 1
	if netDamage < 0 {
		netDamage = 0
	}
	
	dx, dy := g.state.Player.X-enemy.X, g.state.Player.Y-enemy.Y
	
	// 目潰し状態の場合は敵の名前を「何者」に変更
	enemyDisplayName := enemy.Name
	if g.state.Player.StatusAilments.Blind > 0 {
		enemyDisplayName = "何者"
	}
	
	action := Action{
		Duration: 0.5,
		Message:  fmt.Sprintf("%sから%dダメージを受けた", enemyDisplayName, netDamage),
		Execute: func(g *Game) {
			enemy.AttackTimer = 0.5
			enemy.AttackDirection = determineDirection(dx, dy)
			g.state.Player.Health -= netDamage
			if g.state.Player.Health < 0 {
				g.state.Player.Health = 0
			}
		},
	}
	
	g.Enqueue(action)
}

func (g *Game) AttackEnemyFromBlindEnemy(attackerIndex, targetIndex int) {
	attacker := &g.state.Enemies[attackerIndex]
	target := &g.state.Enemies[targetIndex]
	
	// 敵同士の攻撃（目潰し状態の敵が他の敵を攻撃）
	netDamage := attacker.AttackPower - target.DefensePower + rand.Intn(3) - 1
	if netDamage < 0 {
		netDamage = 0
	}
	
	dx, dy := target.X-attacker.X, target.Y-attacker.Y
	
	// 目潰し状態の場合は敵の名前を「何者」に変更
	attackerDisplayName := attacker.Name
	targetDisplayName := target.Name
	if g.state.Player.StatusAilments.Blind > 0 {
		attackerDisplayName = "何者"
		targetDisplayName = "何者"
	}
	
	action := Action{
		Duration: 0.5,
		Message:  fmt.Sprintf("%sが%sを攻撃して%dダメージを与えた", attackerDisplayName, targetDisplayName, netDamage),
		Execute: func(g *Game) {
			attacker.AttackTimer = 0.5
			attacker.AttackDirection = determineDirection(dx, dy)
			target.Health -= netDamage
			
			// ダメージを受けた敵の金縛り状態を解除
			if target.StatusAilments.Paralysis {
				target.StatusAilments.Paralysis = false
			}
			
			if target.Health <= 0 {
				// 敵を倒した場合、配列から削除
				g.state.Enemies = append(g.state.Enemies[:targetIndex], g.state.Enemies[targetIndex+1:]...)
			}
		},
	}
	
	g.Enqueue(action)
}

func (g *Game) AttackFromEnemy(enemyIndex int) {
	enemy := &g.state.Enemies[enemyIndex]

	if trap := g.state.Player.SetTrap; trap != nil && trap.GetName() == "炸裂装甲のカード" {
		// If the player has set a trap and it is the '炸裂装甲のカード', the trap will be triggered
		// 目潰し状態の場合は敵の名前を「何者」に変更
		enemyDisplayName := enemy.Name
		if g.state.Player.StatusAilments.Blind > 0 {
			enemyDisplayName = "何者"
		}
		
		action := Action{
			Duration: 0.5,
			Message:  fmt.Sprintf("%sの攻撃。", enemyDisplayName),
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

	// Check if the enemy will perform a special attack (封印状態の敵は特技を使用しない)
	if enemy.SpecialAttack != nil && randomValue <= enemy.SpecialAttackProbability && !enemy.StatusAilments.Seal {
		// Perform the special attack
		enemy.SpecialAttack(enemy, g)
	} else {
		// Perform the normal attack
		netDamage := enemy.AttackPower - g.state.Player.DefensePower + rand.Intn(3) - 1
		if netDamage < 0 { // Ensure damage does not go below 0
			netDamage = 0
		}

		dx, dy := g.state.Player.X-enemy.X, g.state.Player.Y-enemy.Y // プレイヤーと敵の位置の差を計算

		// 目潰し状態の場合は敵の名前を「何者」に変更
		enemyDisplayName := enemy.Name
		if g.state.Player.StatusAilments.Blind > 0 {
			enemyDisplayName = "何者"
		}
		
		action := Action{
			Duration: 0.5,
			Message:  fmt.Sprintf("%sから%dダメージを受けた", enemyDisplayName, netDamage),
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
			enemyIndex := i     // capture the index here
			// 目潰し状態の場合は敵の名前を「何者」に変更
			enemyDisplayName := g.state.Enemies[enemyIndex].Name
			if g.state.Player.StatusAilments.Blind > 0 {
				enemyDisplayName = "何者"
			}
			
			action := Action{
				Duration: 0.5,
				Message:  fmt.Sprintf("%sに%dダメージを与えた。", enemyDisplayName, netDamage),
				Execute: func(g *Game) {
					g.state.Enemies[enemyIndex].Health -= netDamage
					
					// ダメージを受けた敵の金縛り状態を解除
					if g.state.Enemies[enemyIndex].StatusAilments.Paralysis {
						g.state.Enemies[enemyIndex].StatusAilments.Paralysis = false
					}

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

			// 敵への攻撃時にも前方の罠を発見する
			g.checkTrapInFront()

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

		// 攻撃時に前方の罠を発見する
		g.checkTrapInFront()

		return true
	}
	return false
}

// プレイヤーが混乱状態の時のランダム攻撃処理
func (g *Game) attackPlayerConfused() {
	// 8方向のランダムな攻撃方向を選択
	directions := []struct{ dx, dy int }{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0},           {1, 0},
		{-1, 1},  {0, 1},  {1, 1},
	}
	
	direction := directions[rand.Intn(len(directions))]
	
	// プレイヤーの方向を設定
	switch {
	case direction.dx == 1 && direction.dy == 0:
		g.state.Player.Direction = Right
	case direction.dx == -1 && direction.dy == 0:
		g.state.Player.Direction = Left
	case direction.dx == 0 && direction.dy == 1:
		g.state.Player.Direction = Down
	case direction.dx == 0 && direction.dy == -1:
		g.state.Player.Direction = Up
	case direction.dx == 1 && direction.dy == -1:
		g.state.Player.Direction = UpRight
	case direction.dx == 1 && direction.dy == 1:
		g.state.Player.Direction = DownRight
	case direction.dx == -1 && direction.dy == -1:
		g.state.Player.Direction = UpLeft
	case direction.dx == -1 && direction.dy == 1:
		g.state.Player.Direction = DownLeft
	}
	
	// ランダムな方向に攻撃を試行
	attacked := g.CheckForEnemies(direction.dx, direction.dy)
	
	if attacked {
		action := Action{
			Duration: 0.4,
			Message:  "混乱して攻撃した！",
			Execute:  func(g *Game) {},
		}
		g.Enqueue(action)
	} else {
		action := Action{
			Duration: 0.4,
			Message:  "混乱して空振りした",
			Execute:  func(g *Game) {},
		}
		g.Enqueue(action)
	}
}

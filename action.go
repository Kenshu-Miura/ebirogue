//go:build !test

package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
)

// isMouthItem は口封じ状態で使えないアイテム（口にする・読み上げるもの）かどうかを返す。
func isMouthItem(item Item) bool {
	switch item.(type) {
	case *Food, *Potion, *Card:
		return true
	}
	return false
}

// currentItemActions は選択中のインベントリアイテムに表示するアクションメニュー項目を返す。
// 描画（drawActionMenu）とカーソル移動範囲（handleItemActionsInput）で共用する。
func (g *Game) currentItemActions() []string {
	if g.selectedItemIndex >= len(g.state.Player.Inventory) {
		return nil
	}
	item := g.state.Player.Inventory[g.selectedItemIndex]
	if _, isPot := item.(*Pot); isPot {
		return []string{"入れる", "出す", "投げる", "置く", "説明"}
	}
	if equipableItem, isEquipable := item.(Equipable); isEquipable {
		if weapon, ok := item.(*Weapon); ok && hasEquipmentAbility(weapon.Abilities, AbilityEdible) {
			first := "装備"
			if g.state.Player.IsEquipped(weapon) {
				first = "はずす"
			}
			return []string{first, "かじる", "投げる", "置く", "説明"}
		}
		_, isArrow := equipableItem.(*Arrow)
		if g.state.Player.IsEquipped(equipableItem) {
			if isArrow {
				return []string{"はずす", "撃つ", "投げる", "置く", "説明"}
			}
			return []string{"はずす", "投げる", "置く", "説明"}
		}
		if isArrow {
			return []string{"装備", "撃つ", "投げる", "置く", "説明"}
		}
		return []string{"装備", "投げる", "置く", "説明"}
	}
	return []string{"使う", "投げる", "置く", "説明"}
}

func (g *Game) executeGroundItemAction() {
	playerX, playerY := g.state.Player.X, g.state.Player.Y // プレイヤーの座標を取得

	if g.selectedGroundActionIndex == 0 { // Assuming index 0 corresponds to '拾う'
		for i, item := range g.state.Items { // GameStateの全てのアイテムに対してループ
			itemX, itemY := item.GetPosition()        // アイテムの座標を取得
			if itemX == playerX && itemY == playerY { // アイテムの座標とプレイヤーの座標が一致するかチェック
				itemName, identified := displayItemName(item)

				// 床に貼りついた聖域のカードは拾えない
				if isStuckSanctuaryItem(item) {
					g.EnqueueMessage(fmt.Sprintf("%sは床に貼りついていて拾えない", itemName), 0.5)
					break
				}

				// 拾得禁止のカードの効果中は拾えない
				if g.pickupBanned {
					action := Action{
						Duration:     0.5,
						Message:      fmt.Sprintf("カードの効果で%sを拾えない", itemName),
						ItemName:     itemName,
						Execute:      func(g *Game) {},
						IsIdentified: identified,
						NonBlocking:  !g.IsEnemyAdjacent(),
					}
					g.Enqueue(action)
					break
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
		g.closeGroundItemMenu()
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

				// 口封じ状態ではカード・薬・食料を使えない
				if g.state.Player.StatusAilments.MouthSeal > 0 && isMouthItem(item) {
					g.EnqueueMessage("口が封じられていて使えない", 0.4)
					g.closeGroundItemMenu()
					return
				}

				// 足元の壺は拾ってからでないと出し入れできない
				if _, isPot := item.(*Pot); isPot {
					g.EnqueueMessage("壺は拾ってから使おう", 0.4)
					g.closeGroundItemMenu()
					return
				}

				if g.useConsumable(item) {
					// 使用処理はアイテム側のUseへ委譲済み
				} else if caneItem, ok := item.(*Cane); ok {
					if !g.useCane(caneItem) {
						// 残り回数がない場合はメッセージのみでメニューを閉じる
						g.ShowGroundItem = false
						g.GroundItemActioned = false
						g.selectedGroundActionIndex = 0
						g.isActioned = true
						return
					}
				} else if equipableItem, ok := item.(Equipable); ok { // Check if item is of Equipable type

					// インベントリのサイズを確認し、いっぱいの場合はアイテムを拾わない
					if len(g.state.Player.Inventory) >= 20 {
						action := Action{
							Duration: 0.5,
							Message:  fmt.Sprintf("持ち物がいっぱいで%sを拾えなかった", item.GetName()),
							ItemName: item.GetName(),
							Execute:  func(g *Game) {},
						}
						g.Enqueue(action)
						break
					}

					var message string
					identified := false
					itemName := getItemNameWithSharpness(equipableItem)

					// アクセサリは装備しても識別されない
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
						g.capSatietyForSatori()
						g.stats.counts(itemKind(item)).Equipped++
					}
					g.PickUpItem(item, i)

					action := Action{
						Duration:     0.5,
						Message:      message,
						ItemName:     itemName,
						Execute:      func(g *Game) {},
						IsIdentified: identified,
					}
					g.Enqueue(action)
					// 装備した武器・防具が呪われていた場合はメッセージを表示
					g.enqueueCursedEquipReveal(equipableItem, itemName, identified)
				}

				g.closeGroundItemMenu()
				g.isActioned = true
				if g.itemTargetEffect != nil || g.useIdentifyItem {
					g.GroundItemActioned = true
					g.showInventory = true
					g.isActioned = false
					return
				}
			}
		}
	}

	if g.selectedGroundActionIndex == 3 { // Assuming index 3 corresponds to '投げる'
		for i, item := range g.state.Items { // GameStateの全てのアイテムに対してループ
			itemX, itemY := item.GetPosition()        // アイテムの座標を取得
			if itemX == playerX && itemY == playerY { // アイテムの座標とプレイヤーの座標が一致するかチェック
				g.selectedGroundItemIndex = i

				g.stats.counts(itemKind(item)).Thrown++
				g.throwWithCallbacks(item, 10)

				g.closeGroundItemMenu()
				g.isActioned = true
			}
		}
	}

	if g.selectedGroundActionIndex == 4 { // Assuming index 4 corresponds to '説明'
		if g.currentGroundItem != nil {
			g.groundItemDescriptionText = g.currentGroundItem.GetDescription()
			g.showGroundItemDescription = true
		}
		// 説明の場合はメニューを閉じずに説明ウィンドウのみ表示
		g.GroundItemActioned = false
		g.groundMenuJustOpened = false
	}
}

func (g *Game) executeAction() {
	// かつおぶしだけに増えた項目を処理し、後続の既存添字へ対応させる。
	if w, ok := g.state.Player.Inventory[g.selectedItemIndex].(*Weapon); ok && hasEquipmentAbility(w.Abilities, AbilityEdible) {
		if g.selectedActionIndex == 1 {
			g.gnawWeapon(w)
			return
		}
		if g.selectedActionIndex > 1 {
			g.selectedActionIndex--
		}
	}

	// 壺の「入れる」「出す」はここで処理する（それ以降の項目は汎用処理へ流す）
	if pot, isPot := g.state.Player.Inventory[g.selectedItemIndex].(*Pot); isPot {
		if g.selectedActionIndex == 0 { // 入れる
			g.startPotInsert(pot)
			return
		}
		if g.selectedActionIndex == 1 { // 出す
			g.executePotTakeOut(pot)
			return
		}
	}

	if g.selectedActionIndex == 0 { // Assuming index 0 corresponds to '使う' or '装備'
		item := g.state.Player.Inventory[g.selectedItemIndex]

		// 口封じ状態ではカード・薬・食料を使えない
		if g.state.Player.StatusAilments.MouthSeal > 0 && isMouthItem(item) {
			g.EnqueueMessage("口が封じられていて使えない", 0.4)
			g.showItemActions = false
			g.showInventory = false
			g.selectedItemIndex = 0
			g.selectedActionIndex = 0
			return
		}

		if g.useConsumable(item) {
			// 使用処理はアイテム側のUseへ委譲済み
		} else if caneItem, ok := item.(*Cane); ok {
			if !g.useCane(caneItem) {
				// 残り回数がない場合はメッセージのみでメニューを閉じる
				g.showItemActions = false
				g.showInventory = false
				g.isActioned = true
				g.selectedItemIndex = 0
				g.selectedActionIndex = 0
				return
			}
		} else if equipableItem, ok := item.(Equipable); ok { // Check if item is of Equipable type
			var message string
			identified := false
			itemName := getItemNameWithSharpness(equipableItem)

			// Check if the item is already equipped
			if g.state.Player.IsEquipped(equipableItem) {
				if isCursedEquipment(equipableItem) {
					// 呪われた装備ははずせない
					message = fmt.Sprintf("%sをはずせない。", itemName)
				} else if unequipMessage, err := g.state.Player.UnequipItem(equipableItem); err != nil {
					message = fmt.Sprintf("%sをはずせない。", itemName)
				} else {
					message = unequipMessage
				}
			} else {
				// アクセサリは装備しても識別されない
				if _, ok := equipableItem.(*Accessory); !ok {
					equipableItem.SetIdentified(true) // Set the item as identified when equipping
					identified = true
				}
				itemName = getItemNameWithSharpness(equipableItem)

				if equipMessage, err := g.state.Player.EquipItem(equipableItem); err != nil {
					message = fmt.Sprintf("%sを装備できない。", itemName)
				} else {
					message = equipMessage
					g.capSatietyForSatori()
					g.stats.counts(itemKind(equipableItem)).Equipped++
				}
			}

			action := Action{
				Duration:     0.5,
				Message:      message,
				ItemName:     itemName,
				Execute:      func(g *Game) {},
				IsIdentified: identified,
			}
			g.Enqueue(action)
			// 装備した武器・防具が呪われていた場合はメッセージを表示
			g.enqueueCursedEquipReveal(equipableItem, itemName, identified)
		}

		// 識別・白紙の書き込みは選択ウィンドウで確定するまでターンを消費しない
		if g.itemTargetEffect != nil {
			g.showItemActions = false
			g.selectedItemIndex = 0
			return
		}
		if !g.useIdentifyItem && !g.showBlankCardMenu {
			g.showInventory = false
			g.isActioned = true
		}
		g.showItemActions = false
		g.selectedItemIndex = 0
		// 「使う」「装備」はここで完結する（最後の1個を使うと所持品が空になるため、以降の処理へ進まない）
		return
	}

	// 矢の「撃つ」アクション処理
	item := g.state.Player.Inventory[g.selectedItemIndex]
	if arrow, isArrow := item.(*Arrow); isArrow && g.selectedActionIndex == 1 {
		if !g.shootArrow(arrow, 10) {
			g.dPressed = false
			g.EnqueueMessage("矢が残っていません", 0.5)
		}

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
	if _, isPot := item.(*Pot); isPot && g.selectedActionIndex > 1 {
		// 壺の場合、「入れる」「出す」が先頭にあるので、インデックスを調整
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
					isCursedEquipped = isCursedEquipment(equipableItem)
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
			// Continue with the throwing logic if the item is not cursed and equipped
			g.stats.counts(itemKind(item)).Thrown++
			g.throwWithCallbacks(item, 10)
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
		itemName, identified := displayItemName(g.state.Player.Inventory[g.selectedItemIndex])

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

					// 聖域のカードは床に置くと貼りついて拾えなくなる
					if card, ok := selectedItem.(*Card); ok && card.Name == sanctuaryCardName {
						card.Stuck = true
						g.EnqueueMessage(fmt.Sprintf("%sは床に貼りついた", card.Name), 0.5)
					}

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
	if g.playerDead && action.Message != "" {
		g.debugf("DEBUG: processAction called for dead player: Message='%s'", action.Message)
	}
	// 表示されるメッセージを履歴へ記録する
	g.messageLog.Add(action.Message)
	// 実際のアクションの実行ロジックはアクションオブジェクトのExecuteメソッドに委譲
	action.Execute(g)
	g.ActionDurationCounter = action.Duration // record the duration of the next action
}

// Enqueue adds a new attack to the attack queue
func (aq *ActionQueue) Enqueue(action Action) {
	aq.Queue = append(aq.Queue, action)
}

// enqueueEnemyNormalAttack は敵の通常攻撃（ダメージ計算・攻撃アニメーション・死亡チェック）をキューへ追加する。
// 痛恨の一撃（F4）の判定は確率が0より大きいときだけ乱数を使う。
func (g *Game) enqueueEnemyNormalAttack(enemyIndex int) {
	enemy := &g.state.Enemies[enemyIndex]

	attack := EnemyAttack{Enemy: enemy, CriticalPercent: enemyCriticalPercent(enemy), Multiplier: 1.0}
	g.runBeforeEnemyAttackHooks(&attack)
	critical := attack.CriticalPercent > 0 && g.enemyCriticalIntn()(100) < attack.CriticalPercent
	multiplier := attack.Multiplier
	if critical {
		multiplier *= enemyCriticalMultiplier
	}
	netDamage := rollEnemyAttackDamageWith(enemy.AttackPower, g.state.Player.DefensePower, multiplier, g.damageIntn())
	defense := g.resolvePlayerDefense(netDamage, EnemyDamageNormal, EnemyAttackMelee, enemy, critical)
	enemyName := g.enemyDisplayName(enemy.Name)
	enemyID, enemyX, enemyY := enemy.ID, enemy.X, enemy.Y
	source := enemyAttackSource(enemy.ID, enemy.Name)
	message := g.shieldDefenseMessage(enemyName, defense)
	if critical && !defense.Evaded && !defense.Reflected {
		message = "痛恨の一撃！" + message
	}

	// 2マス先へ届く攻撃（テッポウエビ系）でも向きが合うよう符号だけを使う
	dx, dy := sign(g.state.Player.X-enemy.X), sign(g.state.Player.Y-enemy.Y)

	action := Action{
		Duration: 0.5,
		Message:  message,
		Execute: func(g *Game) {
			enemy.AttackTimer = 0.5 // AttackTimerを設定することで敵の攻撃アニメーションが実行される
			if MonsterDefinitions[enemyID].IsBoss && g.livingBossIndex(enemyID) < 0 {
				return
			}
			enemy.AttackDirection = determineDirection(dx, dy) // 敵の攻撃方向を計算
			g.applyPlayerShieldDefense(defense, source, enemyID, enemyX, enemyY)
		},
	}

	g.Enqueue(action)
}

func (g *Game) AttackFromEnemyBlind(enemyIndex int) {
	// 聖域のカードの上にいるプレイヤーは敵から攻撃されない
	if g.playerOnSanctuary() {
		return
	}
	// 目潰し状態の敵は特技を使用せず、通常攻撃のみ
	g.enqueueEnemyNormalAttack(enemyIndex)
}

func (g *Game) AttackEnemyFromBlindEnemy(attackerIndex, targetIndex int) {
	attacker := &g.state.Enemies[attackerIndex]
	target := &g.state.Enemies[targetIndex]

	// 敵同士の攻撃（目潰し状態の敵が他の敵を攻撃）
	netDamage := rollEnemyAttackDamage(attacker.AttackPower, target.DefensePower, g.damageIntn())

	dx, dy := target.X-attacker.X, target.Y-attacker.Y
	attackerID, attackerX, attackerY := attacker.ID, attacker.X, attacker.Y
	targetID, targetX, targetY := target.ID, target.X, target.Y

	action := Action{
		Duration: 0.5,
		Message:  fmt.Sprintf("%sが%sを攻撃して%dダメージを与えた", g.enemyDisplayName(attacker.Name), g.enemyDisplayName(target.Name), netDamage),
		Execute: func(g *Game) {
			// 予約から実行までに別の攻撃で敵が倒れて並びが変わることがあるため、実行時に ID と座標で引き直す
			targetIndex := g.findEnemyForDefense(targetID, targetX, targetY)
			if targetIndex < 0 {
				return
			}
			attackerIndex := g.findEnemyForDefense(attackerID, attackerX, attackerY)
			if attackerIndex >= 0 {
				g.state.Enemies[attackerIndex].AttackTimer = 0.5
				g.state.Enemies[attackerIndex].AttackDirection = determineDirection(dx, dy)
			}
			target := &g.state.Enemies[targetIndex]
			target.Health -= netDamage
			g.WakeUpSleepingEnemyByAttack(targetIndex)

			// ダメージを受けた敵の金縛り状態を解除
			target.StatusAilments.Paralysis = false

			if target.Health <= 0 {
				if attackerIndex >= 0 {
					g.defeatEnemyByEnemy(attackerIndex, targetIndex)
				} else {
					defeated := *target
					g.dropEnemyHeldItem(targetIndex)
					g.state.Enemies = append(g.state.Enemies[:targetIndex], g.state.Enemies[targetIndex+1:]...)
					g.miniMapDirty = true
					g.onEnemyDefeated(defeated)
				}
			}
		},
	}

	g.Enqueue(action)
}

func (g *Game) AttackFromEnemy(enemyIndex int) {
	// ゾワゾワの敵は海老さんと反対の向きへ攻撃する（特技も使わない）
	if g.state.Enemies[enemyIndex].StatusAilments.Backward > 0 {
		g.enemyAttackBackward(enemyIndex)
		return
	}
	// 聖域のカードの上にいるプレイヤーは敵から攻撃されない
	if g.playerOnSanctuary() {
		return
	}

	enemy := &g.state.Enemies[enemyIndex]

	if trap := g.state.Player.SetTrap; trap != nil && trap.GetName() == "炸裂装甲のカード" {
		// If the player has set a trap and it is the '炸裂装甲のカード', the trap will be triggered
		g.EnqueueMessage(fmt.Sprintf("%sの攻撃。", g.enemyDisplayName(enemy.Name)), 0.5)
		g.EnqueueMessage(fmt.Sprintf("罠カード、%sが発動した。", trap.GetName()), 0.5)

		// 罠は1回で消費する。同じ敵ターンに攻撃してくる別の敵では発動しない
		g.state.Player.SetTrap = nil
		enemyID, enemyX, enemyY, experience := enemy.ID, enemy.X, enemy.Y, enemy.ExperiencePoints
		// ボスは炸裂装甲による即死に耐えるが、威力50の反撃ダメージは受ける。
		if MonsterDefinitions[enemyID].IsBoss {
			damage := rollEnemyAttackDamage(50, enemy.DefensePower, g.damageIntn())
			g.Enqueue(Action{Duration: 0.5, Message: fmt.Sprintf("%sは即死を耐え、%dダメージを受けた。", enemy.Name, damage), Execute: func(g *Game) {
				if index := g.findEnemyForDefense(enemyID, enemyX, enemyY); index >= 0 {
					g.applyDamageToEnemy(index, damage)
				}
			}})
			return
		}
		defeatAction := Action{
			Duration: 0.5,
			Message:  fmt.Sprintf("%sを倒した。", enemy.Name),
			Execute: func(g *Game) {
				// 予約から実行までに敵の並びが変わることがあるため、ID と座標で引き直す
				index := g.findEnemyForDefense(enemyID, enemyX, enemyY)
				if index < 0 {
					return
				}
				defeated := g.state.Enemies[index]
				g.dropEnemyHeldItem(index)
				g.state.Enemies = append(g.state.Enemies[:index], g.state.Enemies[index+1:]...)
				g.miniMapDirty = true

				// 敵の経験値をプレイヤーの所持経験値に加える
				g.state.Player.ExperiencePoints += experience

				g.state.Player.checkLevelUp(g) // レベルアップをチェック
				g.onEnemyDefeated(defeated)
			},
		}
		g.Enqueue(defeatAction)
		return
	}

	// Generate a random float number between 0 and 1 to compare with specialAttackProbability
	randomValue := g.random().Float64()

	// Check if the enemy will perform a special attack (封印状態の敵は特技を使用しない)
	if enemy.SpecialAttack != nil && randomValue <= enemy.SpecialAttackProbability && !enemy.StatusAilments.Seal {
		// Perform the special attack
		enemy.SpecialAttack(enemy, g)
	} else {
		// Perform the normal attack
		g.enqueueEnemyNormalAttack(enemyIndex)
	}
}

func (g *Game) CheckForEnemies(x, y int) bool {
	g.isFrontEnemy = false
	weapon := g.state.Player.EquippedWeapon
	weaponAbilities := []EquipmentAbilityID(nil)
	if weapon != nil {
		weaponAbilities = weapon.Abilities
	}

	g.state.Player.Direction = determineDirection(x, y)
	g.attackTimer = 0.5

	for _, delta := range playerAttackDeltas(x, y, weaponAbilities) {
		targetX := g.state.Player.X + delta.X
		targetY := g.state.Player.Y + delta.Y
		enemyIndex := g.enemyIndexAt(targetX, targetY)
		if enemyIndex < 0 {
			continue
		}
		g.isFrontEnemy = true
		g.revealEnemy(enemyIndex)
		g.enqueuePlayerAttack(enemyIndex, weaponAbilities)
	}

	if !g.isFrontEnemy && !g.enqueueWallDig(x, y, weaponAbilities) {
		g.Enqueue(Action{
			Duration: 0.5,
			Execute: func(g *Game) {
				g.playSE("miss")
				g.isActioned = true
			},
		})
	}

	g.enqueueDisposableWeaponWear(weapon)
	// 攻撃時に前方の罠を発見する
	g.checkTrapInFront()
	g.runAfterPlayerSwingHooks(PlayerSwing{DX: sign(x), DY: sign(y), HitEnemy: g.isFrontEnemy})
	return true
}

// enqueuePlayerAttack は対象1体への命中判定とダメージをキューへ追加する。
// 実行時に座標から敵を引き直し、三方向攻撃で先の敵を倒してもスライスのずれを起こさない。
// 装備能力のフック BeforePlayerAttack は命中判定の前、AfterPlayerMiss・AfterPlayerHit は
// メッセージを表示する Action の Execute 内で呼ぶ。
// 乱数は 命中 → 敵の回避（回避率があるときだけ）→ 会心 → 乱数係数 の順で使う。
func (g *Game) enqueuePlayerAttack(enemyIndex int, weaponAbilities []EquipmentAbilityID) {
	enemy := &g.state.Enemies[enemyIndex]
	targetX, targetY := enemy.X, enemy.Y
	multiplier, slayerEffective := slayerMultiplier(weaponAbilities, enemy.Traits)
	attack := &PlayerAttack{
		Weapon:              g.state.Player.EquippedWeapon,
		Target:              enemy,
		Accuracy:            basePlayerAttackAccuracy,
		SureHit:             hasEquipmentAbility(weaponAbilities, AbilitySureHit),
		EnemyEvasion:        enemyEvasionPercent(enemy),
		Multiplier:          multiplier,
		CriticalDenominator: playerCriticalDenominator,
	}
	if slayerEffective {
		attack.MessagePrefix = "特効！"
	}
	g.runBeforePlayerAttackHooks(attack)

	enemyName := g.enemyDisplayName(enemy.Name)
	hit := rollPlayerAttackHitChance(attack.Accuracy, attack.SureHit, g.attackHitIntn())
	evaded := hit && !attack.SureHit && attack.EnemyEvasion > 0 && g.enemyEvasionIntn()(100) < attack.EnemyEvasion
	if !hit || evaded {
		message := fmt.Sprintf("%sへの攻撃は外れた。", enemyName)
		if evaded {
			message = fmt.Sprintf("%sは攻撃をかわした。", enemyName)
		}
		g.Enqueue(Action{
			Duration: 0.5,
			Message:  message,
			Execute: func(g *Game) {
				g.playSE("miss")
				g.isActioned = true
				attack.Target = nil // 実行時の敵を引き直す（倒れていれば nil）
				if i := g.enemyIndexAt(targetX, targetY); i >= 0 {
					attack.Target = &g.state.Enemies[i]
				}
				g.runAfterPlayerMissHooks(attack)
			},
		})
		return
	}

	total := playerAttackTotal(g.state.Player.AttackPower, g.state.Player.Power, g.state.Player.Level) + attack.AttackBonus
	result := rollDamage(DamageParams{
		Attack:              total,
		Defense:             enemy.DefensePower,
		Multiplier:          attack.Multiplier,
		MinDamage:           minAttackDamage,
		CriticalDenominator: attack.CriticalDenominator,
		ForceCritical:       attack.ForceCritical,
	}, g.damageIntn())
	netDamage := result.Damage
	message := attack.MessagePrefix + fmt.Sprintf("%sに%dダメージを与えた。", enemyName, netDamage)
	if result.Critical {
		message = "会心の一撃！" + message
	}
	weapon := attack.Weapon
	hitEffects := attack.HitEffects

	g.Enqueue(Action{
		Duration: 0.5,
		Message:  message,
		Execute: func(g *Game) {
			currentIndex := g.enemyIndexAt(targetX, targetY)
			if currentIndex < 0 {
				return
			}
			g.playSE("hit")
			target := g.state.Enemies[currentIndex]
			hitResult := PlayerAttackHit{Weapon: weapon, EnemyID: target.ID, Enemy: target, Damage: netDamage,
				Critical: result.Critical, Defeated: target.Health-netDamage <= 0}
			g.WakeUpSleepingEnemyByAttack(currentIndex)
			g.applyDamageToEnemy(currentIndex, netDamage)
			g.isActioned = true
			for _, effect := range hitEffects {
				effect(g)
			}
			g.runAfterPlayerHitHooks(hitResult)
		},
	})
}

// enqueueWallDig はつるはし能力で正面の壁を通路へ変える。
func (g *Game) enqueueWallDig(dx, dy int, weaponAbilities []EquipmentAbilityID) bool {
	if !hasEquipmentAbility(weaponAbilities, AbilityDigWall) || len(g.state.Map) == 0 || len(g.state.Map[0]) == 0 {
		return false
	}
	x := g.state.Player.X + dx
	y := g.state.Player.Y + dy
	width, height := len(g.state.Map[0]), len(g.state.Map)
	if x < 0 || y < 0 || x >= width || y >= height || !isDiggableTile(g.state.Map[y][x], x, y, width, height) {
		return false
	}

	g.Enqueue(Action{
		Duration: 0.5,
		Message:  "壁を掘った。",
		Execute: func(g *Game) {
			if g.digWallAt(x, y) {
				// 海老さんが掘った隣のマスは必ず見えている
				g.state.Map[y][x].Visited = true
				g.state.Map[y][x].Brightness = 1.0
			}
			g.isActioned = true
		},
	})
	return true
}

// enqueueDisposableWeaponWear は使い捨て武器の基礎攻撃力を攻撃1回につき1下げる。
func (g *Game) enqueueDisposableWeaponWear(weapon *Weapon) {
	if weapon == nil {
		return
	}
	nextPower, changed := nextDisposableWeaponAttackPower(weapon.AttackPower, weapon.Abilities)
	if !changed {
		return
	}
	g.Enqueue(Action{
		Duration: 0.3,
		Message:  fmt.Sprintf("%sの攻撃力が1下がった。", weapon.Name),
		Execute: func(g *Game) {
			if g.state.Player.EquippedWeapon != weapon {
				return
			}
			weapon.AttackPower = nextPower
			g.state.Player.AttackPower--
		},
	})
}

// プレイヤーが混乱状態の時のランダム攻撃処理
func (g *Game) attackPlayerConfused() {
	// 8方向のランダムな攻撃方向を選択
	directions := getDirections()
	direction := directions[g.randIntn(len(directions))]

	// プレイヤーの方向を設定
	g.state.Player.Direction = determineDirection(direction.dx, direction.dy)

	// ランダムな方向に攻撃を試行
	if g.CheckForEnemies(direction.dx, direction.dy) {
		g.EnqueueMessage("混乱して攻撃した！", 0.4)
	} else {
		g.EnqueueMessage("混乱して空振りした", 0.4)
	}
}

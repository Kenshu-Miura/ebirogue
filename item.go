//go:build !test

package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
)

func (g *Game) updateItemVisibility() {
	// プレイヤーの座標を取得
	playerX, playerY := g.state.Player.X, g.state.Player.Y

	// 全てのアイテムに対してループを実行
	for _, item := range g.state.Items {
		// アイテムの座標を取得
		itemX, itemY := item.GetPosition()

		// プレイヤーが盲目状態の場合、アイテムを非表示にする
		if g.state.Player.StatusAilments.Blind > 0 {
			item.SetShowOnMiniMap(false)
			g.miniMapDirty = true
		} else {
			// プレイヤーとアイテムが同じ部屋にあるかどうかを確認
			inSameRoom := isSameRoom(playerX, playerY, itemX, itemY, g.rooms)

			if inSameRoom {
				// 同じ部屋にある場合、発見状態に設定し、ミニマップに表示
				item.SetPlayerDiscovered(true)
				item.SetShowOnMiniMap(true)
				g.miniMapDirty = true
			} else if item.GetPlayerDiscovered() || g.playerHasAbility(AbilitySenseItems) || g.playerHasAbility(AbilityClairvoyance) {
				// 同じ部屋にないが一度発見されたアイテムは表示し続ける
				item.SetShowOnMiniMap(true)
				g.miniMapDirty = true
			} else {
				// 発見されていないアイテムは非表示
				g.miniMapDirty = g.miniMapDirty || item.GetShowOnMiniMap()
				item.SetShowOnMiniMap(false)
			}
		}
	}
}

func (g *Game) UpdateThrownItem() {
	// 画像の有無ではなくアイテムの有無で判定する（ヘッドレスでは画像が無い）
	if g.ThrownItem.Item != nil {
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

				if pot, isPot := g.ThrownItem.Item.(*Pot); isPot {
					// 壺は着地・命中時に割れて中身が散らばる
					g.EnqueueMessage(fmt.Sprintf("%sは割れた！", pot.GetName()), 0.5)
					g.scatterPotContents(pot, g.ThrownItemDestination.X, g.ThrownItemDestination.Y)
				} else if itemExists && g.TargetEnemy == nil {
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
					} else if g.TargetEnemy == nil {
						// Place the item normally if no item exists at the destination
						g.state.Items = append(g.state.Items, g.ThrownItem.Item)
					}
				}
				g.miniMapDirty = true

				// Reset the thrown item and its destination
				g.dPressed = false
				g.ThrownItem = ThrownItem{}
				g.ThrownItemDestination = Coordinate{}
				g.TargetEnemy = nil
			}
		}
	}
}

// removeThrownItemFromSource は投げたアイテムを元の場所（所持品または足元）から取り除く。
// 投擲は Action として遅れて実行され、その間に足元メニューなどの状態が変わるため、
// 選択位置ではなくアイテムそのもので探す（足元から投げたのに所持品が消える不具合の対策）。
func (g *Game) removeThrownItemFromSource(item Item) {
	for i, owned := range g.state.Player.Inventory {
		if owned == item {
			g.state.Player.Inventory = append(g.state.Player.Inventory[:i], g.state.Player.Inventory[i+1:]...)
			return
		}
	}
	for i, ground := range g.state.Items {
		if ground == item {
			g.state.Items = append(g.state.Items[:i], g.state.Items[i+1:]...)
			g.GroundItemActioned = false
			g.selectedGroundActionIndex = 0
			return
		}
	}
}

func (g *Game) onTargetHit(target Character, item Item, index int) {
	if _, ok := target.(*Enemy); ok && index >= 0 && index < len(g.state.Enemies) {
		g.revealEnemy(index)
	}

	// Check if the item is of type Cane
	if cane, ok := item.(*Cane); ok {
		// 封印の杖を投げた場合、直接当たった敵を封印状態にする
		if cane.Name == "封印の杖" {
			if _, ok := target.(*Enemy); ok && index >= 0 && index < len(g.state.Enemies) {
				action := Action{
					Duration: 0.5,
					Message:  fmt.Sprintf("%sを封印状態にした", target.GetName()),
					Execute: func(g *Game) {
						g.state.Enemies[index].StatusAilments.Seal = true
						g.isActioned = true
					},
				}
				g.Enqueue(action)
			}
		} else {
			// 他の杖の場合は通常の処理
			cane.Use(g)
		}
	} else if potion, ok := item.(*Potion); ok {
		// 睡眠薬の場合の処理
		if potion.Name == "睡眠薬" {
			message := fmt.Sprintf("%sは眠った", target.GetName())
			guardMessage, immune := "", false
			if _, ok := target.(*Player); ok {
				guardMessage, immune = g.playerImmunityMessage(StatusSleep)
			}
			if immune {
				message = guardMessage
			}
			action := Action{
				Duration: 0.5,
				Message:  message,
				Execute: func(g *Game) {
					// Type assertion to check if target is of type *Player or *Enemy
					if _, ok := target.(*Player); ok {
						// If target is of type *Player（装備能力で無効なら眠らない）
						if !immune {
							g.state.Player.StatusAilments.Sleep = 10
						}
					} else if _, ok := target.(*Enemy); ok && index >= 0 && index < len(g.state.Enemies) {
						// If target is of type *Enemy
						g.state.Enemies[index].StatusAilments.Sleep = 10
					}
					g.isActioned = true
				},
			}
			g.Enqueue(action)
		} else if potion.Name == "混乱薬" {
			// 混乱薬の場合の処理
			message := fmt.Sprintf("%sは混乱した", target.GetName())
			guardMessage, immune := "", false
			if _, ok := target.(*Player); ok {
				guardMessage, immune = g.playerImmunityMessage(StatusConfusion)
			}
			if immune {
				message = guardMessage
			}
			action := Action{
				Duration: 0.5,
				Message:  message,
				Execute: func(g *Game) {
					// Type assertion to check if target is of type *Player or *Enemy
					if _, ok := target.(*Player); ok {
						// If target is of type *Player（装備能力で無効なら混乱しない）
						if !immune {
							g.state.Player.StatusAilments.Confusion = 10
						}
					} else if _, ok := target.(*Enemy); ok && index >= 0 && index < len(g.state.Enemies) {
						// If target is of type *Enemy
						g.state.Enemies[index].StatusAilments.Confusion = 10
					}
					g.isActioned = true
				},
			}
			g.Enqueue(action)
		} else if potion.Name == "目潰し薬" {
			// 目潰し薬の場合の処理
			// 効果を即座に適用（ミニマップを即座に非表示にするため）
			message := fmt.Sprintf("%sは目が見えなくなった", target.GetName())
			if _, ok := target.(*Player); ok {
				if guardMessage, immune := g.playerImmunityMessage(StatusBlind); immune {
					message = guardMessage // 装備能力で無効なら目潰しにならない
				} else {
					g.state.Player.StatusAilments.Blind = 30
				}
				// ミニマップを更新するためのフラグを設定
				g.miniMapDirty = true
				// ミニマップキャッシュを強制的にクリアして即座に更新
				if g.miniMap != nil {
					g.miniMap.Clear()
				}
			} else if _, ok := target.(*Enemy); ok && index >= 0 && index < len(g.state.Enemies) {
				// If target is of type *Enemy - 永続的な目潰し状態にする
				g.state.Enemies[index].StatusAilments.Blind = 999
			}

			action := Action{
				Duration: 0.5,
				Message:  message,
				Execute: func(g *Game) {
					g.isActioned = true
				},
			}
			g.Enqueue(action)
		} else {
			// 通常のポーションの場合の処理
			recovery := recoveredValue(target.GetHealth(), target.GetMaxHealth(), potion.Health, potion.FullRecovery) - target.GetHealth()
			action := Action{
				Duration: 0.5, // Assuming a duration of 0.5 seconds for this action
				Message:  fmt.Sprintf("%sのHPが%d回復した。", target.GetName(), recovery),
				Execute: func(g *Game) {
					// Type assertion to check if target is of type *Player or *Enemy
					if _, ok := target.(*Player); ok {
						// If target is of type *Player
						g.state.Player.Health = recoveredValue(g.state.Player.Health, g.state.Player.GetMaxHealth(), potion.Health, potion.FullRecovery)
					} else if _, ok := target.(*Enemy); ok && index >= 0 && index < len(g.state.Enemies) {
						// If target is of type *Enemy
						g.state.Enemies[index].Health = recoveredValue(g.state.Enemies[index].Health, g.state.Enemies[index].GetMaxHealth(), potion.Health, potion.FullRecovery)
					}
					g.isActioned = true
					// Reset the target character after processing
				},
			}
			g.Enqueue(action)
		}
	} else if card, ok := item.(*Card); ok && card.Name == genocideCardName {
		// ジェノサイドのカードは投げ当てた敵と同系統の敵をこの世から封じる
		if _, isEnemy := target.(*Enemy); isEnemy && index >= 0 && index < len(g.state.Enemies) {
			targetID := g.state.Enemies[index].ID
			targetName := g.state.Enemies[index].Name
			message := fmt.Sprintf("%sの系統はこの世から封じられた", g.enemyDisplayName(targetName))
			if MonsterDefinitions[targetID].IsBoss {
				message = fmt.Sprintf("%sはジェノサイドの力を跳ね返した", g.enemyDisplayName(targetName))
			}
			action := Action{
				Duration: 0.5,
				Message:  message,
				Execute: func(g *Game) {
					g.applyGenocide(targetID)
					g.isActioned = true
				},
			}
			g.Enqueue(action)
		} else {
			g.EnqueueMessage("しかし何も起こらなかった", 0.5)
		}
	} else {
		damage := 0
		critical := false
		if g.dPressed {
			// 射撃はプレイヤーの総攻撃力に矢の攻撃力を加えて計算する
			attack := playerAttackTotal(g.state.Player.AttackPower, g.state.Player.Power, g.state.Player.Level)
			if arrow, ok := item.(*Arrow); ok {
				attack += arrow.AttackPower
			}
			result := rollPlayerAttackDamage(attack, target.GetDefensePower(), 1.0, g.damageIntn())
			damage = result.Damage
			critical = result.Critical
		} else {
			damage = rollThrownItemDamage(g.damageIntn())
		}
		// 目潰し状態の場合は敵の名前を「何者」に変更（targetがEnemyの場合のみ）
		targetDisplayName := target.GetName()
		if _, ok := target.(*Enemy); ok {
			targetDisplayName = g.enemyDisplayName(targetDisplayName)
		}

		hitMessage := fmt.Sprintf("%sに%dのダメージを与えた。", targetDisplayName, damage)
		if critical {
			hitMessage = "会心の一撃！" + hitMessage
		}

		action := Action{
			Duration: 0.5, // Assuming a duration of 0.5 seconds for this action
			Message:  hitMessage,
			Execute: func(g *Game) {
				// Type assertion to check if target is of type *Player or *Enemy
				if _, ok := target.(*Player); ok {
					// If target is of type *Player
					g.damagePlayer(damage, DamageSource{Kind: DamageSourceOther, Detail: item.GetName()})
				} else if _, ok := target.(*Enemy); ok && index >= 0 && index < len(g.state.Enemies) {
					// ダメージ適用・金縛り解除・撃破処理
					g.applyDamageToEnemy(index, damage)
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
			case *Pot:
				return fmt.Sprintf("%s[%d/%d]", item.GetName(), len(item.Contents), item.Capacity) // 壺は中身数と容量を表示
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
			// 床に貼りついた聖域のカードは交換できない
			if isStuckSanctuaryItem(item) {
				g.ActionQueue.Enqueue(Action{
					Duration: 0.4,
					Message:  "聖域のカードは床に貼りついていて交換できない",
					Execute: func(g *Game) {
						g.selectedItemIndex = 0
					},
				})
				break
			}

			// 拾得禁止のカードの効果中は交換でも足元のアイテムを入手できない
			if g.pickupBanned {
				g.ActionQueue.Enqueue(Action{
					Duration: 0.4,
					Message:  "カードの効果で足元のアイテムを拾えない",
					Execute: func(g *Game) {
						g.selectedItemIndex = 0
					},
				})
				break
			}

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

			// Check if the selected inventory item is Equipable and cursed using new system
			if equipableItem, ok := selectedInventoryItem.(Equipable); ok {
				if g.state.Player.IsEquipped(equipableItem) {
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
						// Check if the item is equipped and unequip if necessary using new system
						if equipableItem, ok := selectedInventoryItem.(Equipable); ok {
							if g.state.Player.IsEquipped(equipableItem) {
								g.state.Player.UnequipItem(equipableItem)
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
	g.stats.counts(itemKind(item)).Picked++
	if money, isMoney := item.(*Money); isMoney {
		// itemがMoney型である場合、プレイヤーの所持金を増加させる
		g.state.Player.Cash += money.Amount
	} else {
		// それ以外の場合、アイテムをプレイヤーのインベントリに追加
		g.state.Player.Inventory = append(g.state.Player.Inventory, item)
		if identifiable, ok := item.(Identifiable); ok && !identifiable.IsIdentified() && g.playerHasAbility(AbilityAppraiser) {
			identifiable.SetIdentified(true)
			g.EnqueueMessage(fmt.Sprintf("%sだった。", identifiable.GetName()), 0.5)
		}
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

				// 床に貼りついた聖域のカードは拾わずに上へ乗るだけ
				if isStuckSanctuaryItem(item) {
					action := Action{
						Duration:     0.5,
						Message:      fmt.Sprintf("%sに乗った", itemName),
						ItemName:     itemName,
						Execute:      func(g *Game) {},
						IsIdentified: identified,
						NonBlocking:  !g.IsEnemyAdjacent(),
					}
					g.Enqueue(action)
					break
				}

				// 拾得禁止のカードの効果中は拾わずに上へ乗るだけ
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
					message := fmt.Sprintf("%sを拾った", itemName) // メッセージ全体を作成
					action := Action{
						Duration:     0.8,
						Message:      message,
						ItemName:     itemName,
						Execute:      func(g *Game) { g.PickUpItem(item, i) },
						IsIdentified: identified,
						NonBlocking:  !g.IsEnemyAdjacent(),
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
						NonBlocking:  !g.IsEnemyAdjacent(),
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

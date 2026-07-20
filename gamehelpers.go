//go:build !test

package main

import "fmt"

// EnqueueMessage は表示のみで副作用を持たないメッセージアクションをキューへ追加する。
func (g *Game) EnqueueMessage(message string, duration float64) {
	g.Enqueue(Action{
		Duration: duration,
		Message:  message,
		Execute:  func(g *Game) {},
	})
}

// enemyDisplayName はプレイヤーが目潰し状態のとき敵名を「何者」に置き換える。
func (g *Game) enemyDisplayName(name string) string {
	if g.state.Player.StatusAilments.Blind > 0 {
		return "何者"
	}
	return name
}

// displayItemName はアイテムの表示名と識別済みかどうかを返す。
// 未識別のアイテムは未識別名、識別済みは強化値付きの名前になる。
func displayItemName(item Item) (name string, identified bool) {
	identified = true
	if identifiableItem, ok := item.(Identifiable); ok {
		identified = identifiableItem.IsIdentified()
		if !identified {
			return identifiableItem.GetName(), false
		}
	}
	return getItemNameWithSharpness(item), identified
}

// isCursedEquipment は装備品（武器・防具・矢・アクセサリ）が呪われているかどうかを返す。
func isCursedEquipment(item Equipable) bool {
	switch v := item.(type) {
	case *Weapon:
		return v.Cursed
	case *Armor:
		return v.Cursed
	case *Arrow:
		return v.Cursed
	case *Accessory:
		return v.Cursed
	}
	return false
}

// enqueueCursedEquipReveal は武器・防具を装備した際に呪いが判明するメッセージを追加する。
// アクセサリの呪いは装備時には判明しない。
func (g *Game) enqueueCursedEquipReveal(item Equipable, itemName string, identified bool) {
	cursed := false
	switch v := item.(type) {
	case *Weapon:
		cursed = v.Cursed
	case *Armor:
		cursed = v.Cursed
	}
	if !cursed {
		return
	}
	g.Enqueue(Action{
		Duration:     0.5,
		Message:      fmt.Sprintf("%sは呪われていた。", itemName),
		ItemName:     itemName,
		Execute:      func(g *Game) {},
		IsIdentified: identified,
	})
}

// throwWithCallbacks は標準のコールバックでプレイヤーの位置からアイテムを投げる。
func (g *Game) throwWithCallbacks(item Item, throwRange int) {
	g.ThrowItem(item, throwRange, &g.state.Player, g.state.Map, g.state.Enemies,
		func(item Item, position Coordinate, itemIndex int) {
			g.onWallHit(item, position, itemIndex)
		},
		func(target Character, item Item, index int) {
			g.onTargetHit(target, item, index)
		})
}

// shootArrow は矢を1本消費し、射撃として投射する。
// Dキーとインベントリの「撃つ」で同じ処理を使い、回転・ダメージ判定を統一する。
func (g *Game) shootArrow(arrow *Arrow, throwRange int) bool {
	if arrow == nil || arrow.ShotCount <= 0 {
		return false
	}

	arrow.ShotCount--
	shotArrow := *arrow
	shotArrow.ShotCount = 1

	if arrow.ShotCount == 0 {
		for i, inventoryItem := range g.state.Player.Inventory {
			inventoryArrow, ok := inventoryItem.(*Arrow)
			if ok && inventoryArrow == arrow {
				g.state.Player.Inventory = append(g.state.Player.Inventory[:i], g.state.Player.Inventory[i+1:]...)
				break
			}
		}
		if g.state.Player.EquippedArrow == arrow {
			g.state.Player.EquippedArrow = nil
		}
	}

	// 描画の回転、射撃メッセージ、射撃ダメージの判定に使用される。
	g.dPressed = true
	g.throwWithCallbacks(&shotArrow, throwRange)
	return true
}

// useCane は杖を1回振る。残り回数がない場合はメッセージのみ表示して false を返す。
func (g *Game) useCane(caneItem *Cane) bool {
	if caneItem.Uses <= 0 {
		g.EnqueueMessage(fmt.Sprintf("%sを使った。しかし何も起こらなかった。", caneItem.GetName()), 0.5)
		return false
	}
	// 効果弾として飛ばすための複製を作り、元の杖の残り回数を減らす
	caneItemCopy := *caneItem
	caneItem.Uses--
	caneItemCopy.BaseItem.Type = "Effect"
	g.throwWithCallbacks(&caneItemCopy, 30)
	return true
}

// useConsumable は食料・薬・カード・お金・罠アイテムを使用する。該当しない場合は false を返す。
func (g *Game) useConsumable(item Item) bool {
	switch v := item.(type) {
	case *Food:
		v.Use(g)
	case *Potion:
		v.Use(g)
	case *Card:
		v.Use(g)
	case *Money:
		v.Use(g)
	case *Trap:
		v.Use(g)
	default:
		return false
	}
	return true
}

// defeatEnemy は撃破メッセージを表示し、敵を取り除いて経験値を加算する。
func (g *Game) defeatEnemy(index int) {
	enemy := g.state.Enemies[index]
	g.EnqueueMessage(fmt.Sprintf("%sを倒した。", enemy.Name), 0.5)
	g.dropEnemyHeldItem(index)
	g.state.Enemies = append(g.state.Enemies[:index], g.state.Enemies[index+1:]...)
	g.state.Player.ExperiencePoints += enemy.ExperiencePoints
	g.state.Player.checkLevelUp(g)
}

// dropEnemyHeldItem は盗賊系の敵が持つアイテムを撃破地点へ戻す。
func (g *Game) dropEnemyHeldItem(index int) {
	enemy := &g.state.Enemies[index]
	if enemy.HeldItem == nil {
		return
	}
	enemy.HeldItem.SetPosition(enemy.X, enemy.Y)
	enemy.HeldItem.SetShowOnMiniMap(true)
	enemy.HeldItem.SetPlayerDiscovered(true)
	g.state.Items = append(g.state.Items, enemy.HeldItem)
	g.miniMapDirty = true
	enemy.HeldItem = nil
	enemy.Fleeing = false
}

// applyDamageToEnemy は敵へダメージを与えて金縛りを解除し、倒した場合は撃破処理を行う。
func (g *Game) applyDamageToEnemy(index, damage int) {
	g.state.Enemies[index].Health -= damage
	g.state.Enemies[index].StatusAilments.Paralysis = false
	if g.state.Enemies[index].Health <= 0 {
		g.defeatEnemy(index)
	}
}

// closeGroundItemMenu は足元メニューの表示状態と選択をリセットする。
func (g *Game) closeGroundItemMenu() {
	g.ShowGroundItem = false
	g.GroundItemActioned = false
	g.selectedGroundActionIndex = 0
	g.groundMenuJustOpened = false
}

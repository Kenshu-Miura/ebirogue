//go:build !test

package main

import (
	"fmt"
	"sort"
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

var restoreSatiety = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)

	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを食べた", item.GetName()),
		Execute: func(g *Game) {
		},
	}
	g.Enqueue(action)
	if foodItem, ok := item.(*Food); ok && g.state.Player.Satiety == g.state.Player.MaxSatiety {
		action := Action{
			Duration: 0.4,
			Message:  fmt.Sprintf("最大満腹度が%d上昇した。", foodItem.MaxStatBonus),
			Execute: func(g *Game) {
				g.state.Player.MaxSatiety += foodItem.MaxStatBonus
				g.state.Player.Satiety = g.state.Player.MaxSatiety
				g.capSatietyForSatori()
			},
		}
		g.Enqueue(action)
	} else {
		if foodItem, ok := item.(*Food); ok {
			recovered := recoveredValue(g.state.Player.Satiety, g.state.Player.MaxSatiety, foodItem.Satiety, foodItem.FullRecovery)
			recovered = satoriSatietyCap(recovered, g.playerHasAbility(AbilitySatori))
			action := Action{
				Duration: 0.4,
				Message:  fmt.Sprintf("満腹度が%d回復した。", recovered-g.state.Player.Satiety),
				Execute: func(g *Game) {
					g.state.Player.Satiety = recovered
					g.capSatietyForSatori()
				},
			}
			g.Enqueue(action)
		}
	}
	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

var restoreHP = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)

	// アクションの生成
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを食べた", item.GetName()),
		Execute:  func(g *Game) {},
	}
	g.Enqueue(action)

	// HP回復の処理
	if potionItem, ok := item.(*Potion); ok && g.state.Player.Health == g.state.Player.MaxHealth {
		action := Action{
			Duration: 0.4,
			Message:  fmt.Sprintf("最大HPが%d上昇した。", potionItem.MaxStatBonus),
			Execute: func(g *Game) {
				g.state.Player.MaxHealth += potionItem.MaxStatBonus
				if !statusActive(g.state.Player.StatusAilments, StatusNoHealing) {
					g.state.Player.Health = g.state.Player.MaxHealth
				}
			},
		}
		g.Enqueue(action)
	} else {
		if potionItem, ok := item.(*Potion); ok {
			recovered := recoveredValue(g.state.Player.Health, g.state.Player.MaxHealth, potionItem.Health, potionItem.FullRecovery)
			if statusActive(g.state.Player.StatusAilments, StatusNoHealing) {
				recovered = g.state.Player.Health
			}
			action := Action{
				Duration: 0.4,
				Message:  fmt.Sprintf("HPが%d回復した。", recovered-g.state.Player.Health),
				Execute: func(g *Game) {
					g.state.Player.Health = g.state.Player.recoveredHP(potionItem.Health, potionItem.FullRecovery)
					g.clearPlayerStatus(StatusNigiri)
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
			// プレイヤーの向いている方向の1マス先を対象にする
			dx, dy := directionToDelta(g.state.Player.Direction)
			targetX, targetY := g.state.Player.X+dx, g.state.Player.Y+dy
			for i, enemy := range g.state.Enemies {
				if enemy.X == targetX && enemy.Y == targetY {
					action := Action{
						Duration: 0.5,
						Message:  fmt.Sprintf("%sに30ダメージを与えた。", g.enemyDisplayName(g.state.Enemies[i].Name)),
						Execute: func(g *Game) {
							// ダメージ適用・金縛り解除・撃破処理
							g.applyDamageToEnemy(i, 30)
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
	// 足元から使った場合もあるため、ほかの効果と同じく使用元を判定する
	item, isInventoryItem := determineItemSource(g)
	moneyItem, ok := item.(*Money)
	if !ok {
		return
	}
	action := Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%dを入手した。", moneyItem.Amount),
		Execute: func(g *Game) {
			g.state.Player.Cash += moneyItem.Amount
		},
	}
	g.Enqueue(action)
	removeUsedItem(g, isInventoryItem)
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
			defer g.enemyTrapAfterMove(g.TargetEnemyIndex, g.state.Enemies[g.TargetEnemyIndex].X, g.state.Enemies[g.TargetEnemyIndex].Y)
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
			g.applyEnemyStatus(g.TargetEnemyIndex, StatusSeal, 1)
			g.TargetEnemyIndex = -1
		},
	}
	g.Enqueue(action)

}

// rollIdentifyAll は真実の眼のカードが持ち物すべてを識別するかどうかを判定する。
func rollIdentifyAll(intn func(int) int) bool {
	return intn(5) == 0
}

// identifyAllInventory は所持アイテムのうち識別可能なものをすべて識別し、識別した数を返す。
func identifyAllInventory(g *Game) int {
	identified := 0
	for _, item := range g.state.Player.Inventory {
		if identifiableItem, ok := item.(Identifiable); ok && !identifiableItem.GetIdentified() {
			identifiableItem.SetIdentified(true)
			identified++
		}
	}
	return identified
}

// 壺拡大のカード: 持っている壺すべての容量を1増やす
var expandPotsCard = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.EnqueueMessage(fmt.Sprintf("%sを使った。", item.GetName()), 0.4)

	if len(g.inventoryPots()) == 0 {
		g.EnqueueMessage("しかし何も起こらなかった。", 0.4)
	} else {
		g.Enqueue(Action{
			Duration: 0.4,
			Message:  "壺が大きくなった！",
			Execute: func(g *Game) {
				for _, pot := range g.inventoryPots() {
					pot.Capacity = expandedPotCapacity(pot.Capacity)
				}
			},
		})
	}
	removeUsedItem(g, isInventoryItem)
}

// 吸い出しのカード: 持っている壺すべての中身を壺を壊さずに取り出す
var suckOutPotsCard = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使った。", item.GetName()),
		Execute: func(g *Game) {
			moved, leftover := g.suckOutAllPots()
			if moved == 0 {
				g.EnqueueMessage("しかし何も起こらなかった。", 0.4)
				return
			}
			g.EnqueueMessage("壺の中身が飛び出してきた！", 0.4)
			if leftover {
				g.EnqueueMessage("持ち物がいっぱいで全部は取り出せなかった。", 0.4)
			}
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// 真実の眼のカード: 対象を選ぶ画面を開く。カードの消費と全識別の判定は対象を確定したとき
// （executeItemIdentify）に行うので、選択を取り消しても判定を振り直せない。
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

// uncursePlayerBelongings は装備品と所持品の呪いをすべて解き、解いた装備の名前を返す。
func uncursePlayerBelongings(player *Player) []string {
	uncursed := []string{}
	uncurseAccessory := func(a *Accessory) {
		if a == nil || !a.Cursed {
			return
		}
		// 呪われたアクセサリは装備補正が反転しているため、解呪の前後で装備効果を付け直す
		equipped := player.IsEquipped(a)
		if equipped {
			a.UpdatePlayerStats(player, false)
		}
		a.Cursed = false
		if equipped {
			a.UpdatePlayerStats(player, true)
		}
		uncursed = append(uncursed, a.GetName())
	}

	for _, item := range player.Inventory {
		switch v := item.(type) {
		case *Weapon:
			if v.Cursed {
				v.Cursed = false
				uncursed = append(uncursed, v.GetName())
			}
		case *Armor:
			if v.Cursed {
				v.Cursed = false
				uncursed = append(uncursed, v.GetName())
			}
		case *Arrow:
			if v.Cursed {
				v.Cursed = false
				uncursed = append(uncursed, v.GetName())
			}
		case *Accessory:
			uncurseAccessory(v)
		}
	}

	// 装備品はインベントリと同じポインタを共有するが、念のため装備欄も確認する
	if w := player.EquippedWeapon; w != nil && w.Cursed {
		w.Cursed = false
		uncursed = append(uncursed, w.GetName())
	}
	if a := player.EquippedArmor; a != nil && a.Cursed {
		a.Cursed = false
		uncursed = append(uncursed, a.GetName())
	}
	if ar := player.EquippedArrow; ar != nil && ar.Cursed {
		ar.Cursed = false
		uncursed = append(uncursed, ar.GetName())
	}
	uncurseAccessory(player.EquippedAccessories[0])
	uncurseAccessory(player.EquippedAccessories[1])

	return uncursed
}

// おはらいのカードは装備品と所持品の呪いをすべて解く。
var removeCurse = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した。白い光が海老さんを包んだ", item.GetName()),
		Execute: func(g *Game) {
			uncursed := uncursePlayerBelongings(&g.state.Player)
			if len(uncursed) == 0 {
				g.EnqueueMessage("しかし何も起こらなかった", 0.4)
				return
			}
			for _, name := range uncursed {
				g.EnqueueMessage(fmt.Sprintf("%sの呪いが解けた", name), 0.4)
			}
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// 武器強化のカードは装備中の武器の強化値を1上げ、呪いも解く。
var reinforceWeapon = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した", item.GetName()),
		Execute: func(g *Game) {
			weapon := g.state.Player.EquippedWeapon
			if weapon == nil {
				g.EnqueueMessage("しかし何も起こらなかった", 0.4)
				return
			}
			if weapon.Cursed {
				weapon.Cursed = false
				g.EnqueueMessage(fmt.Sprintf("%sの呪いが解けた", weapon.GetName()), 0.4)
			}
			weapon.Sharpness++
			// 装備中の武器の強化値上昇を攻撃力へ即時反映する
			g.state.Player.AttackPower++
			g.EnqueueMessage(fmt.Sprintf("%sは強くなった", weapon.GetName()), 0.4)
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// 盾強化のカードは装備中の盾の強化値を1上げ、呪いも解く。
var reinforceArmor = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した", item.GetName()),
		Execute: func(g *Game) {
			armor := g.state.Player.EquippedArmor
			if armor == nil {
				g.EnqueueMessage("しかし何も起こらなかった", 0.4)
				return
			}
			if armor.Cursed {
				armor.Cursed = false
				g.EnqueueMessage(fmt.Sprintf("%sの呪いが解けた", armor.GetName()), 0.4)
			}
			armor.Sharpness++
			// 装備中の盾の強化値上昇を防御力へ即時反映する
			g.state.Player.DefensePower++
			g.EnqueueMessage(fmt.Sprintf("%sは強くなった", armor.GetName()), 0.4)
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// rustProofPlayerEquipment は装備中の武器・盾を錆びない状態にし、対象の名前を返す
func rustProofPlayerEquipment(player *Player) []string {
	var protected []string
	if weapon := player.EquippedWeapon; weapon != nil && !weapon.RustProof {
		weapon.RustProof = true
		protected = append(protected, weapon.GetName())
	}
	if armor := player.EquippedArmor; armor != nil && !armor.RustProof {
		armor.RustProof = true
		protected = append(protected, armor.GetName())
	}
	return protected
}

// さび止めのカードは装備中の武器と盾を錆びない状態にする。
var rustProofCard = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した", item.GetName()),
		Execute: func(g *Game) {
			protected := rustProofPlayerEquipment(&g.state.Player)
			if len(protected) == 0 {
				g.EnqueueMessage("しかし何も起こらなかった", 0.4)
				return
			}
			for _, name := range protected {
				g.EnqueueMessage(fmt.Sprintf("%sは錆びなくなった", name), 0.4)
			}
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// あかりのカードはフロアの地形と敵、アイテムをミニマップへ表示する。
var revealFloor = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した。フロアが明るく照らされた", item.GetName()),
		Execute: func(g *Game) {
			g.revealHiddenTerrain()
			for y := range g.state.Map {
				for x := range g.state.Map[y] {
					g.state.Map[y][x].Visited = true
				}
			}
			for i := range g.state.Enemies {
				g.state.Enemies[i].PlayerDiscovered = true
				g.state.Enemies[i].ShowOnMiniMap = true
			}
			for _, floorItem := range g.state.Items {
				floorItem.SetPlayerDiscovered(true)
				floorItem.SetShowOnMiniMap(true)
			}
			g.miniMapDirty = true
			g.isActioned = true
		},
	})
	removeUsedItem(g, isInventoryItem)
}

func rollVacuumSlashDamage(intn func(int) int) int {
	return 12 + intn(13)
}

func isInRoomWideEffect(playerX, playerY, targetX, targetY int, rooms []Room) bool {
	if isInsideRoom(playerX, playerY, rooms) {
		return isSameRoom(playerX, playerY, targetX, targetY, rooms)
	}
	return abs(targetX-playerX) <= 1 && abs(targetY-playerY) <= 1
}

// 真空斬りのカードは部屋全体、通路では周囲1マスの敵へダメージを与える。
var vacuumSlash = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	damage := rollVacuumSlashDamage(g.randIntn)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した。真空の刃が走った", item.GetName()),
		Execute: func(g *Game) {
			playerX, playerY := g.state.Player.GetPosition()
			survivors := g.state.Enemies[:0]
			defeatedExperience := 0
			var defeated []Enemy
			for i := range g.state.Enemies {
				enemy := g.state.Enemies[i]
				if !isInRoomWideEffect(playerX, playerY, enemy.X, enemy.Y, g.rooms) {
					survivors = append(survivors, enemy)
					continue
				}

				enemy.Health -= damage
				wakeFromSleep(&enemy.StatusAilments)
				clearStatus(&enemy.StatusAilments, StatusParalysis)
				if enemy.Health <= 0 {
					if !statusActive(enemy.StatusAilments, StatusDecoy) {
						defeatedExperience += enemy.ExperiencePoints
					}
					g.EnqueueMessage(fmt.Sprintf("%sを倒した。", enemy.Name), 0.4)
					defeated = append(defeated, enemy)
					continue
				}
				survivors = append(survivors, enemy)
			}
			g.state.Enemies = survivors
			for _, enemy := range defeated {
				g.onEnemyDefeated(enemy)
			}
			if defeatedExperience > 0 {
				g.state.Player.ExperiencePoints += defeatedExperience
				g.state.Player.checkLevelUp(g)
			}
			g.miniMapDirty = true
			g.isActioned = true
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// --- フロアや部屋へ作用するカード ---

// createFloorItemを直接参照するとitemTemplatesとの初期化循環になるため、init時に間接参照を張る
var newRandomFloorItem func(x, y, floor int, difficulty Difficulty, intn func(int) int) Item

func init() {
	newRandomFloorItem = createFloorItem
}

// pickFreeCellsInRoom は部屋の内側から isFree を満たすマスを最大 count 個ランダムに選ぶ。
func pickFreeCellsInRoom(room Room, isFree func(x, y int) bool, count int, intn func(int) int) []Coordinate {
	var free []Coordinate
	for y := room.Y + 1; y < room.Y+room.Height-1; y++ {
		for x := room.X + 1; x < room.X+room.Width-1; x++ {
			if isFree(x, y) {
				free = append(free, Coordinate{X: x, Y: y})
			}
		}
	}
	// Fisher-Yatesシャッフルで順序をランダム化する
	for i := len(free) - 1; i > 0; i-- {
		j := intn(i + 1)
		free[i], free[j] = free[j], free[i]
	}
	if count > len(free) {
		count = len(free)
	}
	return free[:count]
}

// モンスターハウスのカードは今いる部屋を敵とアイテムで埋め尽くす。通路では不発。
var summonMonsterHouse = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した", item.GetName()),
		Execute: func(g *Game) {
			playerX, playerY := g.state.Player.GetPosition()
			room := getPlayerRoom(playerX, playerY, g.rooms)
			if room == nil {
				g.EnqueueMessage("しかし何も起こらなかった", 0.4)
				return
			}
			isFree := func(x, y int) bool {
				if x == playerX && y == playerY {
					return false
				}
				if g.state.Map[y][x].Blocked || g.state.Map[y][x].Type == "stairs" {
					return false
				}
				for _, enemy := range g.state.Enemies {
					if enemy.X == x && enemy.Y == y {
						return false
					}
				}
				for _, floorItem := range g.state.Items {
					itemX, itemY := floorItem.GetPosition()
					if itemX == x && itemY == y {
						return false
					}
				}
				return true
			}
			enemyCount, itemCount := monsterHouseCounts(0, true, g.randIntn)
			cells := pickFreeCellsInRoom(*room, isFree, enemyCount+itemCount, g.randIntn)
			if len(cells) == 0 {
				g.EnqueueMessage("しかし何も起こらなかった", 0.4)
				return
			}
			for i, cell := range cells {
				if i < enemyCount {
					// 階層に応じた敵を起きた状態で配置する（封じられた系統しかいない場合は配置しない）
					monsterID := g.SelectMonsterForSpawn()
					if monsterID < 0 {
						continue
					}
					enemy := CreateEnemyByID(monsterID, cell.X, cell.Y)
					prepareSpawnedEnemy(&enemy, g.Floor, g.randIntn)
					enemy.PlayerDiscovered = true
					enemy.ShowOnMiniMap = true
					g.state.Enemies = append(g.state.Enemies, enemy)
				} else {
					g.state.Items = append(g.state.Items, newRandomFloorItem(cell.X, cell.Y, g.Floor, g.difficulty, g.randIntn))
				}
			}
			g.EnqueueMessage("モンスターハウスだ！", 0.5)
			g.miniMapDirty = true
			g.isActioned = true
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// 敵倍速のカードで敵が倍速になるターン数
const enemyHasteCardTurns = 20

// 敵倍速のカードはフロアにいる敵全員を倍速状態にする。
var hasteAllEnemies = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した", item.GetName()),
		Execute: func(g *Game) {
			if len(g.state.Enemies) == 0 {
				g.EnqueueMessage("しかし何も起こらなかった", 0.4)
				return
			}
			for i := range g.state.Enemies {
				g.applyEnemyStatus(i, StatusHaste, enemyHasteCardTurns)
			}
			g.EnqueueMessage("フロアの敵たちの動きが速くなった！", 0.5)
			g.isActioned = true
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// 地図忘却のカードはフロアの探索情報とミニマップ表示をすべて消す。
var forgetFloorMap = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した。フロアの記憶が頭から消えてしまった", item.GetName()),
		Execute: func(g *Game) {
			for y := range g.state.Map {
				for x := range g.state.Map[y] {
					g.state.Map[y][x].Visited = false
				}
			}
			for i := range g.state.Enemies {
				g.state.Enemies[i].PlayerDiscovered = false
				g.state.Enemies[i].ShowOnMiniMap = false
			}
			for _, floorItem := range g.state.Items {
				floorItem.SetPlayerDiscovered(false)
				floorItem.SetShowOnMiniMap(false)
			}
			g.miniMap = nil
			g.miniMapDirty = true
			g.isActioned = true
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// 拾得禁止のカードはフロアを移るまで床のアイテムを拾えなくする。
var banItemPickup = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した。フロアを移るまで床のアイテムを拾えなくなった", item.GetName()),
		Execute: func(g *Game) {
			g.pickupBanned = true
			g.isActioned = true
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// makeBigRoom はマップ全体をひとつの大部屋へ変換し、その部屋情報を返す。
// 外周は壁になり、内側は階段を残してすべて床になる。
func makeBigRoom(mapGrid [][]Tile) Room {
	height := len(mapGrid)
	width := len(mapGrid[0])
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			visited := mapGrid[y][x].Visited
			if x == 0 || x == width-1 || y == 0 || y == height-1 {
				mapGrid[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true, Visited: visited}
			} else if mapGrid[y][x].Type != "stairs" {
				mapGrid[y][x] = Tile{Type: "floor", Blocked: false, BlockSight: false, Visited: visited}
			}
		}
	}
	room := Room{ID: 0, X: 0, Y: 0, Width: width, Height: height}
	setRoomCenter(&room)
	return room
}

// 大部屋のカードはフロア全体をひとつの大部屋にする。
var expandFloorToBigRoom = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した。壁が崩れ、フロア全体がひとつの大部屋になった", item.GetName()),
		Execute: func(g *Game) {
			g.rooms = []Room{makeBigRoom(g.state.Map)}
			g.miniMap = nil
			g.miniMapDirty = true
			g.isActioned = true
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// 罠のカードで増える罠の数（5〜8個）
func rollExtraTrapCount(intn func(int) int) int {
	return 5 + intn(4)
}

// 罠のカードはフロアの部屋に未発見の罠を追加する。
var increaseFloorTraps = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した。なんだか嫌な予感がする…", item.GetName()),
		Execute: func(g *Game) {
			playerX, playerY := g.state.Player.GetPosition()
			count := rollExtraTrapCount(g.randIntn)
			added := 0
			for attempt := 0; attempt < 100 && added < count; attempt++ {
				room := g.rooms[g.randIntn(len(g.rooms))]
				x := g.randIntn(room.Width-2) + room.X + 1
				y := g.randIntn(room.Height-2) + room.Y + 1
				if x == playerX && y == playerY {
					continue
				}
				if g.state.Map[y][x].Blocked || g.state.Map[y][x].Type == "stairs" {
					continue
				}
				occupied := false
				for _, trap := range g.state.MapTraps {
					if trap.X == x && trap.Y == y {
						occupied = true
						break
					}
				}
				if occupied {
					continue
				}
				g.state.MapTraps = append(g.state.MapTraps, createMapTrapByID(pickFloorTrapID(g.Floor, g.randIntn), x, y))
				added++
			}
			g.isActioned = true
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// --- プレイヤー自身へ作用するカード ---

// splitEnemiesByExplosion は自爆の爆発（周囲1マス）に巻き込まれる敵と生き残る敵を分ける。
func splitEnemiesByExplosion(playerX, playerY int, enemies []Enemy) (survivors []Enemy, destroyed []string) {
	for _, enemy := range enemies {
		if abs(enemy.X-playerX) <= 1 && abs(enemy.Y-playerY) <= 1 && !MonsterDefinitions[enemy.ID].IsBoss {
			destroyed = append(destroyed, enemy.Name)
			continue
		}
		survivors = append(survivors, enemy)
	}
	return survivors, destroyed
}

// 自爆のカードは自分のHPを半分にする代わりに周囲1マスの敵を消し飛ばす。
// 消し飛んだ敵の経験値は入らない。
var selfDestruct = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した。カードが激しく輝いた", item.GetName()),
		Execute: func(g *Game) {
			playerX, playerY := g.state.Player.GetPosition()
			survivors, destroyed := splitEnemiesByExplosion(playerX, playerY, g.state.Enemies)
			damage := mineTrapDamage(g.state.Player.Health)
			defense := g.resolvePlayerShieldDefense(damage, EnemyDamageExplosion, EnemyAttackEnvironment)
			message := fmt.Sprintf("大爆発が起こり、%dダメージを受けた！", defense.Damage)
			if defense.Resisted {
				message = fmt.Sprintf("大爆発が起こったが、%sが威力を抑えて%dダメージを受けた！", g.state.Player.EquippedArmor.Name, defense.Damage)
			}
			g.Enqueue(Action{
				Duration: 0.5,
				Message:  message,
				Execute: func(g *Game) {
					g.state.Enemies = survivors
					g.breakCrackedAround(playerX, playerY, 1)
					g.applyPlayerShieldDefense(defense, DamageSource{Kind: DamageSourceSelfDestruct, Detail: item.GetName()}, -1, -1, -1)
				},
			})
			for _, name := range destroyed {
				g.EnqueueMessage(fmt.Sprintf("%sは爆発に巻き込まれて消し飛んだ", name), 0.4)
			}
			g.miniMapDirty = true
			g.isActioned = true
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// 口封じのカードでプレイヤーの口が封じられるターン数
const mouthSealCardTurns = 15

// 口封じのカードはしばらくの間カード・薬・食料を使えなくする。
var sealPlayerMouth = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した", item.GetName()),
		Execute: func(g *Game) {
			g.isActioned = true
		},
	})
	g.enqueuePlayerStatus(StatusMouthSeal, mouthSealCardTurns, StackDefault, "", false)
	removeUsedItem(g, isInventoryItem)
}

// パワーアップのカードで上昇するパワーの量
const powerBoostAmount = 3

// boostPower はパワーアップのカードの結果を返す。
// パワーが満タンのときは最大パワーが1上がり、パワーも満タンになる。
func boostPower(power, maxPower int) (newPower, newMaxPower int) {
	if power >= maxPower {
		maxPower++
		return maxPower, maxPower
	}
	power += powerBoostAmount
	if power > maxPower {
		power = maxPower
	}
	return power, maxPower
}

// パワーアップのカードはパワーを上昇させる。
var powerUpCard = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した。力がみなぎってきた", item.GetName()),
		Execute: func(g *Game) {
			player := &g.state.Player
			newPower, newMaxPower := boostPower(player.Power, player.MaxPower)
			var message string
			if newMaxPower > player.MaxPower {
				message = fmt.Sprintf("最大パワーが%d上昇した", newMaxPower-player.MaxPower)
			} else {
				message = fmt.Sprintf("パワーが%d上昇した", newPower-player.Power)
			}
			player.Power, player.MaxPower = newPower, newMaxPower
			g.EnqueueMessage(message, 0.4)
			g.isActioned = true
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// 完全回復のカードでHPが満タンだった場合に上がる最大HPの量
const fullHealMaxHealthBonus = 5

// fullHealResult は完全回復のカードの回復結果を返す。
// HPが満タンのときは最大HPが上昇する。
func fullHealResult(health, maxHealth int) (newHealth, newMaxHealth int) {
	if health >= maxHealth {
		maxHealth += fullHealMaxHealthBonus
	}
	return maxHealth, maxHealth
}

// 完全回復のカードはHPを全回復する。満タンなら最大HPが上がる（パワーは戻らない）。
var fullHealCard = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した。温かい光が体を包んだ", item.GetName()),
		Execute: func(g *Game) {
			player := &g.state.Player
			wasFull := player.Health >= player.MaxHealth
			healed := player.MaxHealth - player.Health
			before := player.Health
			player.Health, player.MaxHealth = fullHealResult(player.Health, player.MaxHealth)
			if statusActive(player.StatusAilments, StatusNoHealing) {
				player.Health = before
				healed = 0
			}
			if wasFull {
				g.EnqueueMessage(fmt.Sprintf("最大HPが%d上昇した", fullHealMaxHealthBonus), 0.4)
			} else {
				g.EnqueueMessage(fmt.Sprintf("HPが%d回復した", healed), 0.4)
			}
			g.isActioned = true
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// --- 特殊な使用方法を持つカード ---

// 特殊カードの名前定数。効果の判定に使う。
const (
	blankCardName     = "白紙のカード"
	genocideCardName  = "ジェノサイドのカード"
	sanctuaryCardName = "聖域のカード"
)

// blankCardOptionIDs は白紙のカードへ書き込めるカードテンプレートIDを昇順で返す。
func blankCardOptionIDs() []int {
	ids := []int{}
	for id, template := range itemTemplates {
		if template.ItemType == "Card" && template.Name != blankCardName {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}

// writeCardEffect は白紙のカードへ指定テンプレートのカード効果を書き込む。
// カード以外のテンプレートや白紙自身は書き込めない。
func writeCardEffect(card *Card, templateID int) bool {
	template, ok := itemTemplates[templateID]
	if !ok || template.ItemType != "Card" || template.Name == blankCardName {
		return false
	}
	card.ID = template.ID
	card.Type = template.Type
	card.Name = template.Name
	card.Description = template.Description
	card.Char = template.Char
	card.UseActions = template.UseActions
	return true
}

// blankCardScrollTop は選択カーソルが見えるようにリストの先頭表示位置を返す。
func blankCardScrollTop(index, count, visible int) int {
	if count <= visible {
		return 0
	}
	top := index - visible + 1
	if top < 0 {
		top = 0
	}
	if maxTop := count - visible; top > maxTop {
		top = maxTop
	}
	return top
}

// 白紙のカードは使用すると書き込むカードの選択ウィンドウを開く。
// 書き込むまで消費されず、キャンセルするとターンも消費しない。
var blankCard = func(g *Game) {
	item, _ := determineItemSource(g)
	card, ok := item.(*Card)
	if !ok {
		return
	}
	g.blankCardTarget = card
	g.blankCardIndex = 0
	g.showBlankCardMenu = true
	g.blankCardJustOpened = true
	g.showInventory = false
}

// closeBlankCardMenu は白紙のカードの選択ウィンドウを閉じて状態をリセットする。
func (g *Game) closeBlankCardMenu() {
	g.showBlankCardMenu = false
	g.blankCardJustOpened = false
	g.blankCardTarget = nil
	g.blankCardIndex = 0
}

// ジェノサイドのカードは投げて使うため、読んでも効果はない（消費もしない）。
var genocideCardHint = func(g *Game) {
	item, _ := determineItemSource(g)
	g.EnqueueMessage(fmt.Sprintf("%sは敵に投げ当てると効果を発揮するようだ", item.GetName()), 0.5)
}

// 聖域のカードは床に置いて使うため、読んでも効果はない（消費もしない）。
var sanctuaryCardHint = func(g *Game) {
	item, _ := determineItemSource(g)
	g.EnqueueMessage(fmt.Sprintf("%sは床に置くと効果を発揮するようだ", item.GetName()), 0.5)
}

// monsterFamilyIDs は指定した敵と同系統（基本種と上位種）の敵IDを昇順で返す。
func monsterFamilyIDs(monsterID int) []int {
	// 上位種化の対応を両方向にたどり、基本種〜第3段階まで同じ系統として集める
	family := map[int]bool{monsterID: true}
	for changed := true; changed; {
		changed = false
		for baseID, upperID := range MonsterLevelUpTable {
			if family[baseID] != family[upperID] {
				family[baseID], family[upperID] = true, true
				changed = true
			}
		}
	}
	ids := make([]int, 0, len(family))
	for id := range family {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// applyGenocide は指定した敵と同系統の敵をフロアから消し去り、以後の出現を封じる。
// 消し去った敵の数を返す。経験値は入らない。
func (g *Game) applyGenocide(monsterID int) int {
	if MonsterDefinitions[monsterID].IsBoss {
		return 0
	}
	if g.genocidedMonsterIDs == nil {
		g.genocidedMonsterIDs = map[int]bool{}
	}
	for _, id := range monsterFamilyIDs(monsterID) {
		g.genocidedMonsterIDs[id] = true
	}
	removed := 0
	survivors := g.state.Enemies[:0]
	for i := range g.state.Enemies {
		if g.genocidedMonsterIDs[g.state.Enemies[i].ID] && !MonsterDefinitions[g.state.Enemies[i].ID].IsBoss {
			removed++
			continue
		}
		survivors = append(survivors, g.state.Enemies[i])
	}
	g.state.Enemies = survivors
	g.miniMapDirty = true
	return removed
}

// removeGenocidedEnemies はジェノサイドで封じた系統の敵をフロア生成直後に取り除く。
func (g *Game) removeGenocidedEnemies() {
	if len(g.genocidedMonsterIDs) == 0 {
		return
	}
	survivors := g.state.Enemies[:0]
	for i := range g.state.Enemies {
		if g.genocidedMonsterIDs[g.state.Enemies[i].ID] && !MonsterDefinitions[g.state.Enemies[i].ID].IsBoss {
			continue
		}
		survivors = append(survivors, g.state.Enemies[i])
	}
	g.state.Enemies = survivors
}

// isSanctuaryItem は床に効果を発揮する聖域のカードかどうかを返す。
func isSanctuaryItem(item Item) bool {
	card, ok := item.(*Card)
	return ok && card.Name == sanctuaryCardName
}

// isStuckSanctuaryItem は床に貼りついて拾えない聖域のカードかどうかを返す。
func isStuckSanctuaryItem(item Item) bool {
	card, ok := item.(*Card)
	return ok && card.Name == sanctuaryCardName && card.Stuck
}

// sanctuaryAt は指定座標の床に聖域のカードがあるかどうかを返す。
func (g *Game) sanctuaryAt(x, y int) bool {
	for _, item := range g.state.Items {
		itemX, itemY := item.GetPosition()
		if itemX == x && itemY == y && isSanctuaryItem(item) {
			return true
		}
	}
	return false
}

// playerOnSanctuary はプレイヤーが聖域のカードの上にいるかどうかを返す。
func (g *Game) playerOnSanctuary() bool {
	return g.sanctuaryAt(g.state.Player.X, g.state.Player.Y)
}

// splitEnemiesByRoomWideEffect は部屋全体効果（通路では周囲1マス）の対象と対象外の敵を分ける。
func splitEnemiesByRoomWideEffect(playerX, playerY int, enemies []Enemy, rooms []Room) (survivors []Enemy, destroyed []string) {
	for _, enemy := range enemies {
		if isInRoomWideEffect(playerX, playerY, enemy.X, enemy.Y, rooms) && !MonsterDefinitions[enemy.ID].IsBoss {
			destroyed = append(destroyed, enemy.Name)
			continue
		}
		survivors = append(survivors, enemy)
	}
	return survivors, destroyed
}

// 全滅のカードは同じ部屋（通路では周囲1マス）の敵をすべて消し去る。
// 消し去った敵の経験値は入らない。
var annihilateEnemiesInRoom = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	g.Enqueue(Action{
		Duration: 0.4,
		Message:  fmt.Sprintf("%sを使用した。まばゆい光があたりを包んだ", item.GetName()),
		Execute: func(g *Game) {
			playerX, playerY := g.state.Player.GetPosition()
			survivors, destroyed := splitEnemiesByRoomWideEffect(playerX, playerY, g.state.Enemies, g.rooms)
			if len(destroyed) == 0 {
				g.EnqueueMessage("しかし何も起こらなかった", 0.4)
				return
			}
			g.state.Enemies = survivors
			for _, name := range destroyed {
				g.EnqueueMessage(fmt.Sprintf("%sは光に包まれて消え去った", name), 0.4)
			}
			g.miniMapDirty = true
			g.isActioned = true
		},
	})
	removeUsedItem(g, isInventoryItem)
}

// executeItemIdentify は真実の眼のカードの対象を確定する。低確率で持ち物すべてを識別し
// （識別の巻物相当）、外れれば選んだ道具だけを識別する。どちらでもカードを消費して1ターン進む。
func (g *Game) executeItemIdentify() {
	g.showInventory = false
	cardIndex := g.tmpSelectedItemIndex
	fromInventory := !g.GroundItemActioned
	var card Item
	if fromInventory {
		card = g.state.Player.Inventory[cardIndex]
	} else {
		card = g.state.Items[cardIndex]
	}
	item := g.state.Player.Inventory[g.selectedItemIndex]

	if rollIdentifyAll(g.randIntn) {
		g.Enqueue(Action{
			Duration: 0.5,
			Message:  fmt.Sprintf("%sが強く輝いた。持ち物すべての正体が明らかになった", card.GetName()),
			Execute: func(g *Game) {
				identifyAllInventory(g)
			},
		})
	} else {
		if identifiableItem, ok := item.(Identifiable); ok {
			g.Enqueue(Action{
				Duration:     0.5,
				ItemName:     identifiableItem.GetName(),
				Message:      fmt.Sprintf("%sを識別した。", identifiableItem.GetName()),
				Execute:      func(g *Game) {},
				IsIdentified: identifiableItem.GetIdentified(),
			})
			identifiableItem.SetIdentified(true)
		}

		g.Enqueue(Action{
			Duration:     0.5,
			ItemName:     getItemNameWithSharpness(item),
			Message:      fmt.Sprintf("アイテムの正体は%sだった。", getItemNameWithSharpness(item)),
			Execute:      func(g *Game) {},
			IsIdentified: true,
		})
	}

	if fromInventory {
		g.state.Player.Inventory = append(g.state.Player.Inventory[:cardIndex], g.state.Player.Inventory[cardIndex+1:]...)
	} else {
		g.state.Items = append(g.state.Items[:cardIndex], g.state.Items[cardIndex+1:]...)
	}

	g.tmpSelectedItemIndex = -1
	g.selectedItemIndex = 0
	g.useIdentifyItem = false
	g.GroundItemActioned = false
	g.isActioned = true
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
						g.applyEnemyStatus(i, StatusSleep, 10)              // 10ターン睡眠
						markHasteOnWake(&g.state.Enemies[i].StatusAilments) // 目覚めた時に倍速化する
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
						g.applyEnemyStatus(i, StatusSleep, 10)              // 10ターン睡眠
						markHasteOnWake(&g.state.Enemies[i].StatusAilments) // 目覚めた時に倍速化する
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
						g.applyEnemyStatus(i, StatusConfusion, 10) // 10ターン混乱
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
						g.applyEnemyStatus(i, StatusConfusion, 10) // 10ターン混乱
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
						g.applyEnemyStatus(i, StatusBlind, 10) // 10ターン目潰し状態
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
						g.applyEnemyStatus(i, StatusBlind, 10) // 10ターン目潰し状態
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
					g.applyEnemyStatus(i, StatusParalysis, 1) // 金縛り状態（最終ボスは受け付けない）
					enemiesParalyzed++
				}
			}

			if enemiesParalyzed > 0 {
				g.debugf("周囲の敵%d体を金縛り状態にした", enemiesParalyzed)
			} else {
				g.debugf("周囲に敵がいない")
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
	// 飲んだ薬は装備能力による無効化だけを見る（状態異常耐性の盾は効かない）。
	guard, immune := g.playerImmunitySource(StatusSleep)
	g.EnqueueMessage(fmt.Sprintf("%sを飲んだ", item.GetName()), 0.4)
	g.enqueuePlayerStatus(StatusSleep, 10, StackDefault, guard, immune) // 10ターン睡眠

	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

// 混乱ポーション効果関数 - プレイヤーを混乱状態にする
var confusionPotion = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	// 飲んだ薬は装備能力による無効化だけを見る（状態異常耐性の盾は効かない）。
	guard, immune := g.playerImmunitySource(StatusConfusion)
	g.EnqueueMessage(fmt.Sprintf("%sを飲んだ", item.GetName()), 0.4)
	g.enqueuePlayerStatus(StatusConfusion, 10, StackDefault, guard, immune) // 10ターン混乱

	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

// 目潰しポーション効果関数 - プレイヤーを目潰し状態にする
var blindPotion = func(g *Game) {
	item, isInventoryItem := determineItemSource(g)
	// 飲んだ薬は装備能力による無効化だけを見る（状態異常耐性の盾は効かない）。
	guard, immune := g.playerImmunitySource(StatusBlind)
	g.EnqueueMessage(fmt.Sprintf("%sを飲んだ", item.GetName()), 0.4)
	g.enqueuePlayerStatus(StatusBlind, 30, StackDefault, guard, immune) // 30ターン目潰し状態

	// アイテムの使用後の処理
	removeUsedItem(g, isInventoryItem)
}

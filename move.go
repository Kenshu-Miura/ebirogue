//go:build !test

package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
	"math/rand"
)

func (g *Game) IncrementMoveCount() {
	g.moveCount++
	if g.state.Player.StatusAilments.Poison > 0 {
		g.state.Player.Health = max(0, g.state.Player.Health-2)
		g.state.Player.checkDeath(g)
	}
	// 状態異常のターン数を減らす
	g.decrementStatusAilments()

	// 満腹度が0の場合の処理
	if g.state.Player.Satiety == 0 {
		// 満腹度0の場合、毎ターンHPが1減る
		g.state.Player.Health -= 1
		if g.state.Player.Health < 0 {
			g.state.Player.Health = 0
		}
		// 死亡チェック
		g.state.Player.checkDeath(g)
	} else {
		// 満腹度が0でない場合のみHP自動回復
		if g.moveCount%5 == 0 && g.moveCount != 0 {
			// Recover 1 HP for the player
			g.state.Player.Health += 1
			// Ensure player's health does not exceed MaxHealth
			if g.state.Player.Health > g.state.Player.MaxHealth {
				g.state.Player.Health = g.state.Player.MaxHealth
			}
		}
	}

	armorAbilities := []EquipmentAbilityID(nil)
	if g.state.Player.EquippedArmor != nil {
		armorAbilities = g.state.Player.EquippedArmor.Abilities
	}
	if shouldReduceSatiety(g.moveCount, armorAbilities) {
		g.state.Player.Satiety -= 1
		if g.state.Player.Satiety < 0 {
			g.state.Player.Satiety = 0
		}
	}
}

// プレイヤーが混乱状態の時のランダム移動処理
func (g *Game) movePlayerConfused() bool {
	// 8方向のランダムな移動先を選択
	directions := getDirections()
	direction := directions[rand.Intn(len(directions))]
	newX := g.state.Player.X + direction.dx
	newY := g.state.Player.Y + direction.dy

	// 範囲チェック
	if newX < 0 || newX >= len(g.state.Map[0]) || newY < 0 || newY >= len(g.state.Map) {
		g.EnqueueMessage("混乱している...", 0.4)
		return false // 移動失敗時はターンを消費しない
	}

	// プレイヤーの方向を設定
	g.state.Player.Direction = determineDirection(direction.dx, direction.dy)

	// 移動先に敵がいる場合、移動失敗として扱う
	for _, enemy := range g.state.Enemies {
		if enemy.X == newX && enemy.Y == newY {
			g.EnqueueMessage("混乱している...", 0.4)
			return false // 移動失敗時はターンを消費しない
		}
	}

	// 移動先が通行可能な場合、移動
	if !g.state.Map[newY][newX].Blocked {
		g.EnqueueMessage("混乱して移動した", 0.4)
		g.state.Player.X = newX
		g.state.Player.Y = newY
		// アニメーション用に実際の移動方向を保存
		g.dx, g.dy = direction.dx, direction.dy
		return true // 移動成功時はターンを消費
	}

	// 移動失敗時
	g.EnqueueMessage("混乱している...", 0.4)
	return false // 移動失敗時はターンを消費しない
}

func (p *Player) checkLevelUp(g *Game) {
	if p.Level < 10 && p.Level < len(levelExpRequirements) && p.ExperiencePoints >= levelExpRequirements[p.Level] {
		p.Level++ // レベルアップ
		// 必要に応じて他のプレイヤーステータスをアップデート
		p.MaxHealth += 10

		// レベルアップメッセージを表示
		levelUpMessage := fmt.Sprintf("海老さんはレベル%dに上がった", p.Level)
		levelUpAction := Action{
			Duration: 1.0,
			Message:  levelUpMessage,
			Execute:  func(g *Game) {}, // 何もしない
		}
		g.ActionQueue.Queue = append(g.ActionQueue.Queue, levelUpAction)
	}
}

// プレイヤーの死亡チェック
func (p *Player) checkDeath(g *Game) {
	if p.Health <= 0 && !g.playerDead {
		g.playerDead = true
		g.fadeOutProgress = 0.0
		g.gameResetTimer = 6.0 // 6秒後にリセット（攻撃処理待ち+メッセージ1秒+フェードアウト1秒+待機2秒）

		// 死亡したら中断データを削除する（復活による再開を防ぐ）
		deleteSaveFile()

		// 死亡メッセージは後でActionQueueが空になってから追加する
	}
}

// ゲームリセット機能
func (g *Game) resetGame() {
	// プレイヤーの初期化（座標は後でGenerateRandomMapで設定される）
	player := Player{
		Name:             "海老さん",
		Entity:           Entity{Char: '@'}, // X、Yは0のまま（GenerateRandomMapで設定される）
		Health:           100,
		MaxHealth:        100,
		Satiety:          100,
		MaxSatiety:       100,
		Inventory:        []Item{},
		MaxInventory:     20,
		AttackPower:      3,
		DefensePower:     3,
		ExperiencePoints: 0,
		Level:            1,
		Power:            8,
		MaxPower:         8,
		Direction:        Up,
		Cash:             0,
	}

	// 新しいマップを生成（プレイヤーの座標も部屋の中に設定される）
	mapGrid, enemies, items, _, rooms, traps := GenerateRandomMap(70, 70, 0, &player)

	// ゲーム状態をリセット
	g.state = GameState{
		Map:      mapGrid,
		Player:   player,
		Enemies:  enemies,
		Items:    items,
		MapTraps: traps,
	}

	// その他のゲーム状態をリセット
	g.rooms = rooms // 部屋情報を更新
	g.pickupBanned = false
	g.playerDead = false
	g.deathMessageAdded = false
	g.fadeOutProgress = 0.0
	g.fadeInProgress = 0.0
	g.gameResetTimer = 0.0
	g.starvationBlinkTimer = 0.0
	g.ActionQueue.Queue = []Action{}
	g.moveCount = 0
	g.Animating = false
	g.AnimationProgress = 0.0
	g.ActionDurationCounter = 0.0
	g.isActioned = false
	g.isCombatActive = false

	// インベントリ拡張の状態をリセット
	g.inventoryFilter = CategoryAll
	g.customNames = map[int]string{}
	g.showNameInput = false

	// モンスター湧きシステム再初期化
	g.InitializeSpawnSystem()
}

func isSameRoom(x1, y1, x2, y2 int, rooms []Room) bool {
	var room1, room2 Room
	foundRoom1, foundRoom2 := false, false // New variables to track if room1 and room2 are found

	//log.Printf("Checking if points (%d, %d) and (%d, %d) are in the same room\n", x1, y1, x2, y2) // Log input points
	for _, room := range rooms {
		// Adjust the conditions to check if the points are within the inner boundaries of the room
		if x1 > room.X && x1 < room.X+room.Width-1 && y1 > room.Y && y1 < room.Y+room.Height-1 {
			room1 = room
			foundRoom1 = true // Set foundRoom1 to true if room1 is found
		}
		if x2 > room.X && x2 < room.X+room.Width-1 && y2 > room.Y && y2 < room.Y+room.Height-1 {
			room2 = room
			foundRoom2 = true // Set foundRoom2 to true if room2 is found
		}
	}

	// If either point is not in a room, return false
	if !foundRoom1 || !foundRoom2 {
		return false
	}

	result := room1.ID == room2.ID

	return result
}

func (g *Game) CheatMovePlayer(dx, dy int) bool {
	// dx と dy が両方とも0の場合、移動は発生していない
	if dx == 0 && dy == 0 {
		return false
	}

	newPX := g.state.Player.X + dx
	newPY := g.state.Player.Y + dy

	// Determine the direction based on the change in position
	g.state.Player.Direction = determineDirection(dx, dy)

	// 敵との戦闘チェック
	if g.CheckForEnemies(newPX, newPY) {
		// 戦闘が発生した場合、プレイヤーは移動しない
		return false
	}

	g.state.Player.X = newPX
	g.state.Player.Y = newPY
	g.isActioned = true
	g.PickupItem()
	return true

}

func (g *Game) MovePlayer(dx, dy int) bool {
	// プレイヤーが睡眠状態の場合、移動できない
	if g.state.Player.StatusAilments.Sleep > 0 {
		// 睡眠メッセージを表示
		action := Action{
			Duration: 0.4,
			Message:  "眠っている...",
			Execute:  func(g *Game) {},
		}
		g.Enqueue(action)
		return true // 睡眠状態でもターンを消費する
	}

	// dx と dy が両方とも0の場合、移動は発生していない
	if dx == 0 && dy == 0 {
		return false
	}

	// プレイヤーが混乱状態の場合、入力方向に関係なくランダムな8方向に移動
	if g.state.Player.StatusAilments.Confusion > 0 {
		return g.movePlayerConfused()
	}

	newPX := g.state.Player.X + dx
	newPY := g.state.Player.Y + dy

	for _, enemy := range g.state.Enemies {
		if enemy.X == newPX && enemy.Y == newPY {
			g.state.Player.Direction = determineDirection(dx, dy)
			return false
		}
	}

	// マップ範囲内およびブロックされていないタイル上にあることを確認
	if newPX >= 0 && newPX < len(g.state.Map[0]) && newPY >= 0 && newPY < len(g.state.Map) && !g.state.Map[newPY][newPX].Blocked {
		// Determine the direction based on the change in position
		g.state.Player.Direction = determineDirection(dx, dy)

		g.state.Player.X = newPX
		g.state.Player.Y = newPY
		g.isActioned = true

		// 新しい位置に罠があるかチェック
		g.checkForTrapAtPosition(newPX, newPY)

		g.PickupItem()
		return true
	}
	return false
}

func isOccupied(g *Game, x, y int) bool {
	for _, enemy := range g.state.Enemies {
		if enemy.X == x && enemy.Y == y {
			return true
		}
	}
	// Check if the player is at the specified coordinates
	if g.state.Player.X == x && g.state.Player.Y == y {
		return true
	}
	return false
}

// 状態異常のターン数を減らす関数
func (g *Game) decrementStatusAilments() {
	// プレイヤーの状態異常を減らす
	if g.state.Player.StatusAilments.Confusion > 0 {
		g.state.Player.StatusAilments.Confusion--
	}
	if g.state.Player.StatusAilments.Sleep > 0 {
		g.state.Player.StatusAilments.Sleep--
		// 睡眠状態が治った時のメッセージ
		if g.state.Player.StatusAilments.Sleep == 0 {
			action := Action{
				Duration: 0.4,
				Message:  "目を覚ました",
				Execute:  func(g *Game) {},
			}
			g.Enqueue(action)
		}
	}
	if g.state.Player.StatusAilments.Blind > 0 {
		g.state.Player.StatusAilments.Blind--
		// 目潰し状態が治った時のメッセージ
		if g.state.Player.StatusAilments.Blind == 0 {
			// ミニマップを更新して敵・アイテム・階段を表示
			g.miniMapDirty = true
			action := Action{
				Duration: 0.4,
				Message:  "目が見えるようになった",
				Execute:  func(g *Game) {},
			}
			g.Enqueue(action)
		}
	}
	if g.state.Player.StatusAilments.Poison > 0 {
		g.state.Player.StatusAilments.Poison--
		if g.state.Player.StatusAilments.Poison == 0 {
			g.Enqueue(Action{
				Duration: 0.4,
				Message:  "毒が抜けた",
				Execute:  func(g *Game) {},
			})
		}
	}
	if g.state.Player.StatusAilments.Slow > 0 {
		g.state.Player.StatusAilments.Slow--
		if g.state.Player.StatusAilments.Slow == 0 {
			g.Enqueue(Action{
				Duration: 0.4,
				Message:  "体の速さが元に戻った",
				Execute:  func(g *Game) {},
			})
		}
	}
	if g.state.Player.StatusAilments.MouthSeal > 0 {
		g.state.Player.StatusAilments.MouthSeal--
		if g.state.Player.StatusAilments.MouthSeal == 0 {
			g.Enqueue(Action{
				Duration: 0.4,
				Message:  "口が動くようになった",
				Execute:  func(g *Game) {},
			})
		}
	}

	// 敵の状態異常を減らす
	for i := range g.state.Enemies {
		if g.state.Enemies[i].StatusAilments.Confusion > 0 {
			g.state.Enemies[i].StatusAilments.Confusion--
		}
		if g.state.Enemies[i].StatusAilments.Haste > 0 {
			g.state.Enemies[i].StatusAilments.Haste--
		}
		if g.state.Enemies[i].StatusAilments.Sleep > 0 {
			g.state.Enemies[i].StatusAilments.Sleep--
			// 睡眠のカードで眠らされた敵は目覚めた時に倍速化する
			if g.state.Enemies[i].StatusAilments.Sleep == 0 && wakeFromSleep(&g.state.Enemies[i].StatusAilments) {
				g.Enqueue(Action{
					Duration: 0.4,
					Message:  fmt.Sprintf("%sは目を覚まし、倍速で動き出した", g.state.Enemies[i].Name),
					Execute:  func(g *Game) {},
				})
			}
		}
	}
}

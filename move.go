//go:build !test

package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
)

func (g *Game) IncrementMoveCount() {
	g.moveCount++
	// 状態異常のターン数を減らす
	g.decrementStatusAilments()

	// 満腹度が0の場合の処理
	if g.state.Player.Satiety == 0 {
		// 満腹度0の場合、毎ターンHPが1減る
		g.damagePlayer(1, DamageSource{Kind: DamageSourceStarvation})
		// 死亡チェック
		g.state.Player.checkDeath(g)
	} else {
		// 満腹度が0でない場合のみ、レベルに応じて毎ターンHPが自然回復する
		player := &g.state.Player
		player.Health = player.recoveredHP(naturalRegenPerTurn(player.Level), false)
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

	// 装備能力の毎ターンの処理（回復・しあわせの指輪など、フック OnTurn）
	if !g.playerDead {
		g.runTurnHooks()
	}
}

// プレイヤーが混乱状態の時のランダム移動処理
func (g *Game) movePlayerConfused() bool {
	// 8方向のランダムな移動先を選択
	directions := getDirections()
	direction := directions[g.randIntn(len(directions))]
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

// checkLevelUp は累計経験値で到達できるレベルまで上げる（1回の獲得で複数レベル上がる）。
func (p *Player) checkLevelUp(g *Game) {
	newLevel := levelForExp(p.ExperiencePoints)
	if newLevel <= p.Level {
		return
	}
	p.setLevel(newLevel)
	start := len(g.ActionQueue.Queue)
	g.EnqueueMessage(fmt.Sprintf("海老さんはレベル%dに上がった", p.Level), 1.0)
	g.soundFirstAction(start, "level")
}

// setLevel はレベルを変更し、最大HPをレベルの差分だけ増減する。
// レベルが上がった分は現在HPも増え、下がったときは経験値をそのレベルの下限に揃える。
func (p *Player) setLevel(level int) {
	level = clampPlayerLevel(level)
	diff := levelHealthBonus(level) - levelHealthBonus(p.Level)
	p.MaxHealth = max(1, p.MaxHealth+diff)
	if diff > 0 {
		p.Health += diff
	}
	p.Health = min(p.Health, p.MaxHealth)
	if level < p.Level {
		p.ExperiencePoints = expForLevel(level)
	}
	p.Level = level
}

// プレイヤーの死亡チェック
func (p *Player) checkDeath(g *Game) {
	if p.Health <= 0 && !g.playerDead {
		g.playerDead = true
		g.fadeOutProgress = 0.0
		g.gameResetTimer = 6.0 // 6秒後にリセット（攻撃処理待ち+メッセージ1秒+フェードアウト1秒+待機2秒）

		// 冒険の結果を記録し、中断データを削除する（復活による再開を防ぐ）
		g.finishRun(false)

		// 死亡メッセージは後でActionQueueが空になってから追加する
	}
}

// newInitialPlayer は冒険開始時のプレイヤーを作る（座標は GenerateRandomMap で設定される）。
func newInitialPlayer() Player {
	return Player{
		Name:             "海老さん",
		Entity:           Entity{Char: '@'},
		Health:           initialPlayerHealth,
		MaxHealth:        initialPlayerHealth,
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
}

// ゲームリセット機能（新しいシードで冒険をやり直す）
func (g *Game) resetGame() {
	if !g.headless {
		g.openDifficultySelect()
		return
	}
	g.startNewRun(newRandomSeed())
}

// retrySameSeed は現在の冒険を破棄し、同じシードで1階からやり直す。
func (g *Game) retrySameSeed() {
	seed := g.seed
	g.deleteSave() // 破棄した冒険の中断データから再開しないようにする
	g.startNewRun(seed)
	g.fadeInProgress = 0.0
	g.EnqueueMessage(fmt.Sprintf("シード %d で1階から再挑戦した", seed), 1.0)
}

// startNewRun は指定シードで1階から新しい冒険を始める。
// 同じシードなら同じフロアが生成され、ターン進行の乱数列も同じになる。
func (g *Game) startNewRun(seed int64) {
	g.replay, g.replaySegment = nil, nil
	g.replaySuppressed = false
	g.showDifficultySelect, g.showClearScreen, g.showRunHistory = false, false, false
	g.seed = seed
	g.rng = newTurnRNG(seed)
	// 入力の記録は冒険ごとに取り直す（同じシード＋記録した入力でこの冒険を再生できる）
	if recorder, ok := g.input.(*InputRecorder); ok {
		recorder.Reset()
	}
	g.lastIncrementFrame, g.lastArrowPressFrame, g.lastDashStopFrame = 0, 0, 0
	g.lastDamageSource = DamageSource{}
	g.stats = RunStats{}
	g.bossDefeated = false
	g.cleared = false
	g.lastRunResult = nil
	g.runEnded = false
	player := newInitialPlayer()

	// 新しいマップを生成（プレイヤーの座標も部屋の中に設定される）
	mapGrid, enemies, items, newFloor, rooms, traps := GenerateRandomMapFor(70, 70, 0, &player, newFloorRNG(seed, 1), g.difficulty)
	g.Floor = newFloor
	g.floorTurns = 0
	g.windWarning1Shown = false
	g.windWarning2Shown = false

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
	g.wakeSleepingEnemiesOnArrival()
	g.pickupBanned = false
	g.pendingFloorDescent = false
	g.genocidedMonsterIDs = nil
	g.pendingEnemySpawns = nil
	g.resetShopState() // 店の状態（wave10-B）
	g.showBlankCardMenu = false
	g.blankCardTarget = nil
	g.blankCardIndex = 0
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

	if g.state.Player.StatusAilments.BearTrap > 0 {
		g.EnqueueMessage("トラバサミにかかっていて動けない", 0.4)
		g.isActioned = true
		return true
	}
	// プレイヤーが混乱状態の場合、入力方向に関係なくランダムな8方向に移動
	if g.state.Player.StatusAilments.Confusion > 0 {
		return g.movePlayerConfused()
	}

	newPX := g.state.Player.X + dx
	newPY := g.state.Player.Y + dy

	for i, enemy := range g.state.Enemies {
		if enemy.X == newPX && enemy.Y == newPY {
			g.state.Player.Direction = determineDirection(dx, dy)
			if isEnemyDisguised(enemy) && g.revealEnemy(i) {
				g.isActioned = true
				return true
			}
			if g.trySwapPlacesWithEnemy(i, dx, dy) {
				return true
			}
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

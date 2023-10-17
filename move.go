package main

import (
	_ "image/png" // PNG画像を読み込むために必要
	"math/rand"
)

func moveRandomly(g *Game, i int) {
	enemy := g.state.Enemies[i]
	moved := false
	attemptCount := 0
	maxAttempts := 10 // 最大試行回数

	for !moved && attemptCount < maxAttempts {
		attemptCount++ // 試行回数をインクリメント

		// If the enemy's Direction is uninitialized, select a random direction.
		if enemy.Direction == -1 {
			enemy.Direction = rand.Intn(4)
		}

		switch enemy.Direction {
		case 0: // Up
			newY, newX := enemy.Y-1, enemy.X
			if newY > 0 && !g.state.Map[newY][newX].Blocked && !isOccupied(g, newX, newY) {
				g.state.Enemies[i].Y--
				moved = true
				enemy.Direction = -1 // Reset direction after moving
			} else {
				enemy.Direction = rand.Intn(4) // Select a new random direction if movement is blocked
			}
		case 1: // Down
			newY, newX := enemy.Y+1, enemy.X
			if newY < len(g.state.Map)-1 && !g.state.Map[newY][newX].Blocked && !isOccupied(g, newX, newY) {
				g.state.Enemies[i].Y++
				moved = true
				enemy.Direction = -1 // Reset direction after moving
			} else {
				enemy.Direction = rand.Intn(4) // Select a new random direction if movement is blocked
			}
		case 2: // Left
			newY, newX := enemy.Y, enemy.X-1
			if newX > 0 && !g.state.Map[newY][newX].Blocked && !isOccupied(g, newX, newY) {
				g.state.Enemies[i].X--
				moved = true
				enemy.Direction = -1 // Reset direction after moving
			} else {
				enemy.Direction = rand.Intn(4) // Select a new random direction if movement is blocked
			}
		case 3: // Right
			newY, newX := enemy.Y, enemy.X+1
			if newX < len(g.state.Map[0])-1 && !g.state.Map[newY][newX].Blocked && !isOccupied(g, newX, newY) {
				g.state.Enemies[i].X++
				moved = true
				enemy.Direction = -1 // Reset direction after moving
			} else {
				enemy.Direction = rand.Intn(4) // Select a new random direction if movement is blocked
			}
		}
	}
}

func (g *Game) MoveTowardsPlayer(enemyIndex int) {
	enemy := g.state.Enemies[enemyIndex]
	player := g.state.Player

	// Determine the direction to move based on the player's position.
	dx := player.X - enemy.X
	dy := player.Y - enemy.Y

	// Determine the new position of the enemy.
	newX, newY := enemy.X+sign(dx), enemy.Y+sign(dy)

	// Check for blockages in diagonal movement
	blockUp := enemy.Y > 0 && g.state.Map[enemy.Y-1][enemy.X].Blocked
	blockDown := enemy.Y < len(g.state.Map)-1 && g.state.Map[enemy.Y+1][enemy.X].Blocked
	blockLeft := enemy.X > 0 && g.state.Map[enemy.Y][enemy.X-1].Blocked
	blockRight := enemy.X < len(g.state.Map[0])-1 && g.state.Map[enemy.Y][enemy.X+1].Blocked

	// Log block status
	//log.Printf("Block status - Up: %v, Down: %v, Left: %v, Right: %v\n", blockUp, blockDown, blockLeft, blockRight)

	// Adjust diagonal movement based on block status
	if dx != 0 && dy != 0 { // Diagonal movement
		// Check the block status for the intended diagonal movement
		blockDiagonal := g.state.Map[newY][newX].Blocked
		//log.Printf("Block Diagonal: %v\n", blockDiagonal) // Log the block status for the intended diagonal movement

		if blockDiagonal || ((dx > 0 && dy > 0 && (blockDown || blockRight)) || (dx > 0 && dy < 0 && (blockUp || blockRight)) || (dx < 0 && dy > 0 && (blockDown || blockLeft)) || (dx < 0 && dy < 0 && (blockUp || blockLeft))) {
			// Adjust movement to be only horizontal or vertical
			if rand.Intn(2) == 0 {
				newY = enemy.Y // Reset vertical movement
			} else {
				newX = enemy.X // Reset horizontal movement
			}
		}
	}

	// Check if the new position is blocked or occupied.
	if !g.state.Map[newY][newX].Blocked && !isOccupied(g, newX, newY) {
		g.state.Enemies[enemyIndex].X = newX
		g.state.Enemies[enemyIndex].Y = newY
		//log.Printf("Enemy moved to: (%d, %d)\n", newX, newY) // Log the new position
	} else {
		// If the direct path is blocked, try moving horizontally or vertically.
		blockUp := enemy.Y > 0 && g.state.Map[enemy.Y-1][enemy.X].Blocked
		blockDown := enemy.Y < len(g.state.Map)-1 && g.state.Map[enemy.Y+1][enemy.X].Blocked
		blockLeft := enemy.X > 0 && g.state.Map[enemy.Y][enemy.X-1].Blocked
		blockRight := enemy.X < len(g.state.Map[0])-1 && g.state.Map[enemy.Y][enemy.X+1].Blocked

		//log.Printf("Block status - Up: %v, Down: %v, Left: %v, Right: %v\n", blockUp, blockDown, blockLeft, blockRight) // Log block status

		if dx != 0 && dy != 0 { // Diagonal movement
			if dx > 0 && dy > 0 && !blockDown && !blockRight { // Moving DownRight
				g.state.Enemies[enemyIndex].X++
				g.state.Enemies[enemyIndex].Y++
			} else if dx > 0 && dy < 0 && !blockUp && !blockRight { // Moving UpRight
				g.state.Enemies[enemyIndex].X++
				g.state.Enemies[enemyIndex].Y--
			} else if dx < 0 && dy > 0 && !blockDown && !blockLeft { // Moving DownLeft
				g.state.Enemies[enemyIndex].X--
				g.state.Enemies[enemyIndex].Y++
			} else if dx < 0 && dy < 0 && !blockUp && !blockLeft { // Moving UpLeft
				g.state.Enemies[enemyIndex].X--
				g.state.Enemies[enemyIndex].Y--
			} else if !blockLeft && dx < 0 { // Move Left only
				g.state.Enemies[enemyIndex].X--
			} else if !blockRight && dx > 0 { // Move Right only
				g.state.Enemies[enemyIndex].X++
			} else if !blockUp && dy < 0 { // Move Up only
				g.state.Enemies[enemyIndex].Y--
			} else if !blockDown && dy > 0 { // Move Down only
				g.state.Enemies[enemyIndex].Y++
			}
		} else {
			newX, newY = enemy.X+sign(dx), enemy.Y
			if !g.state.Map[newY][newX].Blocked && !isOccupied(g, newX, newY) {
				g.state.Enemies[enemyIndex].X = newX
			} else {
				newX, newY = enemy.X, enemy.Y+sign(dy)
				if !g.state.Map[newY][newX].Blocked && !isOccupied(g, newX, newY) {
					g.state.Enemies[enemyIndex].Y = newY
				}
			}
		}
		// Log any movement or action taken
		//log.Printf("Final Enemy Position: (%d, %d)\n", g.state.Enemies[enemyIndex].X, g.state.Enemies[enemyIndex].Y)
	}
}

func (g *Game) MoveEnemies() {
	for i, enemy := range g.state.Enemies {
		// Variables to store the difference in position
		dx := enemy.X - g.state.Player.X
		dy := enemy.Y - g.state.Player.Y

		// Calculate Manhattan distance between enemy and player
		distance := abs(dx) + abs(dy)
		if distance >= 7 {
			g.state.Enemies[i].PlayerDiscovered = false
		}

		// Check if the enemy and player are in the same room
		if isSameRoom(enemy.X, enemy.Y, g.state.Player.X, g.state.Player.Y, g.rooms) {
			g.state.Enemies[i].PlayerDiscovered = true
		}

		// Check if the enemy is adjacent or diagonally adjacent to the player
		if abs(dx) <= 1 && abs(dy) <= 1 {
			g.state.Enemies[i].PlayerDiscovered = true
			//log.Printf("Enemy position: (%d, %d), Player position: (%d, %d)\n", enemy.X, enemy.Y, g.state.Player.X, g.state.Player.Y)
			// Determine if there are walls that should prevent attacking
			blockUp := enemy.Y > 0 && g.state.Map[enemy.Y-1][enemy.X].Blocked
			blockDown := enemy.Y < len(g.state.Map)-1 && g.state.Map[enemy.Y+1][enemy.X].Blocked
			blockLeft := enemy.X > 0 && g.state.Map[enemy.Y][enemy.X-1].Blocked
			blockRight := enemy.X < len(g.state.Map[0])-1 && g.state.Map[enemy.Y][enemy.X+1].Blocked

			// Log the values of blockUp, blockDown, blockLeft, blockRight
			//log.Printf("blockUp: %v, blockDown: %v, blockLeft: %v, blockRight: %v\n", blockUp, blockDown, blockLeft, blockRight)

			preventAttack := false

			if dx == 1 && dy == 1 { // Player is to the top-left of enemy
				//log.Printf("the top-left of enemy")
				preventAttack = blockUp || blockLeft
			} else if dx == -1 && dy == 1 { // Player is to the top-right of enemy
				//log.Printf("the top-right of enemy")
				preventAttack = blockUp || blockRight
			} else if dx == 1 && dy == -1 { // Player is to the bottom-left of enemy
				//log.Printf("the bottom-left of enemy")
				preventAttack = blockDown || blockLeft
			} else if dx == -1 && dy == -1 { // Player is to the bottom-right of enemy
				//log.Printf("the bottom-right of enemy")
				preventAttack = blockDown || blockRight
			}

			// Log the value of preventAttack
			//log.Printf("preventAttack: %v\n", preventAttack)

			if preventAttack {
				g.MoveTowardsPlayer(i) // Call function to move enemy towards player
			} else {
				g.DamagePlayer(enemy.AttackPower) // Enemy attacks player with its AttackPower
			}

		} else if g.state.Enemies[i].PlayerDiscovered {
			g.MoveTowardsPlayer(i) // Call function to move enemy towards player
		} else {
			moveRandomly(g, i) // Call function to move enemy randomly
		}
	}
}

func (p *Player) checkLevelUp() {
	if p.Level < 10 && p.Level < len(levelExpRequirements) && p.ExperiencePoints >= levelExpRequirements[p.Level] {
		p.Level++ // レベルアップ
		// 必要に応じて他のプレイヤーステータスをアップデート
		p.MaxHealth += 10
	}
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

	result := room1 == room2
	//if result {
	//log.Printf("Points are in the same room: %v\n", result) // Log result
	//}
	return result
}

func (g *Game) MovePlayer(dx, dy int) bool {
	// dx と dy が両方とも0の場合、移動は発生していない
	if dx == 0 && dy == 0 {
		return false
	}

	newPX := g.state.Player.X + dx
	newPY := g.state.Player.Y + dy

	// 敵との戦闘チェック
	if g.CheckForEnemies(newPX, newPY) {
		// 戦闘が発生した場合、プレイヤーは移動しない
		return false
	}

	// マップ範囲内およびブロックされていないタイル上にあることを確認
	if newPX >= 0 && newPX < len(g.state.Map[0]) && newPY >= 0 && newPY < len(g.state.Map) && !g.state.Map[newPY][newPX].Blocked {
		g.state.Player.X = newPX
		g.state.Player.Y = newPY
		g.IncrementMoveCount() // プレイヤーが移動するたびにカウントを増やす
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

func (g *Game) DamagePlayer(amount int) {
	// Player's DefensePower is considered while receiving damage
	netDamage := amount - g.state.Player.DefensePower
	if netDamage < 0 { // Ensure damage does not go below 0
		netDamage = 0
	}
	g.state.Player.Health -= netDamage
	if g.state.Player.Health < 0 {
		g.state.Player.Health = 0 // Ensure health does not go below 0
	}
}

func (g *Game) CheckForEnemies(x, y int) bool {
	for i, enemy := range g.state.Enemies {
		if enemy.X == x && enemy.Y == y {
			// Player's AttackPower is considered while dealing damage
			netDamage := g.state.Player.AttackPower + g.state.Player.Power + g.state.Player.Level - enemy.DefensePower
			if netDamage < 0 { // Ensure damage does not go below 0
				netDamage = 0
			}
			g.state.Enemies[i].Health -= netDamage
			if g.state.Enemies[i].Health <= 0 {
				// 敵のHealthが0以下の場合、敵を配列から削除
				g.state.Enemies = append(g.state.Enemies[:i], g.state.Enemies[i+1:]...)

				// 敵の経験値をプレイヤーの所持経験値に加える
				g.state.Player.ExperiencePoints += enemy.ExperiencePoints

				g.state.Player.checkLevelUp() // レベルアップをチェック

			} else {
				// Enemy retaliates with its AttackPower
				g.DamagePlayer(enemy.AttackPower)
			}
			g.IncrementMoveCount()
			g.MoveEnemies()
			return true
		}
	}
	return false
}

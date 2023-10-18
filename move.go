package main

import (
	_ "image/png" // PNG画像を読み込むために必要
	"log"
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

func isPositionFree(g *Game, x, y, enemyIndex int) bool {
	// Bounds check
	if x < 0 || y < 0 || x >= len(g.state.Map[0]) || y >= len(g.state.Map) {
		return false
	}

	// Check if the position is blocked on the map.
	if g.state.Map[y][x].Blocked {
		return false
	}

	// Check if the position is occupied by the player.
	if g.state.Player.X == x && g.state.Player.Y == y {
		return false
	}

	// Check if the position is occupied by another enemy.
	for i, enemy := range g.state.Enemies {
		if i != enemyIndex && enemy.X == x && enemy.Y == y {
			return false
		}
	}

	return true
}

func isDiagonallyBlocked(g *Game, x, y int) bool {
	return g.state.Map[y][x].Blocked
}

func isBlocked(g *Game, x, y int) (bool, bool, bool, bool) {
	blockUp := y > 0 && g.state.Map[y-1][x].Blocked
	blockDown := y < len(g.state.Map)-1 && g.state.Map[y+1][x].Blocked
	blockLeft := x > 0 && g.state.Map[y][x-1].Blocked
	blockRight := x < len(g.state.Map[0])-1 && g.state.Map[y][x+1].Blocked
	return blockUp, blockDown, blockLeft, blockRight
}

func (g *Game) MoveTowardsPlayer(enemyIndex int) {
	enemy := g.state.Enemies[enemyIndex]
	player := g.state.Player

	// Log initial positions of the enemy and player
	//log.Printf("Initial positions - Enemy: (%d, %d), Player: (%d, %d)\n", enemy.X, enemy.Y, player.X, player.Y)

	// Determine the direction to move based on the player's position.
	dx := player.X - enemy.X
	dy := player.Y - enemy.Y

	// Log the direction
	//log.Printf("Direction - dx: %d, dy: %d\n", dx, dy)

	// Determine the new position of the enemy.
	newX, newY := enemy.X+sign(dx), enemy.Y+sign(dy)

	// Check for blockages
	blockUp, blockDown, blockLeft, blockRight := isBlocked(g, enemy.X, enemy.Y)
	blockDiagonal := isDiagonallyBlocked(g, newX, newY)

	// Log block status
	//log.Printf("Block status - Up: %v, Down: %v, Left: %v, Right: %v, Diagonal: %v\n", blockUp, blockDown, blockLeft, blockRight, blockDiagonal)

	// Adjust diagonal movement based on block status
	if dx != 0 && dy != 0 { // Diagonal movement
		if blockDiagonal || ((dx > 0 && dy > 0 && (blockDown || blockRight)) ||
			(dx > 0 && dy < 0 && (blockUp || blockRight)) ||
			(dx < 0 && dy > 0 && (blockDown || blockLeft)) ||
			(dx < 0 && dy < 0 && (blockUp || blockLeft))) {
			// Adjust movement to be only horizontal or vertical
			if rand.Intn(2) == 0 {
				newY = enemy.Y // Reset vertical movement
			} else {
				newX = enemy.X // Reset horizontal movement
			}
			//log.Printf("Test1: (%d, %d)\n", newX, newY)
		}
	}

	// Check if the new position is blocked or occupied.
	if isPositionFree(g, newX, newY, enemyIndex) {
		g.state.Enemies[enemyIndex].X = newX
		g.state.Enemies[enemyIndex].Y = newY
		//log.Printf("Test2: (%d, %d)\n", newX, newY)
		// Log successful movement
		//log.Printf("1Enemy moved to: (%d, %d)\n", newX, newY)
	} else {
		// Log failed movement
		//log.Printf("1Failed to move to: (%d, %d)\n", newX, newY)

		// If the direct path is blocked, try moving horizontally or vertically.
		blockUp, blockDown, blockLeft, blockRight = isBlocked(g, enemy.X, enemy.Y)
		//log.Printf("Block status after failed move - Up: %v, Down: %v, Left: %v, Right: %v\n", blockUp, blockDown, blockLeft, blockRight)

		if dx != 0 && dy != 0 { // Diagonal movement
			//log.Println("Attempting diagonal movement")
			if dx > 0 && dy > 0 && !blockDown && !blockRight { // Moving DownRight
				newX, newY = enemy.X+1, enemy.Y+1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					g.state.Enemies[enemyIndex].Y = newY
					log.Printf("2Enemy moved DownRight to: (%d, %d)\n", newX, newY)
				} else {
					//log.Printf("2Failed to move DownRight to: (%d, %d)\n", newX, newY)
					newX, newY = enemy.X+1, enemy.Y
					if isPositionFree(g, newX, newY, enemyIndex) {
						g.state.Enemies[enemyIndex].X = newX
						g.state.Enemies[enemyIndex].Y = newY
						//log.Printf("extra Enemy moved Down to: (%d, %d)\n", newX, newY)
					} else {
						newX, newY = enemy.X+1, enemy.Y+1
						if isPositionFree(g, newX, newY, enemyIndex) {
							g.state.Enemies[enemyIndex].X = newX
							g.state.Enemies[enemyIndex].Y = newY
						}
						//log.Printf("extra Failed to move Down to: (%d, %d)\n", newX, newY)
					}
				}
			} else if dx < 0 && dy > 0 && !blockDown && !blockLeft { // Moving DownLeft
				newX, newY = enemy.X-1, enemy.Y+1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					g.state.Enemies[enemyIndex].Y = newY
					//log.Printf("3Enemy moved DownLeft to: (%d, %d)\n", newX, newY)
				} else {
					//log.Printf("3Failed to move DownLeft to: (%d, %d)\n", newX, newY)
					newX, newY = enemy.X-1, enemy.Y
					if isPositionFree(g, newX, newY, enemyIndex) {
						g.state.Enemies[enemyIndex].X = newX
						g.state.Enemies[enemyIndex].Y = newY
						//log.Printf("extra Enemy moved Down to: (%d, %d)\n", newX, newY)
					} else {
						newX, newY = enemy.X-1, enemy.Y+1
						if isPositionFree(g, newX, newY, enemyIndex) {
							g.state.Enemies[enemyIndex].X = newX
							g.state.Enemies[enemyIndex].Y = newY
						}
						//log.Printf("extra Failed to move Down to: (%d, %d)\n", newX, newY)
					}

				}
			} else if dx > 0 && dy < 0 && !blockUp && !blockRight { // Moving UpRight
				newX, newY = enemy.X+1, enemy.Y-1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					g.state.Enemies[enemyIndex].Y = newY
					//log.Printf("4Enemy moved UpRight to: (%d, %d)\n", newX, newY)
				} else {
					//log.Printf("4Failed to move UpRight to: (%d, %d)\n", newX, newY)
					newX, newY = enemy.X+1, enemy.Y
					if isPositionFree(g, newX, newY, enemyIndex) {
						g.state.Enemies[enemyIndex].X = newX
						g.state.Enemies[enemyIndex].Y = newY
						//log.Printf("extra Enemy moved Up to: (%d, %d)\n", newX, newY)
					} else {
						newX, newY = enemy.X-1, enemy.Y+1
						if isPositionFree(g, newX, newY, enemyIndex) {
							g.state.Enemies[enemyIndex].X = newX
							g.state.Enemies[enemyIndex].Y = newY
						}
						//log.Printf("extra Failed to move Up to: (%d, %d)\n", newX, newY)
					}
				}
			} else if dx < 0 && dy < 0 && !blockUp && !blockLeft { // Moving UpLeft
				newX, newY = enemy.X-1, enemy.Y-1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					g.state.Enemies[enemyIndex].Y = newY
					//log.Printf("5Enemy moved UpLeft to: (%d, %d)\n", newX, newY)
				} else {
					//log.Printf("5Failed to move UpLeft to: (%d, %d)\n", newX, newY)
					newX, newY = enemy.X, enemy.Y-1
					if isPositionFree(g, newX, newY, enemyIndex) {
						g.state.Enemies[enemyIndex].X = newX
						g.state.Enemies[enemyIndex].Y = newY
						//log.Printf("extra Enemy moved Up to: (%d, %d)\n", newX, newY)
					} else {
						newX, newY = enemy.X+1, enemy.Y-1
						if isPositionFree(g, newX, newY, enemyIndex) {
							g.state.Enemies[enemyIndex].X = newX
							g.state.Enemies[enemyIndex].Y = newY
						}
						//log.Printf("extraFailed to move Up to: (%d, %d)\n", newX, newY)
					}
				}
			} else if !blockLeft && dx < 0 { // Move Left only
				newX, newY = enemy.X-1, enemy.Y
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					//log.Printf("6Enemy moved Left to: (%d, %d)\n", newX, newY)
				} else {
					//log.Printf("6Failed to move Left to: (%d, %d)\n", newX, newY)
				}
			} else if !blockRight && dx > 0 { // Move Right only
				newX, newY = enemy.X+1, enemy.Y
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					//log.Printf("7Enemy moved Right to: (%d, %d)\n", newX, newY)
				} else {
					//log.Printf("7Failed to move Right to: (%d, %d)\n", newX, newY)
				}
			} else if !blockUp && dy < 0 { // Move Up only
				newX, newY = enemy.X, enemy.Y-1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].Y = newY
					//log.Printf("8Enemy moved Up to: (%d, %d)\n", newX, newY)
				} else {
					//log.Printf("8Failed to move Up to: (%d, %d)\n", newX, newY)
				}
			} else if !blockDown && dy > 0 { // Move Down only
				newX, newY = enemy.X, enemy.Y+1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].Y = newY
					//log.Printf("9Enemy moved Down to: (%d, %d)\n", newX, newY)
				} else {
					//log.Printf("9Failed to move Down to: (%d, %d)\n", newX, newY)
				}
			}
		} else {
			//log.Println("Attempting horizontal or vertical movement")
			newX, newY = enemy.X+sign(dx), enemy.Y
			if isPositionFree(g, newX, newY, enemyIndex) && (newX != enemy.X || newY != enemy.Y) {
				g.state.Enemies[enemyIndex].X = newX
				//log.Printf("10Enemy moved to: (%d, %d)\n", newX, newY)
			} else {
				// Alternative logic to try moving in a different direction if the enemy remains in the same position
				newX, newY = enemy.X, enemy.Y+sign(dy)
				if isPositionFree(g, newX, newY, enemyIndex) && (newX != enemy.X || newY != enemy.Y) {
					g.state.Enemies[enemyIndex].Y = newY
					//log.Printf("11Enemy moved to: (%d, %d)\n", newX, newY)
				} else {
					//log.Printf("10Failed to move to: (%d, %d)\n", newX, newY)
					// Alternative movement logic to get closer to the player
					if dx != 0 { // If there is horizontal distance
						altX := enemy.X + sign(dx) // Try moving horizontally closer
						if isPositionFree(g, altX, enemy.Y, enemyIndex) {
							g.state.Enemies[enemyIndex].X = altX
							//log.Printf("12Enemy moved to: (%d, %d)\n", altX, enemy.Y)
						}
					}
					if dy != 0 { // If there is vertical distance
						altY := enemy.Y + sign(dy) // Try moving vertically closer
						if isPositionFree(g, enemy.X, altY, enemyIndex) {
							g.state.Enemies[enemyIndex].Y = altY
							log.Printf("13Enemy moved to: (%d, %d)\n", enemy.X, altY)
						}
					}
					// Log if the enemy failed to move closer
					if enemy.X == g.state.Enemies[enemyIndex].X && enemy.Y == g.state.Enemies[enemyIndex].Y {
						//log.Printf("Enemy failed to move closer to player from: (%d, %d)\n", enemy.X, enemy.Y)

						// Try diagonal movement towards the player
						//log.Println("Attempting diagonal movement towards the player")

						// Determine the diagonal directions to try based on the player's position
						var diagDx, diagDy int
						if g.state.Player.X > enemy.X {
							diagDx = 1 // Player is to the right
							diagDy = 1
							newX, newY = enemy.X+diagDx, enemy.Y+diagDy
							if isPositionFree(g, newX, newY, enemyIndex) {
								g.state.Enemies[enemyIndex].X = newX
								g.state.Enemies[enemyIndex].Y = newY
								//log.Printf("Enemy moved diagonally to: (%d, %d)\n", newX, newY)
							} else {
								diagDy = -1
								newX, newY = enemy.X+diagDx, enemy.Y+diagDy
								if isPositionFree(g, newX, newY, enemyIndex) {
									g.state.Enemies[enemyIndex].X = newX
									g.state.Enemies[enemyIndex].Y = newY
									//log.Printf("Enemy moved diagonally to: (%d, %d)\n", newX, newY)
								} else {
									//log.Printf("Failed to move diagonally to: (%d, %d)\n", newX, newY)
								}
								//log.Printf("Failed to move diagonally to: (%d, %d)\n", newX, newY)
							}
						} else if g.state.Player.X < enemy.X {
							diagDx = -1 // Player is to the left
							diagDy = 1
							newX, newY = enemy.X+diagDx, enemy.Y+diagDy
							if isPositionFree(g, newX, newY, enemyIndex) {
								g.state.Enemies[enemyIndex].X = newX
								g.state.Enemies[enemyIndex].Y = newY
								//log.Printf("Enemy moved diagonally to: (%d, %d)\n", newX, newY)
							} else {
								diagDy = -1
								newX, newY = enemy.X+diagDx, enemy.Y+diagDy
								if isPositionFree(g, newX, newY, enemyIndex) {
									g.state.Enemies[enemyIndex].X = newX
									g.state.Enemies[enemyIndex].Y = newY
									//log.Printf("Enemy moved diagonally to: (%d, %d)\n", newX, newY)
								} else {
									//log.Printf("Failed to move diagonally to: (%d, %d)\n", newX, newY)
								}
								//log.Printf("Failed to move diagonally to: (%d, %d)\n", newX, newY)
							}
						} else if g.state.Player.Y > enemy.Y {
							diagDy = 1 // Player is below
							diagDx = 1
							newX, newY = enemy.X+diagDx, enemy.Y+diagDy
							if isPositionFree(g, newX, newY, enemyIndex) {
								g.state.Enemies[enemyIndex].X = newX
								g.state.Enemies[enemyIndex].Y = newY
								//log.Printf("Enemy moved diagonally to: (%d, %d)\n", newX, newY)
							} else {
								diagDx = -1
								newX, newY = enemy.X+diagDx, enemy.Y+diagDy
								if isPositionFree(g, newX, newY, enemyIndex) {
									g.state.Enemies[enemyIndex].X = newX
									g.state.Enemies[enemyIndex].Y = newY
									//log.Printf("Enemy moved diagonally to: (%d, %d)\n", newX, newY)
								} else {
									//log.Printf("Failed to move diagonally to: (%d, %d)\n", newX, newY)
								}
								//log.Printf("Failed to move diagonally to: (%d, %d)\n", newX, newY)
							}
						} else if g.state.Player.Y < enemy.Y {
							diagDy = -1 // Player is above
							diagDx = 1
							newX, newY = enemy.X+diagDx, enemy.Y+diagDy
							if isPositionFree(g, newX, newY, enemyIndex) {
								g.state.Enemies[enemyIndex].X = newX
								g.state.Enemies[enemyIndex].Y = newY
								//log.Printf("Enemy moved diagonally to: (%d, %d)\n", newX, newY)
							} else {
								diagDx = -1
								newX, newY = enemy.X+diagDx, enemy.Y+diagDy
								if isPositionFree(g, newX, newY, enemyIndex) {
									g.state.Enemies[enemyIndex].X = newX
									g.state.Enemies[enemyIndex].Y = newY
									//log.Printf("Enemy moved diagonally to: (%d, %d)\n", newX, newY)
								} else {
									//log.Printf("Failed to move diagonally to: (%d, %d)\n", newX, newY)
								}
								//log.Printf("Failed to move diagonally to: (%d, %d)\n", newX, newY)
							}
						}
					}
				}
			}
		}
	}
}

func (g *Game) MoveEnemies() {
	for i, enemy := range g.state.Enemies {
		// Variables to store the difference in position
		dx := enemy.X - g.state.Player.X
		dy := enemy.Y - g.state.Player.Y

		// Calculate Manhattan distance between enemy and player
		distance := abs(dx) + abs(dy)

		// Check if the enemy and player are in the same room
		inSameRoom := isSameRoom(enemy.X, enemy.Y, g.state.Player.X, g.state.Player.Y, g.rooms)

		if distance >= 15 && !inSameRoom {
			g.state.Enemies[i].PlayerDiscovered = false
		} else if inSameRoom {
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

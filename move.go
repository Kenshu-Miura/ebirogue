package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
	"math/rand"
)

func (g *Game) IncrementMoveCount() {
	g.moveCount++
	// Check if moveCount has increased by 5
	if g.moveCount%5 == 0 && g.moveCount != 0 {
		// Recover 1 HP for the player
		g.state.Player.Health += 1
		// Ensure player's health does not exceed MaxHealth
		if g.state.Player.Health > g.state.Player.MaxHealth {
			g.state.Player.Health = g.state.Player.MaxHealth
		}
	}
	// Existing satiety reduction logic
	if g.moveCount%10 == 0 && g.moveCount != 0 {
		g.state.Player.Satiety -= 1
		if g.state.Player.Satiety < 0 {
			g.state.Player.Satiety = 0
		}
	}
}

func moveEnemy(g *Game, i int, dx, dy int) bool {
	enemy := g.state.Enemies[i]
	newX, newY := enemy.X+dx, enemy.Y+dy

	// Check for blockages
	blockUp, blockDown, blockLeft, blockRight := isBlocked(g, enemy.X, enemy.Y)

	if newX >= 0 && newX < len(g.state.Map[0]) && newY >= 0 && newY < len(g.state.Map) &&
		!g.state.Map[newY][newX].Blocked && !isOccupied(g, newX, newY) && ((dx > 0 && dy > 0 && !(blockDown || blockRight)) ||
		(dx > 0 && dy < 0 && !(blockUp || blockRight)) ||
		(dx < 0 && dy > 0 && !(blockDown || blockLeft)) ||
		(dx < 0 && dy < 0 && !(blockUp || blockLeft)) ||
		(dx == 0 || dy == 0)) { // Allow up, down, left, right movements without additional checks
		g.state.Enemies[i].X = newX
		g.state.Enemies[i].Y = newY
		return true
	}
	return false
}

func moveRandomly(g *Game, i int) {
	enemy := &g.state.Enemies[i] // Get a pointer to the enemy to update its fields
	moved := false
	attemptCount := 0
	maxAttempts := 10 // 最大試行回数

	directions := []int{Up, Down, Left, Right, UpRight, UpLeft, DownRight, DownLeft}

	// If enemy has no direction, set a random one initially
	if enemy.Direction == -1 {
		enemy.Direction = directions[rand.Intn(len(directions))]
	}

	for !moved && attemptCount < maxAttempts {
		attemptCount++ // Increment the attempt count

		// Check if there's a passage to the right or left
		var passLeft, passRight bool
		switch enemy.Direction {
		case Up:
			passLeft = g.state.Map[enemy.Y][enemy.X-1].Type == "corridor"
			passRight = g.state.Map[enemy.Y][enemy.X+1].Type == "corridor"
		case Down:
			passLeft = g.state.Map[enemy.Y][enemy.X+1].Type == "corridor"
			passRight = g.state.Map[enemy.Y][enemy.X-1].Type == "corridor"
		case Left:
			passLeft = g.state.Map[enemy.Y+1][enemy.X].Type == "corridor"
			passRight = g.state.Map[enemy.Y-1][enemy.X].Type == "corridor"
		case Right:
			passLeft = g.state.Map[enemy.Y-1][enemy.X].Type == "corridor"
			passRight = g.state.Map[enemy.Y+1][enemy.X].Type == "corridor"
		}

		var dx, dy int
		if passRight {
			switch enemy.Direction {
			case Up:
				dx, dy = 1, 0
				enemy.Direction = Right
			case Down:
				dx, dy = -1, 0
				enemy.Direction = Left
			case Left:
				dx, dy = 0, -1
				enemy.Direction = Up
			case Right:
				dx, dy = 0, 1
				enemy.Direction = Down
			}
		} else if passLeft {
			switch enemy.Direction {
			case Up:
				dx, dy = -1, 0
				enemy.Direction = Left
			case Down:
				dx, dy = 1, 0
				enemy.Direction = Right
			case Left:
				dx, dy = 0, 1
				enemy.Direction = Down
			case Right:
				dx, dy = 0, -1
				enemy.Direction = Up
			}
		} else {
			// If no passages to the right or left, continue with original logic
			switch enemy.Direction {
			case Up:
				dx, dy = 0, -1
			case Down:
				dx, dy = 0, 1
			case Left:
				dx, dy = -1, 0
			case Right:
				dx, dy = 1, 0
			case UpRight:
				dx, dy = 1, -1
			case UpLeft:
				dx, dy = -1, -1
			case DownRight:
				dx, dy = 1, 1
			case DownLeft:
				dx, dy = -1, 1
			}
		}

		if moveEnemy(g, i, dx, dy) {
			moved = true // Set moved to true if enemy moved successfully
			// Update the enemy's dx and dy fields
			enemy.dx = dx
			enemy.dy = dy
			enemy.Animating = true
			enemy.AnimationProgressInt++
		} else {
			// Determine left and right based on enemy's current direction
			switch enemy.Direction {
			case Up:
				dx, dy = -1, 0 // left is West
			case Down:
				dx, dy = 1, 0 // left is East
			case Left:
				dx, dy = 0, 1 // left is South
			case Right:
				dx, dy = 0, -1 // left is North
			}

			if moveEnemy(g, i, dx, dy) {
				moved = true
				// Update the enemy's dx and dy fields
				enemy.dx = dx
				enemy.dy = dy
				enemy.Animating = true
				enemy.AnimationProgressInt++
				// Update the enemy's direction based on the new movement
				switch enemy.Direction {
				case Up:
					enemy.Direction = Left
				case Down:
					enemy.Direction = Right
				case Left:
					enemy.Direction = Down
				case Right:
					enemy.Direction = Up
				}
			} else {
				// Try the opposite direction if left did not work
				dx, dy = -dx, -dy // This will switch from left to right or right to left
				if moveEnemy(g, i, dx, dy) {
					moved = true
					// Update the enemy's dx and dy fields
					enemy.dx = dx
					enemy.dy = dy
					enemy.Animating = true
					enemy.AnimationProgressInt++
					// Update the enemy's direction based on the new movement
					switch enemy.Direction {
					case Up:
						enemy.Direction = Right
					case Down:
						enemy.Direction = Left
					case Left:
						enemy.Direction = Up
					case Right:
						enemy.Direction = Down
					}
				} else {
					// If neither left nor right works, choose a new random direction
					enemy.Direction = directions[rand.Intn(len(directions))]
				}
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

	// Determine the direction to move based on the player's position.
	dx := player.X - enemy.X
	dy := player.Y - enemy.Y

	// Determine the new position of the enemy.
	newX, newY := enemy.X+sign(dx), enemy.Y+sign(dy)

	// Check for blockages
	blockUp, blockDown, blockLeft, blockRight := isBlocked(g, enemy.X, enemy.Y)
	blockDiagonal := isDiagonallyBlocked(g, newX, newY)

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
		}
	}

	if isPositionFree(g, newX, newY, enemyIndex) {
		g.state.Enemies[enemyIndex].X = newX
		g.state.Enemies[enemyIndex].Y = newY
	} else {
		blockUp, blockDown, blockLeft, blockRight = isBlocked(g, enemy.X, enemy.Y)
		if dx != 0 && dy != 0 { // Diagonal movement
			if dx > 0 && dy > 0 && !blockDown && !blockRight { // Moving DownRight
				newX, newY = enemy.X+1, enemy.Y+1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					g.state.Enemies[enemyIndex].Y = newY
				} else {
					newX, newY = enemy.X+1, enemy.Y
					if isPositionFree(g, newX, newY, enemyIndex) {
						g.state.Enemies[enemyIndex].X = newX
						g.state.Enemies[enemyIndex].Y = newY
					} else {
						newX, newY = enemy.X+1, enemy.Y-1
						if isPositionFree(g, newX, newY, enemyIndex) {
							g.state.Enemies[enemyIndex].X = newX
							g.state.Enemies[enemyIndex].Y = newY
						}
					}
				}
			} else if dx < 0 && dy > 0 && !blockDown && !blockLeft { // Moving DownLeft
				newX, newY = enemy.X-1, enemy.Y+1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					g.state.Enemies[enemyIndex].Y = newY
				} else {
					newX, newY = enemy.X-1, enemy.Y
					if isPositionFree(g, newX, newY, enemyIndex) {
						g.state.Enemies[enemyIndex].X = newX
						g.state.Enemies[enemyIndex].Y = newY
					} else {
						newX, newY = enemy.X-1, enemy.Y-1
						if isPositionFree(g, newX, newY, enemyIndex) {
							g.state.Enemies[enemyIndex].X = newX
							g.state.Enemies[enemyIndex].Y = newY
						}
					}
				}
			} else if dx > 0 && dy < 0 && !blockUp && !blockRight { // Moving UpRight
				newX, newY = enemy.X+1, enemy.Y-1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					g.state.Enemies[enemyIndex].Y = newY
				} else {
					newX, newY = enemy.X+1, enemy.Y
					if isPositionFree(g, newX, newY, enemyIndex) {
						g.state.Enemies[enemyIndex].X = newX
						g.state.Enemies[enemyIndex].Y = newY
					} else {
						newX, newY = enemy.X+1, enemy.Y+1
						if isPositionFree(g, newX, newY, enemyIndex) {
							g.state.Enemies[enemyIndex].X = newX
							g.state.Enemies[enemyIndex].Y = newY
						}
					}
				}
			} else if dx < 0 && dy < 0 && !blockUp && !blockLeft { // Moving UpLeft
				newX, newY = enemy.X-1, enemy.Y-1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					g.state.Enemies[enemyIndex].Y = newY
				} else {
					newX, newY = enemy.X-1, enemy.Y
					if isPositionFree(g, newX, newY, enemyIndex) {
						g.state.Enemies[enemyIndex].X = newX
						g.state.Enemies[enemyIndex].Y = newY
					} else {
						newX, newY = enemy.X, enemy.Y-1
						if isPositionFree(g, newX, newY, enemyIndex) {
							g.state.Enemies[enemyIndex].X = newX
							g.state.Enemies[enemyIndex].Y = newY
						}
					}
				}
			} else if !blockLeft && dx < 0 { // Move Left only
				newX, newY = enemy.X-1, enemy.Y
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
				} else {
					//log.Printf("Failed to move Left to: (%d, %d)\n", newX, newY)
				}
			} else if !blockRight && dx > 0 { // Move Right only
				newX, newY = enemy.X+1, enemy.Y
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
				} else {
					//log.Printf("Failed to move Right to: (%d, %d)\n", newX, newY)
				}
			} else if !blockUp && dy < 0 { // Move Up only
				newX, newY = enemy.X, enemy.Y-1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].Y = newY
				} else {
					//log.Printf("Failed to move Up to: (%d, %d)\n", newX, newY)
				}
			} else if !blockDown && dy > 0 { // Move Down only
				newX, newY = enemy.X, enemy.Y+1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].Y = newY
				} else {
					//log.Printf("Failed to move Down to: (%d, %d)\n", newX, newY)
				}
			}
		} else {
			newX, newY = enemy.X+sign(dx), enemy.Y
			if isPositionFree(g, newX, newY, enemyIndex) && (newX != enemy.X || newY != enemy.Y) {
				g.state.Enemies[enemyIndex].X = newX
			} else {
				newX, newY = enemy.X, enemy.Y+sign(dy)
				if isPositionFree(g, newX, newY, enemyIndex) && (newX != enemy.X || newY != enemy.Y) {
					g.state.Enemies[enemyIndex].Y = newY
				} else {
					if dx != 0 { // If there is horizontal distance
						altX := enemy.X + sign(dx) // Try moving horizontally closer
						if isPositionFree(g, altX, enemy.Y, enemyIndex) {
							g.state.Enemies[enemyIndex].X = altX
						}
					}
					if dy != 0 { // If there is vertical distance
						altY := enemy.Y + sign(dy) // Try moving vertically closer
						if isPositionFree(g, enemy.X, altY, enemyIndex) {
							g.state.Enemies[enemyIndex].Y = altY
						}
					}
					// Log if the enemy failed to move closer
					if enemy.X == g.state.Enemies[enemyIndex].X && enemy.Y == g.state.Enemies[enemyIndex].Y {
						var diagDx, diagDy int
						if g.state.Player.X > enemy.X {
							diagDx = 1 // Player is to the right
							diagDy = 1
							newX, newY = enemy.X+diagDx, enemy.Y+diagDy
							if isPositionFree(g, newX, newY, enemyIndex) {
								g.state.Enemies[enemyIndex].X = newX
								g.state.Enemies[enemyIndex].Y = newY
							} else {
								diagDy = -1
								newX, newY = enemy.X+diagDx, enemy.Y+diagDy
								if isPositionFree(g, newX, newY, enemyIndex) {
									g.state.Enemies[enemyIndex].X = newX
									g.state.Enemies[enemyIndex].Y = newY
								} else {
									//log.Printf("Failed to move diagonally to: (%d, %d)\n", newX, newY)
								}
							}
						} else if g.state.Player.X < enemy.X {
							diagDx = -1 // Player is to the left
							diagDy = 1
							newX, newY = enemy.X+diagDx, enemy.Y+diagDy
							if isPositionFree(g, newX, newY, enemyIndex) {
								g.state.Enemies[enemyIndex].X = newX
								g.state.Enemies[enemyIndex].Y = newY
							} else {
								diagDy = -1
								newX, newY = enemy.X+diagDx, enemy.Y+diagDy
								if isPositionFree(g, newX, newY, enemyIndex) {
									g.state.Enemies[enemyIndex].X = newX
									g.state.Enemies[enemyIndex].Y = newY
								} else {
									//log.Printf("Failed to move diagonally to: (%d, %d)\n", newX, newY)
								}
							}
						} else if g.state.Player.Y > enemy.Y {
							diagDy = 1 // Player is below
							diagDx = 1
							newX, newY = enemy.X+diagDx, enemy.Y+diagDy
							if isPositionFree(g, newX, newY, enemyIndex) {
								g.state.Enemies[enemyIndex].X = newX
								g.state.Enemies[enemyIndex].Y = newY
							} else {
								diagDx = -1
								newX, newY = enemy.X+diagDx, enemy.Y+diagDy
								if isPositionFree(g, newX, newY, enemyIndex) {
									g.state.Enemies[enemyIndex].X = newX
									g.state.Enemies[enemyIndex].Y = newY
								} else {
									//log.Printf("Failed to move diagonally to: (%d, %d)\n", newX, newY)
								}
							}
						} else if g.state.Player.Y < enemy.Y {
							diagDy = -1 // Player is above
							diagDx = 1
							newX, newY = enemy.X+diagDx, enemy.Y+diagDy
							if isPositionFree(g, newX, newY, enemyIndex) {
								g.state.Enemies[enemyIndex].X = newX
								g.state.Enemies[enemyIndex].Y = newY
							} else {
								diagDx = -1
								newX, newY = enemy.X+diagDx, enemy.Y+diagDy
								if isPositionFree(g, newX, newY, enemyIndex) {
									g.state.Enemies[enemyIndex].X = newX
									g.state.Enemies[enemyIndex].Y = newY
								} else {
									//log.Printf("Failed to move diagonally to: (%d, %d)\n", newX, newY)
								}
							}
						}
					}
				}
			}
		}
	}
}

func (g *Game) AttackFromEnemy(enemyIndex int) {
	enemy := g.state.Enemies[enemyIndex]

	netDamage := enemy.AttackPower - g.state.Player.DefensePower
	if netDamage < 0 { // Ensure damage does not go below 0
		netDamage = 0
	}

	g.state.Player.Health -= netDamage
	if g.state.Player.Health < 0 {
		g.state.Player.Health = 0 // Ensure health does not go below 0
	}

	g.descriptionQueue = append(g.descriptionQueue, fmt.Sprintf("%sから%dダメージを受けた", enemy.Name, netDamage))

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
				g.AttackFromEnemy(i) // Call function to attack player
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

	result := room1.ID == room2.ID
	//if result {
	//log.Printf("Points are in the same room: %v\n", result) // Log result
	//}
	return result
}

func (g *Game) CheetMovePlayer(dx, dy int) bool {
	// dx と dy が両方とも0の場合、移動は発生していない
	if dx == 0 && dy == 0 {
		return false
	}

	newPX := g.state.Player.X + dx
	newPY := g.state.Player.Y + dy

	// Determine the direction based on the change in position
	deltaX := newPX - g.state.Player.X
	deltaY := newPY - g.state.Player.Y
	switch {
	case deltaX == 1 && deltaY == 0:
		g.state.Player.Direction = Right
	case deltaX == -1 && deltaY == 0:
		g.state.Player.Direction = Left
	case deltaX == 0 && deltaY == 1:
		g.state.Player.Direction = Down
	case deltaX == 0 && deltaY == -1:
		g.state.Player.Direction = Up
	case deltaX == 1 && deltaY == 1:
		g.state.Player.Direction = DownRight
	case deltaX == -1 && deltaY == 1:
		g.state.Player.Direction = DownLeft
	case deltaX == 1 && deltaY == -1:
		g.state.Player.Direction = UpRight
	case deltaX == -1 && deltaY == -1:
		g.state.Player.Direction = UpLeft
	}

	// 敵との戦闘チェック
	if g.CheckForEnemies(newPX, newPY) {
		// 戦闘が発生した場合、プレイヤーは移動しない
		return false
	}

	g.state.Player.X = newPX
	g.state.Player.Y = newPY
	g.IncrementMoveCount() // プレイヤーが移動するたびにカウントを増やす
	g.PickupItem()
	return true

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
		// Determine the direction based on the change in position
		deltaX := newPX - g.state.Player.X
		deltaY := newPY - g.state.Player.Y
		switch {
		case deltaX == 1 && deltaY == 0:
			g.state.Player.Direction = Right
		case deltaX == -1 && deltaY == 0:
			g.state.Player.Direction = Left
		case deltaX == 0 && deltaY == 1:
			g.state.Player.Direction = Down
		case deltaX == 0 && deltaY == -1:
			g.state.Player.Direction = Up
		case deltaX == 1 && deltaY == 1:
			g.state.Player.Direction = DownRight
		case deltaX == -1 && deltaY == 1:
			g.state.Player.Direction = DownLeft
		case deltaX == 1 && deltaY == -1:
			g.state.Player.Direction = UpRight
		case deltaX == -1 && deltaY == -1:
			g.state.Player.Direction = UpLeft
		}

		g.state.Player.X = newPX
		g.state.Player.Y = newPY
		g.IncrementMoveCount() // プレイヤーが移動するたびにカウントを増やす
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

func (g *Game) CheckForEnemies(x, y int) bool {
	for i, enemy := range g.state.Enemies {
		if enemy.X == x && enemy.Y == y {
			// Player's AttackPower is considered while dealing damage
			netDamage := g.state.Player.AttackPower + g.state.Player.Power + g.state.Player.Level - enemy.DefensePower
			if netDamage < 0 { // Ensure damage does not go below 0
				netDamage = 0
			}
			g.descriptionQueue = append(g.descriptionQueue, fmt.Sprintf("%sに%dダメージを与えた", g.state.Enemies[i].Name, netDamage))

			g.state.Enemies[i].Health -= netDamage
			if g.state.Enemies[i].Health <= 0 {
				// 敵のHealthが0以下の場合、敵を配列から削除
				g.state.Enemies = append(g.state.Enemies[:i], g.state.Enemies[i+1:]...)

				// 敵の経験値をプレイヤーの所持経験値に加える
				g.state.Player.ExperiencePoints += enemy.ExperiencePoints

				g.state.Player.checkLevelUp() // レベルアップをチェック

			} else {
				// Enemy retaliates with its AttackPower
				g.AttackFromEnemy(i) // Call function to attack player
			}
			g.IncrementMoveCount()
			g.MoveEnemies()
			return true
		}
	}
	return false
}

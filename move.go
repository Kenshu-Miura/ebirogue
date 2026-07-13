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
	enemy := &g.state.Enemies[i]

	// 敵が部屋内にいるか通路にいるかを判定
	if isEnemyInRoom(g, enemy) {
		moveEnemyInRoom(g, i)
	} else {
		moveEnemyInCorridor(g, i)
	}
}

// 敵が部屋内にいるかを判定
func isEnemyInRoom(g *Game, enemy *Enemy) bool {
	for _, room := range g.rooms {
		// 部屋の内側の境界をチェック（壁を除く）
		if enemy.X > room.X && enemy.X < room.X+room.Width-1 &&
			enemy.Y > room.Y && enemy.Y < room.Y+room.Height-1 {
			return true
		}
	}
	return false
}

// 敵の周囲の方向を取得（8方向）
func getDirections() []struct{ dx, dy int } {
	return []struct{ dx, dy int }{
		{0, -1},  // Up
		{0, 1},   // Down
		{-1, 0},  // Left
		{1, 0},   // Right
		{-1, -1}, // UpLeft
		{1, -1},  // UpRight
		{-1, 1},  // DownLeft
		{1, 1},   // DownRight
	}
}

// 4方向のみ取得
func getMainDirections() []struct{ dx, dy int } {
	return []struct{ dx, dy int }{
		{0, -1}, // Up
		{0, 1},  // Down
		{-1, 0}, // Left
		{1, 0},  // Right
	}
}

// 部屋内での敵の移動
func moveEnemyInRoom(g *Game, i int) {
	// まず通路への移動を試行
	if moveTowardsCorridor(g, i) {
		return
	}

	// 通路への移動ができない場合、部屋の外周を歩き回る
	moveAroundRoomPerimeter(g, i)
}

// 通路に向かって移動
func moveTowardsCorridor(g *Game, i int) bool {
	enemy := &g.state.Enemies[i]
	directions := getMainDirections()

	// 各方向をチェックして通路を探す
	var corridorDirections []struct{ dx, dy int }

	for _, dir := range directions {
		newX, newY := enemy.X+dir.dx, enemy.Y+dir.dy

		// 境界チェック
		if newX < 0 || newY < 0 || newX >= len(g.state.Map[0]) || newY >= len(g.state.Map) {
			continue
		}

		// 通路または床タイルへの移動をチェック
		tile := g.state.Map[newY][newX]
		if (tile.Type == "corridor" || tile.Type == "floor") && !tile.Blocked {
			// その位置が移動可能かチェック
			if isPositionFree(g, newX, newY, i) {
				corridorDirections = append(corridorDirections, dir)
			}
		}
	}

	// 通路方向がある場合、ランダムに一つ選んで移動
	if len(corridorDirections) > 0 {
		chosenDir := corridorDirections[rand.Intn(len(corridorDirections))]
		return moveEnemy(g, i, chosenDir.dx, chosenDir.dy)
	}

	return false
}

// 部屋の外周を歩き回る
func moveAroundRoomPerimeter(g *Game, i int) {
	enemy := &g.state.Enemies[i]

	// 方向が初期化されていない場合、ランダムに設定
	if enemy.Direction == Uninitialized {
		directions := []Direction{Up, Down, Left, Right}
		enemy.Direction = directions[rand.Intn(len(directions))]
	}

	// 現在の方向で移動を試行
	var dx, dy int
	switch enemy.Direction {
	case Up:
		dx, dy = 0, -1
	case Down:
		dx, dy = 0, 1
	case Left:
		dx, dy = -1, 0
	case Right:
		dx, dy = 1, 0
	default:
		// 斜め方向の場合は主方向に変換
		directions := []Direction{Up, Down, Left, Right}
		enemy.Direction = directions[rand.Intn(len(directions))]
		dx, dy = 0, -1 // とりあえず上方向
	}

	// 移動可能かチェック
	if moveEnemy(g, i, dx, dy) {
		enemy.dx = dx
		enemy.dy = dy
		enemy.Animating = true
		return
	}

	// 移動できない場合、右回りで次の方向を試す
	nextDirections := map[Direction]Direction{
		Up:    Right,
		Right: Down,
		Down:  Left,
		Left:  Up,
	}

	for attempts := 0; attempts < 4; attempts++ {
		enemy.Direction = nextDirections[enemy.Direction]

		switch enemy.Direction {
		case Up:
			dx, dy = 0, -1
		case Down:
			dx, dy = 0, 1
		case Left:
			dx, dy = -1, 0
		case Right:
			dx, dy = 1, 0
		}

		if moveEnemy(g, i, dx, dy) {
			enemy.dx = dx
			enemy.dy = dy
			enemy.Animating = true
			return
		}
	}
}

// 通路での敵の移動
func moveEnemyInCorridor(g *Game, i int) {
	enemy := &g.state.Enemies[i]

	// 方向が初期化されていない場合、ランダムに設定
	if enemy.Direction == Uninitialized {
		directions := []Direction{Up, Down, Left, Right}
		enemy.Direction = directions[rand.Intn(len(directions))]
	}

	// まず直進を試行
	if moveStraightInCorridor(g, i) {
		return
	}

	// 直進できない場合、行き止まり処理
	handleDeadEnd(g, i)
}

// 通路で直進移動
func moveStraightInCorridor(g *Game, i int) bool {
	enemy := &g.state.Enemies[i]

	var dx, dy int
	switch enemy.Direction {
	case Up:
		dx, dy = 0, -1
	case Down:
		dx, dy = 0, 1
	case Left:
		dx, dy = -1, 0
	case Right:
		dx, dy = 1, 0
	default:
		// 斜め方向の場合は主方向に変換
		directions := []Direction{Up, Down, Left, Right}
		enemy.Direction = directions[rand.Intn(len(directions))]
		return moveStraightInCorridor(g, i) // 再帰呼び出し
	}

	// 直進方向に移動可能かチェック
	if moveEnemy(g, i, dx, dy) {
		enemy.dx = dx
		enemy.dy = dy
		enemy.Animating = true
		return true
	}

	return false
}

// 行き止まり処理
func handleDeadEnd(g *Game, i int) {
	enemy := &g.state.Enemies[i]

	// 左右の方向を取得
	leftDx, leftDy, rightDx, rightDy := getLeftRightDirections(enemy.Direction)

	// まず左方向を試行
	if moveEnemy(g, i, leftDx, leftDy) {
		enemy.dx = leftDx
		enemy.dy = leftDy
		enemy.Animating = true
		enemy.Direction = getDirectionFromMovement(leftDx, leftDy)
		return
	}

	// 左がダメなら右方向を試行
	if moveEnemy(g, i, rightDx, rightDy) {
		enemy.dx = rightDx
		enemy.dy = rightDy
		enemy.Animating = true
		enemy.Direction = getDirectionFromMovement(rightDx, rightDy)
		return
	}

	// 左右もダメなら背後に戻る
	backDx, backDy := getOppositeDirection(enemy.Direction)
	if moveEnemy(g, i, backDx, backDy) {
		enemy.dx = backDx
		enemy.dy = backDy
		enemy.Animating = true
		enemy.Direction = getDirectionFromMovement(backDx, backDy)
	}
}

// 現在の方向に対する左右の方向を取得
func getLeftRightDirections(dir Direction) (leftDx, leftDy, rightDx, rightDy int) {
	switch dir {
	case Up:
		return -1, 0, 1, 0 // Left: 左, Right: 右
	case Down:
		return 1, 0, -1, 0 // Left: 右, Right: 左
	case Left:
		return 0, 1, 0, -1 // Left: 下, Right: 上
	case Right:
		return 0, -1, 0, 1 // Left: 上, Right: 下
	default:
		return -1, 0, 1, 0 // デフォルトは上向きの場合の左右
	}
}

// 逆方向を取得
func getOppositeDirection(dir Direction) (dx, dy int) {
	switch dir {
	case Up:
		return 0, 1 // Down
	case Down:
		return 0, -1 // Up
	case Left:
		return 1, 0 // Right
	case Right:
		return -1, 0 // Left
	default:
		return 0, 1 // デフォルトは下
	}
}

// 移動量から方向を取得
func getDirectionFromMovement(dx, dy int) Direction {
	switch {
	case dx == 0 && dy == -1:
		return Up
	case dx == 0 && dy == 1:
		return Down
	case dx == -1 && dy == 0:
		return Left
	case dx == 1 && dy == 0:
		return Right
	default:
		return Up // デフォルト
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
	enemy := &g.state.Enemies[enemyIndex]
	player := g.state.Player

	// Determine the direction to move based on the player's position.
	orgX, orgY := enemy.X, enemy.Y
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

	//log.Printf("Enemy %d: (%d, %d) -> (%d, %d)\n", enemyIndex, enemy.X, enemy.Y, newX, newY)

	if isPositionFree(g, newX, newY, enemyIndex) {
		g.state.Enemies[enemyIndex].X = newX
		g.state.Enemies[enemyIndex].Y = newY
		enemy.dx = enemy.X - orgX
		enemy.dy = enemy.Y - orgY
		enemy.Animating = true
	} else {
		blockUp, blockDown, blockLeft, blockRight = isBlocked(g, enemy.X, enemy.Y)
		if dx != 0 && dy != 0 { // Diagonal movement
			if dx > 0 && dy > 0 && !blockDown && !blockRight { // Moving DownRight
				newX, newY = enemy.X+1, enemy.Y+1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					g.state.Enemies[enemyIndex].Y = newY
					enemy.dx = 1
					enemy.dy = 1
					enemy.Animating = true
				} else {
					newX, newY = enemy.X+1, enemy.Y
					if isPositionFree(g, newX, newY, enemyIndex) {
						g.state.Enemies[enemyIndex].X = newX
						g.state.Enemies[enemyIndex].Y = newY
						enemy.dx = 1
						enemy.dy = 0
						enemy.Animating = true
					} else {
						newX, newY = enemy.X+1, enemy.Y-1
						if isPositionFree(g, newX, newY, enemyIndex) {
							g.state.Enemies[enemyIndex].X = newX
							g.state.Enemies[enemyIndex].Y = newY
							enemy.dx = 1
							enemy.dy = -1
							enemy.Animating = true
						}
					}
				}
			} else if dx < 0 && dy > 0 && !blockDown && !blockLeft { // Moving DownLeft
				newX, newY = enemy.X-1, enemy.Y+1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					g.state.Enemies[enemyIndex].Y = newY
					enemy.dx = -1
					enemy.dy = 1
					enemy.Animating = true
				} else {
					newX, newY = enemy.X-1, enemy.Y
					if isPositionFree(g, newX, newY, enemyIndex) {
						g.state.Enemies[enemyIndex].X = newX
						g.state.Enemies[enemyIndex].Y = newY
						enemy.dx = -1
						enemy.dy = 0
						enemy.Animating = true
					} else {
						newX, newY = enemy.X-1, enemy.Y-1
						if isPositionFree(g, newX, newY, enemyIndex) {
							g.state.Enemies[enemyIndex].X = newX
							g.state.Enemies[enemyIndex].Y = newY
							enemy.dx = -1
							enemy.dy = -1
							enemy.Animating = true
						}
					}
				}
			} else if dx > 0 && dy < 0 && !blockUp && !blockRight { // Moving UpRight
				newX, newY = enemy.X+1, enemy.Y-1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					g.state.Enemies[enemyIndex].Y = newY
					enemy.dx = 1
					enemy.dy = -1
					enemy.Animating = true
				} else {
					newX, newY = enemy.X+1, enemy.Y
					if isPositionFree(g, newX, newY, enemyIndex) {
						g.state.Enemies[enemyIndex].X = newX
						g.state.Enemies[enemyIndex].Y = newY
						enemy.dx = 1
						enemy.dy = 0
						enemy.Animating = true
					} else {
						newX, newY = enemy.X, enemy.Y-1
						if isPositionFree(g, newX, newY, enemyIndex) {
							g.state.Enemies[enemyIndex].X = newX
							g.state.Enemies[enemyIndex].Y = newY
							enemy.dx = 0
							enemy.dy = -1
							enemy.Animating = true
						}
					}
				}
			} else if dx < 0 && dy < 0 && !blockUp && !blockLeft { // Moving UpLeft
				newX, newY = enemy.X-1, enemy.Y-1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					g.state.Enemies[enemyIndex].Y = newY
					enemy.dx = -1
					enemy.dy = -1
					enemy.Animating = true
				} else {
					newX, newY = enemy.X-1, enemy.Y
					if isPositionFree(g, newX, newY, enemyIndex) {
						g.state.Enemies[enemyIndex].X = newX
						g.state.Enemies[enemyIndex].Y = newY
						enemy.dx = -1
						enemy.dy = 0
						enemy.Animating = true
					} else {
						newX, newY = enemy.X, enemy.Y-1
						if isPositionFree(g, newX, newY, enemyIndex) {
							g.state.Enemies[enemyIndex].X = newX
							g.state.Enemies[enemyIndex].Y = newY
							enemy.dx = 0
							enemy.dy = -1
							enemy.Animating = true
						}
					}
				}
			} else if !blockLeft && dx < 0 { // Move Left only
				newX, newY = enemy.X-1, enemy.Y
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					enemy.dx = -1
					enemy.dy = 0
					enemy.Animating = true
				} else {
					//log.Printf("Failed to move Left to: (%d, %d)\n", newX, newY)
				}
			} else if !blockRight && dx > 0 { // Move Right only
				newX, newY = enemy.X+1, enemy.Y
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].X = newX
					enemy.dx = 1
					enemy.dy = 0
					enemy.Animating = true
				} else {
					//log.Printf("Failed to move Right to: (%d, %d)\n", newX, newY)
				}
			} else if !blockUp && dy < 0 { // Move Up only
				newX, newY = enemy.X, enemy.Y-1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].Y = newY
					enemy.dx = 0
					enemy.dy = -1
					enemy.Animating = true
				} else {
					//log.Printf("Failed to move Up to: (%d, %d)\n", newX, newY)
				}
			} else if !blockDown && dy > 0 { // Move Down only
				newX, newY = enemy.X, enemy.Y+1
				if isPositionFree(g, newX, newY, enemyIndex) {
					g.state.Enemies[enemyIndex].Y = newY
					enemy.dx = 0
					enemy.dy = 1
					enemy.Animating = true
				} else {
					//log.Printf("Failed to move Down to: (%d, %d)\n", newX, newY)
				}
			}
		} else {
			newX, newY = enemy.X+sign(dx), enemy.Y
			if isPositionFree(g, newX, newY, enemyIndex) && (newX != enemy.X || newY != enemy.Y) {
				g.state.Enemies[enemyIndex].X = newX
				enemy.dx = sign(dx)
				enemy.dy = 0
				enemy.Animating = true
			} else {
				newX, newY = enemy.X, enemy.Y+sign(dy)
				if isPositionFree(g, newX, newY, enemyIndex) && (newX != enemy.X || newY != enemy.Y) {
					g.state.Enemies[enemyIndex].Y = newY
					enemy.dx = 0
					enemy.dy = sign(dy)
					enemy.Animating = true
				} else {
					if dx != 0 { // If there is horizontal distance
						altX := enemy.X + sign(dx) // Try moving horizontally closer
						if isPositionFree(g, altX, enemy.Y, enemyIndex) {
							g.state.Enemies[enemyIndex].X = altX
							enemy.dx = sign(dx)
							enemy.dy = 0
							enemy.Animating = true
						}
					}
					if dy != 0 { // If there is vertical distance
						altY := enemy.Y + sign(dy) // Try moving vertically closer
						if isPositionFree(g, enemy.X, altY, enemyIndex) {
							g.state.Enemies[enemyIndex].Y = altY
							enemy.dx = 0
							enemy.dy = sign(dy)
							enemy.Animating = true
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
								enemy.dx = diagDx
								enemy.dy = diagDy
								enemy.Animating = true
							} else {
								diagDy = -1
								newX, newY = enemy.X+diagDx, enemy.Y+diagDy
								if isPositionFree(g, newX, newY, enemyIndex) {
									enemy.dx = diagDx
									enemy.dy = diagDy
									enemy.Animating = true
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
								enemy.dx = diagDx
								enemy.dy = diagDy
								enemy.Animating = true
							} else {
								diagDy = -1
								newX, newY = enemy.X+diagDx, enemy.Y+diagDy
								if isPositionFree(g, newX, newY, enemyIndex) {
									g.state.Enemies[enemyIndex].X = newX
									g.state.Enemies[enemyIndex].Y = newY
									enemy.dx = diagDx
									enemy.dy = diagDy
									enemy.Animating = true
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
								enemy.dx = diagDx
								enemy.dy = diagDy
								enemy.Animating = true
							} else {
								diagDx = -1
								newX, newY = enemy.X+diagDx, enemy.Y+diagDy
								if isPositionFree(g, newX, newY, enemyIndex) {
									g.state.Enemies[enemyIndex].X = newX
									g.state.Enemies[enemyIndex].Y = newY
									enemy.dx = diagDx
									enemy.dy = diagDy
									enemy.Animating = true
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
								enemy.dx = diagDx
								enemy.dy = diagDy
								enemy.Animating = true

							} else {
								diagDx = -1
								newX, newY = enemy.X+diagDx, enemy.Y+diagDy
								if isPositionFree(g, newX, newY, enemyIndex) {
									g.state.Enemies[enemyIndex].X = newX
									g.state.Enemies[enemyIndex].Y = newY
									enemy.dx = diagDx
									enemy.dy = diagDy
									enemy.Animating = true
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

func determineDirection(dx, dy int) Direction {
	switch {
	case dx == 1 && dy == 0:
		return Right
	case dx == -1 && dy == 0:
		return Left
	case dx == 0 && dy == 1:
		return Down
	case dx == 0 && dy == -1:
		return Up
	case dx == 1 && dy == 1:
		return DownRight
	case dx == -1 && dy == 1:
		return DownLeft
	case dx == 1 && dy == -1:
		return UpRight
	case dx == -1 && dy == -1:
		return UpLeft
	default:
		return Up // or any default direction you'd prefer
	}
}

func (g *Game) moveEnemyConfused(i int) {
	enemy := &g.state.Enemies[i]

	// 8方向のランダムな移動先を選択
	directions := []struct{ dx, dy int }{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	direction := directions[rand.Intn(len(directions))]
	newX := enemy.X + direction.dx
	newY := enemy.Y + direction.dy

	// 範囲チェック
	if newX < 0 || newX >= len(g.state.Map[0]) || newY < 0 || newY >= len(g.state.Map) {
		return
	}

	// 移動先がプレイヤーの場合、攻撃を行う
	if newX == g.state.Player.X && newY == g.state.Player.Y {
		g.AttackFromEnemy(i)
		return
	}

	// 移動先が通行可能で他の敵がいない場合、移動
	if !g.state.Map[newY][newX].Blocked && !isOccupied(g, newX, newY) {
		enemy.X = newX
		enemy.Y = newY
		enemy.dx = direction.dx
		enemy.dy = direction.dy
		enemy.Animating = true
	}
}

// プレイヤーが混乱状態の時のランダム移動処理
func (g *Game) movePlayerConfused() bool {
	// 8方向のランダムな移動先を選択
	directions := []struct{ dx, dy int }{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	direction := directions[rand.Intn(len(directions))]
	newX := g.state.Player.X + direction.dx
	newY := g.state.Player.Y + direction.dy

	// 範囲チェック
	if newX < 0 || newX >= len(g.state.Map[0]) || newY < 0 || newY >= len(g.state.Map) {
		// 混乱メッセージを表示（移動失敗）
		action := Action{
			Duration: 0.4,
			Message:  "混乱している...",
			Execute:  func(g *Game) {},
		}
		g.Enqueue(action)
		return false // 移動失敗時はターンを消費しない
	}

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

	// 移動先に敵がいる場合、移動失敗として扱う
	for _, enemy := range g.state.Enemies {
		if enemy.X == newX && enemy.Y == newY {
			action := Action{
				Duration: 0.4,
				Message:  "混乱している...",
				Execute:  func(g *Game) {},
			}
			g.Enqueue(action)
			return false // 移動失敗時はターンを消費しない
		}
	}

	// 移動先が通行可能な場合、移動
	if !g.state.Map[newY][newX].Blocked {
		action := Action{
			Duration: 0.4,
			Message:  "混乱して移動した",
			Execute:  func(g *Game) {},
		}
		g.Enqueue(action)

		g.state.Player.X = newX
		g.state.Player.Y = newY
		// アニメーション用に実際の移動方向を保存
		g.dx, g.dy = direction.dx, direction.dy
		return true // 移動成功時はターンを消費
	}

	// 移動失敗時
	action := Action{
		Duration: 0.4,
		Message:  "混乱している...",
		Execute:  func(g *Game) {},
	}
	g.Enqueue(action)
	return false // 移動失敗時はターンを消費しない
}

func (g *Game) moveEnemyBlind(i int) {
	enemy := &g.state.Enemies[i]

	// 方向が初期化されていない場合、ランダムに設定
	if enemy.Direction == Uninitialized {
		directions := []Direction{Up, Down, Left, Right, UpRight, UpLeft, DownRight, DownLeft}
		enemy.Direction = directions[rand.Intn(len(directions))]
	}

	// 現在の方向に基づいて移動先を計算
	var dx, dy int
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

	newX := enemy.X + dx
	newY := enemy.Y + dy

	// 範囲チェック
	if newX < 0 || newX >= len(g.state.Map[0]) || newY < 0 || newY >= len(g.state.Map) {
		g.changeBlindEnemyDirection(i)
		return
	}

	// 壁にぶつかった場合、方向を変更
	if g.state.Map[newY][newX].Blocked {
		g.changeBlindEnemyDirection(i)
		return
	}

	// 移動先にプレイヤーがいる場合、攻撃（特技は使用しない）
	if newX == g.state.Player.X && newY == g.state.Player.Y {
		g.AttackFromEnemyBlind(i)
		return
	}

	// 移動先に他の敵がいる場合、攻撃
	for j, otherEnemy := range g.state.Enemies {
		if j != i && otherEnemy.X == newX && otherEnemy.Y == newY {
			g.AttackEnemyFromBlindEnemy(i, j)
			return
		}
	}

	// 移動可能な場合、移動
	enemy.X = newX
	enemy.Y = newY
	enemy.dx = dx
	enemy.dy = dy
	enemy.Animating = true
}

func (g *Game) changeBlindEnemyDirection(i int) {
	enemy := &g.state.Enemies[i]

	// 現在の方向以外の7方向から選択
	allDirections := []Direction{Up, Down, Left, Right, UpRight, UpLeft, DownRight, DownLeft}
	availableDirections := make([]Direction, 0)

	for _, dir := range allDirections {
		if dir != enemy.Direction {
			availableDirections = append(availableDirections, dir)
		}
	}

	if len(availableDirections) > 0 {
		enemy.Direction = availableDirections[rand.Intn(len(availableDirections))]
	}
}

func (g *Game) MoveEnemies() {
	for i, enemy := range g.state.Enemies {
		// 仮眠状態の敵の起床チェック（隣接時）
		if enemy.StatusAilments.Sleep == -1 {
			g.WakeUpSleepingEnemyByProximity(i)
		}

		// 睡眠状態の敵は移動できない
		if enemy.StatusAilments.Sleep > 0 || enemy.StatusAilments.Sleep == -1 {
			continue
		}

		// 金縛り状態の敵は移動も攻撃もできない
		if enemy.StatusAilments.Paralysis {
			continue
		}

		// 混乱状態の敵は周囲8マスからランダムに移動
		if enemy.StatusAilments.Confusion > 0 {
			g.moveEnemyConfused(i)
			continue
		}

		// 目潰し状態の敵は直進移動
		if enemy.StatusAilments.Blind > 0 {
			g.moveEnemyBlind(i)
			continue
		}

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
			return false
		}
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

	// 敵の状態異常を減らす
	for i := range g.state.Enemies {
		if g.state.Enemies[i].StatusAilments.Confusion > 0 {
			g.state.Enemies[i].StatusAilments.Confusion--
		}
		if g.state.Enemies[i].StatusAilments.Sleep > 0 {
			g.state.Enemies[i].StatusAilments.Sleep--
		}
	}
}

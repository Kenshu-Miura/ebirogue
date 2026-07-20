//go:build !test

package main

import (
	"math/rand"
)

// 敵の行動AI（追跡・巡回・状態異常時の行動）をまとめたファイル。

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
		enemy.Direction = determineDirection(leftDx, leftDy)
		return
	}

	// 左がダメなら右方向を試行
	if moveEnemy(g, i, rightDx, rightDy) {
		enemy.dx = rightDx
		enemy.dy = rightDy
		enemy.Animating = true
		enemy.Direction = determineDirection(rightDx, rightDy)
		return
	}

	// 左右もダメなら背後に戻る
	backDx, backDy := getOppositeDirection(enemy.Direction)
	if moveEnemy(g, i, backDx, backDy) {
		enemy.dx = backDx
		enemy.dy = backDy
		enemy.Animating = true
		enemy.Direction = determineDirection(backDx, backDy)
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

// tryMoveEnemy は敵を (dx, dy) 方向へ1マス動かせるなら動かして true を返す。
func (g *Game) tryMoveEnemy(enemyIndex, dx, dy int) bool {
	if dx == 0 && dy == 0 {
		return false
	}
	enemy := &g.state.Enemies[enemyIndex]
	newX, newY := enemy.X+dx, enemy.Y+dy
	if !isPositionFree(g, newX, newY, enemyIndex) {
		return false
	}
	enemy.X = newX
	enemy.Y = newY
	enemy.dx = dx
	enemy.dy = dy
	enemy.Animating = true
	return true
}

// MoveTowardsPlayer は敵をプレイヤーへ1マス近づける。
// まっすぐ近づけない場合は enemyMoveFallbacks の代替候補を優先順に試す。
func (g *Game) MoveTowardsPlayer(enemyIndex int) {
	enemy := &g.state.Enemies[enemyIndex]
	player := g.state.Player

	dx := player.X - enemy.X
	dy := player.Y - enemy.Y
	stepX, stepY := sign(dx), sign(dy)

	// 斜め移動が壁に阻まれる場合は縦か横だけの移動へ切り替える
	if dx != 0 && dy != 0 {
		blockUp, blockDown, blockLeft, blockRight := isBlocked(g, enemy.X, enemy.Y)
		blockDiagonal := isDiagonallyBlocked(g, enemy.X+stepX, enemy.Y+stepY)
		if blockDiagonal || ((dx > 0 && dy > 0 && (blockDown || blockRight)) ||
			(dx > 0 && dy < 0 && (blockUp || blockRight)) ||
			(dx < 0 && dy > 0 && (blockDown || blockLeft)) ||
			(dx < 0 && dy < 0 && (blockUp || blockLeft))) {
			if rand.Intn(2) == 0 {
				stepY = 0 // 縦移動をやめる
			} else {
				stepX = 0 // 横移動をやめる
			}
		}
	}

	// まずプレイヤーへ直進する
	if g.tryMoveEnemy(enemyIndex, stepX, stepY) {
		return
	}

	// 直進できない場合は方向別の代替候補を順に試す
	for _, step := range enemyMoveFallbacks(g, enemyIndex, dx, dy) {
		if g.tryMoveEnemy(enemyIndex, step[0], step[1]) {
			return
		}
	}
}

// enemyMoveFallbacks は直進できない敵の代替移動候補を優先順で返す。
// dx, dy はプレイヤーまでの距離（enemyMoveFallbacks 内では符号のみ意味を持つ）。
func enemyMoveFallbacks(g *Game, enemyIndex, dx, dy int) [][2]int {
	enemy := &g.state.Enemies[enemyIndex]

	if dx != 0 && dy != 0 { // 斜め方向へ向かっている場合
		blockUp, blockDown, blockLeft, blockRight := isBlocked(g, enemy.X, enemy.Y)
		switch {
		case dx > 0 && dy > 0 && !blockDown && !blockRight: // 右下へ
			return [][2]int{{1, 1}, {1, 0}, {1, -1}}
		case dx < 0 && dy > 0 && !blockDown && !blockLeft: // 左下へ
			return [][2]int{{-1, 1}, {-1, 0}, {-1, -1}}
		case dx > 0 && dy < 0 && !blockUp && !blockRight: // 右上へ
			return [][2]int{{1, -1}, {1, 0}, {0, -1}}
		case dx < 0 && dy < 0 && !blockUp && !blockLeft: // 左上へ
			return [][2]int{{-1, -1}, {-1, 0}, {0, -1}}
		case !blockLeft && dx < 0: // 壁沿いに左へ
			return [][2]int{{-1, 0}}
		case !blockRight && dx > 0: // 壁沿いに右へ
			return [][2]int{{1, 0}}
		case !blockUp && dy < 0: // 壁沿いに上へ
			return [][2]int{{0, -1}}
		case !blockDown && dy > 0: // 壁沿いに下へ
			return [][2]int{{0, 1}}
		}
		return nil
	}

	// 縦横一直線の場合: 直進 → プレイヤー側へ斜めに回り込む
	candidates := [][2]int{{sign(dx), 0}, {0, sign(dy)}}
	switch {
	case dx > 0: // プレイヤーは右
		candidates = append(candidates, [2]int{1, 1}, [2]int{1, -1})
	case dx < 0: // プレイヤーは左
		candidates = append(candidates, [2]int{-1, 1}, [2]int{-1, -1})
	case dy > 0: // プレイヤーは下
		candidates = append(candidates, [2]int{1, 1}, [2]int{-1, 1})
	case dy < 0: // プレイヤーは上
		candidates = append(candidates, [2]int{1, -1}, [2]int{-1, -1})
	}
	return candidates
}

func (g *Game) moveEnemyConfused(i int) {
	enemy := &g.state.Enemies[i]

	// 8方向のランダムな移動先を選択
	directions := getDirections()

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

func (g *Game) moveEnemyBlind(i int) {
	enemy := &g.state.Enemies[i]

	// 方向が初期化されていない場合、ランダムに設定
	if enemy.Direction == Uninitialized {
		directions := []Direction{Up, Down, Left, Right, UpRight, UpLeft, DownRight, DownLeft}
		enemy.Direction = directions[rand.Intn(len(directions))]
	}

	// 現在の方向に基づいて移動先を計算
	dx, dy := directionToDelta(enemy.Direction)

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
	for i := range g.state.Enemies {
		g.actEnemy(i)
		// 倍速状態の敵はもう一度行動する
		if g.state.Enemies[i].StatusAilments.Haste > 0 {
			g.actEnemy(i)
		}
	}
}

// actEnemy はインデックスで指定した敵を1回行動させる
func (g *Game) actEnemy(i int) {
	enemy := g.state.Enemies[i]

	// 仮眠状態の敵の起床チェック（隣接時）
	if enemy.StatusAilments.Sleep == -1 {
		g.WakeUpSleepingEnemyByProximity(i)
	}

	// 睡眠状態の敵は移動できない
	if enemy.StatusAilments.Sleep > 0 || enemy.StatusAilments.Sleep == -1 {
		return
	}

	// 金縛り状態の敵は移動も攻撃もできない
	if enemy.StatusAilments.Paralysis {
		return
	}

	// 混乱状態の敵は周囲8マスからランダムに移動
	if enemy.StatusAilments.Confusion > 0 {
		g.moveEnemyConfused(i)
		return
	}

	// 目潰し状態の敵は直進移動
	if enemy.StatusAilments.Blind > 0 {
		g.moveEnemyBlind(i)
		return
	}

	// 盗品を持つ敵は、通常の追跡より逃走を優先する。
	if enemy.Fleeing && enemy.HeldItem != nil {
		g.moveFleeingEnemy(i)
		return
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

	// 射程・射線条件を満たす遠距離役は、接近する前に固有の攻撃を行う。
	// 封印・混乱・目潰し・睡眠などは上の共通状態処理で無効化される。
	if g.tryEnemyRangedAttack(i) {
		return
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

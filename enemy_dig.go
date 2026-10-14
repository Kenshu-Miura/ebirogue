//go:build !test

package main

// 壁を掘る処理と「近づくまで追ってこない」行動の共通処理（段4の共通基盤。アナジャコ系など）。

// digWallAt は (x, y) の壁を通路に変え、掘れたら true を返す。つるはしと同じ条件（isDiggableTile:
// 通れない壁で、マップの外周ではない）で判定する。掘った通路は、海老さんから見えている（明るい）
// マスなら踏破済みにしてミニマップへ出す。
func (g *Game) digWallAt(x, y int) bool {
	if len(g.state.Map) == 0 || len(g.state.Map[0]) == 0 {
		return false
	}
	width, height := len(g.state.Map[0]), len(g.state.Map)
	if x < 0 || y < 0 || x >= width || y >= height || !isDiggableTile(g.state.Map[y][x], x, y, width, height) {
		return false
	}
	tile := &g.state.Map[y][x]
	tile.Type = "corridor"
	tile.Blocked = false
	tile.BlockSight = false
	g.updateTileBrightness()
	if tile.Brightness == 1.0 {
		tile.Visited = true
	}
	g.miniMapDirty = true
	return true
}

// stepEnemyDigging は敵 i を (dx, dy) へ1マス動かす。行き先が掘れる壁なら掘ってから進む（同じ行動で）。
// 斜めの壁は掘らない（海老さんが斜めに通れない穴を作らないため）。動けたら true。
func (g *Game) stepEnemyDigging(i, dx, dy int) bool {
	if dx == 0 && dy == 0 {
		return false
	}
	enemy := &g.state.Enemies[i]
	x, y := enemy.X+dx, enemy.Y+dy
	if (dx == 0 || dy == 0) && g.inMapBounds(x, y) && g.state.Map[y][x].Blocked {
		if !g.digWallAt(x, y) {
			return false
		}
	}
	if !g.tryMoveEnemy(i, dx, dy) {
		return false
	}
	g.state.Enemies[i].Direction = determineDirection(dx, dy)
	return true
}

// stepEnemyTowardDigging は敵 i を (targetX, targetY) へ1マス近づける。間に壁があれば縦か横に掘って進む。
// 離れている向き（縦・横）を先に試し、掘っても進めなければ普通の接近（stepEnemyToward）に任せる。
func (g *Game) stepEnemyTowardDigging(i, targetX, targetY int) bool {
	enemy := g.state.Enemies[i]
	dx, dy := targetX-enemy.X, targetY-enemy.Y
	if dx == 0 && dy == 0 {
		return false
	}
	// 斜めに通れるならそのまま進む
	if dx != 0 && dy != 0 && g.tryMoveEnemy(i, sign(dx), sign(dy)) {
		return true
	}
	steps := [][2]int{{sign(dx), 0}, {0, sign(dy)}}
	if abs(dy) > abs(dx) {
		steps[0], steps[1] = steps[1], steps[0]
	}
	for _, step := range steps {
		if (step[0] != 0 || step[1] != 0) && g.stepEnemyDigging(i, step[0], step[1]) {
			return true
		}
	}
	return g.stepEnemyToward(i, targetX, targetY)
}

// enemyHoldsUntilNear は「近づくまで追ってこない」敵 i が、まだ海老さんを追わないなら true を返す。
// 海老さんがチェビシェフ距離 radius 以内に来るか、敵がダメージを受けている（HP が減っている）と追い始め
// （Enemy.Alerted を true にし）、以後は false を返す。追わない間の動き（うろつく・掘るなど）は呼び出し側が決める。
func (g *Game) enemyHoldsUntilNear(i, radius int) bool {
	enemy := &g.state.Enemies[i]
	if enemy.Alerted {
		return false
	}
	near := max(abs(enemy.X-g.state.Player.X), abs(enemy.Y-g.state.Player.Y)) <= radius
	if near || enemy.Health < enemy.MaxHealth {
		enemy.Alerted = true
		enemy.PlayerDiscovered = true
		return false
	}
	return true
}

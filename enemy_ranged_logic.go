package main

// straightLineDirection は2点が縦・横・斜め45度の直線上にある場合、
// 始点から終点へ向かう1マス分の移動量と距離を返す。
func straightLineDirection(fromX, fromY, toX, toY int) (stepX, stepY, distance int, ok bool) {
	dx := toX - fromX
	dy := toY - fromY
	if dx == 0 && dy == 0 {
		return 0, 0, 0, false
	}
	if dx != 0 && dy != 0 && abs(dx) != abs(dy) {
		return 0, 0, 0, false
	}
	return sign(dx), sign(dy), max(abs(dx), abs(dy)), true
}

// hasClearStraightLine は射程内の直線上に壁・閉じた扉・別の敵がないかを返す。
func hasClearStraightLine(mapState [][]Tile, enemies []Enemy, attackerIndex, fromX, fromY, toX, toY, maxRange int) bool {
	stepX, stepY, distance, ok := straightLineDirection(fromX, fromY, toX, toY)
	if !ok || distance > maxRange || len(mapState) == 0 {
		return false
	}

	for step := 1; step < distance; step++ {
		x := fromX + step*stepX
		y := fromY + step*stepY
		if y < 0 || y >= len(mapState) || x < 0 || x >= len(mapState[y]) {
			return false
		}
		tile := mapState[y][x]
		if tile.Blocked || tile.BlockSight {
			return false
		}
		for i, enemy := range enemies {
			if i != attackerIndex && enemy.X == x && enemy.Y == y {
				return false
			}
		}
	}
	return true
}

func withinRangedDistance(fromX, fromY, toX, toY, minRange, maxRange int) bool {
	distance := max(abs(toX-fromX), abs(toY-fromY))
	return distance >= minRange && distance <= maxRange
}

func withinBlastRadius(centerX, centerY, targetX, targetY, radius int) bool {
	return max(abs(targetX-centerX), abs(targetY-centerY)) <= radius
}

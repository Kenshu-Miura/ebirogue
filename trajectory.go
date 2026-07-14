package main

// computeTrajectory は始点 (x, y) から (dx, dy) 方向へ最大 throwRange マス進む
// 矢・杖の魔法弾・投擲アイテムの射線を計算する。
// 戻り値は通過するマスの列（到達地点を含む）、到達地点、敵に命中するかどうか。
// stopOnWallTile が true の場合（杖の魔法弾）は壁のマス自体が到達地点となり、
// false の場合は壁の1マス手前が到達地点となる（ThrowItem と同じ挙動）。
func computeTrajectory(x, y, dx, dy, throwRange int, mapState [][]Tile, enemies []Enemy, stopOnWallTile bool) ([]Coordinate, Coordinate, bool) {
	path := []Coordinate{}
	if (dx == 0 && dy == 0) || len(mapState) == 0 {
		return path, Coordinate{X: x, Y: y}, false
	}
	for i := 1; i <= throwRange; i++ {
		targetX, targetY := x+i*dx, y+i*dy
		outOfBounds := targetY < 0 || targetY >= len(mapState) || targetX < 0 || targetX >= len(mapState[0])
		if outOfBounds || mapState[targetY][targetX].Type == "wall" {
			if stopOnWallTile && !outOfBounds {
				// 杖の魔法弾は壁のマス自体に到達する
				landing := Coordinate{X: targetX, Y: targetY}
				path = append(path, landing)
				return path, landing, false
			}
			// 矢・投擲アイテムは壁の1マス手前に落ちる
			return path, Coordinate{X: x + (i-1)*dx, Y: y + (i-1)*dy}, false
		}
		path = append(path, Coordinate{X: targetX, Y: targetY})
		for _, enemy := range enemies {
			if enemy.X == targetX && enemy.Y == targetY {
				return path, Coordinate{X: targetX, Y: targetY}, true
			}
		}
	}
	// 何にも当たらなかった場合は最大射程の地点に到達する
	return path, Coordinate{X: x + throwRange*dx, Y: y + throwRange*dy}, false
}

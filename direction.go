//go:build !test

package main

// 方向（Direction）と1マス分の移動量 (dx, dy) の相互変換をまとめたファイル。

// determineDirection は移動量 (dx, dy) から向きを返す。
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
		return Up // デフォルトは上向き
	}
}

// directionToDelta は向きから1マス分の移動量 (dx, dy) を返す。
func directionToDelta(direction Direction) (int, int) {
	switch direction {
	case Up:
		return 0, -1
	case Down:
		return 0, 1
	case Left:
		return -1, 0
	case Right:
		return 1, 0
	case UpRight:
		return 1, -1
	case DownRight:
		return 1, 1
	case UpLeft:
		return -1, -1
	case DownLeft:
		return -1, 1
	}
	return 0, 0
}

// getDirections は8方向の移動量一覧を返す。
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

// getMainDirections は上下左右4方向の移動量一覧を返す。
func getMainDirections() []struct{ dx, dy int } {
	return []struct{ dx, dy int }{
		{0, -1}, // Up
		{0, 1},  // Down
		{-1, 0}, // Left
		{1, 0},  // Right
	}
}

// getLeftRightDirections は現在の方向に対する左右の移動量を返す。
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

// getOppositeDirection は逆方向の移動量を返す。
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

package main

// 足踏み・ダッシュを自動停止させるための危険判定ヘルパー。
// テストビルドでも動作するよう、両ビルドで共通の型のみを使う純粋関数として実装する。

// enemyWithinDistance は (px, py) からチェビシェフ距離 dist 以内に敵がいるかを判定する
func enemyWithinDistance(px, py, dist int, enemies []Enemy) bool {
	for i := range enemies {
		if abs(enemies[i].X-px) <= dist && abs(enemies[i].Y-py) <= dist {
			return true
		}
	}
	return false
}

// corridorBranch は (x, y) から (dx, dy) 方向へ進むとき、
// 進行方向と垂直な方向に通路の分岐があるかを判定する（斜め移動と停止中は対象外）
func corridorBranch(mapGrid [][]Tile, x, y, dx, dy int) bool {
	if dx == 0 && dy == 0 {
		return false
	}
	if dx != 0 && dy != 0 {
		return false
	}
	// 進行方向と垂直な2方向のタイルを調べる
	sides := []struct{ sx, sy int }{
		{x + dy, y + dx},
		{x - dy, y - dx},
	}
	for _, s := range sides {
		if s.sy < 0 || s.sy >= len(mapGrid) || s.sx < 0 || s.sx >= len(mapGrid[0]) {
			continue
		}
		if mapGrid[s.sy][s.sx].Type == "corridor" {
			return true
		}
	}
	return false
}

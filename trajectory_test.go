package main

import "testing"

// makeTrajectoryMap は外周が壁で内側が床のマップを生成するテスト用ヘルパー
func makeTrajectoryMap(w, h int) [][]Tile {
	grid := makeTileGrid(w, h, "floor")
	for x := 0; x < w; x++ {
		grid[0][x] = Tile{Type: "wall"}
		grid[h-1][x] = Tile{Type: "wall"}
	}
	for y := 0; y < h; y++ {
		grid[y][0] = Tile{Type: "wall"}
		grid[y][w-1] = Tile{Type: "wall"}
	}
	return grid
}

func TestComputeTrajectoryMaxRange(t *testing.T) {
	// 20x20の床マップで右方向へ射程10：何にも当たらず最大射程に到達する
	mapState := makeTrajectoryMap(20, 20)
	path, landing, hitEnemy := computeTrajectory(2, 10, 1, 0, 10, mapState, nil, false)

	if hitEnemy {
		t.Error("敵がいないのに命中と判定された")
	}
	if landing.X != 12 || landing.Y != 10 {
		t.Errorf("最大射程の到達地点が (12, 10) ではなく (%d, %d)", landing.X, landing.Y)
	}
	if len(path) != 10 {
		t.Errorf("通過マス数が10ではなく%d", len(path))
	}
	if path[0].X != 3 || path[0].Y != 10 {
		t.Errorf("射線の始点が (3, 10) ではなく (%d, %d)", path[0].X, path[0].Y)
	}
}

func TestComputeTrajectoryWallStop(t *testing.T) {
	// 矢・投擲アイテムは壁の1マス手前に落ちる
	mapState := makeTrajectoryMap(10, 10)
	path, landing, hitEnemy := computeTrajectory(5, 5, 1, 0, 10, mapState, nil, false)

	if hitEnemy {
		t.Error("敵がいないのに命中と判定された")
	}
	if landing.X != 8 || landing.Y != 5 {
		t.Errorf("壁の手前 (8, 5) ではなく (%d, %d) に到達した", landing.X, landing.Y)
	}
	// 通過マスは (6,5) (7,5) (8,5) の3マスで壁は含まない
	if len(path) != 3 {
		t.Errorf("通過マス数が3ではなく%d", len(path))
	}
}

func TestComputeTrajectoryStopOnWallTile(t *testing.T) {
	// 杖の魔法弾は壁のマス自体に到達する
	mapState := makeTrajectoryMap(10, 10)
	path, landing, hitEnemy := computeTrajectory(5, 5, 1, 0, 30, mapState, nil, true)

	if hitEnemy {
		t.Error("敵がいないのに命中と判定された")
	}
	if landing.X != 9 || landing.Y != 5 {
		t.Errorf("壁のマス (9, 5) ではなく (%d, %d) に到達した", landing.X, landing.Y)
	}
	if len(path) == 0 || path[len(path)-1] != landing {
		t.Errorf("射線の終端が到達地点と一致しない: %v", path)
	}
}

func TestComputeTrajectoryEnemyHit(t *testing.T) {
	// 射線上の敵に命中して止まる
	mapState := makeTrajectoryMap(20, 20)
	enemies := []Enemy{{Entity: Entity{X: 8, Y: 10}}}
	path, landing, hitEnemy := computeTrajectory(2, 10, 1, 0, 10, mapState, enemies, false)

	if !hitEnemy {
		t.Error("射線上の敵に命中しなかった")
	}
	if landing.X != 8 || landing.Y != 10 {
		t.Errorf("敵の位置 (8, 10) ではなく (%d, %d) に到達した", landing.X, landing.Y)
	}
	if len(path) != 6 {
		t.Errorf("通過マス数が6ではなく%d", len(path))
	}
}

func TestComputeTrajectoryEnemyBehindWall(t *testing.T) {
	// 壁の向こうの敵には命中しない
	mapState := makeTrajectoryMap(20, 20)
	mapState[10][6] = Tile{Type: "wall"}
	enemies := []Enemy{{Entity: Entity{X: 8, Y: 10}}}
	_, landing, hitEnemy := computeTrajectory(2, 10, 1, 0, 10, mapState, enemies, false)

	if hitEnemy {
		t.Error("壁の向こうの敵に命中と判定された")
	}
	if landing.X != 5 || landing.Y != 10 {
		t.Errorf("壁の手前 (5, 10) ではなく (%d, %d) に到達した", landing.X, landing.Y)
	}
}

func TestComputeTrajectoryAdjacentWall(t *testing.T) {
	// 目の前が壁の場合は足元（始点）に落ちる
	mapState := makeTrajectoryMap(10, 10)
	path, landing, hitEnemy := computeTrajectory(8, 5, 1, 0, 10, mapState, nil, false)

	if hitEnemy {
		t.Error("敵がいないのに命中と判定された")
	}
	if landing.X != 8 || landing.Y != 5 {
		t.Errorf("足元 (8, 5) ではなく (%d, %d) に到達した", landing.X, landing.Y)
	}
	if len(path) != 0 {
		t.Errorf("通過マスは無いはずが%dマスあった", len(path))
	}
}

func TestComputeTrajectoryDiagonal(t *testing.T) {
	// 斜め方向の射線も直線で計算される
	mapState := makeTrajectoryMap(20, 20)
	enemies := []Enemy{{Entity: Entity{X: 6, Y: 6}}}
	path, landing, hitEnemy := computeTrajectory(3, 3, 1, 1, 10, mapState, enemies, false)

	if !hitEnemy {
		t.Error("斜めの射線上の敵に命中しなかった")
	}
	if landing.X != 6 || landing.Y != 6 {
		t.Errorf("敵の位置 (6, 6) ではなく (%d, %d) に到達した", landing.X, landing.Y)
	}
	if len(path) != 3 {
		t.Errorf("通過マス数が3ではなく%d", len(path))
	}
}

func TestComputeTrajectoryNoDirection(t *testing.T) {
	// 方向が未設定の場合は始点に留まる
	mapState := makeTrajectoryMap(10, 10)
	path, landing, hitEnemy := computeTrajectory(5, 5, 0, 0, 10, mapState, nil, false)

	if hitEnemy || len(path) != 0 {
		t.Errorf("方向未設定でも射線が計算された: path=%v", path)
	}
	if landing.X != 5 || landing.Y != 5 {
		t.Errorf("始点 (5, 5) ではなく (%d, %d) に到達した", landing.X, landing.Y)
	}
}

func TestComputeTrajectoryOutOfBounds(t *testing.T) {
	// マップ外は壁と同様に扱い、範囲外アクセスしない（杖でもマップ外には到達しない）
	mapState := makeTileGrid(10, 10, "floor") // 外周も床のマップ
	_, landing, hitEnemy := computeTrajectory(5, 5, 1, 0, 30, mapState, nil, true)

	if hitEnemy {
		t.Error("敵がいないのに命中と判定された")
	}
	if landing.X != 9 || landing.Y != 5 {
		t.Errorf("マップ端 (9, 5) ではなく (%d, %d) に到達した", landing.X, landing.Y)
	}
}

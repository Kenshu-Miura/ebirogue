package main

import "testing"

// makeTileGrid は指定サイズ・タイル種別のマップを生成するテスト用ヘルパー
func makeTileGrid(w, h int, typ string) [][]Tile {
	grid := make([][]Tile, h)
	for y := range grid {
		grid[y] = make([]Tile, w)
		for x := range grid[y] {
			grid[y][x] = Tile{Type: typ}
		}
	}
	return grid
}

func TestEnemyWithinDistance(t *testing.T) {
	enemies := []Enemy{{Entity: Entity{X: 5, Y: 5}}}

	tests := []struct {
		name   string
		px, py int
		dist   int
		want   bool
	}{
		{"斜め1マス", 4, 4, 2, true},
		{"距離ちょうど2マス", 5, 7, 2, true},
		{"3マス離れている", 5, 8, 2, false},
		{"遠く離れている", 1, 1, 2, false},
		{"同じマス", 5, 5, 2, true},
	}
	for _, tt := range tests {
		if got := enemyWithinDistance(tt.px, tt.py, tt.dist, enemies); got != tt.want {
			t.Errorf("%s: enemyWithinDistance(%d, %d, %d) = %v, want %v",
				tt.name, tt.px, tt.py, tt.dist, got, tt.want)
		}
	}

	if enemyWithinDistance(5, 5, 2, nil) {
		t.Errorf("敵がいない場合は false を返すべき")
	}
}

func TestCorridorBranchDetectsSideCorridor(t *testing.T) {
	// 中央 (2,2) を上方向へ移動中、右 (3,2) が通路なら分岐と判定する
	grid := makeTileGrid(5, 5, "wall")
	grid[2][2] = Tile{Type: "corridor"}
	grid[1][2] = Tile{Type: "corridor"}
	grid[2][3] = Tile{Type: "corridor"}

	if !corridorBranch(grid, 2, 2, 0, -1) {
		t.Errorf("横に通路がある場合は分岐と判定すべき")
	}
}

func TestCorridorBranchNoBranch(t *testing.T) {
	// 左右が壁の一本道では分岐と判定しない
	grid := makeTileGrid(5, 5, "wall")
	grid[1][2] = Tile{Type: "corridor"}
	grid[2][2] = Tile{Type: "corridor"}
	grid[3][2] = Tile{Type: "corridor"}

	if corridorBranch(grid, 2, 2, 0, -1) {
		t.Errorf("一本道では分岐と判定すべきでない")
	}
}

func TestCorridorBranchHorizontalMove(t *testing.T) {
	// 右方向へ移動中、下 (2,3) が通路なら分岐と判定する
	grid := makeTileGrid(5, 5, "wall")
	grid[2][2] = Tile{Type: "corridor"}
	grid[2][3] = Tile{Type: "corridor"}
	grid[3][2] = Tile{Type: "corridor"}

	if !corridorBranch(grid, 2, 2, 1, 0) {
		t.Errorf("上下に通路がある場合は分岐と判定すべき")
	}
}

func TestCorridorBranchIgnoresDiagonalAndIdle(t *testing.T) {
	grid := makeTileGrid(5, 5, "corridor")

	if corridorBranch(grid, 2, 2, 1, 1) {
		t.Errorf("斜め移動では分岐判定しない")
	}
	if corridorBranch(grid, 2, 2, 0, 0) {
		t.Errorf("停止中は分岐判定しない")
	}
}

func TestCorridorBranchOutOfBounds(t *testing.T) {
	// マップ端でも範囲外アクセスせず false を返す
	grid := makeTileGrid(3, 3, "wall")

	if corridorBranch(grid, 0, 0, 0, -1) {
		t.Errorf("範囲外のタイルは分岐と判定すべきでない")
	}
	if corridorBranch(grid, 2, 2, 1, 0) {
		t.Errorf("範囲外のタイルは分岐と判定すべきでない")
	}
}

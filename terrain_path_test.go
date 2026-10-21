package main

import "testing"

// pathTestGrid は文字列からマップを作る（'.' 床、'#' 壁、'~' 水路）。
func pathTestGrid(rows ...string) [][]Tile {
	grid := make([][]Tile, len(rows))
	for y, row := range rows {
		grid[y] = make([]Tile, len(row))
		for x, c := range row {
			switch c {
			case '#':
				grid[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
			case '~':
				grid[y][x] = Tile{Type: "water", Blocked: true}
			default:
				grid[y][x] = Tile{Type: "floor"}
			}
		}
	}
	return grid
}

func TestEnemyPathStepGoesAroundWater(t *testing.T) {
	grid := pathTestGrid(
		"#########",
		"#.......#",
		"#.~~~~~.#",
		"#.~~~~~.#",
		"#.......#",
		"#########",
	)
	// 敵 (4,4) から海老さん (4,1) へ。真上は水路なので横へ回る。
	dx, dy, ok := enemyPathStep(grid, 4, 4, 4, 1, nil, enemyPathMaxSteps, nil)
	if !ok || dy == -1 && dx == 0 {
		t.Fatalf("step = %d,%d ok=%v", dx, dy, ok)
	}
	// 1歩ずつたどると目標の隣に着く。
	x, y := 4, 4
	for i := 0; i < 10 && max(abs(x-4), abs(y-1)) > 1; i++ {
		dx, dy, ok := enemyPathStep(grid, x, y, 4, 1, nil, enemyPathMaxSteps, nil)
		if !ok || grid[y+dy][x+dx].Blocked {
			t.Fatalf("stuck at %d,%d", x, y)
		}
		x, y = x+dx, y+dy
	}
	if max(abs(x-4), abs(y-1)) > 1 {
		t.Fatalf("did not arrive: %d,%d", x, y)
	}
	// 浮遊の敵は水路の上をまっすぐ進む。
	if dx, dy, ok := enemyPathStep(grid, 4, 4, 4, 1, []EnemyTrait{EnemyTraitFloating}, enemyPathMaxSteps, nil); !ok || dx != 0 || dy != -1 {
		t.Fatalf("floating step = %d,%d ok=%v", dx, dy, ok)
	}
	// 回り道をふさぐと経路は無い。
	blocked := func(x, y int) bool { return y == 2 || y == 3 }
	if _, _, ok := enemyPathStep(grid, 4, 4, 4, 1, nil, enemyPathMaxSteps, blocked); ok {
		t.Fatal("path through blocked tiles")
	}
	// 歩数の上限より遠ければ見つけない。
	if _, _, ok := enemyPathStep(grid, 4, 4, 4, 1, nil, 3, nil); ok {
		t.Fatal("path longer than maxSteps")
	}
}

// 斜めは縦横の隣が Blocked なら進まない（ゲームの敵の移動と同じ規則）。
func TestEnemyPathStepNoCornerCutting(t *testing.T) {
	grid := pathTestGrid(
		"#####",
		"#..##",
		"#.~.#",
		"#####",
	)
	dx, dy, ok := enemyPathStep(grid, 3, 2, 2, 1, nil, enemyPathMaxSteps, nil)
	if ok {
		t.Fatalf("cut the corner: %d,%d", dx, dy)
	}
}

func TestWaterBetween(t *testing.T) {
	grid := pathTestGrid(
		".....",
		"..~..",
		".....",
	)
	if !waterBetween(grid, 0, 0, 4, 2) || !waterBetween(grid, 2, 2, 2, 0) {
		t.Fatal("water not found")
	}
	if waterBetween(grid, 0, 0, 1, 2) || waterBetween(grid, 3, 0, 4, 2) {
		t.Fatal("water outside the rectangle")
	}
}

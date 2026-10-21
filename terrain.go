package main

// 地形の抽選率（非ボス階、フロアごとに1回）。調整用の数値はここに集約する。
var terrainRates = []struct{ from, water, cracked, hidden int }{
	{1, 0, 0, 0}, {2, 0, 50, 0}, {4, 45, 50, 0}, {6, 45, 50, 40},
}

func terrainBlocksWalking(tile Tile) bool {
	return tile.Blocked || tile.Type == "water" || tile.Type == "hidden_corridor" || tile.Type == "cracked_wall"
}

func terrainStopsProjectile(tile Tile) bool {
	return tile.Type != "water" && (tile.Blocked || tile.Type == "wall" || tile.Type == "cracked_wall" || tile.Type == "hidden_corridor")
}

func enemyTerrainPassable(tile Tile, traits []EnemyTrait) bool {
	if tile.Type == "hidden_corridor" {
		return true
	}
	if tile.Type == "water" {
		return hasEnemyTrait(traits, EnemyTraitFloating) || hasEnemyTrait(traits, EnemyTraitAquatic)
	}
	return !tile.Blocked
}

// 既存の配置を避け、元からある通常の通路を一切消さずに地形を足す。
// protect は海老さん・階段・敵・道具・罠の座標。
func generateTerrain(grid [][]Tile, rooms []Room, floor int, intn func(int) int, protect map[Coordinate]bool) {
	if isBossFloor(floor) {
		return
	}
	rate := terrainRates[0]
	for _, r := range terrainRates {
		if floor >= r.from {
			rate = r
		}
	}
	if rate.hidden > 0 && intn(100) < rate.hidden {
		used := spanningTreeEdges(rooms)
		if e, ok := extraLoopEdge(rooms, used); ok {
			used = append(used, e)
		}
		if e, ok := extraLoopEdge(rooms, used); ok {
			copyGrid := make([][]Tile, len(grid))
			for y := range grid {
				copyGrid[y] = append([]Tile(nil), grid[y]...)
			}
			for _, leg := range planCorridor(rooms[e[0]].Center, rooms[e[1]].Center, rooms) {
				carveLeg(copyGrid, leg, rooms)
			}
			for y := 1; y < len(grid)-1; y++ {
				for x := 1; x < len(grid[y])-1; x++ {
					if grid[y][x].Blocked && copyGrid[y][x].Type == "corridor" && !protect[Coordinate{X: x, Y: y}] {
						grid[y][x] = Tile{Type: "hidden_corridor", Blocked: true, BlockSight: true}
					}
				}
			}
		}
	}
	if rate.water > 0 && intn(100) < rate.water {
		for _, room := range rooms {
			// 壁際・出入口・中心は残す。小さい部屋には無理に作らない。
			budget := (room.Width - 2) * (room.Height - 2) / 4
			x0, y0 := room.X+2, room.Y+2
			if room.Width < 7 || room.Height < 7 {
				continue
			}
			x0 += intn(room.Width - 4)
			y0 += intn(room.Height - 4)
			for y := y0; y < room.Y+room.Height-2 && y < y0+2; y++ {
				for x := x0; x < room.X+room.Width-2 && budget > 0; x++ {
					p := Coordinate{X: x, Y: y}
					if protect[p] || p == room.Center || grid[y][x].Type != "floor" {
						continue
					}
					old := grid[y][x]
					grid[y][x] = Tile{Type: "water", Blocked: true}
					if !floorConnected(grid, rooms) {
						grid[y][x] = old
					} else {
						budget--
					}
				}
			}
		}
	}
	if rate.cracked > 0 && intn(100) < rate.cracked {
		for _, room := range rooms {
			x, y := room.X+intn(room.Width), room.Y
			if intn(2) == 0 {
				y = room.Y + room.Height - 1
			}
			if x > 0 && y > 0 && y < len(grid)-1 && x < len(grid[y])-1 && grid[y][x].Type == "wall" {
				grid[y][x].Type = "cracked_wall"
			}
		}
		// 通路の脇にも少数だけ置く。通常の通路自体は塞がない。
		var sides []Coordinate
		for y := 1; y < len(grid)-1; y++ {
			for x := 1; x < len(grid[y])-1; x++ {
				if grid[y][x].Type != "other" {
					continue
				}
				for _, d := range []Coordinate{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
					if grid[y+d.Y][x+d.X].Type == "corridor" {
						sides = append(sides, Coordinate{X: x, Y: y})
						break
					}
				}
			}
		}
		for n := 0; n < 2 && len(sides) > 0; n++ {
			i := intn(len(sides))
			p := sides[i]
			grid[p.Y][p.X] = Tile{Type: "cracked_wall", Blocked: true, BlockSight: true}
			sides = append(sides[:i], sides[i+1:]...)
		}
	}
}

// enemyPathMaxSteps は水路を回り込む経路探索の最大の歩数。
const enemyPathMaxSteps = 40

// waterBetween は (x0, y0) と (x1, y1) を対角とする長方形（両端を含む）に水路があるかどうか。
func waterBetween(grid [][]Tile, x0, y0, x1, y1 int) bool {
	for y := min(y0, y1); y <= max(y0, y1); y++ {
		if y < 0 || y >= len(grid) {
			continue
		}
		for x := min(x0, x1); x <= max(x0, x1); x++ {
			if x >= 0 && x < len(grid[y]) && grid[y][x].Type == "water" {
				return true
			}
		}
	}
	return false
}

// enemyPathSteps は経路探索で試す向き（縦横を先、斜めを後）。順番は結果の決め方なので変えない。
var enemyPathSteps = [8][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}, {1, -1}, {1, 1}, {-1, 1}, {-1, -1}}

// enemyPathStep は、traits の敵が通れるマスだけを通って (fx, fy) から (tx, ty) へ向かう最短経路の
// 最初の1歩を返す（maxSteps 歩まで）。目標のマスは通れなくてもよい。blocked のマス（聖域など）は通らない。
// 斜めは、ゲームの敵の移動と同じく、縦横の隣のどちらかが Blocked なら進まない。ほかの敵は考えない。
// 乱数は使わない。
func enemyPathStep(grid [][]Tile, fx, fy, tx, ty int, traits []EnemyTrait, maxSteps int, blocked func(x, y int) bool) (int, int, bool) {
	if len(grid) == 0 || (fx == tx && fy == ty) {
		return 0, 0, false
	}
	height, width := len(grid), len(grid[0])
	inside := func(x, y int) bool { return x >= 0 && y >= 0 && y < height && x < width }
	first := make([]int8, width*height) // 0: 未訪問、1〜8: 最初の1歩の向き+1、-1: 出発点
	dist := make([]int, width*height)
	start := fy*width + fx
	first[start] = -1
	queue := []int{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if dist[cur] >= maxSteps {
			continue
		}
		x, y := cur%width, cur/width
		for dir, s := range enemyPathSteps {
			nx, ny := x+s[0], y+s[1]
			if !inside(nx, ny) {
				continue
			}
			next := ny*width + nx
			if first[next] != 0 {
				continue
			}
			if s[0] != 0 && s[1] != 0 && (grid[y][nx].Blocked || grid[ny][x].Blocked) {
				continue
			}
			goal := nx == tx && ny == ty
			if !goal && (!enemyTerrainPassable(grid[ny][nx], traits) || (blocked != nil && blocked(nx, ny))) {
				continue
			}
			if cur == start {
				first[next] = int8(dir + 1)
			} else {
				first[next] = first[cur]
			}
			if goal {
				step := enemyPathSteps[first[next]-1]
				return step[0], step[1], true
			}
			dist[next] = dist[cur] + 1
			queue = append(queue, next)
		}
	}
	return 0, 0, false
}

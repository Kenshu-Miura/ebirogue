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

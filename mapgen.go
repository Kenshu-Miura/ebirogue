package main

// マップ生成の純粋ロジック（部屋・通路の計画と接続判定）。
// Ebiten に依存しないため通常ビルドとテストビルドの両方でコンパイルされる。

import "math"

type Room struct {
	ID            int
	X, Y          int
	Width, Height int
	Center        Coordinate
}

func (r *Room) IsSeparatedBy(other Room, tiles int) bool {
	// Horizontal separation
	if r.X+r.Width+tiles <= other.X || other.X+other.Width+tiles <= r.X {
		return true
	}
	// Vertical separation
	if r.Y+r.Height+tiles <= other.Y || other.Y+other.Height+tiles <= r.Y {
		return true
	}
	return false
}

func setRoomCenter(room *Room) {
	// Calculate the center coordinates
	centerX := room.X + room.Width/2
	centerY := room.Y + room.Height/2

	// If the calculated center coordinates are even, increment them by 1 to make them odd
	// （中心座標を奇数に揃えることで、平行に走る通路同士が隣接して2列になるのを防ぐ）
	if centerX%2 == 0 {
		centerX++
	}
	if centerY%2 == 0 {
		centerY++
	}

	// Set the center coordinates
	room.Center = Coordinate{X: centerX, Y: centerY}
}

// carveRoom は部屋の外周を壁、内側を床としてマップへ書き込む
func carveRoom(mapGrid [][]Tile, room Room) {
	for y := room.Y; y < room.Y+room.Height; y++ {
		for x := room.X; x < room.X+room.Width; x++ {
			if x == room.X || x == room.X+room.Width-1 || y == room.Y || y == room.Y+room.Height-1 {
				mapGrid[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
			} else {
				mapGrid[y][x] = Tile{Type: "floor", Blocked: false, BlockSight: false}
			}
		}
	}
}

func isInsideRoom(x, y int, rooms []Room) bool {
	for _, room := range rooms {
		if x > room.X && x < room.X+room.Width-1 &&
			y > room.Y && y < room.Y+room.Height-1 {
			return true
		}
	}
	return false
}

func isInsideRoomOrOnBoundary(x, y int, rooms []Room) bool {
	for _, room := range rooms {
		if x >= room.X && x <= room.X+room.Width-1 &&
			y >= room.Y && y <= room.Y+room.Height-1 {
			return true
		}
	}
	return false
}

func isOnBoundary(x, y int, room Room) bool {
	left := room.X
	right := room.X + room.Width - 1
	top := room.Y
	bottom := room.Y + room.Height - 1

	// Check if (x, y) is on the left, right, top, or bottom edge of the room
	isOnLeftEdge := x == left && y >= top && y <= bottom
	isOnRightEdge := x == right && y >= top && y <= bottom
	isOnTopEdge := y == top && x >= left && x <= right
	isOnBottomEdge := y == bottom && x >= left && x <= right

	return isOnLeftEdge || isOnRightEdge || isOnTopEdge || isOnBottomEdge
}

// corridorLeg は軸に平行な通路の1区間（両端を含む。x1==x2 か y1==y2 のどちらかが成り立つ）
type corridorLeg struct {
	x1, y1, x2, y2 int
}

// 通路経路の減点の重み。壁と平行に重なる経路は壁を破壊するため大きく減点し、
// 壁を垂直に横切る（穴を1つ開けるだけの）経路は小さく減点する。
const (
	penaltyWallParallel = 10 // 部屋の壁ラインと平行に重なるタイル1枚ごと
	penaltyWallCross    = 1  // 部屋の壁を垂直に横切るごと
)

// overlapLength は閉区間 [aLo, aHi] と [bLo, bHi] の重なりタイル数を返す
func overlapLength(aLo, aHi, bLo, bHi int) int {
	return max(0, min(aHi, bHi)-max(aLo, bLo)+1)
}

// legPenalty は区間が部屋の壁をどれだけ損なうかを減点として返す
func legPenalty(leg corridorLeg, rooms []Room) int {
	penalty := 0
	xLo, xHi := min(leg.x1, leg.x2), max(leg.x1, leg.x2)
	yLo, yHi := min(leg.y1, leg.y2), max(leg.y1, leg.y2)
	for _, room := range rooms {
		left, right := room.X, room.X+room.Width-1
		top, bottom := room.Y, room.Y+room.Height-1
		if leg.x1 == leg.x2 { // 縦の区間
			if leg.x1 == left || leg.x1 == right {
				// 左右の壁ラインと平行に重なる（壁が通路化して部屋が開いてしまう）
				penalty += overlapLength(yLo, yHi, top, bottom) * penaltyWallParallel
			} else if leg.x1 > left && leg.x1 < right {
				// 上下の壁を垂直に横切る
				if yLo <= top && top <= yHi {
					penalty += penaltyWallCross
				}
				if yLo <= bottom && bottom <= yHi {
					penalty += penaltyWallCross
				}
			}
		} else { // 横の区間
			if leg.y1 == top || leg.y1 == bottom {
				penalty += overlapLength(xLo, xHi, left, right) * penaltyWallParallel
			} else if leg.y1 > top && leg.y1 < bottom {
				if xLo <= left && left <= xHi {
					penalty += penaltyWallCross
				}
				if xLo <= right && right <= xHi {
					penalty += penaltyWallCross
				}
			}
		}
	}
	return penalty
}

func corridorPenalty(legs []corridorLeg, rooms []Room) int {
	total := 0
	for _, leg := range legs {
		total += legPenalty(leg, rooms)
	}
	return total
}

// planCorridor は c1 から c2 への L 字経路のうち、部屋の壁を損なわない方を返す。
// 縦→横と横→縦の2通りを比較し、減点が少ない方を採用する。
func planCorridor(c1, c2 Coordinate, rooms []Room) []corridorLeg {
	vThenH := []corridorLeg{
		{c1.X, c1.Y, c1.X, c2.Y},
		{c1.X, c2.Y, c2.X, c2.Y},
	}
	hThenV := []corridorLeg{
		{c1.X, c1.Y, c2.X, c1.Y},
		{c2.X, c1.Y, c2.X, c2.Y},
	}
	if corridorPenalty(hThenV, rooms) < corridorPenalty(vThenH, rooms) {
		return hThenV
	}
	return vThenH
}

// carveLeg は区間を通路タイルとして掘る。部屋の内側（床）は変更せず、
// 壁・外周にかかるタイルは通路化する（部屋の出入口になる）。
func carveLeg(mapGrid [][]Tile, leg corridorLeg, rooms []Room) {
	for x := min(leg.x1, leg.x2); x <= max(leg.x1, leg.x2); x++ {
		for y := min(leg.y1, leg.y2); y <= max(leg.y1, leg.y2); y++ {
			if !isInsideRoom(x, y, rooms) {
				mapGrid[y][x] = Tile{Type: "corridor", Blocked: false, BlockSight: false}
			}
		}
	}
}

// roomDistSq は部屋の中心同士の距離の2乗を返す（比較専用なので平方根は取らない）
func roomDistSq(a, b Room) int {
	dx := a.Center.X - b.Center.X
	dy := a.Center.Y - b.Center.Y
	return dx*dx + dy*dy
}

// spanningTreeEdges は Prim 法で全部屋を結ぶ最小全域木の辺（部屋インデックスの組）を返す。
// 全部屋の連結を保証しつつ、生成順の循環接続で生じていた長大な通路を避ける。
func spanningTreeEdges(rooms []Room) [][2]int {
	n := len(rooms)
	if n < 2 {
		return nil
	}
	inTree := make([]bool, n)
	inTree[0] = true
	var edges [][2]int
	for len(edges) < n-1 {
		bestFrom, bestTo, bestDist := -1, -1, math.MaxInt
		for i := 0; i < n; i++ {
			if !inTree[i] {
				continue
			}
			for j := 0; j < n; j++ {
				if inTree[j] {
					continue
				}
				if d := roomDistSq(rooms[i], rooms[j]); d < bestDist {
					bestFrom, bestTo, bestDist = i, j, d
				}
			}
		}
		if bestTo < 0 {
			break
		}
		edges = append(edges, [2]int{bestFrom, bestTo})
		inTree[bestTo] = true
	}
	return edges
}

// extraLoopEdge は未使用の辺のうち最短のものを返し、一方通行にならない回遊路を作る
func extraLoopEdge(rooms []Room, used [][2]int) ([2]int, bool) {
	n := len(rooms)
	if n < 3 {
		return [2]int{}, false
	}
	usedSet := make(map[[2]int]bool, len(used))
	for _, e := range used {
		usedSet[[2]int{min(e[0], e[1]), max(e[0], e[1])}] = true
	}
	best, bestDist := [2]int{-1, -1}, math.MaxInt
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if usedSet[[2]int{i, j}] {
				continue
			}
			if d := roomDistSq(rooms[i], rooms[j]); d < bestDist {
				best, bestDist = [2]int{i, j}, d
			}
		}
	}
	if best[0] < 0 {
		return best, false
	}
	return best, true
}

// connectRooms は最小全域木＋ループ辺1本で全部屋を通路でつなぐ
func connectRooms(rooms []Room, mapGrid [][]Tile) {
	if len(rooms) < 2 {
		return
	}
	edges := spanningTreeEdges(rooms)
	if extra, ok := extraLoopEdge(rooms, edges); ok {
		edges = append(edges, extra)
	}
	for _, e := range edges {
		for _, leg := range planCorridor(rooms[e[0]].Center, rooms[e[1]].Center, rooms) {
			carveLeg(mapGrid, leg, rooms)
		}
	}
}

// roomWallsIntact は部屋の壁が通路で3タイル以上連続して置き換えられていないか
// （通路が壁と平行に走って部屋が開いてしまっていないか）を確認する
func roomWallsIntact(mapGrid [][]Tile, rooms []Room) bool {
	const maxRun = 2
	corridorRunTooLong := func(tiles []*Tile) bool {
		run := 0
		for _, tile := range tiles {
			if tile.Type == "corridor" {
				run++
				if run > maxRun {
					return true
				}
			} else {
				run = 0
			}
		}
		return false
	}
	for _, room := range rooms {
		left, right := room.X, room.X+room.Width-1
		top, bottom := room.Y, room.Y+room.Height-1
		for _, y := range []int{top, bottom} {
			var row []*Tile
			for x := left; x <= right; x++ {
				row = append(row, &mapGrid[y][x])
			}
			if corridorRunTooLong(row) {
				return false
			}
		}
		for _, x := range []int{left, right} {
			var col []*Tile
			for y := top; y <= bottom; y++ {
				col = append(col, &mapGrid[y][x])
			}
			if corridorRunTooLong(col) {
				return false
			}
		}
	}
	return true
}

// floorConnected は全部屋の中心が歩行可能タイル経由で互いに到達可能かを判定する
func floorConnected(mapGrid [][]Tile, rooms []Room) bool {
	if len(rooms) == 0 || len(mapGrid) == 0 {
		return false
	}
	height, width := len(mapGrid), len(mapGrid[0])
	start := rooms[0].Center
	if mapGrid[start.Y][start.X].Blocked {
		return false
	}
	visited := make([][]bool, height)
	for y := range visited {
		visited[y] = make([]bool, width)
	}
	queue := []Coordinate{start}
	visited[start.Y][start.X] = true
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := cur.X+d[0], cur.Y+d[1]
			if nx < 0 || ny < 0 || nx >= width || ny >= height {
				continue
			}
			if visited[ny][nx] || mapGrid[ny][nx].Blocked {
				continue
			}
			visited[ny][nx] = true
			queue = append(queue, Coordinate{X: nx, Y: ny})
		}
	}
	for _, room := range rooms {
		if !visited[room.Center.Y][room.Center.X] {
			return false
		}
	}
	return true
}

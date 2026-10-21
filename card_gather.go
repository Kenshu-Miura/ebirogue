package main

import "sort"

// gatherDistance はチェビシェフ距離。距離が同じなら上から下、左から右に並べる。
func gatherDistance(a, b Coordinate) int {
	dx, dy := a.X-b.X, a.Y-b.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

func sortGatherPositions(positions []Coordinate, center Coordinate) {
	sort.SliceStable(positions, func(i, j int) bool {
		a, b := positions[i], positions[j]
		if da, db := gatherDistance(a, center), gatherDistance(b, center); da != db {
			return da < db
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.X < b.X
	})
}

// gatherFloorPositions はマップと占有座標だけから、集合先の空き床を求める。
func gatherFloorPositions(tiles [][]Tile, center Coordinate, occupied []Coordinate) []Coordinate {
	blocked := make(map[Coordinate]bool, len(occupied)+1)
	blocked[center] = true
	for _, p := range occupied {
		blocked[p] = true
	}
	positions := []Coordinate{}
	for y, row := range tiles {
		for x, tile := range row {
			p := Coordinate{X: x, Y: y}
			if !tile.Blocked && (tile.Type == "floor" || tile.Type == "corridor") && !blocked[p] {
				positions = append(positions, p)
			}
		}
	}
	sortGatherPositions(positions, center)
	return positions
}

func rollGamble(cash int, intn func(int) int) (newCash int, won bool) {
	if intn(2) == 0 {
		return cash*2 + 100, true
	}
	return 0, false
}

type troubleKind int

const (
	troubleNone troubleKind = iota
	troubleStatus
	troubleHunger
	troubleCurse
	troublePower
)

// troubleToFix は回復の優先順位だけを決める。状態異常は悪いものだけを対象にする。
func troubleToFix(s StatusAilments, satiety int, cursed bool, power, maxPower int) troubleKind {
	if hasBadStatus(s) {
		return troubleStatus
	}
	if satiety == 0 {
		return troubleHunger
	}
	if cursed {
		return troubleCurse
	}
	if power < maxPower {
		return troublePower
	}
	return troubleNone
}

package main

// 吹き飛ばし・引き寄せ（content-expansion-plan.md の F3）の経路計算。Game に依存しない純粋関数。

// DisplacementStop は吹き飛ばし・引き寄せが止まった理由。
type DisplacementStop int

const (
	DisplacementFinished     DisplacementStop = iota // 指定のマス数を動き切った
	DisplacementHitWall                              // 壁（通れない地形・マップ外・角抜け）に当たった
	DisplacementHitCharacter                         // 敵・プレイヤーに当たった
)

// displacementPath は (x, y) から (dx, dy) 方向へ最大 distance マス進んだ経路を返す（出発点は含まない）。
// terrainOpen は地形として通れるか、occupied はキャラクターがいるかを返す。
// 斜めに進むときは、角の左右どちらかの地形が通れなければ角抜けになるため壁に当たった扱いにする
// （敵の斜め移動と同じ規則）。
func displacementPath(x, y, dx, dy, distance int, terrainOpen, occupied func(x, y int) bool) ([]Coordinate, DisplacementStop) {
	dx, dy = sign(dx), sign(dy)
	if dx == 0 && dy == 0 {
		return nil, DisplacementFinished
	}
	var path []Coordinate
	for step := 0; step < distance; step++ {
		nx, ny := x+dx, y+dy
		if !terrainOpen(nx, ny) || (dx != 0 && dy != 0 && (!terrainOpen(x+dx, y) || !terrainOpen(x, y+dy))) {
			return path, DisplacementHitWall
		}
		if occupied(nx, ny) {
			return path, DisplacementHitCharacter
		}
		path = append(path, Coordinate{X: nx, Y: ny})
		x, y = nx, ny
	}
	return path, DisplacementFinished
}

// forwardDropPositions は (x, y) から (dx, dy) 方向の1〜maxDistance マス先を近い順に返し、
// 続けて周囲8マスを返す（持ち物を前方へ落とす処理の置き場所の候補。重複なし）。
// 置けるかどうか（床か・道具や罠が無いか）は呼び出し側の free で判定する。
// 前方の途中に壁があれば、その先は候補にしない。
func forwardDropPositions(x, y, dx, dy, maxDistance int, free func(x, y int) bool) []Coordinate {
	dx, dy = sign(dx), sign(dy)
	seen := map[Coordinate]bool{}
	var positions []Coordinate
	add := func(c Coordinate) {
		if !seen[c] && free(c.X, c.Y) {
			seen[c] = true
			positions = append(positions, c)
		}
	}
	if dx != 0 || dy != 0 {
		for i := 1; i <= maxDistance; i++ {
			c := Coordinate{X: x + dx*i, Y: y + dy*i}
			if !free(c.X, c.Y) {
				break
			}
			add(c)
		}
	}
	for _, d := range []Coordinate{{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1}} {
		add(Coordinate{X: x + d.X, Y: y + d.Y})
	}
	return positions
}

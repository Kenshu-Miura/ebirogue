//go:build !test

package main

// expert の段9の地形への対処（wave11-A）。
//
//   - 割れた壁（botFeature "crackedwall"）: 見えている割れた壁は、壊す手数（crackedWallBreakCost）を足して経路に入れる。
//     遠回りになるなら、通常攻撃で壊して通る（move が正面の割れた壁を殴る）。店の壁の割れた壁は壊さない
//     （壊して店を出ると泥棒になる）。斜めには壊さない。
//   - 隠し通路（botFeature "hiddensearch"）: 行き先が無いとき（部屋と通路を見尽くし、敵・罠を越えても階段へ
//     行けない）だけ、通路の行き止まりと部屋の壁ぎわで正面を調べる（通常攻撃。1回で50%）。1階あたり
//     hiddenSearchMax 回まで。マップ生成は隠し通路が無くても全部屋がつながるように作るので、ふだんは使われない。
//   - 水路: 今までどおり通れないマスとして避ける（経路の無い向こう岸の道具は目標にならない）。追う敵は、まっすぐの
//     距離ではなく歩いて届く距離（expertChaseSteps 歩）で選ぶ（botFeature "enemypath"。詰めて倒しに行く遠距離攻撃の
//     敵も同じ）。水路の向こうの敵へ向かう1歩と、道具・ほかの敵へ向かう1歩が入れ替わって、同じ2マスを往復し続けたため。
//
// 突風（滞在ターン切れ）で倒れた冒険を調べて見つけた、地形以外の行き詰まりもここで扱う。
//   - ニゲハゼ系（72〜74）は逃げ回るだけで攻撃してこない倍速の敵。脅威に数えると通路へ下がる→追う→下がるを
//     繰り返したので、脅威に数えず、自分からは追わない（botFeature "runaway"。隣に来れば殴る）。
//   - 商売中の店がある階では道具寄せのカードを使わない（botFeature "gathershop"、expertBuffUsable）。

const (
	expertChaseSteps     = 12 // 追う敵はこの歩数以内に隣へ行ける敵だけ
	crackedWallBreakCost = 2  // 割れた壁を通る経路の追加の手数（壊す1回＋余裕。少しの近道のために部屋の壁を壊さない）
	hiddenSearchMax      = 40 // 1階あたりの隠し通路を探す回数の上限
	hiddenSearchPerSpot  = 2  // 1巡で同じ場所を調べる回数（1回で50%で見つかる）
)

// crackedWallUsable は (x, y) が経路に入れてよい割れた壁かどうか（見えていて、外周でも店の壁でもない）。
func (b *bot) crackedWallUsable(x, y int) bool {
	if b.persona != personaExpert || !botFeature("crackedwall") {
		return false
	}
	g := b.g
	m := g.state.Map
	if y <= 0 || y >= len(m)-1 || x <= 0 || x >= len(m[y])-1 {
		return false
	}
	tile := m[y][x]
	if tile.Type != "cracked_wall" || !tile.Visited {
		return false
	}
	if shop, ok := g.shopRoom(); ok && roomContainsWithWalls(shop, x, y) {
		return false
	}
	return true
}

// knownCrackedWall はこの階に経路に入れてよい割れた壁があるかどうか。
func (b *bot) knownCrackedWall() bool {
	if b.persona != personaExpert || !botFeature("crackedwall") {
		return false
	}
	for y, row := range b.g.state.Map {
		for x, tile := range row {
			if tile.Type == "cracked_wall" && b.crackedWallUsable(x, y) {
				return true
			}
		}
	}
	return false
}

// breaksCrackedWall は (dx, dy) の1歩が正面の割れた壁を壊す（殴る）ことになるかどうか。
func (b *bot) breaksCrackedWall(dx, dy int) bool {
	if dx != 0 && dy != 0 {
		return false
	}
	p := b.g.state.Player
	return b.crackedWallUsable(p.X+dx, p.Y+dy)
}

// pathStepWeighted は pathStepThrough と同じ探索を、割れた壁を crackedWallBreakCost 余分にかかるマスとして行う
// （手数の少ない順に広げるバケツ式の最短経路）。割れた壁へは縦横からだけ入り、割れた壁から斜めには出ない。
func (b *bot) pathStepWeighted(goal func(x, y int) bool, blocked []bool) (int, int, bool) {
	g := b.g
	height, width := len(g.state.Map), len(g.state.Map[0])
	cracked := func(x, y int) bool { return b.crackedWallUsable(x, y) }
	stepOK := func(x, y, dx, dy int) bool {
		nx, ny := x+dx, y+dy
		if dx != 0 && dy != 0 {
			return !cracked(x, y) && !cracked(nx, ny) && b.canStep(x, y, dx, dy)
		}
		if b.canStep(x, y, dx, dy) {
			return true
		}
		return nx >= 0 && ny >= 0 && nx < width && ny < height && cracked(nx, ny)
	}
	start := g.state.Player.Y*width + g.state.Player.X
	first := make([]int, width*height)
	dist := make([]int, width*height)
	first[start] = -1
	buckets := [][]int{{start}}
	bestDist, bestDir := 1<<30, 0
	for d := 0; d < len(buckets) && d+1 < bestDist; d++ {
		for i := 0; i < len(buckets[d]); i++ {
			cur := buckets[d][i]
			if dist[cur] != d {
				continue
			}
			x, y := cur%width, cur/width
			for dir, step := range botSteps {
				if !stepOK(x, y, step[0], step[1]) {
					continue
				}
				nx, ny := x+step[0], y+step[1]
				next := ny*width + nx
				cost := 1
				if cracked(nx, ny) {
					cost += crackedWallBreakCost
				}
				nd := d + cost
				dirFromStart := dir + 1
				if cur != start {
					dirFromStart = first[cur]
				}
				if goal(nx, ny) {
					if nd < bestDist {
						bestDist, bestDir = nd, dirFromStart
					}
					continue
				}
				if next == start || blocked[next] || (first[next] != 0 && dist[next] <= nd) {
					continue
				}
				first[next], dist[next] = dirFromStart, nd
				for len(buckets) <= nd {
					buckets = append(buckets, nil)
				}
				buckets[nd] = append(buckets[nd], next)
			}
		}
	}
	if bestDir == 0 {
		return 0, 0, false
	}
	s := botSteps[bestDir-1]
	return s[0], s[1], true
}

// --- 隠し通路を探す ---

type hiddenSpot struct{ floor, x, y, dx, dy int }

// hiddenSearchFacing は (x, y) に立って調べる向き（隠し通路かもしれない壁）を返す。
// 通路の行き止まり（通れる隣が1つだけ）では開いた側の反対、部屋の壁ぎわでは壁の側。
func (b *bot) hiddenSearchFacing(x, y int) (int, int, bool) {
	g := b.g
	m := g.state.Map
	if m[y][x].Blocked || m[y][x].Type == "stairs" || g.enemyIndexAt(x, y) >= 0 {
		return 0, 0, false
	}
	open := 0
	for _, s := range botSteps {
		if b.canStep(x, y, s[0], s[1]) {
			open++
		}
	}
	for _, d := range [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}} {
		wx, wy := x+d[0], y+d[1]
		if wy <= 0 || wy >= len(m)-1 || wx <= 0 || wx >= len(m[wy])-1 {
			continue
		}
		wall := m[wy][wx]
		if !wall.Blocked || wall.Type == "water" || wall.Type == "cracked_wall" {
			continue
		}
		if b.searched[hiddenSpot{g.Floor, x, y, d[0], d[1]}] >= hiddenSearchPerSpot*(1+b.searchRound) {
			continue
		}
		deadEnd := m[y][x].Type == "corridor" && open <= 1
		wallSide := g.roomIDAt(x, y) >= 0 && wall.Visited
		if deadEnd || wallSide {
			return d[0], d[1], true
		}
	}
	return 0, 0, false
}

// expertSearchHidden は行き先が無いとき、隠し通路を探しに怪しい場所へ向かい、正面を調べる。行動したら true。
func (b *bot) expertSearchHidden() (bool, error) {
	g := b.g
	if b.persona != personaExpert || !botFeature("hiddensearch") || b.bossAlive() {
		return false, nil
	}
	if b.searchFloor != g.Floor {
		b.searchFloor, b.searchCount, b.searchRound = g.Floor, 0, 0
	}
	if b.searchCount >= hiddenSearchMax {
		return false, nil
	}
	p := g.state.Player
	if dx, dy, ok := b.hiddenSearchFacing(p.X, p.Y); ok {
		if b.searched == nil {
			b.searched = map[hiddenSpot]int{}
		}
		b.searched[hiddenSpot{g.Floor, p.X, p.Y, dx, dy}]++
		b.searchCount++
		b.act = "searchHidden"
		return true, b.attack(dx, dy)
	}
	spot := func(x, y int) bool { _, _, ok := b.hiddenSearchFacing(x, y); return ok }
	for round := 0; round < 2; round++ {
		if dx, dy, ok := b.pathStep(spot); ok {
			b.act = "toSearch"
			return true, b.move(dx, dy)
		}
		if b.searchCount == 0 {
			break // 調べる場所が無い
		}
		b.searchRound++ // どこも調べ終えたら、もう一巡する
	}
	return false, nil
}

// --- 追う敵の選び方 ---

// isRunawayEnemy は逃げ回るだけで攻撃してこないニゲハゼ系（72〜74）に見える敵かどうか。
func isRunawayEnemy(e Enemy) bool {
	id := apparentEnemyID(e)
	return id >= 72 && id <= 74
}

// worthChasing は expert が自分から近づいて倒しに行く敵かどうか。
func (b *bot) worthChasing(e Enemy) bool {
	if botFeature("runaway") && isRunawayEnemy(e) {
		return false
	}
	if b.houseAvoidsEnemy(e) { // 敵の多く残ったモンスターハウスの部屋へは戻らない（bot_house_test.go）
		return false
	}
	if !botFeature("enemypath") || b.bossAlive() {
		return true // ボス戦は今までどおり
	}
	next := func(x, y int) bool { return max(abs(x-e.X), abs(y-e.Y)) <= 1 }
	p := b.g.state.Player
	if next(p.X, p.Y) {
		return true
	}
	_, _, _, ok := b.pathStepWithin(next, expertChaseSteps)
	return ok
}

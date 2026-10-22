//go:build !test

package main

// expert の探索の切り上げと、降りる前の休憩（wave11-C）。
//   - leavefloor: 階段を見つけていて、見えている拾う価値のある道具が無く、階の滞在ターンか
//     満腹度（手持ちの食料を含む）が目安を超えたら降りる。突風の前（botGustRushTurns）には敵・罠を
//     押しのけてでも階段へ向かう。
//   - restbeforestairs: 降りる前に、敵がいず満腹度に余裕があれば階段の上で足踏みして HP を戻す。

const (
	botLeaveFloorTurns   = 500  // 階段を見つけていて、この階にこれより長く居たら降りる
	botLeaveFloorSatiety = 100  // 階段を見つけていて、満腹度と手持ちの食料の合計がこれ未満なら降りる
	botGustRushTurns     = 1000 // この階にこれより長く居たら、敵・罠を押しのけてでも階段へ向かう（風は1200〜）
	botStairsRestMax     = 80   // 階段の上で休む最大のターン数（1階あたり）
	botStairsRestSatiety = 30   // 満腹度がこれ以下なら階段の上で休まない
)

// stairsKnown は階段のマスを見つけているか（見えたマスは Visited になる）。
func (b *bot) stairsKnown() bool {
	for _, row := range b.g.state.Map {
		for _, tile := range row {
			if tile.Type == "stairs" && tile.Visited {
				return true
			}
		}
	}
	return false
}

// expertCutExploration は探索を切り上げて降りるか。food は手持ちと壺の食料の満腹度の見積もり。
func (b *bot) expertCutExploration(food int) bool {
	g := b.g
	if !b.stairsKnown() {
		return false
	}
	if _, _, ok := b.pathStep(b.expertItemGoalFunc()); ok {
		return false // 見えている拾う価値のある道具を先に拾う
	}
	return g.floorTurns > botLeaveFloorTurns || g.state.Player.Satiety+food < botLeaveFloorSatiety
}

// expertRushStairs は突風の前に、ふつうの経路で階段へ行けなければ敵・罠を押しのけて向かう。動いたら true。
func (b *bot) expertRushStairs() (bool, error) {
	g := b.g
	if !botFeature("leavefloor") || g.floorTurns <= botGustRushTurns || b.bossAlive() || b.unpaidTotal() > 0 {
		return false, nil
	}
	if _, _, ok := b.expertPathStep(b.stairsGoal); ok {
		return false, nil // ふつうの経路で向かう（目的地の先頭は階段）
	}
	dx, dy, ok := b.pathStepThrough(b.stairsGoal, true)
	if !ok {
		return false, nil
	}
	b.act = "rushStairs"
	p := g.state.Player
	if isOccupied(g, p.X+dx, p.Y+dy) {
		return true, b.attack(dx, dy)
	}
	return true, b.move(dx, dy)
}

// expertRestBeforeStairs は降りる前に階段の上で休むか（休むなら回数を数える）。
func (b *bot) expertRestBeforeStairs() bool {
	g := b.g
	if !botFeature("restbeforestairs") {
		return false
	}
	if b.restFloor != g.Floor {
		b.restFloor, b.stairsRest = g.Floor, 0
	}
	p := &g.state.Player
	if p.Health >= p.MaxHealth || b.stairsRest >= botStairsRestMax || p.StatusAilments.NoHealing > 0 ||
		p.Satiety <= botStairsRestSatiety || g.floorTurns >= botGustRushTurns || b.bossAlive() || b.inRangedFire(p.X, p.Y) {
		return false
	}
	if _, ok := b.nearestEnemy(expertRestEnemyRange); ok {
		return false
	}
	b.stairsRest++
	return true
}

// resetShopFloor は階が変わったら店の記録を消す。
func (b *bot) resetShopFloor() {
	if b.shopFloor != b.g.Floor {
		b.shopFloor, b.shopRejected, b.buyTarget = b.g.Floor, nil, nil
	}
}

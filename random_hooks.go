//go:build !test

package main

// randHooks はテストで特定の乱数判定だけを固定するための差し替え口。
// nil の項目は冒険の乱数系列（g.rng）を使う。
type randHooks struct {
	damage           func(int) int // ダメージの乱数係数・会心判定
	attackHit        func(int) int // プレイヤー攻撃の命中判定
	shieldDefense    func(int) int // 盾の回避・反撃判定
	statusResistance func(int) int // 盾の状態異常耐性判定
}

func (g *Game) hookOrRandIntn(hook func(int) int) func(int) int {
	if hook != nil {
		return hook
	}
	return g.randIntn
}

func (g *Game) damageIntn() func(int) int { return g.hookOrRandIntn(g.randHooks.damage) }

func (g *Game) attackHitIntn() func(int) int { return g.hookOrRandIntn(g.randHooks.attackHit) }

func (g *Game) shieldDefenseIntn() func(int) int { return g.hookOrRandIntn(g.randHooks.shieldDefense) }

func (g *Game) statusResistanceIntn() func(int) int {
	return g.hookOrRandIntn(g.randHooks.statusResistance)
}

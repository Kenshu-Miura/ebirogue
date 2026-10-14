//go:build !test

package main

// randHooks はテストで特定の乱数判定だけを固定するための差し替え口。
// nil の項目は冒険の乱数系列（g.rng）を使う。
type randHooks struct {
	damage           func(int) int // ダメージの乱数係数・会心判定
	attackHit        func(int) int // プレイヤー攻撃の命中判定
	shieldDefense    func(int) int // 盾の回避・反撃判定
	statusResistance func(int) int // 盾の状態異常耐性判定
	ability          func(int) int // 装備能力のフック内の判定（g.abilityIntn）
	enemyCritical    func(int) int // 敵の痛恨の一撃の判定
	enemyEvasion     func(int) int // 敵が通常攻撃をかわす判定
	displacement     func(int) int // 持ち物を落とす処理の対象・位置の抽選
	projectile       func(int) int // 投げた道具・矢が敵に当たるかの判定（命中率が100%未満のときだけ）
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

// abilityIntn は装備能力のフックの中で使う乱数源。
func (g *Game) abilityIntn() func(int) int { return g.hookOrRandIntn(g.randHooks.ability) }

func (g *Game) enemyCriticalIntn() func(int) int { return g.hookOrRandIntn(g.randHooks.enemyCritical) }

func (g *Game) enemyEvasionIntn() func(int) int { return g.hookOrRandIntn(g.randHooks.enemyEvasion) }

func (g *Game) displacementIntn() func(int) int { return g.hookOrRandIntn(g.randHooks.displacement) }

func (g *Game) projectileIntn() func(int) int { return g.hookOrRandIntn(g.randHooks.projectile) }

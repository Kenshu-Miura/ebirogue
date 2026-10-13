//go:build !test

package main

import "fmt"

func physicalTrapEffect(id int, name string, power int) func(*Game) {
	return func(g *Game) {
		damage := rollEnemyAttackDamage(power, g.state.Player.DefensePower, g.damageIntn())
		defense := g.resolvePlayerShieldDefense(damage, EnemyDamageNormal, EnemyAttackEnvironment)
		g.Enqueue(Action{Duration: 0.5, Message: fmt.Sprintf("%sで%dダメージを受けた", name, defense.Damage), Execute: func(g *Game) {
			g.applyPlayerShieldDefense(defense, trapSource(id, name), -1, -1, -1)
		}})
	}
}

var woodArrowTrapEffect = physicalTrapEffect(8, "木の矢の罠", woodArrowTrapPower)
var ironArrowTrapEffect = physicalTrapEffect(9, "鉄の矢の罠", ironArrowTrapPower)
var fallingRockTrapEffect = physicalTrapEffect(10, "落石の罠", fallingRockTrapPower)

var largeMineTrapEffect = func(g *Game) {
	x, y := g.state.Player.GetPosition()
	defense := g.resolvePlayerShieldDefense(largeMineTrapDamage(g.state.Player.Health), EnemyDamageExplosion, EnemyAttackEnvironment)
	g.Enqueue(Action{Duration: 0.5, Message: fmt.Sprintf("大型地雷が爆発し、%dダメージを受けた", defense.Damage), Execute: func(g *Game) {
		// 後ろから処理し、撃破によるスライス削除で対象を飛ばさない。
		for i := len(g.state.Enemies) - 1; i >= 0; i-- {
			enemy := g.state.Enemies[i]
			if abs(enemy.X-x) <= 1 && abs(enemy.Y-y) <= 1 && (enemy.X != x || enemy.Y != y) {
				damage := rollEnemyAttackDamage(largeMineEnemyPower, 0, g.damageIntn())
				g.applyDamageToEnemy(i, damage)
			}
		}
		g.applyPlayerShieldDefense(defense, trapSource(11, "大型地雷"), -1, -1, -1)
	}})
}

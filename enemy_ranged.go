//go:build !test

package main

import (
	"fmt"
	"math/rand"
)

const rangedAttackEffectDuration = 0.45

type RangedAttackEffect struct {
	Kind             RangedAttackKind
	OriginX, OriginY int
	TargetX, TargetY int
	BlastRadius      int
	Timer            float64
}

func (g *Game) canEnemyUseRangedAttack(enemyIndex int) bool {
	enemy := &g.state.Enemies[enemyIndex]
	attack := enemy.RangedAttack
	if attack.Kind == RangedAttackNone || enemy.StatusAilments.Seal || !enemy.PlayerDiscovered {
		return false
	}

	playerX, playerY := g.state.Player.X, g.state.Player.Y
	if !withinRangedDistance(enemy.X, enemy.Y, playerX, playerY, attack.MinRange, attack.MaxRange) {
		return false
	}

	switch attack.Kind {
	case RangedAttackArrow, RangedAttackExplosion:
		return hasClearStraightLine(g.state.Map, g.state.Enemies, enemyIndex, enemy.X, enemy.Y, playerX, playerY, attack.MaxRange)
	case RangedAttackRock:
		// 石は山なりに投げるため、壁・扉・ほかの敵を越えられる。
		return true
	}
	return false
}

// tryEnemyRangedAttack は遠距離攻撃が可能ならアクションを積み、行動済みとして true を返す。
func (g *Game) tryEnemyRangedAttack(enemyIndex int) bool {
	if !g.canEnemyUseRangedAttack(enemyIndex) {
		return false
	}

	enemy := &g.state.Enemies[enemyIndex]
	attack := enemy.RangedAttack
	targetX, targetY := g.state.Player.X, g.state.Player.Y
	dx, dy := sign(targetX-enemy.X), sign(targetY-enemy.Y)
	damage := rollEnemyRangedDamage(attack.AttackPower, g.state.Player.DefensePower, rand.Intn)

	var message string
	switch attack.Kind {
	case RangedAttackArrow:
		message = fmt.Sprintf("%sが矢を放った。海老さんは%dダメージを受けた", g.enemyDisplayName(enemy.Name), damage)
	case RangedAttackRock:
		message = fmt.Sprintf("%sが障害物越しに石を投げた。海老さんは%dダメージを受けた", g.enemyDisplayName(enemy.Name), damage)
	case RangedAttackExplosion:
		message = fmt.Sprintf("%sの爆発弾。周囲が爆風に包まれ、海老さんは%dダメージを受けた", g.enemyDisplayName(enemy.Name), damage)
	}

	originX, originY := enemy.X, enemy.Y
	g.Enqueue(Action{
		Duration: rangedAttackEffectDuration,
		Message:  message,
		Execute: func(g *Game) {
			enemy.AttackTimer = 0.5
			enemy.AttackDirection = determineDirection(dx, dy)
			enemy.Direction = enemy.AttackDirection
			g.rangedAttackEffect = RangedAttackEffect{
				Kind:        attack.Kind,
				OriginX:     originX,
				OriginY:     originY,
				TargetX:     targetX,
				TargetY:     targetY,
				BlastRadius: attack.BlastRadius,
				Timer:       rangedAttackEffectDuration,
			}

			// 爆発弾は着弾点の周囲1マスを攻撃範囲とする。
			// 現在の敵AIは海老さんのいるマスを狙うため、移動していなければ必ず巻き込まれる。
			if attack.Kind != RangedAttackExplosion || withinBlastRadius(targetX, targetY, g.state.Player.X, g.state.Player.Y, attack.BlastRadius) {
				g.state.Player.Health -= damage
				if g.state.Player.Health < 0 {
					g.state.Player.Health = 0
				}
				g.state.Player.checkDeath(g)
			}
			if attack.Kind == RangedAttackExplosion {
				g.enqueueExplosionCollateral(enemy.ID, originX, originY, targetX, targetY, attack)
			}
		},
	})
	return true
}

// enqueueExplosionCollateral は同じ敵ターンで予約済みの行動が終わった後に、
// 着弾点周囲のモンスターへ巻き込みダメージを適用する。
func (g *Game) enqueueExplosionCollateral(attackerID, attackerX, attackerY, targetX, targetY int, attack RangedAttackDefinition) {
	hasTarget := false
	for _, enemy := range g.state.Enemies {
		if enemy.ID == attackerID && enemy.X == attackerX && enemy.Y == attackerY {
			continue
		}
		if withinBlastRadius(targetX, targetY, enemy.X, enemy.Y, attack.BlastRadius) {
			hasTarget = true
			break
		}
	}
	if !hasTarget {
		return
	}

	g.Enqueue(Action{
		Duration: 0.3,
		Message:  "爆風が周囲のモンスターを巻き込んだ",
		Execute: func(g *Game) {
			for i := len(g.state.Enemies) - 1; i >= 0; i-- {
				target := &g.state.Enemies[i]
				if target.ID == attackerID && target.X == attackerX && target.Y == attackerY {
					continue
				}
				if !withinBlastRadius(targetX, targetY, target.X, target.Y, attack.BlastRadius) {
					continue
				}
				target.Health -= rollEnemyRangedDamage(attack.AttackPower, target.DefensePower, rand.Intn)
				target.StatusAilments.Sleep = 0
				target.StatusAilments.Paralysis = false
				if target.Health <= 0 {
					g.state.Enemies = append(g.state.Enemies[:i], g.state.Enemies[i+1:]...)
				}
			}
			g.miniMapDirty = true
		},
	})
}

func (g *Game) UpdateRangedAttackEffect() {
	if g.rangedAttackEffect.Timer <= 0 {
		return
	}
	g.rangedAttackEffect.Timer -= 1.0 / 60.0
	if g.rangedAttackEffect.Timer <= 0 {
		g.rangedAttackEffect = RangedAttackEffect{}
	}
}

//go:build !test

package main

import (
	"fmt"
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

	// 聖域のカードの上にいるプレイヤーは遠距離攻撃も受けない
	if g.playerOnSanctuary() {
		return false
	}

	playerX, playerY := g.state.Player.X, g.state.Player.Y
	if !withinRangedDistance(enemy.X, enemy.Y, playerX, playerY, attack.MinRange, attack.MaxRange) {
		return false
	}

	switch attack.Kind {
	case RangedAttackArrow, RangedAttackExplosion, RangedAttackFire, RangedAttackMagic:
		return hasClearStraightLine(g.state.Map, g.state.Enemies, enemyIndex, enemy.X, enemy.Y, playerX, playerY, attack.MaxRange)
	case RangedAttackRock:
		// 石は山なりに投げるため、壁・扉・ほかの敵を越えられる。
		return true
	}
	return false
}

// rangedAttackName は死因の表示に使う遠距離攻撃の名前を返す。
func rangedAttackName(kind RangedAttackKind) string {
	switch kind {
	case RangedAttackArrow:
		return "矢"
	case RangedAttackRock:
		return "投石"
	case RangedAttackExplosion:
		return "爆発弾"
	case RangedAttackFire:
		return "炎"
	case RangedAttackMagic:
		return "魔法弾"
	default:
		return ""
	}
}

func enemyDamageKindForRangedAttack(kind RangedAttackKind) EnemyDamageKind {
	switch kind {
	case RangedAttackExplosion:
		return EnemyDamageExplosion
	case RangedAttackFire:
		return EnemyDamageFire
	case RangedAttackMagic:
		return EnemyDamageMagic
	default:
		return EnemyDamageNormal
	}
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
	damage := rollEnemyAttackDamage(attack.AttackPower, g.state.Player.DefensePower, g.damageIntn())
	defense := g.resolvePlayerDefense(damage, enemyDamageKindForRangedAttack(attack.Kind), EnemyAttackRanged, enemy, false)
	enemyName := g.enemyDisplayName(enemy.Name)

	var attackMessage string
	switch attack.Kind {
	case RangedAttackArrow:
		attackMessage = fmt.Sprintf("%sが矢を放った。", enemyName)
	case RangedAttackRock:
		attackMessage = fmt.Sprintf("%sが障害物越しに石を投げた。", enemyName)
	case RangedAttackExplosion:
		attackMessage = fmt.Sprintf("%sの爆発弾が炸裂した。", enemyName)
	case RangedAttackFire:
		attackMessage = fmt.Sprintf("%sが灼熱の炎を吐いた。", enemyName)
	case RangedAttackMagic:
		attackMessage = fmt.Sprintf("%sが魔法弾を放った。", enemyName)
	}
	message := attackMessage + g.shieldDefenseMessage(enemyName, defense)

	originX, originY, attackerID := enemy.X, enemy.Y, enemy.ID
	source := enemyRangedSource(enemy.ID, enemy.Name, rangedAttackName(attack.Kind))
	g.Enqueue(Action{
		Duration: rangedAttackEffectDuration,
		Message:  message,
		Execute: func(g *Game) {
			if MonsterDefinitions[attackerID].IsBoss && g.livingBossIndex(attackerID) < 0 {
				return
			}
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
				g.applyPlayerShieldDefense(defense, source, attackerID, originX, originY)
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
			defeatedCount := 0
			for i := len(g.state.Enemies) - 1; i >= 0; i-- {
				target := &g.state.Enemies[i]
				if target.ID == attackerID && target.X == attackerX && target.Y == attackerY {
					continue
				}
				if !withinBlastRadius(targetX, targetY, target.X, target.Y, attack.BlastRadius) {
					continue
				}
				target.Health -= rollEnemyAttackDamage(attack.AttackPower, target.DefensePower, g.damageIntn())
				clearStatus(&target.StatusAilments, StatusSleep)
				clearStatus(&target.StatusAilments, StatusParalysis)
				if target.Health <= 0 {
					defeated := *target
					g.dropEnemyHeldItem(i)
					g.state.Enemies = append(g.state.Enemies[:i], g.state.Enemies[i+1:]...)
					defeatedCount++
					g.onEnemyDefeated(defeated)
				}
			}
			if attackerIndex := g.enemyIndexAt(attackerX, attackerY); attackerIndex >= 0 {
				for range defeatedCount {
					if !g.levelUpEnemy(attackerIndex) {
						break
					}
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

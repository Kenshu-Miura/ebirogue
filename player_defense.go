//go:build !test

package main

import (
	"fmt"
	"math/rand"
)

// 盾能力の乱数源。テストでは差し替えて決定的にする。
var shieldDefenseRandInt = rand.Intn
var statusResistanceRandInt = rand.Intn

func equippedArmorAbilities(player *Player) []EquipmentAbilityID {
	if player.EquippedArmor == nil {
		return nil
	}
	return player.EquippedArmor.Abilities
}

func (g *Game) resolvePlayerShieldDefense(damage int, kind EnemyDamageKind, delivery EnemyAttackDelivery) ShieldDefenseResult {
	return resolveShieldDefense(damage, kind, delivery, equippedArmorAbilities(&g.state.Player), shieldDefenseRandInt)
}

func (g *Game) playerResistsStatus() bool {
	return rollStatusResistance(equippedArmorAbilities(&g.state.Player), statusResistanceRandInt)
}

func (g *Game) playerBlocksTheft() bool {
	return blocksTheft(equippedArmorAbilities(&g.state.Player))
}

func (g *Game) shieldDefenseMessage(enemyName string, result ShieldDefenseResult) string {
	armorName := "盾"
	if g.state.Player.EquippedArmor != nil {
		armorName = g.state.Player.EquippedArmor.Name
	}
	switch {
	case result.Evaded:
		return fmt.Sprintf("%sで%sの攻撃をかわした", armorName, enemyName)
	case result.Reflected:
		return fmt.Sprintf("%sが攻撃を反射し、%sに%dダメージを返した", armorName, enemyName, result.CounterDamage)
	case result.CounterDamage > 0:
		return fmt.Sprintf("%sから%dダメージを受け、%sに%dダメージを返した", enemyName, result.Damage, enemyName, result.CounterDamage)
	case result.Resisted:
		return fmt.Sprintf("%sが威力を抑え、%sから%dダメージを受けた", armorName, enemyName, result.Damage)
	default:
		return fmt.Sprintf("%sから%dダメージを受けた", enemyName, result.Damage)
	}
}

func (g *Game) findEnemyForDefense(id, x, y int) int {
	for i := range g.state.Enemies {
		enemy := &g.state.Enemies[i]
		if enemy.ID == id && enemy.X == x && enemy.Y == y {
			return i
		}
	}
	return -1
}

// applyPlayerShieldDefense は遅延Actionの実行時にダメージ、反射・カウンター、盾劣化を適用する。
func (g *Game) applyPlayerShieldDefense(result ShieldDefenseResult, attackerID, attackerX, attackerY int) {
	if result.Damage > 0 {
		g.state.Player.Health = max(0, g.state.Player.Health-result.Damage)
	}

	if result.WearArmor {
		armor := g.state.Player.EquippedArmor
		if armor != nil {
			if next, changed := nextDisposableArmorSharpness(armor.Sharpness, armor.Abilities); changed {
				armor.Sharpness = next
				g.state.Player.DefensePower--
				g.EnqueueMessage(fmt.Sprintf("%sは被弾で弱くなった", armor.Name), 0.4)
			}
		}
	}

	if result.CounterDamage > 0 {
		if index := g.findEnemyForDefense(attackerID, attackerX, attackerY); index >= 0 {
			g.applyDamageToEnemy(index, result.CounterDamage)
		}
	}
	g.state.Player.checkDeath(g)
}

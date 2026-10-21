//go:build !test

package main

import (
	"fmt"
)

func equippedArmorAbilities(player *Player) []EquipmentAbilityID {
	if player.EquippedArmor == nil {
		return nil
	}
	return player.EquippedArmor.Abilities
}

// resolvePlayerShieldDefense は攻撃者を渡さない被ダメージ（罠・遠距離攻撃など）を解決する。
func (g *Game) resolvePlayerShieldDefense(damage int, kind EnemyDamageKind, delivery EnemyAttackDelivery) ShieldDefenseResult {
	return g.resolvePlayerDefense(damage, kind, delivery, nil, false)
}

// resolvePlayerDefense は盾の能力（回避→反射→属性軽減→カウンター）を解決した後、
// 残ったダメージへ装備能力のフック ModifyIncomingDamage を適用する。
// attacker は攻撃した敵（無ければ nil）、critical は敵の痛恨の一撃かどうか。
// カウンターのダメージはフックで軽減する前のダメージから決まる。
func (g *Game) resolvePlayerDefense(damage int, kind EnemyDamageKind, delivery EnemyAttackDelivery, attacker *Enemy, critical bool) ShieldDefenseResult {
	result := resolveShieldDefense(damage, kind, delivery, equippedArmorAbilities(&g.state.Player), g.shieldDefenseIntn())
	if result.Damage <= 0 {
		return result
	}
	incoming := IncomingDamage{Damage: result.Damage, Kind: kind, Delivery: delivery, Attacker: attacker, Critical: critical}
	g.runModifyIncomingDamageHooks(&incoming)
	if incoming.Evaded {
		return ShieldDefenseResult{Evaded: true, Guard: incoming.EvadedBy}
	}
	result.Damage = incoming.Damage
	if len(incoming.Effects) > 0 {
		result.Effects = &incoming.Effects
	}
	if result.Damage == 0 {
		result.WearArmor = false
	}
	return result
}

func (g *Game) playerResistsStatus() bool {
	return rollStatusResistance(equippedArmorAbilities(&g.state.Player), g.statusResistanceIntn())
}

func (g *Game) playerBlocksTheft() bool {
	return blocksTheft(equippedArmorAbilities(&g.state.Player))
}

func (g *Game) shieldDefenseMessage(enemyName string, result ShieldDefenseResult) string {
	armorName := "盾"
	if g.state.Player.EquippedArmor != nil {
		armorName = g.state.Player.EquippedArmor.Name
	}
	if result.Guard != "" {
		armorName = result.Guard
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
// source は死因の記録に使うダメージの原因。
func (g *Game) applyPlayerShieldDefense(result ShieldDefenseResult, source DamageSource, attackerID, attackerX, attackerY int) {
	if !result.Evaded && !result.Reflected && (source.Kind == DamageSourceEnemyAttack || source.Kind == DamageSourceEnemyRanged) && statusActive(g.state.Player.StatusAilments, StatusNigiri) {
		g.state.Player.Satiety = max(0, g.state.Player.Satiety-5)
	}
	if result.Effects != nil {
		for _, effect := range *result.Effects {
			effect(g)
		}
	}
	if result.Damage > 0 {
		g.damagePlayer(result.Damage, source)
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

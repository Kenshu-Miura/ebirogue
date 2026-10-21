//go:build !test

package main

import "fmt"

const (
	AbilityGiantFan      EquipmentAbilityID = "giant_fan"
	AbilityCriticalGuard EquipmentAbilityID = "critical_guard"
	AbilityTossWard      EquipmentAbilityID = "toss_ward"
	AbilityLongThrow     EquipmentAbilityID = "long_throw"
	AbilityPoorThrow     EquipmentAbilityID = "poor_throw"
	AbilitySureThrow     EquipmentAbilityID = "sure_throw"
	AbilityBounceThrow   EquipmentAbilityID = "bounce_throw"
	AbilityRapidShot     EquipmentAbilityID = "rapid_shot"
	AbilityKnockOffWard  EquipmentAbilityID = "knock_off_ward"
	AbilityCriticalCurse EquipmentAbilityID = "critical_curse"
	AbilityDoubleEdge    EquipmentAbilityID = "double_edge"
)

func init() {
	statusImmunities[AbilityTossWard] = []StatusKind{StatusItemToss}
	statusImmunities[AbilityKnockOffWard] = []StatusKind{StatusKnockOff}
	registerAbilityHooks(AbilityGiantFan, AbilityHooks{AfterPlayerHit: func(g *Game, hit PlayerAttackHit, _ int) {
		if hit.Defeated {
			return
		}
		index := g.findEnemyForDefense(hit.EnemyID, hit.Enemy.X, hit.Enemy.Y)
		if index < 0 || g.abilityIntn()(100) >= 25 {
			return
		}
		dx := sign(hit.Enemy.X - g.state.Player.X)
		dy := sign(hit.Enemy.Y - g.state.Player.Y)
		g.knockbackEnemy(index, dx, dy, 10, 5)
	}})
	registerAbilityHooks(AbilityCriticalGuard, AbilityHooks{ModifyIncomingDamage: func(_ *Game, incoming *IncomingDamage, _ int) {
		if incoming.Critical {
			incoming.Damage = max(1, incoming.Damage/3)
		}
	}})
	registerAbilityHooks(AbilityLongThrow, AbilityHooks{ModifyProjectile: func(_ *Game, p *Projectile, _ int) {
		if stage3PhysicalProjectile(p) {
			p.PierceWalls, p.PierceTargets = true, true
		}
	}})
	registerAbilityHooks(AbilityPoorThrow, AbilityHooks{ModifyProjectile: func(_ *Game, p *Projectile, _ int) {
		if stage3PhysicalProjectile(p) {
			p.HitPercent = 0
		}
	}})
	registerAbilityHooks(AbilitySureThrow, AbilityHooks{ModifyProjectile: func(_ *Game, p *Projectile, _ int) {
		if stage3PhysicalProjectile(p) {
			p.SureHit = true
		}
	}})
	registerAbilityHooks(AbilityBounceThrow, AbilityHooks{ModifyProjectile: func(_ *Game, p *Projectile, stacks int) {
		if stage3PhysicalProjectile(p) {
			p.Bounces += 3 * stacks
		}
	}})
	registerAbilityHooks(AbilityRapidShot, AbilityHooks{ModifyProjectile: func(_ *Game, p *Projectile, stacks int) {
		if p.Kind == ProjectileShot {
			p.Volley = 1 + stacks
		}
	}})
	criticalCurse := func(_ *Game, attack *EnemyAttack, stacks int) { attack.CriticalPercent += 10 * stacks }
	registerAbilityHooks(AbilityCriticalCurse, AbilityHooks{BeforeEnemyAttack: criticalCurse})
	registerAbilityHooks(AbilityDoubleEdge, AbilityHooks{
		BeforeEnemyAttack: criticalCurse,
		BeforePlayerAttack: func(_ *Game, attack *PlayerAttack, stacks int) {
			attack.CriticalDenominator = min(attack.CriticalDenominator, 8/stacks)
		},
	})
	expansionHelpPages = append(expansionHelpPages,
		HelpPage{Title: "指輪：飛び道具・痛恨", Lines: []string{
			"遠投の指輪：投げた物・矢が壁と敵を突き抜ける",
			"ヘタ投げの指輪：投げた物・矢が必ず外れる。呪い付き",
			"百発百中の指輪：投げた物・矢が必ず当たる",
			"反射の指輪：投げた物・矢が壁で3回反射。2つで6回",
			"連射の指輪：一度に矢を2本撃つ。2つで3本",
			"弾きよけの指輪：装備を弾き飛ばされない",
			"痛恨の指輪：敵の痛恨率+10%。2つで+20%。呪い付き",
			"諸刃の指輪：会心1/8、敵の痛恨率+10%",
			"諸刃を2つ：会心1/4、敵の痛恨率+20%",
			"不動の指輪：吹き飛ばし・引き寄せ・場所替えを防ぐ",
		}},
		HelpPage{Title: "道具：追加の盾・武器・カード", Lines: []string{
			"巨大ハリセン：命中後25%で敵を10マス吹き飛ばす",
			"巨大ハリセン：壁や敵にぶつかると5ダメージ",
			"痛恨よけの盾：痛恨のダメージを1/3にする（最低1）",
			"いたずらよけの盾：持ち物を投げ捨てられない",
			"不動の盾：吹き飛ばし・引き寄せ・場所替えを防ぐ",
			"ざぶとん：吹き飛ばされたときの衝突ダメージを防ぐ",
			"ゾワゾワのカード：周囲の敵が20ターン反対向きに攻撃",
		}},
	)
}

func stage3PhysicalProjectile(p *Projectile) bool {
	return p.Kind == ProjectileThrow || p.Kind == ProjectileShot
}

// ゾワゾワは周囲1マスの8方向に適用する。ボスの短縮は共通処理に任せる。
func backwardCard(g *Game) {
	item, inventory := determineItemSource(g)
	g.Enqueue(Action{Duration: 0.4, Message: fmt.Sprintf("%sを使用した", item.GetName()), Execute: func(g *Game) {
		count := 0
		for i := range g.state.Enemies {
			enemy := &g.state.Enemies[i]
			dx, dy := enemy.X-g.state.Player.X, enemy.Y-g.state.Player.Y
			if abs(dx) <= 1 && abs(dy) <= 1 && (dx != 0 || dy != 0) {
				g.applyEnemyStatus(i, StatusBackward, 20)
				count++
				g.EnqueueMessage(fmt.Sprintf("%sはゾワゾワした！", g.enemyDisplayName(enemy.Name)), 0.4)
			}
		}
		if count == 0 {
			g.EnqueueMessage("しかし何も起こらなかった", 0.4)
		}
	}})
	removeUsedItem(g, inventory)
}

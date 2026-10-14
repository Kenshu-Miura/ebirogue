//go:build !test

package main

// 盾の能力 ID とフック。共通の定数・登録表の初期値は変更しない。
const (
	AbilityDragonGuard  EquipmentAbilityID = "dragon_guard"
	AbilityNigiriGuard  EquipmentAbilityID = "nigiri_guard"
	AbilityPuppetGuard  EquipmentAbilityID = "puppet_guard"
	AbilityMoneyGuard   EquipmentAbilityID = "money_guard"
	AbilitySatietyGuard EquipmentAbilityID = "satiety_guard"
	AbilityHungerGuard  EquipmentAbilityID = "hunger_guard"
	AbilityGuts         EquipmentAbilityID = "guts"
	AbilityPeakGuard    EquipmentAbilityID = "peak_guard"
	AbilitySatori       EquipmentAbilityID = "satori"
	AbilityLuckyGuard   EquipmentAbilityID = "lucky_guard"
	AbilityCurseGuard   EquipmentAbilityID = "curse_guard"
	AbilityShellDodge   EquipmentAbilityID = "shell_dodge"
)

func (g *Game) capSatietyForSatori() {
	g.state.Player.Satiety = satoriSatietyCap(g.state.Player.Satiety, g.playerHasAbility(AbilitySatori))
}

func init() {
	statusImmunities[AbilityNigiriGuard] = []StatusKind{StatusFoodTransform}
	statusImmunities[AbilityPuppetGuard] = []StatusKind{StatusManipulation}
	registerAbilityHooks(AbilityDragonGuard, AbilityHooks{ModifyIncomingDamage: func(g *Game, incoming *IncomingDamage, stacks int) {
		if incoming.Kind == EnemyDamageFire || (incoming.Attacker != nil && hasEnemyTrait(incoming.Attacker.Traits, EnemyTraitDragon)) {
			incoming.Damage = max(1, incoming.Damage/2)
		}
	}})
	registerAbilityHooks(AbilityMoneyGuard, AbilityHooks{ModifyIncomingDamage: func(g *Game, incoming *IncomingDamage, stacks int) {
		if incoming.Delivery != EnemyAttackMelee || g.state.Player.Cash < 20 {
			return
		}
		reduced := max(1, incoming.Damage*2/3)
		if reduced >= incoming.Damage {
			return
		}
		incoming.Damage = reduced
		incoming.Effects = append(incoming.Effects, func(g *Game) { g.state.Player.Cash = max(0, g.state.Player.Cash-20) })
	}})
	registerAbilityHooks(AbilitySatietyGuard, AbilityHooks{ModifyIncomingDamage: func(g *Game, incoming *IncomingDamage, stacks int) {
		if incoming.Delivery != EnemyAttackMelee || g.state.Player.Satiety < 10 {
			return
		}
		reduced := max(1, incoming.Damage*2/3)
		if reduced >= incoming.Damage {
			return
		}
		incoming.Damage = reduced
		incoming.Effects = append(incoming.Effects, func(g *Game) { g.state.Player.Satiety = max(0, g.state.Player.Satiety-2) })
	}})
	registerAbilityHooks(AbilityHungerGuard, AbilityHooks{ModifyIncomingDamage: func(g *Game, incoming *IncomingDamage, stacks int) {
		if g.state.Player.Satiety == 0 {
			incoming.Damage = max(1, incoming.Damage/2)
		}
	}})
	registerAbilityHooks(AbilityGuts, AbilityHooks{PreventDeath: func(g *Game, damage int, source DamageSource, stacks int) bool {
		if source.Kind == DamageSourceStarvation || g.abilityIntn()(3) != 0 {
			return false
		}
		g.EnqueueMessage("根性で踏みとどまった！", 0.4)
		return true
	}})
	registerAbilityHooks(AbilityPeakGuard, AbilityHooks{ModifyIncomingDamage: func(g *Game, incoming *IncomingDamage, stacks int) {
		if g.state.Player.Health == g.state.Player.MaxHealth {
			incoming.Damage = max(1, incoming.Damage/2)
		}
	}})
	registerAbilityHooks(AbilityLuckyGuard, AbilityHooks{ModifyIncomingDamage: func(g *Game, incoming *IncomingDamage, stacks int) {
		if g.state.Player.Health%10 == 7 {
			incoming.Damage = max(1, incoming.Damage/2)
		}
	}})
	registerAbilityHooks(AbilityCurseGuard, AbilityHooks{ModifyIncomingDamage: func(g *Game, incoming *IncomingDamage, stacks int) {
		armor := g.state.Player.EquippedArmor
		if armor != nil && armor.Cursed && hasEquipmentAbility(armor.Abilities, AbilityCurseGuard) {
			incoming.Damage = max(1, incoming.Damage*4/5)
		}
	}})
	registerAbilityHooks(AbilityShellDodge, AbilityHooks{ModifyIncomingDamage: func(g *Game, incoming *IncomingDamage, stacks int) {
		if incoming.Delivery == EnemyAttackRanged && incoming.Kind != EnemyDamageExplosion && g.abilityIntn()(100) < 50 {
			incoming.Evaded = true
			if armor := g.state.Player.EquippedArmor; armor != nil {
				incoming.EvadedBy = armor.Name
			}
		}
	}})
}

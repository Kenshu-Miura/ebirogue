//go:build !test

package main

const (
	AbilityKeenEdge      EquipmentAbilityID = "keen_edge"
	AbilityDoubleStrike  EquipmentAbilityID = "double_strike"
	AbilityCriticalUp    EquipmentAbilityID = "critical_up"
	AbilityLifeSteal     EquipmentAbilityID = "life_steal"
	AbilityMoneyStrike   EquipmentAbilityID = "money_strike"
	AbilitySatietyStrike EquipmentAbilityID = "satiety_strike"
	AbilityHungerStrike  EquipmentAbilityID = "hunger_strike"
	AbilityChargeStrike  EquipmentAbilityID = "charge_strike"
	AbilityComboStrike   EquipmentAbilityID = "combo_strike"
	AbilityLuckySeven    EquipmentAbilityID = "lucky_seven"
	AbilityMallet        EquipmentAbilityID = "mallet"
	AbilityTrapProbe     EquipmentAbilityID = "trap_probe"
	AbilityEdible        EquipmentAbilityID = "edible"
	AbilityCursePower    EquipmentAbilityID = "curse_power"
	AbilityFlameBlade    EquipmentAbilityID = "flame_blade"
)

func init() {
	registerAbilityHooks(AbilityKeenEdge, AbilityHooks{BeforePlayerAttack: func(g *Game, a *PlayerAttack, stacks int) { a.Accuracy += 5 }})
	registerAbilityHooks(AbilityCriticalUp, AbilityHooks{BeforePlayerAttack: func(g *Game, a *PlayerAttack, stacks int) { a.CriticalDenominator = min(a.CriticalDenominator, 8) }})
	registerAbilityHooks(AbilityHungerStrike, AbilityHooks{BeforePlayerAttack: func(g *Game, a *PlayerAttack, stacks int) {
		if g.state.Player.Satiety == 0 {
			a.Multiplier *= 2
		}
	}})
	registerAbilityHooks(AbilityLuckySeven, AbilityHooks{BeforePlayerAttack: func(g *Game, a *PlayerAttack, stacks int) {
		if g.state.Player.Health%10 == 7 {
			a.ForceCritical = true
		}
	}})
	registerAbilityHooks(AbilityCursePower, AbilityHooks{BeforePlayerAttack: func(g *Game, a *PlayerAttack, stacks int) {
		if a.Weapon != nil && a.Weapon.Cursed {
			a.AttackBonus += 3
		}
	}})
	registerAbilityHooks(AbilityMoneyStrike, AbilityHooks{BeforePlayerAttack: func(g *Game, a *PlayerAttack, stacks int) {
		if g.state.Player.Cash >= 20 {
			a.Multiplier *= 1.5
			a.HitEffects = append(a.HitEffects, func(g *Game) { g.state.Player.Cash -= 20 })
		}
	}})
	registerAbilityHooks(AbilitySatietyStrike, AbilityHooks{BeforePlayerAttack: func(g *Game, a *PlayerAttack, stacks int) {
		if g.state.Player.Satiety >= 10 {
			a.Multiplier *= 1.5
			a.HitEffects = append(a.HitEffects, func(g *Game) { g.state.Player.Satiety -= 2 })
		}
	}})
	registerAbilityHooks(AbilityLifeSteal, AbilityHooks{AfterPlayerHit: func(g *Game, hit PlayerAttackHit, stacks int) {
		p := &g.state.Player
		p.Health = recoveredValue(p.Health, p.MaxHealth, max(1, hit.Damage/4), false)
	}})
	registerAbilityHooks(AbilityDoubleStrike, AbilityHooks{AfterPlayerHit: func(g *Game, hit PlayerAttackHit, stacks int) {
		if hit.Weapon == nil || hit.Defeated || g.inDoubleStrike || g.abilityIntn()(100) >= 25 {
			return
		}
		i := g.enemyIndexAt(hit.Enemy.X, hit.Enemy.Y)
		if i < 0 {
			return
		}
		g.inDoubleStrike = true
		g.EnqueueMessage("連撃！", 0.3)
		g.enqueuePlayerAttack(i, hit.Weapon.Abilities)
		// 命中・外れの Action を実行してから解除する。
		g.Enqueue(Action{Execute: func(g *Game) { g.inDoubleStrike = false }})
	}})
	registerAbilityHooks(AbilityChargeStrike, AbilityHooks{
		BeforePlayerAttack: func(g *Game, a *PlayerAttack, stacks int) {
			if a.Weapon != nil && a.Weapon.AbilityCounter >= 2 {
				a.SureHit, a.ForceCritical = true, true
				w := a.Weapon
				a.HitEffects = append(a.HitEffects, func(g *Game) { w.AbilityCounter = 0 })
			}
		},
		AfterPlayerMiss: func(g *Game, a *PlayerAttack, stacks int) {
			if a.Weapon != nil {
				a.Weapon.AbilityCounter++
			}
		},
	})
	registerAbilityHooks(AbilityComboStrike, AbilityHooks{
		BeforePlayerAttack: func(g *Game, a *PlayerAttack, stacks int) {
			if a.Weapon != nil && a.Weapon.AbilityCounter >= 4 {
				a.SureHit, a.ForceCritical = true, true
				w := a.Weapon
				// 同じ Execute 内の AfterPlayerHit で0へ戻す。発動した一撃は次の連続命中に数えない。
				a.HitEffects = append(a.HitEffects, func(g *Game) { w.AbilityCounter = -1 })
			}
		},
		AfterPlayerHit: func(g *Game, hit PlayerAttackHit, stacks int) {
			if hit.Weapon != nil {
				hit.Weapon.AbilityCounter++
			}
		},
		AfterPlayerMiss: func(g *Game, a *PlayerAttack, stacks int) {
			if a.Weapon != nil {
				a.Weapon.AbilityCounter = 0
			}
		},
	})
	registerAbilityHooks(AbilityMallet, AbilityHooks{AfterPlayerSwing: enqueueMalletSwing})
	registerAbilityHooks(AbilityTrapProbe, AbilityHooks{AfterPlayerSwing: enqueueTrapProbeSwing})
	registerAbilityHooks(AbilityFlameBlade, AbilityHooks{AfterPlayerSwing: enqueueFlameBladeSwing})
}

// wearWeapon はさび止めに関係なく強化値を下げ、装備中の攻撃力へ反映する。
func (g *Game) wearWeapon(w *Weapon) {
	w.Sharpness--
	if g.state.Player.EquippedWeapon == w {
		g.state.Player.AttackPower--
	}
}

func enqueueMalletSwing(g *Game, swing PlayerSwing, stacks int) {
	if swing.HitEnemy {
		return
	}
	w := g.state.Player.EquippedWeapon
	x, y := g.state.Player.X+swing.DX, g.state.Player.Y+swing.DY
	g.Enqueue(Action{Execute: func(g *Game) {
		if g.enemyIndexAt(x, y) >= 0 {
			return
		}
		destroyed := false
		if i := g.mapTrapIndexAt(x, y); i >= 0 {
			g.state.MapTraps = append(g.state.MapTraps[:i], g.state.MapTraps[i+1:]...)
			destroyed = true
		} else {
			for i, item := range g.state.Items {
				ix, iy := item.GetPosition()
				if ix == x && iy == y {
					g.state.Items = append(g.state.Items[:i], g.state.Items[i+1:]...)
					destroyed = true
					break
				}
			}
		}
		if destroyed {
			g.miniMapDirty = true
			g.EnqueueMessage("木づちで壊した。", 0.3)
			if w != nil && g.abilityIntn()(100) < 25 {
				g.wearWeapon(w)
			}
		}
	}})
}

func enqueueTrapProbeSwing(g *Game, swing PlayerSwing, stacks int) {
	x, y := g.state.Player.X, g.state.Player.Y
	g.Enqueue(Action{Execute: func(g *Game) {
		found := false
		for distance := 1; distance <= 2; distance++ {
			for side := -1; side <= 1; side++ {
				tx, ty := x+distance*swing.DX-side*swing.DY, y+distance*swing.DY+side*swing.DX
				if i := g.mapTrapIndexAt(tx, ty); i >= 0 && !g.state.MapTraps[i].Discovered {
					g.state.MapTraps[i].Discovered = true
					found = true
				}
			}
		}
		if found {
			g.miniMapDirty = true
			g.EnqueueMessage("罠を見つけた。", 0.3)
		}
	}})
}

func enqueueFlameBladeSwing(g *Game, swing PlayerSwing, stacks int) {
	if g.state.Player.Health != g.state.Player.MaxHealth {
		return
	}
	x, y := g.state.Player.X, g.state.Player.Y
	g.Enqueue(Action{Execute: func(g *Game) {
		_, target, hit := computeTrajectory(x, y, swing.DX, swing.DY, 10, g.state.Map, g.state.Enemies, false)
		if hit {
			if i := g.enemyIndexAt(target.X, target.Y); i >= 0 {
				g.applyDamageToEnemy(i, 20)
				g.EnqueueMessage("太刀の炎が20ダメージを与えた。", 0.3)
			}
		}
	}})
}

func (g *Game) gnawWeapon(w *Weapon) {
	g.showItemActions, g.showInventory = false, false
	g.selectedItemIndex, g.selectedActionIndex = 0, 0
	if g.state.Player.StatusAilments.MouthSeal > 0 {
		g.EnqueueMessage("口が封じられていてかじれない", 0.4)
		return
	}
	g.Enqueue(Action{Message: "かつおぶしをかじった。", Duration: 0.4, Execute: func(g *Game) {
		p := &g.state.Player
		satiety, _, consumed := gnawResult(p.Satiety, p.MaxSatiety, w.Sharpness)
		p.Satiety = satiety
		if consumed {
			if p.EquippedWeapon == w {
				_, _ = p.UnequipItem(w)
			}
			for i, item := range p.Inventory {
				if item == w {
					p.Inventory = append(p.Inventory[:i], p.Inventory[i+1:]...)
					break
				}
			}
		} else {
			g.wearWeapon(w)
		}
	}})
	g.isActioned = true
}

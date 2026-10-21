//go:build !test

package main

const (
	AbilityRegeneration  EquipmentAbilityID = "regeneration"
	AbilityStomachExpand EquipmentAbilityID = "stomach_expand"
	AbilityStomachShrink EquipmentAbilityID = "stomach_shrink"
	AbilityAlwaysCursed  EquipmentAbilityID = "always_cursed"
	AbilityHappiness     EquipmentAbilityID = "happiness"
	AbilityPoisonWard    EquipmentAbilityID = "poison_ward"
	AbilityConfusionWard EquipmentAbilityID = "confusion_ward"
	AbilitySleepWard     EquipmentAbilityID = "sleep_ward"
	AbilityRustWard      EquipmentAbilityID = "rust_ward"
	AbilityCurseWard     EquipmentAbilityID = "curse_ward"
	AbilitySenseEnemies  EquipmentAbilityID = "sense_enemies"
	AbilitySenseItems    EquipmentAbilityID = "sense_items"
	AbilityClairvoyance  EquipmentAbilityID = "clairvoyance"
	AbilityAppraiser     EquipmentAbilityID = "appraiser"
	AbilityLevitation    EquipmentAbilityID = "levitation"
	AbilityHighJump      EquipmentAbilityID = "high_jump"
	AbilityExplosionRing EquipmentAbilityID = "explosion_ring"
	AbilityLeakItems     EquipmentAbilityID = "leak_items"
	AbilityLeakMoney     EquipmentAbilityID = "leak_money"
	AbilityMonsterCall   EquipmentAbilityID = "monster_call"
	AbilityTrapIncrease  EquipmentAbilityID = "trap_increase"
)

func init() {
	for ability, kind := range map[EquipmentAbilityID]StatusKind{
		AbilityPoisonWard: StatusPoison, AbilityConfusionWard: StatusConfusion,
		AbilitySleepWard: StatusSleep, AbilityRustWard: StatusRust, AbilityCurseWard: StatusCurse,
	} {
		statusImmunities[ability] = []StatusKind{kind}
	}
	registerAbilityHooks(AbilityRegeneration, AbilityHooks{OnTurn: func(g *Game, stacks int) {
		p := &g.state.Player
		if p.Satiety > 0 {
			p.Health = p.recoveredHP(stacks, false)
		}
		if g.moveCount%20 == 0 {
			p.Satiety = max(0, p.Satiety-stacks)
		}
	}})
	registerAbilityHooks(AbilityHappiness, AbilityHooks{OnTurn: func(g *Game, stacks int) {
		g.state.Player.ExperiencePoints += stacks
		g.state.Player.checkLevelUp(g)
	}})
	registerAbilityHooks(AbilityHighJump, AbilityHooks{OnTurn: func(g *Game, stacks int) {
		if ringChance(g.abilityIntn()(100), 2, stacks) {
			g.EnqueueMessage("海老さんは高く飛び上がった！", 0.5)
			warpTrapEffect(g)
		}
	}})
	registerAbilityHooks(AbilityExplosionRing, AbilityHooks{OnTurn: func(g *Game, stacks int) {
		if ringChance(g.abilityIntn()(200), 1, stacks) {
			g.Enqueue(Action{Duration: 0.5, Message: "爆発の指輪が大爆発した！", Execute: explodeRing})
		}
	}})
	registerAbilityHooks(AbilityLeakItems, AbilityHooks{OnTurn: func(g *Game, stacks int) {
		if ringChance(g.abilityIntn()(100), 3, stacks) {
			g.Enqueue(Action{Execute: func(g *Game) {
				dx, dy := directionToDelta(g.state.Player.Direction)
				g.dropPlayerItemsForward(1, dx, dy)
			}})
		}
	}})
	registerAbilityHooks(AbilityLeakMoney, AbilityHooks{OnTurn: func(g *Game, stacks int) {
		if g.state.Player.Cash > 0 && ringChance(g.abilityIntn()(100), 10, stacks) {
			g.Enqueue(Action{Execute: func(g *Game) {
				p := &g.state.Player
				amount := ringMoneyDrop(p.Cash)
				if amount == 0 || !g.itemDropFree(p.X, p.Y) {
					return
				}
				money := buildItemFromTemplate(0, p.X, p.Y).(*Money)
				money.Amount = amount
				p.Cash -= amount
				g.state.Items = append(g.state.Items, money)
				g.updateItemVisibility()
				g.EnqueueMessage("小銭を落とした", 0.4)
			}})
		}
	}})
	registerAbilityHooks(AbilityTrapIncrease, AbilityHooks{OnTurn: func(g *Game, stacks int) {
		if g.moveCount%50 == 0 {
			g.Enqueue(Action{Execute: func(g *Game) { g.addRingTraps(stacks) }})
		}
	}})
	expansionHelpPages = append(expansionHelpPages,
		HelpPage{Title: "指輪（1）", Lines: []string{
			"回復：毎ターンHP+1、20ターンごと満腹度-1", "胃拡張：最大満腹度+25", "胃縮小：最大満腹度-25、必ず呪い付き",
			"しあわせ：毎ターン経験値+1", "毒消し：毒によるパワー低下を防ぐ", "混乱よけ：混乱を防ぐ", "睡眠よけ：睡眠を防ぐ", "錆よけ：武器・盾のさびを防ぐ", "呪いよけ：持ち物への呪いを防ぐ", "気配察知：敵をミニマップに表示する",
		}}, HelpPage{Title: "指輪（2）", Lines: []string{
			"道具感知：床の道具をミニマップに表示", "透視：敵と床の道具をミニマップに表示", "鑑定師：拾った道具を識別する", "浮遊：罠を踏まず、隠れた罠を発見する", "高飛び：毎ターン2%でワープする", "爆発：毎ターン1/200で大爆発、自分はHP1", "垂れ流し：毎ターン3%で道具を落とす、呪い付き", "小銭垂れ流し：毎ターン10%で小銭を落とす", "魔物呼び：湧き間隔が半分、2つで1/3", "罠増し：50ターンごとに隠れた罠を1個追加",
		}})
}

func explodeRing(g *Game) {
	g.explodeTraps(g.state.Player.X, g.state.Player.Y, 1, true, 0)
	p := &g.state.Player
	for i := len(g.state.Enemies) - 1; i >= 0; i-- {
		e := g.state.Enemies[i]
		if abs(e.X-p.X) > 1 || abs(e.Y-p.Y) > 1 {
			continue
		}
		damage := e.Health
		if MonsterDefinitions[e.ID].IsBoss {
			damage = 50
		}
		g.applyDamageToEnemy(i, damage)
	}
	if p.Health > 1 {
		g.damagePlayer(p.Health-1, DamageSource{Kind: DamageSourceSelfDestruct, Detail: "爆発の指輪"})
	}
}

func (g *Game) addRingTraps(count int) {
	for n := 0; n < count; n++ {
		candidates := []Coordinate{}
		for y, row := range g.state.Map {
			for x, tile := range row {
				if tile.Type == "floor" && !tile.Blocked && g.itemDropFree(x, y) && (x != g.state.Player.X || y != g.state.Player.Y) {
					candidates = append(candidates, Coordinate{X: x, Y: y})
				}
			}
		}
		if len(candidates) == 0 {
			return
		}
		intn := g.abilityIntn()
		pos := candidates[intn(len(candidates))]
		id := pickTemplateID(availableTrapIDs(randomMapTrapIDs, g.Floor), intn, 0)
		g.state.MapTraps = append(g.state.MapTraps, createMapTrapByID(id, pos.X, pos.Y))
	}
}

//go:build !test

package main

// 敵用の効果はプレイヤー用と分離し、Action 実行時の対象を引き直す。
func init() {
	for _, id := range []int{0, 1, 2, 3, 5, 6, 7, 8, 9, 10, 11} {
		id := id
		t := mapTrapTemplates[id]
		t.EnemyEffect = func(g *Game, i int, credit bool) { g.applyTrapToEnemy(id, i, credit) }
		mapTrapTemplates[id] = t
	}
}

func (g *Game) enemyTrapAfterMove(i, oldX, oldY int) {
	if i < 0 || i >= len(g.state.Enemies) {
		return
	}
	e := g.state.Enemies[i]
	if e.X == oldX && e.Y == oldY {
		return
	}
	if MonsterDefinitions[e.ID].IsBoss || hasEnemyTrait(e.Traits, EnemyTraitFloating) || e.SpecialMovement == SpecialMovementCreateTrap || isEnemyDisguised(e) || !g.terrainOpenAt(e.X, e.Y) {
		return
	}
	if g.mapTrapIndexAt(e.X, e.Y) < 0 {
		return
	}
	g.Enqueue(Action{Execute: func(g *Game) {
		i := g.findEnemyForDefense(e.ID, e.X, e.Y)
		if i < 0 {
			return
		}
		g.activateInteractionTrap(e.X, e.Y, false, false, 0)
	}})
}

func (g *Game) queueLandedTrap(x, y int, credit bool) {
	g.Enqueue(Action{Execute: func(g *Game) { g.activateInteractionTrap(x, y, credit, true, 0) }})
}

func (g *Game) activateInteractionTrap(x, y int, credit, discover bool, depth int) {
	ti := g.mapTrapIndexAt(x, y)
	if ti < 0 {
		return
	}
	trap := g.state.MapTraps[ti]
	i := g.enemyIndexAt(x, y)
	effect := mapTrapTemplates[trap.TemplateID].EnemyEffect
	if !discover && (i < 0 || effect == nil) {
		return
	}
	failed := g.randIntn(100) < trap.FailRate
	if !discover && failed {
		return
	}
	if discover || (g.state.Player.StatusAilments.Blind == 0 && (g.IsInPlayerVisibleRoom(x, y) || max(abs(x-g.state.Player.X), abs(y-g.state.Player.Y)) <= 1)) {
		g.state.MapTraps[ti].Discovered = true
		g.miniMapDirty = true
	}
	if failed {
		return
	}
	g.EnqueueMessage(trap.Name+"が作動した", 0.5)
	g.executeAudioCue("trap")
	if trap.TemplateID == 3 || trap.TemplateID == 11 {
		// 地雷は一度爆発すると消える。中心を先に除き、再帰で同じ罠を踏まない。
		g.state.MapTraps = append(g.state.MapTraps[:ti], g.state.MapTraps[ti+1:]...)
		g.trapBlast(trap, credit, depth)
		return
	}
	if i >= 0 && effect != nil {
		effect(g, i, credit)
	}
	if g.state.Player.X == x && g.state.Player.Y == y && discover {
		g.steppedTrapSource = trapSource(trap.TemplateID, trap.Name)
		if trap.Effect != nil {
			trap.Effect(g)
		}
	}
}

func (g *Game) removeTrapEnemy(i int) {
	e := g.state.Enemies[i]
	g.dropEnemyHeldItem(i)
	g.state.Enemies = append(g.state.Enemies[:i], g.state.Enemies[i+1:]...)
	g.onEnemyDefeated(e)
	g.miniMapDirty = true
}

func (g *Game) trapEnemyDamage(i, damage int, credit bool) {
	if !credit && damage >= g.state.Enemies[i].Health {
		g.removeTrapEnemy(i)
		return
	}
	g.applyDamageToEnemy(i, damage)
}

func (g *Game) applyTrapToEnemy(id, i int, credit bool) {
	switch id {
	case 0:
		g.applyEnemyStatus(i, StatusSleep, 10)
	case 2:
		g.applyEnemyStatus(i, StatusSlow, 10)
	case 7:
		g.applyEnemyStatus(i, StatusBearTrap, 6)
	case 1:
		g.trapEnemyDamage(i, 5, credit)
	case 8, 9, 10:
		power := map[int]int{8: woodArrowTrapPower, 9: ironArrowTrapPower, 10: fallingRockTrapPower}[id]
		target := g.state.Enemies[i]
		g.trapEnemyDamage(i, rollEnemyAttackDamageWith(power, target.DefensePower, statDefenseMultiplier(target.StatusAilments), g.damageIntn()), credit)
	case 6:
		g.removeTrapEnemy(i)
	case 5:
		candidates := []Coordinate{}
		for y, row := range g.state.Map {
			for x, tile := range row {
				if tile.Type == "floor" && !tile.Blocked && !isOccupied(g, x, y) && g.mapTrapIndexAt(x, y) < 0 {
					candidates = append(candidates, Coordinate{X: x, Y: y})
				}
			}
		}
		if dest, ok := rollWarpDestination(candidates, g.randIntn); ok {
			e := &g.state.Enemies[i]
			e.X, e.Y = dest.X, dest.Y
			e.dx, e.dy = 0, 0
			e.Animating = false
			g.miniMapDirty = true
		}
	}
}

func (g *Game) trapBlast(trap MapTrap, credit bool, depth int) {
	for i := len(g.state.Enemies) - 1; i >= 0; i-- {
		e := g.state.Enemies[i]
		if !withinBlastRadius(trap.X, trap.Y, e.X, e.Y, 1) {
			continue
		}
		damage := mineTrapDamage(e.Health)
		if trap.TemplateID == 11 {
			damage = rollEnemyAttackDamage(largeMineEnemyPower, 0, g.damageIntn())
		}
		g.trapEnemyDamage(i, damage, credit)
	}
	p := g.state.Player
	if withinBlastRadius(trap.X, trap.Y, p.X, p.Y, 1) {
		damage := mineTrapDamage(p.Health)
		if trap.TemplateID == 11 {
			damage = largeMineTrapDamage(p.Health)
		}
		g.explodePlayer(damage, trapSource(trap.TemplateID, trap.Name))
	}
	g.explodeItemsAround(trap.X, trap.Y, 1)
	g.explodeTraps(trap.X, trap.Y, 1, credit, depth)
}

// 誘爆は深さ3で止め、範囲内のほかの罠は破壊する。
func (g *Game) explodeTraps(x, y, radius int, credit bool, depth int) {
	traps := append([]MapTrap(nil), g.state.MapTraps...)
	for _, trap := range traps {
		if !withinBlastRadius(x, y, trap.X, trap.Y, radius) {
			continue
		}
		ti := g.mapTrapIndexAt(trap.X, trap.Y)
		if ti < 0 {
			continue
		}
		g.state.MapTraps = append(g.state.MapTraps[:ti], g.state.MapTraps[ti+1:]...)
		if (trap.TemplateID == 3 || trap.TemplateID == 11) && depth < 3 {
			g.trapBlast(trap, credit, depth+1)
		}
	}
	g.miniMapDirty = true
}

func (g *Game) removeMineAt(x, y int) {
	i := g.mapTrapIndexAt(x, y)
	if i >= 0 && (g.state.MapTraps[i].TemplateID == 3 || g.state.MapTraps[i].TemplateID == 11) {
		g.state.MapTraps = append(g.state.MapTraps[:i], g.state.MapTraps[i+1:]...)
	}
}

// 倍速の2歩目より先に、1歩目に踏んだ罠を解決する。
func (g *Game) enemyOnActiveTrap(i int) bool {
	e := g.state.Enemies[i]
	ti := g.mapTrapIndexAt(e.X, e.Y)
	return ti >= 0 && mapTrapTemplates[g.state.MapTraps[ti].TemplateID].EnemyEffect != nil && !MonsterDefinitions[e.ID].IsBoss && !hasEnemyTrait(e.Traits, EnemyTraitFloating) && e.SpecialMovement != SpecialMovementCreateTrap
}

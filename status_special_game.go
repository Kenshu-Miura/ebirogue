//go:build !test

package main

// specialEnemyTarget は身代わりを優先し、暴走中は最も近い相手を選ぶ。
// -1 は海老さん、-2 は標的なし。透明な海老さんは隣接時だけ標的になる。
func (g *Game) specialEnemyTarget(i int) int {
	e := g.state.Enemies[i]
	p := g.state.Player
	pd := max(abs(e.X-p.X), abs(e.Y-p.Y))
	target, distance := -2, int(^uint(0)>>1)
	decoy := false
	for j, other := range g.state.Enemies {
		if j == i || other.Health <= 0 || peacefulShopkeeper(other) {
			continue
		}
		isDecoy := statusActive(other.StatusAilments, StatusDecoy)
		if !isDecoy && (!statusActive(e.StatusAilments, StatusBerserk) || statusActive(other.StatusAilments, StatusInvisible)) {
			continue
		}
		d := max(abs(e.X-other.X), abs(e.Y-other.Y))
		// 身代わりの誘引は同じ部屋か8マス以内。
		if isDecoy && d > 8 && !isSameRoom(e.X, e.Y, other.X, other.Y, g.rooms) {
			continue
		}
		if (isDecoy && !decoy) || (isDecoy == decoy && d < distance) {
			target, distance, decoy = j, d, isDecoy
		}
	}
	if !decoy && (!statusActive(p.StatusAilments, StatusInvisible) || pd <= 1) && pd <= distance {
		return -1
	}
	return target
}

func (g *Game) actSpecialEnemyTarget(i int) bool {
	target := g.specialEnemyTarget(i)
	if target == -1 {
		if statusActive(g.state.Enemies[i].StatusAilments, StatusBerserk) {
			e, p := g.state.Enemies[i], g.state.Player
			if max(abs(e.X-p.X), abs(e.Y-p.Y)) <= 1 && g.specialMeleeClear(e.X, e.Y, p.X, p.Y) {
				g.enqueueEnemyNormalAttack(i)
			} else {
				g.stepEnemyToward(i, p.X, p.Y)
			}
			return true
		}
		return false
	}
	if target == -2 {
		g.state.Enemies[i].PlayerDiscovered = false
		moveRandomly(g, i)
		return true
	}
	e, t := g.state.Enemies[i], g.state.Enemies[target]
	if max(abs(e.X-t.X), abs(e.Y-t.Y)) <= 1 && g.specialMeleeClear(e.X, e.Y, t.X, t.Y) {
		g.AttackEnemyFromBlindEnemy(i, target)
	} else {
		g.stepEnemyToward(i, t.X, t.Y)
	}
	return true
}

func (g *Game) specialMeleeClear(x, y, tx, ty int) bool {
	if x == tx || y == ty {
		return true
	}
	return !g.state.Map[y][tx].Blocked && !g.state.Map[ty][x].Blocked
}

// 自動行動も通常の isActioned → CheckCombatState の経路で1ターン進める。
func (g *Game) actBerserkPlayer() {
	p := &g.state.Player
	nearest, distance := -1, int(^uint(0)>>1)
	for i, e := range g.state.Enemies {
		if e.Health <= 0 || !g.specialEnemyVisible(e) || isEnemyDisguised(e) {
			continue
		}
		d := max(abs(e.X-p.X), abs(e.Y-p.Y))
		if d < distance {
			nearest, distance = i, d
		}
	}
	if nearest >= 0 {
		e := g.state.Enemies[nearest]
		dx, dy := sign(e.X-p.X), sign(e.Y-p.Y)
		p.Direction = determineDirection(dx, dy)
		if distance == 1 && g.specialMeleeClear(p.X, p.Y, e.X, e.Y) {
			g.CheckForEnemies(dx, dy)
		} else {
			g.MovePlayer(dx, dy)
		}
	}
	g.isActioned = true
	// 敵なし・移動不可でも次フレームで通常のキュー処理へ進む。
	g.Enqueue(Action{Duration: statusMessageDuration, Execute: func(g *Game) {}})
}

func (g *Game) specialEnemyVisible(e Enemy) bool {
	return !statusActive(e.StatusAilments, StatusInvisible) || g.playerHasAbility(AbilitySenseEnemies) || g.playerHasAbility(AbilityClairvoyance)
}

func (g *Game) specialPlayerDefense() int {
	if statusActive(g.state.Player.StatusAilments, StatusNigiri) {
		return 0
	}
	return g.state.Player.DefensePower
}

func (g *Game) specialItemUseBlocked() bool {
	return statusActive(g.state.Player.StatusAilments, StatusNigiri) || statusActive(g.state.Player.StatusAilments, StatusBerserk)
}

func (g *Game) rejectSpecialItemUse() bool {
	if !g.specialItemUseBlocked() {
		return false
	}
	g.EnqueueMessage("今の姿では道具を使えない", statusMessageDuration)
	return true
}

//go:build !test

package main

// 敵は罠を踏まない（開発者の決定、2026-10-22）。罠が作動するのは海老さんが踏んだとき（maptraps.go の
// stepOnTrap）と、海老さんが投げた道具・撃った矢が罠の上に落ちたとき（queueLandedTrap）だけ。
// 地雷・大型地雷の爆風は周りの敵にも当たる（trapBlast）。

func (g *Game) queueLandedTrap(x, y int, credit bool) {
	g.Enqueue(Action{Execute: func(g *Game) { g.activateLandedTrap(x, y, credit) }})
}

// activateLandedTrap は道具・矢が落ちたマスの罠を発見済みにして作動させる。
// 効果は海老さんがそのマスにいるときだけ受け、地雷は爆発して消える。
func (g *Game) activateLandedTrap(x, y int, credit bool) {
	ti := g.mapTrapIndexAt(x, y)
	if ti < 0 {
		return
	}
	trap := g.state.MapTraps[ti]
	failed := g.randIntn(100) < trap.FailRate
	g.state.MapTraps[ti].Discovered = true
	g.miniMapDirty = true
	if failed {
		return
	}
	g.EnqueueMessage(trap.Name+"が作動した", 0.5)
	g.executeAudioCue("trap")
	if trap.TemplateID == 3 || trap.TemplateID == 11 {
		// 地雷は一度爆発すると消える。中心を先に除き、再帰で同じ罠を踏まない。
		g.state.MapTraps = append(g.state.MapTraps[:ti], g.state.MapTraps[ti+1:]...)
		hp := g.state.Player.Health
		g.trapBlast(trap, credit, 0, &hp)
		return
	}
	if g.state.Player.X == x && g.state.Player.Y == y {
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
	if !credit && peacefulShopkeeper(g.state.Enemies[i]) {
		return // 海老さんが起こしたのでない罠の爆発では、怒っていない店主は傷つかない（wave10-B）
	}
	if !credit && damage >= g.state.Enemies[i].Health {
		g.removeTrapEnemy(i)
		return
	}
	g.applyDamageToEnemy(i, damage)
}

// trapBlast は地雷1つの爆発。海老さんへのダメージは積んだ Action で後から減らすので、
// 同じ誘爆の中では hp（それまでの爆発を引いた後の見込みの HP）から割合を計算する。
// 現在の HP から計算すると、2つの地雷がそれぞれ同じ HP の半分を削って倒れてしまう。
func (g *Game) trapBlast(trap MapTrap, credit bool, depth int, hp *int) {
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
		damage := mineTrapDamage(*hp)
		if trap.TemplateID == 11 {
			damage = largeMineTrapDamage(*hp)
		}
		*hp -= g.explodePlayer(damage, trapSource(trap.TemplateID, trap.Name))
	}
	g.explodeItemsAround(trap.X, trap.Y, 1)
	g.explodeTrapsChain(trap.X, trap.Y, 1, credit, depth, hp)
}

// 誘爆は深さ3で止め、範囲内のほかの罠は破壊する。
// 海老さんの HP は呼んだ時点の値から数える（自分のダメージを積んでから呼ぶ側は、先に HP を減らしておく）。
func (g *Game) explodeTraps(x, y, radius int, credit bool, depth int) {
	hp := g.state.Player.Health
	g.explodeTrapsChain(x, y, radius, credit, depth, &hp)
}

func (g *Game) explodeTrapsChain(x, y, radius int, credit bool, depth int, hp *int) {
	g.breakCrackedAround(x, y, radius)
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
			g.trapBlast(trap, credit, depth+1, hp)
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

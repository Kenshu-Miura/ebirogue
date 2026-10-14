//go:build !test

package main

import "fmt"

const (
	bossShellDefense        = 8
	bossShellDuration       = 3
	bossSummonLimit         = 2
	bossInkDuration         = 3
	bossCrabControlTurns    = 3
	bossOctopusControlTurns = 2
	bossDragonControlTurns  = 1
	bossDragonRageDivisor   = 2
)

// テーブル生成後に登録し、召喚処理からテーブルを参照する初期化循環を避ける。
func init() {
	for id, special := range map[int]SpecialAttackFunc{42: crabBossSpecial, 43: octopusBossSpecial} {
		definition := MonsterDefinitions[id]
		definition.SpecialAttack = special
		MonsterDefinitions[id] = definition
	}
}

func bossDefinition(id int, name, kind string, hp, attack, defense, exp int, special SpecialAttackFunc) MonsterDefinition {
	return MonsterDefinition{ID: id, Name: name, Type: kind, Char: 'B', Health: hp, MaxHealth: hp,
		AttackPower: attack, DefensePower: defense, ExperiencePoints: exp, IsBoss: true,
		SpecialAttack: special, SpecialAttackProbability: 0.35}
}

// ボスの妨害は短時間だけ有効。封印・鈍足・目潰しは通常どおり有効。
// 最終ボスだけ金縛りに耐性を持つ。仮眠（-1）は通常の起床処理に任せる。
func normalizeBossStatus(e *Enemy) {
	if e.Type != "BossCrab" && e.Type != "BossOctopus" && e.Type != "BossDragon" {
		return
	}
	limit := 0
	switch e.ID {
	case 42:
		limit = bossCrabControlTurns
	case 43:
		limit = bossOctopusControlTurns
	case 44:
		limit = bossDragonControlTurns
		e.StatusAilments.Paralysis = false
	}
	if limit == 0 {
		return
	}
	e.StatusAilments.Sleep = min(e.StatusAilments.Sleep, limit)
	e.StatusAilments.Confusion = min(e.StatusAilments.Confusion, limit)
}

func advanceBossState(e *Enemy) {
	if e.BossShellTurns > 0 {
		e.BossShellTurns--
		if e.BossShellTurns == 0 || e.StatusAilments.Seal {
			e.DefensePower -= bossShellDefense
			e.BossShellTurns = 0
		}
	}
}

func bossExtraAction(e Enemy) bool {
	return e.Type == "BossDragon" && e.Health <= e.MaxHealth/bossDragonRageDivisor && !e.StatusAilments.Seal
}

func (g *Game) livingBossIndex(id int) int {
	for i := range g.state.Enemies {
		if g.state.Enemies[i].ID == id && g.state.Enemies[i].Health > 0 {
			return i
		}
	}
	return -1
}

func crabBossSpecial(e *Enemy, g *Game) {
	id := e.ID
	shell := g.randIntn(2) == 0
	message := "鉄壁ガニ王は手下を呼んだ！"
	if shell {
		message = "鉄壁ガニ王は殻にこもった！"
	}
	g.Enqueue(Action{Duration: 0.5, Message: message, Execute: func(g *Game) {
		i := g.livingBossIndex(id)
		if i < 0 || g.state.Enemies[i].StatusAilments.Seal {
			return
		}
		if shell {
			e := &g.state.Enemies[i]
			if e.BossShellTurns == 0 {
				e.DefensePower += bossShellDefense
			}
			e.BossShellTurns = bossShellDuration
			return
		}
		g.summonBossMinions(i)
	}})
}

func (g *Game) summonBossMinions(i int) {
	if g.genocidedMonsterIDs[5] {
		return
	}
	boss := g.state.Enemies[i]
	count := 0
	for _, enemy := range g.state.Enemies {
		if enemy.BossMinion {
			count++
		}
	}
	for _, d := range getDirections() {
		if count >= bossSummonLimit || len(g.state.Enemies) >= MaxEnemiesOnFloor {
			break
		}
		x, y := boss.X+d.dx, boss.Y+d.dy
		if !isPositionFree(g, x, y, i) {
			continue
		}
		minion := CreateEnemyByID(5, x, y)
		minion.BossMinion = true
		minion.PlayerDiscovered = true
		g.state.Enemies = append(g.state.Enemies, minion)
		count++
	}
	g.miniMapDirty = true
}

func octopusBossSpecial(e *Enemy, g *Game) {
	id := e.ID
	if g.randIntn(2) == 0 {
		g.Enqueue(Action{Duration: 0.5, Message: "深海ダコ将軍が墨を吐いた！", Execute: func(g *Game) {
			i := g.livingBossIndex(id)
			if i >= 0 && !g.state.Enemies[i].StatusAilments.Seal && !g.playerOnSanctuary() && !g.playerBlocksStatus(StatusBlind) {
				g.state.Player.StatusAilments.Blind = max(g.state.Player.StatusAilments.Blind, bossInkDuration)
			}
		}})
		return
	}
	// 連撃も共通の盾防御・死因記録・ダメージ式を経由する。
	if i := g.livingBossIndex(id); i >= 0 {
		g.EnqueueMessage("深海ダコ将軍の二連撃！", 0.3)
		g.enqueueEnemyNormalAttack(i)
		g.enqueueEnemyNormalAttack(i)
	}
}

// 遠くからの手下召喚・場所替えは共通の状態処理後にだけ選ぶ。
func (g *Game) tryBossSpecial(i int) bool {
	e := &g.state.Enemies[i]
	if e.Type != "BossCrab" && e.Type != "BossOctopus" && e.Type != "BossDragon" {
		return false
	}
	if !e.PlayerDiscovered || e.StatusAilments.Seal || g.playerOnSanctuary() {
		return false
	}
	distance := max(abs(e.X-g.state.Player.X), abs(e.Y-g.state.Player.Y))
	if e.ID == 44 {
		kind := RangedAttackFire
		if g.randIntn(2) == 0 {
			kind = RangedAttackMagic
		}
		e.RangedAttack = RangedAttackDefinition{Kind: kind, MinRange: 2, MaxRange: 6, AttackPower: 18}
		return false
	}
	if distance < 2 || distance > 5 || !isSameRoom(e.X, e.Y, g.state.Player.X, g.state.Player.Y, g.rooms) || g.randIntn(4) != 0 {
		return false
	}
	if e.ID == 42 {
		e.SpecialAttack(e, g)
		return true
	}
	if e.ID != 43 || g.playerImmovable() {
		return false
	}
	g.Enqueue(Action{Duration: 0.5, Message: "深海ダコ将軍と場所が入れ替わった！", Execute: func(g *Game) {
		i := g.livingBossIndex(43)
		if i < 0 || g.state.Enemies[i].StatusAilments.Seal || g.playerOnSanctuary() || g.playerImmovable() {
			return
		}
		e := &g.state.Enemies[i]
		e.X, g.state.Player.X = g.state.Player.X, e.X
		e.Y, g.state.Player.Y = g.state.Player.Y, e.Y
		e.Animating = false
		g.miniMapDirty = true
		g.updateTileBrightness()
		g.checkForTrapAtPosition(g.state.Player.X, g.state.Player.Y)
	}})
	return true
}

func bossDefeatMessage(id int) string {
	return fmt.Sprintf("%sを打ち破った！", MonsterDefinitions[id].Name)
}

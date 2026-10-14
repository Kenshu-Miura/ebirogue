//go:build !test

package main

import "fmt"

// 位置を動かす共通処理（content-expansion-plan.md の F3）: 吹き飛ばし・引き寄せ・持ち物を落とす。
// どれも Action の Execute の中から呼ぶ（その場で位置を変え、結果のメッセージを積む）。
// 不動（AbilityImmovable）は吹き飛ばし・引き寄せ・場所替えを、受け身（AbilityBreakFall）は
// 吹き飛ばされてぶつかったときのダメージを防ぐ。

// DisplacementResult は吹き飛ばし・引き寄せ1回の結果。
type DisplacementResult struct {
	Moved     int              // 実際に動いたマス数
	Stop      DisplacementStop // 止まった理由
	Prevented bool             // 不動で動かなかった
	Damage    int              // ぶつかって受けたダメージ（受け身なら0）
}

// playerImmovable は不動の能力で、吹き飛ばし・引き寄せ・場所替えを受けないかを返す。
func (g *Game) playerImmovable() bool {
	return g.playerHasAbility(AbilityImmovable)
}

// playerBreaksFall は受け身の能力で、吹き飛ばされてぶつかってもダメージを受けないかを返す。
func (g *Game) playerBreaksFall() bool {
	return g.playerHasAbility(AbilityBreakFall)
}

// terrainOpenAt はマップ内で通れる地形かを返す（キャラクターの有無は見ない）。
func (g *Game) terrainOpenAt(x, y int) bool {
	return y >= 0 && y < len(g.state.Map) && x >= 0 && x < len(g.state.Map[y]) && !g.state.Map[y][x].Blocked
}

// relocatePlayer はプレイヤーを (x, y) へ移し、視界・踏破済みの表示を更新する（罠は踏まない）。
func (g *Game) relocatePlayer(x, y int) {
	g.state.Player.X, g.state.Player.Y = x, y
	g.dx, g.dy = 0, 0
	g.Animating = false
	g.updateTileBrightness()
	g.MarkVisitedTiles(x, y)
	g.MarkRoomVisited(x, y)
	g.miniMapDirty = true
}

// knockbackPlayer はプレイヤーを (dx, dy) 方向へ最大 distance マス吹き飛ばす。
// 壁や敵に当たって止まると collisionDamage を受ける（受け身なら受けない）。
// 止まったマスに罠があれば踏む。不動なら動かない。source は死因の記録に使う。
func (g *Game) knockbackPlayer(dx, dy, distance, collisionDamage int, source DamageSource) DisplacementResult {
	if g.playerImmovable() {
		g.EnqueueMessage("海老さんは踏みとどまった", 0.4)
		return DisplacementResult{Prevented: true}
	}
	player := &g.state.Player
	path, stop := displacementPath(player.X, player.Y, dx, dy, distance, g.terrainOpenAt, func(x, y int) bool { return g.enemyIndexAt(x, y) >= 0 })
	result := DisplacementResult{Moved: len(path), Stop: stop}
	if len(path) > 0 {
		end := path[len(path)-1]
		g.relocatePlayer(end.X, end.Y)
		g.EnqueueMessage(fmt.Sprintf("海老さんは%dマス吹き飛ばされた", len(path)), 0.4)
	}
	if stop != DisplacementFinished && collisionDamage > 0 {
		if g.playerBreaksFall() {
			g.EnqueueMessage("海老さんは受け身をとった", 0.4)
		} else {
			result.Damage = collisionDamage
			g.damagePlayer(collisionDamage, source)
			g.EnqueueMessage(fmt.Sprintf("ぶつかって%dダメージを受けた", collisionDamage), 0.4)
			g.state.Player.checkDeath(g)
		}
	}
	if len(path) > 0 && !g.playerDead {
		g.checkForTrapAtPosition(player.X, player.Y)
	}
	return result
}

// knockbackEnemy は敵 index を (dx, dy) 方向へ最大 distance マス吹き飛ばす。
// 壁やほかの敵・プレイヤーに当たって止まると collisionDamage を受ける（倒れたら通常の撃破処理）。
// ボスは吹き飛ばない。
func (g *Game) knockbackEnemy(index, dx, dy, distance, collisionDamage int) DisplacementResult {
	if index < 0 || index >= len(g.state.Enemies) {
		return DisplacementResult{}
	}
	enemy := &g.state.Enemies[index]
	if MonsterDefinitions[enemy.ID].IsBoss {
		g.EnqueueMessage(fmt.Sprintf("%sはびくともしない", g.enemyDisplayName(enemy.Name)), 0.4)
		return DisplacementResult{Prevented: true}
	}
	path, stop := displacementPath(enemy.X, enemy.Y, dx, dy, distance, g.terrainOpenAt, func(x, y int) bool { return isOccupied(g, x, y) })
	result := DisplacementResult{Moved: len(path), Stop: stop}
	name := g.enemyDisplayName(enemy.Name)
	if len(path) > 0 {
		end := path[len(path)-1]
		enemy.X, enemy.Y = end.X, end.Y
		enemy.dx, enemy.dy = 0, 0
		enemy.Animating = false
		g.miniMapDirty = true
		g.EnqueueMessage(fmt.Sprintf("%sは%dマス吹き飛んだ", name, len(path)), 0.4)
	}
	if stop != DisplacementFinished && collisionDamage > 0 {
		result.Damage = collisionDamage
		g.EnqueueMessage(fmt.Sprintf("%sはぶつかって%dダメージを受けた", name, collisionDamage), 0.4)
		g.applyDamageToEnemy(index, collisionDamage)
	}
	return result
}

// pullPlayer はプレイヤーを (dx, dy) 方向へ最大 distance マス引き寄せる（引き寄せる敵の手前で止まる）。
// 途中のマスに罠があれば、そこで止まって罠を踏む。不動なら動かない。ぶつかってもダメージは無い。
func (g *Game) pullPlayer(dx, dy, distance int) DisplacementResult {
	if g.playerImmovable() {
		g.EnqueueMessage("海老さんは踏みとどまった", 0.4)
		return DisplacementResult{Prevented: true}
	}
	player := &g.state.Player
	path, stop := displacementPath(player.X, player.Y, dx, dy, distance, g.terrainOpenAt, func(x, y int) bool { return g.enemyIndexAt(x, y) >= 0 })
	for i, c := range path {
		if g.mapTrapIndexAt(c.X, c.Y) >= 0 {
			path, stop = path[:i+1], DisplacementFinished
			break
		}
	}
	result := DisplacementResult{Moved: len(path), Stop: stop}
	if len(path) == 0 {
		return result
	}
	end := path[len(path)-1]
	g.relocatePlayer(end.X, end.Y)
	g.EnqueueMessage("海老さんは引き寄せられた", 0.4)
	g.checkForTrapAtPosition(end.X, end.Y)
	return result
}

// mapTrapIndexAt は (x, y) にある罠の添字を返す（無ければ -1）。
func (g *Game) mapTrapIndexAt(x, y int) int {
	for i, trap := range g.state.MapTraps {
		if trap.X == x && trap.Y == y {
			return i
		}
	}
	return -1
}

// itemDropFree は持ち物を落とせる床（部屋の床か通路、道具・罠・階段が無い）かを返す。
func (g *Game) itemDropFree(x, y int) bool {
	if !g.terrainOpenAt(x, y) {
		return false
	}
	tile := g.state.Map[y][x]
	if tile.Type != "floor" && tile.Type != "corridor" {
		return false
	}
	for _, item := range g.state.Items {
		if ix, iy := item.GetPosition(); ix == x && iy == y {
			return false
		}
	}
	return g.mapTrapIndexAt(x, y) < 0
}

// dropPlayerItemsForward は所持品から count 個（装備中・床に貼りついた聖域のカードを除く）を選び、
// プレイヤーの (dx, dy) 方向の1〜2マス先へ、置けなければ周囲8マスへ落とす。落とした道具を返す。
// 置き場所が足りない分は落とさない。落とす道具の抽選は g.displacementIntn() を使い、置き場所は前方に近い順。
func (g *Game) dropPlayerItemsForward(count, dx, dy int) []Item {
	player := &g.state.Player
	positions := forwardDropPositions(player.X, player.Y, dx, dy, 2, g.itemDropFree)
	candidates := []int{}
	for i, item := range player.Inventory {
		if equipment, ok := item.(Equipable); ok && player.IsEquipped(equipment) {
			continue
		}
		if isStuckSanctuaryItem(item) {
			continue
		}
		candidates = append(candidates, i)
	}
	intn := g.displacementIntn()
	selected := rollItemTrapIndices(candidates, min(count, len(positions)), intn)
	items := make([]Item, len(selected))
	for i, index := range selected {
		items[i] = player.Inventory[index]
	}
	// 前方に近い置き場所から順に使う
	for i, item := range items {
		g.dropItemFromTrap(item, positions[i])
		g.EnqueueMessage(fmt.Sprintf("%sを落とした", item.GetName()), 0.4)
	}
	return items
}

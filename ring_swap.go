//go:build !test

package main

import "fmt"

// すれちがいの指輪（ID 203、段4）: 敵のいるマスへ動くと、その敵と位置を入れ替えて進む。
//
// 決めたこと（テストは content_rings_stage4_test.go）:
//   - 入れ替えは普通の移動と同じく1ターン。進んだ先の罠を踏み、足元の道具を拾う。敵は海老さんが
//     いたマスへ移る（そのマスの罠は踏まない）。
//   - ボスとは入れ替わらない（ボスは吹き飛ばし・場所替えでも動かない）。普通に敵へ向かって歩いたときと
//     同じく、何も起きずターンも進まない。
//   - 壁の中にいる敵（ユウレイエビ系）とは入れ替わらない（海老さんは壁に入れない）。
//   - 海老さんが聖域のカードの上にいると入れ替わらない（敵は聖域に入れない）。
//   - 眠っている・金縛りの敵とも入れ替わり、起こさない。トラバサミ・睡眠・混乱中の海老さんは
//     もともと狙った向きへ動けないので入れ替わらない。
//   - 道具・階段に化けた敵は、従来どおり正体を現す（1ターン）だけで入れ替わらない。別の敵に化けた敵とは
//     入れ替わり、正体は現さない（攻撃ではないため）。

// AbilitySwapPlaces はすれちがいの指輪の能力。
const AbilitySwapPlaces EquipmentAbilityID = "swap_places"

func init() {
	expansionHelpPages = append(expansionHelpPages, HelpPage{Title: "指輪（3）", Lines: []string{
		"すれちがい：敵のいるマスへ歩くと位置を入れ替える",
		"すれちがい：ボス・壁の中の敵とは入れ替わらない",
		"すれちがい：海老さんが聖域の上にいると入れ替わらない",
	}})
}

// trySwapPlacesWithEnemy はすれちがいの指輪で、(dx, dy) の隣にいる敵 i と位置を入れ替える。
// 入れ替えたら true（1ターンの移動として扱う）。MovePlayer が敵のいるマスへ動こうとしたときに呼ぶ。
func (g *Game) trySwapPlacesWithEnemy(i, dx, dy int) bool {
	if !g.playerHasAbility(AbilitySwapPlaces) || i < 0 || i >= len(g.state.Enemies) {
		return false
	}
	enemy := &g.state.Enemies[i]
	player := &g.state.Player
	if MonsterDefinitions[enemy.ID].IsBoss || !g.terrainOpenAt(enemy.X, enemy.Y) || g.playerOnSanctuary() {
		return false
	}
	if enemy.X != player.X+dx || enemy.Y != player.Y+dy {
		return false
	}
	defer g.enemyTrapAfterMove(i, enemy.X, enemy.Y)
	oldX, oldY := player.X, player.Y
	player.X, player.Y = enemy.X, enemy.Y
	player.Direction = determineDirection(dx, dy)
	enemy.X, enemy.Y = oldX, oldY
	enemy.dx, enemy.dy = -dx, -dy
	enemy.Animating = true
	g.miniMapDirty = true
	g.isActioned = true
	g.EnqueueMessage(fmt.Sprintf("%sとすれちがった", g.enemyDisplayName(enemy.Name)), 0.3)
	g.checkForTrapAtPosition(player.X, player.Y)
	g.PickupItem()
	return true
}

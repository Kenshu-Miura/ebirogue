//go:build !test

package main

import (
	_ "image/png" // PNG画像を読み込むために必要
	"math"
)

func (g *Game) updateEnemyYOffset() {
	if g.isCombatActive {
		// 戦闘がアクティブな場合は、オフセットを0に保持
		g.enemyYOffset = 0
	} else {
		// 敵のY座標オフセットの更新
		g.enemyYOffsetTimer++
		if g.enemyYOffsetTimer >= 30 { // 0.5秒ごとに変更 (60FPSを仮定)
			if g.enemyYOffset == -3 {
				g.enemyYOffset = 0 // オフセットを0に変更
			} else {
				g.enemyYOffset = -3 // オフセットを-3に変更
			}
			g.enemyYOffsetTimer = 0
		}
	}
}

func (g *Game) UpdateAttackTimer() {
	// Check the attack timer and reset temporary player position if needed
	if g.attackTimer > 0 {
		progress := 1 - g.attackTimer/0.5 // progress ranges from 0 to 1 over 0.5 seconds
		angle := math.Pi * progress       // angle ranges from 0 to Pi
		value := 30 * math.Sin(angle)     // value ranges from 0 to 20 to 0

		g.tmpPlayerOffsetX = value
		g.tmpPlayerOffsetY = value

		g.attackTimer -= (1 / 60.0) // assuming Update is called 60 times per second
		if g.attackTimer <= 0 {
			g.attackTimer = 0 // reset timer
			g.tmpPlayerOffsetX = 0
			g.tmpPlayerOffsetY = 0
		}
	}
}

func (g *Game) HandleAnimationProgress() {
	if g.Animating && g.fastForward {
		g.Animating = false
		g.AnimationProgressInt = 0
		return
	}
	if g.Animating {
		g.AnimationProgressInt += 1
		if g.AnimationProgressInt >= 10 {
			g.Animating = false
			g.AnimationProgressInt = 0
		}
	}
}

func (g *Game) HandleEnemyAttackTimers() {
	for i := range g.state.Enemies {
		if g.state.Enemies[i].AttackTimer > 0 {
			progress := 1 - g.state.Enemies[i].AttackTimer/0.5
			angle := math.Pi * progress
			value := 30 * math.Sin(angle)

			switch g.state.Enemies[i].AttackDirection {
			case Up:
				g.state.Enemies[i].OffsetY = int(-value)
			case Down:
				g.state.Enemies[i].OffsetY = int(value)
			case Left:
				g.state.Enemies[i].OffsetX = int(-value)
			case Right:
				g.state.Enemies[i].OffsetX = int(value)
			case UpRight:
				g.state.Enemies[i].OffsetX = int(value)
				g.state.Enemies[i].OffsetY = int(-value)
			case DownRight:
				g.state.Enemies[i].OffsetX = int(value)
				g.state.Enemies[i].OffsetY = int(value)
			case UpLeft:
				g.state.Enemies[i].OffsetX = int(-value)
				g.state.Enemies[i].OffsetY = int(-value)
			case DownLeft:
				g.state.Enemies[i].OffsetX = int(-value)
				g.state.Enemies[i].OffsetY = int(value)
			}

			g.state.Enemies[i].AttackTimer -= (1 / 60.0)
		} else {
			g.state.Enemies[i].OffsetX = 0
			g.state.Enemies[i].OffsetY = 0
		}
	}
}

func (g *Game) HandleActionQueue() {
	if len(g.ActionQueue.Queue) > 0 {
		if g.ActionDurationCounter <= 0 {
			action := g.ActionQueue.Queue[0]
			if g.playerDead {
				g.debugf("DEBUG: Processing action while dead: Message='%s', Duration=%f", action.Message, action.Duration)
			}
			g.processAction(action)
			if g.fastForward {
				g.ActionDurationCounter = 0 // 演出スキップ: メッセージの表示時間を待たない
			}
		}
	}

	if g.ActionDurationCounter > 0 {
		g.ActionDurationCounter -= (1 / 60.0) // decrement the counter every frame
	}

	if len(g.ActionQueue.Queue) > 0 && g.ActionDurationCounter <= 0 {
		if g.playerDead {
			g.debugf("DEBUG: Removing action from queue, remaining: %d", len(g.ActionQueue.Queue)-1)
		}
		g.ActionDurationCounter = 0
		g.ActionQueue.Queue = g.ActionQueue.Queue[1:]
	}

	if len(g.ActionQueue.Queue) == 0 && g.isCombatActive && g.ActionDurationCounter <= 0 {
		g.isCombatActive = false // reset the combat active flag when the queue is empty
		// プレイヤーが死亡している場合は、showDescriptionをfalseにしない（死亡メッセージを表示するため）
		if !g.playerDead {
			g.showDescription = false
		}
	}
}

// CheckCombatState は行動の処理が終わった後に1ターンを進める。
// 行動を消費する操作（移動・攻撃・射撃・足踏み・道具・扉・睡眠中の自動行動など）は
// isActioned を立てるだけにし、ターン数・湧き・風の警告（AdvanceTurn）と敵ターンはここで1回だけ処理する。
func (g *Game) CheckCombatState() {
	if g.isActioned {
		if !g.isCombatActive {
			haste := g.state.Player.StatusAilments.Haste > 0 && g.state.Player.StatusAilments.Slow <= 0
			g.AdvanceTurn()
			g.IncrementMoveCount()
			if !haste || g.moveCount%2 == 0 {
				g.MoveEnemies()
			}
			if shouldTakeExtraEnemyTurn(g.state.Player.StatusAilments.Slow, g.moveCount) {
				g.MoveEnemies()
			}
			g.isActioned = false
		}
	}
}

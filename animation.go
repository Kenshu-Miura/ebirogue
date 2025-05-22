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
			g.processAction(action)
		}
	}

	if g.ActionDurationCounter > 0 {
		g.ActionDurationCounter -= (1 / 60.0) // decrement the counter every frame
	}

	if len(g.ActionQueue.Queue) > 0 && g.ActionDurationCounter <= 0 {
		g.ActionDurationCounter = 0
		g.ActionQueue.Queue = g.ActionQueue.Queue[1:]
	}

	if len(g.ActionQueue.Queue) == 0 && g.isCombatActive && g.ActionDurationCounter <= 0 {
		g.isCombatActive = false // reset the combat active flag when the queue is empty
		g.showDescription = false
	}
}

func (g *Game) CheckCombatState() {
	if g.isActioned {
		if !g.isCombatActive {
			g.IncrementMoveCount()
			g.MoveEnemies()
			g.isActioned = false
		}
	}
}

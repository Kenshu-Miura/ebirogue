package main

import (
	"fmt"
	"strings"
)

func mineTrapDamage(currentHealth int) int {
	if currentHealth <= 1 {
		return 0
	}
	return currentHealth / 2
}

func shouldTakeExtraEnemyTurn(slowTurns, moveCount int) bool {
	return slowTurns > 0 && moveCount%2 == 0
}

// 睡眠のカードで眠らされた敵が目覚めた時に倍速化するターン数
const hasteOnWakeTurns = 5

// wakeFromSleep は睡眠状態を解除し、目覚め時倍速化フラグが立っていれば倍速状態にする。
// 倍速化した場合は true を返す。
func wakeFromSleep(status *StatusAilments) bool {
	status.Sleep = 0
	if !status.HasteOnWake {
		return false
	}
	status.HasteOnWake = false
	status.Haste = hasteOnWakeTurns
	return true
}

func formatPlayerStatus(status StatusAilments) string {
	statuses := make([]string, 0, 8)
	if status.Poison > 0 {
		statuses = append(statuses, fmt.Sprintf("毒(%d)", status.Poison))
	}
	if status.Slow > 0 {
		statuses = append(statuses, fmt.Sprintf("鈍足(%d)", status.Slow))
	}
	if status.Haste > 0 {
		statuses = append(statuses, fmt.Sprintf("倍速(%d)", status.Haste))
	}
	if status.Sleep > 0 {
		statuses = append(statuses, fmt.Sprintf("睡眠(%d)", status.Sleep))
	}
	if status.Confusion > 0 {
		statuses = append(statuses, fmt.Sprintf("混乱(%d)", status.Confusion))
	}
	if status.Blind > 0 {
		statuses = append(statuses, fmt.Sprintf("目潰し(%d)", status.Blind))
	}
	if status.MouthSeal > 0 {
		statuses = append(statuses, fmt.Sprintf("口封じ(%d)", status.MouthSeal))
	}
	if status.Paralysis {
		statuses = append(statuses, "金縛り")
	}
	if status.Seal {
		statuses = append(statuses, "封印")
	}
	if len(statuses) == 0 {
		return "正常"
	}
	return strings.Join(statuses, " ")
}

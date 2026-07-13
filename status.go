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

func formatPlayerStatus(status StatusAilments) string {
	statuses := make([]string, 0, 7)
	if status.Poison > 0 {
		statuses = append(statuses, fmt.Sprintf("毒(%d)", status.Poison))
	}
	if status.Slow > 0 {
		statuses = append(statuses, fmt.Sprintf("鈍足(%d)", status.Slow))
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

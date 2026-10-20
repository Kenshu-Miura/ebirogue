package main

func mineTrapDamage(currentHealth int) int {
	if currentHealth <= 1 {
		return 0
	}
	return currentHealth / 2
}

func shouldTakeExtraEnemyTurn(slowTurns, moveCount int) bool {
	return slowTurns > 0 && moveCount%2 == 0
}

// drainedPower はパワーを amount だけ下げた値と、実際に下がった量を返す。
// パワーは0未満にならず、すでに0以下なら変えない。
func drainedPower(power, amount int) (newPower, drained int) {
	if power <= 0 || amount <= 0 {
		return power, 0
	}
	drained = min(amount, power)
	return power - drained, drained
}

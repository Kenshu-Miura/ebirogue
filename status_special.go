package main

// 特殊状態の攻撃倍率。通常攻撃の共通ダメージ式へ渡す。
func berserkMultiplier(s StatusAilments) float64 {
	if statusActive(s, StatusBerserk) {
		return 1.5
	}
	return 1
}

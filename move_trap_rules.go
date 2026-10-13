package main

// rollWarpDestination は呼び出し側で通行可能・無人と確認した候補から選ぶ。
func rollWarpDestination(candidates []Coordinate, intn func(int) int) (Coordinate, bool) {
	if len(candidates) == 0 {
		return Coordinate{}, false
	}
	return candidates[intn(len(candidates))], true
}

// 落下は防御力に依存しない小ダメージ（4〜5）。共通ダメージ式を使う。
func rollPitfallDamage(intn func(int) int) int {
	return rollDamage(DamageParams{Attack: 5, MinDamage: 1}, intn).Damage
}

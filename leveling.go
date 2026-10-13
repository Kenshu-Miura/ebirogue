package main

import "math"

// プレイヤーのレベルと成長曲線（docs/agents/balance-level99.md の決定事項）。
// バランス調整はこのファイルの定数と式を変更する。
const (
	maxPlayerLevel      = 99
	initialPlayerHealth = 40
	// 必要経験値の式 round(expCurveScale × (L−1)^expCurvePower × expCurveGrowth^(L−1)) の係数。
	expCurveScale  = 10.0
	expCurvePower  = 1.5
	expCurveGrowth = 1.04
	// 総攻撃力へのレベルの寄与 round(levelAttackScale × √(L−1))。
	levelAttackScale = 3.0
)

// levelHealthGains はレベルアップ1回あたりの最大HPの増加量。
// UpTo 以下のレベルへ上がるときに Gain だけ増える（レベルが高いほど少ない）。
var levelHealthGains = []struct{ UpTo, Gain int }{
	{10, 8},
	{30, 5},
	{60, 3},
	{maxPlayerLevel, 2},
}

// playerLevelExpTable は各レベルに到達するための累計経験値（添字がレベル。0と1は0）。
var playerLevelExpTable = buildPlayerLevelExpTable()

func buildPlayerLevelExpTable() []int {
	table := make([]int, maxPlayerLevel+1)
	for level := 2; level <= maxPlayerLevel; level++ {
		l := float64(level - 1)
		table[level] = int(math.Round(expCurveScale * math.Pow(l, expCurvePower) * math.Pow(expCurveGrowth, l)))
	}
	return table
}

// expForLevel はレベル level に到達するための累計経験値を返す。範囲外は端の値に丸める。
func expForLevel(level int) int {
	return playerLevelExpTable[clampPlayerLevel(level)]
}

// levelForExp は累計経験値 exp で到達できるレベルを返す（1〜maxPlayerLevel）。
func levelForExp(exp int) int {
	level := 1
	for level < maxPlayerLevel && exp >= playerLevelExpTable[level+1] {
		level++
	}
	return level
}

// levelAttackBonus は総攻撃力に加わるレベルの補正（Lv1 で 0、Lv99 で 30）。
func levelAttackBonus(level int) int {
	return int(math.Round(levelAttackScale * math.Sqrt(float64(clampPlayerLevel(level)-1))))
}

// levelHealthBonus は Lv1 からレベル level までに増える最大HPの合計。
func levelHealthBonus(level int) int {
	level = clampPlayerLevel(level)
	total := 0
	for l := 2; l <= level; l++ {
		for _, step := range levelHealthGains {
			if l <= step.UpTo {
				total += step.Gain
				break
			}
		}
	}
	return total
}

func clampPlayerLevel(level int) int {
	if level < 1 {
		return 1
	}
	if level > maxPlayerLevel {
		return maxPlayerLevel
	}
	return level
}

package main

// 物理罠の基礎威力。防御力と共通のダメージ乱数で軽減する。
const (
	woodArrowTrapPower         = 12
	ironArrowTrapPower         = 24
	fallingRockTrapPower       = 30
	largeMineEnemyPower        = 40
	largeMineHealthNumerator   = 3
	largeMineHealthDenominator = 4
)

// 大型地雷は現在HPの3/4（切り上げ）。HP1でも危険が残る。
func largeMineTrapDamage(health int) int {
	if health <= 0 {
		return 0
	}
	return health - health*(largeMineHealthDenominator-largeMineHealthNumerator)/largeMineHealthDenominator
}

package main

// 物理罠の基礎威力。防御力と共通のダメージ乱数で軽減する。
// 初期HP40（Lv1・防御3）で木の矢が最大HPの約1割、鉄の矢が約2.5割、落石が約3割
// （最大でも約4割）になるよう決めた固定値。階層では変えない。根拠は
// docs/agents/balance-level99.md の「残った課題」。
const (
	woodArrowTrapPower         = 6
	ironArrowTrapPower         = 12
	fallingRockTrapPower       = 16
	largeMineEnemyPower        = 40
	largeMineHealthNumerator   = 3
	largeMineHealthDenominator = 4
)

// 毒で下がるパワー。毒は状態異常ではなく即時の効果で、毒矢の罠（5ダメージ＋パワー低下）と
// 腐った食料が使う（毒ヘビ系の毒攻撃も1）。HPは減らさないので毒で倒れることは無い。
// 下がったパワーはパワーアップのカードなど（boostPower）で戻す。
const poisonPowerDrain = 1

// 敵が毒矢の罠を踏んだときの攻撃力低下のターン数（敵にはパワーが無いので攻撃力低下で表す）。
// 弱気の薬・弱体の杖と同じ30ターン。
const poisonArrowEnemyAttackDownTurns = 30

// 大型地雷は現在HPの3/4（切り上げ）。HP1でも危険が残る。
func largeMineTrapDamage(health int) int {
	if health <= 0 {
		return 0
	}
	return health - health*(largeMineHealthDenominator-largeMineHealthNumerator)/largeMineHealthDenominator
}

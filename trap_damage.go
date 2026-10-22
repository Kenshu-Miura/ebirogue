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

// 装備・所持品の罠と腐った食料の数値（wave2-A で決めた仮の値を wave9-A でここへ集めた。値は変えていない）。
//   - 不発率20%: 装備はずし・腐敗・散乱の罠。ダメージの無い罠の標準（毒矢・鈍足・サビなどと同じ20%）に合わせた。
//     罠の出やすさは不発率ではなく階層別の重み（trapBandWeights）で調整する。
//   - 散乱の最大3個: 所持品の上限20の15%。満杯でも拾い直しに数ターンで済み、食料をまとめて失いにくい数。
//   - 腐ったウインナーの満腹度10: ウインナー（50）の1/5。餓死の直前に食べればHP減少を約100ターン止められるが、
//     パワーが下がる（poisonPowerDrain）ので普段は食べたくない量。
const (
	itemTrapFailRate     = 20
	scatterTrapMaxItems  = 3
	rottenSausageSatiety = 10
)

// 大型地雷は現在HPの3/4（切り上げ）。HP1でも危険が残る。
func largeMineTrapDamage(health int) int {
	if health <= 0 {
		return 0
	}
	return health - health*(largeMineHealthDenominator-largeMineHealthNumerator)/largeMineHealthDenominator
}

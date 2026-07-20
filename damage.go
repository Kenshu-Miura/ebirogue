package main

import (
	"math"
	"math/rand"
)

// ダメージ計算のチューニング用定数。
// SFC シレンの「防御力1につき一定割合の軽減」「乱数 7/8〜9/8」を参考に、
// このゲームのバランスへ合わせて調整できるようここへまとめる。
const (
	// 防御力1あたりのダメージ軽減率（乗算）。値を下げるほど防御が強くなる。
	defenseMitigationRate = 15.0 / 16.0
	// 乱数係数は (randomFactorBase + intn(randomFactorSpread)) / randomFactorDenominator。
	// 既定では 112/128(=7/8) 〜 144/128(=9/8) の範囲になる。
	randomFactorBase        = 112
	randomFactorSpread      = 33
	randomFactorDenominator = 128.0
	// プレイヤーの通常攻撃・射撃の会心率（1/N）。0にすると会心なし。
	playerCriticalDenominator = 20
	// 会心の一撃は防御力を無視した上でこの倍率がかかる。
	criticalMultiplier = 1.5
	// 特効武器の倍率。
	slayerDamageMultiplier = 1.5
	// 命中した攻撃の最低保証ダメージ。
	minAttackDamage = 1
)

// damageRandInt はダメージ計算で使う乱数源。テストから差し替えて決定的にできる。
var damageRandInt = rand.Intn

// DamageParams は1回のダメージ計算への入力。
// 新しい補正（属性・状態異常・地形など）は Multiplier へ乗算で合成する。
type DamageParams struct {
	Attack              int     // 攻撃側の総攻撃力
	Defense             int     // 防御側の防御力
	Multiplier          float64 // 特効などの追加倍率。0以下は1.0扱い
	MinDamage           int     // 命中時の最低保証ダメージ
	CriticalDenominator int     // 1/N で会心の一撃。0なら会心なし
}

// DamageResult はダメージ計算の結果。
type DamageResult struct {
	Damage   int
	Critical bool
}

// defenseFactor は防御力による軽減係数（0.0〜1.0）を返す。
func defenseFactor(defense int) float64 {
	if defense <= 0 {
		return 1.0
	}
	return math.Pow(defenseMitigationRate, float64(defense))
}

// damageRandomFactor はダメージの揺らぎ係数（既定 7/8〜9/8）を返す。
func damageRandomFactor(intn func(int) int) float64 {
	return (randomFactorBase + float64(intn(randomFactorSpread))) / randomFactorDenominator
}

// rollDamage は「攻撃力 × 防御係数 × 乱数係数 × 倍率」でダメージを算出する。
// 乱数は会心判定 → 乱数係数の順で消費する。
func rollDamage(params DamageParams, intn func(int) int) DamageResult {
	critical := params.CriticalDenominator > 0 && intn(params.CriticalDenominator) == 0
	value := float64(params.Attack)
	if critical {
		value *= criticalMultiplier
	} else {
		value *= defenseFactor(params.Defense)
	}
	value *= damageRandomFactor(intn)
	if params.Multiplier > 0 {
		value *= params.Multiplier
	}
	damage := int(value)
	if damage < params.MinDamage {
		damage = params.MinDamage
	}
	return DamageResult{Damage: damage, Critical: critical}
}

// playerAttackTotal は装備込み攻撃力にちからとレベルの補正を加えた総攻撃力を返す。
func playerAttackTotal(attackPower, power, level int) int {
	return attackPower + power + level
}

// rollPlayerAttackDamage はプレイヤーの通常攻撃・射撃のダメージを算出する。
func rollPlayerAttackDamage(attack, defense int, multiplier float64, intn func(int) int) DamageResult {
	return rollDamage(DamageParams{
		Attack:              attack,
		Defense:             defense,
		Multiplier:          multiplier,
		MinDamage:           minAttackDamage,
		CriticalDenominator: playerCriticalDenominator,
	}, intn)
}

// rollEnemyAttackDamage は敵の通常攻撃・遠距離攻撃のダメージを算出する。会心なし。
func rollEnemyAttackDamage(attack, defense int, intn func(int) int) int {
	return rollDamage(DamageParams{
		Attack:    attack,
		Defense:   defense,
		MinDamage: minAttackDamage,
	}, intn).Damage
}

// rollThrownItemDamage は武器・矢以外の投擲アイテムの固定ダメージ（1〜3）。
func rollThrownItemDamage(intn func(int) int) int {
	return intn(3) + 1
}

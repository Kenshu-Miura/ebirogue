package main

import "math/rand"

// EquipmentAbilityID identifies an equipment ability independently of its name.
type EquipmentAbilityID string

const (
	// AbilitySatietyConservation slows the player's satiety consumption.
	AbilitySatietyConservation EquipmentAbilityID = "satiety_conservation"
	// 特効能力は、対応する系統の敵への通常攻撃ダメージを増加させる。
	AbilityDragonSlayer  EquipmentAbilityID = "dragon_slayer"
	AbilityGhostSlayer   EquipmentAbilityID = "ghost_slayer"
	AbilityOneEyeSlayer  EquipmentAbilityID = "one_eye_slayer"
	AbilityDrainerSlayer EquipmentAbilityID = "drainer_slayer"
	// 攻撃方法を変える武器能力。
	AbilityThreeWayAttack EquipmentAbilityID = "three_way_attack"
	AbilitySureHit        EquipmentAbilityID = "sure_hit"
	AbilityDigWall        EquipmentAbilityID = "dig_wall"
	AbilityDisposable     EquipmentAbilityID = "disposable"
	// 盾の特殊防御能力。
	AbilityExplosionResistance EquipmentAbilityID = "explosion_resistance"
	AbilityFireResistance      EquipmentAbilityID = "fire_resistance"
	AbilityMagicResistance     EquipmentAbilityID = "magic_resistance"
	AbilityTheftResistance     EquipmentAbilityID = "theft_resistance"
	AbilityStatusResistance    EquipmentAbilityID = "status_resistance"
	AbilityEvasion             EquipmentAbilityID = "evasion"
	AbilityReflection          EquipmentAbilityID = "reflection"
	AbilityCounter             EquipmentAbilityID = "counter"
	AbilitySatietyHunger       EquipmentAbilityID = "satiety_hunger"
	AbilityDisposableArmor     EquipmentAbilityID = "disposable_armor"
)

// EnemyTrait は、特効などの判定に使う敵の分類タグを表す。
type EnemyTrait string

const (
	EnemyTraitDragon  EnemyTrait = "dragon"
	EnemyTraitGhost   EnemyTrait = "ghost"
	EnemyTraitOneEye  EnemyTrait = "one_eye"
	EnemyTraitDrainer EnemyTrait = "drainer"
)

const (
	baseSatietyLossInterval     = 10
	hungrySatietyLossInterval   = 5
	shieldResistanceNumerator   = 1
	shieldResistanceDenominator = 2
	shieldEvasionPercent        = 25
	shieldCounterNumerator      = 1
	shieldCounterDenominator    = 2
	statusResistancePercent     = 50
)

const basePlayerAttackAccuracy = 90

// attackHitRandInt は命中判定の乱数源。テストでは差し替えて決定的にする。
var attackHitRandInt = rand.Intn

func hasEquipmentAbility(abilities []EquipmentAbilityID, target EquipmentAbilityID) bool {
	for _, ability := range abilities {
		if ability == target {
			return true
		}
	}
	return false
}

func hasEnemyTrait(traits []EnemyTrait, target EnemyTrait) bool {
	for _, trait := range traits {
		if trait == target {
			return true
		}
	}
	return false
}

// slayerMultiplier は、武器能力と敵の分類が一致したときのダメージ倍率を返す。
// 複数の特効条件が一致しても倍率は重複させず、rollDamage の Multiplier へ乗算で渡す。
func slayerMultiplier(abilities []EquipmentAbilityID, traits []EnemyTrait) (float64, bool) {
	matched := (hasEquipmentAbility(abilities, AbilityDragonSlayer) && hasEnemyTrait(traits, EnemyTraitDragon)) ||
		(hasEquipmentAbility(abilities, AbilityGhostSlayer) && hasEnemyTrait(traits, EnemyTraitGhost)) ||
		(hasEquipmentAbility(abilities, AbilityOneEyeSlayer) && hasEnemyTrait(traits, EnemyTraitOneEye)) ||
		(hasEquipmentAbility(abilities, AbilityDrainerSlayer) && hasEnemyTrait(traits, EnemyTraitDrainer))
	if !matched {
		return 1.0, false
	}
	return slayerDamageMultiplier, true
}

func satietyLossInterval(armorAbilities []EquipmentAbilityID) int {
	if hasEquipmentAbility(armorAbilities, AbilitySatietyHunger) {
		return hungrySatietyLossInterval
	}
	if hasEquipmentAbility(armorAbilities, AbilitySatietyConservation) {
		return baseSatietyLossInterval * 2
	}
	return baseSatietyLossInterval
}

// EnemyDamageKind は盾の属性耐性を判定するための敵攻撃属性を表す。
type EnemyDamageKind int

const (
	EnemyDamageNormal EnemyDamageKind = iota
	EnemyDamageExplosion
	EnemyDamageFire
	EnemyDamageMagic
)

// EnemyAttackDelivery は回避・反射・カウンターの対象になる攻撃方法を表す。
type EnemyAttackDelivery int

const (
	EnemyAttackMelee EnemyAttackDelivery = iota
	EnemyAttackRanged
	EnemyAttackEnvironment
)

// ShieldDefenseResult は盾能力を適用した1回の被弾結果。
type ShieldDefenseResult struct {
	Damage        int
	Evaded        bool
	Resisted      bool
	Reflected     bool
	CounterDamage int
	WearArmor     bool
}

func hasDamageResistance(abilities []EquipmentAbilityID, kind EnemyDamageKind) bool {
	switch kind {
	case EnemyDamageExplosion:
		return hasEquipmentAbility(abilities, AbilityExplosionResistance)
	case EnemyDamageFire:
		return hasEquipmentAbility(abilities, AbilityFireResistance)
	case EnemyDamageMagic:
		return hasEquipmentAbility(abilities, AbilityMagicResistance)
	default:
		return false
	}
}

func reducedShieldDamage(damage int) int {
	if damage <= 0 {
		return 0
	}
	reduced := damage * shieldResistanceNumerator / shieldResistanceDenominator
	return max(1, reduced)
}

// resolveShieldDefense は回避→反射→属性軽減→カウンターの順で盾能力を解決する。
// 爆発は範囲攻撃のため反射できず、環境ダメージは回避・反射・カウンターの対象外とする。
func resolveShieldDefense(damage int, kind EnemyDamageKind, delivery EnemyAttackDelivery, abilities []EquipmentAbilityID, intn func(int) int) ShieldDefenseResult {
	result := ShieldDefenseResult{Damage: max(0, damage)}
	if result.Damage == 0 {
		return result
	}

	if delivery != EnemyAttackEnvironment && hasEquipmentAbility(abilities, AbilityEvasion) && intn(100) < shieldEvasionPercent {
		result.Damage = 0
		result.Evaded = true
		return result
	}
	if delivery == EnemyAttackRanged && kind != EnemyDamageExplosion && hasEquipmentAbility(abilities, AbilityReflection) {
		result.Damage = 0
		result.Reflected = true
		result.CounterDamage = damage
		return result
	}
	if hasDamageResistance(abilities, kind) {
		result.Damage = reducedShieldDamage(result.Damage)
		result.Resisted = true
	}
	if delivery == EnemyAttackMelee && hasEquipmentAbility(abilities, AbilityCounter) {
		result.CounterDamage = max(1, result.Damage*shieldCounterNumerator/shieldCounterDenominator)
	}
	result.WearArmor = result.Damage > 0 && hasEquipmentAbility(abilities, AbilityDisposableArmor)
	return result
}

func blocksTheft(abilities []EquipmentAbilityID) bool {
	return hasEquipmentAbility(abilities, AbilityTheftResistance)
}

// rollStatusResistance は状態異常耐性盾の50%無効化を判定する。
func rollStatusResistance(abilities []EquipmentAbilityID, intn func(int) int) bool {
	return hasEquipmentAbility(abilities, AbilityStatusResistance) && intn(100) < statusResistancePercent
}

func nextDisposableArmorSharpness(sharpness int, abilities []EquipmentAbilityID) (int, bool) {
	if !hasEquipmentAbility(abilities, AbilityDisposableArmor) {
		return sharpness, false
	}
	return sharpness - 1, true
}

func shouldReduceSatiety(moveCount int, armorAbilities []EquipmentAbilityID) bool {
	return moveCount > 0 && moveCount%satietyLossInterval(armorAbilities) == 0
}

// playerAttackDeltas は武器能力に応じた攻撃対象方向を返す。
// 三方向攻撃は入力方向を正面として、左右45度の隣接マスも対象にする。
func playerAttackDeltas(dx, dy int, abilities []EquipmentAbilityID) []Coordinate {
	front := Coordinate{X: sign(dx), Y: sign(dy)}
	if !hasEquipmentAbility(abilities, AbilityThreeWayAttack) {
		return []Coordinate{front}
	}
	left := Coordinate{X: sign(dx + dy), Y: sign(dy - dx)}
	right := Coordinate{X: sign(dx - dy), Y: sign(dy + dx)}
	return []Coordinate{front, left, right}
}

// rollPlayerAttackHit は通常武器の90%命中と必中能力の100%命中を判定する。
func rollPlayerAttackHit(abilities []EquipmentAbilityID, intn func(int) int) bool {
	if hasEquipmentAbility(abilities, AbilitySureHit) {
		return true
	}
	return intn(100) < basePlayerAttackAccuracy
}

// nextDisposableWeaponAttackPower は攻撃後の使い捨て武器の基礎攻撃力を返す。
func nextDisposableWeaponAttackPower(attackPower int, abilities []EquipmentAbilityID) (int, bool) {
	if !hasEquipmentAbility(abilities, AbilityDisposable) || attackPower <= 0 {
		return attackPower, false
	}
	return attackPower - 1, true
}

// isDiggableTile はつるはしで掘れる未通行タイルかを返す。マップ外への脱出を防ぐため外周は掘れない。
func isDiggableTile(tile Tile, x, y, width, height int) bool {
	if x <= 0 || y <= 0 || x >= width-1 || y >= height-1 {
		return false
	}
	return tile.Blocked && (tile.Type == "wall" || tile.Type == "other")
}

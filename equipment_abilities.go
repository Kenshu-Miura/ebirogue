package main

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
)

// EnemyTrait は、特効などの判定に使う敵の分類タグを表す。
type EnemyTrait string

const (
	EnemyTraitDragon  EnemyTrait = "dragon"
	EnemyTraitGhost   EnemyTrait = "ghost"
	EnemyTraitOneEye  EnemyTrait = "one_eye"
	EnemyTraitDrainer EnemyTrait = "drainer"
)

const baseSatietyLossInterval = 10

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
	if hasEquipmentAbility(armorAbilities, AbilitySatietyConservation) {
		return baseSatietyLossInterval * 2
	}
	return baseSatietyLossInterval
}

func shouldReduceSatiety(moveCount int, armorAbilities []EquipmentAbilityID) bool {
	return moveCount > 0 && moveCount%satietyLossInterval(armorAbilities) == 0
}

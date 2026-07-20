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

// applySlayerBonus は、武器能力と敵の分類が一致したときだけダメージを1.5倍（端数切り上げ）にする。
// 複数の特効条件が一致しても倍率は重複させない。
func applySlayerBonus(damage int, abilities []EquipmentAbilityID, traits []EnemyTrait) (int, bool) {
	if damage <= 0 {
		return damage, false
	}

	matched := (hasEquipmentAbility(abilities, AbilityDragonSlayer) && hasEnemyTrait(traits, EnemyTraitDragon)) ||
		(hasEquipmentAbility(abilities, AbilityGhostSlayer) && hasEnemyTrait(traits, EnemyTraitGhost)) ||
		(hasEquipmentAbility(abilities, AbilityOneEyeSlayer) && hasEnemyTrait(traits, EnemyTraitOneEye)) ||
		(hasEquipmentAbility(abilities, AbilityDrainerSlayer) && hasEnemyTrait(traits, EnemyTraitDrainer))
	if !matched {
		return damage, false
	}
	return damage + (damage+1)/2, true
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

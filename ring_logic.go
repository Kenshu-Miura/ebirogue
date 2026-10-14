package main

// accessorySatietyBonus は指輪1個の最大満腹度補正。能力の定数は通常ビルド側に置く。
func accessorySatietyBonus(abilities []EquipmentAbilityID) int {
	bonus := 0
	if hasEquipmentAbility(abilities, "stomach_expand") {
		bonus += 25
	}
	if hasEquipmentAbility(abilities, "stomach_shrink") {
		bonus -= 25
	}
	return bonus
}

func ringSpawnInterval(base, stacks int) int {
	if stacks <= 0 {
		return base
	}
	return max(5, base/(1+stacks))
}

func ringChance(roll, perStack, stacks int) bool { return stacks > 0 && roll < perStack*stacks }

func ringMoneyDrop(cash int) int {
	if cash <= 0 {
		return 0
	}
	return max(1, cash/50)
}

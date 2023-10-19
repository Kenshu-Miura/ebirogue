package main

func createEnemy(x, y int) Enemy {
	var enemyType, enemyName, enemyChar string
	var enemyAP, enemyDP int
	var enemyHealth, enemyMaxHealth, enemyExperiencePoints int
	var enemyDirection int
	randomValue := localRand.Intn(2) // Store the random value to ensure it's only generated once
	switch randomValue {
	case 0:
		enemyType = "Shrimp"
		enemyName = "海老"
		enemyChar = "E"
		enemyAP = 4
		enemyDP = 2
		enemyHealth = 30
		enemyMaxHealth = 30
		enemyExperiencePoints = 5
		enemyDirection = Down
	case 1:
		enemyType = "Snake"
		enemyName = "蛇"
		enemyChar = "S"
		enemyAP = 7
		enemyDP = 1
		enemyHealth = 50
		enemyMaxHealth = 50
		enemyExperiencePoints = 10
		enemyDirection = Down

	default:
		enemyType = "Shrimp"
		enemyName = "海老"
		enemyChar = "E"
		enemyAP = 4
		enemyDP = 2
		enemyHealth = 30
		enemyMaxHealth = 30
		enemyExperiencePoints = 5
		enemyDirection = Down
	}

	return Enemy{
		Entity:           Entity{X: x, Y: y, Char: rune(enemyChar[0])},
		Health:           enemyHealth,
		MaxHealth:        enemyMaxHealth,
		Name:             enemyName,
		AttackPower:      enemyAP,
		DefensePower:     enemyDP,
		Type:             enemyType,
		ExperiencePoints: enemyExperiencePoints,
		Direction:        enemyDirection,
		PlayerDiscovered: false,
	}
}

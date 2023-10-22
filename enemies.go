package main

type Enemy struct {
	Entity               // Enemy inherits fields from Entity
	ID                   int
	dx, dy               int // 敵の移動方向
	Name                 string
	Health               int
	MaxHealth            int
	AttackPower          int       // Attack power
	DefensePower         int       // Defense power
	Type                 string    // Type of enemy (e.g., "orc", "goblin", "slime", etc.)
	ExperiencePoints     int       // Experience points enemy holds
	PlayerDiscovered     bool      // Whether the enemy has discovered the player
	Direction            int       // Uninitialized: uninitialized, Up: Up, Down: Down, Left: Left, Right: Right, UpRight: UpRight, DownRight: DownRight, UpLeft: UpLeft, DownLeft: DownLeft
	AnimationProgressInt int       // アニメーション進行度
	Animating            bool      // アニメーション中かどうか
	AttackDirection      Direction // 敵の攻撃方向
	AttackTimer          float64   // 敵の攻撃アニメーションを制御するタイマー (0.0 から 0.5 まで)
	OffsetX, OffsetY     int       // アニメーションのオフセット
}

func createEnemy(x, y int) Enemy {
	var enemyType, enemyName, enemyChar string
	var enemyAP, enemyDP, enemyID int
	var enemyHealth, enemyMaxHealth, enemyExperiencePoints int
	var enemyDirection int
	randomValue := localRand.Intn(2) // Store the random value to ensure it's only generated once
	switch randomValue {
	case 0:
		enemyID = 0
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
		enemyID = 1
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
		enemyID = 0
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
		ID:               enemyID,
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

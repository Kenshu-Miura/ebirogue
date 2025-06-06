//go:build !test
// +build !test

package main

import (
	"fmt"
	"math"
)

type SpecialAttackFunc func(e *Enemy, g *Game)

type Enemy struct {
	Entity                   // Enemy inherits fields from Entity
	ID                       int
	dx, dy                   int // 敵の移動方向
	Name                     string
	Health                   int
	MaxHealth                int
	AttackPower              int               // Attack power
	DefensePower             int               // Defense power
	Type                     string            // Type of enemy (e.g., "orc", "goblin", "slime", etc.)
	ExperiencePoints         int               // Experience points enemy holds
	PlayerDiscovered         bool              // Whether the enemy has discovered the player
	Direction                Direction         // Uninitialized: uninitialized, Up: Up, Down: Down, Left: Left, Right: Right, UpRight: UpRight, DownRight: DownRight, UpLeft: UpLeft, DownLeft: DownLeft
	AnimationProgressInt     int               // アニメーション進行度
	Animating                bool              // アニメーション中かどうか
	AttackDirection          Direction         // 敵の攻撃方向
	AttackTimer              float64           // 敵の攻撃アニメーションを制御するタイマー (0.0 から 0.5 まで)
	OffsetX, OffsetY         int               // アニメーションのオフセット
	SpecialAttack            SpecialAttackFunc // 敵の特殊攻撃処理
	SpecialAttackProbability float64           // 敵が特殊攻撃を使ってくる確率 (0.0 to 1.0)
	ShowOnMiniMap            bool
}

func (g *Game) updateEnemyVisibility() {
	playerX, playerY := g.state.Player.GetPosition()
	for i := range g.state.Enemies {
		enemy := &g.state.Enemies[i] // get the address of the enemy instance
		enemyX, enemyY := enemy.GetPosition()

		// Check if the player and enemy are in the same room
		inSameRoom := isSameRoom(playerX, playerY, enemyX, enemyY, g.rooms)

		// Check if the player and enemy are adjacent
		adjacent := (math.Abs(float64(playerX-enemyX)) <= 1 && math.Abs(float64(playerY-enemyY)) <= 1)

		if inSameRoom || adjacent || enemy.PlayerDiscovered {
			g.miniMapDirty = true
			enemy.SetShowOnMiniMap(true)
		} else {
			enemy.SetShowOnMiniMap(false)
		}
	}
}

func createEnemy(x, y int) Enemy {
	var enemyType, enemyName, enemyChar string
	var enemyAP, enemyDP, enemyID int
	var enemyHealth, enemyMaxHealth, enemyExperiencePoints int
	var enemyDirection Direction
	var specialAttack SpecialAttackFunc
	var specialAttackProbability float64
	randomValue := localRand.Intn(2) // Store the random value to ensure it's only generated once and correct the range to 2 for two cases
	switch randomValue {
	case 0:
		enemyID = 0
		enemyType = "Shrimp"
		enemyName = "エビ"
		enemyChar = "E"
		enemyAP = 4
		enemyDP = 2
		enemyHealth = 20
		enemyMaxHealth = 20
		enemyExperiencePoints = 5
		enemyDirection = Down
		specialAttack = nil // No special attack for Shrimp
		specialAttackProbability = 0.0
	case 1:
		enemyID = 1
		enemyType = "Snake"
		enemyName = "毒ヘビ"
		enemyChar = "S"
		enemyAP = 7
		enemyDP = 1
		enemyHealth = 30
		enemyMaxHealth = 30
		enemyExperiencePoints = 10
		enemyDirection = Down
		specialAttack = func(e *Enemy, g *Game) {
			if g.state.Player.Power > 0 {
				action := Action{
					Duration: 0.5,
					Message:  fmt.Sprintf("%sの毒攻撃。海老さんのパワーが1下がった。", e.Name),
					Execute: func(g *Game) {
						g.state.Player.Power--

					},
				}
				g.Enqueue(action)
			}
		}
		specialAttackProbability = 0.3 // Assuming a 100% chance to use special attack for simplicity, adjust as necessary
	}

	return Enemy{
		Entity:                   Entity{X: x, Y: y, Char: rune(enemyChar[0])},
		ID:                       enemyID,
		Health:                   enemyHealth,
		MaxHealth:                enemyMaxHealth,
		Name:                     enemyName,
		AttackPower:              enemyAP,
		DefensePower:             enemyDP,
		Type:                     enemyType,
		ExperiencePoints:         enemyExperiencePoints,
		Direction:                enemyDirection,
		PlayerDiscovered:         false,
		SpecialAttack:            specialAttack,
		SpecialAttackProbability: specialAttackProbability,
	}
}

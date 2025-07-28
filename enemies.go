//go:build !test

package main

import (
	"fmt"
	"math"
	"math/rand"
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
	StatusAilments           StatusAilments    // 状態異常
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

		// プレイヤーが盲目状態の場合、敵を非表示にする
		if g.state.Player.StatusAilments.Blind > 0 {
			enemy.SetShowOnMiniMap(false)
			g.miniMapDirty = true
		} else if inSameRoom || adjacent || enemy.PlayerDiscovered {
			g.miniMapDirty = true
			enemy.SetShowOnMiniMap(true)
		} else {
			enemy.SetShowOnMiniMap(false)
		}
	}
}

// 統一されたモンスター定義テーブル
type MonsterDefinition struct {
	ID                       int
	Type                     string
	Name                     string
	Char                     rune
	Health                   int
	MaxHealth                int
	AttackPower              int
	DefensePower             int
	ExperiencePoints         int
	SpecialAttack            SpecialAttackFunc
	SpecialAttackProbability float64
}

// モンスター定義テーブル
var MonsterDefinitions = map[int]MonsterDefinition{
	0: {
		ID:                       0,
		Type:                     "Shrimp",
		Name:                     "エビ",
		Char:                     'E',
		Health:                   20,
		MaxHealth:                20,
		AttackPower:              4,
		DefensePower:             2,
		ExperiencePoints:         5,
		SpecialAttack:            nil,
		SpecialAttackProbability: 0.0,
	},
	1: {
		ID:                       1,
		Type:                     "Snake",
		Name:                     "毒ヘビ",
		Char:                     'S',
		Health:                   30,
		MaxHealth:                30,
		AttackPower:              7,
		DefensePower:             1,
		ExperiencePoints:         10,
		SpecialAttack: func(e *Enemy, g *Game) {
			var message string
			if g.state.Player.Power > 0 {
				message = fmt.Sprintf("%sの毒攻撃。海老さんのパワーが1下がった。", func() string {
					if g.state.Player.StatusAilments.Blind > 0 {
						return "何者"
					}
					return e.Name
				}())
			} else {
				message = fmt.Sprintf("%sの毒攻撃。しかし海老さんのパワーはこれ以上下がらない", func() string {
					if g.state.Player.StatusAilments.Blind > 0 {
						return "何者"
					}
					return e.Name
				}())
			}
			action := Action{
				Duration: 0.5,
				Message:  message,
				Execute: func(g *Game) {
					if g.state.Player.Power > 0 {
						g.state.Player.Power--
					}
				},
			}
			g.Enqueue(action)
		},
		SpecialAttackProbability: 0.3,
	},
}

// 統一されたモンスター生成関数
func CreateEnemyByID(id, x, y int) Enemy {
	def, exists := MonsterDefinitions[id]
	if !exists {
		// デフォルトはエビ
		def = MonsterDefinitions[0]
	}

	return Enemy{
		Entity:                   Entity{X: x, Y: y, Char: def.Char},
		ID:                       def.ID,
		Health:                   def.Health,
		MaxHealth:                def.MaxHealth,
		Name:                     def.Name,
		AttackPower:              def.AttackPower,
		DefensePower:             def.DefensePower,
		Type:                     def.Type,
		ExperiencePoints:         def.ExperiencePoints,
		Direction:                Down,
		PlayerDiscovered:         false,
		SpecialAttack:            def.SpecialAttack,
		SpecialAttackProbability: def.SpecialAttackProbability,
		ShowOnMiniMap:            true,
	}
}

func createEnemy(x, y int) Enemy {
	randomValue := rand.Intn(2)
	return CreateEnemyByID(randomValue, x, y)
}

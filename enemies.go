//go:build !test

package main

import (
	"fmt"
	"math"
	"math/rand"
)

type SpecialAttackFunc func(e *Enemy, g *Game)

type RangedAttackKind int

const (
	RangedAttackNone RangedAttackKind = iota
	RangedAttackArrow
	RangedAttackRock
	RangedAttackExplosion
)

type RangedAttackDefinition struct {
	Kind        RangedAttackKind
	MinRange    int
	MaxRange    int
	AttackPower int
	BlastRadius int
}

type SpecialMovement int

const (
	SpecialMovementNone SpecialMovement = iota
	SpecialMovementWallPass
	SpecialMovementDoubleSpeed
	SpecialMovementWarp
	SpecialMovementSwap
	SpecialMovementCreateTrap
)

type EnemyDisguise int

const (
	EnemyDisguiseNone EnemyDisguise = iota
	EnemyDisguiseItem
	EnemyDisguiseStairs
)

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
	RangedAttack             RangedAttackDefinition
	SpecialMovement          SpecialMovement
	Disguise                 EnemyDisguise
	Revealed                 bool
	ShowOnMiniMap            bool
	StatusAilments           StatusAilments // 状態異常
	HeldItem                 Item           // 盗賊系の敵が持ち去ったアイテム
	Fleeing                  bool           // 盗品を持ってプレイヤーから逃走中かどうか
}

func isEnemyDisguised(enemy Enemy) bool {
	return enemy.Disguise != EnemyDisguiseNone && !enemy.Revealed && !enemy.StatusAilments.Seal
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
		if isEnemyDisguised(*enemy) && (inSameRoom || adjacent) {
			enemy.PlayerDiscovered = true
		}

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
	RangedAttack             RangedAttackDefinition
	SpecialMovement          SpecialMovement
	Disguise                 EnemyDisguise
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
		ID:               1,
		Type:             "Snake",
		Name:             "毒ヘビ",
		Char:             'S',
		Health:           30,
		MaxHealth:        30,
		AttackPower:      7,
		DefensePower:     1,
		ExperiencePoints: 10,
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
	2: {
		ID:                       2,
		Type:                     "Mamuru",
		Name:                     "マムル",
		Char:                     'M',
		Health:                   8,
		MaxHealth:                8,
		AttackPower:              3,
		DefensePower:             1,
		ExperiencePoints:         3,
		SpecialAttack:            nil,
		SpecialAttackProbability: 0,
	},
	3: {
		ID:               3,
		Type:             "Honey",
		Name:             "くねくねハニー",
		Char:             'H',
		Health:           24,
		MaxHealth:        24,
		AttackPower:      5,
		DefensePower:     3,
		ExperiencePoints: 12,
		SpecialAttack: func(e *Enemy, g *Game) {
			enemyName := e.Name
			if g.state.Player.StatusAilments.Blind > 0 {
				enemyName = "何者"
			}
			g.Enqueue(Action{
				Duration: 0.5,
				Message:  fmt.Sprintf("%sのくねくね踊り。海老さんは鈍足になった。", enemyName),
				Execute: func(g *Game) {
					if g.state.Player.StatusAilments.Slow < 8 {
						g.state.Player.StatusAilments.Slow = 8
					}
				},
			})
		},
		SpecialAttackProbability: 0.25,
	},
	4: {
		ID:               4,
		Type:             "Harisenbow",
		Name:             "ハリセンボウ",
		Char:             'B',
		Health:           18,
		MaxHealth:        18,
		AttackPower:      6,
		DefensePower:     2,
		ExperiencePoints: 8,
		RangedAttack: RangedAttackDefinition{
			Kind:        RangedAttackArrow,
			MinRange:    2,
			MaxRange:    8,
			AttackPower: 9,
		},
	},
	5: {
		ID:               5,
		Type:             "Ishigani",
		Name:             "イシガニ",
		Char:             'C',
		Health:           32,
		MaxHealth:        32,
		AttackPower:      7,
		DefensePower:     5,
		ExperiencePoints: 15,
		RangedAttack: RangedAttackDefinition{
			Kind:        RangedAttackRock,
			MinRange:    2,
			MaxRange:    5,
			AttackPower: 12,
		},
	},
	6: {
		ID:               6,
		Type:             "BombUrchin",
		Name:             "バクダンウニ",
		Char:             'U',
		Health:           45,
		MaxHealth:        45,
		AttackPower:      10,
		DefensePower:     6,
		ExperiencePoints: 25,
		RangedAttack: RangedAttackDefinition{
			Kind:        RangedAttackExplosion,
			MinRange:    2,
			MaxRange:    6,
			AttackPower: 16,
			BlastRadius: 1,
		},
	},
	7: {
		ID:                       7,
		Type:                     "ThiefHermitCrab",
		Name:                     "コソドロヤドカリ",
		Char:                     'T',
		Health:                   26,
		MaxHealth:                26,
		AttackPower:              5,
		DefensePower:             4,
		ExperiencePoints:         14,
		SpecialAttack:            func(e *Enemy, g *Game) { enqueueStealAttack(e, g, rand.Intn) },
		SpecialAttackProbability: 0.45,
	},
	8: {
		ID:                       8,
		Type:                     "NigiriShrimp",
		Name:                     "にぎりエビ",
		Char:                     'N',
		Health:                   34,
		MaxHealth:                34,
		AttackPower:              7,
		DefensePower:             4,
		ExperiencePoints:         18,
		SpecialAttack:            func(e *Enemy, g *Game) { enqueueFoodTransformationAttack(e, g, rand.Intn) },
		SpecialAttackProbability: 0.35,
	},
	9: {
		ID:                       9,
		Type:                     "CurseCrab",
		Name:                     "ノロイガニ",
		Char:                     'C',
		Health:                   38,
		MaxHealth:                38,
		AttackPower:              8,
		DefensePower:             6,
		ExperiencePoints:         22,
		SpecialAttack:            func(e *Enemy, g *Game) { enqueueCurseAttack(e, g, rand.Intn) },
		SpecialAttackProbability: 0.35,
	},
	10: {
		ID:                       10,
		Type:                     "PuppeteerJellyfish",
		Name:                     "あやつりクラゲ",
		Char:                     'J',
		Health:                   42,
		MaxHealth:                42,
		AttackPower:              9,
		DefensePower:             5,
		ExperiencePoints:         28,
		SpecialAttack:            func(e *Enemy, g *Game) { enqueueManipulationAttack(e, g, rand.Intn) },
		SpecialAttackProbability: 0.3,
	},
	11: {
		ID:               11,
		Type:             "GhostShrimp",
		Name:             "ユウレイエビ",
		Char:             'G',
		Health:           28,
		MaxHealth:        28,
		AttackPower:      8,
		DefensePower:     2,
		ExperiencePoints: 18,
		SpecialMovement:  SpecialMovementWallPass,
	},
	12: {
		ID:               12,
		Type:             "MantisShrimp",
		Name:             "ハヤテシャコ",
		Char:             'F',
		Health:           22,
		MaxHealth:        22,
		AttackPower:      6,
		DefensePower:     3,
		ExperiencePoints: 20,
		SpecialMovement:  SpecialMovementDoubleSpeed,
	},
	13: {
		ID:               13,
		Type:             "WarpJellyfish",
		Name:             "ワープクラゲ",
		Char:             'W',
		Health:           30,
		MaxHealth:        30,
		AttackPower:      8,
		DefensePower:     4,
		ExperiencePoints: 24,
		SpecialMovement:  SpecialMovementWarp,
	},
	14: {
		ID:               14,
		Type:             "SwapOctopus",
		Name:             "イレカエダコ",
		Char:             'O',
		Health:           36,
		MaxHealth:        36,
		AttackPower:      9,
		DefensePower:     5,
		ExperiencePoints: 30,
		SpecialMovement:  SpecialMovementSwap,
	},
	15: {
		ID:               15,
		Type:             "TrapHermitCrab",
		Name:             "ワナシヤドカリ",
		Char:             'R',
		Health:           40,
		MaxHealth:        40,
		AttackPower:      8,
		DefensePower:     7,
		ExperiencePoints: 32,
		SpecialMovement:  SpecialMovementCreateTrap,
	},
	16: {
		ID:               16,
		Type:             "MimicClam",
		Name:             "化け貝",
		Char:             'Q',
		Health:           34,
		MaxHealth:        34,
		AttackPower:      10,
		DefensePower:     6,
		ExperiencePoints: 28,
		Disguise:         EnemyDisguiseItem,
	},
	17: {
		ID:               17,
		Type:             "MimicClam",
		Name:             "化け貝",
		Char:             'Q',
		Health:           42,
		MaxHealth:        42,
		AttackPower:      12,
		DefensePower:     7,
		ExperiencePoints: 36,
		Disguise:         EnemyDisguiseStairs,
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
		RangedAttack:             def.RangedAttack,
		SpecialMovement:          def.SpecialMovement,
		Disguise:                 def.Disguise,
		Revealed:                 def.Disguise == EnemyDisguiseNone,
		ShowOnMiniMap:            true,
	}
}

func createEnemy(x, y int) Enemy {
	randomValue := rand.Intn(len(MonsterDefinitions))
	return CreateEnemyByID(randomValue, x, y)
}

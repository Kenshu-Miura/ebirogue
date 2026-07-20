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
	RangedAttackFire
	RangedAttackMagic
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
	Traits                   []EnemyTrait
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
	Traits                   []EnemyTrait
}

// upperMonsterDefinition は基本種の固有能力を引き継ぎ、上位種の能力値と表示情報を設定する。
func upperMonsterDefinition(base MonsterDefinition, id int, monsterType, name string, health, attack, defense, experience int) MonsterDefinition {
	base.ID = id
	base.Type = monsterType
	base.Name = name
	base.Health = health
	base.MaxHealth = health
	base.AttackPower = attack
	base.DefensePower = defense
	base.ExperiencePoints = experience
	return base
}

// baseMonsterDefinitions は各系統の基本種を定義する。
var baseMonsterDefinitions = map[int]MonsterDefinition{
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
		Traits:           []EnemyTrait{EnemyTraitDrainer},
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
		Traits:           []EnemyTrait{EnemyTraitOneEye, EnemyTraitDrainer},
		SpecialAttack: func(e *Enemy, g *Game) {
			enemyName := e.Name
			if g.state.Player.StatusAilments.Blind > 0 {
				enemyName = "何者"
			}
			resisted := g.playerResistsStatus()
			message := fmt.Sprintf("%sのくねくね踊り。海老さんは鈍足になった。", enemyName)
			if resisted {
				message = fmt.Sprintf("%sのくねくね踊り。しかし%sが鈍足を防いだ。", enemyName, g.state.Player.EquippedArmor.Name)
			}
			g.Enqueue(Action{
				Duration: 0.5,
				Message:  message,
				Execute: func(g *Game) {
					if !resisted && g.state.Player.StatusAilments.Slow < 8 {
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
		Traits:           []EnemyTrait{EnemyTraitGhost},
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
	36: {
		ID:               36,
		Type:             "SeaDragon",
		Name:             "海竜",
		Char:             'D',
		Health:           68,
		MaxHealth:        68,
		AttackPower:      15,
		DefensePower:     9,
		ExperiencePoints: 55,
		Traits:           []EnemyTrait{EnemyTraitDragon},
	},
	38: {
		ID:               38,
		Type:             "FlameSquid",
		Name:             "火吹きイカ",
		Char:             'I',
		Health:           48,
		MaxHealth:        48,
		AttackPower:      10,
		DefensePower:     5,
		ExperiencePoints: 30,
		RangedAttack: RangedAttackDefinition{
			Kind:        RangedAttackFire,
			MinRange:    2,
			MaxRange:    5,
			AttackPower: 14,
		},
	},
	39: {
		ID:               39,
		Type:             "MagicJellyfish",
		Name:             "マホウクラゲ",
		Char:             'J',
		Health:           52,
		MaxHealth:        52,
		AttackPower:      11,
		DefensePower:     6,
		ExperiencePoints: 36,
		RangedAttack: RangedAttackDefinition{
			Kind:        RangedAttackMagic,
			MinRange:    2,
			MaxRange:    6,
			AttackPower: 15,
		},
	},
}

// buildMonsterDefinitions は基本種へ固有能力を継承した上位種を加える。
func buildMonsterDefinitions() map[int]MonsterDefinition {
	definitions := make(map[int]MonsterDefinition, 42)
	for id, definition := range baseMonsterDefinitions {
		definitions[id] = definition
	}

	definitions[18] = upperMonsterDefinition(definitions[0], 18, "GreatShrimp", "大エビ", 45, 11, 6, 30)
	definitions[19] = upperMonsterDefinition(definitions[2], 19, "CaveMamuru", "あなぐらマムル", 24, 7, 3, 12)
	definitions[20] = upperMonsterDefinition(definitions[1], 20, "VenomSnake", "猛毒ヘビ", 58, 13, 4, 32)
	definitions[21] = upperMonsterDefinition(definitions[3], 21, "TwistHoney", "ぐねぐねハニー", 50, 10, 7, 36)
	definitions[22] = upperMonsterDefinition(definitions[4], 22, "KingHarisenbow", "ハリセンオウ", 42, 11, 5, 30)
	definitions[23] = upperMonsterDefinition(definitions[5], 23, "BoulderCrab", "ガンセキガニ", 60, 12, 9, 44)
	definitions[24] = upperMonsterDefinition(definitions[6], 24, "DynamiteUrchin", "ダイナマイトウニ", 82, 17, 10, 72)
	definitions[25] = upperMonsterDefinition(definitions[7], 25, "MasterThiefHermitCrab", "オオドロボウヤドカリ", 52, 10, 7, 40)
	definitions[26] = upperMonsterDefinition(definitions[8], 26, "NigiriMasterShrimp", "にぎり親方エビ", 64, 12, 8, 48)
	definitions[27] = upperMonsterDefinition(definitions[9], 27, "CurseLordCrab", "タタリガニ", 70, 14, 10, 58)
	definitions[28] = upperMonsterDefinition(definitions[10], 28, "PuppetMasterJellyfish", "しはいクラゲ", 76, 15, 9, 66)
	definitions[29] = upperMonsterDefinition(definitions[11], 29, "ReaperShrimp", "シニガミエビ", 54, 14, 5, 48)
	definitions[30] = upperMonsterDefinition(definitions[12], 30, "LightningMantisShrimp", "イナズマシャコ", 46, 11, 6, 50)
	definitions[31] = upperMonsterDefinition(definitions[13], 31, "TeleportJellyfish", "テレポクラゲ", 58, 14, 8, 56)
	definitions[32] = upperMonsterDefinition(definitions[14], 32, "ExchangeOctopus", "トッカエダコ", 68, 15, 9, 64)
	definitions[33] = upperMonsterDefinition(definitions[15], 33, "TrapMasterHermitCrab", "ワナマスターヤドカリ", 74, 14, 11, 68)
	definitions[34] = upperMonsterDefinition(definitions[16], 34, "MimicConch", "化けホラ貝", 62, 16, 10, 58)
	definitions[35] = upperMonsterDefinition(definitions[17], 35, "MimicConch", "化けホラ貝", 74, 18, 11, 70)
	definitions[37] = upperMonsterDefinition(definitions[36], 37, "AzureSeaDragon", "蒼海竜", 108, 22, 14, 110)
	definitions[40] = upperMonsterDefinition(definitions[38], 40, "InfernoSquid", "業火イカ", 88, 17, 9, 70)
	definitions[41] = upperMonsterDefinition(definitions[39], 41, "ArcaneJellyfish", "ゲンソウクラゲ", 94, 18, 10, 82)

	for id, attackPower := range map[int]int{22: 14, 23: 18, 24: 24, 40: 22, 41: 24} {
		definition := definitions[id]
		definition.RangedAttack.AttackPower = attackPower
		definitions[id] = definition
	}
	return definitions
}

// MonsterDefinitions は基本種と上位種を含む統一モンスター定義テーブル。
var MonsterDefinitions = buildMonsterDefinitions()

// MonsterLevelUpTable は敵がほかの敵を倒したときに変化する同系統の上位種を定義する。
var MonsterLevelUpTable = map[int]int{
	0:  18, // エビ -> 大エビ
	1:  20, // 毒ヘビ -> 猛毒ヘビ
	2:  19, // マムル -> あなぐらマムル
	3:  21, // くねくねハニー -> ぐねぐねハニー
	4:  22, // ハリセンボウ -> ハリセンオウ
	5:  23, // イシガニ -> ガンセキガニ
	6:  24, // バクダンウニ -> ダイナマイトウニ
	7:  25, // コソドロヤドカリ -> オオドロボウヤドカリ
	8:  26, // にぎりエビ -> にぎり親方エビ
	9:  27, // ノロイガニ -> タタリガニ
	10: 28, // あやつりクラゲ -> しはいクラゲ
	11: 29, // ユウレイエビ -> シニガミエビ
	12: 30, // ハヤテシャコ -> イナズマシャコ
	13: 31, // ワープクラゲ -> テレポクラゲ
	14: 32, // イレカエダコ -> トッカエダコ
	15: 33, // ワナシヤドカリ -> ワナマスターヤドカリ
	16: 34, // 道具に擬態する化け貝 -> 化けホラ貝
	17: 35, // 階段に擬態する化け貝 -> 化けホラ貝
	36: 37, // 海竜 -> 蒼海竜
	38: 40, // 火吹きイカ -> 業火イカ
	39: 41, // マホウクラゲ -> ゲンソウクラゲ
}

// levelUpEnemy は敵を同系統の上位種へ変化させる。
// 位置・状態異常・発見状態・盗品などの可変状態は維持し、能力値と固有能力だけを上位種へ更新する。
func (g *Game) levelUpEnemy(index int) bool {
	if index < 0 || index >= len(g.state.Enemies) {
		return false
	}

	enemy := &g.state.Enemies[index]
	nextID, ok := MonsterLevelUpTable[enemy.ID]
	if !ok {
		return false
	}
	definition, ok := MonsterDefinitions[nextID]
	if !ok {
		return false
	}

	oldName := enemy.Name
	enemy.ID = definition.ID
	enemy.Char = definition.Char
	enemy.Name = definition.Name
	enemy.Health = definition.MaxHealth
	enemy.MaxHealth = definition.MaxHealth
	enemy.AttackPower = definition.AttackPower
	enemy.DefensePower = definition.DefensePower
	enemy.Type = definition.Type
	enemy.ExperiencePoints = definition.ExperiencePoints
	enemy.SpecialAttack = definition.SpecialAttack
	enemy.SpecialAttackProbability = definition.SpecialAttackProbability
	enemy.RangedAttack = definition.RangedAttack
	enemy.SpecialMovement = definition.SpecialMovement
	enemy.Disguise = definition.Disguise
	enemy.Traits = append([]EnemyTrait(nil), definition.Traits...)
	g.EnqueueMessage(fmt.Sprintf("%sは%sにレベルアップした。", oldName, enemy.Name), 0.5)
	return true
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
		Traits:                   append([]EnemyTrait(nil), def.Traits...),
		Revealed:                 def.Disguise == EnemyDisguiseNone,
		ShowOnMiniMap:            true,
	}
}

func createEnemy(x, y int) Enemy {
	randomValue := rand.Intn(len(MonsterDefinitions))
	return CreateEnemyByID(randomValue, x, y)
}

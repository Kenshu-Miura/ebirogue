//go:build !test

package main

import (
	"fmt"
	"math/rand"
)

// モンスター湧きシステムの定数
const (
	MaxEnemiesOnFloor = 19 // フロア内の敵の上限数
	MinSpawnInterval  = 20 // 最小湧き間隔（ターン）
	MaxSpawnInterval  = 30 // 最大湧き間隔（ターン）
	SpawnCheckRange   = 8  // プレイヤーからこの距離内では湧かない
)

// 階層別モンスタースポーンテーブル
type MonsterSpawnEntry struct {
	MonsterID int // 敵のID
	Weight    int // 出現重み
}

// 各階層のスポーンテーブル
var FloorSpawnTables = map[int][]MonsterSpawnEntry{
	1: {{0, 70}, {1, 30}}, // 1階: エビ70%, ヘビ30%
	2: {{0, 50}, {1, 50}}, // 2階: エビ50%, ヘビ50%
	3: {{0, 30}, {1, 70}}, // 3階: エビ30%, ヘビ70%
	// より深い階層は3階と同じ設定を使用
}

// プレイヤーターン進行時の処理
func (g *Game) AdvanceTurn() {
	g.turnCount++

	// モンスター湧きチェック
	if g.ShouldSpawnMonster() {
		g.TrySpawnMonster()
	}
}

// モンスターを湧かせるべきかチェック
func (g *Game) ShouldSpawnMonster() bool {
	// 敵の上限チェック
	if len(g.state.Enemies) >= MaxEnemiesOnFloor {
		return false
	}

	// 湧き間隔チェック
	turnsSinceLastSpawn := g.turnCount - g.lastSpawnTurn
	return turnsSinceLastSpawn >= g.spawnInterval
}

// モンスター湧き試行
func (g *Game) TrySpawnMonster() {
	// 有効なスポーン位置を検索
	spawnPos := g.FindValidSpawnPosition()
	if spawnPos == nil {
		// 有効な位置がない場合は次回に延期
		g.SetNextSpawnInterval()
		return
	}

	// モンスター生成
	monsterID := g.SelectMonsterForSpawn()
	enemy := g.CreateEnemyByID(monsterID, spawnPos.X, spawnPos.Y)

	// ゲーム状態に追加
	g.state.Enemies = append(g.state.Enemies, enemy)

	// ログ出力
	fmt.Printf("モンスターが湧きました: %s 座標(%d, %d)\n", enemy.Name, spawnPos.X, spawnPos.Y)

	// 次回湧き時間設定
	g.lastSpawnTurn = g.turnCount
	g.SetNextSpawnInterval()
}

// 有効なスポーン位置を検索
func (g *Game) FindValidSpawnPosition() *Coordinate {
	playerX, playerY := g.state.Player.X, g.state.Player.Y
	maxAttempts := 100

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// ランダムな部屋を選択
		roomIndex := rand.Intn(len(g.rooms))
		room := g.rooms[roomIndex]

		// 部屋内のランダムな位置
		x := room.X + 1 + rand.Intn(room.Width-2)
		y := room.Y + 1 + rand.Intn(room.Height-2)

		// 位置が有効かチェック
		if g.IsValidSpawnPosition(x, y, playerX, playerY) {
			return &Coordinate{X: x, Y: y}
		}
	}

	return nil // 有効な位置が見つからなかった
}

// スポーン位置が有効かチェック
func (g *Game) IsValidSpawnPosition(x, y, playerX, playerY int) bool {
	// マップ範囲チェック
	if x < 0 || x >= len(g.state.Map[0]) || y < 0 || y >= len(g.state.Map) {
		return false
	}

	// 床タイルかチェック
	if g.state.Map[y][x].Type != "floor" || g.state.Map[y][x].Blocked {
		return false
	}

	// プレイヤーからの距離チェック
	dx := abs(x - playerX)
	dy := abs(y - playerY)
	if dx < SpawnCheckRange && dy < SpawnCheckRange {
		return false
	}

	// プレイヤーの視界内かチェック（同じ部屋内では湧かない）
	if g.IsInPlayerVisibleRoom(x, y) {
		return false
	}

	// 既存の敵との位置重複チェック
	for _, enemy := range g.state.Enemies {
		if enemy.X == x && enemy.Y == y {
			return false
		}
	}

	// 既存アイテムとの重複チェック
	for _, item := range g.state.Items {
		itemX, itemY := item.GetPosition()
		if itemX == x && itemY == y {
			return false
		}
	}

	return true
}

// プレイヤーの視界内（同じ部屋）かチェック
func (g *Game) IsInPlayerVisibleRoom(x, y int) bool {
	playerX, playerY := g.state.Player.X, g.state.Player.Y

	// プレイヤーがいる部屋を特定
	for _, room := range g.rooms {
		if playerX >= room.X && playerX < room.X+room.Width && playerY >= room.Y && playerY < room.Y+room.Height {
			// 同じ部屋内かチェック
			if x >= room.X && x < room.X+room.Width && y >= room.Y && y < room.Y+room.Height {
				return true
			}
			break
		}
	}

	return false
}

// 階層に応じたモンスター選択
func (g *Game) SelectMonsterForSpawn() int {
	floor := g.Floor

	// 階層に応じたスポーンテーブル取得
	spawnTable := FloorSpawnTables[floor]
	if spawnTable == nil {
		// 定義されていない階層は最後の階層設定を使用
		spawnTable = FloorSpawnTables[3]
	}

	// 重み付き選択
	totalWeight := 0
	for _, entry := range spawnTable {
		totalWeight += entry.Weight
	}

	randomValue := rand.Intn(totalWeight)
	currentWeight := 0

	for _, entry := range spawnTable {
		currentWeight += entry.Weight
		if randomValue < currentWeight {
			return entry.MonsterID
		}
	}

	// フォールバック
	return 0
}

// IDに応じた敵生成
func (g *Game) CreateEnemyByID(id, x, y int) Enemy {
	switch id {
	case 0:
		// エビ
		return Enemy{
			Entity:           Entity{X: x, Y: y, Char: 'E'},
			ID:               0,
			Name:             "エビ",
			Health:           20,
			MaxHealth:        20,
			AttackPower:      4,
			DefensePower:     2,
			Type:             "Shrimp",
			ExperiencePoints: 5,
			PlayerDiscovered: false,
			Direction:        Uninitialized,
			ShowOnMiniMap:    true,
		}
	case 1:
		// ヘビ
		return Enemy{
			Entity:           Entity{X: x, Y: y, Char: 'S'},
			ID:               1,
			Name:             "毒ヘビ",
			Health:           30,
			MaxHealth:        30,
			AttackPower:      7,
			DefensePower:     1,
			Type:             "Snake",
			ExperiencePoints: 10,
			PlayerDiscovered: false,
			Direction:        Uninitialized,
			ShowOnMiniMap:    true,
			SpecialAttack: func(e *Enemy, g *Game) {
				if g.state.Player.Power > 0 {
					g.state.Player.Power--
					action := Action{
						Duration: 0.5,
						Message:  "毒でパワーが下がった！",
						Execute:  func(g *Game) {},
					}
					g.Enqueue(action)
				}
			},
			SpecialAttackProbability: 0.3,
		}
	default:
		// デフォルトはエビ
		return g.CreateEnemyByID(0, x, y)
	}
}

// 次回湧き間隔を設定
func (g *Game) SetNextSpawnInterval() {
	g.spawnInterval = MinSpawnInterval + rand.Intn(MaxSpawnInterval-MinSpawnInterval+1)
}

// 初期化時の湧き設定
func (g *Game) InitializeSpawnSystem() {
	g.turnCount = 0
	g.lastSpawnTurn = 0
	g.SetNextSpawnInterval()
}

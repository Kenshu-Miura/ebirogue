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
	1: {{0, 50}, {2, 35}, {1, 15}},          // 1階: 基本敵を中心に出現
	2: {{0, 30}, {2, 30}, {1, 25}, {3, 15}}, // 2階: くねくねハニーが登場
	3: {{0, 15}, {2, 15}, {1, 35}, {3, 35}}, // 3階以降: 状態異常を使う敵が増える
	// より深い階層は3階と同じ設定を使用
}

// プレイヤーターン進行時の処理
func (g *Game) AdvanceTurn() {
	g.turnCount++
	g.floorTurns++ // フロア滞在ターン数を増加

	// フロア滞在時間チェック
	g.checkFloorTimeWarnings()

	// 仮眠状態の敵の起床チェック
	g.CheckSleepingEnemyWakeUp()

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
	enemy := CreateEnemyByID(monsterID, spawnPos.X, spawnPos.Y)

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

// 仮眠状態の敵の起床チェック
func (g *Game) CheckSleepingEnemyWakeUp() {
	playerX, playerY := g.state.Player.X, g.state.Player.Y

	for i := range g.state.Enemies {
		enemy := &g.state.Enemies[i]

		// 仮眠状態（Sleep = -1）の敵のみチェック
		if enemy.StatusAilments.Sleep != -1 {
			continue
		}

		enemyX, enemyY := enemy.GetPosition()

		// プレイヤーが同じ部屋に入った場合
		if isSameRoom(playerX, playerY, enemyX, enemyY, g.rooms) {
			if rand.Float64() < 0.5 { // 50%の確率で起床
				enemy.StatusAilments.Sleep = 0
			}
		}
	}
}

// 隣接時や攻撃時の起床チェック（他の関数から呼び出される）
func (g *Game) WakeUpSleepingEnemyByProximity(enemyIndex int) {
	enemy := &g.state.Enemies[enemyIndex]

	// 仮眠状態（Sleep = -1）の敵のみチェック
	if enemy.StatusAilments.Sleep != -1 {
		return
	}

	playerX, playerY := g.state.Player.X, g.state.Player.Y
	enemyX, enemyY := enemy.GetPosition()

	// プレイヤーが隣接している場合
	adjacent := (abs(playerX-enemyX) <= 1 && abs(playerY-enemyY) <= 1)
	if adjacent && rand.Float64() < 0.5 { // 50%の確率で起床
		enemy.StatusAilments.Sleep = 0
	}
}

// 攻撃による確実な起床
func (g *Game) WakeUpSleepingEnemyByAttack(enemyIndex int) {
	enemy := &g.state.Enemies[enemyIndex]

	// 仮眠状態の敵は攻撃を受けると必ず起床
	if enemy.StatusAilments.Sleep == -1 {
		enemy.StatusAilments.Sleep = 0
	}
}

// フロア滞在時間警告チェック
func (g *Game) checkFloorTimeWarnings() {
	// 1200ターン経過時の警告
	if g.floorTurns >= 1200 && !g.windWarning1Shown {
		g.windWarning1Shown = true
		windAction := Action{
			Duration:    2.0, // 2秒間表示
			Message:     "風が吹いてきた…",
			Execute:     func(g *Game) {},
			NonBlocking: false, // 入力をブロック
		}
		g.ActionQueue.Queue = append(g.ActionQueue.Queue, windAction)
		return
	}

	// 1300ターン経過時の警告
	if g.floorTurns >= 1300 && !g.windWarning2Shown {
		g.windWarning2Shown = true
		windAction := Action{
			Duration:    2.0, // 2秒間表示
			Message:     "風が吹いてきた…さっきより強いぞ",
			Execute:     func(g *Game) {},
			NonBlocking: false, // 入力をブロック
		}
		g.ActionQueue.Queue = append(g.ActionQueue.Queue, windAction)
		return
	}

	// 1400ターン経過時の死亡
	if g.floorTurns >= 1400 {
		g.playerDead = true
		g.gameResetTimer = 2.0 // 2秒後にリセット
		// 死亡したら中断データを削除する（復活による再開を防ぐ）
		deleteSaveFile()
		windDeathAction := Action{
			Duration: 2.0, // 2秒間表示
			Message:  "突風だ！海老さんは風に飛ばされた",
			Execute: func(g *Game) {
				// プレイヤー死亡処理は既にplayerDead=trueで実行される
			},
			NonBlocking: false,
		}
		g.ActionQueue.Queue = append(g.ActionQueue.Queue, windDeathAction)
	}
}

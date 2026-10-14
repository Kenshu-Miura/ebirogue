//go:build !test

package main

import ()

// モンスター湧きシステムの定数
const (
	MaxEnemiesOnFloor = 19 // フロア内の敵の上限数
	MinSpawnInterval  = 20 // 最小湧き間隔（ターン）
	MaxSpawnInterval  = 30 // 最大湧き間隔（ターン）
	SpawnCheckRange   = 8  // プレイヤーからこの距離内では湧かない
	deepestSpawnFloor = 30 // これより深い階は最深階のテーブルを使う
)

// 階層別モンスタースポーンテーブル
type MonsterSpawnEntry struct {
	MonsterID int // 敵のID
	Weight    int // 出現重み
}

// 各階層のスポーンテーブル
var FloorSpawnTables = map[int][]MonsterSpawnEntry{
	1: {{0, 50}, {2, 35}, {1, 15}},
	2: {{0, 15}, {2, 15}, {1, 20}, {3, 15}, {4, 10}, {7, 10}, {12, 15}},
	3: {{1, 15}, {3, 15}, {4, 10}, {5, 10}, {7, 10}, {8, 10}, {9, 5}, {11, 10}, {13, 15}, {38, 5}},
	4: {{3, 10}, {5, 10}, {6, 10}, {8, 10}, {9, 10}, {10, 10}, {11, 10}, {12, 10}, {14, 10}, {15, 5}, {16, 5}, {38, 8}, {39, 5}},
	5: {{5, 10}, {6, 10}, {9, 10}, {10, 10}, {11, 10}, {12, 10}, {13, 10}, {14, 10}, {15, 10}, {16, 10}, {19, 8}, {20, 5}, {21, 5}, {22, 5}, {25, 5}, {38, 10}, {39, 8}},
	6: {{6, 10}, {9, 10}, {10, 10}, {11, 10}, {12, 10}, {13, 10}, {14, 10}, {15, 10}, {16, 10}, {17, 10}, {18, 8}, {19, 8}, {20, 5}, {21, 5}, {22, 5}, {23, 5}, {25, 5}, {26, 5}, {27, 5}, {29, 5}, {30, 5}, {38, 10}, {39, 10}},
	7: {{6, 10}, {10, 10}, {13, 10}, {14, 10}, {15, 10}, {16, 5}, {17, 5}, {18, 8}, {19, 5}, {23, 8}, {24, 5}, {26, 5}, {27, 5}, {28, 5}, {29, 5}, {30, 5}, {31, 5}, {32, 5}, {33, 5}, {34, 5}, {36, 8}, {38, 8}, {39, 8}, {40, 5}, {41, 5}},
	8: {{6, 10}, {10, 10}, {13, 10}, {14, 10}, {15, 10}, {16, 5}, {17, 5}, {18, 10}, {19, 5}, {23, 5}, {24, 8}, {27, 5}, {28, 8}, {29, 5}, {30, 5}, {31, 8}, {32, 8}, {33, 8}, {34, 5}, {35, 8}, {36, 8}, {37, 4}, {38, 5}, {39, 5}, {40, 10}, {41, 10}},
}

// deepFloorSpawnBands は9〜30階の出現テーブル。From〜To 階で同じテーブルを使う。
// 階層で敵の能力値は変えず、第2段階から第3段階（ID 45〜65）への入れ替わりで強さを上げる。
var deepFloorSpawnBands = []struct {
	From, To int
	Entries  []MonsterSpawnEntry
}{
	{9, 12, []MonsterSpawnEntry{
		{6, 5}, {10, 5}, {13, 5}, {14, 5}, {15, 5}, {16, 4}, {17, 4}, {18, 8}, {23, 6}, {24, 8}, {27, 6}, {28, 8}, {29, 6}, {30, 6},
		{31, 8}, {32, 8}, {33, 8}, {34, 5}, {35, 6}, {36, 6}, {37, 5}, {40, 8}, {41, 8},
		{45, 5}, {47, 5}, {48, 5}, {52, 5}, {56, 3}, {57, 5},
	}},
	{13, 16, []MonsterSpawnEntry{
		{18, 5}, {23, 5}, {24, 5}, {27, 5}, {28, 5}, {29, 5}, {31, 5}, {32, 5}, {33, 5}, {35, 5}, {37, 5}, {40, 5}, {41, 5},
		{45, 8}, {46, 6}, {47, 6}, {48, 8}, {49, 6}, {50, 6}, {52, 6}, {53, 6}, {56, 6}, {57, 6}, {58, 5}, {61, 4},
	}},
	{17, 20, []MonsterSpawnEntry{
		{24, 4}, {28, 4}, {32, 4}, {33, 4}, {37, 6}, {40, 4}, {41, 4},
		{45, 6}, {46, 8}, {48, 6}, {49, 8}, {50, 8}, {51, 5}, {52, 6}, {53, 8}, {54, 6}, {55, 6}, {56, 6}, {57, 6}, {58, 6},
		{59, 6}, {60, 5}, {61, 5}, {62, 4}, {64, 4}, {65, 4},
	}},
	{21, 25, []MonsterSpawnEntry{
		{37, 6},
		{45, 8}, {46, 8}, {47, 3}, {48, 8}, {49, 8}, {50, 8}, {51, 8}, {52, 8}, {53, 8}, {54, 8}, {55, 8}, {56, 8}, {57, 8},
		{58, 8}, {59, 8}, {60, 8}, {61, 6}, {62, 6}, {63, 5}, {64, 8}, {65, 8},
	}},
	{26, 30, []MonsterSpawnEntry{
		{45, 6}, {46, 8}, {47, 2}, {48, 6}, {49, 8}, {50, 8}, {51, 8}, {52, 8}, {53, 8}, {54, 8}, {55, 8}, {56, 8}, {57, 8},
		{58, 8}, {59, 8}, {60, 8}, {61, 6}, {62, 6}, {63, 8}, {64, 10}, {65, 10},
	}},
}

func init() {
	for _, band := range deepFloorSpawnBands {
		for floor := band.From; floor <= band.To; floor++ {
			FloorSpawnTables[floor] = band.Entries
		}
	}
	// 追加計画の敵（enemy_expansion.go）の出現階は、帯を展開した後に階ごとに加える。
	applyMonsterFamilySpawns(FloorSpawnTables, allMonsterFamilies())
}

// プレイヤーターン進行時の処理
func (g *Game) AdvanceTurn() {
	g.turnCount++
	g.floorTurns++ // フロア滞在ターン数を増加

	// フロア滞在時間チェック
	g.checkFloorTimeWarnings()

	// 遅れて現れる敵（ヨロイエビ系など、enemy_behaviors.go）
	g.processPendingEnemySpawns()

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

	// モンスター生成（ジェノサイドで全系統が封じられている場合は湧かない）
	monsterID := g.SelectMonsterForSpawn()
	if monsterID < 0 {
		g.SetNextSpawnInterval()
		return
	}
	enemy := CreateEnemyByID(monsterID, spawnPos.X, spawnPos.Y)
	prepareSpawnedEnemy(&enemy, g.Floor, g.randIntn)

	// ゲーム状態に追加
	g.state.Enemies = append(g.state.Enemies, enemy)

	// ログ出力
	g.debugf("モンスターが湧きました: %s 座標(%d, %d)\n", enemy.Name, spawnPos.X, spawnPos.Y)

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
		roomIndex := g.randIntn(len(g.rooms))
		room := g.rooms[roomIndex]

		// 部屋内のランダムな位置
		x := room.X + 1 + g.randIntn(room.Width-2)
		y := room.Y + 1 + g.randIntn(room.Height-2)

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

// 階層に応じたモンスター選択。全系統が封じられている場合は-1を返す。
func (g *Game) SelectMonsterForSpawn() int {
	return selectMonsterForFloor(g.Floor, g.genocidedMonsterIDs, g.randIntn)
}

// selectMonsterForFloor は階層別テーブルから敵IDを重み付きで選ぶ。
// banned（ジェノサイドで封じられた敵ID）は選択から除外し、候補がなければ-1を返す。
// 乱数関数を注入できるため、マップ初期配置とテストでも同じ選択規則を使える。
func selectMonsterForFloor(floor int, banned map[int]bool, intn func(int) int) int {

	// 階層に応じたスポーンテーブル取得
	spawnTable := FloorSpawnTables[floor]
	if spawnTable == nil {
		// 定義されていない深い階層は最後の階層設定を使用
		spawnTable = FloorSpawnTables[deepestSpawnFloor]
	}

	// 重み付き選択（封じられた系統は除外）
	totalWeight := 0
	for _, entry := range spawnTable {
		if banned[entry.MonsterID] {
			continue
		}
		totalWeight += entry.Weight
	}
	if totalWeight == 0 {
		return -1
	}

	randomValue := intn(totalWeight)
	currentWeight := 0

	for _, entry := range spawnTable {
		if banned[entry.MonsterID] {
			continue
		}
		currentWeight += entry.Weight
		if randomValue < currentWeight {
			return entry.MonsterID
		}
	}

	// フォールバック
	return -1
}

// 次回湧き間隔を設定
func (g *Game) SetNextSpawnInterval() {
	g.spawnInterval = MinSpawnInterval + g.randIntn(MaxSpawnInterval-MinSpawnInterval+1)
	g.spawnInterval = ringSpawnInterval(g.spawnInterval, g.playerAbilityCount(AbilityMonsterCall))
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
			if g.random().Float64() < 0.5 { // 50%の確率で起床
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
	if adjacent && g.random().Float64() < 0.5 { // 50%の確率で起床
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
		g.lastDamageSource = DamageSource{Kind: DamageSourceWind}
		g.finishRun(false)
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

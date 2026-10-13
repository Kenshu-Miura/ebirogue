package main

// Difficulty は冒険の難易度。冒険開始時に選び、途中で変更しない（SaveData に保存する）。
type Difficulty int

const (
	DifficultyBeginner     Difficulty = iota // 初級: 10階のボスを倒すとクリア
	DifficultyIntermediate                   // 中級: 20階のボスを倒すとクリア
	DifficultyAdvanced                       // 上級（標準）: 30階のボスを倒すとクリア
)

// allDifficulties は選択肢・集計の並び順。
var allDifficulties = []Difficulty{DifficultyBeginner, DifficultyIntermediate, DifficultyAdvanced}

const (
	bossFloorInterval = 10 // ボス階の間隔（10・20・30階）
	finalFloor        = 30 // 最下層。これより下には進めない
)

func (d Difficulty) String() string {
	switch d {
	case DifficultyBeginner:
		return "初級"
	case DifficultyIntermediate:
		return "中級"
	case DifficultyAdvanced:
		return "上級"
	default:
		return "不明"
	}
}

// valid は定義済みの難易度かどうかを返す。
func (d Difficulty) valid() bool {
	return d >= DifficultyBeginner && d <= DifficultyAdvanced
}

// goalFloor は難易度ごとのクリア条件となるボス階（10・20・30）を返す。
func goalFloor(d Difficulty) int {
	switch d {
	case DifficultyBeginner:
		return 10
	case DifficultyIntermediate:
		return 20
	default:
		return finalFloor
	}
}

// isBossFloor はボスが配置される階（10・20・30階）かどうかを返す。難易度に関係なく共通。
func isBossFloor(floor int) bool {
	return floor > 0 && floor <= finalFloor && floor%bossFloorInterval == 0
}

// bossIDForFloor はボス階に配置するボスの敵 ID を返す（docs/agents/README.md の ID 割り当て: 42〜44）。
func bossIDForFloor(floor int) (int, bool) {
	switch floor {
	case 10:
		return 42, true
	case 20:
		return 43, true
	case 30:
		return 44, true
	default:
		return 0, false
	}
}

// stairsUsable はその階の階段を使えるかどうかを返す。
// ボス階ではボスを倒すまで使えず、最下層（30階）より下へは進めない。
func stairsUsable(floor int, bossDefeated bool) bool {
	if floor >= finalFloor {
		return false
	}
	return !isBossFloor(floor) || bossDefeated
}

// isClearingBossFloor はその階のボスを倒すと冒険クリアになるかどうかを返す。
func isClearingBossFloor(floor int, d Difficulty) bool {
	return isBossFloor(floor) && floor == goalFloor(d)
}

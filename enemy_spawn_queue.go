package main

// PendingEnemySpawn は遅れて現れる敵1体の予約（ヨロイエビ系が倒れた後のユウレイエビ系など）。
// フロアを移動すると予約は消える。中断セーブ（SaveData.PendingEnemySpawns）に含める。
type PendingEnemySpawn struct {
	MonsterID int
	X, Y      int // 現れる位置の中心
	Turns     int // 残りターン。0 以下になったターンに現れる
}

// tickPendingEnemySpawns は予約の残りターンを1減らし、現れる予約（due）と待ち続ける予約（waiting）に分ける。
// 元のスライスは書き換えない。
func tickPendingEnemySpawns(spawns []PendingEnemySpawn) (due, waiting []PendingEnemySpawn) {
	for _, spawn := range spawns {
		spawn.Turns--
		if spawn.Turns <= 0 {
			due = append(due, spawn)
		} else {
			waiting = append(waiting, spawn)
		}
	}
	return due, waiting
}

// nearestFreeOffsets は中心から近い順（チェビシェフ距離、同じ距離なら上→下・左→右）の
// 半径 radius 以内の相対座標を返す（中心を含む）。
func nearestFreeOffsets(radius int) []Coordinate {
	offsets := []Coordinate{{X: 0, Y: 0}}
	for distance := 1; distance <= radius; distance++ {
		for dy := -distance; dy <= distance; dy++ {
			for dx := -distance; dx <= distance; dx++ {
				if max(abs(dx), abs(dy)) == distance {
					offsets = append(offsets, Coordinate{X: dx, Y: dy})
				}
			}
		}
	}
	return offsets
}

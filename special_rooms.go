package main

// RoomRole は特別な部屋の用途。通常の部屋は空文字として旧セーブとも互換にする。
type RoomRole string

const RoomRoleMonsterHouse RoomRole = "monster_house"

// selectSpecialRoom は開始地点・階段・既に役割のある部屋を避け、添字を返す。
func selectSpecialRoom(rooms []Room, start, stairs Coordinate, intn func(int) int) int {
	var candidates []int
	for i, room := range rooms {
		if room.Role == "" && !roomContains(room, start.X, start.Y) && !roomContains(room, stairs.X, stairs.Y) {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		return -1
	}
	return candidates[intn(len(candidates))]
}

func roomContains(r Room, x, y int) bool {
	return x > r.X && x < r.X+r.Width-1 && y > r.Y && y < r.Y+r.Height-1
}

func monsterHouseChance(floor int) int {
	if floor < 4 || isBossFloor(floor) {
		return 0
	}
	if floor < 10 {
		return 5
	}
	return 8
}

// カードは従来の小規模な群れ、自然生成は通行可能な床面積に比例する。
func monsterHouseCounts(floorCells int, card bool, intn func(int) int) (int, int) {
	if card {
		return rollMonsterHouseEnemyCount(intn), rollMonsterHouseItemCount(intn)
	}
	return floorCells / 3, floorCells / 5
}

func rollMonsterHouseEnemyCount(intn func(int) int) int { return 5 + intn(4) }
func rollMonsterHouseItemCount(intn func(int) int) int  { return 2 + intn(2) }

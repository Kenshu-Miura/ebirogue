package main

// 特別な部屋（wave10-A のモンスターハウス・wave10-B の店）の共通処理。
// 部屋の役割と部屋の選び方は 10-A と同じもの（10-A の special_rooms.go に合わせた）。統合時は1つにまとめる。

// RoomRole は特別な部屋の用途。通常の部屋は空文字として旧セーブとも互換にする。
type RoomRole string

// RoomRoleShop は店の部屋（wave10-B）。
const RoomRoleShop RoomRole = "shop"

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

// roomContainsWithWalls は (x, y) が部屋の壁の線（出入口を含む）の内側にあるか。
func roomContainsWithWalls(r Room, x, y int) bool {
	return x >= r.X && x <= r.X+r.Width-1 && y >= r.Y && y <= r.Y+r.Height-1
}

// --- 店（wave10-B） ---

const (
	shopFirstFloor     = 3  // 店が出始める階
	shopChancePercent  = 10 // 店が出る確率（%）
	shopMinItems       = 5  // 商品の数の下限
	shopMaxItems       = 12 // 商品の数の上限
	shopItemRerollMax  = 8  // 小銭を引いたときの引き直しの上限
	shopGuardDogCount  = 3  // 泥棒をしたときに現れる番犬の数
	shopEliteDogFloor  = 16 // この階から強い番犬（128）が現れる
	shopAlarmSpawnDiv  = 3  // 泥棒の後の湧き間隔の割る数
	shopAlarmSpawnMin  = 4  // 泥棒の後の湧き間隔の下限（ターン）
	shopKeeperHasteTrn = 9999
)

// shopChance は floor 階で店が出る確率（%）。1〜2階とボス階には出ない。
func shopChance(floor int) int {
	if floor < shopFirstFloor || isBossFloor(floor) {
		return 0
	}
	return shopChancePercent
}

// rollShopItemCount は商品の数（shopMinItems〜shopMaxItems、置ける床の数まで）。
func rollShopItemCount(freeCells int, intn func(int) int) int {
	return min(shopMinItems+intn(shopMaxItems-shopMinItems+1), freeCells)
}

// shopAlarmSpawnInterval は泥棒をした階の湧き間隔（通常の間隔を縮める）。
func shopAlarmSpawnInterval(interval int) int {
	return max(interval/shopAlarmSpawnDiv, shopAlarmSpawnMin)
}

// roomExits は部屋の壁の線上にある通れるマス（通路の出入口）を、上の辺から順に返す。
func roomExits(mapGrid [][]Tile, r Room) []Coordinate {
	var exits []Coordinate
	for y := r.Y; y <= r.Y+r.Height-1; y++ {
		for x := r.X; x <= r.X+r.Width-1; x++ {
			if roomContains(r, x, y) || y < 0 || y >= len(mapGrid) || x < 0 || x >= len(mapGrid[y]) {
				continue
			}
			if !mapGrid[y][x].Blocked {
				exits = append(exits, Coordinate{X: x, Y: y})
			}
		}
	}
	return exits
}

// shopDoorway は出入口 exit のすぐ内側（部屋の床）のマス。出入りの通り道なので商品・店主を置かない。
func shopDoorway(r Room, exit Coordinate) Coordinate {
	x := min(max(exit.X, r.X+1), r.X+r.Width-2)
	y := min(max(exit.Y, r.Y+1), r.Y+r.Height-2)
	return Coordinate{X: x, Y: y}
}

// shopKeeperSpot は店主の立つマス。出入口の内側のマスの、壁に沿った隣（通り道はふさがない）。
// reserved（ほかの出入口の内側）は避ける。置けなければ ok は false。
func shopKeeperSpot(r Room, exit Coordinate, reserved map[Coordinate]bool) (Coordinate, bool) {
	door := shopDoorway(r, exit)
	candidates := []Coordinate{{X: door.X, Y: door.Y - 1}, {X: door.X, Y: door.Y + 1}}
	if exit.Y == r.Y || exit.Y == r.Y+r.Height-1 { // 上下の壁の出入口は左右へずらす
		candidates = []Coordinate{{X: door.X - 1, Y: door.Y}, {X: door.X + 1, Y: door.Y}}
	}
	for _, c := range candidates {
		if roomContains(r, c.X, c.Y) && !reserved[c] {
			return c, true
		}
	}
	return Coordinate{}, false
}

//go:build !test

package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (g *Game) handleFadingOut() {
	g.fadeAlpha += 1.0 / 60 // 1秒かけて暗くする
	if g.fadeAlpha >= 1.0 {
		g.fadeAlpha = 1.0
		if g.frameCounter == 0 {
			// マップ生成
			mapGrid, enemies, items, newFloor, newRoom, traps := GenerateRandomMap(70, 70, g.Floor, &g.state.Player)
			// 新しいマップ情報を設定
			g.miniMap = nil
			g.state.Map = mapGrid
			g.state.Enemies = enemies
			g.state.Items = items
			g.state.MapTraps = traps
			g.Floor = newFloor
			g.rooms = newRoom
			// ジェノサイドで封じた系統は初期配置からも取り除く
			g.removeGenocidedEnemies()
			// フロア変更時のリセット処理
			g.floorTurns = 0
			g.windWarning1Shown = false
			g.windWarning2Shown = false
			g.pickupBanned = false // 拾得禁止はフロア移動で解除
			// フロア到着時にオートセーブ（クラッシュ時の復旧用）
			g.autoSave()
		}
		g.frameCounter++
		if g.frameCounter >= 60 { // 1秒経過した後
			g.fadingOut = false
			g.fadingIn = true
			g.frameCounter = 0 // フレームカウンターをリセット
		}
	}
}

func (g *Game) handleFadingIn() {
	g.fadeAlpha -= 1.0 / 60 // 1秒かけて明るくする
	if g.fadeAlpha <= 0.0 {
		g.fadeAlpha = 0.0
		g.fadingIn = false
		g.showStairsPrompt = false
	}
}

func getPlayerRoom(playerX, playerY int, rooms []Room) *Room {
	for i := range rooms {
		room := &rooms[i]
		if playerX >= room.X && playerX <= room.X+room.Width-1 &&
			playerY >= room.Y && playerY <= room.Y+room.Height-1 {
			return room
		}
	}
	return nil
}

func (g *Game) updateTileBrightness() {
	playerX, playerY := g.state.Player.GetPosition()
	inRoom := isInsideRoom(playerX, playerY, g.rooms)

	playerRoom := getPlayerRoom(playerX, playerY, g.rooms) // プレイヤーの部屋を取得

	for y, row := range g.state.Map {
		for x := range row {
			if inRoom && playerRoom != nil {
				// Check if the tile is in the same room as the player
				if x >= playerRoom.X && x <= playerRoom.X+playerRoom.Width-1 &&
					y >= playerRoom.Y && y <= playerRoom.Y+playerRoom.Height-1 {
					g.state.Map[y][x].Brightness = 1.0 // Fully bright
				} else {
					g.state.Map[y][x].Brightness = 0.2 // Fully dark
				}
			} else {
				// Check if the tile is adjacent to the player
				adjacent := (math.Abs(float64(playerX-x)) <= 1 && math.Abs(float64(playerY-y)) <= 1)
				if adjacent {
					g.state.Map[y][x].Brightness = 1.0 // Fully bright
				} else {
					g.state.Map[y][x].Brightness = 0.2 // Fully dark
				}
			}
		}
	}
}

func (g *Game) MarkVisitedTiles(playerX, playerY int) {
	// 現在のタイルを取得
	currentTile := &g.state.Map[playerY][playerX]

	// プレイヤーがタイルを訪れたことをマーク
	currentTile.Visited = true

	// 隣接タイルをマーク
	directions := []struct{ dx, dy int }{
		{0, 1}, {1, 0}, {0, -1}, {-1, 0}, // 上、右、下、左
		{1, 1}, {1, -1}, {-1, 1}, {-1, -1}, // 右上、右下、左上、左下
	}

	for _, dir := range directions {
		adjX, adjY := playerX+dir.dx, playerY+dir.dy
		if adjX >= 0 && adjX < len(g.state.Map[0]) && adjY >= 0 && adjY < len(g.state.Map) {
			adjTile := &g.state.Map[adjY][adjX]
			if adjTile.Type == "floor" || adjTile.Type == "corridor" {
				adjTile.Visited = true
			}
		}
	}
}

func (g *Game) MarkRoomVisited(playerX, playerY int) {
	// プレイヤーが新しい部屋に入ったかどうかを確認
	for _, room := range g.rooms {
		if isSameRoom(playerX, playerY, room.Center.X, room.Center.Y, g.rooms) {
			// プレイヤーが部屋に入ったので、部屋の全てのタイルを訪れたものとしてマーク
			for y := room.Y; y < room.Y+room.Height; y++ {
				for x := room.X; x < room.X+room.Width; x++ {
					g.state.Map[y][x].Visited = true
				}
			}
			break // 一つの部屋しかマークする必要はないので、ループを抜ける
		}
	}
}

func (g *Game) CheckPlayerMovement() {
	// プレイヤーが移動したかどうかを確認する
	playerMoved := g.prevPlayerX != g.state.Player.X || g.prevPlayerY != g.state.Player.Y

	// プレイヤーが移動したか、マップが変更された場合、
	// ミニマップを再描画する必要があることを示すフラグを設定します。
	if playerMoved {
		g.miniMapDirty = true
	}

	// プレイヤーの現在の座標を保存する
	g.prevPlayerX = g.state.Player.X
	g.prevPlayerY = g.state.Player.Y
}

// handleStairsPrompt handles user input for the stairs prompt.
func (g *Game) handleStairsPrompt() {
	if g.showStairsPrompt {
		if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
			g.selectedOption = (g.selectedOption + 1) % 2
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
			g.selectedOption = (g.selectedOption + 1) % 2
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			if g.selectedOption == 0 { // "Proceed" is selected
				g.fadingOut = true // 暗転開始
				g.fadeAlpha = 0.0
			} else { // "Cancel" is selected
				g.selectedOption = 0
				g.ignoreStairs = true
			}
			g.showStairsPrompt = false // Close the prompt window
			g.selectedOption = 0
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyX) {
			g.selectedOption = 0
			g.ignoreStairs = true
			g.showStairsPrompt = false // Close the prompt window
		}
	}
}

// ResetStairsIgnoreFlag resets the ignoreStairs flag when player moves away from stairs.
func (g *Game) ResetStairsIgnoreFlag() {
	player := &g.state.Player
	playerTile := g.state.Map[player.Y][player.X]
	if playerTile.Type != "stairs" {
		g.ignoreStairs = false
	}
}

func (g *Game) checkForStairs() {
	player := &g.state.Player
	playerTile := g.state.Map[player.Y][player.X]

	// インベントリ表示中はSキーが整頓操作のため階段プロンプトを開かない
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && g.ignoreStairs && playerTile.Type == "stairs" && !g.showInventory {
		g.showStairsPrompt = true
		g.ignoreStairs = false // Optionally reset ignoreStairs flag
		return
	}

	if playerTile.Type == "stairs" && !g.ignoreStairs && !g.showStairsPrompt {
		g.showStairsPrompt = true
	}
}

func logCurrentRoom(player Player, rooms []Room) string {
	for _, room := range rooms {
		// Check if the player is within the bounds of the current room
		if player.X >= room.X && player.X < room.X+room.Width &&
			player.Y >= room.Y && player.Y < room.Y+room.Height {
			return fmt.Sprintf("Room ID: %d\n", room.ID)
		}
	}
	return ""
}

func logRooms(rooms []Room) {
	for _, room := range rooms {
		fmt.Printf("Room ID: %d\n", room.ID)
		fmt.Printf("  Center: X=%d, Y=%d\n", room.Center.X, room.Center.Y)
	}
}

// Helper function to calculate the distance between two points
func distance(x1, y1, x2, y2 int) int {
	dx := x2 - x1
	dy := y2 - y1
	return int(math.Sqrt(float64(dx*dx + dy*dy)))
}

// Helper function to check if the distance between the center of the new room
// and the center of any existing room is within a specific range
func isWithinDistanceRange(newRoom Room, rooms []Room, minDistance, maxDistance int) bool {
	for _, room := range rooms {
		dist := distance(newRoom.Center.X, newRoom.Center.Y, room.Center.X, room.Center.Y)
		if dist < minDistance || dist > maxDistance {
			return false
		}
	}
	return true
}

func generateRooms(mapGrid [][]Tile, width, height, numRooms int) []Room {
	var rooms []Room

	for i := 0; i < numRooms; i++ { // Attempt to create a specified number of rooms
		for attempt := 0; attempt < 100; attempt++ { // Limit of 100 attempts per room
			var roomX, roomY, roomWidth, roomHeight int

			// If there are already rooms created, try to align the new room with one of them
			if len(rooms) > 0 {
				alignWith := rooms[rand.Intn(len(rooms))] // Randomly select a room to align with

				// Randomly decide to align horizontally or vertically
				if rand.Intn(2) == 0 {
					// Align horizontally
					roomWidth = rand.Intn(10) + 6 // Random width between 6 and 15
					roomHeight = alignWith.Height // Match the height of the room to align with
					roomX = rand.Intn(width-roomWidth-1) + 1
					roomY = alignWith.Y
				} else {
					// Align vertically
					roomWidth = alignWith.Width    // Match the width of the room to align with
					roomHeight = rand.Intn(10) + 6 // Random height between 6 and 15
					roomX = alignWith.X
					roomY = rand.Intn(height-roomHeight-1) + 1
				}
			} else {
				// If this is the first room, generate random dimensions and position
				roomWidth = rand.Intn(min(10, width-2)) + 6   // Random width between 6 and 15, but not exceeding map width
				roomHeight = rand.Intn(min(10, height-2)) + 6 // Random height between 6 and 15, but not exceeding map height
				roomX = rand.Intn(width-roomWidth-1) + 1
				roomY = rand.Intn(height-roomHeight-1) + 1
			}

			newRoom := Room{
				ID:     i, // Assign the unique ID to the room
				X:      roomX,
				Y:      roomY,
				Width:  roomWidth,
				Height: roomHeight,
			}
			// 中心座標は距離判定に使うため先に確定させる
			setRoomCenter(&newRoom)

			valid := true
			for _, room := range rooms {
				if !newRoom.IsSeparatedBy(room, 5) {
					valid = false
					break
				}
			}

			if valid {
				// New validation to ensure rooms are not too far apart
				if !isWithinDistanceRange(newRoom, rooms, 10, 100) { // Assume min distance is 10 and max distance is 50 for now
					continue // Skip the rest of the loop and try again if the room is too far or too close
				}
				rooms = append(rooms, newRoom)
				carveRoom(mapGrid, newRoom)
				break // Exit the inner loop as soon as a room is successfully created
			}
		}
	}

	return rooms
}

func generateEnemies(rooms []Room, playerRoom Room, floor int) []Enemy {
	var enemies []Enemy

	// 4~9体のモンスターを生成
	numEnemies := 4 + rand.Intn(6) // 4 + (0~5) = 4~9

	// 使用済み部屋を追跡
	usedRooms := make(map[int]bool)
	usedRooms[playerRoom.ID] = true // プレイヤーの部屋は除外

	for i := 0; i < numEnemies; i++ {
		var enemyRoom Room
		var enemyX, enemyY int
		maxAttempts := 100

		for attempt := 0; attempt < maxAttempts; attempt++ {
			enemyRoom = rooms[rand.Intn(len(rooms))]

			// できる限り異なる部屋に配置（全部屋使用済みの場合は重複を許可）
			if len(usedRooms) < len(rooms) && usedRooms[enemyRoom.ID] {
				continue
			}

			enemyX = rand.Intn(enemyRoom.Width-2) + enemyRoom.X + 1
			enemyY = rand.Intn(enemyRoom.Height-2) + enemyRoom.Y + 1

			// 位置の重複チェック
			occupied := false
			for _, enemy := range enemies {
				if enemy.X == enemyX && enemy.Y == enemyY {
					occupied = true
					break
				}
			}

			if !occupied {
				usedRooms[enemyRoom.ID] = true
				break
			}
		}

		// 敵を生成し、50%の確率で仮眠状態にする
		// （ジェノサイドで封じた系統はフロア移動時のremoveGenocidedEnemiesで取り除かれる）
		enemyID := selectMonsterForFloor(floor, nil, rand.Intn)
		enemy := CreateEnemyByID(enemyID, enemyX, enemyY)
		if rand.Float64() < 0.5 {
			enemy.StatusAilments.Sleep = -1 // -1で仮眠状態を表現（通常の睡眠と区別）
		}

		enemies = append(enemies, enemy)
	}
	return enemies
}

func generateItems(rooms []Room) []Item {
	var items []Item
	for i := 0; i < 10; i++ {
		var itemRoom Room
		var itemX, itemY int
		for {
			itemRoom = rooms[rand.Intn(len(rooms))]
			itemX = rand.Intn(itemRoom.Width-2) + itemRoom.X + 1
			itemY = rand.Intn(itemRoom.Height-2) + itemRoom.Y + 1
			occupied := false
			for _, item := range items {
				newitemX, newitemY := item.GetPosition()
				if itemX == newitemX && itemY == newitemY {
					occupied = true
					break
				}
			}
			if !occupied {
				break
			}
		}

		items = append(items, createItem(itemX, itemY))
	}
	return items
}

// maxMapGenAttempts はマップ生成のリトライ上限。
// 全部屋が連結になるまで作り直す（通常は1回目で成功する）。
const maxMapGenAttempts = 30

func GenerateRandomMap(width, height, currentFloor int, player *Player) ([][]Tile, []Enemy, []Item, int, []Room, []MapTrap) {
	var mapGrid [][]Tile
	var rooms []Room

	for attempt := 0; attempt < maxMapGenAttempts; attempt++ {
		// Step 1: Initialize all tiles to "other" type
		mapGrid = make([][]Tile, height)
		for y := range mapGrid {
			mapGrid[y] = make([]Tile, width)
			for x := range mapGrid[y] {
				mapGrid[y][x] = Tile{Type: "other", Blocked: true, BlockSight: true}
			}
		}

		rooms = generateRooms(mapGrid, width, height, 6) // Step 2: Generate rooms
		if len(rooms) < 2 {
			continue
		}

		connectRooms(rooms, mapGrid)

		// 全部屋が通路でつながっていて、壁が通路に破壊されていないフロアだけを採用する
		if floorConnected(mapGrid, rooms) && roomWallsIntact(mapGrid, rooms) {
			break
		}
	}

	// プレイヤーの新しい位置を設定
	playerRoom := rooms[rand.Intn(len(rooms))]
	playerX := rand.Intn(playerRoom.Width-2) + playerRoom.X + 1  // Exclude walls
	playerY := rand.Intn(playerRoom.Height-2) + playerRoom.Y + 1 // Exclude walls
	player.Entity.X = playerX
	player.Entity.Y = playerY

	// 階段タイルを配置するためのランダムな部屋を選択
	stairsRoom := rooms[rand.Intn(len(rooms))]
	// 階段のランダムな位置を選ぶ（壁を避ける）
	stairsX := rand.Intn(stairsRoom.Width-2) + stairsRoom.X + 1
	stairsY := rand.Intn(stairsRoom.Height-2) + stairsRoom.Y + 1
	// 階段タイルを配置
	mapGrid[stairsY][stairsX] = Tile{Type: "stairs", Blocked: false, BlockSight: false}

	newFloor := currentFloor + 1

	// Call the newly created functions to generate enemies and items
	enemies := generateEnemies(rooms, playerRoom, newFloor)
	items := generateItems(rooms)

	// Generate map traps
	traps := generateMapTraps(rooms)

	return mapGrid, enemies, items, newFloor, rooms, traps
}

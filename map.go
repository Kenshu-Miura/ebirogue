//go:build !test
// +build !test

package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Room struct {
	ID            int
	X, Y          int
	Width, Height int
	Center        Coordinate
}

func (g *Game) handleFadingOut() {
	g.fadeAlpha += 1.0 / 60 // 1秒かけて暗くする
	if g.fadeAlpha >= 1.0 {
		g.fadeAlpha = 1.0
		if g.frameCounter == 0 {
			// マップ生成
			mapGrid, enemies, items, newFloor, newRoom := GenerateRandomMap(70, 70, g.Floor, &g.state.Player)
			// 新しいマップ情報を設定
			g.miniMap = nil
			g.state.Map = mapGrid
			g.state.Enemies = enemies
			g.state.Items = items
			g.Floor = newFloor
			g.rooms = newRoom
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

func isInsideRoom(x, y int, rooms []Room) bool {
	for _, room := range rooms {
		if x > room.X && x < room.X+room.Width-1 &&
			y > room.Y && y < room.Y+room.Height-1 {
			return true
		}
	}
	return false
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

	if inpututil.IsKeyJustPressed(ebiten.KeyS) && g.ignoreStairs && playerTile.Type == "stairs" {
		g.showStairsPrompt = true
		g.ignoreStairs = false // Optionally reset ignoreStairs flag
		return
	}

	if playerTile.Type == "stairs" && !g.ignoreStairs && !g.showStairsPrompt {
		g.showStairsPrompt = true
	}
}

func isInsideRoomOrOnBoundary(x, y int, rooms []Room) bool {
	for _, room := range rooms {
		if x >= room.X && x <= room.X+room.Width-1 &&
			y >= room.Y && y <= room.Y+room.Height-1 {
			return true
		}
	}
	return false
}

func placeDoor(mapGrid [][]Tile, x, y int) {
	mapGrid[y][x] = Tile{Type: "door", Blocked: false, BlockSight: true}
}

func drawCorridor(mapGrid [][]Tile, room1, room2 Room, rooms []Room) {
	// Get the center coordinates of the rooms
	x1, y1 := room1.Center.X, room1.Center.Y
	x2, y2 := room2.Center.X, room2.Center.Y

	// Determine the turning point
	turnX, turnY := x1, y2

	_, _, vertexDetected := detectVertex(mapGrid, x1, y1, turnX, turnY, rooms)
	if vertexDetected {
		turnX, turnY = x2, y1
	}

	_, _, vertexDetected = detectVertex(mapGrid, turnX, turnY, x2, y2, rooms)

	if vertexDetected {
		turnX, turnY = x2, y1
	}

	// Draw vertical corridor from the center of room1 to the turning point
	drawSegment(mapGrid, x1, y1, x1, turnY, rooms)

	// Draw horizontal corridor from the turning point to the center of room2
	drawSegment(mapGrid, turnX, turnY, x2, turnY, rooms)
}

func detectVertex(mapGrid [][]Tile, startX, startY, endX, endY int, rooms []Room) (int, int, bool) {
	// Determine the direction of the scan based on the start and end coordinates
	deltaX := 0
	deltaY := 0
	if startX != endX {
		deltaX = increment(startX, endX)
	} else {
		deltaY = increment(startY, endY)
	}

	// Initialize current position to the start coordinates
	currentX := startX
	currentY := startY

	// Continue the scan until the end point is reached or an edge is detected
	for currentX != endX || currentY != endY {
		// Update the current position
		currentX += deltaX
		currentY += deltaY
		for _, room := range rooms {
			// Calculate the vertices of the room
			topLeftX, topLeftY := room.X, room.Y
			topRightX, topRightY := room.X+room.Width-1, room.Y
			bottomLeftX, bottomLeftY := room.X, room.Y+room.Height-1
			bottomRightX, bottomRightY := room.X+room.Width-1, room.Y+room.Height-1

			// Check if the current position is near any of the vertices of the room
			if (currentX == topLeftX && currentY == topLeftY) ||
				(currentX == topRightX && currentY == topRightY) ||
				(currentX == bottomLeftX && currentY == bottomLeftY) ||
				(currentX == bottomRightX && currentY == bottomRightY) {
				// Vertex detected, stop the scan and return the current position
				return currentX, currentY, true
			}
		}
	}

	// No vertex detected, return the original turnY value and false
	return currentX, currentY, false
}

func increment(start, end int) int {
	if start < end {
		return 1 // Increment positively
	}
	return -1 // Increment negatively
}

func drawSegment(mapGrid [][]Tile, startX, startY, endX, endY int, rooms []Room) {
	for x := min(startX, endX); x <= max(startX, endX); x++ {
		for y := min(startY, endY); y <= max(startY, endY); y++ {
			isBoundary := false
			for _, room := range rooms {
				if isOnBoundary(x, y, room) {
					isBoundary = true
					//placeDoor(mapGrid, x, y)
					mapGrid[y][x] = Tile{Type: "corridor", Blocked: false, BlockSight: false}
					break
				}
			}
			if !isBoundary && !isInsideRoomOrOnBoundary(x, y, rooms) {
				mapGrid[y][x] = Tile{Type: "corridor", Blocked: false, BlockSight: false}
			}
		}
	}
}

func isOnBoundary(x, y int, room Room) bool {
	left := room.X
	right := room.X + room.Width - 1
	top := room.Y
	bottom := room.Y + room.Height - 1

	// Check if (x, y) is on the left, right, top, or bottom edge of the room
	isOnLeftEdge := x == left && y >= top && y <= bottom
	isOnRightEdge := x == right && y >= top && y <= bottom
	isOnTopEdge := y == top && x >= left && x <= right
	isOnBottomEdge := y == bottom && x >= left && x <= right

	return isOnLeftEdge || isOnRightEdge || isOnTopEdge || isOnBottomEdge
}

func connectRooms(rooms []Room, mapGrid [][]Tile) {
	if len(rooms) == 0 {
		fmt.Println("No rooms to connect")
		return
	}

	// Step 2: Connect each room to its nearest neighbor
	for _, room := range rooms {
		nearestNeighbor := findNearestNeighbor(room, rooms)
		// Assuming drawCorridor is updated to take Room structs or center coordinates as arguments
		drawCorridor(mapGrid, room, nearestNeighbor, rooms)
	}

	// Step 3: Connect all rooms in a circular manner
	for i := 0; i < len(rooms); i++ {
		// Get the next room index, wrapping back to 0 if at the end of the rooms slice
		nextRoomIndex := (i + 1) % len(rooms)
		// Again, assuming drawCorridor is updated to take Room structs or center coordinates as arguments
		drawCorridor(mapGrid, rooms[i], rooms[nextRoomIndex], rooms)
	}

	fmt.Println("All rooms are connected")
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

// Updated calculateDistance function to accept Room structures as arguments
func calculateDistance(room1, room2 Room) float64 {
	deltaX := float64(room2.Center.X - room1.Center.X)
	deltaY := float64(room2.Center.Y - room1.Center.Y)
	return math.Sqrt(deltaX*deltaX + deltaY*deltaY)
}

func findNearestNeighbor(room Room, rooms []Room) Room {
	minDistance := math.MaxFloat64
	var nearestRoom Room

	for _, neighbor := range rooms {
		// Skip if it's the same room
		if room.ID == neighbor.ID {
			continue
		}

		distance := calculateDistance(room, neighbor) // Updated to pass Room structures
		if distance < minDistance {
			minDistance = distance
			nearestRoom = neighbor
		}
	}

	return nearestRoom
}

func (r *Room) IsSeparatedBy(other Room, tiles int) bool {
	// Horizontal separation
	if r.X+r.Width+tiles <= other.X || other.X+other.Width+tiles <= r.X {
		return true
	}
	// Vertical separation
	if r.Y+r.Height+tiles <= other.Y || other.Y+other.Height+tiles <= r.Y {
		return true
	}
	return false
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
				alignWith := rooms[localRand.Intn(len(rooms))] // Randomly select a room to align with

				// Randomly decide to align horizontally or vertically
				if localRand.Intn(2) == 0 {
					// Align horizontally
					roomWidth = localRand.Intn(10) + 6 // Random width between 6 and 15
					roomHeight = alignWith.Height      // Match the height of the room to align with
					roomX = localRand.Intn(width-roomWidth-1) + 1
					roomY = alignWith.Y
				} else {
					// Align vertically
					roomWidth = alignWith.Width         // Match the width of the room to align with
					roomHeight = localRand.Intn(10) + 6 // Random height between 6 and 15
					roomX = alignWith.X
					roomY = localRand.Intn(height-roomHeight-1) + 1
				}
			} else {
				// If this is the first room, generate random dimensions and position
				roomWidth = localRand.Intn(min(10, width-2)) + 6   // Random width between 6 and 15, but not exceeding map width
				roomHeight = localRand.Intn(min(10, height-2)) + 6 // Random height between 6 and 15, but not exceeding map height
				roomX = localRand.Intn(width-roomWidth-1) + 1
				roomY = localRand.Intn(height-roomHeight-1) + 1
			}

			newRoom := Room{
				ID:     i, // Assign the unique ID to the room
				X:      roomX,
				Y:      roomY,
				Width:  roomWidth,
				Height: roomHeight,
			}
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
				setRoomCenter(&newRoom)
				rooms = append(rooms, newRoom)
				for y := roomY; y < roomY+roomHeight; y++ {
					for x := roomX; x < roomX+roomWidth; x++ {
						if x == roomX || x == roomX+roomWidth-1 || y == roomY || y == roomY+roomHeight-1 {
							mapGrid[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
						} else {
							mapGrid[y][x] = Tile{Type: "floor", Blocked: false, BlockSight: false}
						}
					}
				}
				break // Exit the inner loop as soon as a room is successfully created
			}
		}
	}

	return rooms
}

func setRoomCenter(room *Room) {
	// Calculate the center coordinates
	centerX := room.X + room.Width/2
	centerY := room.Y + room.Height/2

	// If the calculated center coordinates are even, increment them by 1 to make them odd
	if centerX%2 == 0 {
		centerX++
	}
	if centerY%2 == 0 {
		centerY++
	}

	// Set the center coordinates
	room.Center = Coordinate{X: centerX, Y: centerY}
}

func generateEnemies(rooms []Room, playerRoom Room) []Enemy {
	var enemies []Enemy
	for i := 0; i < 1; i++ {
		var enemyRoom Room
		var enemyX, enemyY int
		for {
			enemyRoom = rooms[localRand.Intn(len(rooms))]
			if enemyRoom.ID != playerRoom.ID {
				enemyX = localRand.Intn(enemyRoom.Width-2) + enemyRoom.X + 1
				enemyY = localRand.Intn(enemyRoom.Height-2) + enemyRoom.Y + 1
				occupied := false
				for _, enemy := range enemies {
					if enemy.X == enemyX && enemy.Y == enemyY {
						occupied = true
						break
					}
				}
				if !occupied {
					break
				}
			}
		}

		enemies = append(enemies, createEnemy(enemyX, enemyY))
	}
	return enemies
}

func generateItems(rooms []Room) []Item {
	var items []Item
	for i := 0; i < 10; i++ {
		var itemRoom Room
		var itemX, itemY int
		for {
			itemRoom = rooms[localRand.Intn(len(rooms))]
			itemX = localRand.Intn(itemRoom.Width-2) + itemRoom.X + 1
			itemY = localRand.Intn(itemRoom.Height-2) + itemRoom.Y + 1
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

func GenerateRandomMap(width, height, currentFloor int, player *Player) ([][]Tile, []Enemy, []Item, int, []Room) {
	// Step 1: Initialize all tiles to "other" type
	mapGrid := make([][]Tile, height)
	for y := range mapGrid {
		mapGrid[y] = make([]Tile, width)
		for x := range mapGrid[y] {
			mapGrid[y][x] = Tile{Type: "other", Blocked: true, BlockSight: true}
		}
	}

	// Generate a random float between 0 and 1
	//randomFloat := rand.Float64()

	// Apply exponential decay
	//decayFactor := 0.5 // Adjust this value to control the rate of decay
	//prob := math.Pow(randomFloat, decayFactor)

	// Scale and transform to get the number of rooms between 4 and 10
	//numRooms := int(prob*7) + 4                              // This will give a value between 4 and 10 with a decreasing probability as the number of rooms increases
	rooms := generateRooms(mapGrid, width, height, 6) // Step 2: Generate rooms

	connectRooms(rooms, mapGrid)

	// プレイヤーの新しい位置を設定
	playerRoom := rooms[localRand.Intn(len(rooms))]
	playerX := localRand.Intn(playerRoom.Width-2) + playerRoom.X + 1  // Exclude walls
	playerY := localRand.Intn(playerRoom.Height-2) + playerRoom.Y + 1 // Exclude walls
	player.Entity.X = playerX
	player.Entity.Y = playerY

	// 階段タイルを配置するためのランダムな部屋を選択
	stairsRoom := rooms[localRand.Intn(len(rooms))]
	// 階段のランダムな位置を選ぶ（壁を避ける）
	stairsX := localRand.Intn(stairsRoom.Width-2) + stairsRoom.X + 1
	stairsY := localRand.Intn(stairsRoom.Height-2) + stairsRoom.Y + 1
	// 階段タイルを配置
	mapGrid[stairsY][stairsX] = Tile{Type: "stairs", Blocked: false, BlockSight: false}

	// Call the newly created functions to generate enemies and items
	enemies := generateEnemies(rooms, playerRoom)
	items := generateItems(rooms)

	return mapGrid, enemies, items, currentFloor + 1, rooms
}

package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
	"math"
	"math/rand"
)

func isInsideRoomOrOnBoundary(x, y int, rooms []Room) bool {
	for _, room := range rooms {
		if x >= room.X && x <= room.X+room.Width-1 &&
			y >= room.Y && y <= room.Y+room.Height-1 {
			return true
		}
	}
	return false
}

func generateCorridorStartPoints(rooms []Room) []Coordinate {
	var corridorStartPoints []Coordinate

	for _, room := range rooms {
		// Top edge
		corridorStartPoints = append(corridorStartPoints, Coordinate{X: room.X + room.Width/2, Y: room.Y})

		// Bottom edge
		corridorStartPoints = append(corridorStartPoints, Coordinate{X: room.X + room.Width/2, Y: room.Y + room.Height - 1})

		// Left edge
		corridorStartPoints = append(corridorStartPoints, Coordinate{X: room.X, Y: room.Y + room.Height/2})

		// Right edge
		corridorStartPoints = append(corridorStartPoints, Coordinate{X: room.X + room.Width - 1, Y: room.Y + room.Height/2})
	}

	return corridorStartPoints
}

func validateAndPlaceDoor(mapGrid [][]Tile, x, y int) {
	// Check adjacent tiles to see if there is a corridor tile
	adjacentCorridor := false
	directions := []Coordinate{{0, -1}, {0, 1}, {-1, 0}, {1, 0}} // Up, Down, Left, Right
	for _, dir := range directions {
		newX, newY := x+dir.X, y+dir.Y
		if mapGrid[newY][newX].Type == "corridor" {
			adjacentCorridor = true
			break
		}
	}

	// If adjacent to a corridor, place a door; otherwise, place a wall
	if adjacentCorridor {
		//mapGrid[y][x] = Tile{Type: "door", Blocked: true, BlockSight: true}
		mapGrid[y][x] = Tile{Type: "corridor", Blocked: false, BlockSight: true}
		//fmt.Printf("Door placed at coordinates (%d, %d)\n", x, y) // Log door position
	} else {
		mapGrid[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
		//fmt.Printf("Wall placed at coordinates (%d, %d) as no adjacent corridor was found\n", x, y) // Log wall position
	}
}

func isCorridor(tile Tile) bool {
	return tile.Type == "corridor"
}

func isCorridorConnected(mapGrid [][]Tile, x1, y1, x2, y2 int) bool {
	// Determine the turning points
	turnX1, turnY1 := x1, (y1+y2)/2
	turnX2, turnY2 := x2, (y1+y2)/2

	// Check vertical corridor from the starting point to the first turning point, excluding the start and end points
	for y := min(y1, turnY1) + 1; y < max(y1, turnY1); y++ {
		for dx := -1; dx <= 1; dx++ {
			neighbor := mapGrid[y][x1+dx]
			if neighbor.Type == "wall" {
				return false // A wall is touching the corridor
			}
		}
	}

	// Check horizontal corridor from the first turning point to the second turning point
	for x := min(turnX1, turnX2) + 1; x < max(turnX1, turnX2); x++ {
		for dy := -1; dy <= 1; dy++ {
			neighbor := mapGrid[turnY1+dy][x]
			if neighbor.Type == "wall" {
				return false // A wall is touching the corridor
			}
		}
	}

	// Check vertical corridor from the second turning point to the end point, excluding the start and end points
	for y := min(turnY2, y2) + 1; y < max(turnY2, y2); y++ {
		for dx := -1; dx <= 1; dx++ {
			neighbor := mapGrid[y][x2+dx]
			if neighbor.Type == "wall" {
				return false // A wall is touching the corridor
			}
		}
	}

	return true // No walls are touching the corridor
}

func drawCorridor(mapGrid [][]Tile, x1, y1, x2, y2 int, rooms []Room, doorPositions []Coordinate) {
	// Determine the turning points
	turnX1, turnY1 := x1, (y1+y2)/2
	turnX2, turnY2 := x2, (y1+y2)/2

	// Check if the corridor can be connected without intersecting walls
	if !isCorridorConnected(mapGrid, x1, y1, x2, y2) {
		fmt.Println("Corridor cannot be connected without intersecting walls")
		return
	}

	// Draw vertical corridor from the starting point to the first turning point
	for y := min(y1, turnY1); y <= max(y1, turnY1); y++ {
		if !isInsideRoomOrOnBoundary(x1, y, rooms) && !isCorridor(mapGrid[y][x1]) {
			mapGrid[y][x1] = Tile{Type: "corridor", Blocked: false, BlockSight: false}
		}
	}

	// Draw horizontal corridor from the first turning point to the second turning point
	for x := min(turnX1, turnX2); x <= max(turnX1, turnX2); x++ {
		if !isInsideRoomOrOnBoundary(x, turnY1, rooms) && !isCorridor(mapGrid[turnY1][x]) {
			mapGrid[turnY1][x] = Tile{Type: "corridor", Blocked: false, BlockSight: false}
		}
	}

	// Draw vertical corridor from the second turning point to the end point
	for y := min(turnY2, y2); y <= max(turnY2, y2); y++ {
		if !isInsideRoomOrOnBoundary(x2, y, rooms) && !isCorridor(mapGrid[y][x2]) {
			mapGrid[y][x2] = Tile{Type: "corridor", Blocked: false, BlockSight: false}
		}
	}
}

func connectRooms(rooms []Room, mapGrid [][]Tile) {
	if len(rooms) == 0 {
		fmt.Println("No rooms to connect")
		return
	}

	var doorPositions []Coordinate
	corridorStartPoints := generateCorridorStartPoints(rooms)

	// Iterate through all corridor start points and try to connect them to each other
	for i, startPoint := range corridorStartPoints {
		for j, endPoint := range corridorStartPoints {
			// Avoid connecting a point to itself
			if i != j {
				isConnectable := isCorridorConnected(mapGrid, startPoint.X, startPoint.Y, endPoint.X, endPoint.Y)
				if isConnectable {
					drawCorridor(mapGrid, startPoint.X, startPoint.Y, endPoint.X, endPoint.Y, rooms, doorPositions)
					// Store door positions for later
					doorPositions = append(doorPositions, startPoint, endPoint)
				}
			}
		}
	}

	for _, pos := range doorPositions {
		validateAndPlaceDoor(mapGrid, pos.X, pos.Y) // Use the new function to validate and place doors
	}

	//fmt.Println("All rooms are connected")
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

func generateEnemies(rooms []Room, playerRoom Room) []Enemy {
	var enemies []Enemy
	for i := 0; i < 10; i++ {
		var enemyRoom Room
		var enemyX, enemyY int
		for {
			enemyRoom = rooms[localRand.Intn(len(rooms))]
			if enemyRoom != playerRoom {
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

		// Randomly select enemy type
		var enemyType, enemyName, enemyChar string
		var enemyAP, enemyDP int
		var enemyHealth, enemyMaxHealth, enemyExperiencePoints int
		if localRand.Intn(2) == 0 { // 50% chance for each type
			enemyType = "Shrimp"
			enemyName = "海老"
			enemyChar = "E"
			enemyAP = 4
			enemyDP = 2
			enemyHealth = 30
			enemyMaxHealth = 30
			enemyExperiencePoints = 5
		} else {
			enemyType = "Snake"
			enemyName = "蛇"
			enemyChar = "S"
			enemyAP = 7
			enemyDP = 1
			enemyHealth = 50
			enemyMaxHealth = 50
			enemyExperiencePoints = 10
		}

		enemies = append(enemies, Enemy{
			Entity:           Entity{X: enemyX, Y: enemyY, Char: rune(enemyChar[0])},
			Health:           enemyHealth,
			MaxHealth:        enemyMaxHealth,
			Name:             enemyName,
			AttackPower:      enemyAP,
			DefensePower:     enemyDP,
			Type:             enemyType,
			ExperiencePoints: enemyExperiencePoints,
			PlayerDiscovered: false,
		})
	}
	return enemies
}

func generateItems(rooms []Room) []Entity {
	var items []Entity
	for i := 0; i < 5; i++ {
		var itemRoom Room
		var itemX, itemY int
		for {
			itemRoom = rooms[localRand.Intn(len(rooms))]
			itemX = localRand.Intn(itemRoom.Width-2) + itemRoom.X + 1
			itemY = localRand.Intn(itemRoom.Height-2) + itemRoom.Y + 1
			occupied := false
			for _, item := range items {
				if item.X == itemX && item.Y == itemY {
					occupied = true
					break
				}
			}
			if !occupied {
				break
			}
		}
		items = append(items, Entity{
			X:    itemX,
			Y:    itemY,
			Char: '!',
		})
	}
	return items
}

func GenerateRandomMap(width, height, currentFloor int, player *Player) ([][]Tile, []Enemy, []Entity, int, []Room) {
	// Step 1: Initialize all tiles to "other" type
	mapGrid := make([][]Tile, height)
	for y := range mapGrid {
		mapGrid[y] = make([]Tile, width)
		for x := range mapGrid[y] {
			mapGrid[y][x] = Tile{Type: "other", Blocked: true, BlockSight: true}
		}
	}

	// Generate a random float between 0 and 1
	randomFloat := rand.Float64()

	// Apply exponential decay
	decayFactor := 0.5 // Adjust this value to control the rate of decay
	prob := math.Pow(randomFloat, decayFactor)

	// Scale and transform to get the number of rooms between 4 and 10
	numRooms := int(prob*7) + 4                              // This will give a value between 4 and 10 with a decreasing probability as the number of rooms increases
	rooms := generateRooms(mapGrid, width, height, numRooms) // Step 2: Generate rooms

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

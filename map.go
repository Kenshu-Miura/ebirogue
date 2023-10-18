package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
	"math"
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
	// Check if the two points have the same x-coordinate (vertical corridor)
	if x1 == x2 {
		for y := min(y1, y2) + 1; y < max(y1, y2); y++ {
			for dx := -1; dx <= 1; dx++ {
				neighbor := mapGrid[y][x1+dx]
				if neighbor.Type == "wall" {
					return false // A wall is touching the corridor
				}
			}
		}
		return true // No walls are touching the corridor
	}

	// Check if the two points have the same y-coordinate (horizontal corridor)
	if y1 == y2 {
		for x := min(x1, x2) + 1; x < max(x1, x2); x++ {
			for dy := -1; dy <= 1; dy++ {
				neighbor := mapGrid[y1+dy][x]
				if neighbor.Type == "wall" {
					return false // A wall is touching the corridor
				}
			}
		}
		return true // No walls are touching the corridor
	}

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

func placeDoor(mapGrid [][]Tile, x, y int) {
	mapGrid[y][x] = Tile{Type: "door", Blocked: false, BlockSight: true}
}

func drawCorridor(mapGrid [][]Tile, room1, room2 Room, rooms []Room) {
	// Get the center coordinates of the rooms
	x1, y1 := room1.Center.X, room1.Center.Y
	x2, y2 := room2.Center.X, room2.Center.Y

	// Determine the turning point
	turnX, turnY := x1, y2

	// Draw vertical corridor from the center of room1 to the turning point
	drawSegment(mapGrid, x1, y1, x1, turnY, rooms)

	// Draw horizontal corridor from the turning point to the center of room2
	drawSegment(mapGrid, turnX, turnY, x2, turnY, rooms)
}

func drawSegment(mapGrid [][]Tile, startX, startY, endX, endY int, rooms []Room) {
	for x := min(startX, endX); x <= max(startX, endX); x++ {
		for y := min(startY, endY); y <= max(startY, endY); y++ {
			isBoundary := false
			for _, room := range rooms {
				if isOnBoundary(x, y, room) {
					isBoundary = true
					placeDoor(mapGrid, x, y)
					break
				}
			}
			if !isBoundary {
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

func removeDoorAtCoord(x, y int, rooms []Room) {
	for i, room := range rooms {
		for j, door := range room.Doors {
			if door.X == x && door.Y == y {
				// Remove the door from the Doors slice
				rooms[i].Doors = append(rooms[i].Doors[:j], rooms[i].Doors[j+1:]...)
				return // Exit function once door is removed
			}
		}
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

func logDoors(rooms []Room) {
	for _, room := range rooms {
		fmt.Printf("Room ID: %d, Doors:\n", room.ID)
		fmt.Printf("  Center: X=%d, Y=%d\n", room.Center.X, room.Center.Y)
		for i, door := range room.Doors {
			fmt.Printf("  Door %d: X=%d, Y=%d\n", i, door.X, door.Y)
		}
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

func setDoorPositions(room *Room) {
	// Top edge
	room.Doors = append(room.Doors, Coordinate{X: room.X + room.Width/2, Y: room.Y})

	// Bottom edge
	room.Doors = append(room.Doors, Coordinate{X: room.X + room.Width/2, Y: room.Y + room.Height - 1})

	// Left edge
	room.Doors = append(room.Doors, Coordinate{X: room.X, Y: room.Y + room.Height/2})

	// Right edge
	room.Doors = append(room.Doors, Coordinate{X: room.X + room.Width - 1, Y: room.Y + room.Height/2})
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
				setDoorPositions(&newRoom) // Set door positions for the new room
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
	room.Center = Coordinate{room.X + room.Width/2, room.Y + room.Height/2}
}

func generateEnemies(rooms []Room, playerRoom Room) []Enemy {
	var enemies []Enemy
	for i := 0; i < 10; i++ {
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

		// Randomly select enemy type
		var enemyType, enemyName, enemyChar string
		var enemyAP, enemyDP int
		var enemyHealth, enemyMaxHealth, enemyExperiencePoints int
		var enemyDirection int
		if localRand.Intn(2) == 0 { // 50% chance for each type
			enemyType = "Shrimp"
			enemyName = "海老"
			enemyChar = "E"
			enemyAP = 4
			enemyDP = 2
			enemyHealth = 30
			enemyMaxHealth = 30
			enemyExperiencePoints = 5
			enemyDirection = Down
		} else {
			enemyType = "Snake"
			enemyName = "蛇"
			enemyChar = "S"
			enemyAP = 7
			enemyDP = 1
			enemyHealth = 50
			enemyMaxHealth = 50
			enemyExperiencePoints = 10
			enemyDirection = Down
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
			Direction:        enemyDirection,
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

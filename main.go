package main

import (
	"fmt"
	"image"
	_ "image/png" // PNG画像を読み込むために必要
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	tileSize = 20 // タイルのサイズを20x20ピクセルに設定
)

type Tile struct {
	Type       string // タイルの種類（例: "floor", "wall", "water" 等）
	Blocked    bool   // タイルが通行可能かどうか
	BlockSight bool   // タイルが視界を遮るかどうか
}

type Entity struct {
	X, Y int  // エンティティの位置
	Char rune // エンティティを表現する文字
}

type Player struct {
	Entity    // PlayerはEntityのフィールドを継承します
	Health    int
	MaxHealth int
}

type Enemy struct {
	Entity    // EnemyはEntityのフィールドを継承します
	Health    int
	MaxHealth int
}

type GameState struct {
	Map     [][]Tile // ゲームのマップ
	Player  Player   // プレイヤーキャラクター
	Enemies []Enemy  // 敵キャラクターのリスト
	Items   []Entity // マップ上のアイテムのリスト
}

type Game struct {
	state      GameState
	playerImg  *ebiten.Image
	enemyImg   *ebiten.Image
	itemImg    *ebiten.Image
	tilesetImg *ebiten.Image
	offsetX    int
	offsetY    int
	moveCount  int
}

var localRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

type Room struct {
	ID            int
	X, Y          int
	Width, Height int
}

func (r Room) Intersects(other Room) bool {
	return r.X < other.X+other.Width && r.X+r.Width > other.X &&
		r.Y < other.Y+other.Height && r.Y+r.Height > other.Y
}

func isCorridor(tile Tile) bool {
	return tile.Type == "corridor"
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

type Coordinate struct {
	X, Y int
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
		mapGrid[y][x] = Tile{Type: "door", Blocked: true, BlockSight: true}
		fmt.Printf("Door placed at coordinates (%d, %d)\n", x, y) // Log door position
	} else {
		mapGrid[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
		fmt.Printf("Wall placed at coordinates (%d, %d) as no adjacent corridor was found\n", x, y) // Log wall position
	}
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

	fmt.Println("All rooms are connected")
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

func GenerateRandomMap(width, height int) ([][]Tile, Player, []Enemy, []Entity) {
	// Step 1: Initialize all tiles to "other" type
	mapGrid := make([][]Tile, height)
	for y := range mapGrid {
		mapGrid[y] = make([]Tile, width)
		for x := range mapGrid[y] {
			mapGrid[y][x] = Tile{Type: "other", Blocked: true, BlockSight: true}
		}
	}

	numRooms := localRand.Intn(6) + 4
	rooms := generateRooms(mapGrid, width, height, numRooms) // Step 2: Generate rooms

	connectRooms(rooms, mapGrid)

	playerRoom := rooms[localRand.Intn(len(rooms))]
	playerX := localRand.Intn(playerRoom.Width-2) + playerRoom.X + 1  // Exclude walls
	playerY := localRand.Intn(playerRoom.Height-2) + playerRoom.Y + 1 // Exclude walls

	player := Player{
		Entity:    Entity{X: playerX, Y: playerY, Char: '@'},
		Health:    100,
		MaxHealth: 100,
	}

	// 敵とアイテムの配列を初期化
	var enemies []Enemy
	var items []Entity

	for i := 0; i < 5; i++ { // ここでは5つの敵と5つのアイテムを生成します
		// ランダムな部屋を選ぶ
		room := rooms[localRand.Intn(len(rooms))]
		// ランダムな位置を選ぶ（壁を避ける）
		enemyX := localRand.Intn(room.Width-2) + room.X + 1
		enemyY := localRand.Intn(room.Height-2) + room.Y + 1
		itemX := localRand.Intn(room.Width-2) + room.X + 1
		itemY := localRand.Intn(room.Height-2) + room.Y + 1

		// 敵とアイテムを配列に追加
		enemies = append(enemies, Enemy{
			Entity:    Entity{X: enemyX, Y: enemyY, Char: 'E'},
			Health:    50,
			MaxHealth: 50,
		})
		items = append(items, Entity{
			X:    itemX,
			Y:    itemY,
			Char: '!',
		})
	}

	return mapGrid, player, enemies, items
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func (g *Game) MovePlayer(dx, dy int) {
	// dx と dy が両方とも0の場合、移動は発生していない
	if dx == 0 && dy == 0 {
		return
	}

	newPX := g.state.Player.X + dx
	newPY := g.state.Player.Y + dy
	// マップ範囲内およびブロックされていないタイル上にあることを確認
	//if newPX >= 0 && newPX < len(g.state.Map[0]) && newPY >= 0 && newPY < len(g.state.Map) && !g.state.Map[newPY][newPX].Blocked {
	g.state.Player.X = newPX
	g.state.Player.Y = newPY
	g.moveCount++ // プレイヤーが移動するたびにカウントを増やす
	//}
}

func (g *Game) Update() error {
	var dx, dy int

	// キーの押下状態を取得
	upPressed := inpututil.IsKeyJustPressed(ebiten.KeyUp)
	downPressed := inpututil.IsKeyJustPressed(ebiten.KeyDown)
	leftPressed := inpututil.IsKeyJustPressed(ebiten.KeyLeft)
	rightPressed := inpututil.IsKeyJustPressed(ebiten.KeyRight)
	aPressed := ebiten.IsKeyPressed(ebiten.KeyA) // Aキーが押されているかどうかをチェック

	if aPressed { // 斜め移動のロジック
		if (upPressed || downPressed) && (leftPressed || rightPressed) {
			if upPressed {
				dy = -1
			}
			if downPressed {
				dy = 1
			}
			if leftPressed {
				dx = -1
			}
			if rightPressed {
				dx = 1
			}
		}
	} else { // 上下左右の移動のロジック
		if upPressed && !downPressed {
			dy = -1
		}
		if downPressed && !upPressed {
			dy = 1
		}
		if leftPressed && !rightPressed {
			dx = -1
		}
		if rightPressed && !leftPressed {
			dx = 1
		}
	}

	g.MovePlayer(dx, dy) // プレイヤーの移動を更新

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// 画面中央の位置を計算
	centerX := (screenWidth-tileSize)/2 - tileSize
	centerY := (screenHeight-tileSize)/2 - tileSize

	// マップのオフセットを計算
	offsetX := centerX - g.state.Player.X*tileSize
	offsetY := centerY - g.state.Player.Y*tileSize

	// タイルの描画
	for y, row := range g.state.Map {
		for x, tile := range row {
			var srcX, srcY int
			switch tile.Type {
			case "wall":
				srcX, srcY = 0, 0 // タイルセット上の壁タイルの位置
			case "corridor":
				srcX, srcY = tileSize, 0 // タイルセット上の床タイルの位置
			case "floor":
				srcX, srcY = 2*tileSize, 0 // タイルセット上の通路タイルの位置
			case "door":
				srcX, srcY = 3*tileSize, 0 // タイルセット上のドアタイルの位置
			default:
				continue // 未知のタイルタイプは描画しない
			}
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(x*tileSize+offsetX), float64(y*tileSize+offsetY))
			screen.DrawImage(g.tilesetImg.SubImage(image.Rect(srcX, srcY, srcX+tileSize, srcY+tileSize)).(*ebiten.Image), opts)
		}
	}

	// プレイヤーを描画
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(centerX), float64(centerY))
	screen.DrawImage(g.playerImg, opts)

	// 敵を描画
	for _, enemy := range g.state.Enemies {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(enemy.X*tileSize+offsetX), float64(enemy.Y*tileSize+offsetY))
		screen.DrawImage(g.enemyImg, opts)
	}

	// アイテムを描画
	for _, item := range g.state.Items {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(item.X*tileSize+offsetX), float64(item.Y*tileSize+offsetY))
		screen.DrawImage(g.itemImg, opts)
	}

	// カウントを画面右上に表示
	countText := fmt.Sprintf("Moves: %d", g.moveCount)
	ebitenutil.DebugPrintAt(screen, countText, screenWidth-100, 10) // Adjust the x-position as needed to align to the right
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 640, 480
}

func main() {
	img, _, err := ebitenutil.NewImageFromFile("img/ebisan.png")
	if err != nil {
		log.Fatalf("failed to load image: %v", err)
	}
	tilesetImg, _, err := ebitenutil.NewImageFromFile("img/tileset.png")
	if err != nil {
		log.Fatalf("failed to load tileset image: %v", err)
	}
	enemyImg, _, err := ebitenutil.NewImageFromFile("img/ebi.png")
	if err != nil {
		log.Fatalf("failed to load enemy image: %v", err)
	}

	itemImg, _, err := ebitenutil.NewImageFromFile("img/kane.png")
	if err != nil {
		log.Fatalf("failed to load item image: %v", err)
	}

	mapGrid, player, enemies, items := GenerateRandomMap(70, 70)

	game := &Game{
		state: GameState{
			Map:     mapGrid,
			Player:  player,
			Enemies: enemies,
			Items:   items,
		},
		playerImg:  img,
		tilesetImg: tilesetImg,
		enemyImg:   enemyImg,
		itemImg:    itemImg,
		offsetX:    0,
		offsetY:    0,
	}

	ebiten.SetWindowSize(1280, 960)
	ebiten.SetWindowTitle("ebirogue")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

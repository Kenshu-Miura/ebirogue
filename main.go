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
	X, Y, Width, Height int
}

func (r Room) Intersects(other Room) bool {
	return r.X < other.X+other.Width && r.X+r.Width > other.X &&
		r.Y < other.Y+other.Height && r.Y+r.Height > other.Y
}

func connectRooms(rooms []Room, mapGrid [][]Tile) {
	for i := 0; i < len(rooms)-1; i++ {
		roomA := rooms[i]
		roomB := rooms[i+1]

		x1, y1 := roomA.X+roomA.Width/2, roomA.Y+roomA.Height/2
		x2, y2 := roomB.X+roomB.Width/2, roomB.Y+roomB.Height/2

		if localRand.Intn(2) == 0 {
			for x := min(x1, x2); x <= max(x1, x2); x++ {
				mapGrid[y1][x] = Tile{Type: "floor", Blocked: false, BlockSight: false}
			}
			for y := min(y1, y2); y <= max(y1, y2); y++ {
				mapGrid[y][x2] = Tile{Type: "floor", Blocked: false, BlockSight: false}
			}
		} else {
			for y := min(y1, y2); y <= max(y1, y2); y++ {
				mapGrid[y][x1] = Tile{Type: "floor", Blocked: false, BlockSight: false}
			}
			for x := min(x1, x2); x <= max(x1, x2); x++ {
				mapGrid[y2][x] = Tile{Type: "floor", Blocked: false, BlockSight: false}
			}
		}
	}
}

func GenerateRandomMap(width, height int) ([][]Tile, Player, []Enemy, []Entity) {
	mapGrid := make([][]Tile, height)
	for y := range mapGrid {
		mapGrid[y] = make([]Tile, width)
		for x := range mapGrid[y] {
			mapGrid[y][x] = Tile{Type: "wall", Blocked: true, BlockSight: true}
		}
	}

	var rooms []Room

	for i := 0; i < 10; i++ { // Attempt to create 10 rooms
		for attempt := 0; attempt < 10; attempt++ { // Limit of 10 attempts per room
			roomWidth := localRand.Intn(10) + 5  // Random width between 5 and 15
			roomHeight := localRand.Intn(10) + 5 // Random height between 5 and 15
			roomX := localRand.Intn(width-roomWidth-1) + 1
			roomY := localRand.Intn(height-roomHeight-1) + 1

			newRoom := Room{roomX, roomY, roomWidth, roomHeight}
			valid := true
			for _, room := range rooms {
				if newRoom.Intersects(room) {
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
			}
		}
	}

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
	if newPX >= 0 && newPX < len(g.state.Map[0]) && newPY >= 0 && newPY < len(g.state.Map) && !g.state.Map[newPY][newPX].Blocked {
		g.state.Player.X = newPX
		g.state.Player.Y = newPY
		g.moveCount++ // プレイヤーが移動するたびにカウントを増やす
	}
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
			case "floor":
				srcX, srcY = tileSize, 0 // タイルセット上の床タイルの位置
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

	mapGrid, player, enemies, items := GenerateRandomMap(100, 100)

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

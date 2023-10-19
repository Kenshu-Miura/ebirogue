package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png" // PNG画像を読み込むために必要
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const (
	tileSize      = 20 // タイルのサイズを20x20ピクセルに設定
	Uninitialized = -1
	Up            = 0
	Down          = 1
	Left          = 2
	Right         = 3
	UpRight       = 4
	DownRight     = 5
	UpLeft        = 6
	DownLeft      = 7
)

var localRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
var mplusNormalFont font.Face
var levelExpRequirements = []int{0, 5, 12, 22, 35, 51, 70, 92, 118, 148, 181} // レベル10までの経験値要件

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
	Entity           // PlayerはEntityのフィールドを継承します
	Health           int
	MaxHealth        int
	AttackPower      int    // 攻撃力
	DefensePower     int    // 防御力
	Power            int    // プレイヤーのパワー
	MaxPower         int    // プレイヤーの最大パワー
	Satiety          int    // 満腹度
	MaxSatiety       int    // 最大満腹度
	Inventory        []Item // 所持アイテム
	MaxInventory     int    // 最大所持アイテム数
	ExperiencePoints int    // 所持経験値
	Level            int    // プレイヤーのレベル
	Direction        int    // Uninitialized: uninitialized, Up: Up, Down: Down, Left: Left, Right: Right, UpRight: UpRight, DownRight: DownRight, UpLeft: UpLeft, DownLeft: DownLeft
}

type Item struct {
	Entity
	Type        string
	Name        string
	Description string
	// 他のアイテムに関連するフィールドもここに追加できます。
}

type Enemy struct {
	Entity           // Enemy inherits fields from Entity
	Name             string
	Health           int
	MaxHealth        int
	AttackPower      int    // Attack power
	DefensePower     int    // Defense power
	Type             string // Type of enemy (e.g., "orc", "goblin", "slime", etc.)
	ExperiencePoints int    // Experience points enemy holds
	PlayerDiscovered bool   // Whether the enemy has discovered the player
	Direction        int    // Uninitialized: uninitialized, Up: Up, Down: Down, Left: Left, Right: Right, UpRight: UpRight, DownRight: DownRight, UpLeft: UpLeft, DownLeft: DownLeft
}

type Room struct {
	ID            int
	X, Y          int
	Width, Height int
	Center        Coordinate
}
type Coordinate struct {
	X, Y int
}
type GameState struct {
	Map     [][]Tile // ゲームのマップ
	Player  Player   // プレイヤーキャラクター
	Enemies []Enemy  // 敵キャラクターのリスト
	Items   []Item   // マップ上のアイテムのリスト
}

type Game struct {
	state          GameState
	rooms          []Room
	playerImg      *ebiten.Image
	ebiImg         *ebiten.Image
	snakeImg       *ebiten.Image
	kaneImg        *ebiten.Image
	cardImg        *ebiten.Image
	mintiaImg      *ebiten.Image
	sausageImg     *ebiten.Image
	tilesetImg     *ebiten.Image
	offsetX        int
	offsetY        int
	moveCount      int
	Floor          int
	lastIncrement  time.Time
	lastArrowPress time.Time // 矢印キーが最後に押された時間を追跡
	showInventory  bool      // true when the inventory window should be displayed
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

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// sign function returns the sign of an integer.
func sign(x int) int {
	if x > 0 {
		return 1
	} else if x < 0 {
		return -1
	}
	return 0
}

func init() {
	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	mplusNormalFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    16,
		DPI:     dpi,
		Hinting: font.HintingVertical,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func (g *Game) checkForStairs() {
	player := &g.state.Player
	playerTile := g.state.Map[player.Y][player.X]

	if playerTile.Type == "stairs" {
		mapGrid, enemies, items, newFloor, newRoom := GenerateRandomMap(70, 70, g.Floor, player)
		g.state.Map = mapGrid
		g.state.Enemies = enemies
		g.state.Items = items
		g.Floor = newFloor
		g.rooms = newRoom
	}
}

func (g *Game) IncrementMoveCount() {
	g.moveCount++
	// Check if moveCount has increased by 5
	if g.moveCount%5 == 0 && g.moveCount != 0 {
		// Recover 1 HP for the player
		g.state.Player.Health += 1
		// Ensure player's health does not exceed MaxHealth
		if g.state.Player.Health > g.state.Player.MaxHealth {
			g.state.Player.Health = g.state.Player.MaxHealth
		}
	}
	// Existing satiety reduction logic
	if g.moveCount%10 == 0 && g.moveCount != 0 {
		g.state.Player.Satiety -= 1
		if g.state.Player.Satiety < 0 {
			g.state.Player.Satiety = 0
		}
	}
}

func (g *Game) PickupItem() {
	playerX, playerY := g.state.Player.X, g.state.Player.Y // プレイヤーの座標を取得

	for i, item := range g.state.Items { // GameStateの全てのアイテムに対してループ
		if item.X == playerX && item.Y == playerY { // アイテムの座標とプレイヤーの座標が一致するかチェック
			g.state.Player.Inventory = append(g.state.Player.Inventory, item) // アイテムをプレイヤーのインベントリに追加

			// アイテムをGameState.Itemsから削除
			g.state.Items = append(g.state.Items[:i], g.state.Items[i+1:]...)

			break // 一致するアイテムが見つかったらループを終了
		}
	}
}

func (g *Game) ToggleInventory() {
	g.showInventory = !g.showInventory
}

func (g *Game) Update() error {

	cPressed := inpututil.IsKeyJustPressed(ebiten.KeyC)
	if cPressed {
		g.ToggleInventory()
		return nil // Skip other updates when the inventory window is active
	}

	if g.showInventory {
		return nil // Skip other updates when the inventory window is active
	}

	dx, dy := g.HandleInput()
	//dx, dy := g.CheetHandleInput()

	moved := g.MovePlayer(dx, dy) // プレイヤーの移動を更新
	//moved := g.CheetMovePlayer(dx, dy) // プレイヤーの移動を更新

	if moved {
		g.MoveEnemies()
	}

	// 扉を開く処理の追加
	spacePressed := inpututil.IsKeyJustPressed(ebiten.KeySpace) // Spaceキーをチェック
	if spacePressed {
		g.OpenDoor()
	}

	g.PickupItem()

	g.checkForStairs()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	centerX := (screenWidth-tileSize)/2 - tileSize
	centerY := (screenHeight-tileSize)/2 - tileSize
	offsetX := centerX - g.state.Player.X*tileSize
	offsetY := centerY - g.state.Player.Y*tileSize

	g.DrawMap(screen, offsetX, offsetY)
	g.DrawItems(screen, offsetX, offsetY)
	g.DrawEnemies(screen, offsetX, offsetY)
	g.DrawHUD(screen)
	g.DrawPlayer(screen, centerX, centerY)

	// Draw the inventory window if the showInventory flag is set
	if g.showInventory {
		windowWidth, windowHeight := 400, 300
		windowX, windowY := (screenWidth-windowWidth)/2, (screenHeight-windowHeight)/2

		// Draw window background
		windowBackground := ebiten.NewImage(windowWidth, windowHeight)
		windowBackground.Fill(color.RGBA{0, 0, 0, 255})
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(windowX), float64(windowY))
		screen.DrawImage(windowBackground, opts)

		// Draw window border
		borderSize := 2
		borderColor := color.RGBA{255, 255, 255, 255}

		borderImg := ebiten.NewImage(screenWidth, screenHeight)
		borderImg.Fill(borderColor)

		// Drawing options for border
		borderOpts := &ebiten.DrawImageOptions{}

		// Top border
		borderOpts.GeoM.Reset()
		borderOpts.GeoM.Translate(float64(windowX-borderSize), float64(windowY-borderSize))
		screen.DrawImage(borderImg.SubImage(image.Rect(0, 0, windowWidth+2*borderSize, borderSize)).(*ebiten.Image), borderOpts)

		// Left border
		borderOpts.GeoM.Reset()
		borderOpts.GeoM.Translate(float64(windowX-borderSize), float64(windowY))
		screen.DrawImage(borderImg.SubImage(image.Rect(0, 0, borderSize, windowHeight)).(*ebiten.Image), borderOpts)

		// Right border
		borderOpts.GeoM.Reset()
		borderOpts.GeoM.Translate(float64(windowX+windowWidth), float64(windowY))
		screen.DrawImage(borderImg.SubImage(image.Rect(0, 0, borderSize, windowHeight)).(*ebiten.Image), borderOpts)

		// Bottom border
		borderOpts.GeoM.Reset()
		borderOpts.GeoM.Translate(float64(windowX-borderSize), float64(windowY+windowHeight))
		screen.DrawImage(borderImg.SubImage(image.Rect(0, 0, windowWidth+2*borderSize, borderSize)).(*ebiten.Image), borderOpts)

		// Draw items
		for i, item := range g.state.Player.Inventory {
			itemText := fmt.Sprintf("%d. %s", i+1, item.Name)
			text.Draw(screen, itemText, mplusNormalFont, windowX+10, windowY+20+(i*20), color.White)
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 640, 480
}

// loadImage is a helper function to load an image from a file.
func loadImage(filepath string) *ebiten.Image {
	img, _, err := ebitenutil.NewImageFromFile(filepath)
	if err != nil {
		log.Fatalf("failed to load image from %s: %v", filepath, err)
	}
	return img
}

// NewGame function initializes a new game and returns a pointer to a Game object.
func NewGame() *Game {
	img := loadImage("img/ebisan.png")
	tilesetImg := loadImage("img/tileset.png")
	ebiImg := loadImage("img/ebi.png")
	kaneImg := loadImage("img/kane.png")
	snakeImg := loadImage("img/snake.png")
	cardImg := loadImage("img/card.png")
	sausageImg := loadImage("img/sausage.png")
	mintiaImg := loadImage("img/mintia.png")

	// プレイヤーの初期化
	player := Player{
		Entity:           Entity{Char: '@'},
		Health:           100,
		MaxHealth:        100,
		Satiety:          100,
		MaxSatiety:       100,
		Inventory:        []Item{},
		MaxInventory:     20,
		AttackPower:      3,
		DefensePower:     3,
		ExperiencePoints: 0,
		Level:            1,
		Power:            8,
		MaxPower:         8,
		Direction:        Down,
	}

	// 最初のマップを生成
	mapGrid, enemies, items, newFloor, newRoom := GenerateRandomMap(70, 70, 0, &player) // 初期階層は1です

	game := &Game{
		state: GameState{
			Map:     mapGrid,
			Player:  player,
			Enemies: enemies,
			Items:   items,
		},
		rooms:      newRoom,
		playerImg:  img,
		tilesetImg: tilesetImg,
		ebiImg:     ebiImg,
		snakeImg:   snakeImg,
		kaneImg:    kaneImg,
		cardImg:    cardImg,
		mintiaImg:  mintiaImg,
		sausageImg: sausageImg,
		offsetX:    0,
		offsetY:    0,
		Floor:      newFloor,
	}

	// Log the contents of game.rooms
	//for i, room := range game.rooms {
	//	log.Printf("Room %d: %+v\n", i, room)
	//}

	return game
}

func main() {
	game := NewGame()

	ebiten.SetWindowSize(1280, 960)
	ebiten.SetWindowTitle("ebirogue")
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

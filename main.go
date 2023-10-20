package main

import (
	_ "image/png" // PNG画像を読み込むために必要
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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
	state               GameState
	rooms               []Room
	playerImg           *ebiten.Image
	ebiImg              *ebiten.Image
	snakeImg            *ebiten.Image
	kaneImg             *ebiten.Image
	cardImg             *ebiten.Image
	mintiaImg           *ebiten.Image
	sausageImg          *ebiten.Image
	tilesetImg          *ebiten.Image
	offsetX             int
	offsetY             int
	moveCount           int
	Floor               int
	lastIncrement       time.Time
	lastArrowPress      time.Time // 矢印キーが最後に押された時間を追跡
	showInventory       bool      // true when the inventory window should be displayed
	selectedItemIndex   int
	showItemActions     bool
	selectedActionIndex int
	showDescription     bool
	showItemDescription bool
	itemdescriptionText string
	descriptionText     string
	descriptionQueue    []string
	nextDescriptionTime time.Time
	Animating           bool
	AnimationProgress   float64
	dx, dy              int
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

func (g *Game) Update() error {
	log.Printf("Update start: AnimationProgress=%v, Animating=%v, dx=%v, dy=%v\n", g.AnimationProgress, g.Animating, g.dx, g.dy) // Logging added

	err := g.handleInventoryInput()
	if err != nil {
		return err
	}

	if !g.showInventory {
		dx, dy := g.HandleInput()

		moved := g.MovePlayer(dx, dy)

		if moved {
			g.MoveEnemies()
			g.Animating = true  // Set the animating flag
			g.dx, g.dy = dx, dy // Save the direction of movement
		}

		// 扉を開く処理の追加
		spacePressed := inpututil.IsKeyJustPressed(ebiten.KeySpace) // Spaceキーをチェック
		if spacePressed {
			g.OpenDoor()
		}
	}

	if g.Animating {
		g.AnimationProgress += 0.1 // Update the animation progress
		if g.AnimationProgress >= 1 {
			g.Animating = false
			g.AnimationProgress = 0
		}
	}

	// メッセージキューを管理する
	g.ManageDescriptions()

	g.checkForStairs()

	log.Printf("Update end: AnimationProgress=%v, Animating=%v, dx=%v, dy=%v\n", g.AnimationProgress, g.Animating, g.dx, g.dy) // Logging added

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	centerX := (screenWidth-tileSize)/2 - tileSize
	centerY := (screenHeight-tileSize)/2 - tileSize
	// Calculate the offsets based on the animation progress and direction
	offsetX := centerX - g.state.Player.X*tileSize - int(float64(g.dx)*g.AnimationProgress*2)*tileSize
	offsetY := centerY - g.state.Player.Y*tileSize - int(float64(g.dy)*g.AnimationProgress*2)*tileSize

	log.Printf("Draw: offsetX=%v, offsetY=%v\n", offsetX, offsetY) // Logging added

	g.DrawMap(screen, offsetX, offsetY)
	g.DrawItems(screen, offsetX, offsetY)
	g.DrawEnemies(screen, offsetX, offsetY)
	g.DrawHUD(screen)
	g.DrawPlayer(screen, centerX, centerY)

	// Draw the inventory window if the showInventory flag is set
	if g.showInventory {
		g.showDescription = false
		if err := g.drawInventoryWindow(screen); err != nil {
			log.Printf("Error drawing inventory window: %v", err)
		}
	}

	g.drawActionMenu(screen)

	g.drawItemDescription(screen)

	g.DrawDescriptions(screen)
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
		Direction:        Up,
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

package main

import (
	_ "image/png" // PNG画像を読み込むために必要
	"log"
	"math"
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
	tileSize      = 30 // タイルのサイズを20x20ピクセルに設定
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
	EquippedWeapon   *Weapon
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

type Attack struct {
	EnemyIndex         int
	Attackdx, Attackdy int
	IsPlayer           bool
	NetDamage          int
	EnemyName          string
}

type Action struct {
	Duration float64     // 行動を処理する時間
	Message  string      // 画面下に表示するメッセージ
	Execute  func(*Game) // 行動を実行する関数
}

type ActionQueue struct {
	Queue []Action
}

type Direction int

type ThrownItem struct {
	Item   Item
	Image  *ebiten.Image
	X, Y   int // 投げられたアイテムの現在の位置
	DX, DY int // アイテムの移動方向と速度
}

type Game struct {
	state                   GameState
	rooms                   []Room
	playerImg               *ebiten.Image
	ebiImg                  *ebiten.Image
	snakeImg                *ebiten.Image
	kaneImg                 *ebiten.Image
	cardImg                 *ebiten.Image
	mintiaImg               *ebiten.Image
	sausageImg              *ebiten.Image
	tilesetImg              *ebiten.Image
	weaponImg               *ebiten.Image
	offsetX                 int
	offsetY                 int
	moveCount               int
	Floor                   int
	lastIncrement           time.Time
	lastArrowPress          time.Time // 矢印キーが最後に押された時間を追跡
	showInventory           bool      // true when the inventory window should be displayed
	selectedItemIndex       int
	showItemActions         bool
	selectedActionIndex     int
	showDescription         bool
	showItemDescription     bool
	itemdescriptionText     string
	descriptionText         string
	nextDescriptionTime     time.Time
	Animating               bool
	AnimationProgress       float64
	dx, dy                  int
	AnimationProgressInt    int
	frameCount              int
	tmpPlayerOffsetX        float64 // プレイヤーの一時的なオフセットX
	tmpPlayerOffsetY        float64 // プレイヤーの一時的なオフセットY
	attackTimer             float64 // 攻撃メッセージのタイマー
	playerAttack            bool    // プレイヤーが攻撃したかどうか
	ActionQueue             ActionQueue
	isCombatActive          bool
	ActionDurationCounter   float64
	isActioned              bool
	zPressed                bool
	xPressed                bool
	ShowGroundItem          bool
	selectedGroundItemIndex int
	GroundItemActioned      bool
	isFrontEnemy            bool
	currentGroundItem       Item
	ThrownItem              ThrownItem
	ThrownItemDestination   Coordinate
	TargetEnemy             *Enemy
	TargetEnemyIndex        int
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

	if !g.showInventory && !g.playerAttack && !g.isCombatActive && !g.ShowGroundItem {
		dx, dy := g.HandleInput()
		//dx, dy := g.CheetHandleInput()

		if g.zPressed && !g.ShowGroundItem {
			g.CheckForEnemies(dx, dy)
			g.zPressed = false
			return nil
		}

		moved := g.MovePlayer(dx, dy)
		//moved := g.CheetMovePlayer(dx, dy)

		if moved {
			g.isActioned = true
			g.Animating = true  // Set the animating flag
			g.xPressed = false  // Reset the xPressed flag
			g.dx, g.dy = dx, dy // Save the direction of movement
		}

		// 扉を開く処理の追加
		spacePressed := inpututil.IsKeyJustPressed(ebiten.KeySpace) // Spaceキーをチェック
		if spacePressed {
			g.OpenDoor()
		}
	}

	// Find item at player's position
	playerX, playerY := g.state.Player.X, g.state.Player.Y
	for _, item := range g.state.Items {
		itemX, itemY := item.GetPosition()
		if itemX == playerX && itemY == playerY {
			g.currentGroundItem = item // Assuming g.currentGroundItem is a field of *Game
			break
		} else {
			g.currentGroundItem = nil
		}
	}

	err := g.handleInventoryInput()
	if err != nil {
		return err
	}

	g.HandleGroundItemInput()

	g.HandleAnimationProgress()

	// Check the attack timer and reset temporary player position if needed
	if g.attackTimer > 0 {
		progress := 1 - g.attackTimer/0.5 // progress ranges from 0 to 1 over 0.5 seconds
		angle := math.Pi * progress       // angle ranges from 0 to Pi
		value := 30 * math.Sin(angle)     // value ranges from 0 to 20 to 0

		g.tmpPlayerOffsetX = value
		g.tmpPlayerOffsetY = value

		g.attackTimer -= (1 / 60.0) // assuming Update is called 60 times per second
		if g.attackTimer <= 0 {
			g.attackTimer = 0 // reset timer
			g.tmpPlayerOffsetX = 0
			g.tmpPlayerOffsetY = 0
			g.playerAttack = false
		}
	}

	if g.ThrownItem.Image != nil {
		g.ThrownItem.X += g.ThrownItem.DX
		g.ThrownItem.Y += g.ThrownItem.DY
		// 必要に応じてアイテムが目的地に到達したかどうかをチェックし、到達したらリストから削除
		if (g.ThrownItem.DX >= 0 && g.ThrownItem.X*tileSize >= g.ThrownItemDestination.X*tileSize) || (g.ThrownItem.DX < 0 && g.ThrownItem.X*tileSize <= g.ThrownItemDestination.X*tileSize) {
			if (g.ThrownItem.DY >= 0 && g.ThrownItem.Y*tileSize >= g.ThrownItemDestination.Y*tileSize) || (g.ThrownItem.DY < 0 && g.ThrownItem.Y*tileSize <= g.ThrownItemDestination.Y*tileSize) {
				if g.TargetEnemy != nil {
					// 敵にアイテムが当たった時の処理を実行
					g.hitEnemyWithItem()
					g.TargetEnemy = nil
				} else {
					g.state.Items = append(g.state.Items, g.ThrownItem.Item)
				}
				g.ThrownItem = ThrownItem{}
				g.ThrownItemDestination = Coordinate{}
			}
		}

	}

	g.HandleEnemyAttackTimers()

	g.ManageDescriptions()

	g.HandleActionQueue()

	g.CheckCombatState()

	g.checkForStairs()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	centerX := (screenWidth-tileSize)/2 - tileSize
	centerY := (screenHeight-tileSize)/2 - tileSize

	offsetX, offsetY := g.CalculateAnimationOffset(screen)

	g.DrawMap(screen, offsetX, offsetY)
	g.DrawItems(screen, offsetX, offsetY)
	g.DrawThrownItem(screen, offsetX, offsetY)
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

	g.DrawGroundItem(screen)
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
	weaponImg := loadImage("img/weapon.png")

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
		rooms:            newRoom,
		playerImg:        img,
		tilesetImg:       tilesetImg,
		ebiImg:           ebiImg,
		snakeImg:         snakeImg,
		kaneImg:          kaneImg,
		cardImg:          cardImg,
		mintiaImg:        mintiaImg,
		sausageImg:       sausageImg,
		weaponImg:        weaponImg,
		offsetX:          0,
		offsetY:          0,
		Floor:            newFloor,
		frameCount:       0,
		tmpPlayerOffsetX: 0,
		tmpPlayerOffsetY: 0,
		playerAttack:     false,
		ActionQueue: ActionQueue{
			Queue: make([]Action, 0),
		},
		isCombatActive: false,
		zPressed:       false,
	}

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

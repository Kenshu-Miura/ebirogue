package main

import (
	_ "image/png" // PNG画像を読み込むために必要
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	tileSize      = 30 // タイルのサイズを30x30ピクセルに設定
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
var levelExpRequirements = []int{0, 5, 12, 22, 35, 51, 70, 92, 118, 148, 181} // レベル10までの経験値要件

type Tile struct {
	Type       string // タイルの種類（例: "floor", "wall", "water" 等）
	Blocked    bool   // タイルが通行可能かどうか
	BlockSight bool   // タイルが視界を遮るかどうか
	Visited    bool   // プレイヤーがこのタイルを通過したかどうか
	Brightness float64
}

type Entity struct {
	X, Y int  // エンティティの位置
	Char rune // エンティティを表現する文字
}

type Player struct {
	Name             string
	Entity           // PlayerはEntityのフィールドを継承します
	Health           int
	MaxHealth        int
	AttackPower      int       // 攻撃力
	DefensePower     int       // 防御力
	Power            int       // プレイヤーのパワー
	MaxPower         int       // プレイヤーの最大パワー
	Satiety          int       // 満腹度
	MaxSatiety       int       // 最大満腹度
	Inventory        []Item    // 所持アイテム
	MaxInventory     int       // 最大所持アイテム数
	ExperiencePoints int       // 所持経験値
	Level            int       // プレイヤーのレベル
	Direction        Direction // Uninitialized: uninitialized, Up: Up, Down: Down, Left: Left, Right: Right, UpRight: UpRight, DownRight: DownRight, UpLeft: UpLeft, DownLeft: DownLeft
	EquippedItems    [5]Item   // Array to hold equipped items
	Cash             int       // 所持金
	SetTrap          Item      // トラップを設置する
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
	Duration     float64     // 行動を処理する時間
	Message      string      // 画面下に表示するメッセージ
	ItemName     string      // アイテム名を追加
	Execute      func(*Game) // 行動を実行する関数
	IsIdentified bool
	NonBlocking  bool // 入力を妨げないアクションかどうか
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
	state                     GameState
	rooms                     []Room
	playerImg                 *ebiten.Image
	ebiImg                    *ebiten.Image
	snakeImg                  *ebiten.Image
	kaneImg                   *ebiten.Image
	cardImg                   *ebiten.Image
	mintiaImg                 *ebiten.Image
	sausageImg                *ebiten.Image
	tilesetImg                *ebiten.Image
	weaponImg                 *ebiten.Image
	armorImg                  *ebiten.Image
	arrowImg                  *ebiten.Image
	caneImg                   *ebiten.Image
	effectImg                 *ebiten.Image
	accessoryImg              *ebiten.Image
	offsetX                   int
	offsetY                   int
	moveCount                 int
	Floor                     int
	lastIncrement             time.Time
	lastArrowPress            time.Time // 矢印キーが最後に押された時間を追跡
	showInventory             bool      // true when the inventory window should be displayed
	selectedItemIndex         int
	showItemActions           bool
	selectedActionIndex       int
	showDescription           bool
	descriptionText           string
	showItemDescription       bool
	itemdescriptionText       string
	Animating                 bool
	AnimationProgress         float64
	dx, dy                    int
	AnimationProgressInt      int
	frameCount                int
	tmpPlayerOffsetX          float64 // プレイヤーの一時的なオフセットX
	tmpPlayerOffsetY          float64 // プレイヤーの一時的なオフセットY
	attackTimer               float64 // 攻撃メッセージのタイマー
	ActionQueue               ActionQueue
	isCombatActive            bool
	ActionDurationCounter     float64
	isActioned                bool
	zPressed                  bool
	xPressed                  bool
	dPressed                  bool
	ShowGroundItem            bool
	selectedGroundActionIndex int
	selectedGroundItemIndex   int
	GroundItemActioned        bool
	isFrontEnemy              bool
	currentGroundItem         Item
	ThrownItem                ThrownItem
	ThrownItemDestination     Coordinate
	TargetEnemy               *Enemy
	TargetEnemyIndex          int
	showStairsPrompt          bool
	selectedOption            int // 0 for "Proceed", 1 for "Cancel"
	ignoreStairs              bool
	miniMap                   *ebiten.Image // ミニマップのキャッシュ
	miniMapDirty              bool          // ミニマップが更新される必要があるかどうかを示すフラグ
	prevPlayerX, prevPlayerY  int           // 前のフレームのプレイヤーの座標
	fadingOut                 bool
	fadingIn                  bool
	fadeAlpha                 float64 // 0.0（透明）から1.0（完全な不透明）の間の値
	frameCounter              int
	enemyYOffset              int
	enemyYOffsetTimer         int
	useidentifyItem           bool
	tmpselectedItemIndex      int
}

// CanAcceptInput returns true if the current queued action allows player input.
func (g *Game) CanAcceptInput() bool {
	if len(g.ActionQueue.Queue) == 0 {
		return true
	}
	return g.ActionQueue.Queue[0].NonBlocking
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

func (g *Game) Update() error {

	if !g.showInventory && g.CanAcceptInput() && !g.ShowGroundItem && !g.showStairsPrompt {
		dx, dy := g.HandleInput()
		//dx, dy := g.CheatHandleInput()

		if g.zPressed && !g.ShowGroundItem {
			g.CheckForEnemies(dx, dy)
			g.zPressed = false
			return nil
		}

		moved := g.MovePlayer(dx, dy)
		//moved := g.CheatMovePlayer(dx, dy)

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

	g.processDKeyPress()

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

	g.MarkVisitedTiles(playerX, playerY)
	g.MarkRoomVisited(playerX, playerY)
	g.CheckPlayerMovement()

	g.updateItemVisibility()
	g.updateEnemyVisibility()

	err := g.handleInventoryInput()
	if err != nil {
		return err
	}

	g.HandleGroundItemInput()

	g.HandleAnimationProgress()

	g.UpdateAttackTimer()

	g.UpdateThrownItem()

	g.updateEnemyYOffset()

	g.HandleEnemyAttackTimers()

	g.ManageDescriptions()

	g.HandleActionQueue()

	g.CheckCombatState()

	g.updateTileBrightness()

	g.checkForStairs()
	g.handleStairsPrompt()
	g.ResetStairsIgnoreFlag()

	// 暗転処理
	if g.fadingOut {
		g.handleFadingOut()
	}
	// 明転処理
	if g.fadingIn {
		g.handleFadingIn()
	}

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

	if g.useidentifyItem {
		g.drawUseIdentifyItemWindow(screen)
	}

	g.drawActionMenu(screen)

	g.drawItemDescription(screen)

	g.DrawDescriptions(screen)

	g.DrawGroundItem(screen)

	g.DrawStairsPrompt(screen)

	g.UpdateAndDrawMiniMap(screen)

	if g.fadeAlpha > 0 {
		g.drawOverlay(screen)
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
	weaponImg := loadImage("img/weapon.png")
	armorImg := loadImage("img/armor.png")
	arrowImg := loadImage("img/arrow.png")
	caneImg := loadImage("img/cane.png")
	effectImg := loadImage("img/effect.png")
	accessoryImg := loadImage("img/ring.png")

	// プレイヤーの初期化
	player := Player{
		Name:             "海老さん",
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
		Cash:             0,
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
		armorImg:         armorImg,
		arrowImg:         arrowImg,
		caneImg:          caneImg,
		effectImg:        effectImg,
		accessoryImg:     accessoryImg,
		offsetX:          0,
		offsetY:          0,
		Floor:            newFloor,
		frameCount:       0,
		tmpPlayerOffsetX: 0,
		tmpPlayerOffsetY: 0,
		ActionQueue: ActionQueue{
			Queue: make([]Action, 0),
		},
		isCombatActive:       false,
		zPressed:             false,
		tmpselectedItemIndex: -1,
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

//go:build !test

package main

import (
	"image/color"
	_ "image/png" // PNG画像を読み込むために必要
	"log"
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

// Go 1.20+では、グローバルなrand関数が自動的にシードされるため、手動シードは不要
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

// 状態異常を管理する構造体
type StatusAilments struct {
	Confusion   int  // 混乱の残りターン数
	Sleep       int  // 睡眠の残りターン数
	Blind       int  // 目潰しの残りターン数
	Poison      int  // 毒の残りターン数
	Slow        int  // 鈍足の残りターン数
	Haste       int  // 倍速の残りターン数
	Paralysis   bool // かなしばり状態
	Seal        bool // 封印状態
	HasteOnWake bool // 睡眠から目覚めた時に倍速化するかどうか
}

type Player struct {
	Name                string
	Entity              // PlayerはEntityのフィールドを継承します
	Health              int
	MaxHealth           int
	AttackPower         int            // 攻撃力
	DefensePower        int            // 防御力
	Power               int            // プレイヤーのパワー
	MaxPower            int            // プレイヤーの最大パワー
	Satiety             int            // 満腹度
	MaxSatiety          int            // 最大満腹度
	Inventory           []Item         // 所持アイテム
	MaxInventory        int            // 最大所持アイテム数
	ExperiencePoints    int            // 所持経験値
	Level               int            // プレイヤーのレベル
	Direction           Direction      // Uninitialized: uninitialized, Up: Up, Down: Down, Left: Left, Right: Right, UpRight: UpRight, DownRight: DownRight, UpLeft: UpLeft, DownLeft: DownLeft
	EquippedWeapon      *Weapon        // 装備中の武器（1個まで）
	EquippedArmor       *Armor         // 装備中の防具（1個まで）
	EquippedArrow       *Arrow         // 装備中の矢（1個まで）
	EquippedAccessories [2]*Accessory  // 装備中のアクセサリー（2個まで）
	EquippedItems       [5]Item        // Array to hold equipped items（後方互換のため残す）
	Cash                int            // 所持金
	SetTrap             Item           // トラップを設置する
	StatusAilments      StatusAilments // 状態異常
}

type Coordinate struct {
	X, Y int
}
type GameState struct {
	Map      [][]Tile  // ゲームのマップ
	Player   Player    // プレイヤーキャラクター
	Enemies  []Enemy   // 敵キャラクターのリスト
	Items    []Item    // マップ上のアイテムのリスト
	MapTraps []MapTrap // マップ上の罠のリスト
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
	NonBlocking  bool
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
	mamuruImg                 *ebiten.Image
	honeyImg                  *ebiten.Image
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
	hatenaImg                 *ebiten.Image
	sleepTrapImg              *ebiten.Image
	poisonArrowTrapImg        *ebiten.Image
	slowTrapImg               *ebiten.Image
	mineTrapImg               *ebiten.Image
	offsetX                   int
	offsetY                   int
	moveCount                 int
	Floor                     int
	lastIncrement             time.Time
	lastArrowPress            time.Time // 矢印キーが最後に押された時間を追跡
	lastDashStop              time.Time // 最後にダッシュが停止した時間
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
	dashStopped               bool // ダッシュ停止状態
	dPressed                  bool
	ShowGroundItem            bool
	selectedGroundActionIndex int
	selectedGroundItemIndex   int
	GroundItemActioned        bool
	isFrontEnemy              bool
	currentGroundItem         Item
	showGroundItemDescription bool
	groundItemDescriptionText string
	ThrownItem                ThrownItem
	ThrownItemDestination     Coordinate
	TargetEnemy               *Enemy
	TargetEnemyIndex          int
	playerDead                bool    // プレイヤーが死亡したかどうか
	deathMessageAdded         bool    // 死亡メッセージが追加済みかどうか
	fadeOutProgress           float64 // フェードアウトの進行度 (0.0 - 1.0)
	fadeInProgress            float64 // フェードインの進行度 (0.0 - 1.0)
	gameResetTimer            float64 // ゲームリセットまでのタイマー
	starvationBlinkTimer      float64 // 満腹度0時の点滅タイマー
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
	useIdentifyItem           bool
	tmpSelectedItemIndex      int
	// モンスター湧きシステム
	turnCount     int // プレイヤーのターン数
	lastSpawnTurn int // 最後にモンスターが湧いたターン
	spawnInterval int // 次回湧きまでのターン数
	// フロア滞在時間システム
	floorTurns        int  // 現在のフロアでの滞在ターン数
	windWarning1Shown bool // 1200ターン警告表示済みフラグ
	windWarning2Shown bool // 1300ターン警告表示済みフラグ
	// フロア効果システム
	pickupBanned bool // 拾得禁止のカードの効果中かどうか（フロア移動で解除）
	// メニューシステム
	showMenu                  bool // メニューウィンドウの表示フラグ
	menuSelectedRow           int  // メニューで選択中の行（0=上, 1=下）
	menuSelectedCol           int  // メニューで選択中の列（0=左, 1=右）
	returnToMenuFromInventory bool // インベントリからメニューに戻るフラグ
	returnToMenuFromGround    bool // 足元メニューからメニューに戻るフラグ
	showEmptyInventoryMessage bool // 空のインベントリメッセージ表示フラグ
	groundMenuJustOpened      bool // 足元メニューが開かれたばかりかどうかのフラグ
	// メッセージ履歴システム
	messageLog       MessageLog // 表示済みメッセージの履歴
	showMessageLog   bool       // メッセージ履歴ウィンドウの表示フラグ
	messageLogScroll int        // 履歴のスクロール量（0で最新）
	// インベントリ拡張（絞り込み・任意名）
	inventoryFilter   ItemCategory   // 絞り込み中のカテゴリ（CategoryAllで全て表示）
	customNames       map[int]string // 未識別アイテム種別（テンプレートID）に付けた任意名
	showNameInput     bool           // 名前入力ウィンドウの表示フラグ
	nameInput         NameInput      // 入力中の名前
	nameInputCursor   int            // 五十音グリッドのカーソル位置
	nameInputTargetID int            // 名前を付ける対象のテンプレートID
	// 設定・中断システム
	settings                GameSettings // ゲーム設定
	showSettings            bool         // 設定ウィンドウの表示フラグ
	settingsSelectedIndex   int          // 設定ウィンドウで選択中の項目
	showSuspendPrompt       bool         // 中断確認ダイアログの表示フラグ
	suspendSelectedOption   int          // 中断確認で選択中の項目（0=中断する, 1=やめる）
	suspendPromptJustOpened bool         // 中断確認が開かれたばかりかどうかのフラグ
	suspendRequested        bool         // 中断セーブが完了しゲーム終了を要求するフラグ
	// ヘルプシステム
	showHelp             bool // ヘルプウィンドウの表示フラグ
	helpPage             int  // ヘルプで表示中のページ番号
	helpScroll           int  // ヘルプページのスクロール量（0で先頭）
	returnToMenuFromHelp bool // ヘルプからメニューに戻るフラグ
}

func (g *Game) CanAcceptInput() bool {
	if len(g.ActionQueue.Queue) == 0 {
		return true
	}
	for _, act := range g.ActionQueue.Queue {
		if !act.NonBlocking {
			return false
		}
	}
	return true
}

func (g *Game) IsEnemyAdjacent() bool {
	px, py := g.state.Player.X, g.state.Player.Y
	for _, enemy := range g.state.Enemies {
		if abs(enemy.X-px) <= 1 && abs(enemy.Y-py) <= 1 {
			return true
		}
	}
	return false
}

func (g *Game) Update() error {
	// 中断セーブ完了後はゲームを終了する
	if g.suspendRequested {
		return ebiten.Termination
	}

	// プレイヤーが死亡している場合の処理
	if g.playerDead {
		// log.Printf("DEBUG: Player is dead, queue length: %d, deathMessageAdded: %v", len(g.ActionQueue.Queue), g.deathMessageAdded)

		// 死亡時でも攻撃アニメーション処理を継続
		g.UpdateAttackTimer()
		g.HandleEnemyAttackTimers()

		// 死亡時でもメッセージ表示のためManageDescriptionsとActionQueueを処理
		g.ManageDescriptions()
		g.HandleActionQueue()

		// ActionQueueが空になったら死亡メッセージを追加
		if !g.deathMessageAdded && len(g.ActionQueue.Queue) == 0 {
			log.Printf("DEBUG: Adding death message to ActionQueue, isCombatActive: %v", g.isCombatActive)
			deathMessage := "海老さんは倒れた"
			deathAction := Action{
				Duration: 1.0, // 1秒間表示
				Message:  deathMessage,
				Execute: func(g *Game) {
					log.Printf("DEBUG: Death message Execute function called")
				},
			}
			g.ActionQueue.Queue = append(g.ActionQueue.Queue, deathAction)
			g.deathMessageAdded = true
			// 戦闘状態を強制的にアクティブにして、メッセージが正常に表示されるようにする
			g.isCombatActive = true
			log.Printf("DEBUG: Death message added, queue length: %d, set isCombatActive to true", len(g.ActionQueue.Queue))
		}

		// ActionQueueが再び空になったら（死亡メッセージ表示完了後）フェードアウト開始
		if g.deathMessageAdded && len(g.ActionQueue.Queue) == 0 && g.fadeOutProgress < 1.0 {
			g.fadeOutProgress += 1.0 / 60.0 // 1秒でフェードアウト完了
			if g.fadeOutProgress > 1.0 {
				g.fadeOutProgress = 1.0
			}
		}

		// リセットタイマーを減らす
		g.gameResetTimer -= 1.0 / 60.0
		if g.gameResetTimer <= 0 {
			g.resetGame()
			g.fadeInProgress = 0.0 // フェードイン開始
		}
		return nil
	}

	// フェードイン処理
	if g.fadeInProgress < 1.0 {
		g.fadeInProgress += 1.0 / 60.0 // 1秒でフェードイン完了
		if g.fadeInProgress > 1.0 {
			g.fadeInProgress = 1.0
		}
	}

	// 満腹度0時の点滅タイマー更新
	if g.state.Player.Satiety == 0 {
		g.starvationBlinkTimer += 1.0 / 60.0
		if g.starvationBlinkTimer >= 1.0 { // 1秒周期で点滅
			g.starvationBlinkTimer = 0.0
		}
	}

	if !g.showInventory && g.CanAcceptInput() && !g.ShowGroundItem && !g.showStairsPrompt && !g.showMenu && !g.showEmptyInventoryMessage && !g.showMessageLog && !g.showSettings && !g.showSuspendPrompt && !g.showHelp {
		// 睡眠状態の場合は自動的にターンを進行させる
		if g.state.Player.StatusAilments.Sleep > 0 {
			// 睡眠時の処理: MovePlayer(0,0)を呼び出して睡眠メッセージを表示し、ターンを進行
			g.MovePlayer(0, 0)
			g.isActioned = true
			return nil
		}
		dx, dy := g.HandleInput()
		//dx, dy := g.CheatHandleInput()

		if g.zPressed && !g.ShowGroundItem {
			// プレイヤーが混乱状態の場合、ランダムな方向に攻撃
			if g.state.Player.StatusAilments.Confusion > 0 {
				g.attackPlayerConfused()
			} else {
				g.CheckForEnemies(dx, dy)
			}
			g.zPressed = false
			// 攻撃でもターン進行
			g.AdvanceTurn()
			return nil
		}

		moved := g.MovePlayer(dx, dy)
		//moved := g.CheatMovePlayer(dx, dy)

		if moved {
			g.isActioned = true
			g.Animating = true // Set the animating flag
			g.xPressed = false // Reset the xPressed flag
			// 混乱状態でない場合のみ入力方向をアニメーション用に保存
			if g.state.Player.StatusAilments.Confusion == 0 {
				g.dx, g.dy = dx, dy // Save the direction of movement
			}
			// ターン進行とモンスター湧きチェック
			g.AdvanceTurn()
		}

		// 扉を開く処理の追加
		spacePressed := inpututil.IsKeyJustPressed(ebiten.KeySpace) // Spaceキーをチェック
		if spacePressed {
			g.OpenDoor()
		}
	}

	// 睡眠状態・メッセージ履歴/メニュー/設定/中断確認/ヘルプの表示中はDキーとF1キーも無効
	if g.state.Player.StatusAilments.Sleep == 0 && !g.showMessageLog && !g.showMenu && !g.showSettings && !g.showSuspendPrompt && !g.showHelp {
		g.processDKeyPress()

		// デバッグ用F1キー処理
		g.processF1KeyPress()
	}

	// Find item at player's position (足元メニュー表示中でない場合のみ更新)
	playerX, playerY := g.state.Player.X, g.state.Player.Y
	if !g.ShowGroundItem {
		for _, item := range g.state.Items {
			itemX, itemY := item.GetPosition()
			if itemX == playerX && itemY == playerY {
				g.currentGroundItem = item // Assuming g.currentGroundItem is a field of *Game
				break
			} else {
				g.currentGroundItem = nil
			}
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

	err = g.handleMenuInput()
	if err != nil {
		return err
	}

	err = g.handleSettingsInput()
	if err != nil {
		return err
	}

	err = g.handleSuspendPromptInput()
	if err != nil {
		return err
	}

	err = g.handleEmptyInventoryMessage()
	if err != nil {
		return err
	}

	err = g.handleMessageLogInput()
	if err != nil {
		return err
	}

	err = g.handleHelpInput()
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
	g.DrawMapTraps(screen, offsetX, offsetY)
	g.DrawThrownItem(screen, offsetX, offsetY)
	g.DrawEnemies(screen, offsetX, offsetY)
	g.DrawTrajectoryPreview(screen, offsetX, offsetY)
	g.DrawHUD(screen)
	g.DrawPlayer(screen, centerX, centerY)

	// Draw the inventory window if the showInventory flag is set
	if g.showInventory {
		g.showDescription = false
		if err := g.drawInventoryWindow(screen); err != nil {
			log.Printf("Error drawing inventory window: %v", err)
		}
		// 名前入力ウィンドウはインベントリの上に重ねて表示する
		if g.showNameInput {
			g.drawNameInputWindow(screen)
		}
	}

	// Draw the menu window if the showMenu flag is set
	if g.showMenu {
		g.drawMenuWindow(screen)
	}

	// Draw the empty inventory message if the showEmptyInventoryMessage flag is set
	if g.showEmptyInventoryMessage {
		g.drawEmptyInventoryMessage(screen)
	}

	// Draw the settings window if the showSettings flag is set
	if g.showSettings {
		g.drawSettingsWindow(screen)
	}

	// Draw the suspend prompt if the showSuspendPrompt flag is set
	if g.showSuspendPrompt {
		g.drawSuspendPrompt(screen)
	}

	if g.useIdentifyItem {
		g.drawUseIdentifyItemWindow(screen)
	}

	g.drawActionMenu(screen)

	g.drawItemDescription(screen)

	g.DrawDescriptions(screen)

	g.DrawGroundItem(screen)

	g.drawGroundItemDescription(screen)

	g.DrawStairsPrompt(screen)

	// Draw the message log window if the showMessageLog flag is set
	if g.showMessageLog {
		g.drawMessageLogWindow(screen)
	}

	// Draw the help window if the showHelp flag is set
	if g.showHelp {
		g.drawHelpWindow(screen)
	}

	g.UpdateAndDrawMiniMap(screen)

	if g.fadeAlpha > 0 {
		g.drawOverlay(screen)
	}

	// 死亡時のフェードアウト処理（メッセージ表示後）
	if g.playerDead && g.deathMessageAdded && len(g.ActionQueue.Queue) == 0 && g.fadeOutProgress > 0 {
		g.drawDeathFade(screen)
	}

	// ゲーム開始時のフェードイン処理
	if g.fadeInProgress < 1.0 {
		g.drawFadeIn(screen)
	}

}

// 死亡時のフェードアウト描画
func (g *Game) drawDeathFade(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	overlay := ebiten.NewImage(screenWidth, screenHeight)

	// フェードアウトの進行度に応じてアルファ値を計算
	alpha := uint8(g.fadeOutProgress * 255)
	color := color.RGBA{0, 0, 0, alpha}
	overlay.Fill(color)

	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(overlay, opts)
}

// ゲーム開始時のフェードイン描画
func (g *Game) drawFadeIn(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	overlay := ebiten.NewImage(screenWidth, screenHeight)

	// フェードインの進行度に応じてアルファ値を計算（逆算）
	alpha := uint8((1.0 - g.fadeInProgress) * 255)
	color := color.RGBA{0, 0, 0, alpha}
	overlay.Fill(color)

	opts := &ebiten.DrawImageOptions{}
	screen.DrawImage(overlay, opts)
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
	mamuruImg := loadImage("img/mamuru.png")
	honeyImg := loadImage("img/honey.png")
	cardImg := loadImage("img/card.png")
	sausageImg := loadImage("img/sausage.png")
	mintiaImg := loadImage("img/mintia.png")
	weaponImg := loadImage("img/weapon.png")
	armorImg := loadImage("img/armor.png")
	arrowImg := loadImage("img/arrow.png")
	caneImg := loadImage("img/cane.png")
	effectImg := loadImage("img/effect.png")
	accessoryImg := loadImage("img/ring.png")
	hatenaImg := loadImage("img/hatena.png")
	sleepTrapImg := loadImage("img/suimin_trap.png")
	poisonArrowTrapImg := loadImage("img/poison_arrow_trap.png")
	slowTrapImg := loadImage("img/slow_trap.png")
	mineTrapImg := loadImage("img/mine_trap.png")

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
	mapGrid, enemies, items, newFloor, newRoom, traps := GenerateRandomMap(70, 70, 0, &player) // 初期階層は1です

	game := &Game{
		state: GameState{
			Map:      mapGrid,
			Player:   player,
			Enemies:  enemies,
			Items:    items,
			MapTraps: traps,
		},
		rooms:              newRoom,
		playerImg:          img,
		tilesetImg:         tilesetImg,
		ebiImg:             ebiImg,
		snakeImg:           snakeImg,
		mamuruImg:          mamuruImg,
		honeyImg:           honeyImg,
		kaneImg:            kaneImg,
		cardImg:            cardImg,
		mintiaImg:          mintiaImg,
		sausageImg:         sausageImg,
		weaponImg:          weaponImg,
		armorImg:           armorImg,
		arrowImg:           arrowImg,
		caneImg:            caneImg,
		effectImg:          effectImg,
		accessoryImg:       accessoryImg,
		hatenaImg:          hatenaImg,
		sleepTrapImg:       sleepTrapImg,
		poisonArrowTrapImg: poisonArrowTrapImg,
		slowTrapImg:        slowTrapImg,
		mineTrapImg:        mineTrapImg,
		offsetX:            0,
		offsetY:            0,
		Floor:              newFloor,
		frameCount:         0,
		tmpPlayerOffsetX:   0,
		tmpPlayerOffsetY:   0,
		ActionQueue: ActionQueue{
			Queue: make([]Action, 0),
		},
		isCombatActive:       false,
		zPressed:             false,
		tmpSelectedItemIndex: -1,
		customNames:          map[int]string{},
	}

	// モンスター湧きシステム初期化
	game.InitializeSpawnSystem()

	// フェードイン状態を初期化
	game.fadeInProgress = 0.0

	// 設定ファイルを読み込む（存在しない・壊れている場合は初期値）
	game.settings = loadSettings()

	// 中断データがあれば再開する（壊れている場合は新しい冒険のまま）
	if game.tryResumeFromSave() {
		log.Println("中断データから冒険を再開します")
	}

	return game
}

func main() {
	game := NewGame()

	ebiten.SetWindowSize(1280, 960)
	ebiten.SetWindowTitle("ebirogue")
	ebiten.SetFullscreen(game.settings.Fullscreen)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}

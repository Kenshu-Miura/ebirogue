package main

import (
	"fmt"
	"image"
	_ "image/png" // PNG画像を読み込むために必要
	"log"

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
	screenWidth, screenHeight := screen.Size()

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
	game := &Game{
		state: GameState{
			Map: [][]Tile{
				{Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}},
				{Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "wall", Blocked: true, BlockSight: true}},
				{Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "wall", Blocked: true, BlockSight: true}},
				{Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "wall", Blocked: true, BlockSight: true}},
				{Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "wall", Blocked: true, BlockSight: true}},
				{Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "wall", Blocked: true, BlockSight: true}},
				{Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "wall", Blocked: true, BlockSight: true}},
				{Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "floor", Blocked: false, BlockSight: false}, Tile{Type: "wall", Blocked: true, BlockSight: true}},
				{Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}, Tile{Type: "wall", Blocked: true, BlockSight: true}},
			},
			Player: Player{
				Entity:    Entity{X: 1, Y: 1, Char: '@'},
				Health:    100,
				MaxHealth: 100,
			},
			Enemies: []Enemy{
				{
					Entity:    Entity{X: 1, Y: 2, Char: 'E'},
					Health:    50,
					MaxHealth: 50,
				},
			},
			Items: []Entity{
				{X: 2, Y: 1, Char: '!'},
			},
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

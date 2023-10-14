package main

import (
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
}

func (g *Game) Update() error {

	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.offsetY += tileSize
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.offsetY -= tileSize
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.offsetX += tileSize
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.offsetX -= tileSize
	}
	return nil

}

func (g *Game) Draw(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Size()

	mapWidth := len(g.state.Map[0]) * tileSize
	mapHeight := len(g.state.Map) * tileSize

	offsetX := (screenWidth-mapWidth)/2 + g.offsetX
	offsetY := (screenHeight-mapHeight)/2 + g.offsetY

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
	// 画面中央の位置を計算
	centerX := (screenWidth - tileSize) / 2
	centerY := (screenHeight - tileSize) / 2
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

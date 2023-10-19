package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png" // PNG画像を読み込むために必要

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

func (g *Game) DrawMap(screen *ebiten.Image, offsetX, offsetY int) {
	for y, row := range g.state.Map {
		for x, tile := range row {
			var srcX, srcY int
			switch tile.Type {
			case "wall":
				srcX, srcY = 0, 0
			case "corridor":
				srcX, srcY = tileSize, 0
			case "floor":
				srcX, srcY = 2*tileSize, 0
			case "door":
				srcX, srcY = 3*tileSize, 0
			case "stairs":
				srcX, srcY = 4*tileSize, 0
			default:
				continue
			}
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(x*tileSize+offsetX), float64(y*tileSize+offsetY))
			screen.DrawImage(g.tilesetImg.SubImage(image.Rect(srcX, srcY, srcX+tileSize, srcY+tileSize)).(*ebiten.Image), opts)
		}
	}
}

func (g *Game) DrawPlayer(screen *ebiten.Image, centerX, centerY int) {
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(centerX), float64(centerY))
	screen.DrawImage(g.playerImg, opts)
}

func (g *Game) DrawItems(screen *ebiten.Image, offsetX, offsetY int) {
	for _, item := range g.state.Items {
		var img *ebiten.Image
		switch item.Type {
		case "Kane":
			img = g.kaneImg
		case "Card":
			img = g.cardImg
		default:
			img = g.sausageImg
		}
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(item.X*tileSize+offsetX), float64(item.Y*tileSize+offsetY))
		screen.DrawImage(img, opts)
	}
}

func (g *Game) DrawEnemies(screen *ebiten.Image, offsetX, offsetY int) {
	for _, enemy := range g.state.Enemies {
		var img *ebiten.Image
		switch enemy.Type {
		case "Snake":
			img = g.snakeImg
		case "Shrimp":
			img = g.ebiImg
		default:
			img = g.ebiImg
		}

		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(enemy.X*tileSize+offsetX), float64(enemy.Y*tileSize+offsetY))
		screen.DrawImage(img, opts)
	}
}

func (g *Game) DrawHUD(screen *ebiten.Image) {
	screenWidth, _ := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Moves count
	MoveText := fmt.Sprintf("ターン数: %3d", g.moveCount)
	text.Draw(screen, MoveText, mplusNormalFont, screenWidth-130, 30, color.White)

	// Player HP
	playerHPText := fmt.Sprintf("HP:%3d/%3d", g.state.Player.Health, g.state.Player.MaxHealth)
	text.Draw(screen, playerHPText, mplusNormalFont, screenWidth-130, 50, color.White)

	// Player Satiety
	playerSatietyText := fmt.Sprintf("満腹度:%3d/%3d", g.state.Player.Satiety, g.state.Player.MaxSatiety)
	text.Draw(screen, playerSatietyText, mplusNormalFont, screenWidth-130, 70, color.White)

	// Player Attack Power
	playerAttackPowerText := fmt.Sprintf("攻撃力: %3d", g.state.Player.AttackPower)
	text.Draw(screen, playerAttackPowerText, mplusNormalFont, screenWidth-130, 90, color.White)

	// Player Defense Power
	playerDefensePowerText := fmt.Sprintf("防御力: %3d", g.state.Player.DefensePower)
	text.Draw(screen, playerDefensePowerText, mplusNormalFont, screenWidth-130, 110, color.White)

	// Player Power
	playerPowerText := fmt.Sprintf("パワー: %2d/%2d", g.state.Player.Power, g.state.Player.MaxPower)
	text.Draw(screen, playerPowerText, mplusNormalFont, screenWidth-130, 130, color.White)

	// Player Experience Points
	playerExpText := fmt.Sprintf("経験値: %3d", g.state.Player.ExperiencePoints)
	text.Draw(screen, playerExpText, mplusNormalFont, screenWidth-130, 150, color.White) // Adjusted y-coordinate to place Experience Points text below Defense Power text

	// Floor level
	floorText := fmt.Sprintf("階層: B%dF", g.Floor)
	text.Draw(screen, floorText, mplusNormalFont, 10, 30, color.White) // x座標とy座標を直接指定

	// Player Level
	playerLevelText := fmt.Sprintf("レベル: %d", g.state.Player.Level)
	text.Draw(screen, playerLevelText, mplusNormalFont, 10, 50, color.White) // x座標とy座標を直接指定

	// Player Coordinate
	playerCoordinateText := fmt.Sprintf("座標: (%d, %d)", g.state.Player.X, g.state.Player.Y)
	text.Draw(screen, playerCoordinateText, mplusNormalFont, 10, 70, color.White) // x座標とy座標を直接指定

	// Player Room
	playerRoomText := logCurrentRoom(g.state.Player, g.rooms)
	text.Draw(screen, playerRoomText, mplusNormalFont, 10, 90, color.White) // x座標とy座標を直接指定

}

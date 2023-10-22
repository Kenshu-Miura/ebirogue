package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png" // PNG画像を読み込むために必要
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

func (g *Game) CalculateAnimationOffset(screen *ebiten.Image) (int, int) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	centerX := (screenWidth-tileSize)/2 - tileSize
	centerY := (screenHeight-tileSize)/2 - tileSize

	animationProgress := (float64(g.AnimationProgressInt) / 10.0) * 2.0
	adjustedProgress := animationProgress
	if g.AnimationProgressInt == 1 {
		adjustedProgress = 0.2
	}

	offsetAdjustmentX, offsetAdjustmentY := 0, 0
	if g.AnimationProgressInt > 0 {
		if g.dx > 0 {
			offsetAdjustmentX = -20
		} else if g.dx < 0 {
			offsetAdjustmentX = 20
		}
		if g.dy > 0 {
			offsetAdjustmentY = -20
		} else if g.dy < 0 {
			offsetAdjustmentY = 20
		}
	}

	offsetX := centerX - g.state.Player.X*tileSize - (int(adjustedProgress*10)*g.dx + offsetAdjustmentX)
	offsetY := centerY - g.state.Player.Y*tileSize - (int(adjustedProgress*10)*g.dy + offsetAdjustmentY)

	return offsetX, offsetY
}

// 敵のアニメーション進行度を更新する関数
func (g *Game) UpdateEnemyAnimation(enemy *Enemy) {
	if enemy.Animating {
		enemy.AnimationProgressInt++
		if enemy.AnimationProgressInt > 20 { // 20フレームでアニメーションを完了
			enemy.Animating = false
			enemy.AnimationProgressInt = 0
		}
	}
}

// 敵のオフセットを計算する関数
func (g *Game) CalculateEnemyOffset(enemy *Enemy) (int, int) {
	animationProgress := (float64(enemy.AnimationProgressInt) / 10.0) * 10.0 // ここを変更
	adjustedProgress := animationProgress
	if enemy.AnimationProgressInt == 1 {
		adjustedProgress = 1.0 // アニメーションの初めのフレームの進行度を調整
	}

	offsetAdjustmentX, offsetAdjustmentY := 0, 0
	if enemy.AnimationProgressInt > 0 {
		if enemy.dx > 0 {
			offsetAdjustmentX = -20
		} else if enemy.dx < 0 {
			offsetAdjustmentX = 20
		}
		if enemy.dy > 0 {
			offsetAdjustmentY = -20
		} else if enemy.dy < 0 {
			offsetAdjustmentY = 20
		}
	}

	offsetX := (int(adjustedProgress)*enemy.dx + offsetAdjustmentX) // ここを変更
	offsetY := (int(adjustedProgress)*enemy.dy + offsetAdjustmentY) // ここを変更
	return offsetX, offsetY
}

func (g *Game) ManageDescriptions() {
	now := time.Now()
	if now.Before(g.nextDescriptionTime) {
		return
	}

	if len(g.ActionQueue.Queue) > 0 {
		action := g.ActionQueue.Queue[0]

		g.descriptionText = action.Message
		g.showDescription = true

		g.nextDescriptionTime = now.Add(500 * time.Millisecond)
	} else if len(g.descriptionQueue) > 0 {
		// Existing logic for handling descriptionQueue
		g.descriptionText = g.descriptionQueue[0]
		g.showDescription = true
		g.descriptionQueue = g.descriptionQueue[1:]

		g.nextDescriptionTime = now.Add(500 * time.Millisecond)
	} else {
		g.showDescription = false
	}
}

func (g *Game) DrawDescriptions(screen *ebiten.Image) {
	if g.showDescription {
		screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
		descriptionWindowWidth, descriptionWindowHeight := 500, 120
		windowX, windowY := (screenWidth-descriptionWindowWidth)/2, screenHeight-descriptionWindowHeight-10

		drawWindowWithBorder(screen, windowX, windowY, descriptionWindowWidth, descriptionWindowHeight, 127)

		// Draw description text
		text.Draw(screen, g.descriptionText, mplusNormalFont, windowX+10, windowY+20, color.White)
	}
}

func drawWindowWithBorder(screen *ebiten.Image, windowX, windowY, windowWidth, windowHeight int, alpha uint8) {
	// Draw window background with specified alpha value
	windowBackground := ebiten.NewImage(windowWidth, windowHeight)
	windowBackground.Fill(color.RGBA{0, 0, 0, alpha}) // Use alpha argument here
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(windowX), float64(windowY))
	screen.DrawImage(windowBackground, opts)

	// Draw window border
	borderSize := 2
	borderColor := color.RGBA{255, 255, 255, 255}

	borderImg := ebiten.NewImage(windowWidth+2*borderSize, windowHeight+2*borderSize)
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
}

func (g *Game) drawItemDescription(screen *ebiten.Image) {
	if g.showItemDescription {
		// Define menu window parameters
		screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
		descriptionWindowWidth, descriptionWindowHeight := 500, 120
		windowX, windowY := (screenWidth-descriptionWindowWidth)/2, screenHeight-descriptionWindowHeight-10

		drawWindowWithBorder(screen, windowX, windowY, descriptionWindowWidth, descriptionWindowHeight, 255)

		// Draw description text
		text.Draw(screen, g.itemdescriptionText, mplusNormalFont, windowX+10, windowY+20, color.White)
	}
}

func (g *Game) drawActionMenu(screen *ebiten.Image) {
	if g.showItemActions {
		// Define menu window parameters
		menuWidth, menuHeight := 200, 100
		menuX, menuY := (screen.Bounds().Dx()-menuWidth)/2, (screen.Bounds().Dy()-menuHeight)/2

		drawWindowWithBorder(screen, menuX, menuY, menuWidth, menuHeight, 255)

		// Draw menu actions
		actions := []string{"使う", "投げる", "置く", "説明"}
		for i, action := range actions {
			textColor := color.White
			yOffset := menuY + 20 + i*20 // Adjust the offset values to position the text correctly
			text.Draw(screen, action, mplusNormalFont, menuX+30, yOffset, textColor)
		}

		// Draw selection pointer
		pointerX := menuX + 10                            // Adjust the X value to position the pointer correctly
		pointerY := menuY + 20 + g.selectedActionIndex*20 // Adjust the offset values to position the pointer correctly
		text.Draw(screen, "→", mplusNormalFont, pointerX, pointerY, color.White)
	}
}

func (g *Game) drawInventoryWindow(screen *ebiten.Image) error {

	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	windowWidth, windowHeight := 400, 300
	windowX, windowY := (screenWidth-windowWidth)/2, (screenHeight-windowHeight)/2

	drawWindowWithBorder(screen, windowX, windowY, windowWidth, windowHeight, 127)

	// Draw items
	const itemsPerColumn = 10 // 1列に表示するアイテムの数
	const columnWidth = 180   // 列の幅 (ピクセル)
	for i, item := range g.state.Player.Inventory {
		itemText := fmt.Sprintf("%d. %s", i+1, item.GetName())

		// 現在の列と行の計算
		column := i / itemsPerColumn
		row := i % itemsPerColumn

		// アイテムテキストの描画位置の計算
		x := windowX + 30 + column*columnWidth
		y := windowY + 30 + row*25

		text.Draw(screen, itemText, mplusNormalFont, x, y, color.White)

		if i == g.selectedItemIndex {
			// Step 3: Draw the pointer next to the selected item
			pointerText := "→"
			text.Draw(screen, pointerText, mplusNormalFont, x-20, y, color.White)
		}
	}

	return nil
}

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
	tmpPlayerOffsetX, tmpPlayerOffsetY := 0.0, 0.0

	w, h := g.playerImg.Bounds().Dx(), g.playerImg.Bounds().Dy()
	opts.GeoM.Translate(float64(-w/2), float64(-h/2)) // Move the image center to the origin

	switch g.state.Player.Direction {
	case Right:
		tmpPlayerOffsetX = g.tmpPlayerOffsetX
		opts.GeoM.Rotate(math.Pi / 2) // Rotate 90 degrees to the right
	case Left:
		tmpPlayerOffsetX = -g.tmpPlayerOffsetX
		opts.GeoM.Rotate(-math.Pi / 2) // Rotate 90 degrees to the left
	case UpRight:
		tmpPlayerOffsetX = g.tmpPlayerOffsetX
		tmpPlayerOffsetY = -g.tmpPlayerOffsetY
		opts.GeoM.Rotate(math.Pi / 4) // Rotate 45 degrees to the right
	case UpLeft:
		tmpPlayerOffsetX = -g.tmpPlayerOffsetX
		tmpPlayerOffsetY = -g.tmpPlayerOffsetY
		opts.GeoM.Rotate(-math.Pi / 4) // Rotate 45 degrees to the left
	case DownRight:
		tmpPlayerOffsetX = g.tmpPlayerOffsetX
		tmpPlayerOffsetY = g.tmpPlayerOffsetY
		opts.GeoM.Rotate(3 * math.Pi / 4) // Rotate 135 degrees to the right
	case DownLeft:
		tmpPlayerOffsetX = -g.tmpPlayerOffsetX
		tmpPlayerOffsetY = g.tmpPlayerOffsetY
		opts.GeoM.Rotate(-3 * math.Pi / 4) // Rotate 135 degrees to the left
	case Down:
		tmpPlayerOffsetY = g.tmpPlayerOffsetY
		opts.GeoM.Rotate(math.Pi) // Rotate 180 degrees
	case Up:
		tmpPlayerOffsetY = -g.tmpPlayerOffsetY
	}

	opts.GeoM.Translate(float64(w/2)+float64(centerX)+tmpPlayerOffsetX, float64(h/2)+float64(centerY)+tmpPlayerOffsetY)
	screen.DrawImage(g.playerImg, opts)
}

func (g *Game) DrawItems(screen *ebiten.Image, offsetX, offsetY int) {
	for _, item := range g.state.Items {
		var img *ebiten.Image
		switch item.GetType() {
		case "Kane":
			img = g.kaneImg
		case "Card":
			img = g.cardImg
		case "Mintia":
			img = g.mintiaImg
		default:
			img = g.sausageImg
		}
		opts := &ebiten.DrawImageOptions{}
		itemX, itemY := item.GetPosition()
		opts.GeoM.Translate(float64(itemX*tileSize+offsetX), float64(itemY*tileSize+offsetY))
		screen.DrawImage(img, opts)
	}
}

func (g *Game) DrawEnemies(screen *ebiten.Image, offsetX, offsetY int) {
	for i := range g.state.Enemies {
		enemy := &g.state.Enemies[i]

		// 敵のアニメーションを更新
		g.UpdateEnemyAnimation(enemy)

		// 敵の描画オフセットを計算
		enemyOffsetX, enemyOffsetY := g.CalculateEnemyOffset(enemy)

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
		// 敵の位置とオフセットを適用して敵を描画
		opts.GeoM.Translate(float64(enemy.X*tileSize+offsetX+enemyOffsetX), float64(enemy.Y*tileSize+offsetY+enemyOffsetY))
		//opts.GeoM.Translate(float64(enemy.X*tileSize+enemyOffsetX), float64(enemy.Y*tileSize+enemyOffsetY))
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
	text.Draw(screen, playerExpText, mplusNormalFont, screenWidth-130, 150, color.White)

	// Player Inventory
	//inventoryText := "所持アイテム:"
	//text.Draw(screen, inventoryText, mplusNormalFont, screenWidth-130, 180, color.White) // Adjust the y-coordinate as needed

	//yCoord := 210 // Starting y-coordinate for the list of items, adjust as needed
	//for _, item := range g.state.Player.Inventory {
	//		itemText := fmt.Sprintf("- %s", item.Name)
	//	text.Draw(screen, itemText, mplusNormalFont, screenWidth-130, yCoord, color.White)
	//	yCoord += 30 // Increment y-coordinate for the next item, adjust the increment value as needed
	//}

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

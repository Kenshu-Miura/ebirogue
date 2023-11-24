package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png" // PNG画像を読み込むために必要
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func (g *Game) drawOverlay(screen *ebiten.Image) {
	// 画面サイズに合わせた黒い画像（オーバーレイ）を作成
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	overlay := ebiten.NewImage(screenWidth, screenHeight)
	overlay.Fill(color.NRGBA{0, 0, 0, 255}) // 完全に黒

	// オーバーレイの描画オプションを設定
	opts := &ebiten.DrawImageOptions{}

	// ColorScaleのインスタンスを作成してアルファ値を設定
	var colorScale ebiten.ColorScale
	colorScale.Scale(1, 1, 1, float32(g.fadeAlpha))

	// ColorScaleを適用
	opts.ColorScale = colorScale

	// オーバーレイを画面に描画
	screen.DrawImage(overlay, opts)
}

func (g *Game) DrawStairsPrompt(screen *ebiten.Image) {
	if g.showStairsPrompt && !g.fadingOut && !g.fadingIn {
		windowX, windowY, windowWidth, windowHeight := 100, 100, 200, 50 // Adjust these values as needed
		drawWindowWithBorder(screen, windowX, windowY, windowWidth, windowHeight, 255)
		options := []string{"進む", "やめる"}
		for i, option := range options {
			text.Draw(screen, option, mplusNormalFont, windowX+i*100+20, windowY+25, color.White) // Adjust these values as needed
		}
		cursorX := windowX + g.selectedOption*100 // Adjust these values as needed
		cursorY := windowY + 25                   // Adjust these values as needed
		text.Draw(screen, "→", mplusNormalFont, cursorX, cursorY, color.White)
	}
}

func (g *Game) UpdateAndDrawMiniMap(screen *ebiten.Image) {
	if g.miniMapDirty {
		// ミニマップを更新
		g.updateMiniMap(screen)
		g.miniMapDirty = false
	}

	// キャッシュされたミニマップイメージをスクリーンに描画
	if g.miniMap != nil {
		screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
		miniMapWidth, miniMapHeight := g.miniMap.Bounds().Dx(), g.miniMap.Bounds().Dy()
		miniMapX := screenWidth - miniMapWidth - 10   // 画面の右端から10ピクセルのマージンを持たせる
		miniMapY := screenHeight - miniMapHeight - 10 // 画面の下端から10ピクセルのマージンを持たせる

		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(miniMapX), float64(miniMapY))
		screen.DrawImage(g.miniMap, opts)
	}
}

func (g *Game) updateMiniMap(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	tilePixelSize := 3
	mapWidth := len(g.state.Map[0])
	mapHeight := len(g.state.Map)
	miniMapWidth := mapWidth * tilePixelSize
	miniMapHeight := mapHeight * tilePixelSize

	// キャッシュされたミニマップイメージを作成または更新
	if g.miniMap == nil || g.miniMap.Bounds().Dx() != miniMapWidth || g.miniMap.Bounds().Dy() != miniMapHeight {
		g.miniMap = ebiten.NewImage(miniMapWidth, miniMapHeight)
	} else {
		// g.miniMapをクリア
		g.miniMap.Clear()
	}

	// ミニマップの描画位置を計算
	miniMapX := screenWidth - miniMapWidth - 10   // 画面の右端から10ピクセルのマージンを持たせる
	miniMapY := screenHeight - miniMapHeight - 10 // 画面の下端から10ピクセルのマージンを持たせる

	// 訪れたタイルを青色半透明で描画するためのイメージを作成
	miniMapTile := ebiten.NewImage(tilePixelSize, tilePixelSize)
	miniMapTile.Fill(color.RGBA{0, 0, 255, 128}) // 青色半透明

	// ミニマップを描画
	for y, row := range g.state.Map {
		for x, tile := range row {
			if tile.Visited && tile.Type != "wall" {
				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(float64(x*tilePixelSize), float64(y*tilePixelSize))

				// tile.Typeが"stairs"であるかどうかをチェック
				if tile.Type == "stairs" {
					// 階段タイル用のボーダーのイメージを作成
					stairsTile := ebiten.NewImage(tilePixelSize, tilePixelSize)
					//borderSize := 1 // ボーダーの幅

					// ボーダーを描画
					for i := 0; i < tilePixelSize; i++ {
						// 上のボーダー
						stairsTile.Set(i, 0, color.White)
						// 下のボーダー
						stairsTile.Set(i, tilePixelSize-1, color.White)
						// 左のボーダー
						stairsTile.Set(0, i, color.White)
						// 右のボーダー
						stairsTile.Set(tilePixelSize-1, i, color.White)
					}

					g.miniMap.DrawImage(stairsTile, opts)
				} else {
					g.miniMap.DrawImage(miniMapTile, opts)
				}
			}
		}
	}

	// プレイヤーの位置を取得
	playerX, playerY := g.state.Player.X, g.state.Player.Y

	// プレイヤーの位置に対応するミニマップ上の座標を計算
	miniMapPlayerX := playerX * tilePixelSize
	miniMapPlayerY := playerY * tilePixelSize

	// 黄色の半透明のイメージを作成
	playerTile := ebiten.NewImage(tilePixelSize, tilePixelSize)
	playerTile.Fill(color.RGBA{255, 255, 0, 128}) // 黄色半透明

	// 黄色の半透明のイメージをミニマップ上のプレイヤーの位置に描画
	playerOpts := &ebiten.DrawImageOptions{}
	playerOpts.GeoM.Translate(float64(miniMapPlayerX), float64(miniMapPlayerY))
	g.miniMap.DrawImage(playerTile, playerOpts)

	// アイテムを青色で描画するためのイメージを作成
	itemTile := ebiten.NewImage(tilePixelSize, tilePixelSize)
	itemTile.Fill(color.RGBA{0, 255, 255, 128}) // 水色半透明

	// ゲームのアイテムリストをループして、ShowOnMiniMapがtrueのアイテムをミニマップに描画
	for _, item := range g.state.Items {
		if item.GetShowOnMiniMap() {
			itemX, itemY := item.GetPosition()
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(itemX*tilePixelSize), float64(itemY*tilePixelSize))
			g.miniMap.DrawImage(itemTile, opts)
		}
	}

	enemyTile := ebiten.NewImage(tilePixelSize, tilePixelSize)
	enemyTile.Fill(color.RGBA{255, 0, 0, 128}) // Red semi-transparent

	//log.Printf("ShowOnMiniMap: %v", g.state.Enemies[0].GetShowOnMiniMap())

	for _, enemy := range g.state.Enemies {
		if enemy.GetShowOnMiniMap() {
			enemyX, enemyY := enemy.GetPosition()
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(enemyX*tilePixelSize), float64(enemyY*tilePixelSize))
			g.miniMap.DrawImage(enemyTile, opts)
		}
	}

	// キャッシュされたミニマップイメージをスクリーンに描画
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(miniMapX), float64(miniMapY))
	screen.DrawImage(g.miniMap, opts)
}

func (g *Game) CalculateAnimationOffset(screen *ebiten.Image) (int, int) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	centerX := (screenWidth-tileSize)/2 - tileSize
	centerY := (screenHeight-tileSize)/2 - tileSize

	animationProgress := (float64(g.AnimationProgressInt) / 10.0) * 3.0
	adjustedProgress := animationProgress
	if g.AnimationProgressInt == 1 {
		adjustedProgress = 0.3
	}

	offsetAdjustmentX, offsetAdjustmentY := 0, 0
	if g.AnimationProgressInt > 0 {
		if g.dx > 0 {
			offsetAdjustmentX = -30
		} else if g.dx < 0 {
			offsetAdjustmentX = 30
		}
		if g.dy > 0 {
			offsetAdjustmentY = -30
		} else if g.dy < 0 {
			offsetAdjustmentY = 30
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
			offsetAdjustmentX = -30
		} else if enemy.dx < 0 {
			offsetAdjustmentX = 30
		}
		if enemy.dy > 0 {
			offsetAdjustmentY = -30
		} else if enemy.dy < 0 {
			offsetAdjustmentY = 30
		}
	}

	offsetX := (int(adjustedProgress)*enemy.dx + offsetAdjustmentX) // ここを変更
	offsetY := (int(adjustedProgress)*enemy.dy + offsetAdjustmentY) // ここを変更
	return offsetX, offsetY
}

func (g *Game) ManageDescriptions() {

	if len(g.ActionQueue.Queue) > 0 {
		action := g.ActionQueue.Queue[0]

		if action.Message != "" {
			g.descriptionText = action.Message
			g.showDescription = true
		}

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

		// アクションを取得
		var action Action
		if len(g.ActionQueue.Queue) > 0 {
			action = g.ActionQueue.Queue[0]
		}

		// 描画するテキストの基本位置
		x := windowX + 10
		y := windowY + 20

		// アイテム名の色を設定
		var itemNameColor color.Color
		itemNameColor = color.White
		if !action.IsIdentified {
			itemNameColor = color.RGBA{R: 255, G: 255, B: 0, A: 255} // 未識別は黄色
		}

		var dr font.Drawer
		dr.Face = mplusNormalFont

		if action.ItemName != "" {
			// アイテム名を含むメッセージを処理
			parts := strings.Split(action.Message, action.ItemName)
			firstPart := parts[0]
			secondPart := ""
			if len(parts) > 1 {
				secondPart = parts[1]
			}

			// 最初の部分を描画
			text.Draw(screen, firstPart, mplusNormalFont, x, y, color.White)
			bounds, _ := dr.BoundString(firstPart)
			x += (bounds.Max.X - bounds.Min.X).Ceil() + 5 // 5ピクセルのスペースを追加

			// アイテム名を描画
			text.Draw(screen, action.ItemName, mplusNormalFont, x, y, itemNameColor)
			bounds, _ = dr.BoundString(action.ItemName)
			x += (bounds.Max.X - bounds.Min.X).Ceil() + 5 // 5ピクセルのスペースを追加

			// 2番目の部分を描画
			text.Draw(screen, secondPart, mplusNormalFont, x, y, color.White)
		} else {
			// アイテム名がない場合はそのままメッセージを描画
			text.Draw(screen, action.Message, mplusNormalFont, x, y, color.White)
		}
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

func (g *Game) DrawGroundItem(screen *ebiten.Image) {
	if g.ShowGroundItem {
		screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
		itemWindowWidth, itemWindowHeight := 400, 26
		itemwindowX, itemwindowY := (screenWidth-itemWindowWidth)/2, (screenHeight-itemWindowHeight)/2
		actionWindowWidth, actionWindowHeight := 100, 90
		actionWindowX, actionWindowY := (screenWidth-actionWindowWidth)/2, (screenHeight-actionWindowHeight)/2

		// Draw item name window
		drawWindowWithBorder(screen, itemwindowX, itemwindowY, itemWindowWidth, itemWindowHeight, 127)
		if g.currentGroundItem != nil {
			groundItemName := getItemNameWithSharpness(g.currentGroundItem)

			// アイテムが識別されているかチェック
			identified := true
			if identifiableItem, ok := g.currentGroundItem.(Identifiable); ok {
				identified = identifiableItem.IsIdentified()
			}

			// テキストの描画位置
			x := itemwindowX + 10
			y := itemwindowY + 20

			var itemNameColor color.Color
			if identified {
				itemNameColor = color.White
			} else {
				itemNameColor = color.RGBA{R: 255, G: 255, B: 0, A: 255} // 未識別は黄色
			}

			// アイテム名を描画
			text.Draw(screen, groundItemName, mplusNormalFont, x, y, itemNameColor)

			// アイテム名の幅を取得して、xの位置を調整
			var dr font.Drawer
			dr.Face = mplusNormalFont
			bounds, _ := dr.BoundString(groundItemName)
			x += (bounds.Max.X - bounds.Min.X).Ceil() + 5 // 5ピクセルのスペースを追加

			// 「が落ちている」の部分を描画
			text.Draw(screen, "が落ちている", mplusNormalFont, x, y, color.White)
			// Draw actions window
			drawWindowWithBorder(screen, actionWindowX, actionWindowY+actionWindowHeight, actionWindowWidth, actionWindowHeight, 127)
			// Draw cursor
			text.Draw(screen, "→", mplusNormalFont, actionWindowX+10, actionWindowY+actionWindowHeight+20+(g.selectedGroundActionIndex*20), color.White)
			// Draw actions
			actions := []string{"拾う", "交換", "使う", "投げる"}
			for index, action := range actions {
				text.Draw(screen, action, mplusNormalFont, actionWindowX+30, actionWindowY+actionWindowHeight+20+(index*20), color.White)
			}

		} else {
			text.Draw(screen, "何も落ちていない", mplusNormalFont, itemwindowX+10, itemwindowY+20, color.White)
		}
	}
}

func (g *Game) drawActionMenu(screen *ebiten.Image) {
	if g.showItemActions {
		// Define menu window parameters
		menuWidth, menuHeight := 200, 100
		menuX, menuY := (screen.Bounds().Dx()-menuWidth)/2, (screen.Bounds().Dy()-menuHeight)/2

		drawWindowWithBorder(screen, menuX, menuY, menuWidth, menuHeight, 255)

		// Draw menu actions
		var actions []string
		item := g.state.Player.Inventory[g.selectedItemIndex]

		if equipableItem, isEquipable := item.(Equipable); isEquipable {
			// Assume function isEquipped returns true if the item is equipped, false otherwise
			if isEquipped(g.state.Player.EquippedItems[:], equipableItem) {
				actions = []string{"はずす", "投げる", "置く", "説明"}
			} else {
				actions = []string{"装備", "投げる", "置く", "説明"}
			}
		} else {
			actions = []string{"使う", "投げる", "置く", "説明"}
		}

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

func (g *Game) drawUseIdentifyItemWindow(screen *ebiten.Image) {
	windowX, windowY, windowWidth, windowHeight := 100, 50, 100, 25 // Adjust these values as needed
	drawWindowWithBorder(screen, windowX, windowY, windowWidth, windowHeight, 127)

	text.Draw(screen, "どれを？", mplusNormalFont, windowX+10, windowY+20, color.White)
}

func (g *Game) drawInventoryWindow(screen *ebiten.Image) error {

	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	windowWidth, windowHeight := 400, 300
	windowX, windowY := (screenWidth-windowWidth)/2, (screenHeight-windowHeight)/2

	drawWindowWithBorder(screen, windowX, windowY, windowWidth, windowHeight, 127)

	// Draw items
	const itemsPerColumn = 10 // 1列に表示するアイテムの数
	const columnWidth = 180   // 列の幅 (ピクセル)

	if len(g.state.Player.Inventory) > 0 {
		for i, item := range g.state.Player.Inventory {
			// アイテムが識別されているかどうかを判断し、表示するテキストを設定
			var itemText string
			var textColor color.Color = color.White // デフォルトのテキストカラーは白
			if identifiableItem, ok := item.(Identifiable); ok && !identifiableItem.IsIdentified() {
				// アイテムが識別されていなければ、GetName()を使って名前を取得し、テキストカラーを黄色に設定
				itemText = identifiableItem.GetName()
				textColor = color.RGBA{0xff, 0xff, 0x00, 0xff} // 黄色
			} else {
				// アイテムが識別されているか、識別可能な型ではない場合はSharpnessを含む名前を取得
				itemText = getItemNameWithSharpness(item)
			}

			// 現在の列と行の計算
			column := i / itemsPerColumn
			row := i % itemsPerColumn

			// アイテムテキストの描画位置の計算
			x := windowX + 30 + column*columnWidth
			y := windowY + 30 + row*25

			text.Draw(screen, itemText, mplusNormalFont, x, y, textColor) // 色を変更

			// Check if the item is equipped and draw "E" if it is
			if equipableItem, ok := item.(Equipable); ok {
				if isEquipped(g.state.Player.EquippedItems[:], equipableItem) {
					var dr font.Drawer
					dr.Dst = screen
					dr.Src = image.NewUniform(color.White)
					dr.Face = mplusNormalFont
					dr.Dot = fixed.Point26_6{
						X: fixed.I(x),
						Y: fixed.I(y),
					}
					// Measure the width of itemText in pixels
					textBounds, _ := dr.BoundString(itemText)
					textWidth := textBounds.Max.X - textBounds.Min.X
					text.Draw(screen, "E", mplusNormalFont, x+int(textWidth)/64+10, y, color.White) // Adjust the x coordinate based on the width of itemText and a small offset
				}
			}

			if i == g.selectedItemIndex {
				// Step 3: Draw the pointer next to the selected item
				pointerText := "→"
				text.Draw(screen, pointerText, mplusNormalFont, x-20, y, color.White)
			}
		}
	} else {
		text.Draw(screen, "何も持っていない", mplusNormalFont, windowX+10, windowY+20, color.White)
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

			// ColorScaleのインスタンスを作成
			var colorScale ebiten.ColorScale

			// Brightnessに基づいて色のスケールを設定
			colorScale.Scale(float32(tile.Brightness), float32(tile.Brightness), float32(tile.Brightness), 1)

			// ColorScaleを適用
			opts.ColorScale = colorScale

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

func (g *Game) getItemImage(item Item) *ebiten.Image {
	var img *ebiten.Image
	switch item.GetType() {
	case "Kane":
		img = g.kaneImg
	case "Card":
		img = g.cardImg
	case "Mintia":
		img = g.mintiaImg
	case "Weapon":
		img = g.weaponImg
	case "Armor":
		img = g.armorImg
	case "Sausage":
		img = g.sausageImg
	case "Arrow":
		img = g.arrowImg
	case "Cane":
		img = g.caneImg
	case "Effect":
		img = g.effectImg
	case "Accessory":
		img = g.accessoryImg
	}
	return img
}

func (g *Game) DrawThrownItem(screen *ebiten.Image, offsetX, offsetY int) {

	if g.ThrownItem.Image != nil {
		// Check if the ThrownItem is of type Arrow
		if _, ok := g.ThrownItem.Item.(*Arrow); ok && g.dPressed {
			opts := &ebiten.DrawImageOptions{}

			// Determine the rotation angle based on the player's direction
			var angle float64
			switch g.state.Player.Direction {
			case Up:
				angle = math.Pi // 180 degrees in radians
			case Down:
				angle = 0 // No rotation
			case Left:
				angle = math.Pi / 2 // 90 degrees in radians
			case Right:
				angle = -math.Pi / 2 // -90 degrees in radians
			case UpLeft:
				angle = 3 * math.Pi / 4 // 135 degrees in radians
			case UpRight:
				angle = -3 * math.Pi / 4 // -135 degrees in radians
			case DownLeft:
				angle = math.Pi / 4 // 45 degrees in radians
			case DownRight:
				angle = -math.Pi / 4 // -45 degrees in radians
			}

			// Rotate the geometry matrix around the center of the image
			w, h := g.ThrownItem.Image.Bounds().Dx(), g.ThrownItem.Image.Bounds().Dy()
			opts.GeoM.Translate(float64(-w)/2, float64(-h)/2)                                                       // Move the origin to the center of the image
			opts.GeoM.Rotate(angle)                                                                                 // Rotate
			opts.GeoM.Translate(float64(w)/2, float64(h)/2)                                                         // Move the origin back
			opts.GeoM.Translate(float64(g.ThrownItem.X*tileSize+offsetX), float64(g.ThrownItem.Y*tileSize+offsetY)) // Translate the geometry matrix to the item's position
			// Draw the image
			screen.DrawImage(g.ThrownItem.Image, opts)
		} else {
			// If it's not an Arrow, draw the image without rotation
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(g.ThrownItem.X*tileSize+offsetX), float64(g.ThrownItem.Y*tileSize+offsetY))
			screen.DrawImage(g.ThrownItem.Image, opts)
		}
	}
}

func (g *Game) DrawItems(screen *ebiten.Image, offsetX, offsetY int) {
	for _, item := range g.state.Items {
		itemX, itemY := item.GetPosition()

		// Check if the tile at the item's position is fully bright
		if g.state.Map[itemY][itemX].Brightness == 1.0 {
			img := g.getItemImage(item)
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(itemX*tileSize+offsetX), float64(itemY*tileSize+offsetY))
			screen.DrawImage(img, opts)
		}
	}
}

func (g *Game) getEnemyImage(enemy Enemy) *ebiten.Image {
	var img *ebiten.Image
	switch enemy.Type {
	case "Snake":
		img = g.snakeImg
	case "Shrimp":
		img = g.ebiImg
	}
	return img
}

func (g *Game) DrawEnemies(screen *ebiten.Image, offsetX, offsetY int) {
	for i := range g.state.Enemies {
		enemy := &g.state.Enemies[i]

		// Check if the tile at the enemy's position is fully bright
		if g.state.Map[enemy.Y][enemy.X].Brightness == 1.0 {

			// 敵のアニメーションを更新
			g.UpdateEnemyAnimation(enemy)

			// 敵の描画オフセットを計算
			enemyOffsetX, enemyOffsetY := g.CalculateEnemyOffset(enemy)
			enemyOffsetX += int(enemy.OffsetX)
			enemyOffsetY += int(enemy.OffsetY)

			enemyOffsetY += g.enemyYOffset // Y座標オフセットの適用

			img := g.getEnemyImage(*enemy)

			opts := &ebiten.DrawImageOptions{}
			// 敵の位置とオフセットを適用して敵を描画
			opts.GeoM.Translate(float64(enemy.X*tileSize+offsetX+enemyOffsetX), float64(enemy.Y*tileSize+offsetY+enemyOffsetY))
			screen.DrawImage(img, opts)
		}
	}
}

func drawBarWithBorder(screen *ebiten.Image, x, y, width, height int, barColor, borderColor color.Color) {
	// バーの背景を描画
	barBackground := ebiten.NewImage(width, height)
	barBackground.Fill(barColor)
	barOpts := &ebiten.DrawImageOptions{}
	barOpts.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(barBackground, barOpts)

	// 枠を描画
	borderSize := 1
	borderImg := ebiten.NewImage(width+2*borderSize, height+2*borderSize)
	borderImg.Fill(borderColor)

	// 上の枠
	borderOpts := &ebiten.DrawImageOptions{}
	borderOpts.GeoM.Translate(float64(x-borderSize), float64(y-borderSize))
	screen.DrawImage(borderImg.SubImage(image.Rect(0, 0, width+2*borderSize, borderSize)).(*ebiten.Image), borderOpts)

	// 左の枠
	borderOpts.GeoM.Reset()
	borderOpts.GeoM.Translate(float64(x-borderSize), float64(y))
	screen.DrawImage(borderImg.SubImage(image.Rect(0, 0, borderSize, height)).(*ebiten.Image), borderOpts)

	// 右の枠
	borderOpts.GeoM.Reset()
	borderOpts.GeoM.Translate(float64(x+width), float64(y))
	screen.DrawImage(borderImg.SubImage(image.Rect(0, 0, borderSize, height)).(*ebiten.Image), borderOpts)

	// 下の枠
	borderOpts.GeoM.Reset()
	borderOpts.GeoM.Translate(float64(x-borderSize), float64(y+height))
	screen.DrawImage(borderImg.SubImage(image.Rect(0, 0, width+2*borderSize, borderSize)).(*ebiten.Image), borderOpts)
}

func (g *Game) DrawHUD(screen *ebiten.Image) {
	screenWidth, _ := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Moves count
	MoveText := fmt.Sprintf("ターン数: %3d", g.moveCount)
	text.Draw(screen, MoveText, mplusNormalFont, screenWidth-130, 30, color.White)

	// Player HP
	playerHPText := fmt.Sprintf("HP:%3d/%3d", g.state.Player.Health, g.state.Player.MaxHealth)
	hpTextWidth := font.MeasureString(mplusSmallFont, playerHPText).Round() / 64
	text.Draw(screen, playerHPText, mplusSmallFont, (screenWidth/2)-(hpTextWidth+110), 20, color.White)

	hpBarMaxWidth := g.state.Player.MaxHealth / 4
	hpBarCurrentWidth := int(float64(hpBarMaxWidth) * (float64(g.state.Player.Health) / float64(g.state.Player.MaxHealth)))

	// 最大HPの値でベースとなる黒色のバーを作成
	baseHpBar := ebiten.NewImage(hpBarMaxWidth, 10)
	baseHpBar.Fill(color.RGBA{255, 0, 0, 127})

	// その値の割合として現在のHPを緑色のバーとして表示
	hpBar := ebiten.NewImage(hpBarCurrentWidth, 10)
	hpBar.Fill(color.RGBA{0, 255, 0, 255})

	// 黒色のバーを描画
	baseHpGeoM := ebiten.GeoM{}
	baseHpGeoM.Translate(float64((screenWidth/2)-30), 10)
	screen.DrawImage(baseHpBar, &ebiten.DrawImageOptions{GeoM: baseHpGeoM})

	// 緑色のバーを描画
	if hpBarCurrentWidth > 0 {
		HPgeoM := ebiten.GeoM{}
		HPgeoM.Translate(float64((screenWidth/2)-30), 10)
		screen.DrawImage(hpBar, &ebiten.DrawImageOptions{GeoM: HPgeoM})
	}

	// 枠を描画
	drawBarWithBorder(screen, (screenWidth/2)-30, 10, hpBarMaxWidth, 10, color.RGBA{0, 0, 0, 0}, color.White)

	// Player Satiety
	playerSatietyText := fmt.Sprintf("満腹度:%3d/%3d", g.state.Player.Satiety, g.state.Player.MaxSatiety)
	satietyTextWidth := font.MeasureString(mplusSmallFont, playerSatietyText).Round() / 64
	text.Draw(screen, playerSatietyText, mplusSmallFont, (screenWidth/2)-(satietyTextWidth+130), 35, color.White)

	satietyBarMaxWidth := g.state.Player.MaxSatiety
	satietyBarCurrentWidth := int(float64(satietyBarMaxWidth) * (float64(g.state.Player.Satiety) / float64(g.state.Player.MaxSatiety)))

	// 満腹度の最大値でベースとなる黒色のバーを作成
	baseSatietyBar := ebiten.NewImage(satietyBarMaxWidth, 10)
	baseSatietyBar.Fill(color.Black)

	// その値の割合として現在の満腹度を黄色のバーとして表示
	if g.state.Player.Satiety > 0 {
		satietyBar := ebiten.NewImage(satietyBarCurrentWidth, 10)
		satietyBar.Fill(color.RGBA{255, 255, 0, 255})

		// 黒色のバーを描画
		baseSatietyGeoM := ebiten.GeoM{}
		baseSatietyGeoM.Translate(float64((screenWidth/2)-30), 25)
		screen.DrawImage(baseSatietyBar, &ebiten.DrawImageOptions{GeoM: baseSatietyGeoM})

		// 黄色のバーを描画
		if satietyBarCurrentWidth > 0 {
			STgeoM := ebiten.GeoM{}
			STgeoM.Translate(float64((screenWidth/2)-30), 25)
			screen.DrawImage(satietyBar, &ebiten.DrawImageOptions{GeoM: STgeoM})
		}
	}

	// 枠を描画
	drawBarWithBorder(screen, (screenWidth/2)-30, 25, satietyBarMaxWidth, 10, color.RGBA{0, 0, 0, 0}, color.White)

	// Player Attack Power
	playerAttackPowerText := fmt.Sprintf("攻撃力: %3d", g.state.Player.AttackPower)
	text.Draw(screen, playerAttackPowerText, mplusNormalFont, screenWidth-130, 50, color.White)

	// Player Defense Power
	playerDefensePowerText := fmt.Sprintf("防御力: %3d", g.state.Player.DefensePower)
	text.Draw(screen, playerDefensePowerText, mplusNormalFont, screenWidth-130, 70, color.White)

	// Player Power
	playerPowerText := fmt.Sprintf("パワー: %2d/%2d", g.state.Player.Power, g.state.Player.MaxPower)
	text.Draw(screen, playerPowerText, mplusNormalFont, screenWidth-130, 90, color.White)

	// Player Experience Points
	playerExpText := fmt.Sprintf("経験値: %3d", g.state.Player.ExperiencePoints)
	text.Draw(screen, playerExpText, mplusNormalFont, screenWidth-130, 110, color.White)

	// Player Cash
	playerCashText := fmt.Sprintf("所持金：%5d円", g.state.Player.Cash)
	text.Draw(screen, playerCashText, mplusNormalFont, screenWidth-130, 130, color.White)

	yCoordinate := 110 // Initial Y-coordinate updated to position below the cash text

	for i, equippedItem := range g.state.Player.EquippedItems {
		equippedItemName := "なし"
		sharpnessText := ""

		// Check if the equipped item is not nil
		if equippedItem != nil {
			if arrowItem, ok := equippedItem.(*Arrow); ok {
				// If the equipped item is of type *Arrow, format the name with shot count
				equippedItemName = fmt.Sprintf("%d本の%s", arrowItem.ShotCount, arrowItem.GetName())
			} else {
				// For other item types, just get the name
				equippedItemName = equippedItem.GetName()

				// Check if the equipped item is of type *Weapon or *Armor to display sharpness
				if weaponItem, ok := equippedItem.(*Weapon); ok && weaponItem.Sharpness != 0 {
					sharpnessText = fmt.Sprintf("%+d", weaponItem.Sharpness) // %+d will include the sign for negative and positive numbers
				} else if armorItem, ok := equippedItem.(*Armor); ok && armorItem.Sharpness != 0 {
					sharpnessText = fmt.Sprintf("%+d", armorItem.Sharpness) // %+d will include the sign for negative and positive numbers
				}
			}
		}

		equippedItemText := fmt.Sprintf("装備%d: %s%s", i+1, equippedItemName, sharpnessText) // i+1 to display item number starting from 1
		text.Draw(screen, equippedItemText, mplusMediumFont, 10, yCoordinate, color.White)
		yCoordinate += 15 // Increment the Y-coordinate to position text below the previous item
	}

	// Player Traps
	playerTrapName := "なし"
	if g.state.Player.SetTrap != nil {
		playerTrapName = g.state.Player.SetTrap.GetName()
		playerTrapName = strings.ReplaceAll(playerTrapName, "のカード", "") // "のカード" を空の文字列で置き換え
	}
	playerTrapText := fmt.Sprintf("罠: %s", playerTrapName)
	text.Draw(screen, playerTrapText, mplusMediumFont, 10, 190, color.White)

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

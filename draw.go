//go:build !test

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

var (
	mplusNormalFont font.Face
	mplusMediumFont font.Face
	mplusSmallFont  font.Face
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
	// 設定でミニマップ表示がOFFの場合は描画しない
	if !g.settings.ShowMiniMap {
		return
	}

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
					// プレイヤーが目潰し状態の場合、階段をミニマップに表示しない
					if g.state.Player.StatusAilments.Blind > 0 {
						// 目潰し状態の時は階段を通常のタイルと同じように表示
						g.miniMap.DrawImage(miniMapTile, opts)
					} else {
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
					}
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

	// プレイヤーが目潰し状態でない場合、アイテムをミニマップに表示
	if g.state.Player.StatusAilments.Blind == 0 {
		// ゲームのアイテムリストをループして、ShowOnMiniMapがtrueのアイテムをミニマップに描画
		for _, item := range g.state.Items {
			if item.GetShowOnMiniMap() {
				itemX, itemY := item.GetPosition()
				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(float64(itemX*tilePixelSize), float64(itemY*tilePixelSize))
				g.miniMap.DrawImage(itemTile, opts)
			}
		}
	}

	enemyTile := ebiten.NewImage(tilePixelSize, tilePixelSize)
	enemyTile.Fill(color.RGBA{255, 0, 0, 128}) // Red semi-transparent

	//log.Printf("ShowOnMiniMap: %v", g.state.Enemies[0].GetShowOnMiniMap())

	// プレイヤーが目潰し状態でない場合、敵をミニマップに表示
	if g.state.Player.StatusAilments.Blind == 0 {
		for _, enemy := range g.state.Enemies {
			if enemy.GetShowOnMiniMap() {
				enemyX, enemyY := enemy.GetPosition()
				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(float64(enemyX*tilePixelSize), float64(enemyY*tilePixelSize))
				g.miniMap.DrawImage(enemyTile, opts)
			}
		}
	}

	// プレイヤーが目潰し状態でない場合、発見済みの罠をミニマップに「×」マークで表示
	if g.state.Player.StatusAilments.Blind == 0 {
		for _, trap := range g.state.MapTraps {
			if trap.Discovered {
				trapX, trapY := trap.X, trap.Y

				// 「×」マークを描画するための準備
				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(float64(trapX*tilePixelSize), float64(trapY*tilePixelSize))

				// 「×」マーク用のタイルを作成
				trapTile := ebiten.NewImage(tilePixelSize, tilePixelSize)
				trapTile.Fill(color.RGBA{0, 0, 0, 0}) // 透明で初期化

				// 3×3ピクセルで「×」パターンを手動設定
				redColor := color.RGBA{255, 0, 0, 255}

				if tilePixelSize >= 3 {
					// 3×3の場合の「×」パターン
					// X . X
					// . X .
					// X . X
					trapTile.Set(0, 0, redColor) // 左上
					trapTile.Set(2, 0, redColor) // 右上
					trapTile.Set(1, 1, redColor) // 中央
					trapTile.Set(0, 2, redColor) // 左下
					trapTile.Set(2, 2, redColor) // 右下
				} else {
					// 小さいサイズの場合は対角線で描画
					for i := 0; i < tilePixelSize; i++ {
						trapTile.Set(i, i, redColor)
						trapTile.Set(tilePixelSize-1-i, i, redColor)
					}
				}

				g.miniMap.DrawImage(trapTile, opts)
			}
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
		// プレイヤーが死亡している場合は、showDescriptionをfalseにしない
		if !g.playerDead {
			g.showDescription = false
		}
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

func (g *Game) drawGroundItemDescription(screen *ebiten.Image) {
	if g.showGroundItemDescription {
		screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
		descriptionWindowWidth, descriptionWindowHeight := 500, 120
		windowX, windowY := (screenWidth-descriptionWindowWidth)/2, screenHeight-descriptionWindowHeight-10

		drawWindowWithBorder(screen, windowX, windowY, descriptionWindowWidth, descriptionWindowHeight, 255)

		// Draw description text
		text.Draw(screen, g.groundItemDescriptionText, mplusNormalFont, windowX+10, windowY+20, color.White)
	}
}

func (g *Game) DrawGroundItem(screen *ebiten.Image) {
	if g.ShowGroundItem {
		screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
		itemWindowWidth, itemWindowHeight := 400, 26
		itemwindowX, itemwindowY := (screenWidth-itemWindowWidth)/2, (screenHeight-itemWindowHeight)/2
		actionWindowWidth, actionWindowHeight := 100, 110
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
			var actions []string
			// g.currentGroundItemがequipableItemであるかどうかをチェック
			if _, ok := g.currentGroundItem.(Equipable); ok {
				// Assume function isEquipped returns true if the item is equipped, false otherwise
				actions = []string{"拾う", "交換", "装備", "投げる", "説明"}
			} else {
				actions = []string{"拾う", "交換", "使う", "投げる", "説明"}
			}
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
			if g.state.Player.IsEquipped(equipableItem) {
				if _, isArrow := equipableItem.(*Arrow); isArrow {
					actions = []string{"はずす", "撃つ", "投げる", "置く", "説明"}
				} else {
					actions = []string{"はずす", "投げる", "置く", "説明"}
				}
			} else {
				if _, isArrow := equipableItem.(*Arrow); isArrow {
					actions = []string{"装備", "撃つ", "投げる", "置く", "説明"}
				} else {
					actions = []string{"装備", "投げる", "置く", "説明"}
				}
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

	// 絞り込み中のカテゴリと操作説明をウィンドウ下部に表示
	footerText := "【" + categoryLabel(g.inventoryFilter) + "】 F:絞り込み S:整頓 N:名前"
	text.Draw(screen, footerText, mplusNormalFont, windowX+15, windowY+windowHeight-8, color.NRGBA{200, 200, 200, 255})

	indices := g.normalizeInventoryView()

	if len(indices) > 0 {
		for displayPos, i := range indices {
			item := g.state.Player.Inventory[i]

			// 表示名を取得（未識別で任意名があればそれを表示）
			itemText, unidentified := g.inventoryItemLabel(item)
			var textColor color.Color = color.White
			if unidentified {
				textColor = color.RGBA{0xff, 0xff, 0x00, 0xff} // 未識別は黄色
			}

			// もしiの値がg.tmpSelectedItemIndexと等しければ、textColorを灰色に設定
			if i == g.tmpSelectedItemIndex {
				textColor = color.RGBA{0x80, 0x80, 0x80, 0xff} // 灰色
			}

			// 表示位置は絞り込み後のリスト内の順序で計算
			column := displayPos / itemsPerColumn
			row := displayPos % itemsPerColumn

			// アイテムテキストの描画位置の計算
			x := windowX + 30 + column*columnWidth
			y := windowY + 30 + row*25

			text.Draw(screen, itemText, mplusNormalFont, x, y, textColor) // 色を変更

			// Check if the item is equipped and draw "E" if it is
			if equipableItem, ok := item.(Equipable); ok {
				if g.state.Player.IsEquipped(equipableItem) {
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

// 未識別アイテムの名前入力ウィンドウを描画する
func (g *Game) drawNameInputWindow(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	windowWidth, windowHeight := 400, 380
	windowX := (screenWidth - windowWidth) / 2
	windowY := (screenHeight - windowHeight) / 2

	drawWindowWithBorder(screen, windowX, windowY, windowWidth, windowHeight, 240)

	// タイトルと入力中の名前
	text.Draw(screen, "名前を入力", mplusMediumFont, windowX+15, windowY+30, color.White)
	nameText := g.nameInput.String()
	if len(g.nameInput.Runes) < maxItemNameLength {
		nameText += "_"
	}
	text.Draw(screen, "名前: "+nameText, mplusNormalFont, windowX+15, windowY+60, color.NRGBA{255, 255, 0, 255})

	// 五十音グリッド
	cellWidth, cellHeight := 36, 30
	gridX := windowX + 20
	gridY := windowY + 80
	for i, r := range kanaChars {
		col := i % kanaGridColumns
		row := i / kanaGridColumns
		x := gridX + col*cellWidth
		y := gridY + row*cellHeight + 20

		// カーソル位置の文字をハイライト表示
		if i == g.nameInputCursor {
			highlight := ebiten.NewImage(cellWidth-4, cellHeight-4)
			highlight.Fill(color.NRGBA{100, 100, 200, 160})
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(x-6), float64(y-cellHeight+10))
			screen.DrawImage(highlight, opts)
		}

		text.Draw(screen, string(r), mplusNormalFont, x, y, color.White)
	}

	// 操作説明
	text.Draw(screen, "Z:入力  X:削除(空で閉じる)  Enter:決定", mplusNormalFont, windowX+15, windowY+windowHeight-15, color.NRGBA{200, 200, 200, 255})
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

			// プレイヤーが目潰し状態で、タイルが階段の場合の処理
			if tile.Type == "stairs" && g.state.Player.StatusAilments.Blind > 0 {
				// プレイヤーが通路にいるかどうかを確認
				playerX, playerY := g.state.Player.X, g.state.Player.Y
				playerTile := g.state.Map[playerY][playerX]

				if playerTile.Type == "corridor" {
					// プレイヤーが通路にいる場合：すべての階段を床タイルのみで表示
					floorOpts := &ebiten.DrawImageOptions{}
					floorOpts.GeoM.Translate(float64(x*tileSize+offsetX), float64(y*tileSize+offsetY))
					floorOpts.ColorScale = colorScale
					screen.DrawImage(g.tilesetImg.SubImage(image.Rect(2*tileSize, 0, 3*tileSize, tileSize)).(*ebiten.Image), floorOpts)
				} else {
					// プレイヤーが部屋にいる場合：床タイル+「？」マークを表示
					floorOpts := &ebiten.DrawImageOptions{}
					floorOpts.GeoM.Translate(float64(x*tileSize+offsetX), float64(y*tileSize+offsetY))
					floorOpts.ColorScale = colorScale
					screen.DrawImage(g.tilesetImg.SubImage(image.Rect(2*tileSize, 0, 3*tileSize, tileSize)).(*ebiten.Image), floorOpts)
					// その上にhatena.pngを描画
					hatenaOpts := &ebiten.DrawImageOptions{}
					hatenaOpts.GeoM.Translate(float64(x*tileSize+offsetX), float64(y*tileSize+offsetY))
					hatenaOpts.ColorScale = colorScale
					screen.DrawImage(g.hatenaImg, hatenaOpts)
				}
			} else {
				screen.DrawImage(g.tilesetImg.SubImage(image.Rect(srcX, srcY, srcX+tileSize, srcY+tileSize)).(*ebiten.Image), opts)
			}
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
	case "Potion":
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
			var img *ebiten.Image
			// プレイヤーが目潰し状態の場合、全てのアイテムをhatena.pngで表示
			if g.state.Player.StatusAilments.Blind > 0 {
				img = g.hatenaImg
			} else {
				img = g.getItemImage(item)
			}
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(itemX*tileSize+offsetX), float64(itemY*tileSize+offsetY))
			screen.DrawImage(img, opts)
		}
	}
}

func (g *Game) DrawMapTraps(screen *ebiten.Image, offsetX, offsetY int) {
	for _, trap := range g.state.MapTraps {
		// 罠が発見済みの場合のみ描画
		if trap.Discovered {
			trapX, trapY := trap.X, trap.Y

			// Check if the tile at the trap's position is fully bright
			if g.state.Map[trapY][trapX].Brightness == 1.0 {
				var img *ebiten.Image
				// 罠の種類に応じて画像を選択（現在は睡眠ガスの罠のみ）
				switch trap.Name {
				case "睡眠ガスの罠":
					img = g.sleepTrapImg
				case "毒矢の罠":
					img = g.poisonArrowTrapImg
				case "鈍足の罠":
					img = g.slowTrapImg
				case "地雷":
					img = g.mineTrapImg
				default:
					img = g.sleepTrapImg // デフォルトで睡眠ガスの罠の画像を使用
				}

				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(float64(trapX*tileSize+offsetX), float64(trapY*tileSize+offsetY))
				screen.DrawImage(img, opts)
			}
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
	case "Mamuru":
		img = g.mamuruImg
	case "Honey":
		img = g.honeyImg
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

			// 睡眠状態（通常・仮眠）と金縛り状態の敵は上下アニメーションを適用しない（封印状態は通常通りアニメーション）
			if enemy.StatusAilments.Sleep == 0 && !enemy.StatusAilments.Paralysis {
				enemyOffsetY += g.enemyYOffset // Y座標オフセットの適用
			}

			var img *ebiten.Image
			// プレイヤーが目潰し状態の場合、全ての敵をebisan.pngで表示
			if g.state.Player.StatusAilments.Blind > 0 {
				img = g.playerImg // ebisan.png
			} else {
				img = g.getEnemyImage(*enemy)
			}

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
	// HPが0の場合は幅1の最小バーを作成（エラー回避）
	if hpBarCurrentWidth <= 0 {
		hpBarCurrentWidth = 1
	}
	hpBar := ebiten.NewImage(hpBarCurrentWidth, 10)
	if g.state.Player.Health > 0 {
		hpBar.Fill(color.RGBA{0, 255, 0, 255})
	} else {
		hpBar.Fill(color.RGBA{255, 0, 0, 255}) // HPが0の場合は赤色
	}

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

	// 満腹度0時の点滅処理
	var satietyTextColor color.Color = color.White
	if g.state.Player.Satiety == 0 {
		// 0.5秒周期で赤と白を切り替え
		if g.starvationBlinkTimer < 0.5 {
			satietyTextColor = color.RGBA{255, 0, 0, 255} // 赤
		} else {
			satietyTextColor = color.White // 白
		}
	}
	text.Draw(screen, playerSatietyText, mplusSmallFont, (screenWidth/2)-(satietyTextWidth+130), 35, satietyTextColor)

	satietyBarMaxWidth := g.state.Player.MaxSatiety
	satietyBarCurrentWidth := int(float64(satietyBarMaxWidth) * (float64(g.state.Player.Satiety) / float64(g.state.Player.MaxSatiety)))

	// 満腹度の最大値でベースとなるバーを作成
	baseSatietyBar := ebiten.NewImage(satietyBarMaxWidth, 10)
	if g.state.Player.Satiety > 0 {
		baseSatietyBar.Fill(color.Black) // 通常時は黒色
	} else {
		// 満腹度0時の点滅処理（背景バー）
		if g.starvationBlinkTimer < 0.5 {
			baseSatietyBar.Fill(color.RGBA{255, 0, 0, 255}) // 赤色
		} else {
			baseSatietyBar.Fill(color.White) // 白色
		}
	}

	// その値の割合として現在の満腹度を黄色のバーとして表示
	// 満腹度が0の場合は幅1の最小バーを作成（エラー回避）
	if satietyBarCurrentWidth <= 0 {
		satietyBarCurrentWidth = 1
	}
	satietyBar := ebiten.NewImage(satietyBarCurrentWidth, 10)
	if g.state.Player.Satiety > 0 {
		satietyBar.Fill(color.RGBA{255, 255, 0, 255})
	} else {
		// 満腹度0時の点滅処理（バー）
		if g.starvationBlinkTimer < 0.5 {
			satietyBar.Fill(color.RGBA{255, 0, 0, 255}) // 赤色
		} else {
			satietyBar.Fill(color.White) // 白色
		}
	}

	// 黒色のバーを描画
	baseSatietyGeoM := ebiten.GeoM{}
	baseSatietyGeoM.Translate(float64((screenWidth/2)-30), 25)
	screen.DrawImage(baseSatietyBar, &ebiten.DrawImageOptions{GeoM: baseSatietyGeoM})

	// 満腹度バーを描画
	STgeoM := ebiten.GeoM{}
	STgeoM.Translate(float64((screenWidth/2)-30), 25)
	screen.DrawImage(satietyBar, &ebiten.DrawImageOptions{GeoM: STgeoM})

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

	// Display equipment using new system
	// Weapon
	weaponName := "なし"
	weaponSharpness := ""
	if g.state.Player.EquippedWeapon != nil {
		weaponName = g.state.Player.EquippedWeapon.GetName()
		if g.state.Player.EquippedWeapon.Sharpness != 0 {
			weaponSharpness = fmt.Sprintf("%+d", g.state.Player.EquippedWeapon.Sharpness)
		}
	}
	weaponText := fmt.Sprintf("武器: %s%s", weaponName, weaponSharpness)
	text.Draw(screen, weaponText, mplusMediumFont, 10, yCoordinate, color.White)
	yCoordinate += 15

	// Armor
	armorName := "なし"
	armorSharpness := ""
	if g.state.Player.EquippedArmor != nil {
		armorName = g.state.Player.EquippedArmor.GetName()
		if g.state.Player.EquippedArmor.Sharpness != 0 {
			armorSharpness = fmt.Sprintf("%+d", g.state.Player.EquippedArmor.Sharpness)
		}
	}
	armorText := fmt.Sprintf("防具: %s%s", armorName, armorSharpness)
	text.Draw(screen, armorText, mplusMediumFont, 10, yCoordinate, color.White)
	yCoordinate += 15

	// Arrow
	arrowName := "なし"
	if g.state.Player.EquippedArrow != nil {
		arrowName = fmt.Sprintf("%d本の%s", g.state.Player.EquippedArrow.ShotCount, g.state.Player.EquippedArrow.GetName())
	}
	arrowText := fmt.Sprintf("矢: %s", arrowName)
	text.Draw(screen, arrowText, mplusMediumFont, 10, yCoordinate, color.White)
	yCoordinate += 15

	// Accessories
	for i, accessory := range g.state.Player.EquippedAccessories {
		accessoryName := "なし"
		if accessory != nil {
			accessoryName = accessory.GetName()
		}
		accessoryText := fmt.Sprintf("装身具%d: %s", i+1, accessoryName)
		text.Draw(screen, accessoryText, mplusMediumFont, 10, yCoordinate, color.White)
		yCoordinate += 15
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

	// Turn count and spawn info
	turnText := fmt.Sprintf("ターン: %d", g.turnCount)
	text.Draw(screen, turnText, mplusNormalFont, 10, 70, color.White)

	nextSpawnTurns := g.lastSpawnTurn + g.spawnInterval - g.turnCount
	if nextSpawnTurns < 0 {
		nextSpawnTurns = 0
	}
	spawnInfoText := fmt.Sprintf("敵数: %d/19 次回湧き: %dターン後", len(g.state.Enemies), nextSpawnTurns)
	text.Draw(screen, spawnInfoText, mplusNormalFont, 10, 90, color.White)

	// Player Coordinate
	playerCoordinateText := fmt.Sprintf("座標: (%d, %d)", g.state.Player.X, g.state.Player.Y)
	text.Draw(screen, playerCoordinateText, mplusNormalFont, 10, 210, color.White) // x座標とy座標を直接指定

	// Player Room
	playerRoomText := logCurrentRoom(g.state.Player, g.rooms)
	text.Draw(screen, playerRoomText, mplusNormalFont, 10, 230, color.White) // x座標とy座標を直接指定

	// 現在の状態異常と残りターンを表示する
	statusText := formatPlayerStatus(g.state.Player.StatusAilments)
	text.Draw(screen, "状態: "+statusText, mplusNormalFont, 10, 250, color.White)

}

// メニューウィンドウの描画
func (g *Game) drawMenuWindow(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// メニューウィンドウのサイズと位置
	menuWidth := 300
	menuHeight := 200
	menuX := (screenWidth - menuWidth) / 2
	menuY := (screenHeight - menuHeight) / 2

	// ウィンドウ背景
	menuWindow := ebiten.NewImage(menuWidth, menuHeight)
	menuWindow.Fill(color.NRGBA{50, 50, 50, 220}) // 半透明の暗い背景

	// ウィンドウの枠線
	for i := 0; i < 2; i++ {
		// 上下の枠線
		for x := 0; x < menuWidth; x++ {
			menuWindow.Set(x, i, color.NRGBA{255, 255, 255, 255})
			menuWindow.Set(x, menuHeight-1-i, color.NRGBA{255, 255, 255, 255})
		}
		// 左右の枠線
		for y := 0; y < menuHeight; y++ {
			menuWindow.Set(i, y, color.NRGBA{255, 255, 255, 255})
			menuWindow.Set(menuWidth-1-i, y, color.NRGBA{255, 255, 255, 255})
		}
	}

	// ウィンドウを描画
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(menuX), float64(menuY))
	screen.DrawImage(menuWindow, opts)

	// メニュー項目のテキスト
	menuItems := [][]string{
		{"道具", "足元"},
		{"設定", "中断"},
	}

	// テキストの描画位置
	itemWidth := menuWidth / 2
	itemHeight := menuHeight / 2

	for row := 0; row < 2; row++ {
		for col := 0; col < 2; col++ {
			// テキストの描画位置を計算
			textX := menuX + col*itemWidth + itemWidth/2 - 30   // 中央寄せのための調整
			textY := menuY + row*itemHeight + itemHeight/2 + 10 // 中央寄せのための調整

			// 選択中の項目をハイライト表示
			textColor := color.White
			if g.menuSelectedRow == row && g.menuSelectedCol == col {
				// 選択中の項目の背景をハイライト
				highlightWindow := ebiten.NewImage(itemWidth-10, 40)
				highlightWindow.Fill(color.NRGBA{100, 100, 200, 100})
				highlightOpts := &ebiten.DrawImageOptions{}
				highlightOpts.GeoM.Translate(float64(menuX+col*itemWidth+5), float64(menuY+row*itemHeight+itemHeight/2-10))
				screen.DrawImage(highlightWindow, highlightOpts)

				// 矢印を描画
				arrowText := "→"
				text.Draw(screen, arrowText, mplusMediumFont, textX-30, textY, color.NRGBA{255, 255, 0, 255})
			}

			// メニュー項目のテキストを描画
			text.Draw(screen, menuItems[row][col], mplusMediumFont, textX, textY, textColor)
		}
	}
}

// 空のインベントリメッセージの描画
func (g *Game) drawEmptyInventoryMessage(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// メッセージウィンドウのサイズと位置
	messageWidth := 300
	messageHeight := 100
	messageX := (screenWidth - messageWidth) / 2
	messageY := (screenHeight - messageHeight) / 2

	// ウィンドウ背景
	messageWindow := ebiten.NewImage(messageWidth, messageHeight)
	messageWindow.Fill(color.NRGBA{50, 50, 50, 220}) // 半透明の暗い背景

	// ウィンドウの枠線
	for i := 0; i < 2; i++ {
		// 上下の枠線
		for x := 0; x < messageWidth; x++ {
			messageWindow.Set(x, i, color.NRGBA{255, 255, 255, 255})
			messageWindow.Set(x, messageHeight-1-i, color.NRGBA{255, 255, 255, 255})
		}
		// 左右の枠線
		for y := 0; y < messageHeight; y++ {
			messageWindow.Set(i, y, color.NRGBA{255, 255, 255, 255})
			messageWindow.Set(messageWidth-1-i, y, color.NRGBA{255, 255, 255, 255})
		}
	}

	// ウィンドウを描画
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(messageX), float64(messageY))
	screen.DrawImage(messageWindow, opts)

	// メッセージテキストを描画
	messageText := "何も持っていない"
	textX := messageX + messageWidth/2 - 80 // 中央寄せのための調整
	textY := messageY + messageHeight/2 + 5
	text.Draw(screen, messageText, mplusMediumFont, textX, textY, color.White)
}

// 設定ウィンドウを描画する
func (g *Game) drawSettingsWindow(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	windowWidth, windowHeight := 340, 200
	windowX := (screenWidth - windowWidth) / 2
	windowY := (screenHeight - windowHeight) / 2

	drawWindowWithBorder(screen, windowX, windowY, windowWidth, windowHeight, 220)

	// タイトル
	text.Draw(screen, "設定", mplusMediumFont, windowX+10, windowY+30, color.White)

	// 設定項目と現在の値
	items := []struct {
		label string
		value bool
	}{
		{"フルスクリーン", g.settings.Fullscreen},
		{"ミニマップ表示", g.settings.ShowMiniMap},
	}

	lineHeight := 35
	for i, item := range items {
		y := windowY + 70 + i*lineHeight

		// 選択中の項目に矢印を表示
		labelColor := color.Color(color.White)
		if g.settingsSelectedIndex == i {
			text.Draw(screen, "→", mplusNormalFont, windowX+15, y, color.NRGBA{255, 255, 0, 255})
		}
		text.Draw(screen, item.label, mplusNormalFont, windowX+45, y, labelColor)

		valueText := "OFF"
		valueColor := color.Color(color.NRGBA{150, 150, 150, 255})
		if item.value {
			valueText = "ON"
			valueColor = color.NRGBA{100, 255, 100, 255}
		}
		text.Draw(screen, valueText, mplusNormalFont, windowX+windowWidth-70, y, valueColor)
	}

	// 操作説明
	text.Draw(screen, "↑↓:選択  ←→:切替  X:戻る", mplusNormalFont, windowX+15, windowY+windowHeight-20, color.NRGBA{200, 200, 200, 255})
}

// 中断確認ダイアログを描画する
func (g *Game) drawSuspendPrompt(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	windowWidth, windowHeight := 360, 110
	windowX := (screenWidth - windowWidth) / 2
	windowY := (screenHeight - windowHeight) / 2

	drawWindowWithBorder(screen, windowX, windowY, windowWidth, windowHeight, 220)

	text.Draw(screen, "冒険を中断して終了しますか？", mplusNormalFont, windowX+20, windowY+30, color.White)

	options := []string{"中断する", "やめる"}
	for i, option := range options {
		x := windowX + 50 + i*150
		y := windowY + 75
		if g.suspendSelectedOption == i {
			text.Draw(screen, "→", mplusNormalFont, x-25, y, color.NRGBA{255, 255, 0, 255})
		}
		text.Draw(screen, option, mplusNormalFont, x, y, color.White)
	}
}

// メッセージ履歴ウィンドウを描画する
func (g *Game) drawMessageLogWindow(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	windowWidth, windowHeight := 540, 420
	windowX := (screenWidth - windowWidth) / 2
	windowY := (screenHeight - windowHeight) / 2

	drawWindowWithBorder(screen, windowX, windowY, windowWidth, windowHeight, 220)

	// タイトルと操作説明
	text.Draw(screen, "メッセージ履歴", mplusMediumFont, windowX+10, windowY+30, color.White)
	text.Draw(screen, "↑↓:スクロール  X:閉じる", mplusNormalFont, windowX+windowWidth-220, windowY+30, color.NRGBA{200, 200, 200, 255})

	messages := g.messageLog.Visible(g.messageLogScroll, messageLogPageSize)
	if len(messages) == 0 {
		text.Draw(screen, "まだメッセージはありません", mplusNormalFont, windowX+20, windowY+70, color.White)
		return
	}

	lineHeight := 22
	for i, msg := range messages {
		text.Draw(screen, msg, mplusNormalFont, windowX+20, windowY+60+i*lineHeight, color.White)
	}

	// さかのぼれる方向を示すインジケーター
	if g.messageLogScroll < g.messageLog.MaxScroll(messageLogPageSize) {
		text.Draw(screen, "↑", mplusNormalFont, windowX+windowWidth-25, windowY+60, color.NRGBA{255, 255, 0, 255})
	}
	if g.messageLogScroll > 0 {
		text.Draw(screen, "↓", mplusNormalFont, windowX+windowWidth-25, windowY+60+(len(messages)-1)*lineHeight, color.NRGBA{255, 255, 0, 255})
	}
}

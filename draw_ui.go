//go:build !test

package main

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// インベントリ・メニュー・説明・各種ウィンドウの描画をまとめたファイル。

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

// メニューウィンドウの描画
func (g *Game) drawMenuWindow(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()

	// メニューウィンドウのサイズと位置
	menuWidth := 300
	menuHeight := 280
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

	// メニュー項目のテキスト（3行目はヘルプのみで全幅を使う）
	menuItems := [][]string{
		{"道具", "足元"},
		{"設定", "中断"},
		{"ヘルプ"},
	}

	// テキストの描画位置
	itemHeight := menuHeight / len(menuItems)

	for row := range menuItems {
		itemWidth := menuWidth / len(menuItems[row])
		for col := range menuItems[row] {
			// テキストの描画位置を計算
			textX := menuX + col*itemWidth + itemWidth/2 - 30   // 中央寄せのための調整
			textY := menuY + row*itemHeight + itemHeight/2 + 10 // 中央寄せのための調整

			// 選択中の項目をハイライト表示（1項目だけの行は列に関係なく選択扱い）
			textColor := color.White
			if g.menuSelectedRow == row && (len(menuItems[row]) == 1 || g.menuSelectedCol == col) {
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

// ヘルプウィンドウを描画する
func (g *Game) drawHelpWindow(screen *ebiten.Image) {
	screenWidth, screenHeight := screen.Bounds().Dx(), screen.Bounds().Dy()
	windowWidth, windowHeight := 540, 420
	windowX := (screenWidth - windowWidth) / 2
	windowY := (screenHeight - windowHeight) / 2

	drawWindowWithBorder(screen, windowX, windowY, windowWidth, windowHeight, 220)

	pages := helpPages()
	page := pages[g.helpPage]

	// タイトルとページ番号、操作説明
	title := fmt.Sprintf("%s (%d/%d)", page.Title, g.helpPage+1, len(pages))
	text.Draw(screen, title, mplusMediumFont, windowX+10, windowY+30, color.White)
	text.Draw(screen, "←→:ページ  ↑↓:スクロール  X:閉じる", mplusNormalFont, windowX+windowWidth-290, windowY+30, color.NRGBA{200, 200, 200, 255})

	lineHeight := 22
	lines := helpVisibleLines(page.Lines, g.helpScroll, helpPageSize)
	for i, line := range lines {
		text.Draw(screen, line, mplusNormalFont, windowX+20, windowY+60+i*lineHeight, color.White)
	}

	// スクロールできる方向を示すインジケーター
	if g.helpScroll > 0 {
		text.Draw(screen, "↑", mplusNormalFont, windowX+windowWidth-25, windowY+60, color.NRGBA{255, 255, 0, 255})
	}
	if g.helpScroll < helpMaxScroll(len(page.Lines), helpPageSize) {
		text.Draw(screen, "↓", mplusNormalFont, windowX+windowWidth-25, windowY+60+(len(lines)-1)*lineHeight, color.NRGBA{255, 255, 0, 255})
	}
}

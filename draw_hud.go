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
)

// HUD・ミニマップ・ステータスバーの描画をまとめたファイル。

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

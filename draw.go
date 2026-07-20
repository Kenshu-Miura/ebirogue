//go:build !test

package main

import (
	"image"
	"image/color"
	_ "image/png" // PNG画像を読み込むために必要
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
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
	case "Pot":
		img = g.potImg
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

var rangedRockEffectImg, rangedBlastEffectImg *ebiten.Image

// DrawRangedAttackEffect は敵の矢・投石・爆発弾を短時間表示する。
func (g *Game) DrawRangedAttackEffect(screen *ebiten.Image, offsetX, offsetY int) {
	effect := g.rangedAttackEffect
	if effect.Timer <= 0 {
		return
	}

	progress := 1 - effect.Timer/rangedAttackEffectDuration
	if progress < 0 {
		progress = 0
	} else if progress > 1 {
		progress = 1
	}
	originX := float64(effect.OriginX*tileSize + offsetX + tileSize/2)
	originY := float64(effect.OriginY*tileSize + offsetY + tileSize/2)
	targetX := float64(effect.TargetX*tileSize + offsetX + tileSize/2)
	targetY := float64(effect.TargetY*tileSize + offsetY + tileSize/2)

	drawProjectile := func(img *ebiten.Image, x, y, angle float64) {
		opts := &ebiten.DrawImageOptions{}
		w, h := img.Bounds().Dx(), img.Bounds().Dy()
		opts.GeoM.Translate(float64(-w)/2, float64(-h)/2)
		opts.GeoM.Rotate(angle)
		opts.GeoM.Translate(x, y)
		screen.DrawImage(img, opts)
	}

	switch effect.Kind {
	case RangedAttackArrow:
		x := originX + (targetX-originX)*progress
		y := originY + (targetY-originY)*progress
		angle := math.Atan2(targetY-originY, targetX-originX) - math.Pi/2
		drawProjectile(g.arrowImg, x, y, angle)
	case RangedAttackRock:
		if rangedRockEffectImg == nil {
			rangedRockEffectImg = ebiten.NewImage(10, 10)
			rangedRockEffectImg.Fill(color.RGBA{105, 105, 115, 255})
		}
		x := originX + (targetX-originX)*progress
		y := originY + (targetY-originY)*progress - math.Sin(math.Pi*progress)*tileSize
		drawProjectile(rangedRockEffectImg, x, y, progress*math.Pi*2)
	case RangedAttackExplosion:
		const flightEnd = 0.65
		if progress < flightEnd {
			flightProgress := progress / flightEnd
			x := originX + (targetX-originX)*flightProgress
			y := originY + (targetY-originY)*flightProgress
			drawProjectile(g.effectImg, x, y, 0)
			return
		}
		if rangedBlastEffectImg == nil {
			rangedBlastEffectImg = ebiten.NewImage(tileSize, tileSize)
			rangedBlastEffectImg.Fill(color.RGBA{255, 105, 20, 150})
		}
		fade := float32(1 - (progress-flightEnd)/(1-flightEnd))
		for dy := -effect.BlastRadius; dy <= effect.BlastRadius; dy++ {
			for dx := -effect.BlastRadius; dx <= effect.BlastRadius; dx++ {
				x, y := effect.TargetX+dx, effect.TargetY+dy
				if y < 0 || y >= len(g.state.Map) || x < 0 || x >= len(g.state.Map[y]) {
					continue
				}
				opts := &ebiten.DrawImageOptions{}
				opts.ColorScale.ScaleAlpha(fade)
				opts.GeoM.Translate(float64(x*tileSize+offsetX), float64(y*tileSize+offsetY))
				screen.DrawImage(rangedBlastEffectImg, opts)
			}
		}
	}
}

// 射線プレビュー用のマーカー画像（初回描画時に生成してキャッシュする）
var trajectoryDotImg, trajectoryLandingImg *ebiten.Image

// trajectoryPreviewSpec は射線プレビューを表示すべきかどうかと、
// その射程・壁のマス自体に到達するか（杖の魔法弾）を返す
func (g *Game) trajectoryPreviewSpec() (throwRange int, stopOnWallTile bool, active bool) {
	// インベントリのアクションメニューで「撃つ」「投げる」「使う（杖）」を選択中
	if g.showItemActions && g.selectedItemIndex >= 0 && g.selectedItemIndex < len(g.state.Player.Inventory) {
		item := g.state.Player.Inventory[g.selectedItemIndex]
		if _, isArrow := item.(*Arrow); isArrow {
			// 矢のメニューは「装備/はずす」「撃つ」「投げる」の並び
			if g.selectedActionIndex == 1 || g.selectedActionIndex == 2 {
				return 10, false, true
			}
			return 0, false, false
		}
		if _, isCane := item.(*Cane); isCane && g.selectedActionIndex == 0 { // 使う（魔法弾）
			return 30, true, true
		}
		if g.selectedActionIndex == 1 { // 投げる
			return 10, false, true
		}
		return 0, false, false
	}

	// 足元メニューで「投げる」「使う（杖）」を選択中
	if g.ShowGroundItem && g.currentGroundItem != nil && !g.showGroundItemDescription {
		if g.selectedGroundActionIndex == 3 { // 投げる
			return 10, false, true
		}
		if _, isCane := g.currentGroundItem.(*Cane); isCane && g.selectedGroundActionIndex == 2 { // 使う（魔法弾）
			return 30, true, true
		}
		return 0, false, false
	}

	// Aキーで向きを変えている間、装備中の矢の射線を表示する
	if ebiten.IsKeyPressed(ebiten.KeyA) && g.state.Player.EquippedArrow != nil &&
		!g.showInventory && !g.showMenu && !g.showSettings && !g.showSuspendPrompt &&
		!g.showMessageLog && !g.showHelp && !g.showStairsPrompt &&
		!g.isCombatActive && g.ThrownItem.Image == nil &&
		g.state.Player.StatusAilments.Sleep == 0 {
		return 10, false, true
	}

	return 0, false, false
}

// DrawTrajectoryPreview は矢・杖・投擲アイテムの射線と到達地点を使用前に表示する
func (g *Game) DrawTrajectoryPreview(screen *ebiten.Image, offsetX, offsetY int) {
	throwRange, stopOnWallTile, active := g.trajectoryPreviewSpec()
	if !active {
		return
	}
	dx, dy := directionToDelta(g.state.Player.Direction)
	if dx == 0 && dy == 0 {
		return
	}

	// 見えている（明るいマスにいる）敵だけを対象にして、未発見の敵の位置が漏れないようにする
	visibleEnemies := make([]Enemy, 0, len(g.state.Enemies))
	for _, enemy := range g.state.Enemies {
		if enemy.Y >= 0 && enemy.Y < len(g.state.Map) && enemy.X >= 0 && enemy.X < len(g.state.Map[0]) &&
			g.state.Map[enemy.Y][enemy.X].Brightness == 1.0 {
			visibleEnemies = append(visibleEnemies, enemy)
		}
	}

	path, landing, _ := computeTrajectory(g.state.Player.X, g.state.Player.Y, dx, dy, throwRange, g.state.Map, visibleEnemies, stopOnWallTile)

	if trajectoryDotImg == nil {
		trajectoryDotImg = ebiten.NewImage(8, 8)
		trajectoryDotImg.Fill(color.RGBA{255, 255, 100, 200})
		trajectoryLandingImg = ebiten.NewImage(22, 22)
		trajectoryLandingImg.Fill(color.RGBA{255, 120, 40, 160})
	}

	// 射線（到達地点の手前まで）を描画。暗いマスには表示しない
	for _, p := range path {
		if p == landing || g.state.Map[p.Y][p.X].Brightness != 1.0 {
			continue
		}
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(p.X*tileSize+offsetX+(tileSize-8)/2), float64(p.Y*tileSize+offsetY+(tileSize-8)/2))
		screen.DrawImage(trajectoryDotImg, opts)
	}

	// 到達地点を描画
	if landing.Y >= 0 && landing.Y < len(g.state.Map) && landing.X >= 0 && landing.X < len(g.state.Map[0]) &&
		g.state.Map[landing.Y][landing.X].Brightness == 1.0 {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(landing.X*tileSize+offsetX+(tileSize-22)/2), float64(landing.Y*tileSize+offsetY+(tileSize-22)/2))
		screen.DrawImage(trajectoryLandingImg, opts)
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
				case "サビの罠":
					img = g.rustTrapImg
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
	case "Harisenbow":
		img = g.harisenbowImg
	case "Ishigani":
		img = g.ishiganiImg
	case "BombUrchin":
		img = g.bakudanUniImg
	case "ThiefHermitCrab":
		img = g.kosodoroYadokariImg
	case "NigiriShrimp":
		img = g.nigiriEbiImg
	case "CurseCrab":
		img = g.noroiGaniImg
	case "PuppeteerJellyfish":
		img = g.ayatsuriKurageImg
	case "GhostShrimp":
		img = g.yureiEbiImg
	case "MantisShrimp":
		img = g.hayateShakoImg
	case "WarpJellyfish":
		img = g.warpKurageImg
	case "SwapOctopus":
		img = g.irekaeDakoImg
	case "TrapHermitCrab":
		img = g.wanashiYadokariImg
	case "MimicClam":
		img = g.mimicGaiImg
	}
	return img
}

func (g *Game) getEnemyDisguiseImage(enemy Enemy) *ebiten.Image {
	switch enemy.Disguise {
	case EnemyDisguiseItem:
		return g.potImg
	case EnemyDisguiseStairs:
		return g.tilesetImg.SubImage(image.Rect(4*tileSize, 0, 5*tileSize, tileSize)).(*ebiten.Image)
	default:
		return nil
	}
}

func (g *Game) DrawEnemies(screen *ebiten.Image, offsetX, offsetY int) {
	for i := range g.state.Enemies {
		enemy := &g.state.Enemies[i]

		// Check if the tile at the enemy's position is fully bright
		if g.state.Map[enemy.Y][enemy.X].Brightness == 1.0 {

			// 擬態中は道具・階段として静止表示する。
			if !isEnemyDisguised(*enemy) {
				g.UpdateEnemyAnimation(enemy)
			}

			// 敵の描画オフセットを計算
			enemyOffsetX, enemyOffsetY := g.CalculateEnemyOffset(enemy)
			enemyOffsetX += int(enemy.OffsetX)
			enemyOffsetY += int(enemy.OffsetY)

			// 睡眠状態（通常・仮眠）と金縛り状態の敵は上下アニメーションを適用しない（封印状態は通常通りアニメーション）
			if !isEnemyDisguised(*enemy) && enemy.StatusAilments.Sleep == 0 && !enemy.StatusAilments.Paralysis {
				enemyOffsetY += g.enemyYOffset // Y座標オフセットの適用
			}

			var img *ebiten.Image
			// プレイヤーが目潰し状態の場合、全ての敵をebisan.pngで表示
			if g.state.Player.StatusAilments.Blind > 0 {
				img = g.playerImg // ebisan.png
			} else if isEnemyDisguised(*enemy) {
				img = g.getEnemyDisguiseImage(*enemy)
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

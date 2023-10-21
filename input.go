package main

import (
	_ "image/png" // PNG画像を読み込むために必要
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (g *Game) OpenDoor() {
	playerX, playerY := g.state.Player.X, g.state.Player.Y
	directions := []struct{ dx, dy int }{
		{0, -1}, // Up
		{0, 1},  // Down
		{-1, 0}, // Left
		{1, 0},  // Right
	}
	for _, dir := range directions {
		tile := g.state.Map[playerY+dir.dy][playerX+dir.dx]
		if tile.Type == "door" {
			g.state.Map[playerY+dir.dy][playerX+dir.dx] = Tile{Type: "corridor"}
			g.MoveEnemies()
			g.IncrementMoveCount()
		}
	}
}

func (g *Game) handleItemActionsInput() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && g.selectedActionIndex > 0 {
		g.selectedActionIndex--
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) && g.selectedActionIndex < 3 {
		g.selectedActionIndex++
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		g.executeAction()
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		g.showItemActions = false // Toggle the item actions menu
		return nil
	}

	return nil
}

func (g *Game) handleInventoryNavigationInput() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && g.selectedItemIndex > 0 {
		g.selectedItemIndex--
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) && g.selectedItemIndex < len(g.state.Player.Inventory)-1 {
		g.selectedItemIndex++
	} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) && g.selectedItemIndex >= 10 {
		g.selectedItemIndex -= 10
	} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) && g.selectedItemIndex < len(g.state.Player.Inventory)-10 {
		g.selectedItemIndex += 10
	} else if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		g.showItemActions = true // Toggle the item actions menu
	}

	return nil
}

func (g *Game) handleItemDescriptionInput() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		g.showItemDescription = false // Toggle the item description
		return nil
	}

	return nil
}

func (g *Game) handleInventoryInput() error {
	cPressed := inpututil.IsKeyJustPressed(ebiten.KeyC)
	if cPressed {
		g.showInventory = true
		g.descriptionQueue = []string{} // g.descriptionQueueの中身をクリア
		return nil                      // Skip other updates when the inventory window is active
	}

	xPressed := inpututil.IsKeyJustPressed(ebiten.KeyX)

	if xPressed && g.showInventory && !g.showItemActions {
		g.showInventory = false
		return nil // Skip other updates when the inventory window is active
	}

	if g.showInventory {
		if g.showItemActions && !g.showItemDescription {
			return g.handleItemActionsInput()
		} else if !g.showItemActions && !g.showItemDescription {
			return g.handleInventoryNavigationInput()
		} else if g.showItemDescription {
			return g.handleItemDescriptionInput()
		}

		return nil // Skip other updates when the inventory window is active
	}

	return nil
}

func (g *Game) CheetHandleInput() (int, int) {
	var dx, dy int

	// キーの押下状態を取得
	upPressed := ebiten.IsKeyPressed(ebiten.KeyUp)
	downPressed := ebiten.IsKeyPressed(ebiten.KeyDown)
	leftPressed := ebiten.IsKeyPressed(ebiten.KeyLeft)
	rightPressed := ebiten.IsKeyPressed(ebiten.KeyRight)
	shiftPressed := ebiten.IsKeyPressed(ebiten.KeyShift) // Shiftキーが押されているかどうかをチェック
	aPressed := ebiten.IsKeyPressed(ebiten.KeyA)         // Aキーが押されているかどうかをチェック

	// 足踏みロジック
	if aPressed && time.Since(g.lastIncrement) >= 100*time.Millisecond {
		g.IncrementMoveCount()
		g.MoveEnemies()
		g.lastIncrement = time.Now() // lastIncrementの更新
	}

	arrowPressed := upPressed || downPressed || leftPressed || rightPressed

	// 矢印キーの押下ロジック
	if arrowPressed && time.Since(g.lastArrowPress) >= 180*time.Millisecond {

		if shiftPressed { // 斜め移動のロジック

			if upPressed && rightPressed {
				dy, dx = -1, 1
			} else if upPressed && leftPressed {
				dy, dx = -1, -1
			} else if downPressed && leftPressed {
				dy, dx = 1, -1
			} else if downPressed && rightPressed {
				dy, dx = 1, 1
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

			// 斜め移動のロジック
			if upPressed && rightPressed {
				dy, dx = -1, 1
			} else if upPressed && leftPressed {
				dy, dx = -1, -1
			} else if downPressed && leftPressed {
				dy, dx = 1, -1
			} else if downPressed && rightPressed {
				dy, dx = 1, 1
			}
		}
		g.lastArrowPress = time.Now() // lastArrowPressの更新
	}

	return dx, dy
}

func (g *Game) HandleInput() (int, int) {
	var dx, dy int

	// キーの押下状態を取得
	upPressed := ebiten.IsKeyPressed(ebiten.KeyUp)
	downPressed := ebiten.IsKeyPressed(ebiten.KeyDown)
	leftPressed := ebiten.IsKeyPressed(ebiten.KeyLeft)
	rightPressed := ebiten.IsKeyPressed(ebiten.KeyRight)
	shiftPressed := ebiten.IsKeyPressed(ebiten.KeyShift) // Shiftキーが押されているかどうかをチェック
	aPressed := ebiten.IsKeyPressed(ebiten.KeyA)         // Aキーが押されているかどうかをチェック

	// 足踏みロジック
	if aPressed && time.Since(g.lastIncrement) >= 180*time.Millisecond {
		g.IncrementMoveCount()
		g.MoveEnemies()
		g.lastIncrement = time.Now() // lastIncrementの更新
	}

	arrowPressed := upPressed || downPressed || leftPressed || rightPressed

	// 矢印キーの押下ロジック
	if arrowPressed && time.Since(g.lastArrowPress) >= 180*time.Millisecond {

		player := g.state.Player
		blockUp := g.state.Map[player.Y-1][player.X].Blocked
		blockDown := g.state.Map[player.Y+1][player.X].Blocked
		blockLeft := g.state.Map[player.Y][player.X-1].Blocked
		blockRight := g.state.Map[player.Y][player.X+1].Blocked

		if shiftPressed { // 斜め移動のロジック

			if upPressed && rightPressed && (!blockUp && !blockRight) {
				dy, dx = -1, 1
			} else if upPressed && leftPressed && (!blockUp && !blockLeft) {
				dy, dx = -1, -1
			} else if downPressed && leftPressed && (!blockDown && !blockLeft) {
				dy, dx = 1, -1
			} else if downPressed && rightPressed && (!blockDown && !blockRight) {
				dy, dx = 1, 1
			}

		} else { // 上下左右の移動のロジック
			blockUpRight := blockUp || blockRight
			blockUpLeft := blockUp || blockLeft
			blockDownLeft := blockDown || blockLeft
			blockDownRight := blockDown || blockRight

			if upPressed && !downPressed && !blockUp {
				dy = -1
			}
			if downPressed && !upPressed && !blockDown {
				dy = 1
			}
			if leftPressed && !rightPressed && !blockLeft {
				dx = -1
			}
			if rightPressed && !leftPressed && !blockRight {
				dx = 1
			}

			// 斜め移動のロジック
			if upPressed && rightPressed && !blockUpRight {
				dy, dx = -1, 1
			} else if upPressed && leftPressed && !blockUpLeft {
				dy, dx = -1, -1
			} else if downPressed && leftPressed && !blockDownLeft {
				dy, dx = 1, -1
			} else if downPressed && rightPressed && !blockDownRight {
				dy, dx = 1, 1
			}
		}
		g.lastArrowPress = time.Now() // lastArrowPressの更新
	}

	return dx, dy
}

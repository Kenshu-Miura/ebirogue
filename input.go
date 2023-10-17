package main

import (
	_ "image/png" // PNG画像を読み込むために必要
	"time"

	"github.com/hajimehoshi/ebiten/v2"
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
	if aPressed && time.Since(g.lastIncrement) >= 100*time.Millisecond {
		g.IncrementMoveCount()
		g.MoveEnemies()
		g.lastIncrement = time.Now() // lastIncrementの更新
	}

	arrowPressed := upPressed || downPressed || leftPressed || rightPressed

	// 矢印キーの押下ロジック
	if arrowPressed && time.Since(g.lastArrowPress) >= 125*time.Millisecond {

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

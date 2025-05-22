package main

import (
	_ "image/png" // PNG画像を読み込むために必要
	"sort"
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
		nx, ny := playerX+dir.dx, playerY+dir.dy
		if ny < 0 || ny >= len(g.state.Map) || nx < 0 || nx >= len(g.state.Map[0]) {
			continue
		}
		tile := g.state.Map[ny][nx]
		if tile.Type == "door" {
			g.state.Map[ny][nx] = Tile{Type: "corridor"}
			g.isActioned = true
		}
	}
}

func (g *Game) processDKeyPress() {

	if inpututil.IsKeyJustPressed(ebiten.KeyD) && !g.showInventory && !g.isCombatActive && !g.ShowGroundItem && !g.showStairsPrompt {
		g.dPressed = true
		// Find the equipped Arrow item
		var equippedArrow *Arrow
		for _, item := range g.state.Player.Inventory {
			if arrow, ok := item.(*Arrow); ok {
				for i, equippedItem := range g.state.Player.EquippedItems {
					if equippedItem == arrow {
						equippedArrow = arrow

						// If an Arrow item is equipped, decrement its ShotCount
						equippedArrow.ShotCount--

						// Check if ShotCount becomes 0, and if so, set the corresponding slot in EquippedItems to nil
						if equippedArrow.ShotCount == 0 {
							g.state.Player.EquippedItems[i] = nil
						}

						break // Break the inner loop as the equipped arrow is found
					}
				}
				if equippedArrow != nil {
					break // Break the outer loop if equippedArrow is found
				}
			}
		}

		// If an Arrow item is equipped, set its ShotCount to 1 and call g.ThrowItem
		if equippedArrow != nil {
			// Create a new arrow item with ShotCount set to 1
			newArrow := &Arrow{
				BaseItem:    equippedArrow.BaseItem,
				ShotCount:   1,
				AttackPower: equippedArrow.AttackPower,
				Cursed:      equippedArrow.Cursed,
				Identified:  equippedArrow.Identified,
			}
			throwRange := 10
			character := &g.state.Player
			mapState := g.state.Map
			enemies := g.state.Enemies
			onWallHit := func(item Item, position Coordinate, itemIndex int) {
				g.onWallHit(item, position, itemIndex)
			}
			onTargetHit := func(target Character, item Item, index int) {
				g.onTargetHit(target, item, index)
			}
			g.ThrowItem(newArrow, throwRange, character, mapState, enemies, onWallHit, onTargetHit)
		} else {
			action := Action{
				Duration: 0.5, // Assuming a duration of 0.5 seconds for this action
				Message:  "矢が装備されていません",
				Execute: func(*Game) {

				},
			}
			g.dPressed = false
			g.Enqueue(action)
		}
	}
}

func (g *Game) HandleGroundItemInput() {
	sPressed := inpututil.IsKeyJustPressed(ebiten.KeyS)
	if sPressed && !g.showInventory && !g.isCombatActive && !g.ShowGroundItem && !g.showStairsPrompt && !g.ignoreStairs {
		g.ShowGroundItem = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyX) && g.ShowGroundItem {
		g.ShowGroundItem = false
		g.selectedGroundActionIndex = 0
	}

	if g.ShowGroundItem && g.currentGroundItem != nil {
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) && g.selectedGroundActionIndex > 0 {
			g.selectedGroundActionIndex--
		} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) && g.selectedGroundActionIndex < 3 {
			g.selectedGroundActionIndex++
		} else if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			g.GroundItemActioned = true // Toggle the item actions menu
		}
		if g.GroundItemActioned {
			if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
				g.executeGroundItemAction()
			}
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
		g.selectedActionIndex = 0
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
	} else if inpututil.IsKeyJustPressed(ebiten.KeyZ) && len(g.state.Player.Inventory) > 0 {
		if g.selectedGroundActionIndex == 1 && g.showInventory {
			if len(g.state.Player.Inventory) > 0 {
				g.executeItemSwap() // execute your item swapping function here
				g.selectedGroundActionIndex = 0
				g.showInventory = false
			}
		} else if g.useidentifyItem && g.tmpselectedItemIndex != g.selectedItemIndex {
			g.executeItemIdentify()
		} else if !g.useidentifyItem {
			g.showItemActions = true // Toggle the item actions menu
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyX) && g.useidentifyItem {
		g.selectedItemIndex = 0
		g.selectedActionIndex = 0
		g.tmpselectedItemIndex = -1
		g.useidentifyItem = false
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		// Sort the inventory by ID
		sort.Slice(g.state.Player.Inventory, func(i, j int) bool {
			return g.state.Player.Inventory[i].GetID() < g.state.Player.Inventory[j].GetID()
		})

		// Initialize a map to keep track of Arrow items with the same ID
		arrowItemsMap := make(map[int][]*Arrow)

		// Populate the map with Arrow items
		for _, item := range g.state.Player.Inventory {
			if arrow, ok := item.(*Arrow); ok {
				arrowItemsMap[arrow.GetID()] = append(arrowItemsMap[arrow.GetID()], arrow)
			}
		}

		// Iterate through the map and merge Arrow items with the same ID
		for _, arrows := range arrowItemsMap {
			if len(arrows) > 1 { // More than one Arrow item with the same ID
				var totalShotCount int
				var equippedArrow *Arrow
				for _, arrow := range arrows {
					totalShotCount += arrow.ShotCount
					// Check if the arrow is equipped
					for _, equippedItem := range g.state.Player.EquippedItems {
						if equippedItem == arrow {
							equippedArrow = arrow
							break
						}
					}
				}

				// If an arrow is equipped, use it as the base arrow
				mergedArrow := equippedArrow
				if mergedArrow == nil {
					mergedArrow = arrows[0] // Use the first arrow as the base arrow if none are equipped
				}
				mergedArrow.ShotCount = totalShotCount // Update the ShotCount of the merged arrow

				// Remove the other arrows from the inventory
				newInventory := []Item{}
				for _, item := range g.state.Player.Inventory {
					keep := true
					for _, arrow := range arrows {
						if item == arrow && arrow != mergedArrow {
							keep = false
							break
						}
					}
					if keep {
						newInventory = append(newInventory, item)
					}
				}
				g.state.Player.Inventory = newInventory // Update the player's inventory
			}
		}
		return nil
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
	if cPressed && !g.ShowGroundItem && !g.showStairsPrompt && !g.showInventory {
		g.showInventory = true
		return nil // Skip other updates when the inventory window is active
	}

	xPressed := inpututil.IsKeyJustPressed(ebiten.KeyX)

	if xPressed && g.showInventory && !g.showItemActions && !g.useidentifyItem {
		g.selectedItemIndex = 0
		g.selectedActionIndex = 0
		g.selectedGroundActionIndex = 0
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

func (g *Game) CheatHandleInput() (int, int) {
	var dx, dy int

	// キーの押下状態を取得
	upPressed := ebiten.IsKeyPressed(ebiten.KeyUp)
	downPressed := ebiten.IsKeyPressed(ebiten.KeyDown)
	leftPressed := ebiten.IsKeyPressed(ebiten.KeyLeft)
	rightPressed := ebiten.IsKeyPressed(ebiten.KeyRight)
	shiftPressed := ebiten.IsKeyPressed(ebiten.KeyShift) // Shiftキーが押されているかどうかをチェック
	aPressed := ebiten.IsKeyPressed(ebiten.KeyA)         // Aキーが押されているかどうかをチェック

	// 足踏みロジック
	if aPressed && time.Since(g.lastIncrement) >= 100*time.Millisecond &&
		!upPressed && !downPressed && !leftPressed && !rightPressed && !g.isCombatActive {
		g.isActioned = true
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
	var dx, dy = 0, 0

	// キーの押下状態を取得
	upPressed := ebiten.IsKeyPressed(ebiten.KeyUp)
	downPressed := ebiten.IsKeyPressed(ebiten.KeyDown)
	leftPressed := ebiten.IsKeyPressed(ebiten.KeyLeft)
	rightPressed := ebiten.IsKeyPressed(ebiten.KeyRight)
	shiftPressed := ebiten.IsKeyPressed(ebiten.KeyShift) // Shiftキーが押されているかどうかをチェック
	aPressed := ebiten.IsKeyPressed(ebiten.KeyA)         // Aキーが押されているかどうかをチェック
	xPressed := ebiten.IsKeyPressed(ebiten.KeyX)         // Xキーが押されているかどうかをチェック

	if aPressed && !g.zPressed {
		if shiftPressed {
			if upPressed && rightPressed {
				g.state.Player.Direction = UpRight
			} else if upPressed && leftPressed {
				g.state.Player.Direction = UpLeft
			} else if downPressed && leftPressed {
				g.state.Player.Direction = DownLeft
			} else if downPressed && rightPressed {
				g.state.Player.Direction = DownRight
			}
		} else {
			if upPressed && rightPressed {
				g.state.Player.Direction = UpRight
			} else if upPressed && leftPressed {
				g.state.Player.Direction = UpLeft
			} else if downPressed && rightPressed {
				g.state.Player.Direction = DownRight
			} else if downPressed && leftPressed {
				g.state.Player.Direction = DownLeft
			} else if upPressed {
				g.state.Player.Direction = Up
			} else if downPressed {
				g.state.Player.Direction = Down
			} else if leftPressed {
				g.state.Player.Direction = Left
			} else if rightPressed {
				g.state.Player.Direction = Right
			}
		}
		return dx, dy
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyZ) && !aPressed && !xPressed {
		g.zPressed = true
		switch g.state.Player.Direction {
		case Up:
			dx, dy = 0, -1
		case UpRight:
			dx, dy = 1, -1
		case Right:
			dx, dy = 1, 0
		case DownRight:
			dx, dy = 1, 1
		case Down:
			dx, dy = 0, 1
		case DownLeft:
			dx, dy = -1, 1
		case Left:
			dx, dy = -1, 0
		case UpLeft:
			dx, dy = -1, -1
		}
		return dx, dy
	}

	arrowPressed := upPressed || downPressed || leftPressed || rightPressed

	player := g.state.Player
	blockUp := g.state.Map[player.Y-1][player.X].Blocked
	blockDown := g.state.Map[player.Y+1][player.X].Blocked
	blockLeft := g.state.Map[player.Y][player.X-1].Blocked
	blockRight := g.state.Map[player.Y][player.X+1].Blocked
	blockUpRight := blockUp || blockRight
	blockUpLeft := blockUp || blockLeft
	blockDownLeft := blockDown || blockLeft
	blockDownRight := blockDown || blockRight

	if xPressed && !arrowPressed {
		// 足踏みロジック
		if ebiten.IsKeyPressed(ebiten.KeyZ) && time.Since(g.lastIncrement) >= 100*time.Millisecond &&
			!upPressed && !downPressed && !leftPressed && !rightPressed && !g.isCombatActive {
			g.isActioned = true
			g.lastIncrement = time.Now() // lastIncrementの更新
		}
	}

	if arrowPressed && xPressed && !ebiten.IsKeyPressed(ebiten.KeyZ) {
		g.xPressed = true

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

		} else {
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

		for _, room := range g.rooms {
			if isOnBoundary(g.state.Player.X+dx, g.state.Player.Y+dy, room) {
				return 0, 0
			}
		}

		nowX, nowY := g.state.Player.X, g.state.Player.Y
		if (g.state.Player.Direction == Up || g.state.Player.Direction == Down) && (g.state.Map[nowY][nowX+1].Type == "corridor" || g.state.Map[nowY][nowX-1].Type == "corridor") ||
			((g.state.Player.Direction == Left || g.state.Player.Direction == Right) && (g.state.Map[nowY+1][nowX].Type == "corridor" || g.state.Map[nowY-1][nowX].Type == "corridor")) {
			return 0, 0
		}
	}

	// 矢印キーの押下ロジック
	if arrowPressed && time.Since(g.lastArrowPress) >= 180*time.Millisecond {

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

			if upPressed {
				g.state.Player.Direction = Up
			}
			if downPressed {
				g.state.Player.Direction = Down
			}
			if leftPressed {
				g.state.Player.Direction = Left
			}
			if rightPressed {
				g.state.Player.Direction = Right
			}
			if upPressed && rightPressed {
				g.state.Player.Direction = UpRight
			}
			if upPressed && leftPressed {
				g.state.Player.Direction = UpLeft
			}
			if downPressed && leftPressed {
				g.state.Player.Direction = DownLeft
			}
			if downPressed && rightPressed {
				g.state.Player.Direction = DownRight
			}

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

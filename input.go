//go:build !test

package main

import (
	_ "image/png" // PNG画像を読み込むために必要
	"log"
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
		equippedArrow := g.state.Player.EquippedArrow
		if g.shootArrow(equippedArrow, 10) {
			// 矢を撃つことでもターン進行
			g.AdvanceTurn()
		} else {
			g.dPressed = false
			g.EnqueueMessage("矢が装備されていません", 0.5)
		}
	}
}

func (g *Game) HandleGroundItemInput() {
	// Sキーによる直接的な足元チェックは削除（メニューシステムに統合）

	if inpututil.IsKeyJustPressed(ebiten.KeyX) && g.ShowGroundItem {
		g.ShowGroundItem = false
		g.selectedGroundActionIndex = 0
		g.GroundItemActioned = false
		g.showGroundItemDescription = false
		g.groundItemDescriptionText = ""
		g.groundMenuJustOpened = false // フラグもリセット
		// メニューから開いた場合は、メニューに戻る
		if g.returnToMenuFromGround {
			g.showMenu = true
			g.returnToMenuFromGround = false
		}
	}

	// Cキーで全てのメニューを閉じる処理
	if inpututil.IsKeyJustPressed(ebiten.KeyC) && g.ShowGroundItem {
		g.ShowGroundItem = false
		g.selectedGroundActionIndex = 0
		g.GroundItemActioned = false
		g.showGroundItemDescription = false
		g.groundItemDescriptionText = ""
		g.groundMenuJustOpened = false // フラグもリセット
		g.showMenu = false
		g.returnToMenuFromGround = false
	}

	if g.ShowGroundItem && g.currentGroundItem != nil && !g.GroundItemActioned {
		// 足元メニューが開かれたばかりの場合は、1フレーム分Zキーの処理をスキップ
		if g.groundMenuJustOpened {
			g.groundMenuJustOpened = false // フラグをリセット
			// Zキー以外の入力は受け付ける
			if inpututil.IsKeyJustPressed(ebiten.KeyUp) && g.selectedGroundActionIndex > 0 {
				g.selectedGroundActionIndex--
			} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) && g.selectedGroundActionIndex < 4 {
				g.selectedGroundActionIndex++
			}
		} else {
			// 通常の足元メニュー選択状態での操作
			if inpututil.IsKeyJustPressed(ebiten.KeyUp) && g.selectedGroundActionIndex > 0 {
				g.selectedGroundActionIndex--
			} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) && g.selectedGroundActionIndex < 4 {
				g.selectedGroundActionIndex++
			} else if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
				g.GroundItemActioned = true // アクション実行状態に移行
			}
		}
	}

	if g.ShowGroundItem && g.currentGroundItem != nil && g.GroundItemActioned {
		// アクション実行状態での操作
		if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
			g.executeGroundItemAction()
		} else if inpututil.IsKeyJustPressed(ebiten.KeyX) {
			// Xキーで説明ウィンドウを閉じる、または前の状態に戻る
			if g.showGroundItemDescription {
				g.showGroundItemDescription = false
				g.groundItemDescriptionText = ""
			} else {
				g.GroundItemActioned = false
			}
		}
	}
}

func (g *Game) handleItemActionsInput() error {
	maxActionIndex := len(g.currentItemActions()) - 1
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) && g.selectedActionIndex > 0 {
		g.selectedActionIndex--
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) && g.selectedActionIndex < maxActionIndex {
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
	// 絞り込み後の表示リストを取得し、選択位置を表示リスト内へ合わせる
	indices := g.normalizeInventoryView()

	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.selectedItemIndex = moveSelection(indices, g.selectedItemIndex, -1)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.selectedItemIndex = moveSelection(indices, g.selectedItemIndex, 1)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.selectedItemIndex = moveSelection(indices, g.selectedItemIndex, -10)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.selectedItemIndex = moveSelection(indices, g.selectedItemIndex, 10)
	} else if inpututil.IsKeyJustPressed(ebiten.KeyZ) && len(g.state.Player.Inventory) > 0 {
		if g.selectedGroundActionIndex == 1 && g.showInventory {
			if len(g.state.Player.Inventory) > 0 {
				g.executeItemSwap() // execute your item swapping function here
				g.selectedGroundActionIndex = 0
				g.showInventory = false
			}
		} else if g.useIdentifyItem && g.tmpSelectedItemIndex != g.selectedItemIndex {
			g.executeItemIdentify()
		} else if g.usePotInsert && g.tmpSelectedItemIndex != g.selectedItemIndex {
			g.executePotInsertSelection()
		} else if !g.useIdentifyItem && !g.usePotInsert {
			g.showItemActions = true // Toggle the item actions menu
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyX) && (g.useIdentifyItem || g.usePotInsert) {
		g.selectedItemIndex = 0
		g.selectedActionIndex = 0
		g.tmpSelectedItemIndex = -1
		g.useIdentifyItem = false
		g.usePotInsert = false
	}

	// Fキー: カテゴリ絞り込みの切り替え（所持しているカテゴリだけを巡回）
	if inpututil.IsKeyJustPressed(ebiten.KeyF) && !g.useIdentifyItem && !g.usePotInsert {
		g.inventoryFilter = nextFilter(g.inventoryFilter, g.presentCategories())
		if indices := g.normalizeInventoryView(); len(indices) > 0 {
			g.selectedItemIndex = indices[0]
		}
	}

	// Sキー: カテゴリ別ソート（整頓）。同じ矢は1つへ統合される
	if inpututil.IsKeyJustPressed(ebiten.KeyS) && !g.useIdentifyItem && !g.usePotInsert {
		g.sortInventory()
		g.normalizeInventoryView()
	}

	// Nキー: 未識別アイテムへ任意の名前を付ける
	if inpututil.IsKeyJustPressed(ebiten.KeyN) && !g.useIdentifyItem && !g.usePotInsert &&
		g.selectedItemIndex < len(g.state.Player.Inventory) {
		item := g.state.Player.Inventory[g.selectedItemIndex]
		if identifiableItem, ok := item.(Identifiable); ok && !identifiableItem.IsIdentified() {
			g.showNameInput = true
			g.nameInputTargetID = item.GetID()
			g.nameInput = NameInput{Runes: []rune(g.customNames[item.GetID()])}
			g.nameInputCursor = 0
		}
	}

	return nil
}

// 未識別アイテムの名前入力ウィンドウの入力処理
func (g *Game) handleNameInputWindow() error {
	// 五十音グリッドのカーソル移動
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.nameInputCursor = moveGridCursor(g.nameInputCursor, -kanaGridColumns, len(kanaChars))
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.nameInputCursor = moveGridCursor(g.nameInputCursor, kanaGridColumns, len(kanaChars))
	} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.nameInputCursor = moveGridCursor(g.nameInputCursor, -1, len(kanaChars))
	} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.nameInputCursor = moveGridCursor(g.nameInputCursor, 1, len(kanaChars))
	}

	// Zキーで選択中の文字を入力
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		g.nameInput.Append(kanaChars[g.nameInputCursor])
	}

	// Xキーで1文字削除。空の状態ならキャンセルして閉じる
	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		if !g.nameInput.Backspace() {
			g.showNameInput = false
		}
	}

	// Enterキーで決定。空の名前で決定すると設定を消去する
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		name := g.nameInput.String()
		if name == "" {
			delete(g.customNames, g.nameInputTargetID)
		} else {
			g.customNames[g.nameInputTargetID] = name
		}
		g.showNameInput = false
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
	// 名前入力中は他のインベントリ操作を受け付けない
	if g.showNameInput {
		return g.handleNameInputWindow()
	}

	cPressed := inpututil.IsKeyJustPressed(ebiten.KeyC)

	// Cキーが押された場合の処理
	if cPressed {
		// 何かメニューが開いている場合は全て閉じる
		if g.showInventory || g.ShowGroundItem || g.showMenu || g.showEmptyInventoryMessage {
			g.showInventory = false
			g.ShowGroundItem = false
			g.showMenu = false
			g.showEmptyInventoryMessage = false
			g.returnToMenuFromInventory = false
			g.returnToMenuFromGround = false
			g.selectedItemIndex = 0
			g.selectedActionIndex = 0
			g.selectedGroundActionIndex = 0
			// アイテム詳細関連のフラグもリセット
			g.showItemActions = false
			g.showItemDescription = false
			g.useIdentifyItem = false
			g.usePotInsert = false
			g.tmpSelectedItemIndex = -1
			return nil
		}

		// 何もメニューが開いていない場合はメニューを開く（メッセージ履歴・設定・中断確認・ヘルプの表示中は除く）
		if !g.showStairsPrompt && !g.showMessageLog && !g.showSettings && !g.showSuspendPrompt && !g.showHelp {
			// 睡眠状態の場合はメニューを開けない
			if g.state.Player.StatusAilments.Sleep > 0 {
				return nil
			}
			g.showMenu = true
			g.menuSelectedRow = 0 // 初期位置は左上（道具）
			g.menuSelectedCol = 0
			return nil // Skip other updates when the menu window is active
		}
	}

	xPressed := inpututil.IsKeyJustPressed(ebiten.KeyX)

	if xPressed && g.showInventory && !g.showItemActions && !g.useIdentifyItem && !g.usePotInsert {
		g.selectedItemIndex = 0
		g.selectedActionIndex = 0
		g.selectedGroundActionIndex = 0
		g.showInventory = false
		// メニューから開いた場合は、メニューに戻る
		if g.returnToMenuFromInventory {
			g.showMenu = true
			g.returnToMenuFromInventory = false
		}
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

// enemyThreatens は敵が接近している（同じ部屋にいる、または2マス以内にいる）かを判定する
func (g *Game) enemyThreatens() bool {
	px, py := g.state.Player.X, g.state.Player.Y
	visibleEnemies := make([]Enemy, 0, len(g.state.Enemies))
	for i := range g.state.Enemies {
		enemy := g.state.Enemies[i]
		if isEnemyDisguised(enemy) {
			continue
		}
		visibleEnemies = append(visibleEnemies, enemy)
	}
	if enemyWithinDistance(px, py, 2, visibleEnemies) {
		return true
	}
	for i := range visibleEnemies {
		enemy := &visibleEnemies[i]
		if isSameRoom(px, py, enemy.X, enemy.Y, g.rooms) {
			return true
		}
	}
	return false
}

// dashDangerReason はダッシュで (dx, dy) へ進む際の危険条件を返す。危険がなければ空文字を返す。
func (g *Game) dashDangerReason(dx, dy int) string {
	// 敵の接近
	if g.enemyThreatens() {
		return "enemy"
	}

	nextX, nextY := g.state.Player.X+dx, g.state.Player.Y+dy

	// 進行方向のアイテム
	for _, item := range g.state.Items {
		itemX, itemY := item.GetPosition()
		if itemX == nextX && itemY == nextY {
			return "item"
		}
	}

	// 進行方向の発見済みの罠
	for _, trap := range g.state.MapTraps {
		if trap.Discovered && trap.X == nextX && trap.Y == nextY {
			return "trap"
		}
	}

	return ""
}

func (g *Game) HandleInput() (int, int) {
	var dx, dy = 0, 0

	// 睡眠状態の場合はキー入力を受け付けない
	if g.state.Player.StatusAilments.Sleep > 0 {
		return 0, 0
	}

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
			// 敵接近時は押しっぱなしでの連続足踏みを自動停止する（Zの押し直しで1回ずつは可能）
			if !g.enemyThreatens() || inpututil.IsKeyJustPressed(ebiten.KeyZ) {
				g.isActioned = true
				g.lastIncrement = time.Now() // lastIncrementの更新
			}
		}
	}

	if arrowPressed && xPressed && !ebiten.IsKeyPressed(ebiten.KeyZ) {
		g.xPressed = true

		if g.dashStopped {
			if time.Since(g.lastDashStop) < 400*time.Millisecond {
				return 0, 0
			}
			g.dashStopped = false // ダッシュ再開
		} else {
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

			// 敵の接近・アイテム・発見済みの罠などの危険条件で自動停止する
			if dx != 0 || dy != 0 {
				if reason := g.dashDangerReason(dx, dy); reason != "" {
					log.Printf("Dash stopped (%s): (%d,%d)", reason, g.state.Player.X+dx, g.state.Player.Y+dy)
					g.dashStopped = true
					g.lastDashStop = time.Now()
					return 0, 0
				}
			}

			for _, room := range g.rooms {
				if isOnBoundary(g.state.Player.X+dx, g.state.Player.Y+dy, room) {
					log.Printf("Dash stopped at entrance: (%d,%d)", g.state.Player.X+dx, g.state.Player.Y+dy)
					g.dashStopped = true
					g.lastDashStop = time.Now()
					return 0, 0
				}
			}

			// 通路の分岐・曲がり角で自動停止する
			nowX, nowY := g.state.Player.X, g.state.Player.Y
			if corridorBranch(g.state.Map, nowX, nowY, dx, dy) {
				log.Printf("Dash stopped at corner: (%d,%d)", nowX, nowY)
				g.dashStopped = true
				g.lastDashStop = time.Now()
				return 0, 0
			}
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

// デバッグ用F1キー処理：混乱薬をインベントリに追加
func (g *Game) processF1KeyPress() {
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		// インベントリに空きがあるかチェック
		if len(g.state.Player.Inventory) < g.state.Player.MaxInventory {
			addItem := g.createItemByID(19, 0, 0) // 座標は仮の値
			g.state.Player.Inventory = append(g.state.Player.Inventory, addItem)

			// アイテム名を動的に取得
			itemName := addItem.GetName()
			action := Action{
				Duration: 0.4,
				Message:  "[DEBUG] " + itemName + "をインベントリに追加した",
				Execute:  func(g *Game) {},
			}
			g.Enqueue(action)
		} else {
			action := Action{
				Duration: 0.4,
				Message:  "[DEBUG] インベントリが満杯です",
				Execute:  func(g *Game) {},
			}
			g.Enqueue(action)
		}
	}
}

// メニューシステムの入力処理
func (g *Game) handleMenuInput() error {
	if !g.showMenu {
		return nil
	}

	// 矢印キーでメニュー項目を移動（3行目はヘルプのみの1項目）
	const menuRowCount = 3
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.menuSelectedRow = (g.menuSelectedRow - 1 + menuRowCount) % menuRowCount
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.menuSelectedRow = (g.menuSelectedRow + 1) % menuRowCount
	} else if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		g.menuSelectedCol = (g.menuSelectedCol - 1 + 2) % 2
	} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.menuSelectedCol = (g.menuSelectedCol + 1) % 2
	}

	// Zキーで決定
	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		if g.menuSelectedRow == 0 && g.menuSelectedCol == 0 {
			// 道具（インベントリ）
			g.showMenu = false
			g.returnToMenuFromInventory = true
			// アイテムが空の場合は「何も持っていない」メッセージを表示
			if len(g.state.Player.Inventory) == 0 {
				g.showEmptyInventoryMessage = true
			} else {
				g.showInventory = true
			}
		} else if g.menuSelectedRow == 0 && g.menuSelectedCol == 1 {
			// 足元
			g.showMenu = false
			g.returnToMenuFromGround = true
			// 足元のアイテムを確認
			playerX, playerY := g.state.Player.X, g.state.Player.Y
			foundItem := false
			for _, item := range g.state.Items {
				itemX, itemY := item.GetPosition()
				if itemX == playerX && itemY == playerY {
					g.currentGroundItem = item
					foundItem = true
					break
				}
			}
			if !foundItem {
				g.currentGroundItem = nil
			}
			g.ShowGroundItem = true
			g.GroundItemActioned = false        // メニュー表示時は初期化
			g.selectedGroundActionIndex = 0     // 選択インデックスをリセット
			g.showGroundItemDescription = false // 説明ウィンドウも初期化
			g.groundItemDescriptionText = ""    // 説明テキストもクリア
			g.groundMenuJustOpened = true       // 足元メニューが開かれたばかりのフラグを設定
		} else if g.menuSelectedRow == 1 && g.menuSelectedCol == 0 {
			// 設定
			g.showMenu = false
			g.showSettings = true
			g.settingsSelectedIndex = 0
		} else if g.menuSelectedRow == 1 && g.menuSelectedCol == 1 {
			// 中断
			if len(g.ActionQueue.Queue) > 0 {
				// 行動処理中は状態が確定していないため中断できない
				g.showMenu = false
				g.Enqueue(Action{
					Duration: 0.8,
					Message:  "行動中は中断できない",
					Execute:  func(*Game) {},
				})
			} else {
				g.showMenu = false
				g.showSuspendPrompt = true
				g.suspendSelectedOption = 0
				g.suspendPromptJustOpened = true
			}
		} else if g.menuSelectedRow == 2 {
			// ヘルプ（3行目は列に関係なくヘルプを開く）
			g.showMenu = false
			g.showHelp = true
			g.returnToMenuFromHelp = true
			g.helpPage = 0
			g.helpScroll = 0
		}
	}

	// Xキーでメニューを閉じる
	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		g.showMenu = false
	}

	return nil
}

// 設定ウィンドウの入力処理
func (g *Game) handleSettingsInput() error {
	if !g.showSettings {
		return nil
	}

	const settingsItemCount = 2

	// ↑↓で項目を移動
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.settingsSelectedIndex = (g.settingsSelectedIndex - 1 + settingsItemCount) % settingsItemCount
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.settingsSelectedIndex = (g.settingsSelectedIndex + 1) % settingsItemCount
	}

	// ←→で設定値を切り替え（即時反映）
	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) || inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		switch g.settingsSelectedIndex {
		case 0:
			g.settings.Fullscreen = !g.settings.Fullscreen
			ebiten.SetFullscreen(g.settings.Fullscreen)
		case 1:
			g.settings.ShowMiniMap = !g.settings.ShowMiniMap
		}
	}

	// Xキーで設定を保存してメニューに戻る
	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		g.showSettings = false
		saveSettings(g.settings)
		g.showMenu = true
	}

	// Cキーで設定を保存して全て閉じる
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.showSettings = false
		saveSettings(g.settings)
	}

	return nil
}

// 中断確認ダイアログの入力処理
func (g *Game) handleSuspendPromptInput() error {
	if !g.showSuspendPrompt {
		return nil
	}

	// メニューのZキー決定と同じフレームで確定しないよう1フレームスキップ
	if g.suspendPromptJustOpened {
		g.suspendPromptJustOpened = false
		return nil
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) || inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		g.suspendSelectedOption = (g.suspendSelectedOption + 1) % 2
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		if g.suspendSelectedOption == 0 { // 中断する
			g.showSuspendPrompt = false
			if err := g.saveSuspendData(); err != nil {
				log.Printf("中断セーブに失敗: %v", err)
				g.Enqueue(Action{
					Duration: 0.8,
					Message:  "中断データの保存に失敗した",
					Execute:  func(*Game) {},
				})
			} else {
				g.suspendRequested = true
			}
		} else { // やめる
			g.showSuspendPrompt = false
			g.showMenu = true
		}
		g.suspendSelectedOption = 0
	}

	// Xキーでメニューに戻る
	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		g.showSuspendPrompt = false
		g.suspendSelectedOption = 0
		g.showMenu = true
	}

	// Cキーで全て閉じる
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.showSuspendPrompt = false
		g.suspendSelectedOption = 0
	}

	return nil
}

// メッセージ履歴ウィンドウの入力処理
func (g *Game) handleMessageLogInput() error {
	if g.showMessageLog {
		// ↑↓でスクロール（押しっぱなしでリピート）
		if logScrollKeyPressed(ebiten.KeyUp) {
			if g.messageLogScroll < g.messageLog.MaxScroll(messageLogPageSize) {
				g.messageLogScroll++
			}
		} else if logScrollKeyPressed(ebiten.KeyDown) {
			if g.messageLogScroll > 0 {
				g.messageLogScroll--
			}
		}

		// XキーまたはLキーで閉じる
		if inpututil.IsKeyJustPressed(ebiten.KeyX) || inpututil.IsKeyJustPressed(ebiten.KeyL) {
			g.showMessageLog = false
			g.messageLogScroll = 0
		}
		return nil
	}

	// 他のウィンドウが開いていない時だけLキーで履歴を開く
	if inpututil.IsKeyJustPressed(ebiten.KeyL) &&
		!g.showInventory && !g.ShowGroundItem && !g.showMenu &&
		!g.showStairsPrompt && !g.showEmptyInventoryMessage &&
		!g.showSettings && !g.showSuspendPrompt && !g.showHelp &&
		g.state.Player.StatusAilments.Sleep == 0 {
		g.showMessageLog = true
		g.messageLogScroll = 0
	}

	return nil
}

// ヘルプウィンドウの入力処理
func (g *Game) handleHelpInput() error {
	if g.showHelp {
		pages := helpPages()

		// ←→でページを切り替え
		if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
			g.helpPage = (g.helpPage - 1 + len(pages)) % len(pages)
			g.helpScroll = 0
		} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
			g.helpPage = (g.helpPage + 1) % len(pages)
			g.helpScroll = 0
		}

		// ↑↓でスクロール（押しっぱなしでリピート）
		if logScrollKeyPressed(ebiten.KeyDown) {
			if g.helpScroll < helpMaxScroll(len(pages[g.helpPage].Lines), helpPageSize) {
				g.helpScroll++
			}
		} else if logScrollKeyPressed(ebiten.KeyUp) {
			if g.helpScroll > 0 {
				g.helpScroll--
			}
		}

		// XキーまたはHキーで閉じる（メニューから開いた場合はメニューへ戻る）
		if inpututil.IsKeyJustPressed(ebiten.KeyX) || inpututil.IsKeyJustPressed(ebiten.KeyH) {
			g.showHelp = false
			g.helpScroll = 0
			if g.returnToMenuFromHelp {
				g.showMenu = true
				g.returnToMenuFromHelp = false
			}
		}

		// Cキーで全て閉じる
		if inpututil.IsKeyJustPressed(ebiten.KeyC) {
			g.showHelp = false
			g.showMenu = false
			g.helpScroll = 0
			g.returnToMenuFromHelp = false
		}
		return nil
	}

	// 他のウィンドウが開いていない時だけHキーでヘルプを開く
	if inpututil.IsKeyJustPressed(ebiten.KeyH) &&
		!g.showInventory && !g.ShowGroundItem && !g.showMenu &&
		!g.showStairsPrompt && !g.showEmptyInventoryMessage &&
		!g.showSettings && !g.showSuspendPrompt && !g.showMessageLog &&
		g.state.Player.StatusAilments.Sleep == 0 {
		g.showHelp = true
		g.helpPage = 0
		g.helpScroll = 0
	}

	return nil
}

// スクロールキーの押下判定（押した瞬間と長押しリピート）
func logScrollKeyPressed(key ebiten.Key) bool {
	d := inpututil.KeyPressDuration(key)
	return d == 1 || (d >= 20 && d%5 == 0)
}

// 空のインベントリメッセージの入力処理
func (g *Game) handleEmptyInventoryMessage() error {
	if !g.showEmptyInventoryMessage {
		return nil
	}

	// Xキーでメニューに戻る
	if inpututil.IsKeyJustPressed(ebiten.KeyX) {
		g.showEmptyInventoryMessage = false
		g.showMenu = true
		g.returnToMenuFromInventory = false
	}

	// Cキーで全メニューを閉じてゲームに戻る
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.showEmptyInventoryMessage = false
		g.showMenu = false
		g.returnToMenuFromInventory = false
	}

	return nil
}

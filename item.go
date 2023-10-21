package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
)

func (g *Game) executeAction() {

	if g.selectedActionIndex == 2 { // Assuming index 2 corresponds to '置く'
		selectedItem := g.state.Player.Inventory[g.selectedItemIndex]
		// Remove the item from inventory
		g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
		// Add the item to the world at the player's current position
		selectedItem.SetPosition(g.state.Player.X, g.state.Player.Y)
		newItem := selectedItem
		g.state.Items = append(g.state.Items, newItem)
		// Set action message
		g.descriptionQueue = append(g.descriptionQueue, fmt.Sprintf("%sを置いた", selectedItem.GetName()))
		g.showItemActions = false
		g.showInventory = false
		g.IncrementMoveCount()
		g.MoveEnemies()
	}

	if g.selectedActionIndex == 3 { // Assuming 0-based index and "説明" is at index 3
		selectedItem := g.state.Player.Inventory[g.selectedItemIndex]
		g.itemdescriptionText = selectedItem.GetDescription()
		g.showItemDescription = true
	}

}

func (g *Game) PickupItem() {
	playerX, playerY := g.state.Player.X, g.state.Player.Y // プレイヤーの座標を取得

	// プレイヤーのインベントリサイズをチェック
	if len(g.state.Player.Inventory) >= 20 {
		return // インベントリが満杯の場合は、何もせずに関数を終了
	}

	for i, item := range g.state.Items { // GameStateの全てのアイテムに対してループ
		itemX, itemY := item.GetPosition()        // アイテムの座標を取得
		if itemX == playerX && itemY == playerY { // アイテムの座標とプレイヤーの座標が一致するかチェック
			g.state.Player.Inventory = append(g.state.Player.Inventory, item) // アイテムをプレイヤーのインベントリに追加

			g.descriptionQueue = append(g.descriptionQueue, fmt.Sprintf("%sを拾った", g.state.Items[i].GetName()))

			// アイテムをGameState.Itemsから削除
			g.state.Items = append(g.state.Items[:i], g.state.Items[i+1:]...)

			break // 一致するアイテムが見つかったらループを終了
		}
	}
}

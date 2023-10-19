package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
	"time"
)

func (g *Game) executeAction() {

	if g.selectedActionIndex == 2 { // Assuming index 2 corresponds to '捨てる'
		selectedItem := g.state.Player.Inventory[g.selectedItemIndex]
		// Remove the item from inventory
		g.state.Player.Inventory = append(g.state.Player.Inventory[:g.selectedItemIndex], g.state.Player.Inventory[g.selectedItemIndex+1:]...)
		// Set action message
		g.descriptionText = fmt.Sprintf("%sを捨てた", selectedItem.Name)
		g.showDescription = true
		g.descriptionTimeout = time.Now().Add(2 * time.Second) // Set timer for 2 seconds
	}

	if g.selectedActionIndex == 3 { // Assuming 0-based index and "説明" is at index 3
		selectedItem := g.state.Player.Inventory[g.selectedItemIndex]
		g.descriptionText = selectedItem.Description
		g.showDescription = true
		g.descriptionTimeout = time.Now().Add(2 * time.Second) // Set a timeout for 2 seconds
	}

}

func (g *Game) PickupItem() {
	playerX, playerY := g.state.Player.X, g.state.Player.Y // プレイヤーの座標を取得

	// プレイヤーのインベントリサイズをチェック
	if len(g.state.Player.Inventory) >= 20 {
		return // インベントリが満杯の場合は、何もせずに関数を終了
	}

	for i, item := range g.state.Items { // GameStateの全てのアイテムに対してループ
		if item.X == playerX && item.Y == playerY { // アイテムの座標とプレイヤーの座標が一致するかチェック
			g.state.Player.Inventory = append(g.state.Player.Inventory, item) // アイテムをプレイヤーのインベントリに追加

			g.descriptionText = fmt.Sprintf("%sを拾った", g.state.Items[i].Name)
			g.showDescription = true
			g.descriptionTimeout = time.Now().Add(2 * time.Second) // Set timer for 2 seconds

			// アイテムをGameState.Itemsから削除
			g.state.Items = append(g.state.Items[:i], g.state.Items[i+1:]...)

			break // 一致するアイテムが見つかったらループを終了
		}
	}
}

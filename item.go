package main

import (
	_ "image/png" // PNG画像を読み込むために必要
	"time"
)

func (g *Game) executeAction() {

	if g.selectedActionIndex == 3 { // Assuming 0-based index and "説明" is at index 3
		selectedItem := g.state.Player.Inventory[g.selectedItemIndex]
		g.descriptionText = selectedItem.Description
		g.showDescription = true
		g.descriptionTimeout = time.Now().Add(2 * time.Second) // Set a timeout for 2 seconds
	}

}

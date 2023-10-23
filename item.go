package main

import (
	"fmt"
	_ "image/png" // PNG画像を読み込むために必要
)

func (g *Game) PickUpItem(item Item, i int) {
	g.state.Player.Inventory = append(g.state.Player.Inventory, item) // アイテムをプレイヤーのインベントリに追加
	// アイテムをGameState.Itemsから削除
	g.state.Items = append(g.state.Items[:i], g.state.Items[i+1:]...)
}

func (g *Game) PickupItem() {
	playerX, playerY := g.state.Player.X, g.state.Player.Y // プレイヤーの座標を取得

	if !g.xPressed {
		for i, item := range g.state.Items { // GameStateの全てのアイテムに対してループ
			itemX, itemY := item.GetPosition()        // アイテムの座標を取得
			if itemX == playerX && itemY == playerY { // アイテムの座標とプレイヤーの座標が一致するかチェック
				// プレイヤーのインベントリサイズをチェック
				if len(g.state.Player.Inventory) < 20 {
					action := Action{
						Duration: 0.2,
						Message:  fmt.Sprintf("%sを拾った", g.state.Items[i].GetName()),
						Execute: func(g *Game) {
							g.PickUpItem(item, i)
						},
					}

					g.Enqueue(action)
					break // 一致するアイテムが見つかったらループを終了
				} else {
					action := Action{
						Duration: 0.5,
						Message:  fmt.Sprintf("持ち物がいっぱいで%sを拾えなかった", g.state.Items[i].GetName()),
						Execute: func(g *Game) {

						},
					}
					g.Enqueue(action)
				}
			}
		}
	} else {
		for i, item := range g.state.Items { // GameStateの全てのアイテムに対してループ
			itemX, itemY := item.GetPosition()        // アイテムの座標を取得
			if itemX == playerX && itemY == playerY { // アイテムの座標とプレイヤーの座標が一致するかチェック
				action := Action{
					Duration: 0.4,
					Message:  fmt.Sprintf("%sに乗った", g.state.Items[i].GetName()),
					Execute: func(g *Game) {
					},
				}
				g.Enqueue(action)
				break // 一致するアイテムが見つかったらループを終了
			}
		}

	}
}

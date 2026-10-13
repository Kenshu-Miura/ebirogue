//go:build debug && !test

package main

// デバッグビルド（go run -tags debug .）でだけ有効になる開発者向けの処理。
// 通常のビルドでは debug_off.go の空の実装が使われる。

// デバッグ用F1キー処理：混乱薬をインベントリに追加
func (g *Game) processF1KeyPress() {
	if g.keys().JustPressed(KeyF1) {
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

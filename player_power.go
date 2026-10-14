//go:build !test

package main

import "fmt"

// drainPlayerPower はパワーを amount 下げ、結果のメッセージを積む。
// 毒ヘビ系の毒攻撃・毒矢の罠・腐った食料の共通処理で、Action の Execute 内から呼ぶ。
// 実際に下がった量を返す（パワーが0なら下がらず0）。
// 装備能力で毒を無効化できる（StatusPoison）ときは下げずに0を返す。
func (g *Game) drainPlayerPower(amount int) int {
	if message, immune := g.playerImmunityMessage(StatusPoison); immune {
		g.EnqueueMessage(message, 0.4)
		return 0
	}
	player := &g.state.Player
	var drained int
	player.Power, drained = drainedPower(player.Power, amount)
	if drained > 0 {
		g.EnqueueMessage(fmt.Sprintf("海老さんのパワーが%d下がった", drained), 0.4)
	} else {
		g.EnqueueMessage("しかし海老さんのパワーはこれ以上下がらない", 0.4)
	}
	return drained
}

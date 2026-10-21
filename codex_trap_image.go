//go:build !test

package main

import "github.com/hajimehoshi/ebiten/v2"

func (g *Game) codexTrapImage(name string) *ebiten.Image {
	var img *ebiten.Image
	switch name {
	case "睡眠ガスの罠":
		img = g.sleepTrapImg
	case "毒矢の罠":
		img = g.poisonArrowTrapImg
	case "鈍足の罠":
		img = g.slowTrapImg
	case "地雷":
		img = g.mineTrapImg
	case "ワープの罠":
		img = g.warpTrapImg
	case "落とし穴":
		img = g.pitfallTrapImg
	case "トラバサミ":
		img = g.bearTrapImg
	case "木の矢の罠":
		img = g.woodArrowTrapImg
	case "鉄の矢の罠":
		img = g.ironArrowTrapImg
	case "落石の罠":
		img = g.fallingRockTrapImg
	case "大型地雷":
		img = g.largeMineTrapImg
	case "装備はずしの罠":
		img = g.unequipTrapImg
	case "腐敗の罠":
		img = g.spoilTrapImg
	case "回復封じの罠":
		img = g.noHealingTrapImg
	case "散乱の罠":
		img = g.scatterTrapImg
	case "サビの罠":
		img = g.rustTrapImg
	default:
		img = g.sleepTrapImg // デフォルトで睡眠ガスの罠の画像を使用
	}
	return img
}

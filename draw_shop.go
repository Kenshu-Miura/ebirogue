//go:build !test

package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

// drawShopPrompt は店の支払い・売却の窓（店主の言葉と、払う・売る／やめる）を描く（wave10-B）。
func (g *Game) drawShopPrompt(screen *ebiten.Image) {
	prompt := g.shopPrompt
	if prompt == nil || g.fadingOut || g.fadingIn {
		return
	}
	windowX, windowY, windowWidth, windowHeight := 60, 90, 520, 80
	drawWindowWithBorder(screen, windowX, windowY, windowWidth, windowHeight, 255)
	text.Draw(screen, prompt.Text, mplusNormalFont, windowX+20, windowY+28, color.White)
	for i, option := range prompt.options() {
		text.Draw(screen, option, mplusNormalFont, windowX+50+i*120, windowY+60, color.White)
	}
	text.Draw(screen, "→", mplusNormalFont, windowX+30+prompt.Selected*120, windowY+60, color.White)
}

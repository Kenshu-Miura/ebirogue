//go:build !test

package main

import "github.com/hajimehoshi/ebiten/v2"

func (g *Game) specialPlayerImage() *ebiten.Image {
	if !statusActive(g.state.Player.StatusAilments, StatusNigiri) {
		return g.playerImg
	}
	if g.nigiriPlayerImg == nil {
		g.nigiriPlayerImg = loadImage("img/player_nigiri.png")
	}
	return g.nigiriPlayerImg
}

func (g *Game) specialDecoyImage() *ebiten.Image {
	if g.decoyImg == nil {
		g.decoyImg = loadImage("img/enemy_decoy.png")
	}
	return g.decoyImg
}

//go:build !game

package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

func (g *Game) drawOverlay(screen *ebiten.Image)                          {}
func (g *Game) DrawStairsPrompt(screen *ebiten.Image)                     {}
func (g *Game) UpdateAndDrawMiniMap(screen *ebiten.Image)                 {}
func (g *Game) CalculateAnimationOffset(screen *ebiten.Image) (int, int)  { return 0, 0 }
func (g *Game) UpdateEnemyAnimation(enemy *Enemy)                         {}
func (g *Game) CalculateEnemyOffset(enemy *Enemy) (int, int)              { return 0, 0 }
func (g *Game) ManageDescriptions()                                       {}
func (g *Game) DrawDescriptions(screen *ebiten.Image)                     {}
func (g *Game) drawItemDescription(screen *ebiten.Image)                  {}
func (g *Game) DrawGroundItem(screen *ebiten.Image)                       {}
func (g *Game) drawActionMenu(screen *ebiten.Image)                       {}
func (g *Game) drawUseIdentifyItemWindow(screen *ebiten.Image)            {}
func (g *Game) drawInventoryWindow(_ *ebiten.Image) error                 { return nil }
func (g *Game) DrawMap(screen *ebiten.Image, offsetX, offsetY int)        {}
func (g *Game) DrawPlayer(screen *ebiten.Image, centerX, centerY int)     {}
func (g *Game) getItemImage(_ Item) *ebiten.Image                         { return nil }
func (g *Game) DrawThrownItem(screen *ebiten.Image, offsetX, offsetY int) {}
func (g *Game) DrawItems(screen *ebiten.Image, offsetX, offsetY int)      {}
func (g *Game) DrawEnemies(screen *ebiten.Image, offsetX, offsetY int)    {}
func (g *Game) DrawHUD(screen *ebiten.Image)                              {}

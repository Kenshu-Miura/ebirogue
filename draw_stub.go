//go:build !game

package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

func (g *Game) drawOverlay(screen *ebiten.Image)                         {}
func (g *Game) DrawStairsPrompt(screen *ebiten.Image)                    {}
func (g *Game) UpdateAndDrawMiniMap(screen *ebiten.Image)                {}
func (g *Game) updateMiniMap(screen *ebiten.Image)                       {}
func (g *Game) CalculateAnimationOffset(screen *ebiten.Image) (int, int) { return 0, 0 }
func (g *Game) UpdateEnemyAnimation(enemy *Enemy)                        {}
func (g *Game) CalculateEnemyOffset(enemy *Enemy) (int, int)             { return 0, 0 }
func (g *Game) ManageDescriptions()                                      {}
func (g *Game) DrawDescriptions(screen *ebiten.Image)                    {}
func drawWindowWithBorder(screen *ebiten.Image, windowX, windowY, windowWidth, windowHeight int, alpha uint8) {
}
func (g *Game) drawItemDescription(screen *ebiten.Image)                  {}
func (g *Game) DrawGroundItem(screen *ebiten.Image)                       {}
func (g *Game) drawActionMenu(screen *ebiten.Image)                       {}
func (g *Game) drawUseIdentifyItemWindow(screen *ebiten.Image)            {}
func (g *Game) drawInventoryWindow(screen *ebiten.Image) error            { return nil }
func (g *Game) DrawMap(screen *ebiten.Image, offsetX, offsetY int)        {}
func (g *Game) DrawPlayer(screen *ebiten.Image, centerX, centerY int)     {}
func (g *Game) getItemImage(item Item) *ebiten.Image                      { return nil }
func (g *Game) DrawThrownItem(screen *ebiten.Image, offsetX, offsetY int) {}
func (g *Game) DrawItems(screen *ebiten.Image, offsetX, offsetY int)      {}
func (g *Game) getEnemyImage(enemy Enemy) *ebiten.Image                   { return nil }
func (g *Game) DrawEnemies(screen *ebiten.Image, offsetX, offsetY int)    {}
func (g *Game) DrawHUD(screen *ebiten.Image)                              {}
func drawBarWithBorder(screen *ebiten.Image, x, y, width, height int, barColor, borderColor color.Color) {
}

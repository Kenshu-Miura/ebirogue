//go:build test

package main

type Image struct{}

func (g *Game) drawOverlay(screen *Image)                          {}
func (g *Game) DrawStairsPrompt(screen *Image)                     {}
func (g *Game) UpdateAndDrawMiniMap(screen *Image)                 {}
func (g *Game) CalculateAnimationOffset(screen *Image) (int, int)  { return 0, 0 }
func (g *Game) UpdateEnemyAnimation(enemy *Enemy)                  {}
func (g *Game) CalculateEnemyOffset(enemy *Enemy) (int, int)       { return 0, 0 }
func (g *Game) ManageDescriptions()                                {}
func (g *Game) DrawDescriptions(screen *Image)                     {}
func (g *Game) drawItemDescription(screen *Image)                  {}
func (g *Game) DrawGroundItem(screen *Image)                       {}
func (g *Game) drawActionMenu(screen *Image)                       {}
func (g *Game) drawUseIdentifyItemWindow(screen *Image)            {}
func (g *Game) drawInventoryWindow(_ *Image) error                 { return nil }
func (g *Game) DrawMap(screen *Image, offsetX, offsetY int)        {}
func (g *Game) DrawPlayer(screen *Image, centerX, centerY int)     {}
func (g *Game) getItemImage(_ Item) *Image                         { return nil }
func (g *Game) DrawThrownItem(screen *Image, offsetX, offsetY int) {}
func (g *Game) DrawItems(screen *Image, offsetX, offsetY int)      {}
func (g *Game) DrawEnemies(screen *Image, offsetX, offsetY int)    {}
func (g *Game) DrawHUD(screen *Image)                              {}

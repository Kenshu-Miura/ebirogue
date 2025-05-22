package ebitenutil

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image"
)

func NewImageFromFile(path string) (*ebiten.Image, image.Image, error) {
	return ebiten.NewImage(0, 0), nil, nil
}

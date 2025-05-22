package ebiten

import (
	"image"
	"image/color"
)

type Image struct{}

type DrawImageOptions struct {
	GeoM       GeoM
	ColorScale ColorScale
}

type GeoM struct{}

func (g *GeoM) Translate(x, y float64) {}
func (g *GeoM) Rotate(theta float64)   {}
func (g *GeoM) Reset()                 {}

type ColorScale struct{}

func (c *ColorScale) Scale(r, g, b, a float32) {}

func NewImage(w, h int) *Image                                { return &Image{} }
func (i *Image) Fill(col interface{})                         {}
func (i *Image) DrawImage(src *Image, opts *DrawImageOptions) {}
func (i *Image) Bounds() image.Rectangle                      { return image.Rect(0, 0, 0, 0) }
func (i *Image) SubImage(r image.Rectangle) image.Image       { return i }
func (i *Image) ColorModel() color.Model                      { return color.RGBAModel }
func (i *Image) At(x, y int) color.Color                      { return color.RGBA{} }

func RunGame(g interface{}) error { return nil }
func SetWindowSize(w, h int)      {}
func SetWindowTitle(title string) {}

type Key int

const (
	KeyA Key = iota
	KeyC
	KeyD
	KeyDown
	KeyLeft
	KeyRight
	KeyS
	KeyShift
	KeySpace
	KeyUp
	KeyX
	KeyZ
)

func IsKeyPressed(k Key) bool { return false }

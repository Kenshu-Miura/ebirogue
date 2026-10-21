//go:build ignore

// 地形の仮画像。正式画像へ差し替えるまでの30px表示用。
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	for _, name := range []string{"water", "cracked_wall"} {
		img := image.NewNRGBA(image.Rect(0, 0, 30, 30))
		for y := 1; y < 29; y++ {
			for x := 1; x < 29; x++ {
				c := color.NRGBA{88, 82, 90, 255}
				if name == "water" {
					c = color.NRGBA{22, 88, 155, 255}
					if (y+x/6)%8 < 2 {
						c = color.NRGBA{94, 195, 225, 255}
					}
				} else {
					if y%9 == 0 || (x+(y/9)%2*12)%15 == 0 {
						c = color.NRGBA{48, 44, 52, 255}
					}
					if x == 15+(y/4)%3 || (y > 14 && x == 15-(y-14)/2) {
						c = color.NRGBA{224, 191, 132, 255}
					}
				}
				img.SetNRGBA(x, y, c)
			}
		}
		f, err := os.Create("img/" + name + ".png")
		if err != nil {
			panic(err)
		}
		if err = png.Encode(f, img); err != nil {
			panic(err)
		}
		f.Close()
	}
}

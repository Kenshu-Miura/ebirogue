//go:build ignore

// ダメージ罠の仮画像を再生成する: go run tools/damage_trap_images.go
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	names := []string{"wood_arrow_trap", "iron_arrow_trap", "falling_rock_trap", "large_mine_trap"}
	colors := []color.NRGBA{{180, 110, 55, 255}, {155, 200, 220, 255}, {160, 160, 180, 255}, {235, 85, 55, 255}}
	for i, name := range names {
		im := image.NewNRGBA(image.Rect(0, 0, 30, 30))
		for y := 5; y < 26; y++ {
			for x := 4; x < 26; x++ {
				edge := x == 4 || x == 25 || y == 5 || y == 25
				if edge {
					im.SetNRGBA(x, y, color.NRGBA{235, 220, 170, 255})
				} else {
					im.SetNRGBA(x, y, color.NRGBA{45, 50, 60, 255})
				}
			}
		}
		for y := 8; y < 23; y++ {
			for x := 7; x < 23; x++ {
				draw := false
				switch i {
				case 0, 1:
					draw = (y >= 14 && y <= 16) || (x >= 17 && x <= 22 && y >= 9 && y <= 21 && abs(y-15) <= 22-x)
				case 2:
					draw = (x-15)*(x-15)+(y-15)*(y-15) <= 48
				case 3:
					draw = abs(x-15) <= 2 || abs(y-15) <= 2 || abs(x-15) == abs(y-15)
				}
				if draw {
					im.SetNRGBA(x, y, colors[i])
				}
			}
		}
		f, err := os.Create("img/" + name + ".png")
		if err != nil {
			panic(err)
		}
		if err = png.Encode(f, im); err != nil {
			panic(err)
		}
		f.Close()
	}
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

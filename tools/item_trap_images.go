//go:build ignore

// 正式な画像に差し替え済み。再実行しない（wave7-D）。

// 装備・所持品の罠の仮画像: go run tools/item_trap_images.go
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "正式な画像に差し替え済み。再実行しない（wave7-D）。")
	os.Exit(1)
	for i, name := range []string{"unequip_trap", "spoil_trap", "scatter_trap"} {
		im := image.NewNRGBA(image.Rect(0, 0, 30, 30))
		colors := []color.NRGBA{{245, 195, 85, 255}, {145, 205, 85, 255}, {115, 200, 245, 255}}
		for y := 5; y < 26; y++ {
			for x := 4; x < 26; x++ {
				c := color.NRGBA{40, 45, 55, 255}
				if x == 4 || x == 25 || y == 5 || y == 25 {
					c = color.NRGBA{235, 225, 180, 255}
				}
				draw := false
				switch i {
				case 0:
					draw = (x >= 13 && x <= 16 && y >= 8 && y <= 19) || (y == 13 && x >= 9 && x <= 21) || (y >= 21 && x >= 9 && x <= 21)
				case 1:
					draw = (x-15)*(x-15)+(y-15)*(y-15) < 40 && (x+y)%3 != 0
				case 2:
					draw = (x >= 8 && x <= 11 && y >= 9 && y <= 12) || (x >= 18 && x <= 21 && y >= 10 && y <= 13) || (x >= 13 && x <= 16 && y >= 19 && y <= 22)
				}
				if draw {
					c = colors[i]
				}
				im.SetNRGBA(x, y, c)
			}
		}
		write(name, im)
	}
	f, err := os.Open("img/sausage.png")
	if err != nil {
		panic(err)
	}
	source, err := png.Decode(f)
	f.Close()
	if err != nil {
		panic(err)
	}
	im := image.NewNRGBA(image.Rect(0, 0, 30, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 30; x++ {
			c := color.NRGBAModel.Convert(source.At(x, y)).(color.NRGBA)
			// 明暗とアルファを保ち、緑がかった腐敗色にする。
			v := uint8((uint32(c.R) + uint32(c.G) + uint32(c.B)) / 3)
			im.SetNRGBA(x, y, color.NRGBA{uint8(uint32(v) * 3 / 4), v, v / 2, c.A})
		}
	}
	write("rotten_sausage", im)
}

func write(name string, im image.Image) {
	f, err := os.Create("img/" + name + ".png")
	if err != nil {
		panic(err)
	}
	if err := png.Encode(f, im); err != nil {
		panic(err)
	}
	if err := f.Close(); err != nil {
		panic(err)
	}
}

//go:build ignore

// 店主と番犬の仮画像（wave10-B）。正式画像へ差し替えるまでの30px表示用。
// go run tools/shop_images.go で img/shopkeeper.png・guard_dog.png・elite_guard_dog.png を作る。
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

type painter struct{ img *image.NRGBA }

func (p painter) rect(x0, y0, x1, y1 int, c color.NRGBA) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			p.img.SetNRGBA(x, y, c)
		}
	}
}

func (p painter) disc(cx, cy, r int, c color.NRGBA) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			if (x-cx)*(x-cx)+(y-cy)*(y-cy) <= r*r {
				p.img.SetNRGBA(x, y, c)
			}
		}
	}
}

// outline は不透明なマスに隣り合う透明なマスを暗い色で縁取る（暗い床でも輪郭が見えるように）。
func (p painter) outline(c color.NRGBA) {
	src := image.NewNRGBA(p.img.Rect)
	copy(src.Pix, p.img.Pix)
	for y := 1; y < 29; y++ {
		for x := 1; x < 29; x++ {
			if src.NRGBAAt(x, y).A != 0 {
				continue
			}
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if src.NRGBAAt(x+d[0], y+d[1]).A != 0 {
					p.img.SetNRGBA(x, y, c)
					break
				}
			}
		}
	}
}

func shopkeeper() *image.NRGBA {
	p := painter{image.NewNRGBA(image.Rect(0, 0, 30, 30))}
	skin := color.NRGBA{240, 200, 160, 255}
	apron := color.NRGBA{40, 120, 70, 255}
	p.rect(8, 15, 21, 26, color.NRGBA{230, 230, 220, 255}) // 服
	p.rect(10, 16, 19, 26, apron)                          // 前掛け
	p.rect(12, 19, 17, 21, color.NRGBA{250, 220, 80, 255}) // 小判の印
	p.disc(15, 10, 6, skin)                                // 顔
	p.rect(8, 3, 21, 5, color.NRGBA{60, 40, 30, 255})      // 帽子
	p.rect(11, 1, 18, 3, color.NRGBA{60, 40, 30, 255})
	p.rect(12, 9, 12, 10, color.NRGBA{20, 20, 20, 255}) // 目
	p.rect(17, 9, 17, 10, color.NRGBA{20, 20, 20, 255})
	p.rect(13, 13, 16, 13, color.NRGBA{170, 60, 60, 255}) // 口
	p.rect(10, 27, 12, 28, color.NRGBA{50, 40, 30, 255})  // 足
	p.rect(17, 27, 19, 28, color.NRGBA{50, 40, 30, 255})
	p.outline(color.NRGBA{15, 15, 15, 255})
	return p.img
}

func guardDog(body, dark color.NRGBA) *image.NRGBA {
	p := painter{image.NewNRGBA(image.Rect(0, 0, 30, 30))}
	p.rect(6, 13, 22, 21, body) // 胴
	p.disc(22, 11, 5, body)     // 頭
	p.rect(24, 12, 27, 15, body)
	p.rect(19, 4, 21, 8, dark) // 耳
	p.rect(24, 4, 26, 8, dark)
	p.rect(23, 9, 23, 10, color.NRGBA{255, 60, 40, 255}) // 目
	p.rect(26, 14, 27, 15, color.NRGBA{20, 20, 20, 255}) // 鼻
	p.rect(7, 22, 9, 27, dark)                           // 足
	p.rect(12, 22, 14, 27, dark)
	p.rect(16, 22, 18, 27, dark)
	p.rect(20, 22, 22, 27, dark)
	p.rect(3, 10, 5, 14, dark)                            // 尻尾
	p.rect(16, 13, 18, 15, color.NRGBA{220, 40, 40, 255}) // 首輪
	p.outline(color.NRGBA{15, 15, 15, 255})
	return p.img
}

func save(name string, img *image.NRGBA) {
	f, err := os.Create("img/" + name + ".png")
	if err != nil {
		panic(err)
	}
	if err = png.Encode(f, img); err != nil {
		panic(err)
	}
	if err = f.Close(); err != nil {
		panic(err)
	}
}

func main() {
	save("shopkeeper", shopkeeper())
	save("guard_dog", guardDog(color.NRGBA{170, 120, 70, 255}, color.NRGBA{100, 70, 40, 255}))
	save("elite_guard_dog", guardDog(color.NRGBA{90, 90, 120, 255}, color.NRGBA{45, 45, 70, 255}))
}

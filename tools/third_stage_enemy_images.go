//go:build ignore

// 第3段階の敵画像を第2段階の画像から作る: go run tools/third_stage_enemy_images.go
// 輪郭・アルファ・明暗を保ち、色相だけを回転させる（AGENTS.md の上位種画像の規則）。
// 彩度の低い（灰色に近い）画像は色相を回しても変化が見えないため、彩度を少し足す。
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

// 第2段階の画像 → 第3段階の画像と色相の回転角（度）
var stages = []struct {
	src, dst string
	hue      float64
}{
	{"dai_ebi", "ousama_ebi", 150},
	{"moudoku_hebi", "gekidoku_hebi", 150},
	{"anagura_mamuru", "iwaya_mamuru", 150},
	{"gunegune_honey", "uneune_honey", 150},
	{"harisen_ou", "harisen_koutei", 150},
	{"ganseki_gani", "iwayama_gani", 150},
	{"dynamite_uni", "megaton_uni", 150},
	{"oodorobou_yadokari", "kaitou_yadokari", 150},
	{"nigiri_oyakata_ebi", "nigiri_meijin_ebi", 150},
	{"tatari_gani", "onryou_gani", 150},
	{"shihai_kurage", "teiou_kurage", 150},
	{"shinigami_ebi", "meifu_ebi", 150},
	{"inazuma_shako", "raijin_shako", 150},
	{"telepo_kurage", "jikuu_kurage", 150},
	{"tokkae_dako", "surikae_dako", 150},
	{"wana_master_yadokari", "wana_ou_yadokari", 150},
	{"bake_horagai", "bake_oumugai", 150},
	{"azure_sea_dragon", "shinen_kairyu", 150},
	{"inferno_squid", "rengoku_ika", 150},
	{"arcane_jellyfish", "kenja_kurage", 150},
}

func main() {
	for _, s := range stages {
		if err := convert("img/"+s.src+".png", "img/"+s.dst+".png", s.hue); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func convert(srcPath, dstPath string, hue float64) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()
	src, err := png.Decode(f)
	if err != nil {
		return fmt.Errorf("%s: %w", srcPath, err)
	}
	b := src.Bounds()
	dst := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			if c.A == 0 {
				continue
			}
			h, s, l := rgbToHSL(c.R, c.G, c.B)
			h = math.Mod(h+hue, 360)
			if s < 0.25 && l > 0.08 && l < 0.92 {
				s += 0.25
			}
			r, g, bl := hslToRGB(h, s, l)
			dst.SetNRGBA(x, y, color.NRGBA{r, g, bl, c.A})
		}
	}
	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()
	return png.Encode(out, dst)
}

func rgbToHSL(r8, g8, b8 uint8) (h, s, l float64) {
	r, g, b := float64(r8)/255, float64(g8)/255, float64(b8)/255
	maxV, minV := math.Max(r, math.Max(g, b)), math.Min(r, math.Min(g, b))
	l = (maxV + minV) / 2
	if maxV == minV {
		return 0, 0, l
	}
	d := maxV - minV
	if l > 0.5 {
		s = d / (2 - maxV - minV)
	} else {
		s = d / (maxV + minV)
	}
	switch maxV {
	case r:
		h = math.Mod((g-b)/d+6, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	return h * 60, s, l
}

func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	to8 := func(v float64) uint8 { return uint8(math.Round(math.Max(0, math.Min(1, v+m)) * 255)) }
	return to8(r), to8(g), to8(b)
}

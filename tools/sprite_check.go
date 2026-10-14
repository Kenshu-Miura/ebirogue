//go:build ignore

// 画像規格と暗い床上の輪郭を検査し、4倍の一覧を生成する。
// go run tools/sprite_check.go -sheet docs/agents/done/img-content-1-sheet.png img/<name>.png ...
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

var weapons = strings.Fields("katana gouken gold_sword shell_cutter uni_breaker sky_blade double_strike_sword great_axe healing_sword koban_club iron_mass_sword hungry_knife charge_sword four_prong_harpoon lucky_rake mallet trap_probe katsuobushi cursed_knife golden_pickaxe kajin_tachi giant_harisen")
var shields = strings.Fields("bronze_shield taikai_shield gold_shield dragon_scale_shield nigiri_guard_shield puppet_guard_shield koban_shield iron_mass_shield hungry_shield guts_shield peak_shield satori_shield lucky_charm cursed_pot_lid turtle_shell critical_guard_shield mischief_guard_shield immovable_shield zabuton")
var enemies = strings.Fields("hibana_umiushi kaen_umiushi gouka_umiushi teppou_ebi taihou_ebi gouhou_ebi nige_haze tonzura_haze kinpika_haze yobikomi_bora yobiatsume_bora daishoukan_bora iyashi_namako jihi_namako seibo_namako zenigata_azarashi kinka_azarashi senryou_azarashi sunamoguri sunakake_moguri sunaarashi_moguri yoroi_ebi kabuto_ebi musha_ebi umi_dango oo_umi_dango daiou_umi_dango kaeru_ankou oo_kaeru_ankou nushi_kaeru_ankou hajiki_gani kengou_gani kensei_gani sabi_hitode kusare_hitode doro_hitode itazura_kawahagi warusa_kawahagi akutare_kawahagi oohasami_gani guillotine_gani danzai_gani fukure_fugu bakuhatsu_fugu daibakuhatsu_fugu ana_jako oo_ana_jako jigoku_ana_jako shibire_ei denki_ei raitei_ei mimic_dako henge_dako senmen_dako kobanzame oobanzame senryouzame uchikaeshi_ebi hajikikaeshi_ebi zenbukaeshi_ebi")

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func main() {
	sheetPath := flag.String("sheet", "", "4倍一覧PNGの保存先")
	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 {
		fmt.Fprintln(os.Stderr, "画像ファイルを指定してください")
		os.Exit(1)
	}
	const cellW, cellH, columns = 256, 162, 6
	sheet := image.NewNRGBA(image.Rect(0, 0, columns*cellW, ((len(paths)+columns-1)/columns)*cellH))
	draw.Draw(sheet, sheet.Bounds(), image.NewUniform(color.NRGBA{20, 20, 24, 255}), image.Point{}, draw.Src)
	failures := 0
	for i, path := range paths {
		bad := func(msg string) { fmt.Printf("%s: %s\n", path, msg); failures++ }
		f, err := os.Open(path)
		if err != nil {
			bad(err.Error())
			continue
		}
		im, err := png.Decode(f)
		f.Close()
		if err != nil {
			bad(err.Error())
			continue
		}
		b := im.Bounds()
		if b.Dx() != 30 || b.Dy() != 30 {
			bad("30x30ではない")
			continue
		}
		// PNG IHDR の色形式6はRGBA。パレットやRGBは規格外。
		raw, err := os.ReadFile(path)
		if err != nil || len(raw) < 26 || raw[25] != 6 {
			bad("PNGがRGBA形式ではない")
		}
		name := strings.TrimSuffix(filepath.Base(path), ".png")
		min, max := .30, .85
		if contains(weapons, name) {
			min, max = .07, .60
		} else if !contains(shields, name) && !contains(enemies, name) {
			bad("種類が未登録")
		}
		alpha := func(x, y int) uint8 {
			if x < 0 || x >= 30 || y < 0 || y >= 30 {
				return 0
			}
			return color.NRGBAModel.Convert(im.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA).A
		}
		n, edgeN, edgeL := 0, 0, 0.0
		border, fringe := false, false
		for y := 0; y < 30; y++ {
			for x := 0; x < 30; x++ {
				c := color.NRGBAModel.Convert(im.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
				if c.A == 0 {
					continue
				}
				if x == 0 || y == 0 || x == 29 || y == 29 {
					border = true
				}
				if c.A == 255 {
					n++
				}
				edge := false
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						if alpha(x+dx, y+dy) == 0 {
							edge = true
						}
					}
				}
				if edge {
					edgeN++
					edgeL += .2126*float64(c.R) + .7152*float64(c.G) + .0722*float64(c.B)
					if c.A != 255 {
						fringe = true
					}
				}
			}
		}
		if border {
			bad("四隅または外周に非透明画素")
		}
		if fringe {
			bad("外側に半透明のにじみ")
		}
		fraction := float64(n) / 900
		if fraction < min || fraction > max {
			bad(fmt.Sprintf("不透明率 %.1f%% (規格 %.0f〜%.0f%%)", fraction*100, min*100, max*100))
		}
		for _, floor := range []color.NRGBA{{38, 25, 21, 255}, {95, 97, 102, 255}} {
			floorL := .2126*float64(floor.R) + .7152*float64(floor.G) + .0722*float64(floor.B)
			if edgeN == 0 {
				bad("外周画素なし")
				break
			}
			diff := edgeL/float64(edgeN) - floorL
			if diff < 0 {
				diff = -diff
			}
			if diff < 20 {
				bad(fmt.Sprintf("床 %v と輪郭の明度差 %.1f < 20", floor, diff))
			}
		}
		cx, cy := i%columns*cellW, i/columns*cellH
		for p, floor := range []color.NRGBA{{38, 25, 21, 255}, {95, 97, 102, 255}} {
			for y := 0; y < 120; y++ {
				for x := 0; x < 120; x++ {
					c := color.NRGBAModel.Convert(im.At(b.Min.X+x/4, b.Min.Y+y/4)).(color.NRGBA)
					if c.A == 0 {
						c = floor
					} else if c.A != 255 {
						a := uint32(c.A)
						c = color.NRGBA{uint8((uint32(c.R)*a + uint32(floor.R)*(255-a)) / 255), uint8((uint32(c.G)*a + uint32(floor.G)*(255-a)) / 255), uint8((uint32(c.B)*a + uint32(floor.B)*(255-a)) / 255), 255}
					}
					sheet.SetNRGBA(cx+p*124+x+4, cy+y+4, c)
				}
			}
		}
		d := font.Drawer{Dst: sheet, Src: image.NewUniform(color.White), Face: basicfont.Face7x13, Dot: fixed.P(cx+4, cy+140)}
		d.DrawString(name)
	}
	if *sheetPath != "" {
		if err := os.MkdirAll(filepath.Dir(*sheetPath), 0755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		f, err := os.Create(*sheetPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		err = png.Encode(f, sheet)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			fmt.Fprintln(os.Stderr, err, closeErr)
			os.Exit(1)
		}
	}
	if failures > 0 {
		fmt.Printf("%d violations\n", failures)
		os.Exit(1)
	}
	fmt.Printf("OK: %d sprites\n", len(paths))
}

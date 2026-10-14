package main

import (
	"image"
	"testing"
)

func TestPlaytestActionMenuHeight(t *testing.T) {
	r := actionMenuRect(640, 480, 5, false)
	if lastBaseline := r.Min.Y + 20 + 4*20; lastBaseline >= r.Max.Y-2 {
		t.Fatal("fifth row touches the bottom border", r)
	}
}

func TestPlaytestActionMenuLayout(t *testing.T) {
	for _, count := range []int{4, 5} {
		r := actionMenuRect(640, 480, count, true)
		// 枠の2pxも含めて画面・HUD・メッセージ・8方向の各マスから離す。
		border := r.Inset(-2)
		if !border.In(image.Rect(0, 0, 640, 480)) || border.Overlaps(image.Rect(510, 0, 640, 135)) || border.Overlaps(image.Rect(70, 350, 570, 470)) {
			t.Fatal("menu overlaps HUD/message or leaves screen", r)
		}
		origin := image.Rect(275, 195, 305, 225)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				for step := 0; step <= 30; step++ {
					tile := origin.Add(image.Pt(dx*30*step, dy*30*step))
					if border.Overlaps(tile) {
						t.Fatalf("menu %v overlaps direction %d,%d tile %v", r, dx, dy, tile)
					}
				}
			}
		}
		for i := 0; i < count; i++ {
			if y := r.Min.Y + 20 + i*20; y >= r.Max.Y-2 {
				t.Fatal("last row touches border")
			}
		}
		normal := actionMenuRect(640, 480, count, false)
		if normal.Dy() != count*20+10 {
			t.Fatal("normal menu does not fit all rows")
		}
	}
}

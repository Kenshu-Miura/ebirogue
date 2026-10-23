//go:build !test

package main

import (
	"bytes"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/basicfont"
	"strings"
	"testing"
)

func TestCodexDrawAllEntriesPreservesRNG(t *testing.T) {
	g, _ := newHeadlessGame(8, DifficultyBeginner)
	g.codex = newCodexData()
	g.codexLoaded = true
	normal, medium, small := mplusNormalFont, mplusMediumFont, mplusSmallFont
	mplusNormalFont, mplusMediumFont, mplusSmallFont = basicfont.Face7x13, basicfont.Face7x13, basicfont.Face7x13
	defer func() { mplusNormalFont, mplusMediumFont, mplusSmallFont = normal, medium, small }()
	before, _ := g.rng.MarshalState()
	turns := g.turnCount
	screen := ebiten.NewImage(640, 480)
	defer screen.Deallocate()
	for tab, records := range []map[int]CodexRecord{g.codex.Items, g.codex.Enemies, g.codex.Traps} {
		g.codexTab = tab
		for _, id := range codexIDs(tab) {
			records[id] = CodexRecord{Seen: true, Identified: true}
		}
		for i := range g.codexEntries() {
			g.codexCursor = i
			g.drawCodex(screen)
		}
	}
	after, _ := g.rng.MarshalState()
	if !bytes.Equal(before, after) || g.turnCount != turns {
		t.Fatal("drawing changed the adventure")
	}
}

// 図鑑のすべての敵・罠に説明文があり（道具を含め詳細の枠に収まる長さ）、ほかの敵の説明を取り違えないこと。
func TestCodexEveryEnemyAndTrapHasDescription(t *testing.T) {
	itemDescription := func(id int) string { return itemTemplates[id].Description }
	for tab, describe := range map[int]func(int) string{0: itemDescription, 1: codexEnemyDescription, 2: codexTrapDescription} {
		for _, id := range codexIDs(tab) {
			desc := describe(id)
			if desc == "" && tab != 0 {
				t.Errorf("タブ%d の ID %d に説明文が無い", tab, id)
			}
			if lines := codexWrap(desc, codexDescWidth, codexDescLines); len(lines) == codexDescLines && strings.HasSuffix(lines[codexDescLines-1], "…") {
				t.Errorf("タブ%d の ID %d の説明文が長すぎて途中で切れる", tab, id)
			}
		}
	}
	if codexEnemyDescription(0) == codexEnemyDescription(11) {
		t.Errorf("エビにユウレイエビの説明が出ている: %q", codexEnemyDescription(0))
	}
	if codexEnemyDescription(36) == codexEnemyDescription(44) {
		t.Errorf("海竜に海竜王の説明が出ている: %q", codexEnemyDescription(36))
	}
	if codexTrapDescription(3) == codexTrapDescription(11) {
		t.Errorf("地雷に大型地雷の説明が出ている: %q", codexTrapDescription(3))
	}
	// 上位種は基本種の説明を引き継ぐ
	for base, upper := range map[int]int{0: 18, 18: 45, 1: 20, 66: 67, 67: 68} {
		if codexEnemyDescription(upper) != codexEnemyDescription(base) {
			t.Errorf("上位種 %d の説明が基本種 %d と違う", upper, base)
		}
	}
}

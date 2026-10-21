//go:build !test

package main

import (
	"bytes"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font/basicfont"
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
		for page := 0; page < len(g.codexEntries()); page++ {
			g.codexPage = page
			g.drawCodex(screen)
		}
	}
	after, _ := g.rng.MarshalState()
	if !bytes.Equal(before, after) || g.turnCount != turns {
		t.Fatal("drawing changed the adventure")
	}
}

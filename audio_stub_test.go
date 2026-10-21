//go:build test

package main

import "testing"

func TestAudioStubNoop(t *testing.T) {
	g := &Game{}
	g.playSE("hit")
	g.playBGM("dungeon")
	g.stopBGM()
}

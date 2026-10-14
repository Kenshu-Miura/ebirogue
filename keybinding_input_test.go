//go:build !test

package main

import (
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestKeyBindingPhysicalInput(t *testing.T) {
	settings := defaultSettings()
	settings.KeyBindings = bindPhysicalKey(settings.KeyBindings, KeyZ, "Q")
	held := map[ebiten.Key]int{ebiten.KeyQ: 1, ebiten.KeyEnter: 1}
	in := ebitenInput{settings: &settings, duration: func(k ebiten.Key) int { return held[k] }}
	if !in.Pressed(KeyZ) || !in.JustPressed(KeyZ) || in.PressDuration(KeyZ) != 1 || !in.Pressed(KeyEnter) {
		t.Fatal("mapped input unavailable")
	}
	held[ebiten.KeyQ] = 10
	if in.JustPressed(KeyZ) || in.PressDuration(KeyZ) != 10 {
		t.Fatal("repeat duration changed")
	}
	delete(held, ebiten.KeyQ)
	held[ebiten.KeyZ] = 1
	if in.Pressed(KeyZ) {
		t.Fatal("old physical key still active")
	}
	for k := Key(0); k < KeyEnter; k++ {
		if ebitenInputDefault := (ebitenInput{settings: &GameSettings{KeyBindings: defaultKeyBindings()}}).physicalKey(k); ebitenInputDefault != ebitenKeys[k] {
			t.Fatalf("default mismatch %s", k)
		}
	}
	for name, physical := range physicalKeys {
		if name != "Shift" && physical.String() != name && physical.String() != "Arrow"+name && !(len(name) == 1 && name[0] >= '0' && name[0] <= '9' && physical.String() == "Digit"+name) {
			t.Fatalf("incorrect mapping %s=%s", name, physical)
		}
	}
}

func TestKeyConfigFlowWithoutTurns(t *testing.T) {
	g, _ := newHeadlessGame(123, DifficultyBeginner)
	g.settings = defaultSettings()
	v := NewVirtualInput()
	g.input = v
	g.showSettings = true
	g.settingsSelectedIndex = 2
	turn := g.turnCount
	v.Press(KeyZ)
	v.Update()
	if err := g.handleSettingsInput(); err != nil || !g.showKeyConfig {
		t.Fatal("settings did not open bindings")
	}
	g.handleKeyConfigFrame("Z", true)
	if g.keyConfigCapturing {
		t.Fatal("opening press captured")
	}
	v.Release(KeyZ)
	v.Update()
	g.handleKeyConfigFrame("", false)
	g.keyConfigSelected = int(KeyZ)
	v.Press(KeyZ)
	v.Update()
	g.handleKeyConfigFrame("Z", true)
	if !g.keyConfigCapturing {
		t.Fatal("capture did not start")
	}
	v.Release(KeyZ)
	v.Update()
	g.handleKeyConfigFrame("", false)
	g.handleKeyConfigFrame("X", true)
	if g.settings.KeyBindings["Z"] != "X" || g.settings.KeyBindings["X"] != "Z" {
		t.Fatal("capture did not swap")
	}
	g.handleKeyConfigFrame("X", true)
	if !g.showKeyConfig {
		t.Fatal("capture leaked into cancel")
	}
	g.handleKeyConfigFrame("", false)
	g.keyConfigSelected = len(keyRoles)
	v.Press(KeyZ)
	v.Update()
	g.handleKeyConfigFrame("", true)
	if !reflect.DeepEqual(g.settings.KeyBindings, defaultKeyBindings()) {
		t.Fatal("reset failed")
	}
	v.Release(KeyZ)
	v.Update()
	g.handleKeyConfigFrame("", false)
	v.Press(KeyX)
	v.Update()
	g.handleKeyConfigFrame("", true)
	if g.showKeyConfig || !g.showSettings || g.turnCount != turn || g.isActioned {
		t.Fatal("UI changed adventure or failed return")
	}
	v.Release(KeyX)
	v.Press(KeyC)
	v.Update()
	g.showKeyConfig = true
	g.handleKeyConfigFrame("", true)
	if g.showKeyConfig || g.showSettings || g.turnCount != turn {
		t.Fatal("menu key did not close settings")
	}
}

//go:build !test

package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// ebitenKeys は論理キーと物理キーの対応表。
var ebitenKeys = [keyCount]ebiten.Key{
	KeyUp:    ebiten.KeyUp,
	KeyDown:  ebiten.KeyDown,
	KeyLeft:  ebiten.KeyLeft,
	KeyRight: ebiten.KeyRight,
	KeyShift: ebiten.KeyShift,
	KeyZ:     ebiten.KeyZ,
	KeyX:     ebiten.KeyX,
	KeyA:     ebiten.KeyA,
	KeyC:     ebiten.KeyC,
	KeyD:     ebiten.KeyD,
	KeySpace: ebiten.KeySpace,
	KeyF:     ebiten.KeyF,
	KeyS:     ebiten.KeyS,
	KeyN:     ebiten.KeyN,
	KeyH:     ebiten.KeyH,
	KeyL:     ebiten.KeyL,
	KeyEnter: ebiten.KeyEnter,
	KeyF1:    ebiten.KeyF1,
}

// ebitenInput は実際のキーボードを読む InputSource。
// inpututil がフレームごとの状態更新を行うため Update では何もしない。
type ebitenInput struct {
	settings *GameSettings
	// テストでは物理キーの状態を注入する。通常は Ebiten から取得する。
	duration func(ebiten.Key) int
}

func (in ebitenInput) physicalKey(k Key) ebiten.Key {
	if in.settings != nil && k >= 0 && int(k) < len(keyRoles) {
		if physical, ok := physicalKeys[bindingName(in.settings.KeyBindings, k)]; ok {
			return physical
		}
	}
	return ebitenKeys[k]
}

func (in ebitenInput) Pressed(k Key) bool {
	if in.duration != nil {
		return in.PressDuration(k) > 0
	}
	return ebiten.IsKeyPressed(in.physicalKey(k))
}

func (in ebitenInput) JustPressed(k Key) bool {
	if in.duration != nil {
		return in.PressDuration(k) == 1
	}
	return inpututil.IsKeyJustPressed(in.physicalKey(k))
}

func (in ebitenInput) PressDuration(k Key) int {
	if in.duration != nil {
		return in.duration(in.physicalKey(k))
	}
	return inpututil.KeyPressDuration(in.physicalKey(k))
}

var physicalKeys = makePhysicalKeys()

func makePhysicalKeys() map[string]ebiten.Key {
	keys := map[string]ebiten.Key{"Up": ebiten.KeyUp, "Down": ebiten.KeyDown, "Left": ebiten.KeyLeft, "Right": ebiten.KeyRight, "Space": ebiten.KeySpace, "Shift": ebiten.KeyShift, "Tab": ebiten.KeyTab, "Backspace": ebiten.KeyBackspace}
	for c := 'A'; c <= 'Z'; c++ {
		keys[string(c)] = ebiten.KeyA + ebiten.Key(c-'A')
	}
	for c := '0'; c <= '9'; c++ {
		keys[string(c)] = ebiten.KeyDigit0 + ebiten.Key(c-'0')
	}
	return keys
}

// capturePhysicalKey は対応するキーだけを読む。左右の Shift は共通扱い。
func capturePhysicalKey() (pressed string, held bool) {
	for _, name := range physicalKeyNames() {
		physical := physicalKeys[name]
		if ebiten.IsKeyPressed(physical) {
			held = true
		}
		if pressed == "" && inpututil.IsKeyJustPressed(physical) {
			pressed = name
		}
	}
	return
}

func (ebitenInput) Update() {}

// keys はゲームが使う入力元を返す。未設定（テストで直接作った Game など）なら実キーボードを使う。
func (g *Game) keys() InputSource {
	if g.input == nil {
		g.input = ebitenInput{settings: &g.settings}
	}
	return g.input
}

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
type ebitenInput struct{}

func (ebitenInput) Pressed(k Key) bool { return ebiten.IsKeyPressed(ebitenKeys[k]) }

func (ebitenInput) JustPressed(k Key) bool { return inpututil.IsKeyJustPressed(ebitenKeys[k]) }

func (ebitenInput) PressDuration(k Key) int { return inpututil.KeyPressDuration(ebitenKeys[k]) }

func (ebitenInput) Update() {}

// keys はゲームが使う入力元を返す。未設定（テストで直接作った Game など）なら実キーボードを使う。
func (g *Game) keys() InputSource {
	if g.input == nil {
		g.input = ebitenInput{}
	}
	return g.input
}

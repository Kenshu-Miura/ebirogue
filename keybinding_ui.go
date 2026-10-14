//go:build !test

package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
)

// handleKeyConfigFrame はキー設定だけを処理し、冒険のターンを進めない。
func (g *Game) handleKeyConfigFrame(physical string, held bool) {
	if g.keyConfigWaitRelease {
		if !held {
			g.keyConfigWaitRelease = false
		}
		return
	}
	if g.keyConfigCapturing {
		if physical != "" {
			g.settings.KeyBindings = bindPhysicalKey(g.settings.KeyBindings, Key(g.keyConfigSelected), physical)
			g.keyConfigCapturing = false
			g.keyConfigWaitRelease = true
			g.saveSettings()
		}
		return
	}
	count := len(keyRoles) + 1
	if g.keys().JustPressed(KeyUp) {
		g.keyConfigSelected = (g.keyConfigSelected + count - 1) % count
	} else if g.keys().JustPressed(KeyDown) {
		g.keyConfigSelected = (g.keyConfigSelected + 1) % count
	}
	if g.keys().JustPressed(KeyC) {
		g.showKeyConfig = false
		g.showSettings = false
		g.saveSettings()
		return
	}
	if g.keys().JustPressed(KeyX) {
		g.showKeyConfig = false
		g.saveSettings()
		return
	}
	if g.keys().JustPressed(KeyZ) {
		if g.keyConfigSelected == len(keyRoles) {
			g.settings.KeyBindings = defaultKeyBindings()
			g.saveSettings()
		} else {
			g.keyConfigCapturing = true
		}
		g.keyConfigWaitRelease = true
	}
}

func (g *Game) keyGuide(guide string) string { return keyGuide(g.settings.KeyBindings, guide) }

func (g *Game) drawKeyConfigWindow(screen *ebiten.Image) {
	drawWindowWithBorder(screen, 30, 10, 580, 460, 240)
	text.Draw(screen, "キー設定", mplusMediumFont, 45, 40, color.White)
	for i := 0; i <= len(keyRoles); i++ {
		label := "初期設定に戻す"
		if i < len(keyRoles) {
			label = fmt.Sprintf("%s：%s", keyRoles[i], bindingName(g.settings.KeyBindings, Key(i)))
		}
		c := color.Color(color.White)
		if i == g.keyConfigSelected {
			label = "→ " + label
			c = color.NRGBA{255, 255, 0, 255}
		}
		text.Draw(screen, label, mplusNormalFont, 50, 65+i*21, c)
	}
	guide := g.keyGuide("↑/↓:選択  Z:変更  X:戻る  C:閉じる")
	if g.keyConfigCapturing {
		guide = "新しいキーを押してください（使用中のキーは入れ替え）"
	}
	text.Draw(screen, guide, mplusSmallFont, 45, 440, color.White)
	text.Draw(screen, "英字・数字・方向・Space・Shift・Tab・Backspace", mplusSmallFont, 45, 460, color.White)
}

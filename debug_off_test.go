//go:build !debug && !test

package main

import "testing"

// 通常のビルドではデバッグ用の F1（混乱薬の追加）が無効になっている。
func TestF1DebugDisabledInNormalBuild(t *testing.T) {
	g, v := newHeadlessGame(1, DifficultyAdvanced)
	tapHeadless(t, g, v, KeyF1)
	if len(g.state.Player.Inventory) != 0 {
		t.Fatalf("F1 added items in a normal build: %d", len(g.state.Player.Inventory))
	}
}

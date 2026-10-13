//go:build !test

package main

import (
	"errors"
	"runtime/debug"
	"testing"
)

// fuzzKeys はバイト列から押すキーを決める表（デバッグ用の F1 は除く）。
var fuzzKeys = []Key{KeyUp, KeyDown, KeyLeft, KeyRight, KeyShift, KeyZ, KeyX, KeyA, KeyC, KeyD, KeySpace, KeyF, KeyS, KeyN, KeyH, KeyL, KeyEnter}

// fuzzMaxInputs は1つの入力列で処理するキー操作の上限（1回の実行時間を抑える）。
const fuzzMaxInputs = 2000

// playInputBytes はバイト列を入力の列として遊び、不変条件違反を返す。
// 先頭8バイトをシード、以降の各バイトを「押すキー（下位5ビット）＋同時押しの有無」として読む。
func playInputBytes(data []byte) (violations []simViolation) {
	var seed int64
	for i := 0; i < 8 && i < len(data); i++ {
		seed = seed<<8 | int64(data[i])
	}
	if len(data) > 8 {
		data = data[8:]
	} else {
		data = nil
	}
	g, in := newHeadlessGame(seed&0x7fffffff, Difficulty(int(seed&0x7fffffff)%len(allDifficulties)))
	defer func() {
		if r := recover(); r != nil {
			violations = append(violations, newViolation(g, "panic", "%v\n%s", r, debug.Stack()))
		}
	}()
	b := newBot(g, in, personaRandom, seed)
	if err := g.RunUntilInputReady(botMaxFrames); err != nil {
		return []simViolation{newViolation(g, "softlock", "%v", err)}
	}
	for i := 0; i < len(data) && i < fuzzMaxInputs && !g.RunEnded(); i++ {
		keys := []Key{fuzzKeys[int(data[i]&0x1f)%len(fuzzKeys)]}
		if data[i]&0x80 != 0 && i+1 < len(data) { // 最上位ビットが立っていれば次のバイトのキーも同時に押す
			i++
			keys = append(keys, fuzzKeys[int(data[i]&0x1f)%len(fuzzKeys)])
		}
		if err := b.tap(keys...); err != nil {
			if errors.Is(err, ErrSoftLock) {
				return append(violations, newViolation(g, "softlock", "%v", err))
			}
			return append(violations, newViolation(g, "error", "%v", err))
		}
		if vs := checkInvariants(g); len(vs) > 0 {
			return append(violations, vs...)
		}
	}
	return violations
}

// FuzzInputSequence はバイト列を入力の列に変換して遊ばせ、不変条件を検査する。
//
//	go test -fuzz=FuzzInputSequence -fuzztime=30m .
func FuzzInputSequence(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 1, 3, 3, 3, 5})                            // 右へ歩いて攻撃
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 2, 8, 5, 5, 1, 5, 5, 8})                   // メニュー→道具→使う
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 3, 0x86, 0, 0x86, 1, 0x86, 2, 0x86, 3, 5}) // ダッシュ
	f.Add([]byte{0, 0, 0, 0, 0, 0, 0, 4, 8, 1, 1, 3, 5, 0, 5, 5, 8})             // 再挑戦・ヘルプ周辺
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, v := range playInputBytes(data) {
			t.Errorf("違反 [%s] floor=%d turn=%d: %s", v.Kind, v.Floor, v.Turn, v.Detail)
		}
	})
}

//go:build !test

package main

import (
	"errors"
	"fmt"
	"log"
)

// ErrSoftLock は上限フレーム内に入力を受け付けられる状態へ戻らなかったことを表す。
var ErrSoftLock = errors.New("ソフトロック: 入力を受け付けられる状態に戻らない")

// newHeadlessGame は画像を読み込まずに、描画なし・演出スキップで動くゲームを作る（BOT・テスト用）。
// 入力は仮想入力（VirtualInput）を返すので、キーを押して操作する。
// Draw は呼ばないこと。セーブファイル・設定ファイルには書き込まない。
func newHeadlessGame(seed int64, difficulty Difficulty) (*Game, *VirtualInput) {
	input := NewVirtualInput()
	g := &Game{
		input:                input,
		headless:             true,
		fastForward:          true,
		difficulty:           difficulty,
		tmpSelectedItemIndex: -1,
		customNames:          map[int]string{},
		settings:             defaultSettings(),
	}
	g.startNewRun(seed)
	g.fadeInProgress = 1.0
	return g, input
}

// StepFrame は1フレームだけゲームを進める。
func (g *Game) StepFrame() error {
	return g.Update()
}

// RunUntilInputReady は入力を受け付けられる状態（または冒険終了）まで最大 maxFrames フレーム進める。
// 入力待ちの状態なら1フレームも進めない。上限を超えたら ErrSoftLock を返す。
func (g *Game) RunUntilInputReady(maxFrames int) error {
	for i := 0; ; i++ {
		if g.runEnded || g.isInputReady() {
			return nil
		}
		if i >= maxFrames {
			return fmt.Errorf("%w（%dフレーム、%d階、%dターン）", ErrSoftLock, maxFrames, g.Floor, g.turnCount)
		}
		if err := g.StepFrame(); err != nil {
			return err
		}
	}
}

// RunEnded はヘッドレスの冒険が終了したか（死亡・クリア）を返す。
func (g *Game) RunEnded() bool {
	return g.runEnded
}

// debugf は開発用のログを出力する。ヘッドレス（大量のシミュレーション）では出力しない。
func (g *Game) debugf(format string, args ...any) {
	if g.headless {
		return
	}
	log.Printf(format, args...)
}

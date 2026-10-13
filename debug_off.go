//go:build !debug && !test

package main

// 通常のビルドではデバッグ用の処理を無効にする（有効にするには go run -tags debug .）。

// processF1KeyPress は通常のビルドでは何もしない（debug_on.go 参照）。
func (g *Game) processF1KeyPress() {}

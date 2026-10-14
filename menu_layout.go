package main

import "image"

// プレビュー時は中央から伸びる水平・垂直・斜め8方向の射線を避ける。
// 右端、HUD と中央の水平線の下、メッセージ欄より上に収める。
func actionMenuRect(screenW, screenH, count int, preview bool) image.Rectangle {
	w, h := 200, count*20+10
	x, y := (screenW-w)/2, (screenH-h)/2
	if preview {
		w = 150
		x, y = screenW-w-10, screenH-120-h-20
	}
	return image.Rect(x, y, x+w, y+h)
}

func previewItemNameRect(screenW int) image.Rectangle {
	return image.Rect(170, 50, screenW-150, 76)
}

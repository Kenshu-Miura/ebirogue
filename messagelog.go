package main

// メッセージ履歴の保持件数と履歴ウィンドウの表示行数
const (
	maxLogMessages     = 100
	messageLogPageSize = 15
)

// MessageLog は画面下に表示されたメッセージの履歴を保持する
type MessageLog struct {
	Messages []string
}

// Add は空でないメッセージを履歴の末尾へ追加し、上限を超えた古いものを削除する
func (ml *MessageLog) Add(msg string) {
	if msg == "" {
		return
	}
	ml.Messages = append(ml.Messages, msg)
	if len(ml.Messages) > maxLogMessages {
		ml.Messages = ml.Messages[len(ml.Messages)-maxLogMessages:]
	}
}

// MaxScroll は表示行数 count のときにさかのぼれる最大スクロール量を返す
func (ml *MessageLog) MaxScroll(count int) int {
	maxScroll := len(ml.Messages) - count
	if maxScroll < 0 {
		return 0
	}
	return maxScroll
}

// Visible はスクロール量 scroll のときに表示するメッセージを古い順で返す。
// scroll=0 で最新のメッセージが末尾に来る。
func (ml *MessageLog) Visible(scroll, count int) []string {
	if scroll < 0 {
		scroll = 0
	}
	if maxScroll := ml.MaxScroll(count); scroll > maxScroll {
		scroll = maxScroll
	}
	end := len(ml.Messages) - scroll
	start := end - count
	if start < 0 {
		start = 0
	}
	return ml.Messages[start:end]
}

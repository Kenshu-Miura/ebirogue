package main

import (
	"fmt"
	"testing"
)

func TestMessageLogAdd(t *testing.T) {
	var ml MessageLog

	// 空文字列は追加されない
	ml.Add("")
	if len(ml.Messages) != 0 {
		t.Errorf("空文字列が追加された: %d件", len(ml.Messages))
	}

	ml.Add("メッセージ1")
	ml.Add("メッセージ2")
	if len(ml.Messages) != 2 {
		t.Fatalf("メッセージ数が期待値と異なる: got %d, want 2", len(ml.Messages))
	}
	if ml.Messages[0] != "メッセージ1" || ml.Messages[1] != "メッセージ2" {
		t.Errorf("メッセージの順序が不正: %v", ml.Messages)
	}
}

func TestMessageLogAddTrimsOldMessages(t *testing.T) {
	var ml MessageLog
	for i := 0; i < maxLogMessages+10; i++ {
		ml.Add(fmt.Sprintf("メッセージ%d", i))
	}
	if len(ml.Messages) != maxLogMessages {
		t.Fatalf("保持件数が上限を超えている: got %d, want %d", len(ml.Messages), maxLogMessages)
	}
	// 最も古いメッセージが削除され、最新のメッセージが残る
	if ml.Messages[0] != "メッセージ10" {
		t.Errorf("先頭メッセージが不正: got %s, want メッセージ10", ml.Messages[0])
	}
	last := ml.Messages[len(ml.Messages)-1]
	want := fmt.Sprintf("メッセージ%d", maxLogMessages+9)
	if last != want {
		t.Errorf("末尾メッセージが不正: got %s, want %s", last, want)
	}
}

func TestMessageLogMaxScroll(t *testing.T) {
	var ml MessageLog

	// メッセージが表示行数以下ならスクロール不可
	if got := ml.MaxScroll(5); got != 0 {
		t.Errorf("空の履歴のMaxScroll: got %d, want 0", got)
	}
	for i := 0; i < 5; i++ {
		ml.Add(fmt.Sprintf("m%d", i))
	}
	if got := ml.MaxScroll(5); got != 0 {
		t.Errorf("行数ちょうどのMaxScroll: got %d, want 0", got)
	}

	ml.Add("m5")
	if got := ml.MaxScroll(5); got != 1 {
		t.Errorf("6件・5行表示のMaxScroll: got %d, want 1", got)
	}
}

func TestMessageLogVisible(t *testing.T) {
	var ml MessageLog
	for i := 0; i < 10; i++ {
		ml.Add(fmt.Sprintf("m%d", i))
	}

	// scroll=0 では最新のメッセージが末尾に来る
	visible := ml.Visible(0, 3)
	if len(visible) != 3 {
		t.Fatalf("表示件数が不正: got %d, want 3", len(visible))
	}
	if visible[0] != "m7" || visible[2] != "m9" {
		t.Errorf("scroll=0の表示範囲が不正: %v", visible)
	}

	// スクロールすると古いメッセージへさかのぼる
	visible = ml.Visible(2, 3)
	if visible[0] != "m5" || visible[2] != "m7" {
		t.Errorf("scroll=2の表示範囲が不正: %v", visible)
	}

	// 最大値を超えるスクロールは最古のメッセージ位置に丸められる
	visible = ml.Visible(100, 3)
	if visible[0] != "m0" || visible[2] != "m2" {
		t.Errorf("スクロール上限丸めの表示範囲が不正: %v", visible)
	}

	// 負のスクロールは0として扱う
	visible = ml.Visible(-1, 3)
	if visible[2] != "m9" {
		t.Errorf("負のスクロールの表示範囲が不正: %v", visible)
	}

	// メッセージが表示行数より少ない場合は全件を返す
	var short MessageLog
	short.Add("a")
	visible = short.Visible(0, 3)
	if len(visible) != 1 || visible[0] != "a" {
		t.Errorf("少数メッセージの表示が不正: %v", visible)
	}
}

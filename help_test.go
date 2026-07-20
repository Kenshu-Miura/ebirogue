package main

import (
	"strings"
	"testing"
)

func TestHelpPagesContent(t *testing.T) {
	pages := helpPages()
	if len(pages) != 3 {
		t.Fatalf("expected 3 help pages, got %d", len(pages))
	}
	for i, page := range pages {
		if page.Title == "" {
			t.Fatalf("help page %d has empty title", i)
		}
		if len(page.Lines) == 0 {
			t.Fatalf("help page %q has no lines", page.Title)
		}
	}

	// 状態異常ページは formatPlayerStatus が表示する全ての状態異常を説明する
	ailments := []string{"毒", "鈍足", "倍速", "睡眠", "混乱", "目潰し", "金縛り", "封印"}
	statusPage := strings.Join(pages[1].Lines, "\n")
	for _, ailment := range ailments {
		if !strings.Contains(statusPage, ailment) {
			t.Errorf("status ailment page should describe %q", ailment)
		}
	}
}

func TestHelpMaxScroll(t *testing.T) {
	if got := helpMaxScroll(10, 15); got != 0 {
		t.Errorf("short content should not scroll, got %d", got)
	}
	if got := helpMaxScroll(20, 15); got != 5 {
		t.Errorf("expected max scroll 5, got %d", got)
	}
	if got := helpMaxScroll(15, 15); got != 0 {
		t.Errorf("content equal to page size should not scroll, got %d", got)
	}
}

func TestHelpVisibleLines(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}

	got := helpVisibleLines(lines, 0, 3)
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Errorf("scroll 0 should show first 3 lines, got %v", got)
	}

	got = helpVisibleLines(lines, 2, 3)
	if len(got) != 3 || got[0] != "c" || got[2] != "e" {
		t.Errorf("scroll 2 should show last 3 lines, got %v", got)
	}

	// 範囲外のスクロール量は丸められる
	got = helpVisibleLines(lines, 10, 3)
	if len(got) != 3 || got[0] != "c" {
		t.Errorf("over-scroll should clamp to last page, got %v", got)
	}
	got = helpVisibleLines(lines, -1, 3)
	if len(got) != 3 || got[0] != "a" {
		t.Errorf("negative scroll should clamp to first page, got %v", got)
	}

	// 表示行数より少ない内容は全て表示される
	got = helpVisibleLines(lines, 0, 10)
	if len(got) != 5 {
		t.Errorf("expected all 5 lines, got %v", got)
	}
}

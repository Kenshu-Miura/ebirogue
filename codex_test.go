package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCodexDecodeAndMerge(t *testing.T) {
	for _, bad := range []string{`{`, `null`, `{"version":2}`, `{"items":{"-1":{}}}`, `{"enemies":{"1":{"defeated":-1}}}`} {
		if _, err := decodeCodex([]byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
	c, err := decodeCodex([]byte(`{"items":{"5":{"identified":true}}}`))
	if err != nil || !c.Items[5].Seen {
		t.Fatal(c, err)
	}
	c.Enemies[0] = CodexRecord{Seen: true, Defeated: 2}
	data, _ := json.Marshal(c)
	back, err := decodeCodex(data)
	if err != nil || !reflect.DeepEqual(c, back) {
		t.Fatal(err, back)
	}
	d := newCodexData()
	d.Enemies[0] = CodexRecord{Deaths: 3}
	merged := mergeCodex(c, d)
	if merged.Enemies[0].Defeated != 2 || merged.Enemies[0].Deaths != 3 || !reflect.DeepEqual(merged, mergeCodex(merged, merged)) {
		t.Fatal(merged)
	}
	n, m, p := codexDiscovery([]int{0, 2}, merged.Enemies)
	if n != 1 || m != 2 || p != 50 {
		t.Fatal(n, m, p)
	}
	n, m, p = codexDiscovery(nil, nil)
	if n != 0 || m != 0 || p != 0 {
		t.Fatal(n, m, p)
	}
}

func TestCodexListCursorAndScroll(t *testing.T) {
	cases := []struct {
		cursor, delta, n int
		wrap             bool
		want             int
	}{
		{0, -1, 10, true, 9}, {0, -1, 10, false, 0}, {9, 1, 10, true, 0}, {9, 1, 10, false, 9}, {3, 1, 10, false, 4}, {0, 1, 0, true, 0},
	}
	for _, c := range cases {
		if got := codexMoveCursor(c.cursor, c.delta, c.n, c.wrap); got != c.want {
			t.Errorf("codexMoveCursor(%d,%d,%d,%v)=%d want %d", c.cursor, c.delta, c.n, c.wrap, got, c.want)
		}
	}
	scrolls := []struct{ cursor, scroll, rows, n, want int }{
		{0, 0, 5, 3, 0},    // 一覧が短い
		{4, 0, 5, 20, 0},   // 見えている
		{5, 0, 5, 20, 1},   // 下へはみ出した
		{2, 6, 5, 20, 2},   // 上へはみ出した
		{19, 0, 5, 20, 15}, // 末尾へ回り込んだ
		{0, 15, 5, 20, 0},  // 先頭へ回り込んだ
		{3, 30, 5, 20, 3},  // 先頭行が範囲外
	}
	for _, c := range scrolls {
		if got := codexScrollFor(c.cursor, c.scroll, c.rows, c.n); got != c.want {
			t.Errorf("codexScrollFor(%d,%d,%d,%d)=%d want %d", c.cursor, c.scroll, c.rows, c.n, got, c.want)
		}
	}
}

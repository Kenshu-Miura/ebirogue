package main

import (
	"bytes"
	"encoding/json"
	"fmt"
)

const codexFileName = "ebirogue_codex.json"

// CodexRecord は冒険をまたぐ発見記録。未知の ID も将来のために保持する。
type CodexRecord struct {
	Seen       bool `json:"seen,omitempty"`
	Identified bool `json:"identified,omitempty"`
	Defeated   int  `json:"defeated,omitempty"`
	Deaths     int  `json:"deaths,omitempty"`
	Stepped    bool `json:"stepped,omitempty"`
}
type CodexData struct {
	Version int                 `json:"version"`
	Items   map[int]CodexRecord `json:"items"`
	Enemies map[int]CodexRecord `json:"enemies"`
	Traps   map[int]CodexRecord `json:"traps"`
}

func newCodexData() CodexData {
	return CodexData{1, map[int]CodexRecord{}, map[int]CodexRecord{}, map[int]CodexRecord{}}
}
func decodeCodex(data []byte) (CodexData, error) {
	c := newCodexData()
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return c, fmt.Errorf("図鑑の形式が不正です")
	}
	if len(data) == 0 {
		return c, nil
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return newCodexData(), err
	}
	if c.Version < 0 || c.Version > 1 || c.Items == nil || c.Enemies == nil || c.Traps == nil {
		return newCodexData(), fmt.Errorf("図鑑の形式が不正です")
	}
	for _, table := range []map[int]CodexRecord{c.Items, c.Enemies, c.Traps} {
		for id, r := range table {
			if id < 0 || r.Defeated < 0 || r.Deaths < 0 {
				return newCodexData(), fmt.Errorf("図鑑の記録が不正です")
			}
			r.Seen = r.Seen || r.Identified || r.Stepped || r.Defeated > 0 || r.Deaths > 0
			table[id] = r
		}
	}
	c.Version = 1
	return c, nil
}

// mergeCodex は同じ記録のスナップショットを冪等に結合する（回数は大きい方）。
func mergeCodex(a, b CodexData) CodexData {
	out := newCodexData()
	for i, dst := range []map[int]CodexRecord{out.Items, out.Enemies, out.Traps} {
		for _, src := range []map[int]CodexRecord{[]map[int]CodexRecord{a.Items, a.Enemies, a.Traps}[i], []map[int]CodexRecord{b.Items, b.Enemies, b.Traps}[i]} {
			for id, r := range src {
				p := dst[id]
				dst[id] = CodexRecord{p.Seen || r.Seen, p.Identified || r.Identified, max(p.Defeated, r.Defeated), max(p.Deaths, r.Deaths), p.Stepped || r.Stepped}
			}
		}
	}
	return out
}
func codexDiscovery(ids []int, records map[int]CodexRecord) (found, total int, percent float64) {
	for _, id := range ids {
		total++
		if records[id].Seen {
			found++
		}
	}
	if total > 0 {
		percent = 100 * float64(found) / float64(total)
	}
	return
}

// codexMoveCursor は一覧のカーソルを delta だけ動かす。端で止め、wrap が true のときだけ反対の端へ回り込む
// （押した瞬間だけ回り込み、長押しのリピートでは端で止まる）。
func codexMoveCursor(cursor, delta, n int, wrap bool) int {
	if n <= 0 {
		return 0
	}
	next := cursor + delta
	switch {
	case next < 0 && wrap:
		return n - 1
	case next >= n && wrap:
		return 0
	}
	return min(max(next, 0), n-1)
}

// codexScrollFor は rows 行見える一覧で、カーソルが見える位置に入る先頭行を返す。
func codexScrollFor(cursor, scroll, rows, n int) int {
	if rows <= 0 || n <= rows {
		return 0
	}
	if cursor < scroll {
		scroll = cursor
	}
	if cursor >= scroll+rows {
		scroll = cursor - rows + 1
	}
	return min(max(scroll, 0), n-rows)
}

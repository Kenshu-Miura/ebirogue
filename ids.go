package main

import "sort"

// sortedTemplateIDs はテンプレートテーブルのキーのうち include を満たすものを昇順で返す。
// include が nil なら全キーを返す。ID に欠番があっても安全にランダム抽選できるよう、
// 「ランダム出現の対象 ID 一覧」を作るために使う。
func sortedTemplateIDs[T any](templates map[int]T, include func(T) bool) []int {
	ids := make([]int, 0, len(templates))
	for id, template := range templates {
		if include == nil || include(template) {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	return ids
}

// pickTemplateID は ids から intn で1つ選ぶ。ids が空なら fallback を返す。
func pickTemplateID(ids []int, intn func(int) int, fallback int) int {
	if len(ids) == 0 {
		return fallback
	}
	return ids[intn(len(ids))]
}

// weightedTemplateIDs は ids の各 ID を weights の回数だけ並べた抽選用の一覧を返す
// （weights に無い ID は1回、0以下なら出さない）。pickTemplateID と組み合わせて重み付きで抽選する。
func weightedTemplateIDs(ids []int, weights map[int]int) []int {
	pool := make([]int, 0, len(ids))
	for _, id := range ids {
		n, ok := weights[id]
		if !ok {
			n = 1
		}
		for i := 0; i < n; i++ {
			pool = append(pool, id)
		}
	}
	return pool
}

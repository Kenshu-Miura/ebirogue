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

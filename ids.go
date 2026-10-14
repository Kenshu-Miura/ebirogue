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

// spawnCategory は床出現のカテゴリ1つ（カテゴリの重みと、カテゴリ内の抽選用の ID 一覧）。
type spawnCategory struct {
	Name   string
	Weight int
	Pool   []int // カテゴリ内の重みの回数だけ ID を並べたもの
}

// buildSpawnCategories はテンプレート ID をカテゴリへ振り分け、カテゴリごとの抽選表を作る。
// categoryOf は ID のカテゴリ名、categoryWeights はカテゴリの重み、itemWeights はカテゴリ内の重み
// （書いていない ID は1）。重みが0以下のカテゴリと、ID が1つも無いカテゴリは入れない。並びはカテゴリ名の昇順。
func buildSpawnCategories(ids []int, categoryOf func(int) string, categoryWeights map[string]int, itemWeights map[int]int) []spawnCategory {
	members := map[string][]int{}
	for _, id := range ids {
		name := categoryOf(id)
		members[name] = append(members[name], id)
	}
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	categories := []spawnCategory{}
	for _, name := range names {
		pool := weightedTemplateIDs(members[name], itemWeights)
		if categoryWeights[name] <= 0 || len(pool) == 0 {
			continue
		}
		categories = append(categories, spawnCategory{Name: name, Weight: categoryWeights[name], Pool: pool})
	}
	return categories
}

// pickFromSpawnCategories はカテゴリを重みで選び、そのカテゴリ内から ID を選ぶ（乱数を2回使う）。
// カテゴリが無ければ fallback を返す。
func pickFromSpawnCategories(categories []spawnCategory, intn func(int) int, fallback int) int {
	total := 0
	for _, category := range categories {
		total += category.Weight
	}
	if total <= 0 {
		return fallback
	}
	r := intn(total)
	for _, category := range categories {
		if r < category.Weight {
			return pickTemplateID(category.Pool, intn, fallback)
		}
		r -= category.Weight
	}
	return fallback
}

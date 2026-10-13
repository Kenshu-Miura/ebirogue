package main

// rollItemTrapIndices は候補を重複なしで選ぶ。入力の順序は変更しない。
func rollItemTrapIndices(candidates []int, limit int, intn func(int) int) []int {
	pool := append([]int(nil), candidates...)
	result := []int{}
	for len(pool) > 0 && len(result) < limit {
		i := intn(len(pool))
		result = append(result, pool[i])
		pool = append(pool[:i], pool[i+1:]...)
	}
	return result
}

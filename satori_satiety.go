package main

// satoriSatietyCap は最大満腹度を変えず、悟りの盾の装備中だけ現在値を制限する。
func satoriSatietyCap(satiety int, equipped bool) int {
	if equipped {
		return min(satiety, 1)
	}
	return satiety
}

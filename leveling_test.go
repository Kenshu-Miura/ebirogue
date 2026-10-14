package main

import "testing"

func TestPlayerLevelExpTable(t *testing.T) {
	if len(playerLevelExpTable) != maxPlayerLevel+1 {
		t.Fatalf("表の長さ = %d, want %d", len(playerLevelExpTable), maxPlayerLevel+1)
	}
	if expForLevel(1) != 0 {
		t.Fatalf("Lv1 の必要経験値 = %d, want 0", expForLevel(1))
	}
	for level := 2; level <= maxPlayerLevel; level++ {
		if playerLevelExpTable[level] <= playerLevelExpTable[level-1] {
			t.Fatalf("Lv%d の必要経験値 %d が Lv%d の %d 以下", level, playerLevelExpTable[level], level-1, playerLevelExpTable[level-1])
		}
	}
	// 決定事項に書いた代表値
	for level, want := range map[int]int{2: 10, 5: 94, 10: 384, 20: 1745, 30: 4870} {
		if got := expForLevel(level); got != want {
			t.Errorf("expForLevel(%d) = %d, want %d", level, got, want)
		}
	}
}

func TestLevelForExp(t *testing.T) {
	cases := []struct{ exp, want int }{
		{0, 1},
		{expForLevel(2) - 1, 1},
		{expForLevel(2), 2},
		{expForLevel(10), 10},                  // 一度に複数レベル
		{expForLevel(31) - 1, 30},              // 境界の直前
		{expForLevel(maxPlayerLevel), 99},      // 上限
		{expForLevel(maxPlayerLevel) * 10, 99}, // 上限で止まる
	}
	for _, c := range cases {
		if got := levelForExp(c.exp); got != c.want {
			t.Errorf("levelForExp(%d) = %d, want %d", c.exp, got, c.want)
		}
	}
}

func TestLevelAttackBonus(t *testing.T) {
	if levelAttackBonus(1) != 0 {
		t.Fatalf("Lv1 の補正 = %d, want 0", levelAttackBonus(1))
	}
	if got := levelAttackBonus(maxPlayerLevel); got != 30 {
		t.Fatalf("Lv99 の補正 = %d, want 30", got)
	}
	for level := 2; level <= maxPlayerLevel; level++ {
		if levelAttackBonus(level) < levelAttackBonus(level-1) {
			t.Fatalf("Lv%d で補正が減った", level)
		}
	}
	if levelAttackBonus(0) != 0 || levelAttackBonus(200) != 30 {
		t.Fatal("範囲外のレベルは端の値に丸める")
	}
}

func TestLevelHealthBonus(t *testing.T) {
	for level, want := range map[int]int{1: 0, 10: 72, 30: 172, 60: 262, 99: 340} {
		if got := levelHealthBonus(level); got != want {
			t.Errorf("levelHealthBonus(%d) = %d, want %d", level, got, want)
		}
	}
}

func TestNaturalRegenPerTurn(t *testing.T) {
	cases := []struct{ level, want int }{{1, 1}, {9, 1}, {10, 2}, {19, 2}, {20, 3}, {99, 3}}
	for _, c := range cases {
		if got := naturalRegenPerTurn(c.level); got != c.want {
			t.Errorf("naturalRegenPerTurn(%d) = %d, want %d", c.level, got, c.want)
		}
	}
}

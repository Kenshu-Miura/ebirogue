package main

import "testing"

// intnMin は常に最小の乱数（0）を返す。会心判定がある場合は必ず会心になる点に注意。
func intnMin(int) int { return 0 }

// intnMax は常に最大の乱数（n-1）を返す。会心判定は必ず失敗する。
func intnMax(n int) int { return n - 1 }

func TestDefenseFactor(t *testing.T) {
	if got := defenseFactor(0); got != 1.0 {
		t.Fatalf("defenseFactor(0) = %f, want 1.0", got)
	}
	if got := defenseFactor(-3); got != 1.0 {
		t.Fatalf("defenseFactor(-3) = %f, want 1.0", got)
	}
	if got := defenseFactor(1); got != defenseMitigationRate {
		t.Fatalf("defenseFactor(1) = %f, want %f", got, defenseMitigationRate)
	}
	if defenseFactor(10) >= defenseFactor(5) {
		t.Fatal("defense factor must decrease as defense increases")
	}
}

func TestDamageRandomFactor(t *testing.T) {
	if got := damageRandomFactor(intnMin); got != 112.0/128.0 {
		t.Fatalf("minimum random factor = %f, want %f", got, 112.0/128.0)
	}
	if got := damageRandomFactor(intnMax); got != 144.0/128.0 {
		t.Fatalf("maximum random factor = %f, want %f", got, 144.0/128.0)
	}
}

func TestRollEnemyAttackDamage(t *testing.T) {
	// 10 × (15/16)^4 ≒ 7.72 に乱数係数 7/8〜9/8 を掛けた範囲
	if got := rollEnemyAttackDamage(10, 4, intnMin); got != 6 {
		t.Fatalf("minimum enemy damage = %d, want 6", got)
	}
	if got := rollEnemyAttackDamage(10, 4, intnMax); got != 8 {
		t.Fatalf("maximum enemy damage = %d, want 8", got)
	}
	// 防御が大きく上回っても最低1ダメージは保証される
	if got := rollEnemyAttackDamage(2, 10, intnMin); got != 1 {
		t.Fatalf("minimum guaranteed damage = %d, want 1", got)
	}
}

func TestRollPlayerAttackDamage(t *testing.T) {
	// intnMax は会心判定に失敗し、乱数係数が最大になる
	result := rollPlayerAttackDamage(12, 2, 1.0, intnMax)
	if result.Critical {
		t.Fatal("intnMax must not trigger a critical hit")
	}
	// 12 × (15/16)^2 × 9/8 ≒ 11.86 → 11
	if result.Damage != 11 {
		t.Fatalf("player damage = %d, want 11", result.Damage)
	}
}

func TestRollPlayerAttackDamageCritical(t *testing.T) {
	// intnMin は会心判定に成功する（防御無視 × 1.5 × 乱数最小 7/8）
	result := rollPlayerAttackDamage(12, 99, 1.0, intnMin)
	if !result.Critical {
		t.Fatal("intnMin must trigger a critical hit")
	}
	// 12 × 1.5 × 7/8 = 15.75 → 15（防御99は無視される）
	if result.Damage != 15 {
		t.Fatalf("critical damage = %d, want 15", result.Damage)
	}
}

func TestRollPlayerAttackDamageWithSlayerMultiplier(t *testing.T) {
	// 29 × (15/16)^9 × 9/8 × 1.5 ≒ 27.38 → 27
	result := rollPlayerAttackDamage(29, 9, slayerDamageMultiplier, intnMax)
	if result.Damage != 27 {
		t.Fatalf("slayer damage = %d, want 27", result.Damage)
	}
}

func TestPlayerAttackTotal(t *testing.T) {
	if got := playerAttackTotal(3, 8, 1); got != 12 {
		t.Fatalf("playerAttackTotal(3, 8, 1) = %d, want 12", got)
	}
}

func TestRollThrownItemDamage(t *testing.T) {
	if got := rollThrownItemDamage(intnMin); got != 1 {
		t.Fatalf("minimum thrown damage = %d, want 1", got)
	}
	if got := rollThrownItemDamage(intnMax); got != 3 {
		t.Fatalf("maximum thrown damage = %d, want 3", got)
	}
}

func TestRollDamageMultiplierComposition(t *testing.T) {
	// 倍率は乗算合成される（0以下は1.0扱い）。16 × 9/8 = 18 で端数なし。
	base := rollDamage(DamageParams{Attack: 16, Defense: 0, MinDamage: 1}, intnMax)
	doubled := rollDamage(DamageParams{Attack: 16, Defense: 0, Multiplier: 2.0, MinDamage: 1}, intnMax)
	if base.Damage != 18 || doubled.Damage != 36 {
		t.Fatalf("damage = (%d, %d), want (18, 36)", base.Damage, doubled.Damage)
	}
	ignored := rollDamage(DamageParams{Attack: 16, Defense: 0, Multiplier: -1.0, MinDamage: 1}, intnMax)
	if ignored.Damage != base.Damage {
		t.Fatalf("non-positive multiplier damage = %d, want %d", ignored.Damage, base.Damage)
	}
}

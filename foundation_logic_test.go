package main

// 共通基盤 F1〜F5（content-expansion-plan.md）の純粋関数のテスト。

import (
	"reflect"
	"testing"
)

func TestCountEquipmentAbilityStacks(t *testing.T) {
	ring := []EquipmentAbilityID{"test_ring"}
	tests := []struct {
		name  string
		lists [][]EquipmentAbilityID
		want  int
	}{
		{name: "none", want: 0},
		{name: "one ring", lists: [][]EquipmentAbilityID{ring, nil}, want: 1},
		{name: "two rings stack", lists: [][]EquipmentAbilityID{ring, ring}, want: 2},
		{name: "weapon and ring stack", lists: [][]EquipmentAbilityID{{"test_ring", AbilitySureHit}, ring}, want: 2},
		{name: "duplicate in one item counts once", lists: [][]EquipmentAbilityID{{"test_ring", "test_ring"}}, want: 1},
		{name: "other ability", lists: [][]EquipmentAbilityID{{AbilitySureHit}}, want: 0},
	}
	for _, tt := range tests {
		if got := countEquipmentAbility("test_ring", tt.lists...); got != tt.want {
			t.Errorf("%s: count = %d, want %d", tt.name, got, tt.want)
		}
	}
	got := uniqueEquipmentAbilities([]EquipmentAbilityID{"b", "a"}, []EquipmentAbilityID{"a", "c"})
	if want := []EquipmentAbilityID{"b", "a", "c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unique abilities = %v, want %v", got, want)
	}
}

func TestAccessoryPowerBonusUsesAbilityID(t *testing.T) {
	if got := accessoryPowerBonus([]EquipmentAbilityID{AbilityPowerBoost}, false); got != 3 {
		t.Fatalf("power ring bonus = %d, want 3", got)
	}
	if got := accessoryPowerBonus([]EquipmentAbilityID{AbilityPowerBoost}, true); got != -3 {
		t.Fatalf("cursed power ring bonus = %d, want -3", got)
	}
	if got := accessoryPowerBonus(nil, false); got != 0 {
		t.Fatalf("ring without ability bonus = %d, want 0", got)
	}
}

func TestStatusImmunityTable(t *testing.T) {
	statusImmunities["test_guard"] = []StatusKind{StatusConfusion, StatusSleep}
	defer delete(statusImmunities, "test_guard")

	abilities := []EquipmentAbilityID{AbilitySureHit, "test_guard"}
	if !abilitiesGrantImmunity(abilities, StatusConfusion) || !abilitiesGrantImmunity(abilities, StatusSleep) {
		t.Fatal("test_guard should grant confusion and sleep immunity")
	}
	if abilitiesGrantImmunity(abilities, StatusBlind) {
		t.Fatal("test_guard should not grant blind immunity")
	}
	if abilitiesGrantImmunity([]EquipmentAbilityID{AbilitySureHit}, StatusConfusion) {
		t.Fatal("ability without an immunity entry should not grant immunity")
	}
	for _, kind := range []StatusKind{StatusConfusion, StatusSleep, StatusBlind, StatusSlow, StatusBearTrap, StatusPoison} {
		if statusKindLabel(kind) == string(kind) {
			t.Errorf("status %q has no Japanese label", kind)
		}
	}
}

func TestSlayerTraitsTable(t *testing.T) {
	for ability, trait := range slayerTraits {
		if m, ok := slayerMultiplier([]EquipmentAbilityID{ability}, []EnemyTrait{trait}); !ok || m != slayerDamageMultiplier {
			t.Errorf("%s vs %s = %v/%v, want slayer", ability, trait, m, ok)
		}
	}
	for _, tt := range []struct {
		ability EquipmentAbilityID
		trait   EnemyTrait
	}{
		{AbilityShellSlayer, EnemyTraitShell},
		{AbilityExplosiveSlayer, EnemyTraitExplosive},
		{AbilityFloatingSlayer, EnemyTraitFloating},
	} {
		if slayerTraits[tt.ability] != tt.trait {
			t.Errorf("slayerTraits[%s] = %q, want %q", tt.ability, slayerTraits[tt.ability], tt.trait)
		}
	}
	// 一致しない属性・複数一致でも倍率は重複しない
	if m, ok := slayerMultiplier([]EquipmentAbilityID{AbilityShellSlayer}, []EnemyTrait{EnemyTraitFloating}); ok || m != 1.0 {
		t.Fatalf("mismatched slayer = %v/%v", m, ok)
	}
	if m, _ := slayerMultiplier([]EquipmentAbilityID{AbilityShellSlayer, AbilityFloatingSlayer}, []EnemyTrait{EnemyTraitShell, EnemyTraitFloating}); m != slayerDamageMultiplier {
		t.Fatalf("double slayer multiplier = %v, want %v", m, slayerDamageMultiplier)
	}
}

func TestRollDamageForceCritical(t *testing.T) {
	maxRoll := func(n int) int { return n - 1 } // 会心判定は外れ、乱数係数は最大
	normal := rollDamage(DamageParams{Attack: 20, Defense: 10, MinDamage: 1, CriticalDenominator: 20}, maxRoll)
	forced := rollDamage(DamageParams{Attack: 20, Defense: 10, MinDamage: 1, CriticalDenominator: 20, ForceCritical: true}, maxRoll)
	if normal.Critical || !forced.Critical || forced.Damage <= normal.Damage {
		t.Fatalf("normal %+v, forced %+v", normal, forced)
	}
	if got := rollEnemyAttackDamageWith(20, 0, enemyCriticalMultiplier, maxRoll); got != rollEnemyAttackDamage(20, 0, maxRoll)*3/2 {
		t.Fatalf("critical enemy damage = %d, want 1.5x of %d", got, rollEnemyAttackDamage(20, 0, maxRoll))
	}
}

func TestRollPlayerAttackHitChance(t *testing.T) {
	calls := 0
	intn := func(int) int { calls++; return 50 }
	if !rollPlayerAttackHitChance(51, false, intn) || rollPlayerAttackHitChance(50, false, intn) {
		t.Fatal("hit chance should compare the roll against accuracy")
	}
	calls = 0
	if !rollPlayerAttackHitChance(0, true, intn) || calls != 0 {
		t.Fatal("sure hit should hit without consuming a roll")
	}
}

func TestDisplacementPath(t *testing.T) {
	// 5x5 の内側が床、(3,1) が壁、(1,3) に敵
	open := func(x, y int) bool { return x >= 1 && x <= 3 && y >= 1 && y <= 3 && !(x == 3 && y == 1) }
	occupied := func(x, y int) bool { return x == 1 && y == 3 }

	path, stop := displacementPath(1, 1, 1, 0, 3, open, occupied)
	if !reflect.DeepEqual(path, []Coordinate{{X: 2, Y: 1}}) || stop != DisplacementHitWall {
		t.Fatalf("east path = %v/%v, want one step then wall", path, stop)
	}
	path, stop = displacementPath(1, 1, 0, 1, 3, open, occupied)
	if !reflect.DeepEqual(path, []Coordinate{{X: 1, Y: 2}}) || stop != DisplacementHitCharacter {
		t.Fatalf("south path = %v/%v, want one step then character", path, stop)
	}
	path, stop = displacementPath(2, 3, 0, -1, 2, open, occupied)
	if !reflect.DeepEqual(path, []Coordinate{{X: 2, Y: 2}, {X: 2, Y: 1}}) || stop != DisplacementFinished {
		t.Fatalf("north path = %v/%v, want two steps", path, stop)
	}
	// (2,2)→(3,1) は壁。(2,1)→(3,2) は角の (3,1) が壁なので角抜け扱い
	path, stop = displacementPath(2, 1, 1, 1, 2, open, occupied)
	if len(path) != 0 || stop != DisplacementHitWall {
		t.Fatalf("diagonal corner = %v/%v, want blocked", path, stop)
	}
	if path, stop = displacementPath(2, 2, 0, 0, 3, open, occupied); path != nil || stop != DisplacementFinished {
		t.Fatalf("zero direction = %v/%v", path, stop)
	}
}

func TestForwardDropPositions(t *testing.T) {
	free := func(x, y int) bool { return x >= 0 && x <= 4 && y >= 0 && y <= 4 && !(x == 4 && y == 2) }
	got := forwardDropPositions(2, 2, 1, 0, 2, free)
	if len(got) != 8 || got[0] != (Coordinate{X: 3, Y: 2}) {
		t.Fatalf("positions = %v, want forward tile first and 8 unique positions", got)
	}
	seen := map[Coordinate]bool{}
	for _, c := range got {
		if seen[c] || (c == Coordinate{X: 4, Y: 2}) {
			t.Fatalf("positions = %v contain duplicates or a blocked tile", got)
		}
		seen[c] = true
	}
	// 前方2マスが空いていれば2マス先も前方として先に並ぶ
	got = forwardDropPositions(1, 2, 1, 0, 2, free)
	if got[0] != (Coordinate{X: 2, Y: 2}) || got[1] != (Coordinate{X: 3, Y: 2}) {
		t.Fatalf("positions = %v, want two forward tiles first", got)
	}
}

func TestSpawnCategoriesKeepCategoryShares(t *testing.T) {
	categoryOf := func(id int) string {
		if id < 10 {
			return "Weapon"
		}
		return "Food"
	}
	weights := map[string]int{"Weapon": 3, "Food": 1, "Unused": 5}
	pick := func(categories []spawnCategory) map[string]int {
		counts := map[string]int{}
		total := 0
		for _, c := range categories {
			total += c.Weight
		}
		// カテゴリの抽選 r を全通り、カテゴリ内の抽選は先頭に固定して数える
		for r := 0; r < total; r++ {
			calls := 0
			id := pickFromSpawnCategories(categories, func(n int) int {
				calls++
				if calls == 1 {
					return r
				}
				return 0
			}, -1)
			counts[categoryOf(id)]++
		}
		return counts
	}

	categories := buildSpawnCategories([]int{1, 2, 10}, categoryOf, weights, nil)
	if len(categories) != 2 || categories[0].Name != "Food" || categories[1].Name != "Weapon" {
		t.Fatalf("categories = %+v, want Food and Weapon only (sorted)", categories)
	}
	before := pick(categories)
	// 武器を2つ追加してもカテゴリの割合は変わらない
	more := buildSpawnCategories([]int{1, 2, 3, 4, 10}, categoryOf, weights, map[int]int{2: 3})
	after := pick(more)
	if !reflect.DeepEqual(before, after) || before["Weapon"] != 3 || before["Food"] != 1 {
		t.Fatalf("category picks before %v after %v, want Weapon 3 : Food 1", before, after)
	}
	if got := len(more[1].Pool); got != 6 {
		t.Fatalf("weapon pool = %v, want in-category weights (1+3+1+1)", more[1].Pool)
	}
	if got := pickFromSpawnCategories(nil, func(int) int { return 0 }, 42); got != 42 {
		t.Fatalf("empty categories pick = %d, want fallback", got)
	}
}

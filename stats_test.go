//go:build !test

package main

import "testing"

// 冒険の統計（拾う・使う・投げる・装備・射撃）が本物の操作で数えられ、セーブ・再開後も残る。
func TestRunStatsCountActions(t *testing.T) {
	g, b := newTurnTestGame(t)
	arrow := buildItemFromTemplate(6, 0, 0).(*Arrow)
	arrow.ShotCount = 3
	g.state.Player.Inventory = []Item{buildItemFromTemplate(1, 0, 0), buildItemFromTemplate(2, 0, 0), buildItemFromTemplate(20, 0, 0), arrow}
	g.state.Player.EquippedArrow = arrow
	g.state.Items = []Item{buildItemFromTemplate(1, 5, 3)}

	steps := []func() error{
		func() error { return useOrFail(b, 0, "使う") },  // ウインナーを食べる
		func() error { return useOrFail(b, 0, "投げる") }, // ミンティアを投げる
		func() error { return useOrFail(b, 0, "装備") },  // こん棒を装備
		func() error { return b.tap(KeyD) },            // 矢を撃つ
		func() error { return b.move(1, 0) },           // 足元のウインナーを拾う
	}
	for _, step := range steps {
		if err := step(); err != nil {
			t.Fatal(err)
		}
		if err := g.RunUntilInputReady(2000); err != nil {
			t.Fatal(err)
		}
	}
	items := g.stats.Items
	if items["Food"] == nil || items["Food"].Used != 1 || items["Food"].Picked != 1 {
		t.Fatalf("food stats = %+v", items["Food"])
	}
	if items["Potion"] == nil || items["Potion"].Thrown != 1 {
		t.Fatalf("potion stats = %+v", items["Potion"])
	}
	if items["Weapon"] == nil || items["Weapon"].Equipped != 1 || items["Arrow"] == nil || items["Arrow"].Shot != 1 {
		t.Fatalf("weapon/arrow stats = %+v / %+v", items["Weapon"], items["Arrow"])
	}
	// テンプレート ID ごとの投げた回数（wave11-0）
	if g.stats.ThrownIDs[2] != 1 {
		t.Fatalf("thrown IDs %+v", g.stats.ThrownIDs)
	}
	if g.stats.UsedIDs[1] != 1 || len(g.stats.RecentUses) != 1 || g.stats.RecentUses[0].TemplateID != 1 {
		t.Fatalf("used IDs %+v, recent %+v", g.stats.UsedIDs, g.stats.RecentUses)
	}
	restored, vs := saveRoundTrip(g)
	if len(vs) > 0 || restored.stats.Items["Food"].Used != 1 || restored.stats.ThrownIDs[2] != 1 {
		t.Fatalf("stats after save round trip: %+v (violations %+v)", restored.stats.Items, vs)
	}
	result := restored.buildRunResult(false)
	if result.Stats.UsedIDs[1] != 1 || len(result.InventoryIDs) != len(result.Inventory) {
		t.Fatalf("run result stats = %+v", result)
	}
}

func TestMistakeKind(t *testing.T) {
	heal := RunResult{DeathCause: enemyAttackSource(0, "エビ"), InventoryIDs: []int{2}}
	starve := RunResult{DeathCause: DamageSource{Kind: DamageSourceStarvation}, InventoryIDs: []int{1}}
	harmful := RunResult{Floor: 3, Turns: 500, DeathCause: enemyAttackSource(0, "エビ"),
		Stats: RunStats{RecentUses: []ItemUseRecord{{TemplateID: 35, Floor: 3, Turn: 450}}}}
	oldHarmful := harmful
	oldHarmful.Stats.RecentUses = []ItemUseRecord{{TemplateID: 35, Floor: 2, Turn: 450}}
	for _, tt := range []struct {
		r    RunResult
		want string
	}{
		{heal, mistakeHealUnused},
		{starve, mistakeFoodUnused},
		{harmful, mistakeHarmfulCard},
		{oldHarmful, ""},
		{RunResult{DeathCause: DamageSource{Kind: DamageSourceSelfDestruct}}, mistakeHarmfulCard},
		{RunResult{DeathCause: DamageSource{Kind: DamageSourceWind}, InventoryIDs: []int{2}}, ""},
		{RunResult{Cleared: true, InventoryIDs: []int{2}}, ""},
	} {
		r := tt.r
		if got := mistakeKind(&r); got != tt.want {
			t.Errorf("mistakeKind(%+v) = %q, want %q", tt.r, got, tt.want)
		}
	}
}

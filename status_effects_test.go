package main

import "testing"

func TestStatusDefsTable(t *testing.T) {
	seen := map[StatusKind]bool{}
	for _, def := range statusDefs {
		if seen[def.Kind] {
			t.Fatalf("duplicate status %q", def.Kind)
		}
		seen[def.Kind] = true
		if def.Name == "" || def.Turns == nil || def.SetTurns == nil {
			t.Fatalf("status %q is missing name or accessors", def.Kind)
		}
		if def.Player == StatusNotApplicable && def.Enemy == StatusNotApplicable {
			t.Fatalf("status %q applies to nobody", def.Kind)
		}
		if def.Stack == StackDefault {
			t.Fatalf("status %q needs a stacking rule", def.Kind)
		}
		// 項目の読み書きが対応していて、ほかの項目に触れないこと
		var s StatusAilments
		def.SetTurns(&s, 1)
		if def.Turns(&s) != 1 {
			t.Fatalf("status %q accessors do not round-trip", def.Kind)
		}
		for _, other := range statusDefs {
			if other.Kind != def.Kind && other.Turns(&s) != 0 {
				t.Fatalf("setting %q changed %q", def.Kind, other.Kind)
			}
		}
		if statusKindLabel(def.Kind) != def.Name {
			t.Fatalf("label of %q = %q", def.Kind, statusKindLabel(def.Kind))
		}
	}
}

func TestStackStatusTurns(t *testing.T) {
	cases := []struct {
		rule                           StatusStack
		current, turns, maxTurns, want int
	}{
		{StackOverwrite, 20, 10, 0, 10},
		{StackLonger, 3, 6, 0, 6},
		{StackLonger, 9, 6, 0, 9},
		{StackAdd, 5, 10, 0, 15},
		{StackAdd, 25, 10, 30, 30},
		{StackAdd, -1, 10, 0, 10}, // 仮眠の -1 には加算しない
	}
	for _, tc := range cases {
		if got := stackStatusTurns(tc.rule, tc.current, tc.turns, tc.maxTurns); got != tc.want {
			t.Errorf("stackStatusTurns(%v, %d, %d, %d) = %d, want %d", tc.rule, tc.current, tc.turns, tc.maxTurns, got, tc.want)
		}
	}
}

func TestAddAndClearStatus(t *testing.T) {
	var s StatusAilments
	addStatus(&s, StatusBearTrap, 6, StackDefault) // 表の規則は長い方
	addStatus(&s, StatusBearTrap, 3, StackDefault)
	if s.BearTrap != 6 {
		t.Fatalf("bear trap = %d", s.BearTrap)
	}
	addStatus(&s, StatusSleep, 10, StackDefault)
	addStatus(&s, StatusSleep, 4, StackDefault) // 上書き
	if s.Sleep != 4 {
		t.Fatalf("sleep = %d", s.Sleep)
	}
	addStatus(&s, StatusSleep, 2, StackLonger) // 規則の指定
	if s.Sleep != 4 {
		t.Fatalf("sleep with StackLonger = %d", s.Sleep)
	}
	addStatus(&s, StatusSeal, 1, StackDefault)
	if !s.Seal || !statusActive(s, StatusSeal) {
		t.Fatal("seal should be set")
	}
	addStatus(&s, StatusCurse, 5, StackDefault) // 表に無い種類は何もしない
	clearStatus(&s, StatusSeal)
	clearStatus(&s, StatusSleep)
	if s.Seal || s.Sleep != 0 || s.BearTrap != 6 {
		t.Fatalf("after clear %+v", s)
	}
}

func TestTickStatuses(t *testing.T) {
	s := StatusAilments{BearTrap: 1, Sleep: 2, Blind: 1, Haste: 1, Backward: 1, Paralysis: true}
	ended := tickStatuses(&s, false)
	if len(ended) != 2 || ended[0] != StatusBearTrap || ended[1] != StatusBlind {
		t.Fatalf("player ended %v", ended)
	}
	// 海老さんの倍速・ゾワゾワは今は付かない状態なので減らない
	if s != (StatusAilments{Sleep: 1, Haste: 1, Backward: 1, Paralysis: true}) {
		t.Fatalf("player after tick %+v", s)
	}

	e := StatusAilments{Sleep: -1, Blind: 5, Haste: 1, BearTrap: 3, MouthSeal: 3}
	ended = tickStatuses(&e, true)
	if len(ended) != 1 || ended[0] != StatusHaste {
		t.Fatalf("enemy ended %v", ended)
	}
	if e != (StatusAilments{Sleep: -1, Blind: 5, BearTrap: 2, MouthSeal: 3}) {
		t.Fatalf("enemy after tick %+v", e)
	}
}

func TestBadStatuses(t *testing.T) {
	if hasBadStatus(StatusAilments{Haste: 3, Backward: 2, Sleep: enemyNapSleep}) {
		t.Fatal("haste, backward and nap are not bad statuses")
	}
	for _, def := range statusDefs {
		if !def.Bad {
			continue
		}
		var s StatusAilments
		def.SetTurns(&s, 1)
		if !hasBadStatus(s) {
			t.Fatalf("%q should be bad", def.Kind)
		}
	}
	s := StatusAilments{Sleep: 3, Seal: true, HasteOnWake: true, Haste: 2}
	clearBadStatuses(&s)
	if s != (StatusAilments{Haste: 2}) {
		t.Fatalf("after clear %+v", s)
	}
}

func TestFormatPlayerStatusOrder(t *testing.T) {
	s := StatusAilments{BearTrap: 1, Slow: 2, Haste: 3, Sleep: 4, Confusion: 5, Blind: 6, MouthSeal: 7, Paralysis: true, Seal: true}
	want := "トラバサミ(1) 鈍足(2) 倍速(3) 睡眠(4) 混乱(5) 目潰し(6) 口封じ(7) 金縛り 封印"
	if got := formatPlayerStatus(s); got != want {
		t.Fatalf("formatPlayerStatus = %q, want %q", got, want)
	}
	if got := formatPlayerStatus(StatusAilments{Sleep: enemyNapSleep}); got != "正常" {
		t.Fatalf("nap should not be shown: %q", got)
	}
}

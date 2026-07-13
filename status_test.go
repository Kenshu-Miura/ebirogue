package main

import "testing"

func TestMineTrapDamage(t *testing.T) {
	tests := []struct {
		health int
		want   int
	}{
		{health: 100, want: 50},
		{health: 9, want: 4},
		{health: 1, want: 0},
		{health: 0, want: 0},
	}

	for _, tt := range tests {
		if got := mineTrapDamage(tt.health); got != tt.want {
			t.Errorf("mineTrapDamage(%d) = %d, want %d", tt.health, got, tt.want)
		}
	}
}

func TestShouldTakeExtraEnemyTurn(t *testing.T) {
	if !shouldTakeExtraEnemyTurn(3, 2) {
		t.Fatal("slow player should give enemies an extra turn on even move counts")
	}
	if shouldTakeExtraEnemyTurn(0, 2) || shouldTakeExtraEnemyTurn(3, 3) {
		t.Fatal("extra enemy turn should require active slow status and an even move count")
	}
}

func TestFormatPlayerStatus(t *testing.T) {
	if got := formatPlayerStatus(StatusAilments{}); got != "正常" {
		t.Fatalf("empty status = %q, want 正常", got)
	}

	status := StatusAilments{Poison: 4, Slow: 2, Paralysis: true}
	if got := formatPlayerStatus(status); got != "毒(4) 鈍足(2) 金縛り" {
		t.Fatalf("formatted status = %q", got)
	}

	if got := formatPlayerStatus(StatusAilments{Haste: 3}); got != "倍速(3)" {
		t.Fatalf("haste status = %q, want 倍速(3)", got)
	}
}

func TestWakeFromSleep(t *testing.T) {
	status := StatusAilments{Sleep: 3, HasteOnWake: true}
	if !wakeFromSleep(&status) {
		t.Fatal("目覚め時倍速化フラグ付きの睡眠解除は倍速化するはず")
	}
	if status.Sleep != 0 || status.Haste != hasteOnWakeTurns || status.HasteOnWake {
		t.Fatalf("unexpected status after wake: %#v", status)
	}

	plain := StatusAilments{Sleep: 3}
	if wakeFromSleep(&plain) {
		t.Fatal("フラグなしの睡眠解除は倍速化しないはず")
	}
	if plain.Sleep != 0 || plain.Haste != 0 {
		t.Fatalf("unexpected status after wake: %#v", plain)
	}
}

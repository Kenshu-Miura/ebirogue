package main

import "testing"

func TestDeathMessageDescribesEachCause(t *testing.T) {
	tests := []struct {
		source DamageSource
		want   string
	}{
		{enemyRangedSource(38, "火吹きイカ", "炎"), "10階で 火吹きイカ の炎にやられた"},
		{enemyAttackSource(0, "エビ"), "10階で エビ にやられた"},
		{trapSource(3, "地雷"), "10階で 地雷 にかかって倒れた"},
		{DamageSource{Kind: DamageSourceStarvation}, "10階で 空腹のあまり倒れた"},
		{DamageSource{Kind: DamageSourcePoison}, "10階で 毒がまわって倒れた"},
		{DamageSource{Kind: DamageSourceSelfDestruct, Detail: "自爆のカード"}, "10階で 自爆のカード の爆発に巻き込まれた"},
		{DamageSource{Kind: DamageSourceWind}, "10階で 突風に吹き飛ばされた"},
		{DamageSource{}, "10階で 倒れた"},
	}
	for _, tt := range tests {
		if got := deathMessage(10, tt.source); got != tt.want {
			t.Errorf("deathMessage(%+v) = %q, want %q", tt.source, got, tt.want)
		}
	}
	if got := enemyRangedSource(38, "火吹きイカ", "炎").Label(); got != "火吹きイカの炎" {
		t.Errorf("Label = %q", got)
	}
}

func TestAppendRunHistoryKeepsLatestWithinLimit(t *testing.T) {
	var data []byte
	var err error
	for i := 0; i < 5; i++ {
		data, err = appendRunHistory(data, RunResult{Seed: int64(i)}, 3)
		if err != nil {
			t.Fatal(err)
		}
	}
	history, err := decodeRunHistory(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 3 || history[0].Seed != 2 || history[2].Seed != 4 {
		t.Fatalf("history = %+v", history)
	}
	// 壊れた記録は作り直す
	data, err = appendRunHistory([]byte("{broken"), RunResult{Seed: 9}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if history, _ = decodeRunHistory(data); len(history) != 1 || history[0].Seed != 9 {
		t.Fatalf("history after broken data = %+v", history)
	}
}

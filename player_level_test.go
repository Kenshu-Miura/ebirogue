//go:build !test

package main

import "testing"

func TestCheckLevelUpRaisesSeveralLevels(t *testing.T) {
	g, _ := newHeadlessGame(1, DifficultyAdvanced)
	p := &g.state.Player
	if p.MaxHealth != initialPlayerHealth || p.Health != initialPlayerHealth {
		t.Fatalf("初期HP = %d/%d, want %d", p.Health, p.MaxHealth, initialPlayerHealth)
	}
	p.Health = 10
	p.ExperiencePoints = expForLevel(10)
	p.checkLevelUp(g)
	if p.Level != 10 {
		t.Fatalf("Level = %d, want 10", p.Level)
	}
	if want := initialPlayerHealth + levelHealthBonus(10); p.MaxHealth != want {
		t.Fatalf("MaxHealth = %d, want %d", p.MaxHealth, want)
	}
	if want := 10 + levelHealthBonus(10); p.Health != want {
		t.Fatalf("Health = %d, want %d（上がった分だけ回復）", p.Health, want)
	}
}

func TestCheckLevelUpStopsAtMaxLevel(t *testing.T) {
	g, _ := newHeadlessGame(1, DifficultyAdvanced)
	p := &g.state.Player
	p.ExperiencePoints = expForLevel(maxPlayerLevel) * 3
	p.checkLevelUp(g)
	if p.Level != maxPlayerLevel {
		t.Fatalf("Level = %d, want %d", p.Level, maxPlayerLevel)
	}
	p.ExperiencePoints += 100000
	p.checkLevelUp(g)
	if p.Level != maxPlayerLevel {
		t.Fatalf("上限を超えた: Level = %d", p.Level)
	}
}

func TestSetLevelDown(t *testing.T) {
	g, _ := newHeadlessGame(1, DifficultyAdvanced)
	p := &g.state.Player
	p.ExperiencePoints = expForLevel(20) + 5
	p.checkLevelUp(g)
	maxAt20 := p.MaxHealth
	p.setLevel(15)
	if p.Level != 15 || p.ExperiencePoints != expForLevel(15) {
		t.Fatalf("Lv%d 経験値%d, want Lv15 経験値%d", p.Level, p.ExperiencePoints, expForLevel(15))
	}
	if want := maxAt20 - (levelHealthBonus(20) - levelHealthBonus(15)); p.MaxHealth != want || p.Health > p.MaxHealth {
		t.Fatalf("HP %d/%d, want 最大 %d", p.Health, p.MaxHealth, want)
	}
}

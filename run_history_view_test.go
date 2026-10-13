package main

import "testing"

func TestNewestRunHistory(t *testing.T) {
	input := []RunResult{{Seed: 1}, {Seed: 2}, {Seed: 3}}
	got := newestRunHistory(input)
	if got[0].Seed != 3 || got[2].Seed != 1 || input[0].Seed != 1 {
		t.Fatal("history must be newest first without changing storage order")
	}
	if len(newestRunHistory(nil)) != 0 {
		t.Fatal("empty history")
	}
}

func TestDifficultyPreferenceCompatibility(t *testing.T) {
	for _, data := range []string{`{"fullscreen":false}`, `{"lastDifficulty":99}`} {
		settings, err := decodeSettings([]byte(data))
		if err != nil || settings.LastDifficulty != DifficultyAdvanced {
			t.Fatal("old or invalid preference should default to advanced")
		}
	}
}

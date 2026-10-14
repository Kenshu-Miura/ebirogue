package main

import "testing"

func TestGnawResult(t *testing.T) {
	for _, tc := range []struct {
		satiety, sharpness, wantSatiety, wantSharpness int
		consumed                                       bool
	}{
		{50, 0, 65, -1, false}, {95, -2, 100, -3, false}, {100, -3, 100, -4, true}, {0, -4, 15, -5, true},
	} {
		s, w, consumed := gnawResult(tc.satiety, 100, tc.sharpness)
		if s != tc.wantSatiety || w != tc.wantSharpness || consumed != tc.consumed {
			t.Fatal(tc, s, w, consumed)
		}
	}
}

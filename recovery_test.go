package main

import "testing"

func TestRecoveredValue(t *testing.T) {
	tests := []struct {
		name         string
		current      int
		maximum      int
		amount       int
		fullRecovery bool
		want         int
	}{
		{name: "fixed recovery", current: 20, maximum: 100, amount: 60, want: 80},
		{name: "recovery is capped", current: 80, maximum: 100, amount: 60, want: 100},
		{name: "full recovery", current: 1, maximum: 150, fullRecovery: true, want: 150},
		{name: "zero recovery", current: 40, maximum: 100, want: 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := recoveredValue(tt.current, tt.maximum, tt.amount, tt.fullRecovery); got != tt.want {
				t.Fatalf("recoveredValue() = %d, want %d", got, tt.want)
			}
		})
	}
}
